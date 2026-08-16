package localapi

import (
	"bufio"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/events"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/pipeline"
)

func newSecurityTestServer(t *testing.T) *Server {
	t.Helper()
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("ok")},
	}
	server, err := New(Config{
		AppName: "Test", Version: "test", StaticFS: fs.FS(assets), SecurityName: "test",
	}, nil, nil, events.NewBus(), pipeline.New(nil, events.NewBus(), slog.Default(), "wss://example.invalid"), nil, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestRejectsNonLoopbackRequest(t *testing.T) {
	server := newSecurityTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/local/v1/bootstrap", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestRejectsUntrustedOrigin(t *testing.T) {
	server := newSecurityTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/local/v1/bootstrap", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestDoesNotServeDevelopmentDocumentation(t *testing.T) {
	server := newSecurityTestServer(t)
	for _, path := range []string{"/docs/implementation-blueprint.md", "/implementation-blueprint.md"} {
		request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1"+path, nil)
		request.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, response.Code)
		}
	}
}

func TestRejectsMissingCSRFToken(t *testing.T) {
	server := newSecurityTestServer(t)
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/local/v1/auth/login", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestEventStreamStartsWithConnectionComment(t *testing.T) {
	server := newSecurityTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(httpServer.URL + "/local/v1/events/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if line != ": connected\n" {
		t.Fatalf("stream prelude = %q", line)
	}
}
