export type CheckState = 'ok' | 'degraded' | 'error' | 'unknown'

export interface Bootstrap {
  appName: string
  version: string
  csrfToken: string
  security: {
    loopbackOnly: boolean
    originProtected: boolean
    sessionEncryption: string
  }
}

export interface ProbeResult {
  name: string
  state: CheckState
  statusCode?: number
  latencyMs: number
  checkedAt: string
  detail: string
}

export interface Diagnostics {
  overall: CheckState
  checks: ProbeResult[]
  database: {
    path: string
    ready: boolean
  }
  vrchat: {
    baseUrl: string
    userAgent: string
  }
}

export interface SessionState {
  status: 'anonymous' | 'two_factor_required' | 'authenticated' | 'unavailable'
  user?: {
    id?: string
    displayName?: string
    currentAvatarThumbnailImageUrl?: string
  }
  methods?: string[]
  message?: string
}

export interface Friend {
  id: string
  displayName: string
  status?: string
  statusDescription?: string
  location?: string
  platform?: string
  lastPlatform?: string
  userIcon?: string
  imageUrl?: string
  currentAvatarThumbnailImageUrl?: string
  online: boolean
}

export interface UserProfile {
  id: string
  displayName: string
  bio?: string
  bioLinks?: string[]
  pronouns?: string
  status?: string
  statusDescription?: string
  location?: string
  platform?: string
  lastPlatform?: string
  state?: string
  developerType?: string
  dateJoined?: string
  lastActivity?: string
  lastLogin?: string
  userIcon?: string
  imageUrl?: string
  currentAvatarImageUrl?: string
  currentAvatarThumbnailImageUrl?: string
  profilePicOverride?: string
  profilePicOverrideThumbnail?: string
  bannerUrl?: string
  isFriend: boolean
  allowAvatarCopying: boolean
  tags?: string[]
  trustLevel: 'visitor' | 'new' | 'user' | 'known' | 'trusted'
  note?: string
}

export interface SelfProfileUpdate {
  status: string
  statusDescription: string
  pronouns: string
  bio: string
  bioLinks: string[]
  allowAvatarCopying: boolean
}

export interface MutualFriend {
  id: string
  displayName: string
  status?: string
  statusDescription?: string
  imageUrl?: string
  profilePicOverride?: string
  currentAvatarThumbnailImageUrl?: string
}

export interface World {
  id: string
  name: string
  description?: string
  authorName?: string
  thumbnailImageUrl?: string
  imageUrl?: string
  capacity?: number
  recommendedCapacity?: number
  occupants?: number
  favorites?: number
  visits?: number
  releaseStatus?: string
  updatedAt?: string
  tags?: string[]
  publicInstances?: PublicInstance[]
}

export interface PublicInstance {
  instanceId: string
  location: string
  userCount: number
  region?: string
  type?: string
}

export interface FavoriteGroup {
  name: string
  displayName?: string
  type: string
  visibility?: string
}

export interface Group {
  id: string
  name: string
  shortCode?: string
  description?: string
  iconUrl?: string
  bannerUrl?: string
  ownerId?: string
  memberCount?: number
  privacy?: string
  memberVisibility?: string
  isRepresenting: boolean
  lastPostCreatedAt?: string
  lastPostReadAt?: string
}

export interface GroupPost {
  id: string
  groupId: string
  authorId?: string
  title?: string
  text?: string
  imageUrl?: string
  visibility?: string
  createdAt?: string
  updatedAt?: string
}

export interface GroupInstance { instanceId: string; location: string; memberCount: number; world: World }

export interface CalendarEvent {
  id: string
  groupId: string
  title: string
  description?: string
  category?: string
  imageUrl?: string
  startsAt: string
  endsAt?: string
  durationInMs?: number
  interestedUserCount?: number
  languages?: string[]
  platforms?: string[]
  accessType?: string
  occurrenceKind?: string
  following: boolean
}

export interface Avatar {
  id: string
  name: string
  description?: string
  authorId?: string
  authorName?: string
  imageUrl?: string
  thumbnailImageUrl?: string
  releaseStatus?: string
  tags?: string[]
  performance?: Record<string, string>
  platforms?: string[]
  updatedAt?: string
}

export interface UpstreamWorldFavorite {
  id: string
  tags?: string[]
  world: World
}

export interface GameLogStatus {
  state: 'starting' | 'waiting' | 'watching' | 'error' | 'disabled'
  directory?: string
  file?: string
  lastReadAt?: string
  events: number
  message?: string
}

