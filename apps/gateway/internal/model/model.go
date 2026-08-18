package model

import (
	"encoding/json"
	"time"
)

type SessionStatus string

const (
	SessionAnonymous         SessionStatus = "anonymous"
	SessionTwoFactorRequired SessionStatus = "two_factor_required"
	SessionAuthenticated     SessionStatus = "authenticated"
	SessionUnavailable       SessionStatus = "unavailable"
)

type User struct {
	ID                             string `json:"id,omitempty"`
	DisplayName                    string `json:"displayName,omitempty"`
	CurrentAvatarThumbnailImageURL string `json:"currentAvatarThumbnailImageUrl,omitempty"`
}

type SessionState struct {
	Status  SessionStatus `json:"status"`
	User    *User         `json:"user,omitempty"`
	Methods []string      `json:"methods,omitempty"`
	Message string        `json:"message,omitempty"`
}

type Friend struct {
	ID                             string   `json:"id"`
	DisplayName                    string   `json:"displayName"`
	Status                         string   `json:"status,omitempty"`
	StatusDescription              string   `json:"statusDescription,omitempty"`
	Location                       string   `json:"location,omitempty"`
	Platform                       string   `json:"platform,omitempty"`
	LastPlatform                   string   `json:"lastPlatform,omitempty"`
	UserIcon                       string   `json:"userIcon,omitempty"`
	ImageURL                       string   `json:"imageUrl,omitempty"`
	CurrentAvatarThumbnailImageURL string   `json:"currentAvatarThumbnailImageUrl,omitempty"`
	CurrentAvatarImageURL          string   `json:"currentAvatarImageUrl,omitempty"`
	ProfilePicOverride             string   `json:"profilePicOverride,omitempty"`
	ProfilePicOverrideThumbnail    string   `json:"profilePicOverrideThumbnail,omitempty"`
	Bio                            string   `json:"bio,omitempty"`
	BioLinks                       []string `json:"bioLinks,omitempty"`
	CurrentAvatarTags              []string `json:"currentAvatarTags,omitempty"`
	LastActivity                   string   `json:"lastActivity,omitempty"`
	LastLogin                      string   `json:"lastLogin,omitempty"`
	LastMobile                     string   `json:"lastMobile,omitempty"`
	DeveloperType                  string   `json:"developerType,omitempty"`
	IsFriend                       bool     `json:"isFriend"`
	Online                         bool     `json:"online"`
}

