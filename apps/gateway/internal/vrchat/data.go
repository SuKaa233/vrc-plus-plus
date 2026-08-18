package vrchat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
)

const (
	friendsCacheKey = "friends:v1"
	friendsCacheTTL = 2 * time.Minute
	userCacheTTL    = 5 * time.Minute
	mutualsCacheTTL = 6 * time.Hour
	worldsCacheTTL  = 10 * time.Minute
	groupsCacheTTL  = 15 * time.Minute
	avatarsCacheTTL = 10 * time.Minute
)

type friendPayload struct {
	ID                             string   `json:"id"`
	DisplayName                    string   `json:"displayName"`
	Status                         string   `json:"status"`
	StatusDescription              string   `json:"statusDescription"`
	Location                       string   `json:"location"`
	Platform                       string   `json:"platform"`
	LastPlatform                   string   `json:"last_platform"`
	UserIcon                       string   `json:"userIcon"`
	ImageURL                       string   `json:"imageUrl"`
	CurrentAvatarThumbnailImageURL string   `json:"currentAvatarThumbnailImageUrl"`
	CurrentAvatarImageURL          string   `json:"currentAvatarImageUrl"`
	ProfilePicOverride             string   `json:"profilePicOverride"`
	ProfilePicOverrideThumbnail    string   `json:"profilePicOverrideThumbnail"`
	Bio                            string   `json:"bio"`
	BioLinks                       []string `json:"bioLinks"`
	CurrentAvatarTags              []string `json:"currentAvatarTags"`
	LastActivity                   string   `json:"last_activity"`
	LastLogin                      string   `json:"last_login"`
	LastMobile                     string   `json:"last_mobile"`
	DeveloperType                  string   `json:"developerType"`
	IsFriend                       bool     `json:"isFriend"`
}

type worldPayload struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	AuthorName          string    `json:"authorName"`
	ThumbnailImageURL   string    `json:"thumbnailImageUrl"`
	ImageURL            string    `json:"imageUrl"`
	Capacity            int       `json:"capacity"`
	RecommendedCapacity int       `json:"recommendedCapacity"`
	Occupants           int       `json:"occupants"`
	Favorites           int       `json:"favorites"`
	Visits              int       `json:"visits"`
	ReleaseStatus       string    `json:"releaseStatus"`
	UpdatedAt           time.Time `json:"updated_at"`
	Tags                []string  `json:"tags"`
	Instances           [][]any   `json:"instances"`
}

type favoritePayload struct {
	ID         string   `json:"id"`
	FavoriteID string   `json:"favoriteId"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags"`
}

type favoriteGroupPayload struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	Visibility  string `json:"visibility"`
}

type groupPayload struct {
	GroupID           string    `json:"groupId"`
	Name              string    `json:"name"`
	ShortCode         string    `json:"shortCode"`
	Description       string    `json:"description"`
	IconURL           string    `json:"iconUrl"`
	BannerURL         string    `json:"bannerUrl"`
	OwnerID           string    `json:"ownerId"`
	MemberCount       int       `json:"memberCount"`
	Privacy           string    `json:"privacy"`
	MemberVisibility  string    `json:"memberVisibility"`
	IsRepresenting    bool      `json:"isRepresenting"`
	LastPostCreatedAt time.Time `json:"lastPostCreatedAt"`
	LastPostReadAt    time.Time `json:"lastPostReadAt"`
}

func (item groupPayload) toModel() model.Group {
	return model.Group{ID: item.GroupID, Name: item.Name, ShortCode: item.ShortCode, Description: item.Description,
		IconURL: item.IconURL, BannerURL: item.BannerURL, OwnerID: item.OwnerID, MemberCount: item.MemberCount,
		Privacy: item.Privacy, MemberVisibility: item.MemberVisibility, IsRepresenting: item.IsRepresenting,
		LastPostCreatedAt: item.LastPostCreatedAt, LastPostReadAt: item.LastPostReadAt}
}

type groupPostPayload struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"groupId"`
	AuthorID   string    `json:"authorId"`
	Title      string    `json:"title"`
	Text       string    `json:"text"`
	ImageURL   string    `json:"imageUrl"`
	Visibility string    `json:"visibility"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type groupInstancePayload struct {
	InstanceID  string       `json:"instanceId"`
	Location    string       `json:"location"`
	MemberCount int          `json:"memberCount"`
	World       worldPayload `json:"world"`
}

type calendarEventPayload struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	Category            string    `json:"category"`
	ImageURL            string    `json:"imageUrl"`
	StartsAt            time.Time `json:"startsAt"`
	EndsAt              time.Time `json:"endsAt"`
	DurationInMS        int64     `json:"durationInMs"`
	InterestedUserCount int       `json:"interestedUserCount"`
	Languages           []string  `json:"languages"`
	Platforms           []string  `json:"platforms"`
	AccessType          string    `json:"accessType"`
	OccurrenceKind      string    `json:"occurrenceKind"`
	UserInterest        *struct {
		IsFollowing bool `json:"isFollowing"`
	} `json:"userInterest"`
}

