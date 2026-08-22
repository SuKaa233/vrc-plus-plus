package presence

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/events"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/security"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
)

const emailSettingKey = "presence.email.settings"
const emailPasswordKey = "presence.email.password"

type Service struct {
	store     *storage.Store
	protector security.Protector
	bus       *events.Bus
	notify    func(string, string) error
	logger    *slog.Logger
	cancel    context.CancelFunc
	mu        sync.Mutex
	pending   map[string]context.CancelFunc
}

func New(store *storage.Store, protector security.Protector, bus *events.Bus, notify func(string, string) error, logger *slog.Logger) *Service {
	return &Service{store: store, protector: protector, bus: bus, notify: notify, logger: logger, pending: map[string]context.CancelFunc{}}
}
func (s *Service) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	stream, unsubscribe := s.bus.Subscribe()
	go func() {
		defer unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-stream:
				if !ok {
					return
				}
				s.handle(ctx, event)
			}
		}
	}()
	go s.digestLoop(ctx)
}
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Service) cancelPending(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel := s.pending[userID]; cancel != nil {
		cancel()
		delete(s.pending, userID)
	}
}

func (s *Service) handle(ctx context.Context, event model.DomainEvent) {
	state := ""
	switch event.Type {
	case "vrc.friend-online":
		state = "online"
	case "vrc.friend-offline":
		state = "offline"
	case "vrc.friend-location":
		s.handleLocation(ctx, event)
		return
	default:
		return
	}
	var payload struct {
		UserID      string `json:"userId"`
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
		User        struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
	}
	if json.Unmarshal(event.Content, &payload) != nil {
		return
	}
	userID := first(payload.UserID, payload.ID, payload.User.ID)
	if userID == "" {
		return
	}
	name := first(payload.DisplayName, payload.User.DisplayName, userID)
	accountID, err := s.store.ActiveAccountID(ctx)
	if err != nil {
		return
	}
	previous, existed, err := s.store.UpdatePresenceState(ctx, accountID, userID, state, event.ObservedAt)
	if err != nil || !existed || previous == state {
		return
	}
	rule, err := s.store.PresenceWatchRule(ctx, accountID, userID)
	if err != nil {
		return
	}
	s.cancelPending(userID)
	if (state == "online" && !rule.NotifyOnline) || (state == "offline" && !rule.NotifyOffline) {
		return
	}
	if rule.DisplayName != "" {
		name = rule.DisplayName
	}
	rule.DisplayName = name
	if state == "online" && rule.MinOnlineSeconds > 0 {
		pendingCtx, cancel := context.WithCancel(ctx)
		s.mu.Lock()
		s.pending[userID] = cancel
		s.mu.Unlock()
		go func() {
			timer := time.NewTimer(time.Duration(rule.MinOnlineSeconds) * time.Second)
			defer timer.Stop()
			select {
			case <-pendingCtx.Done():
				return
			case <-timer.C:
			}
			s.mu.Lock()
			delete(s.pending, userID)
			s.mu.Unlock()
			s.dispatch(pendingCtx, accountID, rule, userID, "online", name+" 已持续在线")
		}()
		return
	}
	message := name + map[string]string{"online": " 已上线", "offline": " 已下线"}[state]
	s.dispatch(ctx, accountID, rule, userID, state, message)
}

func (s *Service) dispatch(ctx context.Context, accountID string, rule model.PresenceWatchRule, userID, eventType, message string) {
	if inQuietHours(rule, time.Now()) {
		for _, channel := range []string{"desktop", "email"} {
			if (channel == "desktop" && !rule.DesktopEnabled) || (channel == "email" && !rule.EmailEnabled) {
				continue
			}
			_, _ = s.store.RecordNotificationDelivery(ctx, accountID, model.NotificationDelivery{UserID: userID, DisplayName: rule.DisplayName, EventType: eventType, Channel: channel, Status: "suppressed", Message: message, ObservedAt: time.Now().UTC(), Error: "免打扰时段"})
		}
		return
	}
	if rule.DesktopEnabled {
		s.deliver(ctx, accountID, model.NotificationDelivery{UserID: userID, DisplayName: rule.DisplayName, EventType: eventType, Channel: "desktop", Message: message, ObservedAt: time.Now().UTC()}, func() error { return s.notify("好友动态", message) })
	}
	if rule.EmailEnabled {
		if rule.EmailMode == "digest" {
			_, _ = s.store.RecordNotificationDelivery(ctx, accountID, model.NotificationDelivery{UserID: userID, DisplayName: rule.DisplayName, EventType: eventType, Channel: "email", Status: "queued", Message: message, ObservedAt: time.Now().UTC()})
			return
		}
		allowed, limitErr := s.store.EmailDeliveryAllowed(ctx, accountID, userID, time.Now().UTC())
		if limitErr != nil || !allowed {
			return
		}
		s.deliver(ctx, accountID, model.NotificationDelivery{UserID: userID, DisplayName: rule.DisplayName, EventType: eventType, Channel: "email", Message: message, ObservedAt: time.Now().UTC()}, func() error { return s.SendEmail(ctx, "VRC++ 好友动态", message) })
	}
}