type UserBadge struct {
	ID          string `json:"id,omitempty"`
	BadgeID     string `json:"badgeId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
}

type RepresentedGroup struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IconURL   string `json:"iconUrl,omitempty"`
	BannerURL string `json:"bannerUrl,omitempty"`
}

type UserProfile struct {
	ID                             string            `json:"id"`
	DisplayName                    string            `json:"displayName"`
	Bio                            string            `json:"bio,omitempty"`
	BioLinks                       []string          `json:"bioLinks,omitempty"`
	Pronouns                       string            `json:"pronouns,omitempty"`
	Status                         string            `json:"status,omitempty"`
	StatusDescription              string            `json:"statusDescription,omitempty"`
	Location                       string            `json:"location,omitempty"`
	Platform                       string            `json:"platform,omitempty"`
	LastPlatform                   string            `json:"lastPlatform,omitempty"`
	State                          string            `json:"state,omitempty"`
	DeveloperType                  string            `json:"developerType,omitempty"`
	DateJoined                     string            `json:"dateJoined,omitempty"`
	LastActivity                   string            `json:"lastActivity,omitempty"`
	LastLogin                      string            `json:"lastLogin,omitempty"`
	UserIcon                       string            `json:"userIcon,omitempty"`
	ImageURL                       string            `json:"imageUrl,omitempty"`
	CurrentAvatarImageURL          string            `json:"currentAvatarImageUrl,omitempty"`
	CurrentAvatarThumbnailImageURL string            `json:"currentAvatarThumbnailImageUrl,omitempty"`
	ProfilePicOverride             string            `json:"profilePicOverride,omitempty"`
	ProfilePicOverrideThumbnail    string            `json:"profilePicOverrideThumbnail,omitempty"`
	BannerURL                      string            `json:"bannerUrl,omitempty"`
	IsFriend                       bool              `json:"isFriend"`
	AllowAvatarCopying             bool              `json:"allowAvatarCopying"`
	Tags                           []string          `json:"tags,omitempty"`
	TrustLevel                     string            `json:"trustLevel"`
	Note                           string            `json:"note,omitempty"`
	LastMobile                     string            `json:"lastMobile,omitempty"`
	InstanceID                     string            `json:"instanceId,omitempty"`
	WorldID                        string            `json:"worldId,omitempty"`
	TravelingToInstance            string            `json:"travelingToInstance,omitempty"`
	TravelingToLocation            string            `json:"travelingToLocation,omitempty"`
	TravelingToWorld               string            `json:"travelingToWorld,omitempty"`
	Languages                      []string          `json:"languages,omitempty"`
	Badges                         []UserBadge       `json:"badges,omitempty"`
	RepresentedGroup               *RepresentedGroup `json:"representedGroup,omitempty"`
	HasVRCPlus                     bool              `json:"hasVrcPlus,omitempty"`
	IsEconomyCreator               bool              `json:"isEconomyCreator,omitempty"`
	AgeVerificationStatus          string            `json:"ageVerificationStatus,omitempty"`
	AgeVerified                    bool              `json:"ageVerified,omitempty"`
	IconURL                        string            `json:"iconUrl,omitempty"`
	IconFrame                      string            `json:"iconFrame,omitempty"`
	BannerColor                    string            `json:"bannerColor,omitempty"`
	BannerType                     string            `json:"bannerType,omitempty"`
	BackgroundType                 string            `json:"backgroundType,omitempty"`
	NameplateEffect                string            `json:"nameplateEffect,omitempty"`
	ProfileEffect                  string            `json:"profileEffect,omitempty"`
	ThemeID                        string            `json:"themeId,omitempty"`
	MutualFriendCount              int               `json:"mutualFriendCount,omitempty"`
	MutualGroupCount               int               `json:"mutualGroupCount,omitempty"`
	ProfileSources                 []string          `json:"profileSources,omitempty"`
	ActivityVisibility             string            `json:"activityVisibility,omitempty"`
}

type SelfProfileUpdate struct {
	Status             string   `json:"status"`
	StatusDescription  string   `json:"statusDescription"`
	Pronouns           string   `json:"pronouns"`
	Bio                string   `json:"bio"`
	BioLinks           []string `json:"bioLinks"`
	AllowAvatarCopying bool     `json:"allowAvatarCopying"`
}

type MutualFriend struct {
	ID                             string `json:"id"`
	DisplayName                    string `json:"displayName"`
	Status                         string `json:"status,omitempty"`
	StatusDescription              string `json:"statusDescription,omitempty"`
	ImageURL                       string `json:"imageUrl,omitempty"`
	ProfilePicOverride             string `json:"profilePicOverride,omitempty"`
	CurrentAvatarThumbnailImageURL string `json:"currentAvatarThumbnailImageUrl,omitempty"`
}

type World struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	Description         string           `json:"description,omitempty"`
	AuthorName          string           `json:"authorName,omitempty"`
	ThumbnailImageURL   string           `json:"thumbnailImageUrl,omitempty"`
	ImageURL            string           `json:"imageUrl,omitempty"`
	Capacity            int              `json:"capacity,omitempty"`
	RecommendedCapacity int              `json:"recommendedCapacity,omitempty"`
	Occupants           int              `json:"occupants,omitempty"`
	Favorites           int              `json:"favorites,omitempty"`
	Visits              int              `json:"visits,omitempty"`
	ReleaseStatus       string           `json:"releaseStatus,omitempty"`
	UpdatedAt           time.Time        `json:"updatedAt,omitempty"`
	Tags                []string         `json:"tags,omitempty"`
	PublicInstances     []PublicInstance `json:"publicInstances,omitempty"`
}

type PublicInstance struct {
	InstanceID string `json:"instanceId"`
	Location   string `json:"location"`
	UserCount  int    `json:"userCount"`
	Region     string `json:"region,omitempty"`
	Type       string `json:"type,omitempty"`
}

type FavoriteGroup struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Type        string `json:"type"`
	Visibility  string `json:"visibility,omitempty"`
}

type Group struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	ShortCode         string    `json:"shortCode,omitempty"`
	Description       string    `json:"description,omitempty"`
	IconURL           string    `json:"iconUrl,omitempty"`
	BannerURL         string    `json:"bannerUrl,omitempty"`
	OwnerID           string    `json:"ownerId,omitempty"`
	MemberCount       int       `json:"memberCount,omitempty"`
	Privacy           string    `json:"privacy,omitempty"`
	MemberVisibility  string    `json:"memberVisibility,omitempty"`
	IsRepresenting    bool      `json:"isRepresenting"`
	LastPostCreatedAt time.Time `json:"lastPostCreatedAt,omitempty"`
	LastPostReadAt    time.Time `json:"lastPostReadAt,omitempty"`
}

type GroupPost struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"groupId"`
	AuthorID   string    `json:"authorId,omitempty"`
	Title      string    `json:"title,omitempty"`
	Text       string    `json:"text,omitempty"`
	ImageURL   string    `json:"imageUrl,omitempty"`
	Visibility string    `json:"visibility,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt,omitempty"`
}

