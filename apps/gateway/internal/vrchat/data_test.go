package vrchat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
)

func TestListFriendsMergesOnlineAndOfflineAndFallsBackToCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/auth/user/friends" {
			http.NotFound(writer, request)
			return
		}
		if request.URL.Query().Get("offline") == "true" {
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": "usr_offline", "displayName": "Beta", "last_activity": "2026-08-17T10:00:00Z", "last_login": "2026-08-16T10:00:00Z", "last_mobile": "2026-08-15T10:00:00Z", "bio": "Hello"}})
			return
		}
		_ = json.NewEncoder(writer).Encode([]map[string]any{{
			"id": "usr_online", "displayName": "Alpha", "location": "wrld_test", "platform": "standalonewindows",
		}})
	}))
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	client, err := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	client.limiter = newRequestLimiter(1000, 1000)
	result, err := client.ListFriends(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "live" || len(result.Items) != 2 || !result.Items[0].Online {
		t.Fatalf("ListFriends() = %#v", result)
	}
	if result.Items[1].LastActivity == "" || result.Items[1].LastMobile == "" || result.Items[1].Bio != "Hello" {
		t.Fatalf("offline friend fields were discarded: %#v", result.Items[1])
	}
	server.Close()
	cached, err := client.ListFriends(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cached.Source != "cache" || len(cached.Items) != 2 {
		t.Fatalf("cached ListFriends() = %#v", cached)
	}
}

func TestGetUserMergesPublicPrivateAndMutualProfileFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/users/usr_alice":
			_ = json.NewEncoder(writer).Encode(map[string]any{"id": "usr_alice", "displayName": "Alice", "date_joined": "2020-01-02", "isFriend": true})
		case "/profile/usr_alice":
			_ = json.NewEncoder(writer).Encode(map[string]any{"id": "usr_alice", "displayName": "Alice Public", "bio": "Public bio", "languages": []string{"eng", "jpn"}, "hasVrcPlus": true, "representedGroup": map[string]any{"id": "grp_test", "name": "Test Group"}, "trustTags": []string{"system_trust_known"}})
		case "/profile/usr_alice/private":
			_ = json.NewEncoder(writer).Encode(map[string]any{"isFriend": true, "status": "active", "activity": map[string]any{"state": "active", "platform": "web", "last_activity": "2026-08-18T08:00:00Z"}})
		case "/users/usr_alice/mutuals":
			_ = json.NewEncoder(writer).Encode(map[string]int{"friends": 7, "groups": 2})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)
	result, err := client.GetUser(context.Background(), "usr_alice")
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("GetUser() = %#v, %v", result, err)
	}
	item := result.Items[0]
	if item.DisplayName != "Alice Public" || item.State != "active" || item.LastActivity == "" || item.MutualFriendCount != 7 || item.MutualGroupCount != 2 || item.RepresentedGroup == nil || !item.HasVRCPlus || item.TrustLevel != "user" || len(item.ProfileSources) != 4 {
		t.Fatalf("merged profile = %#v", item)
	}
	byID, err := client.SearchUsers(context.Background(), "usr_alice", 12)
	if err != nil || len(byID.Items) != 1 || byID.Items[0].DisplayName != "Alice Public" {
		t.Fatalf("SearchUsers(by ID) = %#v, %v", byID, err)
	}
}

func TestSearchWorlds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("search") != "中文 世界" {
			t.Fatalf("search = %q", request.URL.Query().Get("search"))
		}
		_ = json.NewEncoder(writer).Encode([]map[string]any{{
			"id": "wrld_test", "name": "中文世界", "authorName": "Builder", "capacity": 32,
		}})
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)
	result, err := client.SearchWorlds(context.Background(), "中文 世界", 0, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "中文世界" {
		t.Fatalf("SearchWorlds() = %#v", result)
	}
}

