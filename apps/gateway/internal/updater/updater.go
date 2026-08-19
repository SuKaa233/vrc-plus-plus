package updater

import (
	"context"
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

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

const maxInstallerSize = 512 << 20

type asset struct {
	File    string   `json:"file"`
	Size    int64    `json:"size"`
	Mirrors []string `json:"mirrors"`
}

type manifest struct {
	Version      string    `json:"version"`
	PublishedAt  time.Time `json:"publishedAt"`
	ReleaseNotes []string  `json:"releaseNotes,omitempty"`
	WindowsX64   asset     `json:"windowsX64"`
}

type Service struct {
	current, directory string
	sources            []string
	client             *http.Client
	clientMu           sync.RWMutex
	logger             *slog.Logger
	mu                 sync.RWMutex
	status             model.UpdateStatus
	asset              asset
	stagedInstaller    string
}

func (s *Service) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.clientMu.Lock()
	s.client = client
	s.clientMu.Unlock()
}

func (s *Service) httpClient() *http.Client {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	return s.client
}

func New(current, directory string, client *http.Client, logger *slog.Logger, builtInSources ...string) *Service {
	var sources []string
	for _, raw := range strings.Split(os.Getenv("VRC_HARBOR_UPDATE_URLS"), ";") {
		if value := strings.TrimSpace(raw); value != "" {
			sources = append(sources, value)
		}
	}
	for _, group := range builtInSources {
		for _, raw := range strings.Split(group, ";") {
			if value := strings.TrimSpace(raw); value != "" && !containsString(sources, value) {
				sources = append(sources, value)
			}
		}
	}
	defaultSource := defaultManifestSource(current)
	if !containsString(sources, defaultSource) {
		sources = append(sources, defaultSource)
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &Service{
		current:   current,
		directory: directory,
		sources:   sources,
		client:    client,
		logger:    logger,
		status:    model.UpdateStatus{State: "idle", Current: current, Message: "等待自动检查更新"},
	}
}

func defaultManifestSource(current string) string {
	if strings.Contains(current, "-") {
		return "https://github.com/SuKaa233/vrc-plus-plus/releases/download/update-beta/update-manifest.json"
	}
	return "https://github.com/SuKaa233/vrc-plus-plus/releases/latest/download/update-manifest.json"
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *Service) Status() model.UpdateStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Service) StartBackground(ctx context.Context, initialDelay, interval time.Duration, onAvailable func(model.UpdateStatus)) {
	go func() {
		lastNotifiedVersion := ""
		check := func() {
			status := s.Check(ctx)
			if status.State == "available" && status.Latest != "" && status.Latest != lastNotifiedVersion {
				lastNotifiedVersion = status.Latest
				if onAvailable != nil {
					onAvailable(status)
				}
			}
		}
		timer := time.NewTimer(initialDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			check()
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				check()
			}
		}
	}()
}

func (s *Service) Check(ctx context.Context) model.UpdateStatus {
	var failures []string
	for _, source := range s.sources {
		parsed, err := url.Parse(source)
		if err != nil || !allowedUpdateURL(parsed) {
			failures = append(failures, "无效更新源")
			continue
		}
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		response, err := s.httpClient().Do(request)
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
		if err := json.Unmarshal(data, &value); err != nil || value.Version == "" || value.WindowsX64.File == "" {
			failures = append(failures, parsed.Host+": 清单无效")
			continue
		}
		download := firstMirror(source, value.WindowsX64)
		if download == "" {
			failures = append(failures, parsed.Host+": 没有有效的安装程序地址")
			continue
		}
		state := "current"
		message := "当前已是最新版本"
		if newer(value.Version, s.current) {
			state, message = "available", "发现可用更新"
		}
		result := model.UpdateStatus{
			State: state, Current: s.current, Latest: value.Version, PublishedAt: value.PublishedAt,
			Source: source, DownloadURL: download, Size: value.WindowsX64.Size,
			ReleaseNotes: value.ReleaseNotes, Message: message, CheckedAt: time.Now().UTC(),
		}
		s.mu.Lock()
		if s.status.State == "ready" && s.status.Latest == result.Latest {
			result = s.status
			result.CheckedAt = time.Now().UTC()
		}
		s.status, s.asset = result, value.WindowsX64
		s.mu.Unlock()
		return result
	}
	result := model.UpdateStatus{State: "error", Current: s.current, Message: "暂时无法连接更新服务器", CheckedAt: time.Now().UTC()}
	if s.logger != nil && len(failures) > 0 {
		s.logger.Warn("all update sources failed", "failures", strings.Join(failures, "; "))
	}
	s.mu.Lock()
	s.status = result
	s.mu.Unlock()
	return result
}