type GroupInstance struct {
	InstanceID  string `json:"instanceId"`
	Location    string `json:"location"`
	MemberCount int    `json:"memberCount"`
	World       World  `json:"world"`
}

type CalendarEvent struct {
	ID                  string    `json:"id"`
	GroupID             string    `json:"groupId"`
	Title               string    `json:"title"`
	Description         string    `json:"description,omitempty"`
	Category            string    `json:"category,omitempty"`
	ImageURL            string    `json:"imageUrl,omitempty"`
	StartsAt            time.Time `json:"startsAt"`
	EndsAt              time.Time `json:"endsAt,omitempty"`
	DurationInMS        int64     `json:"durationInMs,omitempty"`
	InterestedUserCount int       `json:"interestedUserCount,omitempty"`
	Languages           []string  `json:"languages,omitempty"`
	Platforms           []string  `json:"platforms,omitempty"`
	AccessType          string    `json:"accessType,omitempty"`
	OccurrenceKind      string    `json:"occurrenceKind,omitempty"`
	Following           bool      `json:"following"`
}

type Avatar struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description,omitempty"`
	AuthorID          string            `json:"authorId,omitempty"`
	AuthorName        string            `json:"authorName,omitempty"`
	ImageURL          string            `json:"imageUrl,omitempty"`
	ThumbnailImageURL string            `json:"thumbnailImageUrl,omitempty"`
	ReleaseStatus     string            `json:"releaseStatus,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	Performance       map[string]string `json:"performance,omitempty"`
	Platforms         []string          `json:"platforms,omitempty"`
	UpdatedAt         time.Time         `json:"updatedAt,omitempty"`
}

type UpstreamWorldFavorite struct {
	ID    string   `json:"id"`
	Tags  []string `json:"tags,omitempty"`
	World World    `json:"world"`
}

type InviteRequest struct {
	ReceiverUserID string `json:"receiverUserId"`
	Location       string `json:"location"`
}

type BoopRequest struct {
	EmojiID string `json:"emojiId"`
}

type GameLogStatus struct {
	State      string     `json:"state"`
	Directory  string     `json:"directory,omitempty"`
	File       string     `json:"file,omitempty"`
	LastReadAt *time.Time `json:"lastReadAt,omitempty"`
	Events     int        `json:"events"`
	Message    string     `json:"message,omitempty"`
}

type UpdateStatus struct {
	State        string    `json:"state"`
	Current      string    `json:"current"`
	Latest       string    `json:"latest,omitempty"`
	PublishedAt  time.Time `json:"publishedAt,omitempty"`
	Source       string    `json:"source,omitempty"`
	DownloadURL  string    `json:"downloadUrl,omitempty"`
	Size         int64     `json:"size,omitempty"`
	ReleaseNotes []string  `json:"releaseNotes,omitempty"`
	Message      string    `json:"message,omitempty"`
}

type WorldFavorite struct {
	World     World     `json:"world"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Instance struct {
	ID              string   `json:"id"`
	WorldID         string   `json:"worldId"`
	InstanceID      string   `json:"instanceId"`
	Name            string   `json:"name,omitempty"`
	Type            string   `json:"type,omitempty"`
	Region          string   `json:"region,omitempty"`
	OwnerID         string   `json:"ownerId,omitempty"`
	GroupAccessType string   `json:"groupAccessType,omitempty"`
	UserCount       int      `json:"userCount,omitempty"`
	Capacity        int      `json:"capacity,omitempty"`
	QueueSize       int      `json:"queueSize,omitempty"`
	QueueEnabled    bool     `json:"queueEnabled,omitempty"`
	Active          bool     `json:"active"`
	Full            bool     `json:"full"`
	Tags            []string `json:"tags,omitempty"`
}