func TestUpdateSelfProfileWritesOnlyCurrentUserAndRefreshes(t *testing.T) {
	var userWrite map[string]any
	var profileWrite map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPut && request.URL.Path == "/users/usr_me":
			if err := json.NewDecoder(request.Body).Decode(&userWrite); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"ok": true})
		case request.Method == http.MethodPut && request.URL.Path == "/profile/usr_me":
			if err := json.NewDecoder(request.Body).Decode(&profileWrite); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"ok": true})
		case request.Method == http.MethodGet && request.URL.Path == "/users/usr_me":
			_ = json.NewEncoder(writer).Encode(map[string]any{"id": "usr_me", "displayName": "Harbor User", "status": "active", "bio": "Hello", "bioLinks": []string{"https://example.com"}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)
	client.currentUserID = "usr_me"
	result, err := client.UpdateSelfProfile(context.Background(), model.SelfProfileUpdate{
		Status: "active", StatusDescription: "Available", Pronouns: "they/them", Bio: "Hello",
		BioLinks: []string{"https://example.com", ""}, AllowAvatarCopying: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "usr_me" || userWrite["statusDescription"] != "Available" || profileWrite["bio"] != "Hello" {
		t.Fatalf("result=%#v user=%#v profile=%#v", result, userWrite, profileWrite)
	}
	links, ok := profileWrite["bioLinks"].([]any)
	if !ok || len(links) != 1 {
		t.Fatalf("bioLinks = %#v", profileWrite["bioLinks"])
	}
}

func TestGroupsPostsInstancesAndFavoriteAvatars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/users/usr_me/groups":
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"groupId": "grp_test", "name": "测试群组", "shortCode": "TEST", "memberCount": 42, "isRepresenting": true}})
		case "/groups/grp_test/posts":
			_ = json.NewEncoder(writer).Encode(map[string]any{"posts": []map[string]any{{"id": "not_test", "groupId": "grp_test", "title": "公告", "text": "今晚集合"}}})
		case "/groups/grp_test/instances":
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"instanceId": "123", "location": "wrld_test:123", "memberCount": 3, "world": map[string]any{"id": "wrld_test", "name": "Group World"}}})
		case "/groups/grp_test/members":
			if request.URL.Query().Get("n") != "100" || request.URL.Query().Get("offset") != "0" {
				t.Fatalf("unexpected member query: %s", request.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"groupId": "grp_test", "userId": "usr_member", "roleIds": []string{"grol_test"}, "user": map[string]any{"id": "usr_member", "displayName": "Public Member", "profilePicOverrideThumbnail": "https://example.com/member.png"}}})
		case "/calendar/grp_test":
			_ = json.NewEncoder(writer).Encode(map[string]any{"results": []map[string]any{{"id": "cal_test", "title": "Friday Meetup", "startsAt": "2026-08-21T12:00:00Z", "endsAt": "2026-08-21T14:00:00Z", "interestedUserCount": 8, "userInterest": map[string]any{"isFollowing": true}}}})
		case "/avatars/favorites":
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": "avtr_test", "name": "Avatar", "authorName": "Maker", "performance": map[string]any{"standalonewindows": "Excellent", "standalonewindows-sort": 5}, "unityPackages": []map[string]string{{"platform": "standalonewindows"}}}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)
	groups, err := client.ListUserGroups(context.Background(), "usr_me", false)
	if err != nil || len(groups.Items) != 1 || groups.Items[0].Name != "测试群组" {
		t.Fatalf("groups = %#v, %v", groups, err)
	}
	posts, err := client.ListGroupPosts(context.Background(), "grp_test", false)
	if err != nil || len(posts.Items) != 1 || posts.Items[0].Title != "公告" {
		t.Fatalf("posts = %#v, %v", posts, err)
	}
	instances, err := client.ListGroupInstances(context.Background(), "grp_test", false)
	if err != nil || len(instances.Items) != 1 || instances.Items[0].World.Name != "Group World" {
		t.Fatalf("instances = %#v, %v", instances, err)
	}
	members, err := client.ListGroupMembers(context.Background(), "grp_test", 100, false)
	if err != nil || len(members.Items) != 1 || members.Items[0].DisplayName != "Public Member" || members.Items[0].UserID != "usr_member" {
		t.Fatalf("members = %#v, %v", members, err)
	}
	calendar, err := client.ListGroupCalendarEvents(context.Background(), "grp_test", "2026-08", false)
	if err != nil || len(calendar.Items) != 1 || calendar.Items[0].Title != "Friday Meetup" || !calendar.Items[0].Following {
		t.Fatalf("calendar = %#v, %v", calendar, err)
	}
	avatars, err := client.ListFavoriteAvatars(context.Background(), false)
	if err != nil || len(avatars.Items) != 1 || len(avatars.Items[0].Platforms) != 1 {
		t.Fatalf("avatars = %#v, %v", avatars, err)
	}
}