export interface UpdateStatus {
  state: 'unconfigured' | 'idle' | 'current' | 'available' | 'ready' | 'error' | 'disabled'
  current: string
  latest?: string
  publishedAt?: string
  source?: string
  downloadUrl?: string
  size?: number
  releaseNotes?: string[]
  message?: string
}

export interface WorldFavorite {
  world: World
  note?: string
  createdAt: string
}

export interface Instance {
  id: string
  worldId: string
  instanceId: string
  name?: string
  type?: string
  region?: string
  ownerId?: string
  groupAccessType?: string
  userCount?: number
  capacity?: number
  queueSize?: number
  queueEnabled?: boolean
  active: boolean
  full: boolean
  tags?: string[]
}

export interface VrcNotification {
  id: string
  type: string
  senderUserId?: string
  senderUsername?: string
  message?: string
  seen: boolean
  createdAt?: string
  worldId?: string
  worldName?: string
  instanceId?: string
}

export interface ActivityEvent {
  id: string
  type: string
  userId?: string
  displayName?: string
  worldId?: string
  summary: string
  observedAt: string
}

export interface ActivityInsights {
  totalEvents: number
  coverageDays: number
  heatmap: Array<{ weekday: number; hour: number; count: number }>
  topUsers: Array<{ userId: string; displayName: string; count: number }>
  generatedAt: string
}

export interface FriendActivityInsights {
  userId: string
  totalEvents: number
  coverageDays: number
  togetherMinutes: number
  togetherSessions: number
  distinctWorlds: number
  firstObservedAt?: string
  lastMetAt?: string
  sourceCounts: Record<string, number>
  activeHours: Array<{ hour: number; count: number }>
  commonWorlds: Array<{ worldId: string; count: number; lastSeenAt: string }>
  timeline: ActivityEvent[]
  generatedAt: string
}

export interface FriendStatus {
  isFriend: boolean
  incomingRequest: boolean
  outgoingRequest: boolean
}

export interface DataEnvelope<T> {
  items: T[]
  source: 'live' | 'cache'
  fetchedAt: string
  stale: boolean
  message?: string
  optedOut?: boolean
}

export interface FriendNetworkNode {
  id: string
  displayName: string
  online: boolean
  userIcon?: string
  imageUrl?: string
  currentAvatarThumbnailImageUrl?: string
  scanned: boolean
  optedOut: boolean
  scannedAt?: string
}

export interface FriendNetworkEdge {
  source: string
  target: string
}

export interface FriendNetwork {
  nodes: FriendNetworkNode[]
  edges: FriendNetworkEdge[]
  totalFriends: number
  scannedCount: number
  optedOutCount: number
  generatedAt: string
}

export interface FriendAnnotation {
  userId: string
  note?: string
  group?: string
  color?: string
  tags: string[]
  updatedAt: string
}

export interface CacheStats {
  databaseBytes: number
  entityEntries: number
  entityBytes: number
  mediaFiles: number
  mediaBytes: number
  annotationCount: number
  groupEntries: number
  avatarEntries: number
  worldEntries: number
}

export interface CacheClearResult {
  removedFiles: number
  removedEntries?: number
  freedBytes: number
}

export interface RealtimeStatus {
  state: 'connected' | 'connecting' | 'disconnected' | 'disabled'
  connectedAt?: string
  lastMessageAt?: string
  reconnects: number
  message?: string
}

export interface DomainEvent {
  id: string
  type: string
  observedAt: string
  content?: unknown
}

export interface NetworkState {
  mode: 'system' | 'direct' | 'http' | 'socks5'
  proxyUrl?: string
  label: string
  description: string
}

export class LocalApi {
  private csrfToken = ''

  async bootstrap(): Promise<Bootstrap> {
    const value = await this.request<Bootstrap>('/local/v1/bootstrap')
    this.csrfToken = value.csrfToken
    return value
  }

  diagnostics(refresh = false): Promise<Diagnostics> {
    return this.request(`/local/v1/diagnostics${refresh ? '?refresh=1' : ''}`)
  }

  session(): Promise<SessionState> {
    return this.request('/local/v1/auth/session')
  }