func (s *Service) handleLocation(ctx context.Context, event model.DomainEvent) {
	var payload struct {
		UserID      string                                     `json:"userId"`
		ID          string                                     `json:"id"`
		DisplayName string                                     `json:"displayName"`
		Location    string                                     `json:"location"`
		User        struct{ ID, DisplayName, Location string } `json:"user"`
	}
	if json.Unmarshal(event.Content, &payload) != nil {
		return
	}
	userID := first(payload.UserID, payload.ID, payload.User.ID)
	location := first(payload.Location, payload.User.Location)
	if userID == "" || location == "" {
		return
	}
	accountID, err := s.store.ActiveAccountID(ctx)
	if err != nil {
		return
	}
	previous, existed, err := s.store.UpdatePresenceLocation(ctx, accountID, userID, sanitizeLocation(location), event.ObservedAt)
	if err != nil || !existed || previous == sanitizeLocation(location) {
		return
	}
	rule, err := s.store.PresenceWatchRule(ctx, accountID, userID)
	if err != nil {
		return
	}
	name := first(rule.DisplayName, payload.DisplayName, payload.User.DisplayName, userID)
	nowJoinable := joinableLocation(location)
	wasJoinable := joinableLocation(previous)
	if rule.NotifyJoinable && nowJoinable && !wasJoinable {
		s.dispatch(ctx, accountID, rule, userID, "joinable", name+" 的位置现在可以加入")
		return
	}
	if rule.NotifyLocation {
		s.dispatch(ctx, accountID, rule, userID, "location", name+" 切换了世界")
	}
}

func sanitizeLocation(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if i := strings.Index(value, ":"); i >= 0 {
		return value[:i] + ":" + locationKind(value)
	}
	return value
}
func locationKind(value string) string {
	lower := strings.ToLower(value)
	for _, kind := range []string{"private", "hidden", "offline", "traveling", "friends+", "friends", "invite+", "invite", "public"} {
		if strings.Contains(lower, kind) {
			return kind
		}
	}
	if strings.HasPrefix(lower, "wrld_") {
		return "public"
	}
	return "unknown"
}
func joinableLocation(value string) bool {
	kind := locationKind(value)
	return kind == "public" || kind == "friends" || kind == "friends+"
}

func inQuietHours(rule model.PresenceWatchRule, now time.Time) bool {
	parse := func(value string) (int, bool) {
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return 0, false
		}
		h, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 != nil || e2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, false
		}
		return h*60 + m, true
	}
	start, ok1 := parse(rule.QuietStart)
	end, ok2 := parse(rule.QuietEnd)
	if !ok1 || !ok2 || start == end {
		return false
	}
	minute := now.Hour()*60 + now.Minute()
	if start < end {
		return minute >= start && minute < end
	}
	return minute >= start || minute < end
}