func TestGetUserAndMutualFriends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/users/usr_friend":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"id": "usr_friend", "displayName": "Friend", "bio": "Hello", "date_joined": "2020-01-02",
				"location": "wrld_test:123~region(jp)", "tags": []string{"system_trust_trusted"},
			})
		case "/users/usr_friend/mutuals/friends":
			if request.URL.Query().Get("n") != "100" || request.URL.Query().Get("offset") != "0" {
				t.Fatalf("unexpected mutual query: %s", request.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode([]map[string]any{
				{"id": "usr_b", "displayName": "Beta"},
				{"id": "usr_a", "displayName": "Alpha"},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	client, err := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	client.limiter = newRequestLimiter(1000, 1000)

	profile, err := client.GetUser(context.Background(), "usr_friend")
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Items) != 1 || profile.Items[0].TrustLevel != "known" || profile.Items[0].DateJoined != "2020-01-02" {
		t.Fatalf("GetUser() = %#v", profile)
	}
	mutuals, err := client.ListMutualFriends(context.Background(), "usr_friend", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutuals.Items) != 2 || mutuals.Items[0].DisplayName != "Alpha" {
		t.Fatalf("ListMutualFriends() = %#v", mutuals)
	}

	server.Close()
	cached, err := client.GetUser(context.Background(), "usr_friend")
	if err != nil || cached.Source != "cache" || len(cached.Items) != 1 {
		t.Fatalf("cached GetUser() = %#v, %v", cached, err)
	}
}

func TestFriendNetworkDeduplicatesObservedEdges(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	friends := []map[string]any{
		{"id": "usr_a", "displayName": "Alpha", "online": true},
		{"id": "usr_b", "displayName": "Beta", "online": false},
		{"id": "usr_c", "displayName": "Gamma", "online": false},
	}
	payload, _ := json.Marshal(friends)
	if err := store.SaveCache(ctx, friendsCacheKey, payload, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMutualGraph(ctx, "usr_a", []string{"usr_b", "usr_unknown"}, false); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMutualGraph(ctx, "usr_b", []string{"usr_a"}, false); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMutualGraph(ctx, "usr_c", nil, true); err != nil {
		t.Fatal(err)
	}
	client, err := NewClient("https://example.invalid", "Test/1 contact@example.com", store, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := client.FriendNetwork(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if graph.TotalFriends != 3 || graph.ScannedCount != 3 || graph.OptedOutCount != 1 || len(graph.Edges) != 1 {
		t.Fatalf("FriendNetwork() = %#v", graph)
	}
}

func TestGetUserRejectsInvalidID(t *testing.T) {
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient("https://example.invalid", "Test/1 contact@example.com", store, testProtector{})
	if _, err := client.GetUser(context.Background(), "../../auth"); err == nil {
		t.Fatal("GetUser() accepted an invalid id")
	}
}

func TestUserDiscoveryFriendStatusAndRequest(t *testing.T) {
	requestSent := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/users":
			if request.URL.Query().Get("search") != "Alice" || request.URL.Query().Get("n") != "12" {
				t.Fatalf("unexpected search query: %s", request.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": "usr_alice", "displayName": "Alice"}})
		case "/user/usr_alice/friendStatus":
			_ = json.NewEncoder(writer).Encode(map[string]bool{"isFriend": false, "outgoingRequest": false, "incomingRequest": false})
		case "/user/usr_alice/friendRequest":
			if request.Method != http.MethodPost {
				t.Fatalf("method = %s", request.Method)
			}
			requestSent = true
			writer.WriteHeader(http.StatusOK)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)
	users, err := client.SearchUsers(context.Background(), "Alice", 12)
	if err != nil || len(users.Items) != 1 || users.Items[0].DisplayName != "Alice" {
		t.Fatalf("SearchUsers() = %#v, %v", users, err)
	}
	status, err := client.FriendStatus(context.Background(), "usr_alice")
	if err != nil || status.IsFriend || status.OutgoingRequest {
		t.Fatalf("FriendStatus() = %#v, %v", status, err)
	}
	if err := client.SendFriendRequest(context.Background(), "usr_alice"); err != nil || !requestSent {
		t.Fatalf("SendFriendRequest() = %v, sent=%v", err, requestSent)
	}
}

func TestInstanceNotificationsAndExplicitAction(t *testing.T) {
	actionCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/instances/wrld_test:123~region(jp)":
			_ = json.NewEncoder(writer).Encode(map[string]any{"worldId": "wrld_test", "instanceId": "123~region(jp)", "userCount": 12, "capacity": 40, "region": "jp", "active": true})
		case "/auth/user/notifications":
			_ = json.NewEncoder(writer).Encode([]map[string]any{{
				"id": "not_test", "type": "invite", "senderUsername": "Alpha", "message": "Join me", "seen": false,
				"details": `{"worldId":"wrld_test","worldName":"Test World","instanceId":"123~region(jp)"}`,
			}})
		case "/auth/user/notifications/not_test/see":
			if request.Method != http.MethodPut {
				t.Fatalf("notification action method = %s", request.Method)
			}
			actionCalled = true
			writer.WriteHeader(http.StatusOK)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)

	instance, err := client.GetInstance(context.Background(), "wrld_test:123~region(jp)")
	if err != nil || instance.UserCount != 12 || instance.Region != "jp" {
		t.Fatalf("GetInstance() = %#v, %v", instance, err)
	}
	items, err := client.ListNotifications(context.Background())
	if err != nil || len(items) != 1 || items[0].WorldID != "wrld_test" || items[0].InstanceID == "" {
		t.Fatalf("ListNotifications() = %#v, %v", items, err)
	}
	if err := client.NotificationAction(context.Background(), "not_test", "see"); err != nil || !actionCalled {
		t.Fatalf("NotificationAction() = %v, called=%v", err, actionCalled)
	}
}

func TestInviteAndBoopUseCookieAuthAndCurrentPayloads(t *testing.T) {
	var inviteBody map[string]any
	var boopBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		switch request.URL.Path {
		case "/invite/usr_friend":
			if err := json.NewDecoder(request.Body).Decode(&inviteBody); err != nil {
				t.Fatal(err)
			}
		case "/users/usr_friend/boop":
			if err := json.NewDecoder(request.Body).Decode(&boopBody); err != nil {
				t.Fatal(err)
			}
		default:
			http.NotFound(writer, request)
			return
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	store, _ := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	t.Cleanup(func() { _ = store.Close() })
	client, _ := NewClient(server.URL, "Test/1 contact@example.com", store, testProtector{})
	client.limiter = newRequestLimiter(1000, 1000)
	if err := client.SendInvite(context.Background(), "usr_friend", "wrld_test:123~region(jp)"); err != nil {
		t.Fatal(err)
	}
	if err := client.SendBoop(context.Background(), "usr_friend", "default_hand_wave"); err != nil {
		t.Fatal(err)
	}
	if inviteBody["instanceId"] != "123~region(jp)" || inviteBody["worldId"] != nil {
		t.Fatalf("invite body = %#v", inviteBody)
	}
	if boopBody["emojiId"] != "default_hand_wave" {
		t.Fatalf("boop body = %#v", boopBody)
	}
}
