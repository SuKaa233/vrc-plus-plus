package localapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/diagnostics"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/events"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/gamelog"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/media"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/pipeline"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/updater"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/vrchat"
)

type Config struct {
	AppName      string
	Version      string
	DevOrigin    string
	StaticFS     fs.FS
	SecurityName string
	Store        *storage.Store
	GameLog      *gamelog.Watcher
	Updater      *updater.Service
	Shutdown     chan struct{}
}

type Server struct {
	config      Config
	csrfToken   string
	vrchat      *vrchat.Client
	diagnostics *diagnostics.Service
	events      *events.Bus
	pipeline    *pipeline.Manager
	media       *media.Service
	store       *storage.Store
	gameLog     *gamelog.Watcher
	updater     *updater.Service
	shutdown    chan struct{}
	logger      *slog.Logger
	static      http.Handler
}

func New(config Config, vrchatClient *vrchat.Client, diagnosticService *diagnostics.Service, eventBus *events.Bus, pipelineManager *pipeline.Manager, mediaService *media.Service, logger *slog.Logger) (*Server, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate CSRF token: %w", err)
	}
	server := &Server{
		config:      config,
		csrfToken:   base64.RawURLEncoding.EncodeToString(tokenBytes),
		vrchat:      vrchatClient,
		diagnostics: diagnosticService,
		events:      eventBus,
		pipeline:    pipelineManager,
		media:       mediaService,
		store:       config.Store,
		gameLog:     config.GameLog,
		updater:     config.Updater,
		shutdown:    config.Shutdown,
		logger:      logger,
		static:      http.FileServer(http.FS(config.StaticFS)),
	}
	return server, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /local/v1/bootstrap", s.bootstrap)
	mux.HandleFunc("GET /local/v1/diagnostics", s.getDiagnostics)
	mux.HandleFunc("GET /local/v1/auth/session", s.getSession)
	mux.HandleFunc("POST /local/v1/auth/login", s.login)
	mux.HandleFunc("POST /local/v1/auth/2fa", s.verifyTwoFactor)
	mux.HandleFunc("DELETE /local/v1/auth/session", s.logout)
	mux.HandleFunc("GET /local/v1/friends", s.getFriends)
	mux.HandleFunc("GET /local/v1/friend-network", s.getFriendNetwork)
	mux.HandleFunc("GET /local/v1/friend-annotations", s.listFriendAnnotations)
	mux.HandleFunc("PUT /local/v1/friend-annotations/{userID}", s.saveFriendAnnotation)
	mux.HandleFunc("DELETE /local/v1/friend-annotations/{userID}", s.deleteFriendAnnotation)
	mux.HandleFunc("GET /local/v1/cache", s.getCacheStats)
	mux.HandleFunc("DELETE /local/v1/cache/media", s.clearMediaCache)
	mux.HandleFunc("DELETE /local/v1/cache/entities", s.clearEntityCache)
	mux.HandleFunc("GET /local/v1/world-favorites", s.listWorldFavorites)
	mux.HandleFunc("PUT /local/v1/world-favorites/{worldID}", s.saveWorldFavorite)
	mux.HandleFunc("DELETE /local/v1/world-favorites/{worldID}", s.deleteWorldFavorite)
	mux.HandleFunc("GET /local/v1/vrchat-favorites", s.listUpstreamWorldFavorites)
	mux.HandleFunc("GET /local/v1/vrchat-favorite-groups", s.listFavoriteGroups)
	mux.HandleFunc("GET /local/v1/groups", s.listGroups)
	mux.HandleFunc("GET /local/v1/groups/{groupID}/posts", s.listGroupPosts)
	mux.HandleFunc("GET /local/v1/groups/{groupID}/instances", s.listGroupInstances)
	mux.HandleFunc("GET /local/v1/groups/{groupID}/calendar", s.listGroupCalendarEvents)
	mux.HandleFunc("GET /local/v1/avatars/favorites", s.listFavoriteAvatars)
	mux.HandleFunc("POST /local/v1/vrchat-favorites/{worldID}", s.addUpstreamWorldFavorite)
	mux.HandleFunc("DELETE /local/v1/vrchat-favorites/{favoriteID}", s.deleteUpstreamWorldFavorite)
	mux.HandleFunc("GET /local/v1/instance", s.getInstance)
	mux.HandleFunc("POST /local/v1/invites", s.sendInvite)
	mux.HandleFunc("POST /local/v1/users/{userID}/boop", s.sendBoop)
	mux.HandleFunc("GET /local/v1/game-log/status", s.getGameLogStatus)
	mux.HandleFunc("GET /local/v1/update", s.getUpdateStatus)
	mux.HandleFunc("POST /local/v1/update/check", s.checkUpdate)
	mux.HandleFunc("POST /local/v1/update/download", s.downloadUpdate)
	mux.HandleFunc("POST /local/v1/update/apply", s.applyUpdate)
	mux.HandleFunc("GET /local/v1/activity", s.listActivity)
	mux.HandleFunc("GET /local/v1/activity/insights", s.getActivityInsights)
	mux.HandleFunc("GET /local/v1/activity/users/{userID}/insights", s.getFriendActivityInsights)
	mux.HandleFunc("DELETE /local/v1/activity", s.clearActivity)
	mux.HandleFunc("GET /local/v1/notifications", s.listNotifications)
	mux.HandleFunc("POST /local/v1/notifications/{notificationID}/{action}", s.notificationAction)
	mux.HandleFunc("GET /local/v1/users/{userID}", s.getUser)
	mux.HandleFunc("PUT /local/v1/profile", s.updateSelfProfile)
	mux.HandleFunc("GET /local/v1/users/{userID}/mutual-friends", s.getMutualFriends)
	mux.HandleFunc("GET /local/v1/users/{userID}/friend-status", s.getFriendStatus)
	mux.HandleFunc("POST /local/v1/users/{userID}/friend-request", s.sendFriendRequest)
	mux.HandleFunc("GET /local/v1/discovery/users", s.searchUsers)
	mux.HandleFunc("GET /local/v1/worlds", s.searchWorlds)
	mux.HandleFunc("GET /local/v1/worlds/{worldID}", s.getWorld)
	mux.HandleFunc("GET /local/v1/realtime/status", s.getRealtimeStatus)
	mux.HandleFunc("GET /local/v1/events/stream", s.streamEvents)
	mux.HandleFunc("GET /local/v1/network", s.getNetwork)
	mux.HandleFunc("PUT /local/v1/network", s.updateNetwork)
	if s.media != nil {
		mux.Handle("GET /local/v1/media", s.media)
	}
	mux.HandleFunc("GET /", s.frontend)
	return s.recover(s.securityHeaders(s.loopbackOnly(s.originProtection(s.csrfProtection(mux)))))
}

