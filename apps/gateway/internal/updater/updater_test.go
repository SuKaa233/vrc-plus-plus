package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

func TestCheckFallsBackToSecondSource(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"version":"0.2.0","publishedAt":"2026-08-15T00:00:00Z","windowsX64":{"file":"VRC++-Setup-0.2.0.exe","size":3,"mirrors":[]}}`)
	}))
	defer good.Close()
	t.Setenv("VRC_HARBOR_UPDATE_URLS", "http://127.0.0.1:1/unavailable;"+good.URL+"/update-manifest.json")
	service := New("0.1.0", t.TempDir(), &http.Client{Timeout: time.Second}, nil)
	status := service.Check(context.Background())
	if status.State != "available" || status.Source != good.URL+"/update-manifest.json" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestDownloadStagesInstallerForLocalDevelopment(t *testing.T) {
	installer := []byte("development installer")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/update-manifest.json" {
			writer.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(writer, `{"version":"0.2.0","publishedAt":"2026-08-15T00:00:00Z","windowsX64":{"file":"VRC++-Setup-0.2.0.exe","size":%d,"mirrors":[]}}`, len(installer))
			return
		}
		_, _ = writer.Write(installer)
	}))
	defer server.Close()
	t.Setenv("VRC_HARBOR_UPDATE_URLS", server.URL+"/update-manifest.json")
	t.Setenv("VRC_PLUS_PLUS_ALLOW_UNSIGNED_UPDATES", "1")
	service := New("0.1.0", t.TempDir(), &http.Client{Timeout: time.Second}, nil)
	if status := service.Check(context.Background()); status.State != "available" {
		t.Fatalf("expected available update, got %#v", status)
	}
	status, err := service.Download(context.Background())
	if err != nil || status.State != "ready" {
		t.Fatalf("download failed: status=%#v err=%v", status, err)
	}
	if err := service.LaunchApply(); err == nil {
		t.Fatal("a development text fixture must not launch as an installer")
	}
}

func TestNewerUsesSemanticVersionOrdering(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"0.9.0-beta.2", "0.9.0-beta.1", true},
		{"0.9.0", "0.9.0-beta.1", true},
		{"1.0.0", "0.9.0-beta.1", true},
		{"0.9.0-beta.1", "0.9.0-beta.1", false},
		{"0.8.9", "0.9.0-beta.1", false},
		{"0.9.0-alpha.9", "0.9.0-beta.1", false},
		{"not-a-version", "0.9.0-beta.1", false},
	}
	for _, test := range tests {
		if got := newer(test.candidate, test.current); got != test.want {
			t.Errorf("newer(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

func TestBackgroundChecksNotifyEachVersionOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, `{"version":"0.2.0","publishedAt":"2026-08-19T00:00:00Z","releaseNotes":["new"],"windowsX64":{"file":"VRC++-Setup-0.2.0.exe","size":3,"mirrors":[]}}`)
	}))
	defer server.Close()
	t.Setenv("VRC_HARBOR_UPDATE_URLS", server.URL+"/update-manifest.json")
	service := New("0.1.0", t.TempDir(), &http.Client{Timeout: time.Second}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	notified := make(chan string, 3)
	service.StartBackground(ctx, 5*time.Millisecond, 10*time.Millisecond, func(status model.UpdateStatus) { notified <- status.Latest })
	select {
	case version := <-notified:
		if version != "0.2.0" {
			t.Fatalf("notified version = %q", version)
		}
	case <-time.After(time.Second):
		t.Fatal("background update notification was not delivered")
	}
	select {
	case version := <-notified:
		t.Fatalf("same version was notified twice: %s", version)
	case <-time.After(60 * time.Millisecond):
	}
}
