package vrchat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/local/vrc-web-companion/gateway/internal/model"
	"github.com/local/vrc-web-companion/gateway/internal/storage"
)

type testProtector struct{}

func (testProtector) Name() string { return "test" }
func (testProtector) Protect(value []byte) ([]byte, error) {
	return append([]byte("encrypted:"), value...), nil
}
func (testProtector) Unprotect(value []byte) ([]byte, error) {
	return bytes.TrimPrefix(value, []byte("encrypted:")), nil
}

func TestLoginPersistsAndRestoresCookie(t *testing.T) {
	t.Parallel()
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/auth/user" {
			http.NotFound(writer, request)
			return
		}
		if strings.HasPrefix(request.Header.Get("Authorization"), "Basic ") {
			loginCalls.Add(1)
			http.SetCookie(writer, &http.Cookie{Name: "auth", Value: "authcookie_test", Path: "/", HttpOnly: true})
		} else if cookie, err := request.Cookie("auth"); err != nil || cookie.Value != "authcookie_test" {
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte(`{"error":{"message":"Missing Credentials"}}`))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]string{
			"id": "usr_test", "displayName": "测试用户", "currentAvatarThumbnailImageUrl": "https://example.invalid/avatar.png",
		})
	}))
	defer server.Close()

	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	client, err := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := client.Login(ctx, "name@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != model.SessionAuthenticated || state.User == nil || state.User.ID != "usr_test" {
		t.Fatalf("Login() = %#v", state)
	}

	restarted, err := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restarted.Restore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != model.SessionAuthenticated {
		t.Fatalf("Restore() status = %s", restored.Status)
	}
	if loginCalls.Load() != 1 {
		t.Fatalf("credential login calls = %d, want 1", loginCalls.Load())
	}
}

func TestLoginReturnsTwoFactorMethods(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.SetCookie(writer, &http.Cookie{Name: "auth", Value: "pending", Path: "/"})
		_ = json.NewEncoder(writer).Encode(map[string]any{"requiresTwoFactorAuth": []string{"totp", "otp"}})
	}))
	defer server.Close()
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	state, err := client.Login(context.Background(), "user", "password")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != model.SessionTwoFactorRequired || len(state.Methods) != 2 {
		t.Fatalf("Login() = %#v", state)
	}
}
