package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

func TestSessionRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	want := []byte("encrypted-cookie-data")
	if err := store.SaveSession(ctx, "default", want); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadSession(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("LoadSession() = %q, want %q", got, want)
	}
	if err := store.DeleteSession(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadSession(ctx, "default"); err != ErrNotFound {
		t.Fatalf("LoadSession() error = %v, want ErrNotFound", err)
	}
}

func TestCacheRoundTripAndExpiry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.SaveCache(ctx, "friends:v1", []byte(`[{"id":"usr_test"}]`), -time.Second); err != nil {
		t.Fatal(err)
	}
	entry, err := store.LoadCache(ctx, "friends:v1")
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Expired || !bytes.Contains(entry.Payload, []byte("usr_test")) {
		t.Fatalf("LoadCache() = %#v", entry)
	}
}

func TestSaveProfile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.SaveProfile(ctx, "usr_test", "测试用户", "https://example.invalid/avatar.png"); err != nil {
		t.Fatal(err)
	}
}

func TestSettingRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.SaveSetting(ctx, "network", `{"mode":"system"}`); err != nil {
		t.Fatal(err)
	}
	value, err := store.LoadSetting(ctx, "network")
	if err != nil || value != `{"mode":"system"}` {
		t.Fatalf("LoadSetting() = %q, %v", value, err)
	}
}

func TestMutualGraphReplacesEdgesAndTracksOptOut(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.SaveMutualGraph(ctx, "usr_a", []string{"usr_b", "usr_c", "usr_a", ""}, false); err != nil {
		t.Fatal(err)
	}
	meta, edges, err := store.LoadMutualGraph(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta) != 1 || meta[0].OptedOut || len(edges) != 2 {
		t.Fatalf("initial graph = %#v, %#v", meta, edges)
	}
	if err := store.SaveMutualGraph(ctx, "usr_a", nil, true); err != nil {
		t.Fatal(err)
	}
	meta, edges, err = store.LoadMutualGraph(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta) != 1 || !meta[0].OptedOut || len(edges) != 0 {
		t.Fatalf("opt-out graph = %#v, %#v", meta, edges)
	}
}

func TestFriendAnnotationRoundTripAndCacheStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	want := model.FriendAnnotation{UserID: "usr_test", Note: "今晚一起玩", Group: "常玩", Color: "#5E7CE2", Tags: []string{"中文", "桌游"}}
	if _, err := store.SaveFriendAnnotation(ctx, want); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListFriendAnnotations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UserID != want.UserID || items[0].Note != want.Note || len(items[0].Tags) != 2 {
		t.Fatalf("ListFriendAnnotations() = %#v", items)
	}
	if err := store.SaveCache(ctx, "friends:v1", []byte(`[{"id":"usr_test"}]`), time.Hour); err != nil {
		t.Fatal(err)
	}
	stats, err := store.CacheStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DatabaseBytes <= 0 || stats.EntityEntries != 1 || stats.EntityBytes <= 0 || stats.AnnotationCount != 1 {
		t.Fatalf("CacheStats() = %#v", stats)
	}
	if err := store.DeleteFriendAnnotation(ctx, want.UserID); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListFriendAnnotations(ctx)
	if err != nil || len(items) != 0 {
		t.Fatalf("annotations after delete = %#v, %v", items, err)
	}
}

func TestWorldFavoritesAndSanitizedActivityHistory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	favorite, err := store.SaveWorldFavorite(ctx, model.WorldFavorite{World: model.World{ID: "wrld_test", Name: "测试世界"}, Note: "适合桌游"})
	if err != nil || favorite.CreatedAt.IsZero() {
		t.Fatalf("SaveWorldFavorite() = %#v, %v", favorite, err)
	}
	favorites, err := store.ListWorldFavorites(ctx)
	if err != nil || len(favorites) != 1 || favorites[0].Note != "适合桌游" {
		t.Fatalf("ListWorldFavorites() = %#v, %v", favorites, err)
	}

	content := []byte(`{"userId":"usr_alpha","displayName":"Alpha","location":"wrld_test:123~private(usr_owner)~nonce(secret)"}`)
	if err := store.RecordDomainEvent(ctx, model.DomainEvent{ID: "evt-history", Type: "vrc.friend-location", ObservedAt: time.Now(), Content: content}); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListActivityEvents(ctx, 30, 100)
	if err != nil || len(events) != 1 || events[0].WorldID != "wrld_test" || events[0].Summary != "Alpha 切换了世界" {
		t.Fatalf("ListActivityEvents() = %#v, %v", events, err)
	}
	encoded, _ := json.Marshal(events)
	if bytes.Contains(encoded, []byte("nonce")) || bytes.Contains(encoded, []byte("secret")) || bytes.Contains(encoded, []byte("123")) {
		t.Fatalf("activity history leaked instance details: %s", encoded)
	}
	insights, err := store.ActivityInsights(ctx, 30)
	if err != nil || insights.TotalEvents != 1 || len(insights.TopUsers) != 1 || len(insights.Heatmap) != 1 {
		t.Fatalf("ActivityInsights() = %#v, %v", insights, err)
	}
	joinedAt := time.Now().Add(-50 * time.Minute)
	leftAt := time.Now().Add(-20 * time.Minute)
	player := []byte(`{"userId":"usr_alpha","displayName":"Alpha"}`)
	if err := store.RecordDomainEvent(ctx, model.DomainEvent{ID: "evt-joined", Type: "game.player-joined", ObservedAt: joinedAt, Content: player}); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordDomainEvent(ctx, model.DomainEvent{ID: "evt-left", Type: "game.player-left", ObservedAt: leftAt, Content: player}); err != nil {
		t.Fatal(err)
	}
	friendInsights, err := store.FriendActivityInsights(ctx, "usr_alpha", 30)
	if err != nil || friendInsights.TotalEvents != 3 || friendInsights.TogetherMinutes != 30 || friendInsights.TogetherSessions != 1 || friendInsights.SourceCounts["gameLog"] != 2 || friendInsights.SourceCounts["pipeline"] != 1 || friendInsights.FirstObservedAt == nil || len(friendInsights.Timeline) != 3 || friendInsights.LastMetAt == nil {
		t.Fatalf("FriendActivityInsights() = %#v, %v", friendInsights, err)
	}
	if err := store.ClearActivityEvents(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteWorldFavorite(ctx, "wrld_test"); err != nil {
		t.Fatal(err)
	}
}
