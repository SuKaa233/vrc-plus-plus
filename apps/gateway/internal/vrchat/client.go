package vrchat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/security"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
	"golang.org/x/sync/singleflight"
)

const defaultSessionID = "default"

var ErrUserAgentNotConfigured = errors.New("正式登录前请设置 VRC_HARBOR_USER_AGENT，并填写可联系的邮箱或网址")
var ErrInvalidRequest = errors.New("invalid VRChat request")

type persistedCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path,omitempty"`
	Domain   string    `json:"domain,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"httpOnly,omitempty"`
}

type Client struct {
	baseURL       *url.URL
	userAgent     string
	http          *http.Client
	jar           http.CookieJar
	store         *storage.Store
	protector     security.Protector
	limiter       *requestLimiter
	requests      singleflight.Group
	sessionMu     sync.RWMutex
	currentUserID string
	network       model.NetworkConfig
}

func NewClient(baseURL, userAgent string, store *storage.Store, protector security.Protector) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/") + "/")
	if err != nil {
		return nil, fmt.Errorf("parse VRChat base URL: %w", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	client := &Client{
		baseURL:   parsed,
		userAgent: userAgent,
		jar:       jar,
		store:     store,
		protector: protector,
		limiter:   newRequestLimiter(1, 5),
	}
	client.http = &http.Client{
		Timeout:   15 * time.Second,
		Jar:       jar,
		Transport: mustDefaultTransport(),
	}
	client.network = model.NetworkConfig{Mode: "system"}
	return client, nil
}

func mustDefaultTransport() *http.Transport {
	transport, _ := buildTransport(model.NetworkConfig{Mode: "system"})
	return transport
}

func (c *Client) BaseURL() string   { return strings.TrimRight(c.baseURL.String(), "/") }
func (c *Client) UserAgent() string { return c.userAgent }

func (c *Client) Restore(ctx context.Context) (model.SessionState, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if err := c.loadCookies(ctx); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.currentUserID = ""
			return model.SessionState{Status: model.SessionAnonymous}, nil
		}
		return model.SessionState{Status: model.SessionUnavailable}, err
	}
	state, err := c.currentUser(ctx, "")
	if err != nil {
		var upstream *UpstreamError
		if errors.As(err, &upstream) && upstream.StatusCode == http.StatusUnauthorized {
			return model.SessionState{Status: model.SessionAnonymous, Message: "登录会话已过期"}, nil
		}
		return model.SessionState{Status: model.SessionUnavailable, Message: "已保留本地会话，请稍后重试"}, err
	}
	if state.Status == model.SessionAuthenticated && state.User != nil {
		c.currentUserID = state.User.ID
	}
	return state, nil
}

func (c *Client) Login(ctx context.Context, username, password string) (model.SessionState, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if strings.Contains(c.userAgent, ".invalid") {
		return model.SessionState{}, ErrUserAgentNotConfigured
	}
	if strings.TrimSpace(username) == "" || password == "" {
		return model.SessionState{}, errors.New("用户名和密码不能为空")
	}
	c.clearCookies()
	c.currentUserID = ""
	encodedUser := url.QueryEscape(strings.TrimSpace(username))
	encodedPassword := url.QueryEscape(password)
	authorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(encodedUser+":"+encodedPassword))
	state, err := c.currentUser(ctx, authorization)
	if err != nil {
		c.clearCookies()
		return model.SessionState{}, err
	}
	if err := c.persistCookies(ctx); err != nil {
		return model.SessionState{}, fmt.Errorf("安全保存登录会话失败: %w", err)
	}
	if state.Status == model.SessionAuthenticated && state.User != nil {
		c.currentUserID = state.User.ID
		_ = c.store.SaveProfile(ctx, state.User.ID, state.User.DisplayName, state.User.CurrentAvatarThumbnailImageURL)
	}
	return state, nil
}