type Notification struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	SenderUserID   string    `json:"senderUserId,omitempty"`
	SenderUsername string    `json:"senderUsername,omitempty"`
	Message        string    `json:"message,omitempty"`
	Seen           bool      `json:"seen"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
	WorldID        string    `json:"worldId,omitempty"`
	WorldName      string    `json:"worldName,omitempty"`
	InstanceID     string    `json:"instanceId,omitempty"`
}

type ActivityEvent struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	UserID       string    `json:"userId,omitempty"`
	DisplayName  string    `json:"displayName,omitempty"`
	WorldID      string    `json:"worldId,omitempty"`
	Location     string    `json:"location,omitempty"`
	LocationKind string    `json:"locationKind,omitempty"`
	Summary      string    `json:"summary"`
	ObservedAt   time.Time `json:"observedAt"`
}

type ActivityBucket struct {
	Weekday int `json:"weekday"`
	Hour    int `json:"hour"`
	Count   int `json:"count"`
}

type ActivityTopUser struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Count       int    `json:"count"`
}

type ActivityInsights struct {
	TotalEvents  int               `json:"totalEvents"`
	CoverageDays int               `json:"coverageDays"`
	Heatmap      []ActivityBucket  `json:"heatmap"`
	TopUsers     []ActivityTopUser `json:"topUsers"`
	GeneratedAt  time.Time         `json:"generatedAt"`
}

type FriendActivityHour struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

type FriendActivityWorld struct {
	WorldID    string    `json:"worldId"`
	Count      int       `json:"count"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

type FriendRelationObservation struct {
	PeerID      string    `json:"peerId"`
	DisplayName string    `json:"displayName,omitempty"`
	State       string    `json:"state"`
	ObservedAt  time.Time `json:"observedAt"`
}

type FriendActivityInsights struct {
	UserID           string                      `json:"userId"`
	TotalEvents      int                         `json:"totalEvents"`
	CoverageDays     int                         `json:"coverageDays"`
	TogetherMinutes  int                         `json:"togetherMinutes"`
	TogetherSessions int                         `json:"togetherSessions"`
	DistinctWorlds   int                         `json:"distinctWorlds"`
	FirstObservedAt  *time.Time                  `json:"firstObservedAt,omitempty"`
	LastMetAt        *time.Time                  `json:"lastMetAt,omitempty"`
	SourceCounts     map[string]int              `json:"sourceCounts"`
	LocationKinds    map[string]int              `json:"locationKinds"`
	PrivateVisits    int                         `json:"privateVisits"`
	ActiveHours      []FriendActivityHour        `json:"activeHours"`
	CommonWorlds     []FriendActivityWorld       `json:"commonWorlds"`
	Timeline         []ActivityEvent             `json:"timeline"`
	RelationChanges  []FriendRelationObservation `json:"relationChanges"`
	GeneratedAt      time.Time                   `json:"generatedAt"`
}

type FriendStatus struct {
	IsFriend        bool `json:"isFriend"`
	IncomingRequest bool `json:"incomingRequest"`
	OutgoingRequest bool `json:"outgoingRequest"`
}

type DataEnvelope[T any] struct {
	Items     []T       `json:"items"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetchedAt"`
	Stale     bool      `json:"stale"`
	Message   string    `json:"message,omitempty"`
	OptedOut  bool      `json:"optedOut,omitempty"`
}

type FriendNetworkNode struct {
	ID                             string     `json:"id"`
	DisplayName                    string     `json:"displayName"`
	Online                         bool       `json:"online"`
	UserIcon                       string     `json:"userIcon,omitempty"`
	ImageURL                       string     `json:"imageUrl,omitempty"`
	CurrentAvatarThumbnailImageURL string     `json:"currentAvatarThumbnailImageUrl,omitempty"`
	Scanned                        bool       `json:"scanned"`
	OptedOut                       bool       `json:"optedOut"`
	ScannedAt                      *time.Time `json:"scannedAt,omitempty"`
}

type FriendNetworkEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type FriendNetwork struct {
	Nodes         []FriendNetworkNode `json:"nodes"`
	Edges         []FriendNetworkEdge `json:"edges"`
	TotalFriends  int                 `json:"totalFriends"`
	ScannedCount  int                 `json:"scannedCount"`
	OptedOutCount int                 `json:"optedOutCount"`
	GeneratedAt   time.Time           `json:"generatedAt"`
}

type FriendAnnotation struct {
	UserID    string    `json:"userId"`
	Note      string    `json:"note,omitempty"`
	Group     string    `json:"group,omitempty"`
	Color     string    `json:"color,omitempty"`
	Tags      []string  `json:"tags"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CacheStats struct {
	DatabaseBytes   int64 `json:"databaseBytes"`
	EntityEntries   int64 `json:"entityEntries"`
	EntityBytes     int64 `json:"entityBytes"`
	MediaFiles      int64 `json:"mediaFiles"`
	MediaBytes      int64 `json:"mediaBytes"`
	AnnotationCount int64 `json:"annotationCount"`
	GroupEntries    int64 `json:"groupEntries"`
	AvatarEntries   int64 `json:"avatarEntries"`
	WorldEntries    int64 `json:"worldEntries"`
}

type CacheClearResult struct {
	RemovedFiles   int64 `json:"removedFiles"`
	RemovedEntries int64 `json:"removedEntries,omitempty"`
	FreedBytes     int64 `json:"freedBytes"`
}

type RealtimeStatus struct {
	State         string     `json:"state"`
	ConnectedAt   *time.Time `json:"connectedAt,omitempty"`
	LastMessageAt *time.Time `json:"lastMessageAt,omitempty"`
	Reconnects    int        `json:"reconnects"`
	Message       string     `json:"message,omitempty"`
}

type DomainEvent struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	ObservedAt time.Time       `json:"observedAt"`
	Content    json.RawMessage `json:"content,omitempty"`
}

type PresenceWatchRule struct {
	UserID         string    `json:"userId"`
	DisplayName    string    `json:"displayName"`
	NotifyOnline   bool      `json:"notifyOnline"`
	NotifyOffline  bool      `json:"notifyOffline"`
	DesktopEnabled bool      `json:"desktopEnabled"`
	EmailEnabled   bool      `json:"emailEnabled"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type EmailSettings struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Security   string `json:"security"`
	Username   string `json:"username"`
	From       string `json:"from"`
	To         string `json:"to"`
	Configured bool   `json:"configured"`
}

type NotificationDelivery struct {
	ID          int64      `json:"id"`
	UserID      string     `json:"userId,omitempty"`
	DisplayName string     `json:"displayName,omitempty"`
	EventType   string     `json:"eventType"`
	Channel     string     `json:"channel"`
	Status      string     `json:"status"`
	Message     string     `json:"message"`
	ObservedAt  time.Time  `json:"observedAt"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type NetworkConfig struct {
	Mode     string `json:"mode"`
	ProxyURL string `json:"proxyUrl,omitempty"`
}

type NetworkState struct {
	NetworkConfig
	Label       string `json:"label"`
	Description string `json:"description"`
}

type CheckState string

const (
	CheckOK       CheckState = "ok"
	CheckDegraded CheckState = "degraded"
	CheckError    CheckState = "error"
	CheckUnknown  CheckState = "unknown"
)

type ProbeResult struct {
	Name       string     `json:"name"`
	State      CheckState `json:"state"`
	StatusCode int        `json:"statusCode,omitempty"`
	LatencyMS  int64      `json:"latencyMs"`
	CheckedAt  time.Time  `json:"checkedAt"`
	Detail     string     `json:"detail"`
}

type Diagnostics struct {
	Overall  CheckState    `json:"overall"`
	Checks   []ProbeResult `json:"checks"`
	Database struct {
		Path  string `json:"path"`
		Ready bool   `json:"ready"`
	} `json:"database"`
	VRChat struct {
		BaseURL   string `json:"baseUrl"`
		UserAgent string `json:"userAgent"`
	} `json:"vrchat"`
}