type avatarPayload struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	AuthorID          string         `json:"authorId"`
	AuthorName        string         `json:"authorName"`
	ImageURL          string         `json:"imageUrl"`
	ThumbnailImageURL string         `json:"thumbnailImageUrl"`
	ReleaseStatus     string         `json:"releaseStatus"`
	Tags              []string       `json:"tags"`
	Performance       map[string]any `json:"performance"`
	UnityPackages     []struct {
		Platform string `json:"platform"`
	} `json:"unityPackages"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (item avatarPayload) toModel() model.Avatar {
	platformSet := map[string]bool{}
	platforms := make([]string, 0, len(item.UnityPackages))
	for _, pkg := range item.UnityPackages {
		if pkg.Platform != "" && !platformSet[pkg.Platform] {
			platformSet[pkg.Platform] = true
			platforms = append(platforms, pkg.Platform)
		}
	}
	performance := make(map[string]string)
	for key, value := range item.Performance {
		if text, ok := value.(string); ok && text != "" {
			performance[key] = text
		}
	}
	return model.Avatar{ID: item.ID, Name: item.Name, Description: item.Description, AuthorID: item.AuthorID,
		AuthorName: item.AuthorName, ImageURL: item.ImageURL, ThumbnailImageURL: item.ThumbnailImageURL,
		ReleaseStatus: item.ReleaseStatus, Tags: item.Tags, Performance: performance, Platforms: platforms, UpdatedAt: item.UpdatedAt}
}

type userPayload struct {
	ID                             string                  `json:"id"`
	DisplayName                    string                  `json:"displayName"`
	Bio                            string                  `json:"bio"`
	BioLinks                       []string                `json:"bioLinks"`
	Pronouns                       string                  `json:"pronouns"`
	Status                         string                  `json:"status"`
	StatusDescription              string                  `json:"statusDescription"`
	Location                       string                  `json:"location"`
	Platform                       string                  `json:"platform"`
	LastPlatform                   string                  `json:"last_platform"`
	State                          string                  `json:"state"`
	DeveloperType                  string                  `json:"developerType"`
	DateJoined                     string                  `json:"date_joined"`
	LastActivity                   string                  `json:"last_activity"`
	LastLogin                      string                  `json:"last_login"`
	UserIcon                       string                  `json:"userIcon"`
	ImageURL                       string                  `json:"imageUrl"`
	CurrentAvatarImageURL          string                  `json:"currentAvatarImageUrl"`
	CurrentAvatarThumbnailImageURL string                  `json:"currentAvatarThumbnailImageUrl"`
	ProfilePicOverride             string                  `json:"profilePicOverride"`
	ProfilePicOverrideThumbnail    string                  `json:"profilePicOverrideThumbnail"`
	BannerURL                      string                  `json:"bannerUrl"`
	IsFriend                       bool                    `json:"isFriend"`
	AllowAvatarCopying             bool                    `json:"allowAvatarCopying"`
	Tags                           []string                `json:"tags"`
	TrustTags                      []string                `json:"trustTags"`
	Note                           string                  `json:"note"`
	LastMobile                     string                  `json:"last_mobile"`
	InstanceID                     string                  `json:"instanceId"`
	WorldID                        string                  `json:"worldId"`
	TravelingToInstance            string                  `json:"travelingToInstance"`
	TravelingToLocation            string                  `json:"travelingToLocation"`
	TravelingToWorld               string                  `json:"travelingToWorld"`
	Languages                      []string                `json:"languages"`
	Badges                         []model.UserBadge       `json:"badges"`
	RepresentedGroup               *model.RepresentedGroup `json:"representedGroup"`
	HasVRCPlus                     bool                    `json:"hasVrcPlus"`
	IsEconomyCreator               bool                    `json:"isEconomyCreator"`
	AgeVerificationStatus          string                  `json:"ageVerificationStatus"`
	AgeVerified                    bool                    `json:"ageVerified"`
	IconURL                        string                  `json:"iconUrl"`
	IconFrame                      string                  `json:"iconFrame"`
	BannerColor                    string                  `json:"bannerColor"`
	BannerType                     string                  `json:"bannerType"`
	BackgroundType                 string                  `json:"backgroundType"`
	NameplateEffect                string                  `json:"nameplateEffect"`
	ProfileEffect                  string                  `json:"profileEffect"`
	ThemeID                        string                  `json:"themeId"`
}

type privateProfilePayload struct {
	IsFriend          bool   `json:"isFriend"`
	Note              string `json:"note"`
	Status            string `json:"status"`
	StatusDescription string `json:"statusDescription"`
	Activity          struct {
		InstanceID          string `json:"instanceId"`
		WorldID             string `json:"worldId"`
		Location            string `json:"location"`
		Platform            string `json:"platform"`
		State               string `json:"state"`
		LastActivity        string `json:"last_activity"`
		LastLogin           string `json:"last_login"`
		TravelingToInstance string `json:"travelingToInstance"`
		TravelingToLocation string `json:"travelingToLocation"`
		TravelingToWorld    string `json:"travelingToWorld"`
	} `json:"activity"`
}

type mutualCountsPayload struct {
	Friends int `json:"friends"`
	Groups  int `json:"groups"`
}

type mutualFriendPayload struct {
	ID                             string `json:"id"`
	DisplayName                    string `json:"displayName"`
	Status                         string `json:"status"`
	StatusDescription              string `json:"statusDescription"`
	ImageURL                       string `json:"imageUrl"`
	ProfilePicOverride             string `json:"profilePicOverride"`
	CurrentAvatarThumbnailImageURL string `json:"currentAvatarThumbnailImageUrl"`
}

func (c *Client) ListFriends(ctx context.Context) (model.DataEnvelope[model.Friend], error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	var all []model.Friend
	for _, offline := range []bool{false, true} {
		for offset := 0; offset < 1000; offset += 100 {
			query := url.Values{}
			query.Set("n", "100")
			query.Set("offset", strconv.Itoa(offset))
			query.Set("offline", strconv.FormatBool(offline))
			data, err := c.getJSON(ctx, "auth/user/friends?"+query.Encode())
			if err != nil {
				return c.cachedFriends(ctx, err)
			}
			var payload []friendPayload
			if err := json.Unmarshal(data, &payload); err != nil {
				return c.cachedFriends(ctx, fmt.Errorf("parse VRChat friends: %w", err))
			}
			for _, item := range payload {
				all = append(all, item.toModel(!offline))
			}
			if len(payload) < 100 {
				break
			}
		}
	}

	byID := make(map[string]model.Friend, len(all))
	for _, friend := range all {
		if current, exists := byID[friend.ID]; !exists || friend.Online || !current.Online {
			byID[friend.ID] = friend
		}
	}
	items := make([]model.Friend, 0, len(byID))
	for _, friend := range byID {
		items = append(items, friend)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Online != items[j].Online {
			return items[i].Online
		}
		return strings.ToLower(items[i].DisplayName) < strings.ToLower(items[j].DisplayName)
	})
	return saveEnvelope(ctx, c.store, friendsCacheKey, items, friendsCacheTTL)
}

func (c *Client) GetUser(ctx context.Context, userID string) (model.DataEnvelope[model.UserProfile], error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	if err := validateUserID(userID); err != nil {
		return model.DataEnvelope[model.UserProfile]{}, err
	}
	cacheKey := "user:" + userID
	if cached, cacheErr := loadFreshCache[model.UserProfile](ctx, c.store, cacheKey); cacheErr == nil {
		return cached, nil
	}
	data, err := c.getJSON(ctx, "users/"+url.PathEscape(userID))
	if err != nil {
		return loadCache[model.UserProfile](ctx, c.store, cacheKey, err)
	}
	var payload userPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return loadCache[model.UserProfile](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat user: %w", err))
	}
	profile := payload.toModel()
	profile.ProfileSources = []string{"user"}
	if publicData, publicErr := c.getJSON(ctx, "profile/"+url.PathEscape(userID)); publicErr == nil {
		var public userPayload
		if json.Unmarshal(publicData, &public) == nil {
			mergePublicProfile(&profile, public)
			profile.ProfileSources = append(profile.ProfileSources, "public-profile")
		}
	}
	if privateData, privateErr := c.getJSON(ctx, "profile/"+url.PathEscape(userID)+"/private"); privateErr == nil {
		var private privateProfilePayload
		if json.Unmarshal(privateData, &private) == nil {
			mergePrivateProfile(&profile, private)
			profile.ProfileSources = append(profile.ProfileSources, "private-profile")
			profile.ActivityVisibility = "visible"
		}
	} else {
		profile.ActivityVisibility = "restricted"
	}
	if mutualData, mutualErr := c.getJSON(ctx, "users/"+url.PathEscape(userID)+"/mutuals"); mutualErr == nil {
		var mutual mutualCountsPayload
		if json.Unmarshal(mutualData, &mutual) == nil {
			profile.MutualFriendCount, profile.MutualGroupCount = mutual.Friends, mutual.Groups
			profile.ProfileSources = append(profile.ProfileSources, "mutuals")
		}
	}
	return saveEnvelope(ctx, c.store, cacheKey, []model.UserProfile{profile}, userCacheTTL)
}

func (c *Client) UpdateSelfProfile(ctx context.Context, input model.SelfProfileUpdate) (model.UserProfile, error) {
	input.Status = strings.TrimSpace(input.Status)
	input.StatusDescription = strings.TrimSpace(input.StatusDescription)
	input.Pronouns = strings.TrimSpace(input.Pronouns)
	input.Bio = strings.TrimSpace(input.Bio)
	validStatuses := map[string]bool{"join me": true, "active": true, "ask me": true, "busy": true, "offline": true}
	if !validStatuses[input.Status] {
		return model.UserProfile{}, fmt.Errorf("%w: invalid status", ErrInvalidRequest)
	}
	if len([]rune(input.StatusDescription)) > 32 || len([]rune(input.Pronouns)) > 32 || len([]rune(input.Bio)) > 512 || len(input.BioLinks) > 3 {
		return model.UserProfile{}, fmt.Errorf("%w: profile fields exceed VRChat limits", ErrInvalidRequest)
	}
	links := make([]string, 0, len(input.BioLinks))
	for _, raw := range input.BioLinks {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || len([]rune(value)) > 1000 {
			return model.UserProfile{}, fmt.Errorf("%w: bio links must be valid http or https URLs", ErrInvalidRequest)
		}
		links = append(links, value)
	}
	input.BioLinks = links

	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	userID := c.currentUserID
	if err := validateUserID(userID); err != nil {
		return model.UserProfile{}, errors.New("VRChat login session is unavailable")
	}
	userBody, _ := json.Marshal(map[string]any{
		"status":             input.Status,
		"statusDescription":  input.StatusDescription,
		"pronouns":           input.Pronouns,
		"allowAvatarCopying": input.AllowAvatarCopying,
	})
	response, err := c.do(ctx, http.MethodPut, "users/"+url.PathEscape(userID), bytes.NewReader(userBody), "")
	if response != nil {
		response.Body.Close()
	}
	if err != nil {
		return model.UserProfile{}, err
	}
	profileBody, _ := json.Marshal(map[string]any{"bio": input.Bio, "bioLinks": input.BioLinks})
	response, err = c.do(ctx, http.MethodPut, "profile/"+url.PathEscape(userID), bytes.NewReader(profileBody), "")
	if response != nil {
		response.Body.Close()
	}
	if err != nil {
		return model.UserProfile{}, fmt.Errorf("basic profile saved, but bio update failed: %w", err)
	}
	data, err := c.getJSON(ctx, "users/"+url.PathEscape(userID))
	if err != nil {
		return model.UserProfile{}, fmt.Errorf("profile saved, but refresh failed: %w", err)
	}
	var payload userPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return model.UserProfile{}, fmt.Errorf("parse updated VRChat profile: %w", err)
	}
	return payload.toModel(), nil
}

func (c *Client) SearchUsers(ctx context.Context, search string, limit int) (model.DataEnvelope[model.UserProfile], error) {
	search = strings.TrimSpace(search)
	if strings.HasPrefix(search, "usr_") {
		if err := validateUserID(search); err != nil {
			return model.DataEnvelope[model.UserProfile]{}, err
		}
		return c.GetUser(ctx, search)
	}
	if len([]rune(search)) < 2 || len([]rune(search)) > 64 {
		return model.DataEnvelope[model.UserProfile]{}, fmt.Errorf("%w: user search must contain 2-64 characters", ErrInvalidRequest)
	}
	if limit < 1 || limit > 20 {
		limit = 12
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	query := url.Values{"search": {search}, "n": {strconv.Itoa(limit)}, "offset": {"0"}}
	data, err := c.getJSON(ctx, "users?"+query.Encode())
	if err != nil {
		return model.DataEnvelope[model.UserProfile]{}, err
	}
	var payload []userPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return model.DataEnvelope[model.UserProfile]{}, fmt.Errorf("parse VRChat user search: %w", err)
	}
	items := make([]model.UserProfile, 0, len(payload))
	for _, item := range payload {
		if item.ID != "" {
			items = append(items, item.toModel())
		}
	}
	return model.DataEnvelope[model.UserProfile]{Items: items, Source: "live", FetchedAt: time.Now().UTC()}, nil
}

func (c *Client) FriendStatus(ctx context.Context, userID string) (model.FriendStatus, error) {
	if err := validateUserID(userID); err != nil {
		return model.FriendStatus{}, err
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	data, err := c.getJSON(ctx, "user/"+url.PathEscape(userID)+"/friendStatus")
	if err != nil {
		return model.FriendStatus{}, err
	}
	var status model.FriendStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return model.FriendStatus{}, fmt.Errorf("parse VRChat friend status: %w", err)
	}
	return status, nil
}

func (c *Client) SendFriendRequest(ctx context.Context, userID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	response, err := c.do(ctx, http.MethodPost, "user/"+url.PathEscape(userID)+"/friendRequest", nil, "")
	if response != nil {
		response.Body.Close()
	}
	return err
}

func (c *Client) ListMutualFriends(ctx context.Context, userID string, refresh bool) (model.DataEnvelope[model.MutualFriend], error) {
	if err := validateUserID(userID); err != nil {
		return model.DataEnvelope[model.MutualFriend]{}, err
	}
	cacheKey := "mutual-friends:" + userID
	if !refresh {
		if cached, err := loadFreshCache[model.MutualFriend](ctx, c.store, cacheKey); err == nil {
			return cached, nil
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	items := make([]model.MutualFriend, 0, 100)
	for offset := 0; offset < 1000; offset += 100 {
		query := url.Values{"n": {"100"}, "offset": {strconv.Itoa(offset)}}
		data, err := c.getJSON(ctx, "users/"+url.PathEscape(userID)+"/mutuals/friends?"+query.Encode())
		if err != nil {
			var upstream *UpstreamError
			if errors.As(err, &upstream) && (upstream.StatusCode == http.StatusForbidden || upstream.StatusCode == http.StatusNotFound) {
				if saveErr := c.store.SaveMutualGraph(ctx, userID, nil, true); saveErr != nil {
					return model.DataEnvelope[model.MutualFriend]{}, saveErr
				}
				result, saveErr := saveEnvelope(ctx, c.store, cacheKey, []model.MutualFriend{}, mutualsCacheTTL)
				result.OptedOut = true
				result.Message = "对方未共享共同好友，或该接口当前不可用"
				return result, saveErr
			}
			return loadCache[model.MutualFriend](ctx, c.store, cacheKey, err)
		}
		var payload []mutualFriendPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return loadCache[model.MutualFriend](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat mutual friends: %w", err))
		}
		for _, item := range payload {
			items = append(items, item.toModel())
		}
		if len(payload) < 100 {
			break
		}
	}
	byID := make(map[string]model.MutualFriend, len(items))
	for _, item := range items {
		if item.ID != "" {
			byID[item.ID] = item
		}
	}
	items = items[:0]
	for _, item := range byID {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].DisplayName) < strings.ToLower(items[j].DisplayName)
	})
	mutualIDs := make([]string, 0, len(items))
	for _, item := range items {
		mutualIDs = append(mutualIDs, item.ID)
	}
	if err := c.store.SaveMutualGraph(ctx, userID, mutualIDs, false); err != nil {
		return model.DataEnvelope[model.MutualFriend]{}, err
	}
	return saveEnvelope(ctx, c.store, cacheKey, items, mutualsCacheTTL)
}

func (c *Client) FriendNetwork(ctx context.Context) (model.FriendNetwork, error) {
	entry, err := c.store.LoadCache(ctx, friendsCacheKey)
	if errors.Is(err, storage.ErrNotFound) {
		return model.FriendNetwork{Nodes: []model.FriendNetworkNode{}, Edges: []model.FriendNetworkEdge{}, GeneratedAt: time.Now().UTC()}, nil
	}
	if err != nil {
		return model.FriendNetwork{}, err
	}
	var friends []model.Friend
	if err := json.Unmarshal(entry.Payload, &friends); err != nil {
		return model.FriendNetwork{}, fmt.Errorf("parse local friend snapshot: %w", err)
	}
	meta, storedEdges, err := c.store.LoadMutualGraph(ctx)
	if err != nil {
		return model.FriendNetwork{}, err
	}
	friendByID := make(map[string]model.Friend, len(friends))
	for _, friend := range friends {
		friendByID[friend.ID] = friend
	}
	metaByID := make(map[string]storage.MutualGraphMeta, len(meta))
	for _, item := range meta {
		metaByID[item.FriendID] = item
	}
	nodes := make([]model.FriendNetworkNode, 0, len(friends))
	scannedCount, optedOutCount := 0, 0
	for _, friend := range friends {
		node := model.FriendNetworkNode{
			ID: friend.ID, DisplayName: friend.DisplayName, Online: friend.Online,
			UserIcon: friend.UserIcon, ImageURL: friend.ImageURL,
			CurrentAvatarThumbnailImageURL: friend.CurrentAvatarThumbnailImageURL,
		}
		if item, ok := metaByID[friend.ID]; ok {
			node.Scanned = true
			node.OptedOut = item.OptedOut
			node.ScannedAt = &item.FetchedAt
			scannedCount++
			if item.OptedOut {
				optedOutCount++
			}
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Online != nodes[j].Online {
			return nodes[i].Online
		}
		return strings.ToLower(nodes[i].DisplayName) < strings.ToLower(nodes[j].DisplayName)
	})
	edgeSet := make(map[string]model.FriendNetworkEdge)
	for _, edge := range storedEdges {
		if edge.FriendID == edge.MutualID {
			continue
		}
		if _, ok := friendByID[edge.FriendID]; !ok {
			continue
		}
		if _, ok := friendByID[edge.MutualID]; !ok {
			continue
		}
		source, target := edge.FriendID, edge.MutualID
		if source > target {
			source, target = target, source
		}
		edgeSet[source+"|"+target] = model.FriendNetworkEdge{Source: source, Target: target}
	}
	edges := make([]model.FriendNetworkEdge, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})
	return model.FriendNetwork{
		Nodes: nodes, Edges: edges, TotalFriends: len(nodes), ScannedCount: scannedCount,
		OptedOutCount: optedOutCount, GeneratedAt: time.Now().UTC(),
	}, nil
}

func (c *Client) SearchWorlds(ctx context.Context, search string, offset, limit int) (model.DataEnvelope[model.World], error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	search = strings.TrimSpace(search)
	if limit < 1 || limit > 100 {
		return model.DataEnvelope[model.World]{}, fmt.Errorf("%w: world result limit must be between 1 and 100", ErrInvalidRequest)
	}
	if offset < 0 || offset > 10000 {
		return model.DataEnvelope[model.World]{}, fmt.Errorf("%w: world result offset is invalid", ErrInvalidRequest)
	}
	query := url.Values{}
	query.Set("n", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	query.Set("sort", "popularity")
	query.Set("order", "descending")
	if search != "" {
		query.Set("search", search)
	}
	cacheKey := cacheKey("world-search", query.Encode())
	data, err := c.getJSON(ctx, "worlds?"+query.Encode())
	if err != nil {
		return c.cachedWorlds(ctx, cacheKey, err)
	}
	var payload []worldPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return c.cachedWorlds(ctx, cacheKey, fmt.Errorf("parse VRChat worlds: %w", err))
	}
	items := make([]model.World, 0, len(payload))
	for _, item := range payload {
		items = append(items, item.toModel())
	}
	return saveEnvelope(ctx, c.store, cacheKey, items, worldsCacheTTL)
}

func (c *Client) GetWorld(ctx context.Context, worldID string) (model.DataEnvelope[model.World], error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	worldID = strings.TrimSpace(worldID)
	if !strings.HasPrefix(worldID, "wrld_") || len(worldID) > 80 {
		return model.DataEnvelope[model.World]{}, fmt.Errorf("%w: invalid VRChat world id", ErrInvalidRequest)
	}
	cacheKey := "world:" + worldID
	data, err := c.getJSON(ctx, "worlds/"+url.PathEscape(worldID))
	if err != nil {
		return c.cachedWorlds(ctx, cacheKey, err)
	}
	var payload worldPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return c.cachedWorlds(ctx, cacheKey, fmt.Errorf("parse VRChat world: %w", err))
	}
	return saveEnvelope(ctx, c.store, cacheKey, []model.World{payload.toModel()}, worldsCacheTTL)
}

type instancePayload struct {
	ID              string   `json:"id"`
	WorldID         string   `json:"worldId"`
	InstanceID      string   `json:"instanceId"`
	Name            string   `json:"displayName"`
	Type            string   `json:"type"`
	Region          string   `json:"region"`
	OwnerID         string   `json:"ownerId"`
	GroupAccessType string   `json:"groupAccessType"`
	UserCount       int      `json:"userCount"`
	Capacity        int      `json:"capacity"`
	QueueSize       int      `json:"queueSize"`
	QueueEnabled    bool     `json:"queueEnabled"`
	Active          bool     `json:"active"`
	Full            bool     `json:"full"`
	Tags            []string `json:"tags"`
}

func (c *Client) GetInstance(ctx context.Context, location string) (model.Instance, error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	location = strings.TrimSpace(location)
	parts := strings.SplitN(location, ":", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "wrld_") || parts[1] == "" || len(location) > 500 || strings.ContainsAny(location, "/?#") {
		return model.Instance{}, fmt.Errorf("%w: invalid VRChat instance location", ErrInvalidRequest)
	}
	data, err := c.getJSON(ctx, "instances/"+location)
	if err != nil {
		return model.Instance{}, err
	}
	var payload instancePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return model.Instance{}, fmt.Errorf("parse VRChat instance: %w", err)
	}
	if payload.WorldID == "" {
		payload.WorldID = parts[0]
	}
	if payload.InstanceID == "" {
		payload.InstanceID = parts[1]
	}
	return model.Instance{
		ID: location, WorldID: payload.WorldID, InstanceID: payload.InstanceID, Name: payload.Name,
		Type: payload.Type, Region: payload.Region, OwnerID: payload.OwnerID, GroupAccessType: payload.GroupAccessType,
		UserCount: payload.UserCount, Capacity: payload.Capacity, QueueSize: payload.QueueSize,
		QueueEnabled: payload.QueueEnabled, Active: payload.Active, Full: payload.Full, Tags: payload.Tags,
	}, nil
}

type notificationPayload struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	SenderUserID   string          `json:"senderUserId"`
	SenderUsername string          `json:"senderUsername"`
	Message        string          `json:"message"`
	Details        json.RawMessage `json:"details"`
	Seen           bool            `json:"seen"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (c *Client) ListNotifications(ctx context.Context) ([]model.Notification, error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	query := url.Values{"n": {"50"}, "offset": {"0"}}
	data, err := c.getJSON(ctx, "auth/user/notifications?"+query.Encode())
	if err != nil {
		return nil, err
	}
	var payload []notificationPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse VRChat notifications: %w", err)
	}
	items := make([]model.Notification, 0, len(payload))
	for _, raw := range payload {
		item := model.Notification{ID: raw.ID, Type: raw.Type, SenderUserID: raw.SenderUserID, SenderUsername: raw.SenderUsername, Message: raw.Message, Seen: raw.Seen, CreatedAt: raw.CreatedAt}
		var details map[string]any
		detailsJSON := raw.Details
		var encodedDetails string
		if json.Unmarshal(raw.Details, &encodedDetails) == nil && json.Valid([]byte(encodedDetails)) {
			detailsJSON = []byte(encodedDetails)
		}
		if json.Unmarshal(detailsJSON, &details) == nil {
			if value, ok := details["worldId"].(string); ok {
				item.WorldID = value
			}
			if value, ok := details["worldName"].(string); ok {
				item.WorldName = value
			}
			if value, ok := details["instanceId"].(string); ok {
				item.InstanceID = value
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *Client) ListUserGroups(ctx context.Context, userID string, refresh bool) (model.DataEnvelope[model.Group], error) {
	if err := validateUserID(userID); err != nil {
		return model.DataEnvelope[model.Group]{}, err
	}
	cacheKey := "groups:" + userID
	if !refresh {
		if cached, err := loadFreshCache[model.Group](ctx, c.store, cacheKey); err == nil {
			return cached, nil
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	data, err := c.getJSON(ctx, "users/"+url.PathEscape(userID)+"/groups")
	if err != nil {
		return loadCache[model.Group](ctx, c.store, cacheKey, err)
	}
	var payload []groupPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return loadCache[model.Group](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat groups: %w", err))
	}
	items := make([]model.Group, 0, len(payload))
	for _, item := range payload {
		if strings.HasPrefix(item.GroupID, "grp_") && item.Name != "" {
			items = append(items, item.toModel())
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsRepresenting != items[j].IsRepresenting {
			return items[i].IsRepresenting
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return saveEnvelope(ctx, c.store, cacheKey, items, groupsCacheTTL)
}

func (c *Client) ListGroupPosts(ctx context.Context, groupID string, refresh bool) (model.DataEnvelope[model.GroupPost], error) {
	if !validGroupID(groupID) {
		return model.DataEnvelope[model.GroupPost]{}, fmt.Errorf("%w: invalid group id", ErrInvalidRequest)
	}
	cacheKey := "group-posts:" + groupID
	if !refresh {
		if cached, err := loadFreshCache[model.GroupPost](ctx, c.store, cacheKey); err == nil {
			return cached, nil
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	data, err := c.getJSON(ctx, "groups/"+url.PathEscape(groupID)+"/posts?n=50&offset=0")
	if err != nil {
		return loadCache[model.GroupPost](ctx, c.store, cacheKey, err)
	}
	var payload struct {
		Posts []groupPostPayload `json:"posts"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return loadCache[model.GroupPost](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat group posts: %w", err))
	}
	items := make([]model.GroupPost, 0, len(payload.Posts))
	for _, item := range payload.Posts {
		items = append(items, model.GroupPost{ID: item.ID, GroupID: item.GroupID, AuthorID: item.AuthorID, Title: item.Title, Text: item.Text, ImageURL: item.ImageURL, Visibility: item.Visibility, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return saveEnvelope(ctx, c.store, cacheKey, items, groupsCacheTTL)
}

func (c *Client) ListGroupInstances(ctx context.Context, groupID string, refresh bool) (model.DataEnvelope[model.GroupInstance], error) {
	if !validGroupID(groupID) {
		return model.DataEnvelope[model.GroupInstance]{}, fmt.Errorf("%w: invalid group id", ErrInvalidRequest)
	}
	cacheKey := "group-instances:" + groupID
	if !refresh {
		if cached, err := loadFreshCache[model.GroupInstance](ctx, c.store, cacheKey); err == nil {
			return cached, nil
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	data, err := c.getJSON(ctx, "groups/"+url.PathEscape(groupID)+"/instances")
	if err != nil {
		return loadCache[model.GroupInstance](ctx, c.store, cacheKey, err)
	}
	var payload []groupInstancePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return loadCache[model.GroupInstance](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat group instances: %w", err))
	}
	items := make([]model.GroupInstance, 0, len(payload))
	for _, item := range payload {
		items = append(items, model.GroupInstance{InstanceID: item.InstanceID, Location: item.Location, MemberCount: item.MemberCount, World: item.World.toModel()})
	}
	return saveEnvelope(ctx, c.store, cacheKey, items, 2*time.Minute)
}

func (c *Client) ListGroupCalendarEvents(ctx context.Context, groupID, month string, refresh bool) (model.DataEnvelope[model.CalendarEvent], error) {
	if !validGroupID(groupID) {
		return model.DataEnvelope[model.CalendarEvent]{}, fmt.Errorf("%w: invalid group id", ErrInvalidRequest)
	}
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	monthStart, err := time.Parse("2006-01", month)
	if err != nil {
		return model.DataEnvelope[model.CalendarEvent]{}, fmt.Errorf("%w: invalid calendar month", ErrInvalidRequest)
	}
	cacheKey := "group-calendar:" + groupID + ":" + month
	if !refresh {
		if cached, cacheErr := loadFreshCache[model.CalendarEvent](ctx, c.store, cacheKey); cacheErr == nil {
			return cached, nil
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	query := url.Values{"date": {monthStart.UTC().Format(time.RFC3339)}, "n": {"100"}, "offset": {"0"}}
	data, err := c.getJSON(ctx, "calendar/"+url.PathEscape(groupID)+"?"+query.Encode())
	if err != nil {
		return loadCache[model.CalendarEvent](ctx, c.store, cacheKey, err)
	}
	var payload struct {
		Results []calendarEventPayload `json:"results"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return loadCache[model.CalendarEvent](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat group calendar: %w", err))
	}
	items := make([]model.CalendarEvent, 0, len(payload.Results))
	for _, item := range payload.Results {
		following := item.UserInterest != nil && item.UserInterest.IsFollowing
		items = append(items, model.CalendarEvent{ID: item.ID, GroupID: groupID, Title: item.Title, Description: item.Description, Category: item.Category, ImageURL: item.ImageURL, StartsAt: item.StartsAt, EndsAt: item.EndsAt, DurationInMS: item.DurationInMS, InterestedUserCount: item.InterestedUserCount, Languages: item.Languages, Platforms: item.Platforms, AccessType: item.AccessType, OccurrenceKind: item.OccurrenceKind, Following: following})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartsAt.Before(items[j].StartsAt) })
	return saveEnvelope(ctx, c.store, cacheKey, items, 30*time.Minute)
}

func (c *Client) ListFavoriteAvatars(ctx context.Context, refresh bool) (model.DataEnvelope[model.Avatar], error) {
	const cacheKey = "favorite-avatars:v1"
	if !refresh {
		if cached, err := loadFreshCache[model.Avatar](ctx, c.store, cacheKey); err == nil {
			return cached, nil
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	items := make([]model.Avatar, 0, 100)
	for offset := 0; offset < 1000; offset += 100 {
		data, err := c.getJSON(ctx, "avatars/favorites?n=100&offset="+strconv.Itoa(offset)+"&sort=updated&order=descending&releaseStatus=all")
		if err != nil {
			return loadCache[model.Avatar](ctx, c.store, cacheKey, err)
		}
		var payload []avatarPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return loadCache[model.Avatar](ctx, c.store, cacheKey, fmt.Errorf("parse VRChat favorite avatars: %w", err))
		}
		for _, item := range payload {
			if strings.HasPrefix(item.ID, "avtr_") {
				items = append(items, item.toModel())
			}
		}
		if len(payload) < 100 {
			break
		}
	}
	return saveEnvelope(ctx, c.store, cacheKey, items, avatarsCacheTTL)
}

func validGroupID(groupID string) bool {
	return strings.HasPrefix(groupID, "grp_") && len(groupID) <= 80 && !strings.ContainsAny(groupID, "/?#")
}

func (c *Client) ListFavoriteGroups(ctx context.Context) ([]model.FavoriteGroup, error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	query := url.Values{"n": {"100"}, "offset": {"0"}, "type": {"world"}}
	data, err := c.getJSON(ctx, "favorite/groups?"+query.Encode())
	if err != nil {
		return nil, err
	}
	var payload []favoriteGroupPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse VRChat favorite groups: %w", err)
	}
	items := make([]model.FavoriteGroup, 0, len(payload))
	for _, item := range payload {
		if item.Type == "world" || item.Type == "" {
			items = append(items, model.FavoriteGroup{Name: item.Name, DisplayName: item.DisplayName, Type: "world", Visibility: item.Visibility})
		}
	}
	return items, nil
}

func (c *Client) ListUpstreamWorldFavorites(ctx context.Context) ([]model.UpstreamWorldFavorite, error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	worldData, err := c.getJSON(ctx, "worlds/favorites?n=100&offset=0")
	if err != nil {
		return nil, err
	}
	var worlds []worldPayload
	if err := json.Unmarshal(worldData, &worlds); err != nil {
		return nil, fmt.Errorf("parse VRChat favorite worlds: %w", err)
	}
	favoriteData, err := c.getJSON(ctx, "favorites?n=100&offset=0&type=world")
	if err != nil {
		return nil, err
	}
	var favorites []favoritePayload
	if err := json.Unmarshal(favoriteData, &favorites); err != nil {
		return nil, fmt.Errorf("parse VRChat favorites: %w", err)
	}
	byWorld := make(map[string]favoritePayload, len(favorites))
	for _, item := range favorites {
		byWorld[item.FavoriteID] = item
	}
	items := make([]model.UpstreamWorldFavorite, 0, len(worlds))
	for _, raw := range worlds {
		if !strings.HasPrefix(raw.ID, "wrld_") || raw.Name == "" {
			continue
		}
		favorite := byWorld[raw.ID]
		items = append(items, model.UpstreamWorldFavorite{ID: favorite.ID, Tags: favorite.Tags, World: raw.toModel()})
	}
	return items, nil
}

func (c *Client) AddUpstreamWorldFavorite(ctx context.Context, worldID, group string) (model.UpstreamWorldFavorite, error) {
	if !strings.HasPrefix(worldID, "wrld_") || strings.ContainsAny(worldID, "/?#") {
		return model.UpstreamWorldFavorite{}, fmt.Errorf("%w: invalid VRChat world id", ErrInvalidRequest)
	}
	group = strings.TrimSpace(group)
	if group == "" || len(group) > 64 {
		return model.UpstreamWorldFavorite{}, fmt.Errorf("%w: invalid favorite group", ErrInvalidRequest)
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	body, _ := json.Marshal(map[string]any{"type": "world", "favoriteId": worldID, "tags": []string{group}})
	response, err := c.do(ctx, http.MethodPost, "favorites", bytes.NewReader(body), "application/json")
	if err != nil {
		return model.UpstreamWorldFavorite{}, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return model.UpstreamWorldFavorite{}, err
	}
	var favorite favoritePayload
	if err := json.Unmarshal(data, &favorite); err != nil {
		return model.UpstreamWorldFavorite{}, fmt.Errorf("parse VRChat favorite: %w", err)
	}
	return model.UpstreamWorldFavorite{ID: favorite.ID, Tags: favorite.Tags, World: model.World{ID: worldID}}, nil
}

func (c *Client) DeleteUpstreamWorldFavorite(ctx context.Context, favoriteID string) error {
	if favoriteID == "" || len(favoriteID) > 100 || strings.ContainsAny(favoriteID, "/?#") {
		return fmt.Errorf("%w: invalid favorite id", ErrInvalidRequest)
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	response, err := c.do(ctx, http.MethodDelete, "favorites/"+url.PathEscape(favoriteID), nil, "")
	if response != nil {
		response.Body.Close()
	}
	return err
}

func (c *Client) SendInvite(ctx context.Context, receiverUserID, location string) error {
	if err := validateUserID(receiverUserID); err != nil {
		return err
	}
	parts := strings.SplitN(strings.TrimSpace(location), ":", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "wrld_") || parts[1] == "" || len(location) > 500 {
		return fmt.Errorf("%w: invalid instance location", ErrInvalidRequest)
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	body, _ := json.Marshal(map[string]string{"instanceId": parts[1]})
	response, err := c.do(ctx, http.MethodPost, "invite/"+url.PathEscape(receiverUserID), bytes.NewReader(body), "")
	if response != nil {
		response.Body.Close()
	}
	return err
}

func (c *Client) SendBoop(ctx context.Context, receiverUserID, emojiID string) error {
	if err := validateUserID(receiverUserID); err != nil {
		return err
	}
	emojiID = strings.TrimSpace(emojiID)
	if len(emojiID) < 3 || len(emojiID) > 96 ||
		(!strings.HasPrefix(emojiID, "default_") && !strings.HasPrefix(emojiID, "file_")) {
		return fmt.Errorf("%w: invalid boop emoji id", ErrInvalidRequest)
	}
	for _, char := range emojiID {
		if !(char == '_' || char == '-' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9') {
			return fmt.Errorf("%w: invalid boop emoji id", ErrInvalidRequest)
		}
	}
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	body, _ := json.Marshal(map[string]string{"emojiId": emojiID})
	response, err := c.do(ctx, http.MethodPost, "users/"+url.PathEscape(receiverUserID)+"/boop", bytes.NewReader(body), "")
	if response != nil {
		response.Body.Close()
	}
	return err
}

func (c *Client) NotificationAction(ctx context.Context, notificationID, action string) error {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	if notificationID == "" || len(notificationID) > 100 || strings.ContainsAny(notificationID, "/?#") {
		return fmt.Errorf("%w: invalid notification id", ErrInvalidRequest)
	}
	if action != "see" && action != "hide" && action != "accept" {
		return fmt.Errorf("%w: unsupported notification action", ErrInvalidRequest)
	}
	response, err := c.do(ctx, http.MethodPut, "auth/user/notifications/"+url.PathEscape(notificationID)+"/"+action, bytes.NewReader([]byte(`{}`)), "")
	if response != nil {
		response.Body.Close()
	}
	return err
}

func (c *Client) PipelineCredentials(ctx context.Context) (token, userAgent string, err error) {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	data, err := c.getJSON(ctx, "auth")
	if err != nil {
		return "", "", err
	}
	var payload struct {
		OK    bool   `json:"ok"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", fmt.Errorf("parse VRChat pipeline token: %w", err)
	}
	if !payload.OK || payload.Token == "" {
		return "", "", errors.New("VRChat pipeline token is unavailable")
	}
	return payload.Token, c.userAgent, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string) ([]byte, error) {
	result := c.requests.DoChan(endpoint, func() (any, error) {
		requestCtx, cancel := context.WithTimeout(ctx, 18*time.Second)
		defer cancel()
		response, err := c.do(requestCtx, http.MethodGet, endpoint, nil, "")
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
		if err != nil {
			return nil, fmt.Errorf("read VRChat response: %w", err)
		}
		return data, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case value := <-result:
		if value.Err != nil {
			return nil, value.Err
		}
		data, ok := value.Val.([]byte)
		if !ok {
			return nil, errors.New("unexpected shared VRChat response")
		}
		return data, nil
	}
}

func saveEnvelope[T any](ctx context.Context, store *storage.Store, key string, items []T, ttl time.Duration) (model.DataEnvelope[T], error) {
	fetchedAt := time.Now().UTC()
	payload, err := json.Marshal(items)
	if err != nil {
		return model.DataEnvelope[T]{}, err
	}
	if err := store.SaveCache(ctx, key, payload, ttl); err != nil {
		return model.DataEnvelope[T]{Items: items, Source: "live", FetchedAt: fetchedAt, Message: "live data loaded; local cache write failed"}, nil
	}
	return model.DataEnvelope[T]{Items: items, Source: "live", FetchedAt: fetchedAt}, nil
}

func (c *Client) cachedFriends(ctx context.Context, liveErr error) (model.DataEnvelope[model.Friend], error) {
	return loadCache[model.Friend](ctx, c.store, friendsCacheKey, liveErr)
}

func (c *Client) cachedWorlds(ctx context.Context, key string, liveErr error) (model.DataEnvelope[model.World], error) {
	return loadCache[model.World](ctx, c.store, key, liveErr)
}

func loadCache[T any](ctx context.Context, store *storage.Store, key string, liveErr error) (model.DataEnvelope[T], error) {
	entry, cacheErr := store.LoadCache(ctx, key)
	if cacheErr != nil {
		return model.DataEnvelope[T]{}, liveErr
	}
	var items []T
	if err := json.Unmarshal(entry.Payload, &items); err != nil {
		return model.DataEnvelope[T]{}, liveErr
	}
	return model.DataEnvelope[T]{
		Items: items, Source: "cache", FetchedAt: entry.FetchedAt, Stale: entry.Expired,
		Message: "VRChat live request failed; showing the last local snapshot",
	}, nil
}

func loadFreshCache[T any](ctx context.Context, store *storage.Store, key string) (model.DataEnvelope[T], error) {
	entry, err := store.LoadCache(ctx, key)
	if err != nil || entry.Expired {
		return model.DataEnvelope[T]{}, storage.ErrNotFound
	}
	var items []T
	if err := json.Unmarshal(entry.Payload, &items); err != nil {
		return model.DataEnvelope[T]{}, err
	}
	return model.DataEnvelope[T]{Items: items, Source: "cache", FetchedAt: entry.FetchedAt}, nil
}

func cacheKey(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return prefix + ":" + hex.EncodeToString(digest[:8])
}

func (payload friendPayload) toModel(online bool) model.Friend {
	return model.Friend{
		ID: payload.ID, DisplayName: payload.DisplayName, Status: payload.Status,
		StatusDescription: payload.StatusDescription, Location: payload.Location,
		Platform: payload.Platform, LastPlatform: payload.LastPlatform,
		UserIcon: payload.UserIcon, ImageURL: payload.ImageURL,
		CurrentAvatarThumbnailImageURL: payload.CurrentAvatarThumbnailImageURL,
		CurrentAvatarImageURL:          payload.CurrentAvatarImageURL, ProfilePicOverride: payload.ProfilePicOverride,
		ProfilePicOverrideThumbnail: payload.ProfilePicOverrideThumbnail, Bio: payload.Bio, BioLinks: payload.BioLinks,
		CurrentAvatarTags: payload.CurrentAvatarTags, LastActivity: payload.LastActivity, LastLogin: payload.LastLogin,
		LastMobile: payload.LastMobile, DeveloperType: payload.DeveloperType, IsFriend: payload.IsFriend,
		Online: online,
	}
}

func (payload worldPayload) toModel() model.World {
	world := model.World{
		ID: payload.ID, Name: payload.Name, Description: payload.Description,
		AuthorName: payload.AuthorName, ThumbnailImageURL: payload.ThumbnailImageURL,
		ImageURL: payload.ImageURL, Capacity: payload.Capacity,
		RecommendedCapacity: payload.RecommendedCapacity, Occupants: payload.Occupants,
		Favorites: payload.Favorites, Visits: payload.Visits,
		ReleaseStatus: payload.ReleaseStatus, UpdatedAt: payload.UpdatedAt, Tags: payload.Tags,
	}
	for _, row := range payload.Instances {
		if len(row) < 1 {
			continue
		}
		instanceID, ok := row[0].(string)
		if !ok || instanceID == "" {
			continue
		}
		count := 0
		if len(row) > 1 {
			if value, ok := row[1].(float64); ok {
				count = int(value)
			}
		}
		world.PublicInstances = append(world.PublicInstances, model.PublicInstance{InstanceID: instanceID, Location: payload.ID + ":" + instanceID, UserCount: count, Region: instanceRegion(instanceID), Type: instanceType(instanceID)})
	}
	sort.Slice(world.PublicInstances, func(i, j int) bool { return world.PublicInstances[i].UserCount > world.PublicInstances[j].UserCount })
	return world
}

func instanceRegion(instanceID string) string {
	for _, region := range []string{"usw", "use", "us", "eu", "jp"} {
		if strings.Contains(instanceID, "~region("+region+")") {
			return region
		}
	}
	return ""
}

func instanceType(instanceID string) string {
	switch {
	case strings.Contains(instanceID, "~group("):
		return "group"
	case strings.Contains(instanceID, "~friends+"):
		return "friends+"
	case strings.Contains(instanceID, "~friends("):
		return "friends"
	case strings.Contains(instanceID, "~private("):
		return "private"
	case strings.Contains(instanceID, "~hidden("):
		return "hidden"
	default:
		return "public"
	}
}

func (payload userPayload) toModel() model.UserProfile {
	return model.UserProfile{
		ID: payload.ID, DisplayName: payload.DisplayName, Bio: payload.Bio, BioLinks: payload.BioLinks,
		Pronouns: payload.Pronouns, Status: payload.Status, StatusDescription: payload.StatusDescription,
		Location: payload.Location, Platform: payload.Platform, LastPlatform: payload.LastPlatform,
		State: payload.State, DeveloperType: payload.DeveloperType, DateJoined: payload.DateJoined,
		LastActivity: payload.LastActivity, LastLogin: payload.LastLogin, UserIcon: payload.UserIcon,
		ImageURL: payload.ImageURL, CurrentAvatarImageURL: payload.CurrentAvatarImageURL,
		CurrentAvatarThumbnailImageURL: payload.CurrentAvatarThumbnailImageURL,
		ProfilePicOverride:             payload.ProfilePicOverride, ProfilePicOverrideThumbnail: payload.ProfilePicOverrideThumbnail,
		BannerURL: payload.BannerURL, IsFriend: payload.IsFriend, AllowAvatarCopying: payload.AllowAvatarCopying,
		Tags: payload.Tags, TrustLevel: trustLevel(payload.Tags), Note: payload.Note,
		LastMobile: payload.LastMobile, InstanceID: payload.InstanceID, WorldID: payload.WorldID,
		TravelingToInstance: payload.TravelingToInstance, TravelingToLocation: payload.TravelingToLocation, TravelingToWorld: payload.TravelingToWorld,
		Languages: payload.Languages, Badges: payload.Badges, RepresentedGroup: payload.RepresentedGroup,
		HasVRCPlus: payload.HasVRCPlus, IsEconomyCreator: payload.IsEconomyCreator,
		AgeVerificationStatus: payload.AgeVerificationStatus, AgeVerified: payload.AgeVerified,
		IconURL: payload.IconURL, IconFrame: payload.IconFrame, BannerColor: payload.BannerColor, BannerType: payload.BannerType,
		BackgroundType: payload.BackgroundType, NameplateEffect: payload.NameplateEffect, ProfileEffect: payload.ProfileEffect, ThemeID: payload.ThemeID,
	}
}

func mergePublicProfile(target *model.UserProfile, payload userPayload) {
	if payload.DisplayName != "" {
		target.DisplayName = payload.DisplayName
	}
	target.Bio, target.BioLinks, target.Pronouns = payload.Bio, payload.BioLinks, payload.Pronouns
	target.Languages, target.Badges, target.RepresentedGroup = payload.Languages, payload.Badges, payload.RepresentedGroup
	target.HasVRCPlus, target.IsEconomyCreator = payload.HasVRCPlus, payload.IsEconomyCreator
	target.AgeVerificationStatus, target.AgeVerified = payload.AgeVerificationStatus, payload.AgeVerified
	target.IconURL, target.IconFrame = payload.IconURL, payload.IconFrame
	target.BannerURL, target.BannerColor, target.BannerType, target.BackgroundType = payload.BannerURL, payload.BannerColor, payload.BannerType, payload.BackgroundType
	target.NameplateEffect, target.ProfileEffect, target.ThemeID = payload.NameplateEffect, payload.ProfileEffect, payload.ThemeID
	profileTags := payload.Tags
	if len(payload.TrustTags) > 0 {
		profileTags = payload.TrustTags
	}
	if len(profileTags) > 0 {
		target.Tags, target.TrustLevel = profileTags, trustLevel(profileTags)
	}
}

func mergePrivateProfile(target *model.UserProfile, payload privateProfilePayload) {
	target.IsFriend, target.Note = payload.IsFriend, payload.Note
	if payload.Status != "" {
		target.Status = payload.Status
	}
	if payload.StatusDescription != "" {
		target.StatusDescription = payload.StatusDescription
	}
	activity := payload.Activity
	target.InstanceID, target.WorldID, target.Location = activity.InstanceID, activity.WorldID, activity.Location
	target.Platform, target.State = activity.Platform, activity.State
	target.LastActivity, target.LastLogin = activity.LastActivity, activity.LastLogin
	target.TravelingToInstance, target.TravelingToLocation, target.TravelingToWorld = activity.TravelingToInstance, activity.TravelingToLocation, activity.TravelingToWorld
}

func (payload mutualFriendPayload) toModel() model.MutualFriend {
	return model.MutualFriend{
		ID: payload.ID, DisplayName: payload.DisplayName, Status: payload.Status,
		StatusDescription: payload.StatusDescription, ImageURL: payload.ImageURL,
		ProfilePicOverride:             payload.ProfilePicOverride,
		CurrentAvatarThumbnailImageURL: payload.CurrentAvatarThumbnailImageURL,
	}
}

func validateUserID(userID string) error {
	if !strings.HasPrefix(userID, "usr_") || len(userID) > 80 || strings.ContainsAny(userID, "/?#") {
		return fmt.Errorf("%w: invalid VRChat user id", ErrInvalidRequest)
	}
	return nil
}

func trustLevel(tags []string) string {
	set := make(map[string]bool, len(tags))
	for _, tag := range tags {
		set[tag] = true
	}
	switch {
	case set["system_trust_legend"] || set["system_trust_veteran"]:
		return "trusted"
	case set["system_trust_trusted"]:
		return "known"
	case set["system_trust_known"]:
		return "user"
	case set["system_trust_basic"]:
		return "new"
	default:
		return "visitor"
	}
}