func (c *Client) VerifyTwoFactor(ctx context.Context, method, code string) (model.SessionState, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	endpoint := map[string]string{
		"totp":     "auth/twofactorauth/totp/verify",
		"emailOtp": "auth/twofactorauth/emailotp/verify",
		"otp":      "auth/twofactorauth/otp/verify",
	}[method]
	if endpoint == "" {
		return model.SessionState{}, errors.New("不支持的二步验证方式")
	}
	if strings.TrimSpace(code) == "" {
		return model.SessionState{}, errors.New("验证码不能为空")
	}
	body, _ := json.Marshal(map[string]string{"code": strings.TrimSpace(code)})
	response, err := c.do(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)), "")
	if err != nil {
		return model.SessionState{}, err
	}
	response.Body.Close()
	if err := c.persistCookies(ctx); err != nil {
		return model.SessionState{}, fmt.Errorf("安全保存二步验证会话失败: %w", err)
	}
	state, err := c.currentUser(ctx, "")
	if err == nil && state.User != nil {
		c.currentUserID = state.User.ID
		_ = c.store.SaveProfile(ctx, state.User.ID, state.User.DisplayName, state.User.CurrentAvatarThumbnailImageURL)
	}
	return state, err
}

func (c *Client) Logout(ctx context.Context) (model.SessionState, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	response, err := c.do(ctx, http.MethodPut, "logout", strings.NewReader("{}"), "")
	if response != nil {
		response.Body.Close()
	}
	c.clearCookies()
	c.currentUserID = ""
	deleteErr := c.store.DeleteSession(ctx, defaultSessionID)
	if deleteErr != nil {
		return model.SessionState{}, deleteErr
	}
	if err != nil {
		return model.SessionState{Status: model.SessionAnonymous, Message: "本地会话已清除，上游退出请求未确认"}, nil
	}
	return model.SessionState{Status: model.SessionAnonymous}, nil
}

func (c *Client) Probe(ctx context.Context, name, endpoint string) model.ProbeResult {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	started := time.Now()
	response, err := c.do(ctx, http.MethodGet, endpoint, nil, "")
	result := model.ProbeResult{Name: name, CheckedAt: time.Now().UTC()}
	result.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		var upstream *UpstreamError
		if errors.As(err, &upstream) {
			result.StatusCode = upstream.StatusCode
			if upstream.StatusCode == http.StatusForbidden || upstream.StatusCode == http.StatusTooManyRequests {
				result.State = model.CheckDegraded
				result.Detail = fmt.Sprintf("上游返回 %d，可能受线路、Cloudflare 或限流影响", upstream.StatusCode)
				return result
			}
		}
		result.State = model.CheckError
		result.Detail = sanitizeError(err)
		return result
	}
	defer response.Body.Close()
	result.StatusCode = response.StatusCode
	result.State = model.CheckOK
	result.Detail = fmt.Sprintf("HTTP %d", response.StatusCode)
	return result
}

