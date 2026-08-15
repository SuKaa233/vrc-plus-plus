package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/vrc-web-companion/gateway/internal/vrchat"
	"golang.org/x/sync/singleflight"
)

const maxMediaBytes = 16 << 20

type Service struct {
	directory string
	client    *vrchat.Client
	logger    *slog.Logger
	requests  singleflight.Group
}

func (s *Service) Stats() (files, bytes int64, err error) {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bin") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return 0, 0, infoErr
		}
		files++
		bytes += info.Size()
	}
	return files, bytes, nil
}

func (s *Service) Clear() (files, bytes int64, err error) {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".bin") && !strings.HasSuffix(name, ".type") && !strings.HasSuffix(name, ".tmp") {
			continue
		}
		path := filepath.Join(s.directory, name)
		info, infoErr := entry.Info()
		if infoErr != nil {
			return files, bytes, infoErr
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return files, bytes, removeErr
		}
		if strings.HasSuffix(name, ".bin") {
			files++
			bytes += info.Size()
		}
	}
	return files, bytes, nil
}

func New(directory string, client *vrchat.Client, logger *slog.Logger) (*Service, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create media cache: %w", err)
	}
	return &Service{directory: directory, client: client, logger: logger}, nil
}

func (s *Service) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	rawURL := request.URL.Query().Get("url")
	parsed, err := validateMediaURL(rawURL)
	if err != nil {
		http.Error(writer, "不支持的图片地址", http.StatusBadRequest)
		return
	}
	key := mediaKey(parsed.String())
	path := filepath.Join(s.directory, key+".bin")
	contentTypePath := filepath.Join(s.directory, key+".type")
	if err := s.ensure(request.Context(), parsed, path, contentTypePath); err != nil {
		s.logger.Warn("media cache fetch failed", "host", parsed.Hostname(), "error", safeError(err))
		http.Error(writer, "图片暂时不可用", http.StatusBadGateway)
		return
	}
	contentType, _ := os.ReadFile(contentTypePath)
	if value := strings.TrimSpace(string(contentType)); value != "" {
		writer.Header().Set("Content-Type", value)
	}
	writer.Header().Set("Cache-Control", "private, max-age=86400, stale-if-error=604800")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	file, err := os.Open(path)
	if err != nil {
		http.Error(writer, "图片缓存读取失败", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.Error(writer, "图片缓存读取失败", http.StatusInternalServerError)
		return
	}
	http.ServeContent(writer, request, key, info.ModTime(), file)
}

func (s *Service) ensure(ctx context.Context, remote *url.URL, path, contentTypePath string) error {
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return nil
	}
	result := s.requests.DoChan(path, func() (any, error) {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return nil, nil
		}
		requestCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		httpClient := s.client.HTTPClientSnapshot(30 * time.Second)
		httpClient.CheckRedirect = func(request *http.Request, _ []*http.Request) error {
			_, err := validateMediaURL(request.URL.String())
			return err
		}
		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, remote.String(), nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", s.client.UserAgent())
		request.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/*;q=0.8")
		response, err := httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("image upstream returned HTTP %d", response.StatusCode)
		}
		contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
		if !strings.HasPrefix(contentType, "image/") || contentType == "image/svg+xml" {
			return nil, errors.New("upstream response is not a supported raster image")
		}
		if response.ContentLength > maxMediaBytes {
			return nil, errors.New("image exceeds local cache size limit")
		}
		temporary, err := os.CreateTemp(s.directory, "media-*.tmp")
		if err != nil {
			return nil, err
		}
		temporaryPath := temporary.Name()
		defer os.Remove(temporaryPath)
		written, copyErr := io.Copy(temporary, io.LimitReader(response.Body, maxMediaBytes+1))
		closeErr := temporary.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if written > maxMediaBytes {
			return nil, errors.New("image exceeds local cache size limit")
		}
		if err := os.Rename(temporaryPath, path); err != nil {
			return nil, err
		}
		if err := os.WriteFile(contentTypePath, []byte(contentType), 0o600); err != nil {
			return nil, err
		}
		return nil, nil
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-result:
		return result.Err
	}
}

func validateMediaURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" {
		return nil, errors.New("media URL must use HTTPS")
	}
	if parsed.User != nil {
		return nil, errors.New("media URL credentials are not allowed")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if !allowedHost(host, "vrchat.cloud") && !allowedHost(host, "vrcdn.cloud") && !allowedHost(host, "vrchat.com") {
		return nil, errors.New("media host is not allowlisted")
	}
	return parsed, nil
}

func allowedHost(host, suffix string) bool {
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

func mediaKey(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func safeError(err error) string {
	message := err.Error()
	if len(message) > 160 {
		message = message[:160]
	}
	return message
}
