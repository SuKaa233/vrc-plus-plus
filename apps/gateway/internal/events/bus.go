package events

import (
	"sync"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

type Bus struct {
	mu          sync.RWMutex
	subscribers map[chan model.DomainEvent]struct{}
}

func NewBus() *Bus {
	return &Bus{subscribers: make(map[chan model.DomainEvent]struct{})}
}

func (b *Bus) Subscribe() (<-chan model.DomainEvent, func()) {
	channel := make(chan model.DomainEvent, 32)
	b.mu.Lock()
	b.subscribers[channel] = struct{}{}
	b.mu.Unlock()
	return channel, func() {
		b.mu.Lock()
		if _, ok := b.subscribers[channel]; ok {
			delete(b.subscribers, channel)
			close(channel)
		}
		b.mu.Unlock()
	}
}

func (b *Bus) Publish(event model.DomainEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for channel := range b.subscribers {
		select {
		case channel <- event:
		default:
			// A slow browser must not block the upstream pipeline reader.
		}
	}
}
