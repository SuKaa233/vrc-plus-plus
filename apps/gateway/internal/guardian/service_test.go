package guardian

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/events"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

type testProtector struct{}

func (testProtector) Protect(value []byte) ([]byte, error) {
	return append([]byte("protected:"), value...), nil
}
func (testProtector) Unprotect(value []byte) ([]byte, error) { return value[len("protected:"):], nil }
func (testProtector) Name() string                           { return "test" }

func TestGuardianRecordsAndResumesUnexpectedSession(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	running := true
	launched := ""
	bus := events.NewBus()
	service := New(filepath.Join(t.TempDir(), "guardian-state"), testProtector{}, bus, func(target string) error { launched = target; return nil }, nil, slog.Default())
	service.now = func() time.Time { return now }
	service.process = func() bool { return running }
	service.Start(context.Background())
	defer service.Stop()
	content, _ := json.Marshal(map[string]string{"location": "wrld_test:123~hidden(usr_owner)~region(jp)"})
	service.handleEvent(model.DomainEvent{Type: "game.location", ObservedAt: now.Add(-time.Minute), Content: content, SensitiveLocation: "wrld_test:123~hidden(usr_owner)~region(jp)~nonce(secret)"})
	running = false
	service.observeProcess()
	status := service.Status()
	if status.State != "recovery" || !status.CanResume || status.Last == nil || status.Last.WorldID != "wrld_test" || status.Last.Region != "jp" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if !strings.Contains(status.Last.Location, "nonce(secret)") {
		t.Fatalf("status did not expose the local full location: %q", status.Last.Location)
	}
	if _, err := service.Resume(); err != nil {
		t.Fatal(err)
	}
	if launched == "" || !containsAll(launched, "worldId=wrld_test", "instanceId=123", "nonce%28secret%29", "launch=1") {
		t.Fatalf("launch URL = %q", launched)
	}
}

func TestGuardianMarksCleanExit(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	running := true
	service := New(filepath.Join(t.TempDir(), "guardian-state"), testProtector{}, events.NewBus(), nil, nil, slog.Default())
	service.now = func() time.Time { return now }
	service.process = func() bool { return running }
	service.Start(context.Background())
	defer service.Stop()
	service.handleEvent(model.DomainEvent{Type: "game.location", ObservedAt: now, SensitiveLocation: "wrld_test:1~region(us)", Content: []byte(`{"location":"wrld_test:1~region(us)"}`)})
	service.handleEvent(model.DomainEvent{Type: "game.quit-clean", ObservedAt: now})
	running = false
	service.observeProcess()
	if status := service.Status(); status.ExitKind != "clean" || status.State != "ready" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestGuardianRecoversPersistedCurrentSessionAfterGatewayRestart(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "guardian-state")
	first := New(path, testProtector{}, events.NewBus(), nil, nil, slog.Default())
	first.now = func() time.Time { return now }
	first.process = func() bool { return true }
	first.Start(context.Background())
	first.handleEvent(model.DomainEvent{Type: "game.location", ObservedAt: now.Add(-10 * time.Minute), SensitiveLocation: "wrld_restart:42~region(eu)", Content: []byte(`{"location":"wrld_restart:42~region(eu)"}`)})
	first.Stop()

	second := New(path, testProtector{}, events.NewBus(), nil, nil, slog.Default())
	second.now = func() time.Time { return now }
	second.process = func() bool { return false }
	second.Start(context.Background())
	defer second.Stop()
	status := second.Status()
	if status.Current != nil || status.Last == nil || status.ExitKind != "interrupted" || !status.CanResume {
		t.Fatalf("unexpected restarted status: %#v", status)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