  login(username: string, password: string): Promise<SessionState> {
    return this.request('/local/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  }

  verifyTwoFactor(type: string, code: string): Promise<SessionState> {
    return this.request('/local/v1/auth/2fa', {
      method: 'POST',
      body: JSON.stringify({ type, code }),
    })
  }

  logout(): Promise<SessionState> {
    return this.request('/local/v1/auth/session', { method: 'DELETE' })
  }

  friends(): Promise<DataEnvelope<Friend>> {
    return this.request('/local/v1/friends')
  }

  user(userId: string): Promise<DataEnvelope<UserProfile>> {
    return this.request(`/local/v1/users/${encodeURIComponent(userId)}`)
  }

  updateSelfProfile(value: SelfProfileUpdate): Promise<UserProfile> {
    return this.request('/local/v1/profile', { method: 'PUT', body: JSON.stringify(value) })
  }

  mutualFriends(userId: string, refresh = false): Promise<DataEnvelope<MutualFriend>> {
    return this.request(`/local/v1/users/${encodeURIComponent(userId)}/mutual-friends${refresh ? '?refresh=1' : ''}`)
  }

  searchUsers(query: string, limit = 12): Promise<DataEnvelope<UserProfile>> {
    return this.request(`/local/v1/discovery/users?${new URLSearchParams({ query, limit: String(limit) })}`)
  }

  friendStatus(userId: string): Promise<FriendStatus> {
    return this.request(`/local/v1/users/${encodeURIComponent(userId)}/friend-status`)
  }

  sendFriendRequest(userId: string): Promise<{ ok: boolean }> {
    return this.request(`/local/v1/users/${encodeURIComponent(userId)}/friend-request`, { method: 'POST' })
  }

  friendNetwork(): Promise<FriendNetwork> {
    return this.request('/local/v1/friend-network')
  }

  friendAnnotations(): Promise<FriendAnnotation[]> {
    return this.request('/local/v1/friend-annotations')
  }

  updateFriendAnnotation(userId: string, value: Pick<FriendAnnotation, 'note' | 'group' | 'color' | 'tags'>): Promise<FriendAnnotation> {
    return this.request(`/local/v1/friend-annotations/${encodeURIComponent(userId)}`, {
      method: 'PUT',
      body: JSON.stringify(value),
    })
  }

  cacheStats(): Promise<CacheStats> {
    return this.request('/local/v1/cache')
  }

  clearMediaCache(): Promise<CacheClearResult> {
    return this.request('/local/v1/cache/media', { method: 'DELETE' })
  }

  clearEntityCache(): Promise<CacheClearResult> {
    return this.request('/local/v1/cache/entities', { method: 'DELETE' })
  }

  worlds(search = '', offset = 0, limit = 24): Promise<DataEnvelope<World>> {
    const query = new URLSearchParams({ search, offset: String(offset), limit: String(limit) })
    return this.request(`/local/v1/worlds?${query}`)
  }

  world(worldId: string): Promise<DataEnvelope<World>> {
    return this.request(`/local/v1/worlds/${encodeURIComponent(worldId)}`)
  }

  worldFavorites(): Promise<WorldFavorite[]> {
    return this.request('/local/v1/world-favorites')
  }

  saveWorldFavorite(world: World, note = ''): Promise<WorldFavorite> {
    return this.request(`/local/v1/world-favorites/${encodeURIComponent(world.id)}`, {
      method: 'PUT', body: JSON.stringify({ world, note }),
    })
  }

  deleteWorldFavorite(worldId: string): Promise<void> {
    return this.request(`/local/v1/world-favorites/${encodeURIComponent(worldId)}`, { method: 'DELETE' })
  }

  upstreamWorldFavorites(): Promise<UpstreamWorldFavorite[]> {
    return this.request('/local/v1/vrchat-favorites')
  }

  favoriteGroups(): Promise<FavoriteGroup[]> {
    return this.request('/local/v1/vrchat-favorite-groups')
  }

  groups(userId: string, refresh = false): Promise<DataEnvelope<Group>> {
    return this.request(`/local/v1/groups?${new URLSearchParams({ userId, refresh: refresh ? '1' : '0' })}`)
  }

  groupPosts(groupId: string, refresh = false): Promise<DataEnvelope<GroupPost>> {
    return this.request(`/local/v1/groups/${encodeURIComponent(groupId)}/posts?refresh=${refresh ? '1' : '0'}`)
  }

  groupInstances(groupId: string, refresh = false): Promise<DataEnvelope<GroupInstance>> {
    return this.request(`/local/v1/groups/${encodeURIComponent(groupId)}/instances?refresh=${refresh ? '1' : '0'}`)
  }

  groupCalendar(groupId: string, month = new Date().toISOString().slice(0, 7), refresh = false): Promise<DataEnvelope<CalendarEvent>> {
    return this.request(`/local/v1/groups/${encodeURIComponent(groupId)}/calendar?${new URLSearchParams({ month, refresh: refresh ? '1' : '0' })}`)
  }

  favoriteAvatars(refresh = false): Promise<DataEnvelope<Avatar>> {
    return this.request(`/local/v1/avatars/favorites?refresh=${refresh ? '1' : '0'}`)
  }

  addUpstreamWorldFavorite(worldId: string, group: string): Promise<UpstreamWorldFavorite> {
    return this.request(`/local/v1/vrchat-favorites/${encodeURIComponent(worldId)}`, { method: 'POST', body: JSON.stringify({ group }) })
  }

  deleteUpstreamWorldFavorite(favoriteId: string): Promise<void> {
    return this.request(`/local/v1/vrchat-favorites/${encodeURIComponent(favoriteId)}`, { method: 'DELETE' })
  }

  instance(location: string): Promise<Instance> {
    return this.request(`/local/v1/instance?${new URLSearchParams({ location })}`)
  }

  sendInvite(receiverUserId: string, location: string): Promise<{ ok: boolean }> {
    return this.request('/local/v1/invites', { method: 'POST', body: JSON.stringify({ receiverUserId, location }) })
  }

  sendBoop(userId: string, emojiId: string): Promise<{ ok: boolean }> {
    return this.request(`/local/v1/users/${encodeURIComponent(userId)}/boop`, {
      method: 'POST',
      body: JSON.stringify({ emojiId }),
    })
  }

  gameLogStatus(): Promise<GameLogStatus> {
    return this.request('/local/v1/game-log/status')
  }

  updateStatus(): Promise<UpdateStatus> { return this.request('/local/v1/update') }
  checkUpdate(): Promise<UpdateStatus> { return this.request('/local/v1/update/check', { method: 'POST' }) }
  downloadUpdate(): Promise<UpdateStatus> { return this.request('/local/v1/update/download', { method: 'POST' }) }
  applyUpdate(): Promise<{ restarting: boolean }> { return this.request('/local/v1/update/apply', { method: 'POST' }) }

  activity(days = 30, limit = 100): Promise<ActivityEvent[]> {
    return this.request(`/local/v1/activity?${new URLSearchParams({ days: String(days), limit: String(limit) })}`)
  }

  activityInsights(days = 30): Promise<ActivityInsights> {
    return this.request(`/local/v1/activity/insights?${new URLSearchParams({ days: String(days) })}`)
  }

  friendActivityInsights(userId: string, days = 30): Promise<FriendActivityInsights> {
    return this.request(`/local/v1/activity/users/${encodeURIComponent(userId)}/insights?${new URLSearchParams({ days: String(days) })}`)
  }

  clearActivity(): Promise<void> {
    return this.request('/local/v1/activity', { method: 'DELETE' })
  }

  notifications(): Promise<VrcNotification[]> {
    return this.request('/local/v1/notifications')
  }

  notificationAction(notificationId: string, action: 'see' | 'hide' | 'accept'): Promise<{ ok: boolean }> {
    return this.request(`/local/v1/notifications/${encodeURIComponent(notificationId)}/${action}`, { method: 'POST' })
  }

  realtimeStatus(): Promise<RealtimeStatus> {
    return this.request('/local/v1/realtime/status')
  }

  events(): EventSource {
    return new EventSource('/local/v1/events/stream')
  }

  network(): Promise<NetworkState> {
    return this.request('/local/v1/network')
  }

  updateNetwork(mode: NetworkState['mode'], proxyUrl = ''): Promise<NetworkState> {
    return this.request('/local/v1/network', {
      method: 'PUT',
      body: JSON.stringify({ mode, proxyUrl }),
    })
  }

  mediaUrl(remoteUrl?: string): string {
    if (!remoteUrl) return ''
    return `/local/v1/media?url=${encodeURIComponent(remoteUrl)}`
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers)
    if (init.body) headers.set('Content-Type', 'application/json')
    if (this.csrfToken && init.method && init.method !== 'GET') {
      headers.set('X-CSRF-Token', this.csrfToken)
    }
    const response = await fetch(path, { ...init, headers })
    const payload = await response.json().catch(() => null)
    if (!response.ok) {
      const message = payload?.error?.message ?? `请求失败 (${response.status})`
      throw new Error(message)
    }
    return payload as T
  }
}
