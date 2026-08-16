//go:build windows

package desktop

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	webview "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var ErrUnavailable = errors.New("Microsoft Edge WebView2 Runtime is unavailable")

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	showWindow          = user32.NewProc("ShowWindow")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
)

const swRestore = 9

const webView2ClientID = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

type Manager struct {
	show chan struct{}
}

func New() *Manager {
	return &Manager{show: make(chan struct{}, 1)}
}

func (m *Manager) Show() {
	select {
	case m.show <- struct{}{}:
	default:
	}
}

func (m *Manager) Run(ctx context.Context, target, dataDirectory string) error {
	if !webView2RuntimeAvailable() {
		return ErrUnavailable
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	firstWindow := true
	for {
		if !firstWindow {
			select {
			case <-ctx.Done():
				return nil
			case <-m.show:
			}
		}
		firstWindow = false

		window := webview.NewWithOptions(webview.WebViewOptions{
			Debug:     false,
			DataPath:  filepath.Join(dataDirectory, "webview2"),
			AutoFocus: true,
			WindowOptions: webview.WindowOptions{
				Title:  "VRC++",
				Width:  1440,
				Height: 900,
				Center: true,
			},
		})
		if window == nil {
			return ErrUnavailable
		}
		window.SetSize(960, 640, webview.HintMin)
		window.Navigate(target)

		windowDone := make(chan struct{})
		go m.controlWindow(ctx, window, windowDone)
		window.Run()
		close(windowDone)
	}
}

func webView2RuntimeAvailable() bool {
	paths := []string{
		`SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2ClientID,
		`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webView2ClientID,
	}
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, path := range paths {
			key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			version, _, valueErr := key.GetStringValue("pv")
			_ = key.Close()
			version = strings.TrimSpace(version)
			if valueErr == nil && version != "" && version != "0.0.0.0" {
				return true
			}
		}
	}
	return false
}

func (m *Manager) controlWindow(ctx context.Context, window webview.WebView, done <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			window.Dispatch(window.Destroy)
			return
		case <-done:
			return
		case <-m.show:
			window.Dispatch(func() {
				handle := uintptr(window.Window())
				_, _, _ = showWindow.Call(handle, swRestore)
				_, _, _ = setForegroundWindow.Call(handle)
			})
		}
	}
}