func (c *Client) currentUser(ctx context.Context, authorization string) (model.SessionState, error) {
	response, err := c.do(ctx, http.MethodGet, "auth/user", nil, authorization)
	if err != nil {
		return model.SessionState{}, err
	}
	defer response.Body.Close()
	var payload struct {
		ID                             string   `json:"id"`
		DisplayName                    string   `json:"displayName"`
		CurrentAvatarThumbnailImageURL string   `json:"currentAvatarThumbnailImageUrl"`
		RequiresTwoFactorAuth          []string `json:"requiresTwoFactorAuth"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return model.SessionState{}, fmt.Errorf("无法解析 VRChat 登录响应: %w", err)
	}
	if len(payload.RequiresTwoFactorAuth) > 0 {
		return model.SessionState{Status: model.SessionTwoFactorRequired, Methods: payload.RequiresTwoFactorAuth}, nil
	}
	if payload.ID == "" {
		return model.SessionState{}, errors.New("VRChat 登录响应缺少用户标识")
	}
	return model.SessionState{Status: model.SessionAuthenticated, User: &model.User{
		ID:                             payload.ID,
		DisplayName:                    payload.DisplayName,
		CurrentAvatarThumbnailImageURL: payload.CurrentAvatarThumbnailImageURL,
	}}, nil
}

type UpstreamError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *UpstreamError) Error() string {
	if e.StatusCode == http.StatusUnauthorized {
		return "VRChat 登录凭据无效或会话已过期"
	}
	if e.StatusCode == http.StatusTooManyRequests {
		return "VRChat 请求过于频繁，请稍后再试"
	}
	if e.StatusCode == http.StatusForbidden {
		if e.Message != "" {
			return fmt.Sprintf("VRChat 拒绝请求：%s", e.Message)
		}
		return "VRChat 拒绝了请求，请确认好友关系、实例权限或功能设置"
	}
	if e.Message != "" {
		return fmt.Sprintf("VRChat 返回 %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("VRChat 返回 HTTP %d", e.StatusCode)
}

func (c *Client) do(ctx context.Context, method, endpoint string, body io.Reader, authorization string) (*http.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("等待安全请求间隔: %w", err)
	}
	relative, err := url.Parse(strings.TrimLeft(endpoint, "/"))
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL.ResolveReference(relative).String(), body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("VRChat 网络请求失败: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		message := readErrorMessage(response.Body)
		retryAfter := parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
		if response.StatusCode == http.StatusTooManyRequests {
			c.limiter.Block(retryAfter)
		}
		return nil, &UpstreamError{StatusCode: response.StatusCode, Message: message, RetryAfter: retryAfter}
	}
	return response, nil
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(value + "s"); err == nil {
		return max(0, seconds)
	}
	if retryAt, err := http.ParseTime(value); err == nil && retryAt.After(now) {
		return retryAt.Sub(now)
	}
	return 0
}

func readErrorMessage(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, 64<<10))
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &payload) == nil && payload.Error.Message != "" {
		return strings.Trim(payload.Error.Message, `"`)
	}
	return ""
}

func sanitizeError(err error) string {
	message := err.Error()
	if len(message) > 180 {
		message = message[:180] + "…"
	}
	return message
}

func (c *Client) persistCookies(ctx context.Context) error {
	cookies := c.jar.Cookies(c.baseURL)
	if len(cookies) == 0 {
		return errors.New("VRChat 未返回可保存的会话 Cookie")
	}
	persisted := make([]persistedCookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie.Name != "auth" && cookie.Name != "twoFactorAuth" {
			continue
		}
		persisted = append(persisted, persistedCookie{
			Name: cookie.Name, Value: cookie.Value, Path: cookie.Path,
			Domain: cookie.Domain, Expires: cookie.Expires,
			Secure: cookie.Secure, HTTPOnly: cookie.HttpOnly,
		})
	}
	if len(persisted) == 0 {
		return errors.New("VRChat 会话 Cookie 不完整")
	}
	plain, err := json.Marshal(persisted)
	if err != nil {
		return err
	}
	encrypted, err := c.protector.Protect(plain)
	for index := range plain {
		plain[index] = 0
	}
	if err != nil {
		return err
	}
	return c.store.SaveSession(ctx, defaultSessionID, encrypted)
}

func (c *Client) loadCookies(ctx context.Context) error {
	encrypted, err := c.store.LoadSession(ctx, defaultSessionID)
	if err != nil {
		return err
	}
	plain, err := c.protector.Unprotect(encrypted)
	if err != nil {
		return err
	}
	defer func() {
		for index := range plain {
			plain[index] = 0
		}
	}()
	var persisted []persistedCookie
	if err := json.Unmarshal(plain, &persisted); err != nil {
		return err
	}
	cookies := make([]*http.Cookie, 0, len(persisted))
	for _, cookie := range persisted {
		cookies = append(cookies, &http.Cookie{
			Name: cookie.Name, Value: cookie.Value, Path: cookie.Path,
			Domain: cookie.Domain, Expires: cookie.Expires,
			Secure: cookie.Secure, HttpOnly: cookie.HTTPOnly,
		})
	}
	c.jar.SetCookies(c.baseURL, cookies)
	return nil
}

func (c *Client) clearCookies() {
	jar, _ := cookiejar.New(nil)
	c.jar = jar
	c.http.Jar = jar
}
