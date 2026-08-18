package localapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
)

func (s *Server) presenceAccount(request *http.Request) (string, error) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	return s.store.ActiveAccountID(ctx)
}
func (s *Server) listPresenceWatches(w http.ResponseWriter, r *http.Request) {
	account, err := s.presenceAccount(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	items, err := s.store.ListPresenceWatchRules(ctx, account)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s *Server) savePresenceWatch(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.PathValue("userID"))
	if !strings.HasPrefix(userID, "usr_") {
		writeAPIError(w, 400, "LOCAL_INVALID_REQUEST", "无效的好友 ID", false)
		return
	}
	var input struct {
		model.PresenceWatchRule
		CurrentState string `json:"currentState"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, 400, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	input.UserID = userID
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	account, err := s.presenceAccount(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	item, err := s.store.SavePresenceWatchRule(ctx, account, input.PresenceWatchRule)
	if err != nil {
		writeError(w, err)
		return
	}
	if input.CurrentState == "online" || input.CurrentState == "offline" {
		_, _, _ = s.store.UpdatePresenceState(ctx, account, userID, input.CurrentState, time.Now().UTC())
	}
	writeJSON(w, 200, item)
}
func (s *Server) deletePresenceWatch(w http.ResponseWriter, r *http.Request) {
	account, err := s.presenceAccount(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	if err := s.store.DeletePresenceWatchRule(ctx, account, r.PathValue("userID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) getPresenceEmail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	item, err := s.presence.EmailSettings(ctx)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) savePresenceEmail(w http.ResponseWriter, r *http.Request) {
	var input struct {
		model.EmailSettings
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, 400, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	item, err := s.presence.SaveEmailSettings(ctx, input.EmailSettings, input.Password)
	if err != nil {
		writeAPIError(w, 400, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) testPresenceEmail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 20*time.Second)
	defer cancel()
	if err := s.presence.SendEmail(ctx, "VRC++ 测试邮件", "邮件推送配置成功。以后被关注好友的上下线变化会发送到这里。"); err != nil {
		writeAPIError(w, 502, "LOCAL_EMAIL_FAILED", err.Error(), true)
		return
	}
	writeJSON(w, 200, map[string]bool{"sent": true})
}
func (s *Server) listPresenceDeliveries(w http.ResponseWriter, r *http.Request) {
	account, err := s.presenceAccount(r)
	if errors.Is(err, storage.ErrNotFound) {
		writeJSON(w, 200, []model.NotificationDelivery{})
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	items, err := s.store.ListNotificationDeliveries(ctx, account, 50)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, items)
}
