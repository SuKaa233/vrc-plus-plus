package vrchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/local/vrc-web-companion/gateway/internal/model"
	"github.com/local/vrc-web-companion/gateway/internal/storage"
	"golang.org/x/net/proxy"
)

const networkSettingKey = "network.config.v1"

func LoadNetworkConfig(ctx context.Context, store *storage.Store) (model.NetworkConfig, error) {
	value, err := store.LoadSetting(ctx, networkSettingKey)
	if errors.Is(err, storage.ErrNotFound) {
		return model.NetworkConfig{Mode: "system"}, nil
	}
	if err != nil {
		return model.NetworkConfig{}, err
	}
	var config model.NetworkConfig
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return model.NetworkConfig{}, fmt.Errorf("parse network setting: %w", err)
	}
	if _, err := buildTransport(config); err != nil {
		return model.NetworkConfig{Mode: "system"}, nil
	}
	return config, nil
}

func (c *Client) ApplyNetworkConfig(ctx context.Context, config model.NetworkConfig) (model.NetworkState, error) {
	transport, err := buildTransport(config)
	if err != nil {
		return model.NetworkState{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	config.Mode = strings.ToLower(strings.TrimSpace(config.Mode))
	config.ProxyURL = strings.TrimSpace(config.ProxyURL)
	payload, err := json.Marshal(config)
	if err != nil {
		return model.NetworkState{}, err
	}

	c.sessionMu.Lock()
	oldTransport := c.http.Transport
	c.http.Transport = transport
	c.network = config
	c.sessionMu.Unlock()
	if closer, ok := oldTransport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
	if err := c.store.SaveSetting(ctx, networkSettingKey, string(payload)); err != nil {
		return model.NetworkState{}, err
	}
	return networkState(config), nil
}

func (c *Client) NetworkState() model.NetworkState {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return networkState(c.network)
}

func (c *Client) HTTPClientSnapshot(timeout time.Duration) *http.Client {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return &http.Client{Transport: c.http.Transport, Timeout: timeout}
}

func buildTransport(config model.NetworkConfig) (*http.Transport, error) {
	mode := strings.ToLower(strings.TrimSpace(config.Mode))
	if mode == "" {
		mode = "system"
	}
	transport := &http.Transport{
		MaxIdleConns: 30, MaxIdleConnsPerHost: 8, IdleConnTimeout: 90 * time.Second,
		TLSHandshakeTimeout: 12 * time.Second, ResponseHeaderTimeout: 18 * time.Second,
	}
	switch mode {
	case "system":
		transport.Proxy = http.ProxyFromEnvironment
	case "direct":
		transport.Proxy = nil
	case "http":
		parsed, err := parseProxyURL(config.ProxyURL, "http", "https")
		if err != nil {
			return nil, err
		}
		if err := validateProxyTarget(parsed); err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(parsed)
	case "socks5":
		parsed, err := parseProxyURL(config.ProxyURL, "socks5", "socks5h")
		if err != nil {
			return nil, err
		}
		if err := validateProxyTarget(parsed); err != nil {
			return nil, err
		}
		dialer, err := proxy.SOCKS5("tcp", parsed.Host, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("create SOCKS5 proxy: %w", err)
		}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			type contextDialer interface {
				DialContext(context.Context, string, string) (net.Conn, error)
			}
			if value, ok := dialer.(contextDialer); ok {
				return value.DialContext(ctx, network, address)
			}
			return dialer.Dial(network, address)
		}
	default:
		return nil, errors.New("network mode must be system, direct, http, or socks5")
	}
	return transport, nil
}

func validateProxyTarget(parsed *url.URL) error {
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" || parsed.Port() == "" {
		return errors.New("proxy URL must include a host and port")
	}
	ip := net.ParseIP(host)
	if strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback()) {
		return nil
	}
	return errors.New("custom proxy must use localhost or a loopback IP")
}

func parseProxyURL(raw string, schemes ...string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return nil, errors.New("proxy URL is invalid")
	}
	if parsed.User != nil {
		return nil, errors.New("proxy credentials are not stored; use a local proxy without URL credentials")
	}
	allowed := false
	for _, scheme := range schemes {
		allowed = allowed || strings.EqualFold(parsed.Scheme, scheme)
	}
	if !allowed {
		return nil, fmt.Errorf("proxy scheme must be %s", strings.Join(schemes, " or "))
	}
	return parsed, nil
}

func networkState(config model.NetworkConfig) model.NetworkState {
	state := model.NetworkState{NetworkConfig: config}
	switch config.Mode {
	case "direct":
		state.Label, state.Description = "直连", "不读取代理环境变量，直接访问 VRChat"
	case "http":
		state.Label, state.Description = "HTTP 代理", "REST、Pipeline 与图片共用自定义 HTTP 代理"
	case "socks5":
		state.Label, state.Description = "SOCKS5 代理", "REST、Pipeline 与图片共用本机 SOCKS5 代理"
	default:
		state.Mode = "system"
		state.Label, state.Description = "跟随系统", "读取 HTTP_PROXY、HTTPS_PROXY 与 NO_PROXY 环境变量"
	}
	return state
}