func (s *Service) digestLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.flushDigest(ctx, now)
		}
	}
}
func (s *Service) flushDigest(ctx context.Context, now time.Time) {
	accountID, err := s.store.ActiveAccountID(ctx)
	if err != nil {
		return
	}
	rules, err := s.store.ListPresenceWatchRules(ctx, accountID)
	if err != nil {
		return
	}
	eligible := map[string]bool{}
	for _, rule := range rules {
		if rule.EmailEnabled && rule.EmailMode == "digest" && rule.DigestHour == now.Hour() {
			eligible[rule.UserID] = true
		}
	}
	if len(eligible) == 0 {
		return
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UTC()
	sent, err := s.store.DigestSentSince(ctx, accountID, start)
	if err != nil || sent {
		return
	}
	queued, err := s.store.ListQueuedNotificationDeliveries(ctx, accountID)
	if err != nil {
		return
	}
	items := []model.NotificationDelivery{}
	for _, item := range queued {
		if eligible[item.UserID] {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return
	}
	var body strings.Builder
	body.WriteString("VRC++ 好友动态汇总\n\n")
	for _, item := range items {
		body.WriteString("• " + item.Message + " · " + item.ObservedAt.Local().Format("01-02 15:04") + "\n")
	}
	status := "sent"
	errText := ""
	sentAt := time.Now().UTC()
	if err := s.SendEmail(ctx, "VRC++ 好友动态汇总", body.String()); err != nil {
		status = "failed"
		errText = err.Error()
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	_ = s.store.MarkNotificationDeliveries(ctx, ids, status, errText, &sentAt)
	_, _ = s.store.RecordNotificationDelivery(ctx, accountID, model.NotificationDelivery{EventType: "digest", Channel: "email", Status: status, Message: fmt.Sprintf("汇总 %d 条好友动态", len(items)), ObservedAt: sentAt, SentAt: &sentAt, Error: errText})
}
func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func (s *Service) deliver(ctx context.Context, accountID string, item model.NotificationDelivery, send func() error) {
	item.Status = "sent"
	now := time.Now().UTC()
	item.SentAt = &now
	if err := send(); err != nil {
		item.Status = "failed"
		item.SentAt = nil
		item.Error = err.Error()
		s.logger.Warn("presence notification failed", "channel", item.Channel, "error", err)
	}
	_, _ = s.store.RecordNotificationDelivery(ctx, accountID, item)
}

func (s *Service) EmailSettings(ctx context.Context) (model.EmailSettings, error) {
	raw, err := s.store.LoadSetting(ctx, emailSettingKey)
	if errors.Is(err, storage.ErrNotFound) {
		return model.EmailSettings{Port: 587, Security: "starttls"}, nil
	}
	if err != nil {
		return model.EmailSettings{}, err
	}
	var item model.EmailSettings
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return item, err
	}
	_, err = s.store.LoadSecureSecret(ctx, emailPasswordKey)
	item.Configured = err == nil
	return item, nil
}
func (s *Service) SaveEmailSettings(ctx context.Context, item model.EmailSettings, password string) (model.EmailSettings, error) {
	item.Host = strings.TrimSpace(item.Host)
	item.Username = strings.TrimSpace(item.Username)
	item.From = strings.TrimSpace(item.From)
	item.To = strings.TrimSpace(item.To)
	item.Security = strings.ToLower(strings.TrimSpace(item.Security))
	if item.Port < 1 || item.Port > 65535 {
		return item, errors.New("SMTP 端口无效")
	}
	if item.Security != "tls" && item.Security != "starttls" {
		return item, errors.New("仅支持 TLS 或 STARTTLS")
	}
	for _, value := range []string{item.From, item.To} {
		if _, err := mail.ParseAddress(value); err != nil {
			return item, errors.New("发件或收件邮箱格式无效")
		}
	}
	fromAddress, _ := mail.ParseAddress(item.From)
	toAddress, _ := mail.ParseAddress(item.To)
	if !strings.EqualFold(fromAddress.Address, toAddress.Address) {
		return item, errors.New("为防止滥用，接收邮箱必须与发件邮箱相同")
	}
	if strings.ContainsAny(item.Host, "\r\n") || strings.ContainsAny(item.Username, "\r\n") {
		return item, errors.New("SMTP 配置包含无效字符")
	}
	item.Configured = false
	raw, _ := json.Marshal(item)
	if err := s.store.SaveSetting(ctx, emailSettingKey, string(raw)); err != nil {
		return item, err
	}
	if password != "" {
		cipher, err := s.protector.Protect([]byte(password))
		if err != nil {
			return item, err
		}
		if err := s.store.SaveSecureSecret(ctx, emailPasswordKey, cipher); err != nil {
			return item, err
		}
	}
	_, err := s.store.LoadSecureSecret(ctx, emailPasswordKey)
	item.Configured = err == nil
	return item, nil
}
func (s *Service) SendEmail(ctx context.Context, subject, body string) error {
	settings, err := s.EmailSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return errors.New("邮件推送未启用")
	}
	cipher, err := s.store.LoadSecureSecret(ctx, emailPasswordKey)
	if err != nil {
		return errors.New("尚未保存 SMTP 授权码")
	}
	password, err := s.protector.Unprotect(cipher)
	if err != nil {
		return err
	}
	return sendSMTP(ctx, settings, string(password), subject, body)
}
func sendSMTP(ctx context.Context, c model.EmailSettings, password, subject, body string) error {
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	dialer := net.Dialer{Timeout: 12 * time.Second}
	var conn net.Conn
	var err error
	if c.Security == "tls" {
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if c.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP 服务器不支持 STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if c.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", c.Username, password, c.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(c.From); err != nil {
		return err
	}
	if err := client.Rcpt(c.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", c.From, c.To, subject, body)
	if _, err = writer.Write([]byte(message)); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
