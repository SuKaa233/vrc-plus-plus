package pipeline

import (
	"log/slog"
	"testing"
	"time"

	"github.com/local/vrc-web-companion/gateway/internal/events"
)

func TestHandleMessageNormalizesStringContent(t *testing.T) {
	bus := events.NewBus()
	channel, unsubscribe := bus.Subscribe()
	defer unsubscribe()
	manager := New(nil, bus, slog.Default(), "wss://example.invalid")
	manager.handleMessage([]byte(`{"type":"friend-online","content":"{\"userId\":\"usr_test\"}"}`))
	select {
	case event := <-channel:
		if event.Type != "vrc.friend-online" || string(event.Content) != `{"userId":"usr_test"}` {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("normalized event was not published")
	}
}
