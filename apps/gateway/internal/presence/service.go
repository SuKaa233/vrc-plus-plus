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
}

func New(store *storage.Store, protector security.Protector, bus *events.Bus, notify func(string, string) error, logger *slog.Logger) *Service {
	return &Service{store: store, protector: protector, bus: bus, notify: notify, logger: logger}
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
}
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Service) handle(ctx context.Context, event model.DomainEvent) {
	state := ""
	switch event.Type {
	case "vrc.friend-online":
		state = "online"
	case "vrc.friend-offline":
		state = "offline"
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
	if (state == "online" && !rule.NotifyOnline) || (state == "offline" && !rule.NotifyOffline) {
		return
	}
	if rule.DisplayName != "" {
		name = rule.DisplayName
	}
	message := name + map[string]string{"online": " 已上线", "offline": " 已下线"}[state]
	if rule.DesktopEnabled {
		s.deliver(ctx, accountID, model.NotificationDelivery{UserID: userID, DisplayName: name, EventType: state, Channel: "desktop", Message: message, ObservedAt: event.ObservedAt}, func() error { return s.notify("好友动态", message) })
	}
	if rule.EmailEnabled {
		allowed, limitErr := s.store.EmailDeliveryAllowed(ctx, accountID, userID, time.Now().UTC())
		if limitErr != nil || !allowed {
			return
		}
		s.deliver(ctx, accountID, model.NotificationDelivery{UserID: userID, DisplayName: name, EventType: state, Channel: "email", Message: message, ObservedAt: event.ObservedAt}, func() error { return s.SendEmail(ctx, "VRC++ 好友动态", message) })
	}
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
