package updater

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/vrc-web-companion/gateway/internal/model"
)

type asset struct {
	File    string   `json:"file"`
	SHA256  string   `json:"sha256"`
	Size    int64    `json:"size"`
	Mirrors []string `json:"mirrors"`
}
type manifest struct {
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"publishedAt"`
	WindowsX64  asset     `json:"windowsX64"`
}

type Service struct {
	current, directory string
	sources            []string
	client             *http.Client
	logger             *slog.Logger
	mu                 sync.RWMutex
	status             model.UpdateStatus
	asset              asset
}

func New(current, directory string, client *http.Client, logger *slog.Logger) *Service {
	var sources []string
	for _, raw := range strings.Split(os.Getenv("VRC_HARBOR_UPDATE_URLS"), ";") {
		if value := strings.TrimSpace(raw); value != "" {
			sources = append(sources, value)
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	state := "unconfigured"
	message := "设置 VRC_HARBOR_UPDATE_URLS 后启用双源更新"
	if len(sources) > 0 {
		state, message = "idle", "已配置更新源"
	}
	return &Service{current: current, directory: directory, sources: sources, client: client, logger: logger, status: model.UpdateStatus{State: state, Current: current, Message: message}}
}

func (s *Service) Status() model.UpdateStatus { s.mu.RLock(); defer s.mu.RUnlock(); return s.status }

func (s *Service) Check(ctx context.Context) model.UpdateStatus {
	if len(s.sources) == 0 {
		return s.Status()
	}
	var failures []string
	for _, source := range s.sources {
		parsed, err := url.Parse(source)
		if err != nil || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && parsed.Hostname() == "127.0.0.1")) {
			failures = append(failures, "无效更新源")
			continue
		}
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		response, err := s.client.Do(request)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK {
			failures = append(failures, fmt.Sprintf("%s: HTTP %d", parsed.Host, response.StatusCode))
			continue
		}
		var value manifest
		if err := json.Unmarshal(data, &value); err != nil || value.Version == "" || value.WindowsX64.SHA256 == "" {
			failures = append(failures, parsed.Host+": 清单无效")
			continue
		}
		download := firstMirror(source, value.WindowsX64)
		state := "current"
		message := "当前已是最新版本"
		if newer(value.Version, s.current) {
			state, message = "available", "发现可用更新"
		}
		result := model.UpdateStatus{State: state, Current: s.current, Latest: value.Version, PublishedAt: value.PublishedAt, Source: source, DownloadURL: download, SHA256: strings.ToLower(value.WindowsX64.SHA256), Size: value.WindowsX64.Size, Message: message}
		s.mu.Lock()
		s.status, s.asset = result, value.WindowsX64
		s.mu.Unlock()
		return result
	}
	result := model.UpdateStatus{State: "error", Current: s.current, Message: "所有更新源均不可用：" + strings.Join(failures, "；")}
	s.mu.Lock()
	s.status = result
	s.mu.Unlock()
	return result
}

func firstMirror(source string, item asset) string {
	for _, value := range item.Mirrors {
		if parsed, err := url.Parse(value); err == nil && parsed.Scheme == "https" {
			return value
		}
	}
	base, err := url.Parse(source)
	if err != nil {
		return ""
	}
	base.Path = strings.TrimSuffix(base.Path, filepath.Base(base.Path)) + item.File
	base.RawQuery = ""
	base.Fragment = ""
	return base.String()
}

func (s *Service) Download(ctx context.Context) (model.UpdateStatus, error) {
	status := s.Status()
	if status.State != "available" || status.DownloadURL == "" {
		return status, errors.New("当前没有可下载更新")
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, status.DownloadURL, nil)
	response, err := s.client.Do(request)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return status, fmt.Errorf("download update: HTTP %d", response.StatusCode)
	}
	if err := os.MkdirAll(filepath.Join(s.directory, "updates"), 0o700); err != nil {
		return status, err
	}
	archive := filepath.Join(s.directory, "updates", "vrc-plus-plus-"+safeVersion(status.Latest)+".zip")
	temp := archive + ".part"
	file, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return status, err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, 512<<20))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(temp)
		return status, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return status, closeErr
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, status.SHA256) {
		_ = os.Remove(temp)
		return status, errors.New("更新包 SHA-256 校验失败")
	}
	if err := os.Rename(temp, archive); err != nil {
		return status, err
	}
	staged, err := extractExecutable(archive, filepath.Join(s.directory, "updates", "pending"))
	if err != nil {
		return status, err
	}
	status.State, status.Message = "ready", "更新已校验并暂存，点击安装后会自动重启"
	status.DownloadURL = staged
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
	return status, nil
}

func (s *Service) LaunchApply() error {
	status := s.Status()
	if status.State != "ready" {
		return errors.New("没有已暂存的更新")
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	helper := filepath.Join(s.directory, "updates", "vrc-plus-plus-update-helper.exe")
	if err := copyFile(current, helper); err != nil {
		return fmt.Errorf("prepare update helper: %w", err)
	}
	cmd := exec.Command(helper, "--update-helper", "--replace-from", status.DownloadURL, "--replace-target", current, "--wait-pid", strconv.Itoa(os.Getpid()))
	cmd.Stdout, cmd.Stderr = nil, nil
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = hiddenProcessAttributes()
	}
	return cmd.Start()
}

func RunHelper(source, target string, pid int) error {
	if source == "" || target == "" || filepath.Base(target) == "." {
		return errors.New("invalid update helper paths")
	}
	for i := 0; i < 150; i++ {
		if !processExists(pid) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	backup := target + ".previous"
	_ = os.Remove(backup)
	_ = os.Rename(target, backup)
	if err := copyFile(source, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	cmd := exec.Command(target, "--open-browser=true")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = hiddenProcessAttributes()
	}
	return cmd.Start()
}

func extractExecutable(archive, directory string) (string, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	for _, item := range reader.File {
		if strings.EqualFold(filepath.Base(item.Name), "vrc-plus-plus.exe") {
			input, err := item.Open()
			if err != nil {
				return "", err
			}
			defer input.Close()
			target := filepath.Join(directory, "vrc-plus-plus.exe")
			output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
			if err != nil {
				return "", err
			}
			_, copyErr := io.Copy(output, io.LimitReader(input, 256<<20))
			closeErr := output.Close()
			if copyErr != nil {
				return "", copyErr
			}
			return target, closeErr
		}
	}
	return "", errors.New("更新包中缺少 vrc-plus-plus.exe")
}

func newer(candidate, current string) bool {
	candidateVersion, candidateOK := parseVersion(candidate)
	currentVersion, currentOK := parseVersion(current)
	if !candidateOK || !currentOK {
		return false
	}
	return compareVersion(candidateVersion, currentVersion) > 0
}

type semanticVersion struct {
	major, minor, patch int
	prerelease          []string
}

func parseVersion(value string) (semanticVersion, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	value = strings.SplitN(value, "+", 2)[0]
	coreAndPrerelease := strings.SplitN(value, "-", 2)
	core := strings.Split(coreAndPrerelease[0], ".")
	if len(core) != 3 {
		return semanticVersion{}, false
	}
	parts := make([]int, 3)
	for index, raw := range core {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || raw == "" {
			return semanticVersion{}, false
		}
		parts[index] = parsed
	}
	result := semanticVersion{major: parts[0], minor: parts[1], patch: parts[2]}
	if len(coreAndPrerelease) == 2 {
		if coreAndPrerelease[1] == "" {
			return semanticVersion{}, false
		}
		result.prerelease = strings.Split(coreAndPrerelease[1], ".")
		for _, identifier := range result.prerelease {
			if identifier == "" {
				return semanticVersion{}, false
			}
		}
	}
	return result, true
}

func compareVersion(left, right semanticVersion) int {
	for _, pair := range [][2]int{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if len(left.prerelease) == 0 && len(right.prerelease) > 0 {
		return 1
	}
	if len(left.prerelease) > 0 && len(right.prerelease) == 0 {
		return -1
	}
	for index := 0; index < len(left.prerelease) && index < len(right.prerelease); index++ {
		leftID, rightID := left.prerelease[index], right.prerelease[index]
		leftNumber, leftErr := strconv.Atoi(leftID)
		rightNumber, rightErr := strconv.Atoi(rightID)
		switch {
		case leftErr == nil && rightErr == nil:
			if leftNumber < rightNumber {
				return -1
			}
			if leftNumber > rightNumber {
				return 1
			}
		case leftErr == nil:
			return -1
		case rightErr == nil:
			return 1
		case leftID < rightID:
			return -1
		case leftID > rightID:
			return 1
		}
	}
	if len(left.prerelease) < len(right.prerelease) {
		return -1
	}
	if len(left.prerelease) > len(right.prerelease) {
		return 1
	}
	return 0
}

func safeVersion(value string) string {
	return strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(value)
}
func copyFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, input)
	if closeErr := output.Close(); err == nil {
		err = closeErr
	}
	return err
}