var annotationColorPattern = regexp.MustCompile(`^$|^#[0-9a-fA-F]{6}$`)

func (s *Server) listFriendAnnotations(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	items, err := s.store.ListFriendAnnotations(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) saveFriendAnnotation(writer http.ResponseWriter, request *http.Request) {
	userID := strings.TrimSpace(request.PathValue("userID"))
	if !strings.HasPrefix(userID, "usr_") || len(userID) > 80 || strings.ContainsAny(userID, "/?#") {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "无效的 VRChat 用户 ID", false)
		return
	}
	var input struct {
		Note  string   `json:"note"`
		Group string   `json:"group"`
		Color string   `json:"color"`
		Tags  []string `json:"tags"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	input.Note = strings.TrimSpace(input.Note)
	input.Group = strings.TrimSpace(input.Group)
	input.Color = strings.TrimSpace(input.Color)
	if len([]rune(input.Note)) > 500 || len([]rune(input.Group)) > 32 || !annotationColorPattern.MatchString(input.Color) || len(input.Tags) > 12 {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "备注、分组、颜色或标签超出允许范围", false)
		return
	}
	tags := make([]string, 0, len(input.Tags))
	seen := map[string]bool{}
	for _, raw := range input.Tags {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] {
			continue
		}
		if len([]rune(tag)) > 24 {
			writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "单个标签不能超过 24 个字符", false)
			return
		}
		seen[key] = true
		tags = append(tags, tag)
	}
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	item, err := s.store.SaveFriendAnnotation(ctx, model.FriendAnnotation{UserID: userID, Note: input.Note, Group: input.Group, Color: input.Color, Tags: tags})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (s *Server) deleteFriendAnnotation(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	if err := s.store.DeleteFriendAnnotation(ctx, request.PathValue("userID")); err != nil {
		writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func validWorldID(worldID string) bool {
	return strings.HasPrefix(worldID, "wrld_") && len(worldID) <= 80 && !strings.ContainsAny(worldID, "/?#")
}

func (s *Server) listWorldFavorites(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	items, err := s.store.ListWorldFavorites(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) saveWorldFavorite(writer http.ResponseWriter, request *http.Request) {
	worldID := strings.TrimSpace(request.PathValue("worldID"))
	if !validWorldID(worldID) {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "无效的 VRChat 世界 ID", false)
		return
	}
	var input struct {
		World model.World `json:"world"`
		Note  string      `json:"note"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	input.Note = strings.TrimSpace(input.Note)
	if input.World.ID != worldID || strings.TrimSpace(input.World.Name) == "" || len([]rune(input.Note)) > 300 {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "世界信息或本机备注无效", false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	item, err := s.store.SaveWorldFavorite(ctx, model.WorldFavorite{World: input.World, Note: input.Note})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (s *Server) deleteWorldFavorite(writer http.ResponseWriter, request *http.Request) {
	worldID := strings.TrimSpace(request.PathValue("worldID"))
	if !validWorldID(worldID) {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "无效的 VRChat 世界 ID", false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	if err := s.store.DeleteWorldFavorite(ctx, worldID); err != nil {
		writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) listUpstreamWorldFavorites(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 30*time.Second)
	defer cancel()
	items, err := s.vrchat.ListUpstreamWorldFavorites(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) listFavoriteGroups(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	items, err := s.vrchat.ListFavoriteGroups(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) listGroups(writer http.ResponseWriter, request *http.Request) {
	userID := strings.TrimSpace(request.URL.Query().Get("userId"))
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	items, err := s.vrchat.ListUserGroups(ctx, userID, request.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) listGroupPosts(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	items, err := s.vrchat.ListGroupPosts(ctx, request.PathValue("groupID"), request.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) listGroupInstances(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	items, err := s.vrchat.ListGroupInstances(ctx, request.PathValue("groupID"), request.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) listGroupCalendarEvents(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	items, err := s.vrchat.ListGroupCalendarEvents(ctx, request.PathValue("groupID"), strings.TrimSpace(request.URL.Query().Get("month")), request.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) listFavoriteAvatars(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 35*time.Second)
	defer cancel()
	items, err := s.vrchat.ListFavoriteAvatars(ctx, request.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) addUpstreamWorldFavorite(writer http.ResponseWriter, request *http.Request) {
	worldID := strings.TrimSpace(request.PathValue("worldID"))
	if !validWorldID(worldID) {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "无效的 VRChat 世界 ID", false)
		return
	}
	var input struct {
		Group string `json:"group"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	item, err := s.vrchat.AddUpstreamWorldFavorite(ctx, worldID, input.Group)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (s *Server) deleteUpstreamWorldFavorite(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	if err := s.vrchat.DeleteUpstreamWorldFavorite(ctx, request.PathValue("favoriteID")); err != nil {
		writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) sendInvite(writer http.ResponseWriter, request *http.Request) {
	var input model.InviteRequest
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	if err := s.vrchat.SendInvite(ctx, strings.TrimSpace(input.ReceiverUserID), strings.TrimSpace(input.Location)); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) sendBoop(writer http.ResponseWriter, request *http.Request) {
	var input model.BoopRequest
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	if err := s.vrchat.SendBoop(ctx, strings.TrimSpace(request.PathValue("userID")), strings.TrimSpace(input.EmojiID)); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) getGameLogStatus(writer http.ResponseWriter, _ *http.Request) {
	if s.gameLog == nil {
		writeJSON(writer, http.StatusOK, model.GameLogStatus{State: "disabled", Message: "游戏日志监视未启用"})
		return
	}
	writeJSON(writer, http.StatusOK, s.gameLog.Status())
}

func (s *Server) getUpdateStatus(writer http.ResponseWriter, _ *http.Request) {
	if s.updater == nil {
		writeJSON(writer, http.StatusOK, model.UpdateStatus{State: "disabled", Current: s.config.Version})
		return
	}
	writeJSON(writer, http.StatusOK, s.updater.Status())
}

func (s *Server) checkUpdate(writer http.ResponseWriter, request *http.Request) {
	if s.updater == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_UPDATE_DISABLED", "更新服务未启用", false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 45*time.Second)
	defer cancel()
	writeJSON(writer, http.StatusOK, s.updater.Check(ctx))
}

func (s *Server) downloadUpdate(writer http.ResponseWriter, request *http.Request) {
	if s.updater == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_UPDATE_DISABLED", "更新服务未启用", false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 5*time.Minute)
	defer cancel()
	status, err := s.updater.Download(ctx)
	if err != nil {
		writeAPIError(writer, http.StatusBadGateway, "LOCAL_UPDATE_DOWNLOAD_FAILED", err.Error(), true)
		return
	}
	writeJSON(writer, http.StatusOK, status)
}

func (s *Server) applyUpdate(writer http.ResponseWriter, _ *http.Request) {
	if s.updater == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_UPDATE_DISABLED", "更新服务未启用", false)
		return
	}
	if err := s.updater.LaunchApply(); err != nil {
		writeAPIError(writer, http.StatusConflict, "LOCAL_UPDATE_NOT_READY", err.Error(), false)
		return
	}
	writeJSON(writer, http.StatusAccepted, map[string]bool{"restarting": true})
	if s.shutdown != nil {
		go func() {
			time.Sleep(250 * time.Millisecond)
			select {
			case s.shutdown <- struct{}{}:
			default:
			}
		}()
	}
}

func (s *Server) getInstance(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	item, err := s.vrchat.GetInstance(ctx, request.URL.Query().Get("location"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (s *Server) listActivity(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	items, err := s.store.ListActivityEvents(ctx, queryInt(request, "days", 30), queryInt(request, "limit", 100))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) getActivityInsights(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	item, err := s.store.ActivityInsights(ctx, queryInt(request, "days", 30))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (s *Server) getFriendActivityInsights(writer http.ResponseWriter, request *http.Request) {
	userID := strings.TrimSpace(request.PathValue("userID"))
	if !strings.HasPrefix(userID, "usr_") || len(userID) > 80 || strings.ContainsAny(userID, "/?#") {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "无效的 VRChat 用户 ID", false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	item, err := s.store.FriendActivityInsights(ctx, userID, queryInt(request, "days", 30))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (s *Server) clearActivity(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	if err := s.store.ClearActivityEvents(ctx); err != nil {
		writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) listNotifications(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	items, err := s.vrchat.ListNotifications(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, items)
}

func (s *Server) notificationAction(writer http.ResponseWriter, request *http.Request) {
	action := request.PathValue("action")
	if action != "see" && action != "hide" && action != "accept" {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", "不支持的通知操作", false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	if err := s.vrchat.NotificationAction(ctx, request.PathValue("notificationID"), action); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) getCacheStats(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	stats, err := s.store.CacheStats(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	if s.media != nil {
		stats.MediaFiles, stats.MediaBytes, err = s.media.Stats()
		if err != nil {
			writeError(writer, err)
			return
		}
	}
	writeJSON(writer, http.StatusOK, stats)
}

func (s *Server) clearMediaCache(writer http.ResponseWriter, _ *http.Request) {
	if s.media == nil {
		writeJSON(writer, http.StatusOK, model.CacheClearResult{})
		return
	}
	files, bytes, err := s.media.Clear()
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, model.CacheClearResult{RemovedFiles: files, FreedBytes: bytes})
}

func (s *Server) clearEntityCache(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 10*time.Second)
	defer cancel()
	count, err := s.store.ClearEntityCache(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, model.CacheClearResult{RemovedEntries: count})
}

func (s *Server) getNetwork(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, s.vrchat.NetworkState())
}

func (s *Server) updateNetwork(writer http.ResponseWriter, request *http.Request) {
	var input model.NetworkConfig
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	wasActive := s.pipeline.Active()
	if wasActive {
		s.pipeline.Stop()
	}
	ctx, cancel := contextWithTimeout(request, 10*time.Second)
	defer cancel()
	state, err := s.vrchat.ApplyNetworkConfig(ctx, input)
	if err != nil {
		if wasActive {
			s.pipeline.Start()
		}
		writeError(writer, err)
		return
	}
	if s.updater != nil {
		s.updater.SetHTTPClient(s.vrchat.HTTPClientSnapshot(5 * time.Minute))
	}
	s.diagnostics.Invalidate()
	if wasActive {
		s.pipeline.Start()
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) bootstrap(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"appName":   s.config.AppName,
		"version":   s.config.Version,
		"csrfToken": s.csrfToken,
		"security": map[string]any{
			"loopbackOnly":      true,
			"originProtected":   true,
			"sessionEncryption": s.config.SecurityName,
		},
	})
}

func (s *Server) getDiagnostics(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	writeJSON(writer, http.StatusOK, s.diagnostics.Run(ctx, request.URL.Query().Get("refresh") == "1"))
}

func (s *Server) getSession(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	state, err := s.vrchat.Restore(ctx)
	if err != nil && state.Status != model.SessionUnavailable {
		writeError(writer, err)
		return
	}
	if state.Status == model.SessionAuthenticated {
		s.pipeline.Start()
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) login(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	state, err := s.vrchat.Login(ctx, input.Username, input.Password)
	input.Password = ""
	if err != nil {
		writeError(writer, err)
		return
	}
	if state.Status == model.SessionAuthenticated {
		s.pipeline.Start()
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) verifyTwoFactor(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Type string `json:"type"`
		Code string `json:"code"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	state, err := s.vrchat.VerifyTwoFactor(ctx, input.Type, input.Code)
	input.Code = ""
	if err != nil {
		writeError(writer, err)
		return
	}
	if state.Status == model.SessionAuthenticated {
		s.pipeline.Start()
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) logout(writer http.ResponseWriter, request *http.Request) {
	s.pipeline.Stop()
	ctx, cancel := contextWithTimeout(request, 15*time.Second)
	defer cancel()
	state, err := s.vrchat.Logout(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) getFriends(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 45*time.Second)
	defer cancel()
	result, err := s.vrchat.ListFriends(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getUser(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	result, err := s.vrchat.GetUser(ctx, request.PathValue("userID"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) updateSelfProfile(writer http.ResponseWriter, request *http.Request) {
	var input model.SelfProfileUpdate
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	ctx, cancel := contextWithTimeout(request, 35*time.Second)
	defer cancel()
	result, err := s.vrchat.UpdateSelfProfile(ctx, input)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) searchUsers(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	result, err := s.vrchat.SearchUsers(ctx, request.URL.Query().Get("query"), queryInt(request, "limit", 12))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getFriendStatus(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	result, err := s.vrchat.FriendStatus(ctx, request.PathValue("userID"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) sendFriendRequest(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()
	if err := s.vrchat.SendFriendRequest(ctx, request.PathValue("userID")); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) getMutualFriends(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	result, err := s.vrchat.ListMutualFriends(ctx, request.PathValue("userID"), request.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getFriendNetwork(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 5*time.Second)
	defer cancel()
	result, err := s.vrchat.FriendNetwork(ctx)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) searchWorlds(writer http.ResponseWriter, request *http.Request) {
	offset := queryInt(request, "offset", 0)
	limit := queryInt(request, "limit", 24)
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	result, err := s.vrchat.SearchWorlds(ctx, request.URL.Query().Get("search"), offset, limit)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getWorld(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := contextWithTimeout(request, 25*time.Second)
	defer cancel()
	result, err := s.vrchat.GetWorld(ctx, request.PathValue("worldID"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getRealtimeStatus(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, s.pipeline.Status())
}

func (s *Server) streamEvents(writer http.ResponseWriter, request *http.Request) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		writeAPIError(writer, http.StatusInternalServerError, "LOCAL_STREAM_UNAVAILABLE", "event streaming is unavailable", true)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache, no-transform")
	writer.Header().Set("Connection", "keep-alive")
	_ = http.NewResponseController(writer).SetWriteDeadline(time.Time{})
	channel, unsubscribe := s.events.Subscribe()
	defer unsubscribe()
	_, _ = io.WriteString(writer, ": connected\n\n")
	flusher.Flush()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case event, open := <-channel:
			if !open {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(writer, "id: %s\ndata: %s\n\n", event.ID, payload); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func queryInt(request *http.Request, name string, fallback int) int {
	value := request.URL.Query().Get(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (s *Server) frontend(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/local/") {
		http.NotFound(writer, request)
		return
	}
	path := strings.TrimPrefix(request.URL.Path, "/")
	if strings.HasPrefix(path, "docs/") || strings.HasSuffix(strings.ToLower(path), ".md") {
		http.NotFound(writer, request)
		return
	}
	if path == "" {
		path = "index.html"
	}
	if _, err := fs.Stat(s.config.StaticFS, path); err != nil {
		request.URL.Path = "/"
	}
	s.static.ServeHTTP(writer, request)
}

func (s *Server) loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		host, _, err := net.SplitHostPort(request.RemoteAddr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
			writeAPIError(writer, http.StatusForbidden, "LOCAL_LOOPBACK_REQUIRED", "只允许本机访问", false)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) originProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(writer, request)
			return
		}
		parsed, err := url.Parse(origin)
		allowed := err == nil && parsed.Scheme == "http" && parsed.Host == request.Host
		if !allowed && s.config.DevOrigin != "" {
			allowed = origin == strings.TrimRight(s.config.DevOrigin, "/")
		}
		if !allowed {
			writeAPIError(writer, http.StatusForbidden, "LOCAL_ORIGIN_REJECTED", "请求来源不受信任", false)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) csrfProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead && request.Method != http.MethodOptions {
			if request.Header.Get("X-CSRF-Token") != s.csrfToken {
				writeAPIError(writer, http.StatusForbidden, "LOCAL_CSRF_REJECTED", "安全令牌无效，请刷新页面", false)
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("request panic", "path", request.URL.Path)
				writeAPIError(writer, http.StatusInternalServerError, "LOCAL_INTERNAL_ERROR", "本机网关发生内部错误", true)
			}
		}()
		next.ServeHTTP(writer, request)
	})
}

func decodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("请求格式无效: %w", err)
	}
	return nil
}

func writeError(writer http.ResponseWriter, err error) {
	if errors.Is(err, vrchat.ErrInvalidRequest) {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_INVALID_REQUEST", err.Error(), false)
		return
	}
	if errors.Is(err, vrchat.ErrUserAgentNotConfigured) {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_USER_AGENT_REQUIRED", err.Error(), false)
		return
	}
	var upstream *vrchat.UpstreamError
	if errors.As(err, &upstream) {
		code := "VRC_UPSTREAM_ERROR"
		retryable := upstream.StatusCode == http.StatusTooManyRequests || upstream.StatusCode >= 500
		switch upstream.StatusCode {
		case http.StatusUnauthorized:
			code = "VRC_AUTH_REQUIRED"
		case http.StatusForbidden:
			code = "VRC_FORBIDDEN"
		case http.StatusTooManyRequests:
			code = "VRC_RATE_LIMITED"
		}
		extra := map[string]any{}
		if upstream.RetryAfter > 0 {
			extra["retryAfterMs"] = upstream.RetryAfter.Milliseconds()
			writer.Header().Set("Retry-After", fmt.Sprintf("%d", int(upstream.RetryAfter.Seconds())))
		}
		writeAPIErrorWithExtra(writer, upstream.StatusCode, code, upstream.Error(), retryable, extra)
		return
	}
	writeAPIError(writer, http.StatusBadGateway, "VRC_UPSTREAM_UNAVAILABLE", err.Error(), true)
}

func writeAPIError(writer http.ResponseWriter, status int, code, message string, retryable bool) {
	writeAPIErrorWithExtra(writer, status, code, message, retryable, nil)
}

func writeAPIErrorWithExtra(writer http.ResponseWriter, status int, code, message string, retryable bool, extra map[string]any) {
	payload := map[string]any{
		"code": code, "httpStatus": status, "retryable": retryable, "message": message,
	}
	for key, value := range extra {
		payload[key] = value
	}
	writeJSON(writer, status, map[string]any{"error": payload})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func contextWithTimeout(request *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(request.Context(), timeout)
}
