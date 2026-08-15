package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/local/vrc-web-companion/gateway/internal/model"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db   *sql.DB
	path string
}

func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db, path: path}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ready(ctx context.Context) bool {
	return s.db.PingContext(ctx) == nil
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS schema_migration (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS secure_session (
			account_id TEXT PRIMARY KEY,
			encrypted_cookie_jar BLOB NOT NULL,
			encryption_version INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS account_profile (
			account_id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			last_login_at TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS entity_cache (
			cache_key TEXT PRIMARY KEY,
			payload_json BLOB NOT NULL,
			fetched_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS app_setting (
			setting_key TEXT PRIMARY KEY,
			setting_value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS mutual_graph_friend (
			friend_id TEXT PRIMARY KEY,
			fetched_at TEXT NOT NULL,
			opted_out INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS mutual_graph_edge (
			friend_id TEXT NOT NULL,
			mutual_id TEXT NOT NULL,
			observed_at TEXT NOT NULL,
			PRIMARY KEY(friend_id, mutual_id),
			FOREIGN KEY(friend_id) REFERENCES mutual_graph_friend(friend_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS friend_annotation (
			user_id TEXT PRIMARY KEY,
			note TEXT NOT NULL DEFAULT '',
			group_name TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '',
			tags_json TEXT NOT NULL DEFAULT '[]',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS world_favorite (
			world_id TEXT PRIMARY KEY,
			world_json BLOB NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS activity_event (
			event_id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			user_id TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			world_id TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL,
			observed_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_event_observed_at ON activity_event(observed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_event_user ON activity_event(user_id, observed_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply database migration: %w", err)
		}
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migration(version, applied_at) VALUES(1, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migration(version, applied_at) VALUES(2, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migration(version, applied_at) VALUES(3, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migration(version, applied_at) VALUES(4, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) SaveWorldFavorite(ctx context.Context, favorite model.WorldFavorite) (model.WorldFavorite, error) {
	favorite.CreatedAt = time.Now().UTC()
	payload, err := json.Marshal(favorite.World)
	if err != nil {
		return model.WorldFavorite{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO world_favorite(world_id, world_json, note, created_at) VALUES(?, ?, ?, ?)
		ON CONFLICT(world_id) DO UPDATE SET world_json=excluded.world_json, note=excluded.note`,
		favorite.World.ID, payload, favorite.Note, favorite.CreatedAt.Format(time.RFC3339Nano))
	return favorite, err
}

func (s *Store) ListWorldFavorites(ctx context.Context) ([]model.WorldFavorite, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT world_json, note, created_at FROM world_favorite ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.WorldFavorite{}
	for rows.Next() {
		var item model.WorldFavorite
		var payload []byte
		var createdAt string
		if err := rows.Scan(&payload, &item.Note, &createdAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item.World); err != nil {
			return nil, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteWorldFavorite(ctx context.Context, worldID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM world_favorite WHERE world_id = ?`, worldID)
	return err
}

func (s *Store) RecordDomainEvent(ctx context.Context, event model.DomainEvent) error {
	if (!strings.HasPrefix(event.Type, "vrc.") && !strings.HasPrefix(event.Type, "game.")) || event.Type == "vrc." || event.Type == "game." {
		return nil
	}
	var content map[string]any
	if len(event.Content) > 0 {
		_ = json.Unmarshal(event.Content, &content)
	}
	stringValue := func(keys ...string) string { return nestedString(content, keys...) }
	userID := stringValue("userId", "senderUserId", "id")
	displayName := stringValue("displayName", "senderUsername", "userDisplayName")
	location := stringValue("location", "worldId")
	worldID := ""
	if strings.HasPrefix(location, "wrld_") {
		worldID = strings.SplitN(location, ":", 2)[0]
	}
	summary := activitySummary(event.Type, displayName)
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO activity_event(event_id, event_type, user_id, display_name, world_id, summary, observed_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`, event.ID, event.Type, userID, displayName, worldID, summary, event.ObservedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM activity_event WHERE observed_at < ?`, time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339Nano))
	return err
}

func activitySummary(eventType, displayName string) string {
	name := displayName
	if name == "" {
		name = "一位好友"
	}
	switch {
	case strings.Contains(eventType, "friend-online"):
		return name + " 上线"
	case strings.Contains(eventType, "friend-offline"):
		return name + " 离线"
	case strings.Contains(eventType, "friend-location"):
		return name + " 切换了世界"
	case strings.Contains(eventType, "friend-active"):
		return name + " 进入活跃状态"
	case strings.Contains(eventType, "friend-update"):
		return name + " 更新了资料"
	case strings.Contains(eventType, "notification"):
		return "收到一条 VRChat 通知"
	case strings.Contains(eventType, "player-joined"):
		return name + " 进入了当前实例"
	case strings.Contains(eventType, "player-left"):
		return name + " 离开了当前实例"
	case strings.Contains(eventType, "world-entering"):
		return "正在进入一个世界"
	case eventType == "game.location":
		return "VRChat 已切换实例"
	default:
		return "观察到 " + strings.TrimPrefix(strings.TrimPrefix(eventType, "vrc."), "game.") + " 事件"
	}
}

func (s *Store) ListActivityEvents(ctx context.Context, days, limit int) ([]model.ActivityEvent, error) {
	if days < 1 {
		days = 30
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, event_type, user_id, display_name, world_id, summary, observed_at
		FROM activity_event WHERE observed_at >= ? ORDER BY observed_at DESC LIMIT ?`,
		time.Now().UTC().Add(-time.Duration(days)*24*time.Hour).Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.ActivityEvent{}
	for rows.Next() {
		var item model.ActivityEvent
		var observedAt string
		if err := rows.Scan(&item.ID, &item.Type, &item.UserID, &item.DisplayName, &item.WorldID, &item.Summary, &observedAt); err != nil {
			return nil, err
		}
		item.ObservedAt, err = time.Parse(time.RFC3339Nano, observedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	entry, err := s.LoadCache(ctx, "friends:v1")
	if err == nil {
		var friends []model.Friend
		if json.Unmarshal(entry.Payload, &friends) == nil {
			byID := make(map[string]string, len(friends))
			for _, friend := range friends {
				if friend.DisplayName != "" {
					byID[friend.ID] = friend.DisplayName
				}
			}
			for index := range items {
				if name := byID[items[index].UserID]; name != "" {
					items[index].DisplayName = name
					items[index].Summary = activitySummary(items[index].Type, name)
				}
			}
		}
	}
	return items, nil
}

func nestedString(value any, keys ...string) string {
	keySet := make(map[string]bool, len(keys))
	for _, key := range keys {
		keySet[key] = true
	}
	var walk func(any) string
	walk = func(current any) string {
		switch typed := current.(type) {
		case map[string]any:
			for _, key := range keys {
				if raw, ok := typed[key].(string); ok && raw != "" {
					return raw
				}
			}
			for key, child := range typed {
				if keySet[key] {
					continue
				}
				if found := walk(child); found != "" {
					return found
				}
			}
		case []any:
			for _, child := range typed {
				if found := walk(child); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return walk(value)
}

func (s *Store) ActivityInsights(ctx context.Context, days int) (model.ActivityInsights, error) {
	events, err := s.ListActivityEvents(ctx, days, 500)
	if err != nil {
		return model.ActivityInsights{}, err
	}
	result := model.ActivityInsights{TotalEvents: len(events), GeneratedAt: time.Now().UTC()}
	dates := map[string]bool{}
	buckets := map[[2]int]int{}
	top := map[string]*model.ActivityTopUser{}
	for _, event := range events {
		local := event.ObservedAt.In(time.Local)
		dates[local.Format("2006-01-02")] = true
		key := [2]int{int(local.Weekday()), local.Hour()}
		buckets[key]++
		if event.UserID != "" {
			item := top[event.UserID]
			if item == nil {
				item = &model.ActivityTopUser{UserID: event.UserID, DisplayName: event.DisplayName}
				top[event.UserID] = item
			}
			item.Count++
		}
	}
	result.CoverageDays = len(dates)
	for key, count := range buckets {
		result.Heatmap = append(result.Heatmap, model.ActivityBucket{Weekday: key[0], Hour: key[1], Count: count})
	}
	for _, item := range top {
		result.TopUsers = append(result.TopUsers, *item)
	}
	sort.Slice(result.TopUsers, func(i, j int) bool { return result.TopUsers[i].Count > result.TopUsers[j].Count })
	if len(result.TopUsers) > 8 {
		result.TopUsers = result.TopUsers[:8]
	}
	return result, nil
}

func (s *Store) FriendActivityInsights(ctx context.Context, userID string, days int) (model.FriendActivityInsights, error) {
	if days < 1 || days > 90 {
		days = 30
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, event_type, user_id, display_name, world_id, summary, observed_at
		FROM activity_event WHERE user_id = ? AND observed_at >= ?
		ORDER BY observed_at ASC LIMIT 1000`, userID,
		time.Now().UTC().Add(-time.Duration(days)*24*time.Hour).Format(time.RFC3339Nano))
	if err != nil {
		return model.FriendActivityInsights{}, err
	}
	defer rows.Close()
	events := make([]model.ActivityEvent, 0, 128)
	for rows.Next() {
		var item model.ActivityEvent
		var observedAt string
		if err := rows.Scan(&item.ID, &item.Type, &item.UserID, &item.DisplayName, &item.WorldID, &item.Summary, &observedAt); err != nil {
			return model.FriendActivityInsights{}, err
		}
		item.ObservedAt, err = time.Parse(time.RFC3339Nano, observedAt)
		if err != nil {
			return model.FriendActivityInsights{}, err
		}
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return model.FriendActivityInsights{}, err
	}

	result := model.FriendActivityInsights{UserID: userID, TotalEvents: len(events), GeneratedAt: time.Now().UTC()}
	dates := map[string]bool{}
	hours := map[int]int{}
	worlds := map[string]*model.FriendActivityWorld{}
	var joinedAt *time.Time
	for _, event := range events {
		local := event.ObservedAt.In(time.Local)
		dates[local.Format("2006-01-02")] = true
		hours[local.Hour()]++
		if event.WorldID != "" {
			item := worlds[event.WorldID]
			if item == nil {
				item = &model.FriendActivityWorld{WorldID: event.WorldID}
				worlds[event.WorldID] = item
			}
			item.Count++
			if event.ObservedAt.After(item.LastSeenAt) {
				item.LastSeenAt = event.ObservedAt
			}
		}
		if event.Type == "game.player-joined" {
			joined := event.ObservedAt
			joinedAt = &joined
			result.LastMetAt = &joined
		}
		if event.Type == "game.player-left" {
			left := event.ObservedAt
			result.LastMetAt = &left
			if joinedAt != nil && left.After(*joinedAt) {
				duration := left.Sub(*joinedAt)
				if duration <= 12*time.Hour {
					result.TogetherMinutes += int(duration.Round(time.Minute) / time.Minute)
				}
			}
			joinedAt = nil
		}
	}
	result.CoverageDays = len(dates)
	for hour, count := range hours {
		result.ActiveHours = append(result.ActiveHours, model.FriendActivityHour{Hour: hour, Count: count})
	}
	sort.Slice(result.ActiveHours, func(i, j int) bool {
		if result.ActiveHours[i].Count != result.ActiveHours[j].Count {
			return result.ActiveHours[i].Count > result.ActiveHours[j].Count
		}
		return result.ActiveHours[i].Hour < result.ActiveHours[j].Hour
	})
	if len(result.ActiveHours) > 4 {
		result.ActiveHours = result.ActiveHours[:4]
	}
	for _, item := range worlds {
		result.CommonWorlds = append(result.CommonWorlds, *item)
	}
	sort.Slice(result.CommonWorlds, func(i, j int) bool {
		if result.CommonWorlds[i].Count != result.CommonWorlds[j].Count {
			return result.CommonWorlds[i].Count > result.CommonWorlds[j].Count
		}
		return result.CommonWorlds[i].LastSeenAt.After(result.CommonWorlds[j].LastSeenAt)
	})
	if len(result.CommonWorlds) > 5 {
		result.CommonWorlds = result.CommonWorlds[:5]
	}
	for index := len(events) - 1; index >= 0 && len(result.Timeline) < 20; index-- {
		result.Timeline = append(result.Timeline, events[index])
	}
	if result.ActiveHours == nil {
		result.ActiveHours = []model.FriendActivityHour{}
	}
	if result.CommonWorlds == nil {
		result.CommonWorlds = []model.FriendActivityWorld{}
	}
	if result.Timeline == nil {
		result.Timeline = []model.ActivityEvent{}
	}
	return result, nil
}

func (s *Store) ClearActivityEvents(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM activity_event`)
	return err
}

func (s *Store) SaveFriendAnnotation(ctx context.Context, item model.FriendAnnotation) (model.FriendAnnotation, error) {
	item.UpdatedAt = time.Now().UTC()
	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return model.FriendAnnotation{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO friend_annotation(user_id, note, group_name, color, tags_json, updated_at)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET note=excluded.note, group_name=excluded.group_name,
			color=excluded.color, tags_json=excluded.tags_json, updated_at=excluded.updated_at`,
		item.UserID, item.Note, item.Group, item.Color, string(tagsJSON), item.UpdatedAt.Format(time.RFC3339Nano))
	return item, err
}

func (s *Store) ListFriendAnnotations(ctx context.Context) ([]model.FriendAnnotation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, note, group_name, color, tags_json, updated_at FROM friend_annotation ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.FriendAnnotation{}
	for rows.Next() {
		var item model.FriendAnnotation
		var tagsJSON, updatedAt string
		if err := rows.Scan(&item.UserID, &item.Note, &item.Group, &item.Color, &tagsJSON, &updatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tagsJSON), &item.Tags); err != nil {
			return nil, err
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteFriendAnnotation(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM friend_annotation WHERE user_id = ?`, userID)
	return err
}

func (s *Store) CacheStats(ctx context.Context) (model.CacheStats, error) {
	var stats model.CacheStats
	if info, err := os.Stat(s.path); err == nil {
		stats.DatabaseBytes = info.Size()
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(LENGTH(payload_json)), 0) FROM entity_cache`).Scan(&stats.EntityEntries, &stats.EntityBytes); err != nil {
		return model.CacheStats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(CASE WHEN cache_key LIKE 'groups:%' OR cache_key LIKE 'group-%' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN cache_key LIKE 'favorite-avatars:%' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN cache_key LIKE 'world:%' OR cache_key LIKE 'world-search:%' THEN 1 ELSE 0 END), 0)
		FROM entity_cache`).Scan(&stats.GroupEntries, &stats.AvatarEntries, &stats.WorldEntries); err != nil {
		return model.CacheStats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM friend_annotation`).Scan(&stats.AnnotationCount); err != nil {
		return model.CacheStats{}, err
	}
	return stats, nil
}

func (s *Store) ClearEntityCache(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM entity_cache`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type MutualGraphMeta struct {
	FriendID  string
	FetchedAt time.Time
	OptedOut  bool
}

type MutualGraphEdge struct {
	FriendID string
	MutualID string
}

func (s *Store) SaveMutualGraph(ctx context.Context, friendID string, mutualIDs []string, optedOut bool) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO mutual_graph_friend(friend_id, fetched_at, opted_out)
		VALUES(?, ?, ?)
		ON CONFLICT(friend_id) DO UPDATE SET fetched_at = excluded.fetched_at, opted_out = excluded.opted_out`,
		friendID, now, optedOut); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mutual_graph_edge WHERE friend_id = ?`, friendID); err != nil {
		return err
	}
	if !optedOut {
		for _, mutualID := range mutualIDs {
			if mutualID == "" || mutualID == friendID {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO mutual_graph_edge(friend_id, mutual_id, observed_at) VALUES(?, ?, ?)`,
				friendID, mutualID, now); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) LoadMutualGraph(ctx context.Context) ([]MutualGraphMeta, []MutualGraphEdge, error) {
	metaRows, err := s.db.QueryContext(ctx, `SELECT friend_id, fetched_at, opted_out FROM mutual_graph_friend`)
	if err != nil {
		return nil, nil, err
	}
	defer metaRows.Close()
	var meta []MutualGraphMeta
	for metaRows.Next() {
		var item MutualGraphMeta
		var fetchedAt string
		if err := metaRows.Scan(&item.FriendID, &fetchedAt, &item.OptedOut); err != nil {
			return nil, nil, err
		}
		item.FetchedAt, err = time.Parse(time.RFC3339Nano, fetchedAt)
		if err != nil {
			return nil, nil, err
		}
		meta = append(meta, item)
	}
	if err := metaRows.Err(); err != nil {
		return nil, nil, err
	}
	edgeRows, err := s.db.QueryContext(ctx, `SELECT friend_id, mutual_id FROM mutual_graph_edge`)
	if err != nil {
		return nil, nil, err
	}
	defer edgeRows.Close()
	var edges []MutualGraphEdge
	for edgeRows.Next() {
		var edge MutualGraphEdge
		if err := edgeRows.Scan(&edge.FriendID, &edge.MutualID); err != nil {
			return nil, nil, err
		}
		edges = append(edges, edge)
	}
	return meta, edges, edgeRows.Err()
}

func (s *Store) SaveSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_setting(setting_key, setting_value, updated_at)
		VALUES(?, ?, ?)
		ON CONFLICT(setting_key) DO UPDATE SET
			setting_value = excluded.setting_value,
			updated_at = excluded.updated_at`,
		key, value, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) LoadSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT setting_value FROM app_setting WHERE setting_key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

type CacheEntry struct {
	Payload   []byte
	FetchedAt time.Time
	Expired   bool
}

func (s *Store) SaveCache(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entity_cache(cache_key, payload_json, fetched_at, expires_at)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(cache_key) DO UPDATE SET
			payload_json = excluded.payload_json,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at`,
		key, payload, now.Format(time.RFC3339Nano), now.Add(ttl).Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) LoadCache(ctx context.Context, key string) (CacheEntry, error) {
	var payload []byte
	var fetchedAtText, expiresAtText string
	err := s.db.QueryRowContext(ctx,
		`SELECT payload_json, fetched_at, expires_at FROM entity_cache WHERE cache_key = ?`, key,
	).Scan(&payload, &fetchedAtText, &expiresAtText)
	if errors.Is(err, sql.ErrNoRows) {
		return CacheEntry{}, ErrNotFound
	}
	if err != nil {
		return CacheEntry{}, err
	}
	fetchedAt, err := time.Parse(time.RFC3339Nano, fetchedAtText)
	if err != nil {
		return CacheEntry{}, fmt.Errorf("parse cache fetched time: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtText)
	if err != nil {
		return CacheEntry{}, fmt.Errorf("parse cache expiry time: %w", err)
	}
	return CacheEntry{Payload: payload, FetchedAt: fetchedAt, Expired: time.Now().UTC().After(expiresAt)}, nil
}

func (s *Store) SaveSession(ctx context.Context, accountID string, encrypted []byte) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO secure_session(account_id, encrypted_cookie_jar, encryption_version, updated_at)
		VALUES(?, ?, 1, ?)
		ON CONFLICT(account_id) DO UPDATE SET
			encrypted_cookie_jar = excluded.encrypted_cookie_jar,
			encryption_version = excluded.encryption_version,
			updated_at = excluded.updated_at`,
		accountID, encrypted, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) LoadSession(ctx context.Context, accountID string) ([]byte, error) {
	var encrypted []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT encrypted_cookie_jar FROM secure_session WHERE account_id = ?`, accountID,
	).Scan(&encrypted)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return encrypted, err
}

func (s *Store) DeleteSession(ctx context.Context, accountID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM secure_session WHERE account_id = ?`, accountID)
	return err
}

func (s *Store) SaveProfile(ctx context.Context, accountID, displayName, avatarURL string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE account_profile SET active = 0;
		INSERT INTO account_profile(account_id, display_name, avatar_url, last_login_at, active)
		VALUES(?, ?, ?, ?, 1)
		ON CONFLICT(account_id) DO UPDATE SET
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			last_login_at = excluded.last_login_at,
			active = 1`,
		accountID, displayName, avatarURL, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}
