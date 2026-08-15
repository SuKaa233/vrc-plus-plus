package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckFallsBackToSecondSource(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"version":"0.2.0","publishedAt":"2026-08-15T00:00:00Z","windowsX64":{"file":"release.zip","sha256":"abc","size":3,"mirrors":[]}}`)
	}))
	defer good.Close()
	t.Setenv("VRC_HARBOR_UPDATE_URLS", "http://127.0.0.1:1/unavailable;"+good.URL+"/update-manifest.json")
	service := New("0.1.0", t.TempDir(), &http.Client{Timeout: time.Second}, nil)
	status := service.Check(context.Background())
	if status.State != "available" || status.Source != good.URL+"/update-manifest.json" {
		t.Fatalf("unexpected status: %#v", status)
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
