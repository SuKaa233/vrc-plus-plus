package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/local/vrc-web-companion/gateway/internal/events"
	"github.com/local/vrc-web-companion/gateway/internal/model"
	"github.com/local/vrc-web-companion/gateway/internal/vrchat"
)

type Manager struct {
	client      *vrchat.Client
	bus         *events.Bus
	logger      *slog.Logger
	pipelineURL string

	mu      sync.RWMutex
	status  model.RealtimeStatus
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
	serial  atomic.Uint64
}

func New(client *vrchat.Client, bus *events.Bus, logger *slog.Logger, pipelineURL string) *Manager {
	return &Manager{
		client: client, bus: bus, logger: logger,
		pipelineURL: strings.TrimRight(pipelineURL, "/"),
		status:      model.RealtimeStatus{State: "disabled", Message: "sign in to enable realtime updates"},
	}
}

func (m *Manager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.done = make(chan struct{})
	m.running = true
	m.status = model.RealtimeStatus{State: "connecting", Message: "connecting to VRChat Pipeline"}
	done := m.done
	m.mu.Unlock()
	m.publishStatus()
	go m.run(ctx, done)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.status = model.RealtimeStatus{State: "disabled", Message: "sign in to enable realtime updates"}
		m.mu.Unlock()
		m.publishStatus()
		return
	}
	cancel := m.cancel
	done := m.done
	m.mu.Unlock()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func (m *Manager) Status() model.RealtimeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) Active() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) run(ctx context.Context, done chan struct{}) {
	defer func() {
		m.mu.Lock()
		m.running = false
		m.cancel = nil
		m.status = model.RealtimeStatus{State: "disabled", Message: "sign in to enable realtime updates"}
		close(done)
		m.mu.Unlock()
		m.publishStatus()
	}()

	backoff := time.Second
	for ctx.Err() == nil {
		err := m.connect(ctx)
		if ctx.Err() != nil {
			return
		}
		m.mu.Lock()
		m.status.State = "disconnected"
		m.status.Message = safeMessage(err)
		m.status.Reconnects++
		m.status.ConnectedAt = nil
		m.mu.Unlock()
		m.publishStatus()
		m.logger.Warn("VRChat Pipeline disconnected", "error", safeMessage(err))

		jitter := time.Duration(rand.Int63n(max(1, int64(backoff/3))))
		timer := time.NewTimer(backoff + jitter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = min(60*time.Second, backoff*2)
		m.mu.Lock()
		m.status.State = "connecting"
		m.status.Message = "reconnecting to VRChat Pipeline"
		m.mu.Unlock()
		m.publishStatus()
	}
}

func (m *Manager) connect(ctx context.Context) error {
	authCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	token, userAgent, err := m.client.PipelineCredentials(authCtx)
	cancel()
	if err != nil {
		return err
	}
	endpoint, err := url.Parse(m.pipelineURL)
	if err != nil {
		return fmt.Errorf("parse Pipeline URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("auth", token)
	endpoint.RawQuery = query.Encode()

	connection, response, err := websocket.Dial(ctx, endpoint.String(), &websocket.DialOptions{
		HTTPClient: m.client.HTTPClientSnapshot(20 * time.Second),
		HTTPHeader: http.Header{"User-Agent": []string{userAgent}},
	})
	if response != nil && response.Body != nil {
		response.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("connect to VRChat Pipeline: %w", err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "local gateway stopping")
	connection.SetReadLimit(2 << 20)

	now := time.Now().UTC()
	m.mu.Lock()
	m.status.State = "connected"
	m.status.Message = "receiving realtime updates"
	m.status.ConnectedAt = &now
	m.mu.Unlock()
	m.publishStatus()

	for {
		_, data, err := connection.Read(ctx)
		if err != nil {
			return err
		}
		m.handleMessage(data)
	}
}

func (m *Manager) handleMessage(data []byte) {
	var outer struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(data, &outer) != nil || outer.Type == "" {
		return
	}
	content := outer.Content
	var encoded string
	if json.Unmarshal(outer.Content, &encoded) == nil {
		if candidate := json.RawMessage(encoded); json.Valid(candidate) {
			content = candidate
		} else {
			content, _ = json.Marshal(encoded)
		}
	}
	now := time.Now().UTC()
	m.mu.Lock()
	m.status.LastMessageAt = &now
	m.mu.Unlock()
	m.bus.Publish(model.DomainEvent{
		ID:   fmt.Sprintf("evt-%d-%d", now.UnixMilli(), m.serial.Add(1)),
		Type: "vrc." + outer.Type, ObservedAt: now, Content: content,
	})
}

func (m *Manager) publishStatus() {
	status := m.Status()
	payload, _ := json.Marshal(status)
	m.bus.Publish(model.DomainEvent{
		ID:   fmt.Sprintf("status-%d-%d", time.Now().UnixMilli(), m.serial.Add(1)),
		Type: "pipeline.status", ObservedAt: time.Now().UTC(), Content: payload,
	})
}

func safeMessage(err error) string {
	if err == nil {
		return "Pipeline connection closed"
	}
	message := err.Error()
	for _, marker := range []string{"?auth=", "&auth="} {
		if index := strings.Index(message, marker); index >= 0 {
			message = message[:index] + marker + "[redacted]"
		}
	}
	if len(message) > 180 {
		message = message[:180]
	}
	return message
}