func allowedUpdateURL(parsed *url.URL) bool {
	return parsed.Scheme == "https" || (parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost"))
}

func firstMirror(source string, item asset) string {
	for _, value := range item.Mirrors {
		if parsed, err := url.Parse(value); err == nil && allowedUpdateURL(parsed) {
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
	if !allowedUpdateURL(base) {
		return ""
	}
	return base.String()
}

func (s *Service) Download(ctx context.Context) (model.UpdateStatus, error) {
	status := s.Status()
	if status.State != "available" || status.DownloadURL == "" {
		return status, errors.New("当前没有可下载更新")
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, status.DownloadURL, nil)
	response, err := s.httpClient().Do(request)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return status, fmt.Errorf("download update: HTTP %d", response.StatusCode)
	}
	updatesDirectory := filepath.Join(s.directory, "updates")
	if err := os.MkdirAll(updatesDirectory, 0o700); err != nil {
		return status, err
	}
	installer := filepath.Join(updatesDirectory, "VRC++-Setup-"+safeVersion(status.Latest)+".exe")
	temp := installer + ".part"
	file, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return status, err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maxInstallerSize+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(temp)
		if copyErr != nil {
			return status, copyErr
		}
		return status, closeErr
	}
	if written > maxInstallerSize {
		_ = os.Remove(temp)
		return status, errors.New("更新安装程序超过 512 MB 限制")
	}
	if status.Size > 0 && written != status.Size {
		_ = os.Remove(temp)
		return status, fmt.Errorf("更新安装程序大小不匹配：期望 %d，实际 %d", status.Size, written)
	}
	if err := verifyInstaller(temp); err != nil {
		if os.Getenv("VRC_PLUS_PLUS_ALLOW_UNSIGNED_UPDATES") != "1" {
			_ = os.Remove(temp)
			return status, fmt.Errorf("安装程序签名验证失败：%w", err)
		}
		if s.logger != nil {
			s.logger.Warn("unsigned update accepted by local development override", "error", err)
		}
	}
	_ = os.Remove(installer)
	if err := os.Rename(temp, installer); err != nil {
		return status, err
	}
	status.State, status.Message = "ready", "更新已下载，点击后将自动安装并重启"
	s.mu.Lock()
	s.stagedInstaller = installer
	s.status = status
	s.mu.Unlock()
	return status, nil
}

func (s *Service) LaunchApply() error {
	status := s.Status()
	if status.State != "ready" {
		return errors.New("没有已暂存的更新")
	}
	s.mu.RLock()
	installer := s.stagedInstaller
	s.mu.RUnlock()
	if installer == "" {
		return errors.New("更新安装程序不存在")
	}
	if _, err := os.Stat(installer); err != nil {
		return fmt.Errorf("读取更新安装程序：%w", err)
	}
	cmd := exec.Command(installer, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/SP-", "/CURRENTUSER", "/CLOSEAPPLICATIONS", "/UPDATE=1")
	cmd.Stdout, cmd.Stderr = nil, nil
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = hiddenProcessAttributes()
	}
	return cmd.Start()
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
