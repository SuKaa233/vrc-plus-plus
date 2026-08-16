package events

import (
	"testing"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

func TestPublishAndUnsubscribe(t *testing.T) {
	bus := NewBus()
	channel, unsubscribe := bus.Subscribe()
	event := model.DomainEvent{ID: "evt-1", Type: "friend-online", ObservedAt: time.Now()}
	bus.Publish(event)
	select {
	case got := <-channel:
		if got.ID != event.ID {
			t.Fatalf("event id = %q", got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not published")
	}
	unsubscribe()
}
