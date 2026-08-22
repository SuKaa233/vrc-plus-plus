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

type testReader struct {
	instance model.Instance
	friends  model.DataEnvelope[model.Friend]
}

func (reader *testReader) GetInstance(context.Context, string) (model.Instance, error) {
	return reader.instance, nil
}
func (reader *testReader) ListFriends(context.Context) (model.DataEnvelope[model.Friend], error) {
	return reader.friends, nil
}

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
	service := New(filepath.Join(t.TempDir(), "guardian-state"), testProtector{}, bus, nil, func(target string) error { launched = target; return nil }, nil, slog.Default())
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
	service := New(filepath.Join(t.TempDir(), "guardian-state"), testProtector{}, events.NewBus(), nil, nil, nil, slog.Default())
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
	first := New(path, testProtector{}, events.NewBus(), nil, nil, nil, slog.Default())
	first.now = func() time.Time { return now }
	first.process = func() bool { return true }
	first.Start(context.Background())
	first.handleEvent(model.DomainEvent{Type: "game.location", ObservedAt: now.Add(-10 * time.Minute), SensitiveLocation: "wrld_restart:42~region(eu)", Content: []byte(`{"location":"wrld_restart:42~region(eu)"}`)})
	first.Stop()

	second := New(path, testProtector{}, events.NewBus(), nil, nil, nil, slog.Default())
	second.now = func() time.Time { return now }
	second.process = func() bool { return false }
	second.Start(context.Background())
	defer second.Stop()
	status := second.Status()
	if status.Current != nil || status.Last == nil || status.ExitKind != "interrupted" || !status.CanResume {
		t.Fatalf("unexpected restarted status: %#v", status)
	}
}

func TestSlotWatchDetectsAvailableSpace(t *testing.T) {
	now := time.Date(2026, 8, 22, 13, 0, 0, 0, time.UTC)
	reader := &testReader{instance: model.Instance{WorldID: "wrld_test", UserCount: 39, Capacity: 40, Active: true, Full: false}}
	service := New(filepath.Join(t.TempDir(), "guardian-state"), testProtector{}, events.NewBus(), reader, nil, nil, slog.Default())
	service.now = func() time.Time { return now }
	if err := service.StartSlotWatch("wrld_test:123~region(jp)", "Test World", time.Hour); err != nil {
		t.Fatal(err)
	}
	service.checkWatches(context.Background())
	status := service.Status()
	if status.SlotWatch == nil || status.SlotWatch.State != "available" || status.SlotWatch.UserCount != 39 || status.SlotWatch.Capacity != 40 {
		t.Fatalf("unexpected slot watch: %#v", status.SlotWatch)
	}
}

func TestMigrationWatchGroupsFriendsAtNewLocation(t *testing.T) {
	now := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	reader := &testReader{friends: model.DataEnvelope[model.Friend]{Items: []model.Friend{
		{ID: "usr_a", DisplayName: "A", Online: true, Location: "wrld_new:7~region(jp)"},
		{ID: "usr_b", DisplayName: "B", Online: true, Location: "wrld_new:7~region(jp)"},
		{ID: "usr_other", DisplayName: "Other", Online: true, Location: "wrld_new:7~region(jp)"},
	}}}
	service := New(filepath.Join(t.TempDir(), "guardian-state"), testProtector{}, events.NewBus(), reader, nil, nil, slog.Default())
	service.now = func() time.Time { return now }
	service.state.Last = &Session{Location: "wrld_old:1", Participants: []Participant{{UserID: "usr_a", DisplayName: "A"}, {UserID: "usr_b", DisplayName: "B"}, {DisplayName: "No ID"}}}
	if err := service.StartMigrationWatch(30 * time.Minute); err != nil {
		t.Fatal(err)
	}
	service.checkWatches(context.Background())
	status := service.Status()
	if status.Migration == nil || len(status.Migration.Tracked) != 2 || len(status.Migration.Destinations) != 1 || len(status.Migration.Destinations[0].People) != 2 {
		t.Fatalf("unexpected migration watch: %#v", status.Migration)
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
