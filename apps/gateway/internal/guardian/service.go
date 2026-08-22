package guardian

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/events"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/security"
)

const recoveryTTL = 24 * time.Hour

type Session struct {
	WorldID          string        `json:"worldId"`
	WorldName        string        `json:"worldName,omitempty"`
	InstanceID       string        `json:"instanceId"`
	Location         string        `json:"location"`
	LocationKind     string        `json:"locationKind"`
	AccessOwnerID    string        `json:"accessOwnerId,omitempty"`
	GroupID          string        `json:"groupId,omitempty"`
	Region           string        `json:"region,omitempty"`
	JoinedAt         time.Time     `json:"joinedAt"`
	LastObservedAt   time.Time     `json:"lastObservedAt"`
	ParticipantCount int           `json:"participantCount"`
	Participants     []Participant `json:"participants,omitempty"`
}

type Participant struct {
	UserID      string `json:"userId,omitempty"`
	DisplayName string `json:"displayName"`
}

type Status struct {
	GameRunning bool       `json:"gameRunning"`
	State       string     `json:"state"`
	ExitKind    string     `json:"exitKind,omitempty"`
	ExitAt      *time.Time `json:"exitAt,omitempty"`
	Current     *Session   `json:"current,omitempty"`
	Last        *Session   `json:"last,omitempty"`
	CanResume   bool       `json:"canResume"`
	Dismissed   bool       `json:"dismissed"`
	Message     string     `json:"message"`
}

type diskState struct {
	Current   *Session   `json:"current,omitempty"`
	Last      *Session   `json:"last,omitempty"`
	ExitKind  string     `json:"exitKind,omitempty"`
	ExitAt    *time.Time `json:"exitAt,omitempty"`
	Dismissed bool       `json:"dismissed"`
}

type Service struct {
	mu        sync.RWMutex
	state     diskState
	running   bool
	cleanQuit bool
	players   map[string]Participant
	worldName string
	path      string
	protector security.Protector
	events    *events.Bus
	launch    func(string) error
	notify    func(string, string) error
	logger    *slog.Logger
	cancel    context.CancelFunc
	now       func() time.Time
	process   func() bool
	lastSave  time.Time
}

func New(path string, protector security.Protector, bus *events.Bus, launch func(string) error, notify func(string, string) error, logger *slog.Logger) *Service {
	return &Service{path: path, protector: protector, events: bus, launch: launch, notify: notify, logger: logger, players: map[string]Participant{}, now: func() time.Time { return time.Now().UTC() }, process: vrchatProcessRunning}
}

func (s *Service) Start(parent context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	if err := s.loadLocked(); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Warn("could not load guardian state", "error", err)
	}
	s.running = s.process()
	if !s.running && s.state.Current != nil {
		lastObserved := s.state.Current.LastObservedAt
		s.state.Last = s.state.Current
		s.state.Current = nil
		s.state.ExitAt = &lastObserved
		s.state.ExitKind = "interrupted"
		s.state.Dismissed = false
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	channel, unsubscribe := s.events.Subscribe()
	go func() {
		defer unsubscribe()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-channel:
				if !ok {
					return
				}
				s.handleEvent(event)
			case <-ticker.C:
				s.observeProcess()
			}
		}
	}()
}

func (s *Service) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	_ = s.saveLocked()
	s.mu.Unlock()
}

func (s *Service) handleEvent(event model.DomainEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var content struct{ DisplayName, UserID, Location, WorldName string }
	_ = json.Unmarshal(event.Content, &content)
	if event.Type != "game.quit-clean" && strings.HasPrefix(event.Type, "game.") && !s.running {
		if !s.process() {
			return
		}
		s.running = true
	}
	switch event.Type {
	case "game.world-entering":
		s.worldName = strings.TrimSpace(content.WorldName)
	case "game.location":
		location := strings.TrimSpace(event.SensitiveLocation)
		if location == "" {
			location = strings.TrimSpace(content.Location)
		}
		worldID, instanceID := splitLocation(location)
		if worldID == "" || instanceID == "" {
			return
		}
		now := event.ObservedAt.UTC()
		s.state.Current = &Session{WorldID: worldID, WorldName: s.worldName, InstanceID: publicInstanceID(instanceID), Location: location, LocationKind: locationKind(instanceID), AccessOwnerID: locationModifierValue(instanceID, "private", "friends", "friends+", "hidden"), GroupID: locationModifierValue(instanceID, "group"), Region: locationRegion(instanceID), JoinedAt: now, LastObservedAt: now}
		s.state.Dismissed = false
		s.cleanQuit = false
		s.players = map[string]Participant{}
		_ = s.saveLocked()
		s.lastSave = s.now()
	case "game.player-joined":
		if s.state.Current == nil {
			return
		}
		key := strings.TrimSpace(content.UserID)
		if key == "" {
			key = strings.TrimSpace(content.DisplayName)
		}
		if key != "" {
			s.players[key] = Participant{UserID: strings.TrimSpace(content.UserID), DisplayName: strings.TrimSpace(content.DisplayName)}
		}
		s.syncParticipantsLocked()
		s.state.Current.LastObservedAt = event.ObservedAt.UTC()
		s.saveSnapshotLocked()
	case "game.player-left":
		if s.state.Current == nil {
			return
		}
		key := strings.TrimSpace(content.UserID)
		if key == "" {
			key = strings.TrimSpace(content.DisplayName)
		}
		delete(s.players, key)
		s.syncParticipantsLocked()
		s.state.Current.LastObservedAt = event.ObservedAt.UTC()
		s.saveSnapshotLocked()
	case "game.quit-clean":
		s.cleanQuit = true
	}
}

func (s *Service) observeProcess() {
	running := s.process()
	s.mu.Lock()
	if running == s.running {
		s.mu.Unlock()
		return
	}
	wasRunning := s.running
	s.running = running
	if running {
		s.cleanQuit = false
		s.mu.Unlock()
		return
	}
	if !wasRunning || s.state.Current == nil {
		s.mu.Unlock()
		return
	}
	now := s.now()
	s.state.Last = s.state.Current
	s.state.Current = nil
	s.state.ExitAt = &now
	if s.cleanQuit {
		s.state.ExitKind = "clean"
	} else {
		s.state.ExitKind = "unexpected"
	}
	s.state.Dismissed = false
	exitKind := s.state.ExitKind
	_ = s.saveLocked()
	s.mu.Unlock()
	if exitKind == "unexpected" && s.notify != nil {
		_ = s.notify("VRC++ 已保存返回现场", "检测到 VRChat 意外退出；打开 VRC++ 可以尝试返回刚才的实例。")
	}
}

func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := Status{GameRunning: s.running, Current: cloneSession(s.state.Current), Last: cloneSession(s.state.Last), ExitKind: s.state.ExitKind, ExitAt: s.state.ExitAt, Dismissed: s.state.Dismissed}
	if s.running && result.Current != nil {
		result.State, result.Message = "protecting", "正在守护当前 VRChat 会话"
		return result
	}
	if result.Last != nil && result.ExitAt != nil && s.now().Sub(*result.ExitAt) <= recoveryTTL {
		result.CanResume = result.Last.WorldID != "" && result.Last.InstanceID != ""
		if result.ExitKind == "unexpected" && !result.Dismissed {
			result.State, result.Message = "recovery", "检测到异常退出，已保存最后实例"
		} else {
			result.State, result.Message = "ready", "可以继续最近一次会话"
		}
		return result
	}
	result.State, result.Message = "idle", "等待 VRChat 会话"
	return result
}

func (s *Service) Resume() (string, error) {
	s.mu.RLock()
	last := cloneSession(s.state.Last)
	s.mu.RUnlock()
	if last == nil || last.Location == "" {
		return "", fmt.Errorf("没有可恢复的 VRChat 实例")
	}
	worldID, instanceID := splitLocation(last.Location)
	if worldID == "" || instanceID == "" {
		return "", fmt.Errorf("最后实例地址不完整")
	}
	query := url.Values{"worldId": {worldID}, "instanceId": {instanceID}, "launch": {"1"}}
	target := "https://vrchat.com/home/launch?" + query.Encode()
	if s.launch == nil {
		return "", fmt.Errorf("当前平台不支持启动链接")
	}
	if err := s.launch(target); err != nil {
		return "", err
	}
	return target, nil
}

func (s *Service) Dismiss() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Dismissed = true
	return s.saveLocked()
}

func (s *Service) saveLocked() error {
	plain, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	cipher, err := s.protector.Protect(plain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, []byte(base64.RawStdEncoding.EncodeToString(cipher)), 0o600)
}

func (s *Service) loadLocked() error {
	encoded, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	cipher, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return err
	}
	plain, err := s.protector.Unprotect(cipher)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, &s.state)
}

func splitLocation(location string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(location), ":", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "wrld_") {
		return "", ""
	}
	return parts[0], parts[1]
}
func publicInstanceID(instance string) string {
	if i := strings.Index(instance, "~"); i >= 0 {
		return instance[:i]
	}
	return instance
}
func locationKind(instance string) string {
	for _, item := range []struct{ marker, kind string }{{"~private(", "invite"}, {"~friends+(", "friends_plus"}, {"~friends(", "friends"}, {"~hidden(", "invite_plus"}, {"~group(", "group"}} {
		if strings.Contains(instance, item.marker) {
			return item.kind
		}
	}
	return "public"
}
func locationRegion(instance string) string {
	marker := "~region("
	start := strings.Index(instance, marker)
	if start < 0 {
		return ""
	}
	rest := instance[start+len(marker):]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	return rest[:end]
}
func locationModifierValue(instance string, names ...string) string {
	for _, name := range names {
		marker := "~" + name + "("
		start := strings.Index(instance, marker)
		if start < 0 {
			continue
		}
		rest := instance[start+len(marker):]
		if end := strings.Index(rest, ")"); end >= 0 {
			return rest[:end]
		}
	}
	return ""
}
func (s *Service) syncParticipantsLocked() {
	if s.state.Current == nil {
		return
	}
	items := make([]Participant, 0, len(s.players))
	for _, item := range s.players {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].DisplayName) < strings.ToLower(items[j].DisplayName)
	})
	s.state.Current.Participants = items
	s.state.Current.ParticipantCount = len(items)
}
func (s *Service) saveSnapshotLocked() {
	now := s.now()
	if !s.lastSave.IsZero() && now.Sub(s.lastSave) < 5*time.Second {
		return
	}
	if err := s.saveLocked(); err == nil {
		s.lastSave = now
	}
}
func cloneSession(value *Session) *Session {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Participants = append([]Participant(nil), value.Participants...)
	return &copy
}
