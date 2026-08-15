<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  Activity,
  ALargeSmall,
  Bell,
  Box,
  Building2,
  CheckCircle2,
  CircleAlert,
  Clock3,
  Compass,
  Gauge,
  Gamepad2,
  GlobeLock,
  KeyRound,
  LoaderCircle,
  LockKeyhole,
  Languages,
  LogOut,
  Moon,
  Network,
  PlugZap,
  Radio,
  Route,
  RefreshCw,
  Search,
  Settings2,
  Sparkles,
  ShieldCheck,
  Sun,
  Users,
  Globe2,
  History,
  WifiOff,
  X,
} from '@lucide/vue'
import { preferredFriendAvatar } from './media'
import {
  LocalApi,
  type Bootstrap,
  type CacheStats,
  type DataEnvelope,
  type Diagnostics,
  type DomainEvent,
  type ActivityEvent,
  type ActivityInsights,
  type Friend,
  type FriendActivityInsights,
  type FriendAnnotation,
  type FriendStatus,
  type FriendNetwork,
  type FavoriteGroup,
  type GameLogStatus,
  type Instance,
  type MutualFriend,
  type NetworkState,
  type RealtimeStatus,
  type SessionState,
  type SelfProfileUpdate,
  type UserProfile,
  type UpdateStatus,
  type UpstreamWorldFavorite,
  type VrcNotification,
  type World,
  type WorldFavorite,
} from './api'
import ActivityView from './ActivityView.vue'
import CompassView from './CompassView.vue'
import JourneyView from './JourneyView.vue'
import DiscoveryView from './DiscoveryView.vue'
import FriendDetailDrawer from './FriendDetailDrawer.vue'
import FriendListView from './FriendListView.vue'
import FriendNetworkView from './FriendNetworkView.vue'
import GroupCenter from './GroupCenter.vue'
import AvatarCabinet from './AvatarCabinet.vue'
import GlobalSearch from './GlobalSearch.vue'
import SystemCenter from './SystemCenter.vue'
import TodayView from './TodayView.vue'
import NotificationCenter from './NotificationCenter.vue'
import RecentAccess from './RecentAccess.vue'
import WorldDetailDrawer from './WorldDetailDrawer.vue'
import WorldMemory from './WorldMemory.vue'
import SelfProfileDialog from './SelfProfileDialog.vue'
import { summarizeNetworkDelta } from './friend-network'
import { applyTheme, type Theme } from './theme'
import { applyUIScale, readUIScale } from './accessibility'
import { applyInterfaceLocale, interfaceText, readInterfaceLocale, type InterfaceLocale } from './locale'

const api = new LocalApi()
const bootstrap = ref<Bootstrap | null>(null)
const diagnostics = ref<Diagnostics | null>(null)
const session = ref<SessionState>({ status: 'unavailable', message: '正在读取本机会话' })
const loading = ref(true)
const probing = ref(false)
const submitting = ref(false)
const error = ref('')
const username = ref('')
const password = ref('')
const twoFactorType = ref('totp')
const twoFactorCode = ref('')
const friends = ref<DataEnvelope<Friend> | null>(null)
const worlds = ref<DataEnvelope<World> | null>(null)
const realtime = ref<RealtimeStatus>({ state: 'disabled', reconnects: 0 })
const dataLoading = ref(false)
const worldSearch = ref('')
type View = 'overview' | 'compass' | 'journey' | 'friends' | 'discovery' | 'worlds' | 'groups' | 'avatars' | 'network' | 'activity' | 'notifications'
const activeView = ref<View>('overview')
const settingsOpen = ref(false)
const displaySettingsOpen = ref(false)
const network = ref<NetworkState>({ mode: 'system', label: '跟随系统', description: '读取系统代理配置' })
const networkMode = ref<NetworkState['mode']>('system')
const proxyUrl = ref('')
const savingNetwork = ref(false)
const theme = ref<Theme>((document.documentElement.dataset.theme as Theme) || 'dark')
const uiScale = ref(readUIScale())
const locale = ref<InterfaceLocale>(readInterfaceLocale())
const selfProfileOpen = ref(false)
const selfProfile = ref<UserProfile | null>(null)
const selfProfileLoading = ref(false)
const selfProfileSaving = ref(false)
const selfProfileError = ref('')
const selectedFriend = ref<Friend | null>(null)
const friendProfile = ref<UserProfile | null>(null)
const friendWorld = ref<World | null>(null)
const mutualFriends = ref<MutualFriend[]>([])
const friendActivityInsights = ref<FriendActivityInsights | null>(null)
const selectedFriendStatus = ref<FriendStatus | null>(null)
const friendRequestActingID = ref('')
const boopActingID = ref('')
const boopMessage = ref('')
const friendDetailLoading = ref(false)
const friendDetailError = ref('')
const copyMessage = ref('')
const friendNetwork = ref<FriendNetwork | null>(null)
const networkScanning = ref(false)
const networkScanProcessed = ref(0)
const networkScanTotal = ref(0)
const networkScanMessage = ref('')
const annotations = ref<Record<string, FriendAnnotation>>({})
const annotationSaving = ref(false)
const cacheStats = ref<CacheStats | null>(null)
const systemCenterLoading = ref(false)
const worldFavorites = ref<Record<string, WorldFavorite>>({})
const upstreamWorldFavorites = ref<Record<string, UpstreamWorldFavorite>>({})
const hydratedWorlds = ref<Record<string, World>>({})
const favoriteGroups = ref<FavoriteGroup[]>([])
const selectedWorld = ref<World | null>(null)
const selectedInstance = ref<Instance | null>(null)
const selectedInstanceLocation = ref('')
const worldDetailLoading = ref(false)
const worldFavoriteSaving = ref(false)
const worldUpstreamSaving = ref(false)
const inviteSending = ref(false)
const gameLogStatus = ref<GameLogStatus>({ state: 'starting', events: 0 })
const updateStatus = ref<UpdateStatus>({ state: 'idle', current: '' })
const updateLoading = ref(false)
const activityEvents = ref<ActivityEvent[]>([])
const activityInsights = ref<ActivityInsights | null>(null)
const activityLoading = ref(false)
const notifications = ref<VrcNotification[]>([])
const notificationsLoading = ref(false)
const notificationActingID = ref('')
const discoveryResults = ref<UserProfile[]>([])
const discoveryStatuses = ref<Record<string, FriendStatus>>({})
const discoveryLoading = ref(false)
type RecentAccessItem = { kind: 'friend' | 'world' | 'avatar'; id: string; title: string; subtitle?: string; imageUrl?: string; openedAt: string }
const recentAccess = ref<RecentAccessItem[]>([])
let eventSource: EventSource | null = null
let friendRefreshTimer: number | undefined
let activityRefreshTimer: number | undefined
let friendDetailRequest = 0
let copyMessageTimer: number | undefined
let networkScanCancelled = false
const worldHydrationAttempted = new Set<string>()

const statusCopy = computed(() => {
  if (session.value.status === 'authenticated') return l('已连接 VRChat', 'VRChat connected')
  if (session.value.status === 'two_factor_required') return l('等待二步验证', 'Two-factor authentication required')
  if (session.value.status === 'anonymous') return l('尚未登录', 'Not signed in')
  return l('上游暂不可用', 'Upstream unavailable')
})

const overallLabel = computed(() => {
  const state = diagnostics.value?.overall
  if (state === 'ok') return '连接正常'
  if (state === 'degraded') return '部分受限'
  if (state === 'error') return '连接异常'
  return '尚未检测'
})

const realtimeLabel = computed(() => {
  if (realtime.value.state === 'connected') return l('实时已连接', 'Realtime connected')
  if (realtime.value.state === 'connecting') return l('实时连接中', 'Connecting realtime')
  if (realtime.value.state === 'disconnected') return l('实时重连中', 'Reconnecting realtime')
  return l('实时未启用', 'Realtime disabled')
})

const onlineFriends = computed(() => friends.value?.items.filter((friend) => friend.online).length ?? 0)
const visibleFriends = computed(() => {
  const items = friends.value?.items ?? []
  return items.slice(0, 9)
})
const visibleWorlds = computed(() => {
  const byID = new Map<string, World>()
  for (const item of Object.values(hydratedWorlds.value)) byID.set(item.id, item)
  for (const item of Object.values(worldFavorites.value)) byID.set(item.world.id, item.world)
  for (const item of Object.values(upstreamWorldFavorites.value)) if (!byID.has(item.world.id)) byID.set(item.world.id, item.world)
  for (const item of worlds.value?.items ?? []) if (!byID.has(item.id)) byID.set(item.id, item)
  const items = [...byID.values()]
  return activeView.value === 'overview' ? items.slice(0, 8) : items
})
const showFriends = computed(() => false)
const showWorlds = computed(() => activeView.value === 'worlds')
const selectedAnnotation = computed(() => selectedFriend.value ? annotations.value[selectedFriend.value.id] ?? null : null)
const selectedWorldFavorite = computed(() => selectedWorld.value ? worldFavorites.value[selectedWorld.value.id] ?? null : null)
const selectedUpstreamFavorite = computed(() => selectedWorld.value ? upstreamWorldFavorites.value[selectedWorld.value.id] ?? null : null)
const unreadNotifications = computed(() => notifications.value.filter((item) => !item.seen).length)
const viewCopy = computed(() => (locale.value === 'en' ? {
  overview: ['Dashboard', `Hello, ${session.value.user?.displayName ?? 'Traveler'}`, 'A clear view of today.'],
  compass: ['Compass', 'Where to tonight', 'Worlds that fit your current friends.'],
  journey: ['Journey', 'Friend motion and trails', 'See recent movement and circle intersections.'],
  friends: ['Friends', 'Friend activity', 'Find, filter and review friends.'],
  discovery: ['Discover', 'Discover people', 'Search users and recent encounters.'],
  network: ['Network', 'Friend network', 'Explore circles and mutual links.'],
  worlds: ['Worlds', 'Worlds and instances', 'Discover worlds you can join.'],
  groups: ['Groups', 'Group center', 'Groups, events and posts.'],
  avatars: ['Avatars', 'Avatar collection', 'Organize favorite avatars.'],
  activity: ['History', 'Activity history', 'What happened recently.'],
  notifications: ['Notifications', 'Requests and invites', 'Handle notifications and friend requests.'],
} : {
  overview: ['工作台', `你好，${session.value.user?.displayName ?? '旅行者'}`, '今天有什么新情况。'],
  compass: ['罗盘', '今晚去哪', '看看适合加入的世界。'],
  journey: ['旅程', '好友动向与足迹', '看见圈层交汇与最近去向。'],
  friends: ['好友', '好友动态', '找人、筛选和查看近况。'],
  discovery: ['发现', '发现好友', '搜索用户和最近遇见的人。'],
  network: ['关系网', '好友关系网', '看看好友圈与共同连接。'],
  worlds: ['世界', '世界与实例', '发现世界与可加入实例。'],
  groups: ['群组', '群组中心', '群组、活动与公告。'],
  avatars: ['头像', '头像收藏', '整理收藏头像。'],
  activity: ['历史', '活动记录', '最近发生了什么。'],
  notifications: ['通知', '通知与请求', '处理邀请与好友请求。'],
})[activeView.value])

function l(chinese: string, english: string) { return interfaceText(locale.value, chinese, english) }

function friendImage(friend: Friend) {
  return api.mediaUrl(preferredFriendAvatar(friend))
}

function worldImage(world: World) {
  return api.mediaUrl(world.thumbnailImageUrl || world.imageUrl)
}

function userImage() {
  return api.mediaUrl(session.value.user?.currentAvatarThumbnailImageUrl)
}

function friendLocation(friend: Friend) {
  if (!friend.online) return '离线'
  if (!friend.location || friend.location === 'private') return '私人实例'
  if (friend.location === 'traveling') return '正在切换世界'
  return friend.location.startsWith('wrld_') ? '在线世界' : friend.location
}

async function initialize() {
  loading.value = true
  error.value = ''
  try {
    bootstrap.value = await api.bootstrap()
    const [diag, currentSession, networkResult] = await Promise.allSettled([
      api.diagnostics(),
      api.session(),
      api.network(),
    ])
    if (diag.status === 'fulfilled') diagnostics.value = diag.value
    if (currentSession.status === 'fulfilled') session.value = currentSession.value
    if (networkResult.status === 'fulfilled') setNetworkState(networkResult.value)
    if (session.value.status === 'authenticated') await loadAppData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '本机网关连接失败'
  } finally {
    loading.value = false
  }
}

function setNetworkState(value: NetworkState) {
  network.value = value
  networkMode.value = value.mode
  proxyUrl.value = value.proxyUrl ?? ''
}

async function saveNetwork() {
  savingNetwork.value = true
  error.value = ''
  try {
    const value = await api.updateNetwork(networkMode.value, proxyUrl.value.trim())
    setNetworkState(value)
    diagnostics.value = await api.diagnostics(true)
    settingsOpen.value = false
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '网络设置保存失败'
  } finally {
    savingNetwork.value = false
  }
}

function selectView(view: View) {
  activeView.value = view
  if (view === 'network' || view === 'journey') void loadFriendNetwork()
  if (view === 'journey') void refreshSystemCenter()
  if (view === 'activity' || view === 'discovery') void loadActivity()
  if (view === 'notifications') void loadNotifications()
}

function recentKey() { return `vrc-harbor-recent:${session.value.user?.id ?? 'default'}` }
function loadRecentAccess() {
  try { recentAccess.value = JSON.parse(localStorage.getItem(recentKey()) || '[]') } catch { recentAccess.value = [] }
}
function recordRecent(item: Omit<RecentAccessItem, 'openedAt'>) {
  const entry = { ...item, openedAt: new Date().toISOString() }
  recentAccess.value = [entry, ...recentAccess.value.filter((value) => value.kind !== item.kind || value.id !== item.id)].slice(0, 12)
  localStorage.setItem(recentKey(), JSON.stringify(recentAccess.value))
}
function openRecent(item: RecentAccessItem) {
  if (item.kind === 'friend') {
    const friend = friends.value?.items.find((value) => value.id === item.id)
    if (friend) void openFriend(friend); else void resolveSearchUser(item.id)
  } else if (item.kind === 'world') {
    const world = visibleWorlds.value.find((value) => value.id === item.id)
    if (world) void openWorldDetail(world); else void resolveSearchWorld(item.id)
  }
}

async function loadFriendNetwork() {
  try {
    friendNetwork.value = await api.friendNetwork()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '共同好友网络读取失败'
  }
}

function networkScanCandidates(requestedIDs: string[] = []) {
  const nodes = friendNetwork.value?.nodes ?? []
  const nodeByID = new Map(nodes.map((node) => [node.id, node]))
  const requested = new Set(requestedIDs)
  const hasRequested = requested.size > 0
  const staleBefore = Date.now() - 7 * 24 * 60 * 60 * 1000
  return [...(friends.value?.items ?? [])].filter((friend) => {
    if (hasRequested) return requested.has(friend.id)
    const node = nodeByID.get(friend.id)
    if (!node?.scanned || !node.scannedAt) return true
    return new Date(node.scannedAt).getTime() < staleBefore
  }).sort((left, right) => {
    if (hasRequested) return requestedIDs.indexOf(left.id) - requestedIDs.indexOf(right.id)
    const leftNode = nodeByID.get(left.id)
    const rightNode = nodeByID.get(right.id)
    if (leftNode?.scanned !== rightNode?.scanned) return leftNode?.scanned ? 1 : -1
    if (left.online !== right.online) return left.online ? -1 : 1
    return (leftNode?.scannedAt ?? '').localeCompare(rightNode?.scannedAt ?? '')
  }).slice(0, 20)
}

const networkScanEstimate = computed(() => networkScanCandidates().length)

async function startNetworkScan(requestedIDs: string[] = []) {
  if (networkScanning.value) return
  const candidates = networkScanCandidates(requestedIDs)
  if (!candidates.length) {
    networkScanMessage.value = requestedIDs.length ? '没有找到可扫描的所选好友' : '没有需要更新的节点：已扫描数据均在 7 天有效期内'
    return
  }
  networkScanCancelled = false
  const beforeNetwork = friendNetwork.value
  networkScanning.value = true
  networkScanProcessed.value = 0
  networkScanTotal.value = candidates.length
  networkScanMessage.value = requestedIDs.length
    ? `正在扫描手动选择的 ${candidates.length} 位好友`
    : `本次将观察 ${candidates.length} 位好友；新连线数量以返回结果为准`
  try {
    for (const friend of candidates) {
      if (networkScanCancelled) break
      networkScanMessage.value = `正在扫描 ${friend.displayName}`
      try {
        await api.mutualFriends(friend.id, true)
      } catch (cause) {
        const message = cause instanceof Error ? cause.message : '扫描失败'
        if (message.includes('频繁') || message.includes('429')) {
          networkScanMessage.value = 'VRChat 提示请求过于频繁，本批已提前停止'
          break
        }
      }
      networkScanProcessed.value += 1
      if (networkScanProcessed.value % 3 === 0) await loadFriendNetwork()
    }
    await loadFriendNetwork()
    if (networkScanCancelled) networkScanMessage.value = `已停止，本批完成 ${networkScanProcessed.value} 位`
    else if (networkScanProcessed.value === networkScanTotal.value) {
      const delta = summarizeNetworkDelta(beforeNetwork, friendNetwork.value)
      networkScanMessage.value = `本批完成：观察 ${networkScanProcessed.value} 位，新增 ${delta.addedScanned} 个已扫描节点、${delta.addedEdges} 条连线，${delta.addedConnected} 位好友进入连接区`
    }
  } finally {
    networkScanning.value = false
  }
}

async function saveFriendAnnotation(value: { note: string; group: string; color: string; tags: string[] }) {
  if (!selectedFriend.value) return
  annotationSaving.value = true
  try {
    const saved = await api.updateFriendAnnotation(selectedFriend.value.id, value)
    annotations.value = { ...annotations.value, [saved.userId]: saved }
  } catch (cause) {
    friendDetailError.value = cause instanceof Error ? cause.message : '本机备注保存失败'
  } finally {
    annotationSaving.value = false
  }
}

async function resolveSearchUser(userID: string) {
  const existing = friends.value?.items.find((friend) => friend.id === userID)
  if (existing) return void openFriend(existing)
  try {
    const result = await api.user(userID)
    const profile = result.items[0]
    if (profile) void openFriend({ id: profile.id, displayName: profile.displayName, status: profile.status, location: profile.location, platform: profile.platform, userIcon: profile.userIcon, currentAvatarThumbnailImageUrl: profile.currentAvatarThumbnailImageUrl, online: profile.state === 'online' })
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '用户读取失败'
  }
}

function friendFromProfile(profile: UserProfile): Friend {
  return {
    id: profile.id,
    displayName: profile.displayName,
    status: profile.status,
    statusDescription: profile.statusDescription,
    location: profile.location,
    platform: profile.platform,
    lastPlatform: profile.lastPlatform,
    userIcon: profile.userIcon,
    imageUrl: profile.imageUrl,
    currentAvatarThumbnailImageUrl: profile.currentAvatarThumbnailImageUrl,
    online: profile.state === 'online' || (!!profile.location && profile.location !== 'offline'),
  }
}

async function searchDiscoveryUsers(query: string) {
  discoveryLoading.value = true
  error.value = ''
  try {
    const result = query.startsWith('usr_') ? await api.user(query) : await api.searchUsers(query)
    discoveryResults.value = result.items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '用户搜索失败'
  } finally {
    discoveryLoading.value = false
  }
}

function openDiscoveryUser(profile: UserProfile) {
  void openFriend(friendFromProfile(profile))
}

async function sendUserFriendRequest(profile: UserProfile) {
  friendRequestActingID.value = profile.id
  try {
    const current = discoveryStatuses.value[profile.id] ?? await api.friendStatus(profile.id)
    discoveryStatuses.value = { ...discoveryStatuses.value, [profile.id]: current }
    if (current.isFriend || current.outgoingRequest || current.incomingRequest) {
      if (selectedFriend.value?.id === profile.id) selectedFriendStatus.value = current
      return
    }
    if (!window.confirm(`向“${profile.displayName}”发送好友请求？此操作会立即提交到 VRChat。`)) return
    await api.sendFriendRequest(profile.id)
    const status = await api.friendStatus(profile.id)
    discoveryStatuses.value = { ...discoveryStatuses.value, [profile.id]: status }
    if (selectedFriend.value?.id === profile.id) selectedFriendStatus.value = status
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '好友请求发送失败'
  } finally {
    friendRequestActingID.value = ''
  }
}

async function sendFriendBoop(emojiId: string) {
  const friend = selectedFriend.value
  if (!friend || boopActingID.value) return
  if (!window.confirm(`向“${friend.displayName}”发送一次戳一戳？`)) return
  boopActingID.value = friend.id
  boopMessage.value = ''
  friendDetailError.value = ''
  try {
    await api.sendBoop(friend.id, emojiId)
    if (selectedFriend.value?.id === friend.id) boopMessage.value = '已发送到 VRChat'
  } catch (cause) {
    if (selectedFriend.value?.id === friend.id) friendDetailError.value = cause instanceof Error ? cause.message : '戳一戳发送失败'
  } finally {
    boopActingID.value = ''
  }
}

async function resolveSearchWorld(worldID: string) {
  try {
    const result = await api.world(worldID)
    const item = result.items[0]
    if (item) selectSearchWorld(item)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '世界读取失败'
  }
}

function selectSearchWorld(world: World) {
  const rest = (worlds.value?.items ?? []).filter((item) => item.id !== world.id)
  worlds.value = { items: [world, ...rest], source: 'live', fetchedAt: new Date().toISOString(), stale: false }
  worldSearch.value = world.name
  activeView.value = 'worlds'
  void openWorldDetail(world, '')
}

async function openWorldDetail(world: World, location = '') {
  recordRecent({ kind: 'world', id: world.id, title: world.name, subtitle: world.authorName, imageUrl: world.thumbnailImageUrl || world.imageUrl })
  closeFriendDetail()
  selectedWorld.value = world
  selectedInstance.value = null
  selectedInstanceLocation.value = location.startsWith(`${world.id}:`) ? location : ''
	worldDetailLoading.value = true
	try {
	  const detail = await api.world(world.id)
	  selectedWorld.value = detail.items[0] ?? world
	  if (selectedWorld.value) hydratedWorlds.value = { ...hydratedWorlds.value, [selectedWorld.value.id]: selectedWorld.value }
	} catch {
	  // Search results still provide a usable world summary while the detail endpoint is unavailable.
	}
	if (!selectedInstanceLocation.value) { worldDetailLoading.value = false; return }
	try {
    selectedInstance.value = await api.instance(selectedInstanceLocation.value)
  } catch {
    // The world and explicit launch location remain useful when instance metadata is private.
  } finally {
    worldDetailLoading.value = false
  }
}

async function selectPublicInstance(location: string) {
  if (!selectedWorld.value || !location.startsWith(`${selectedWorld.value.id}:`)) return
  selectedInstanceLocation.value = location
  selectedInstance.value = null
  worldDetailLoading.value = true
  try { selectedInstance.value = await api.instance(location) } catch { /* summary remains usable */ }
  finally { worldDetailLoading.value = false }
}

function closeWorldDetail() {
  selectedWorld.value = null
  selectedInstance.value = null
  selectedInstanceLocation.value = ''
  worldDetailLoading.value = false
}

async function saveWorldFavorite(note: string) {
  if (!selectedWorld.value) return
  worldFavoriteSaving.value = true
  try {
    const item = await api.saveWorldFavorite(selectedWorld.value, note)
    worldFavorites.value = { ...worldFavorites.value, [item.world.id]: item }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '本机世界收藏保存失败'
  } finally {
    worldFavoriteSaving.value = false
  }
}

async function removeWorldFavorite() {
  if (!selectedWorld.value || !window.confirm('只删除本机收藏和备注，不会修改 VRChat 收藏。继续吗？')) return
  worldFavoriteSaving.value = true
  try {
    await api.deleteWorldFavorite(selectedWorld.value.id)
    const next = { ...worldFavorites.value }
    delete next[selectedWorld.value.id]
    worldFavorites.value = next
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '本机世界收藏删除失败'
  } finally {
    worldFavoriteSaving.value = false
  }
}

async function addUpstreamFavorite(group: string) {
  if (!selectedWorld.value || !window.confirm(`把“${selectedWorld.value.name}”加入 VRChat 上游收藏？此操作会立即写入你的 VRChat 账号。`)) return
  worldUpstreamSaving.value = true
  try {
    await api.addUpstreamWorldFavorite(selectedWorld.value.id, group)
    const items = await api.upstreamWorldFavorites()
    upstreamWorldFavorites.value = Object.fromEntries(items.map((item) => [item.world.id, item]))
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'VRChat 收藏保存失败' }
  finally { worldUpstreamSaving.value = false }
}

async function removeUpstreamFavorite() {
  const favorite = selectedUpstreamFavorite.value
  if (!favorite || !window.confirm('从 VRChat 上游收藏中移除这个世界？本机备注不会删除。')) return
  worldUpstreamSaving.value = true
  try {
    await api.deleteUpstreamWorldFavorite(favorite.id)
    const next = { ...upstreamWorldFavorites.value }; delete next[favorite.world.id]; upstreamWorldFavorites.value = next
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'VRChat 收藏移除失败' }
  finally { worldUpstreamSaving.value = false }
}

async function sendWorldInvite(receiverUserId: string) {
  const friend = friends.value?.items.find((item) => item.id === receiverUserId)
  if (!selectedWorld.value || !selectedInstanceLocation.value || !friend) return
  if (!window.confirm(`邀请 ${friend.displayName} 前往“${selectedWorld.value.name}”？此操作会立即发送到 VRChat。`)) return
  inviteSending.value = true
  try { await api.sendInvite(receiverUserId, selectedInstanceLocation.value) }
  catch (cause) { error.value = cause instanceof Error ? cause.message : '邀请发送失败' }
  finally { inviteSending.value = false }
}

async function runUpdateAction(action: 'check' | 'download' | 'apply') {
  updateLoading.value = true
  try {
    if (action === 'check') updateStatus.value = await api.checkUpdate()
    if (action === 'download') updateStatus.value = await api.downloadUpdate()
    if (action === 'apply' && window.confirm('安装更新会重启 VRC++。继续吗？')) await api.applyUpdate()
  } catch (cause) { error.value = cause instanceof Error ? cause.message : '更新操作失败' }
  finally { updateLoading.value = false }
}

async function loadActivity() {
  activityLoading.value = true
  const [eventsResult, insightsResult] = await Promise.allSettled([api.activity(), api.activityInsights()])
  if (eventsResult.status === 'fulfilled') activityEvents.value = eventsResult.value
  if (insightsResult.status === 'fulfilled') activityInsights.value = insightsResult.value
  if (eventsResult.status === 'rejected' && insightsResult.status === 'rejected') error.value = '本机活动历史读取失败'
  activityLoading.value = false
}

async function clearActivity() {
  if (!window.confirm('将永久清除这台电脑上的活动事件历史。好友备注、关系网和登录会话不会删除。继续吗？')) return
  try {
    await api.clearActivity()
    await loadActivity()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '活动历史清除失败'
  }
}

function selectActivityUser(userID: string) {
  const friend = friends.value?.items.find((item) => item.id === userID)
  if (friend) void openFriend(friend)
  else void resolveSearchUser(userID)
}

async function loadNotifications() {
  notificationsLoading.value = true
  try {
    notifications.value = await api.notifications()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'VRChat 通知读取失败'
  } finally {
    notificationsLoading.value = false
  }
}

async function actOnNotification(item: VrcNotification, action: 'see' | 'hide' | 'accept') {
  const actionLabel = action === 'accept' ? '接受这条好友请求' : action === 'hide' ? '从 VRChat 通知列表隐藏此项' : '将此通知标记为已读'
  if (!window.confirm(`${actionLabel}？此操作会立即提交到 VRChat。`)) return
  notificationActingID.value = item.id
  try {
    await api.notificationAction(item.id, action)
    await loadNotifications()
    if (action === 'accept') friends.value = await api.friends()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '通知操作失败'
  } finally {
    notificationActingID.value = ''
  }
}

async function openNotificationWorld(worldID: string, location: string) {
  try {
    const result = await api.world(worldID)
    const world = result.items[0]
    if (world) await openWorldDetail(world, location)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '邀请世界读取失败'
  }
}

async function refreshSystemCenter() {
  systemCenterLoading.value = true
  try {
    const [diag, cache, realtimeResult, logResult, updateResult] = await Promise.allSettled([api.diagnostics(true), api.cacheStats(), api.realtimeStatus(), api.gameLogStatus(), api.updateStatus()])
    if (diag.status === 'fulfilled') diagnostics.value = diag.value
    if (cache.status === 'fulfilled') cacheStats.value = cache.value
    if (realtimeResult.status === 'fulfilled') realtime.value = realtimeResult.value
	if (logResult.status === 'fulfilled') gameLogStatus.value = logResult.value
	if (updateResult.status === 'fulfilled') updateStatus.value = updateResult.value
	const failures = [diag, cache, realtimeResult, logResult, updateResult].filter((result) => result.status === 'rejected')
	if (failures.length === 5) error.value = '系统各层状态暂时都无法读取'
  } finally {
    systemCenterLoading.value = false
  }
}

async function clearMediaCache() {
  if (!window.confirm('只清理可重新下载的图片缓存。登录会话、备注、关系网和布局不会删除。继续吗？')) return
  systemCenterLoading.value = true
  try {
    await api.clearMediaCache()
    cacheStats.value = await api.cacheStats()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '图片缓存清理失败'
  } finally {
    systemCenterLoading.value = false
  }
}

function stopNetworkScan() {
  networkScanCancelled = true
  networkScanMessage.value = '将在当前请求完成后停止'
}

function openNetworkFriend(userID: string) {
  const friend = friends.value?.items.find((item) => item.id === userID)
  if (friend) void openFriend(friend)
}

async function openCompassWorld(worldID: string, location = '') {
  const known = visibleWorlds.value.find((item) => item.id === worldID)
  if (known) { await openWorldDetail(known, location); return }
  try { const result = await api.world(worldID); const world=result.items[0]; if(world) await openWorldDetail(world, location) }
  catch (cause) { error.value = cause instanceof Error ? cause.message : '路线世界读取失败' }
}

async function hydrateVisibleWorlds(worldIDs: string[]) {
  const candidates = [...new Set(worldIDs)].filter((worldID) => worldID.startsWith('wrld_') && !worldHydrationAttempted.has(worldID)).slice(0, 12)
  for (const worldID of candidates) {
    worldHydrationAttempted.add(worldID)
    try {
      const result = await api.world(worldID)
      const world = result.items[0]
      if (world) hydratedWorlds.value = { ...hydratedWorlds.value, [world.id]: world }
    } catch {
      // Keep the location usable and do not retry in a tight render loop.
    }
  }
}

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  applyTheme(theme.value)
}

function setUIScale(event: Event) {
  uiScale.value = Number((event.target as HTMLInputElement).value) / 100
}

function toggleLocale() {
  locale.value = locale.value === 'zh-CN' ? 'en' : 'zh-CN'
}

async function openSelfProfile() {
  const userID = session.value.user?.id
  if (!userID) return
  selfProfileOpen.value = true
  selfProfileLoading.value = true
  selfProfileError.value = ''
  try {
    const result = await api.user(userID)
    selfProfile.value = result.items[0] ?? null
    if (!selfProfile.value) selfProfileError.value = l('没有读取到个人资料', 'Profile data was empty')
  } catch (cause) {
    selfProfileError.value = cause instanceof Error ? cause.message : l('个人资料读取失败', 'Unable to load profile')
  } finally {
    selfProfileLoading.value = false
  }
}

async function saveSelfProfile(value: SelfProfileUpdate) {
  if (!window.confirm(l('这些资料会立即写入你的 VRChat 账号，继续吗？', 'These changes will be written to your VRChat account now. Continue?'))) return
  selfProfileSaving.value = true
  selfProfileError.value = ''
  try {
    selfProfile.value = await api.updateSelfProfile(value)
  } catch (cause) {
    selfProfileError.value = cause instanceof Error ? cause.message : l('个人资料保存失败', 'Unable to save profile')
  } finally {
    selfProfileSaving.value = false
  }
}

function worldIDFromLocation(value?: string) {
  if (!value?.startsWith('wrld_')) return ''
  return value.split(':')[0]
}

async function openFriend(friend: Friend) {
  recordRecent({ kind: 'friend', id: friend.id, title: friend.displayName, subtitle: friend.statusDescription, imageUrl: preferredFriendAvatar(friend) })
  const requestID = ++friendDetailRequest
  selectedFriend.value = friend
  friendProfile.value = null
  friendWorld.value = null
  mutualFriends.value = []
  friendActivityInsights.value = null
  selectedFriendStatus.value = null
  friendDetailError.value = ''
  boopMessage.value = ''
  friendDetailLoading.value = true
  copyMessage.value = ''

  const knownFriend = friends.value?.items.some((item) => item.id === friend.id) ?? false
  if (knownFriend) selectedFriendStatus.value = { isFriend: true, incomingRequest: false, outgoingRequest: false }
  let loadedProfile: UserProfile | null = null
  let loadedLocation = friend.location || ''
  const profileRequest = api.user(friend.id).then((result) => {
    loadedProfile = result.items[0] ?? null
    loadedLocation = loadedProfile?.location || loadedLocation
    if (requestID === friendDetailRequest) friendProfile.value = loadedProfile
  }).catch((cause) => {
    if (requestID === friendDetailRequest) friendDetailError.value = cause instanceof Error ? cause.message : '好友资料读取失败'
  })
  const mutualRequest = api.mutualFriends(friend.id).then((result) => {
    if (requestID === friendDetailRequest) mutualFriends.value = result.items
  }).catch(() => { /* profile remains usable when mutuals are unavailable */ })
  const insightRequest = api.friendActivityInsights(friend.id).then((result) => {
    if (requestID === friendDetailRequest) friendActivityInsights.value = result
  }).catch(() => { /* empty local insight state is explicit in the drawer */ })
  const statusRequest = (knownFriend ? Promise.resolve(selectedFriendStatus.value!) : api.friendStatus(friend.id)).then((result) => {
    if (requestID === friendDetailRequest) selectedFriendStatus.value = result
  }).catch(() => { /* unknown relationship state disables no existing read-only feature */ })
  await Promise.all([profileRequest, mutualRequest, insightRequest, statusRequest])
  if (requestID !== friendDetailRequest) return

  const worldID = worldIDFromLocation(loadedLocation)
  if (worldID) {
    try {
      const result = await api.world(worldID)
      if (requestID === friendDetailRequest) friendWorld.value = result.items[0] ?? null
    } catch {
      // The profile remains useful when instance world details are unavailable.
    }
  }
  if (requestID === friendDetailRequest) friendDetailLoading.value = false
}

function openMutualFriend(mutual: MutualFriend) {
  const existing = friends.value?.items.find((friend) => friend.id === mutual.id)
  void openFriend(existing ?? {
    id: mutual.id,
    displayName: mutual.displayName,
    status: mutual.status,
    statusDescription: mutual.statusDescription,
    imageUrl: mutual.imageUrl,
    currentAvatarThumbnailImageUrl: mutual.currentAvatarThumbnailImageUrl,
    online: false,
  })
}

function closeFriendDetail() {
  friendDetailRequest += 1
  selectedFriend.value = null
  friendProfile.value = null
  friendWorld.value = null
  mutualFriends.value = []
  friendActivityInsights.value = null
  selectedFriendStatus.value = null
  friendDetailLoading.value = false
  friendDetailError.value = ''
  boopMessage.value = ''
}

async function copyFriendID() {
  if (!selectedFriend.value) return
  try {
    await navigator.clipboard.writeText(selectedFriend.value.id)
    copyMessage.value = '已复制'
  } catch {
    copyMessage.value = '复制失败'
  }
  window.clearTimeout(copyMessageTimer)
  copyMessageTimer = window.setTimeout(() => { copyMessage.value = '' }, 1800)
}

async function loadAppData() {
  loadRecentAccess()
  dataLoading.value = true
  let primaryError: unknown = null
  const tasks = [
    api.friends().then((value) => { friends.value = value }).catch((cause) => { primaryError ??= cause }),
    api.worlds('', 0, 24).then((value) => { worlds.value = value }).catch((cause) => { primaryError ??= cause }),
    api.realtimeStatus().then((value) => { realtime.value = value }),
    api.friendAnnotations().then((value) => { annotations.value = Object.fromEntries(value.map((item) => [item.userId, item])) }),
    api.worldFavorites().then((value) => { worldFavorites.value = Object.fromEntries(value.map((item) => [item.world.id, item])) }),
    api.upstreamWorldFavorites().then((value) => { upstreamWorldFavorites.value = Object.fromEntries(value.map((item) => [item.world.id, item])) }),
    api.favoriteGroups().then((value) => { favoriteGroups.value = value }),
    api.gameLogStatus().then((value) => { gameLogStatus.value = value }),
    api.updateStatus().then((value) => { updateStatus.value = value }),
    api.activity(30, 500).then((value) => { activityEvents.value = value }),
    api.activityInsights(30).then((value) => { activityInsights.value = value }),
    api.notifications().then((value) => { notifications.value = value }),
  ]
  await Promise.allSettled(tasks)
  if (primaryError) error.value = primaryError instanceof Error ? primaryError.message : '读取 VRChat 数据失败'
  dataLoading.value = false
  connectEvents()
}

async function searchWorlds() {
  dataLoading.value = true
  error.value = ''
  try {
    worlds.value = await api.worlds(worldSearch.value.trim(), 0, 24)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '世界搜索失败'
  } finally {
    dataLoading.value = false
  }
}

function connectEvents() {
  if (eventSource || session.value.status !== 'authenticated') return
  eventSource = api.events()
  eventSource.onmessage = (message) => {
    const event = JSON.parse(message.data) as DomainEvent
    if (event.type === 'pipeline.status' && event.content) {
      realtime.value = event.content as RealtimeStatus
    }
    if (event.type.startsWith('vrc.friend-') || event.type.includes('friend')) {
      window.clearTimeout(friendRefreshTimer)
      friendRefreshTimer = window.setTimeout(async () => {
        try { friends.value = await api.friends() } catch { /* keep the current snapshot */ }
      }, 1200)
    }
    if (event.type.startsWith('vrc.')) {
      window.clearTimeout(activityRefreshTimer)
      activityRefreshTimer = window.setTimeout(() => {
        if (activeView.value === 'activity') void loadActivity()
        if (event.type.includes('notification') && activeView.value === 'notifications') void loadNotifications()
      }, 1500)
    }
  }
}

function closeEvents() {
  eventSource?.close()
  eventSource = null
  window.clearTimeout(friendRefreshTimer)
  window.clearTimeout(activityRefreshTimer)
}

async function refreshDiagnostics() {
  probing.value = true
  error.value = ''
  try {
    diagnostics.value = await api.diagnostics(true)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '检测失败'
  } finally {
    probing.value = false
  }
}

async function login() {
  submitting.value = true
  error.value = ''
  try {
    session.value = await api.login(username.value.trim(), password.value)
    password.value = ''
    if (session.value.methods?.length) {
      twoFactorType.value = session.value.methods.includes('totp')
        ? 'totp'
        : session.value.methods[0]
    }
    if (session.value.status === 'authenticated') await loadAppData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '登录失败'
  } finally {
    password.value = ''
    submitting.value = false
  }
}

async function verifyTwoFactor() {
  submitting.value = true
  error.value = ''
  try {
    session.value = await api.verifyTwoFactor(twoFactorType.value, twoFactorCode.value.trim())
    twoFactorCode.value = ''
    if (session.value.status === 'authenticated') await loadAppData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '验证失败'
  } finally {
    submitting.value = false
  }
}

async function logout() {
  submitting.value = true
  try {
    closeEvents()
    closeFriendDetail()
    session.value = await api.logout()
    friends.value = null
    worlds.value = null
    friendNetwork.value = null
    worldFavorites.value = {}
    activityEvents.value = []
    activityInsights.value = null
    notifications.value = []
  } finally {
    submitting.value = false
  }
}

function checkIcon(state: string) {
  if (state === 'ok') return CheckCircle2
  if (state === 'degraded') return CircleAlert
  if (state === 'error') return WifiOff
  return Clock3
}

onMounted(initialize)
watch(settingsOpen, (open) => { if (open) void refreshSystemCenter() })
watch(uiScale, (value) => { uiScale.value = applyUIScale(value) })
watch(locale, (value) => { locale.value = applyInterfaceLocale(value) })
onBeforeUnmount(() => {
  networkScanCancelled = true
  closeEvents()
  window.clearTimeout(copyMessageTimer)
  window.clearTimeout(activityRefreshTimer)
})
</script>

<template>
  <div class="app-shell" :class="{ 'console-mode': session.status === 'authenticated' }">
    <template v-if="session.status !== 'authenticated'">
      <header class="topbar">
        <a class="brand" href="#"><span class="brand-mark"><img :src="'/assets/vrc-plus-plus-mark.png'" alt="" /></span><span><strong>VRC++</strong><small>本地 VRChat 管理工具</small></span></a>
        <div class="topbar-actions"><button class="theme-button" :title="theme === 'dark' ? '切换浅色模式' : '切换深色模式'" @click="toggleTheme"><Sun v-if="theme === 'dark'" :size="17" /><Moon v-else :size="17" /></button><button class="top-action" @click="settingsOpen = true"><Settings2 :size="16" /> 网络设置</button><div class="topbar-status"><span class="pulse-dot"></span><span>本机网关</span><code>{{ bootstrap?.version ?? '—' }}</code></div></div>
      </header>
      <main class="landing-main">
        <section class="hero modern-hero">
          <div class="eyebrow"><ShieldCheck :size="15" /> 账号会话只保存在这台电脑</div>
          <h1>VRChat 的本地控制台</h1>
          <p>查看好友状态、搜索世界并诊断网络连接。页面资源全部内置，API、实时消息和图片可以统一使用你的本机代理。</p>
          <div class="hero-actions"><button class="primary-button" :disabled="probing" @click="refreshDiagnostics"><LoaderCircle v-if="probing" class="spin" :size="18" /><RefreshCw v-else :size="18" />{{ probing ? '正在检测' : '检测当前线路' }}</button><button class="ghost-button" @click="settingsOpen = true"><PlugZap :size="17" /> {{ network.label }}</button></div>
          <div class="hero-proof"><span><CheckCircle2 :size="15" /> 无公共 CDN</span><span><CheckCircle2 :size="15" /> 图片本机缓存</span><span><CheckCircle2 :size="15" /> Cookie 系统加密</span></div>
        </section>
        <div v-if="error" class="error-banner"><CircleAlert :size="18" /> {{ error }}</div>
        <section class="landing-grid" :class="{ loading }">
          <article class="panel login-panel">
            <div class="panel-heading"><div><span class="panel-kicker">账号登录</span><h2>{{ statusCopy }}</h2></div><div class="icon-tile"><KeyRound :size="22" /></div></div>
            <form v-if="session.status === 'two_factor_required'" class="login-form" @submit.prevent="verifyTwoFactor"><label>验证方式<select v-model="twoFactorType"><option v-for="method in session.methods" :key="method" :value="method">{{ method === 'totp' ? '验证器 TOTP' : method === 'emailOtp' ? '邮件验证码' : '恢复码' }}</option></select></label><label>验证码<input v-model="twoFactorCode" autocomplete="one-time-code" placeholder="输入验证码" required /></label><button class="primary-button full" :disabled="submitting">完成验证</button><small class="form-note">验证码仅由本机网关发送到 VRChat。</small></form>
            <form v-else class="login-form" @submit.prevent="login"><label>VRChat 用户名或邮箱<input v-model="username" autocomplete="username" placeholder="name@example.com" required /></label><label>密码<input v-model="password" type="password" autocomplete="current-password" placeholder="••••••••" required /></label><button class="primary-button full" :disabled="submitting"><LoaderCircle v-if="submitting" class="spin" :size="18" /><LockKeyhole v-else :size="18" />{{ submitting ? '正在安全连接' : '连接我的 VRChat' }}</button><small class="form-note">密码不落库、不进入浏览器存储，仅保存系统加密后的登录 Cookie。</small></form>
          </article>
          <article class="panel route-panel"><div class="panel-heading"><div><span class="panel-kicker">连接状态</span><h2>{{ overallLabel }}</h2></div><div class="status-orb" :data-state="diagnostics?.overall ?? 'unknown'"><Activity :size="22" /></div></div><div class="network-summary"><GlobeLock :size="20" /><div><strong>{{ network.label }}</strong><small>{{ network.description }}</small></div><button @click="settingsOpen = true">切换</button></div><div class="check-list"><div v-for="check in diagnostics?.checks" :key="check.name" class="check-row"><component :is="checkIcon(check.state)" :size="19" :data-state="check.state" /><div><strong>{{ check.name }}</strong><small>{{ check.detail }}</small></div><span>{{ check.latencyMs }} ms</span></div><div v-if="!diagnostics?.checks.length" class="empty-state">等待本机线路检测</div></div></article>
        </section>
      </main>
    </template>

    <template v-else>
      <aside class="sidebar">
        <div class="sidebar-brand"><span class="brand-mark"><img :src="'/assets/vrc-plus-plus-mark.png'" alt="" /></span><div><strong>VRC++</strong><small>{{ bootstrap?.version }}</small></div></div>
        <nav><button :class="{ active: activeView === 'overview' }" @click="selectView('overview')"><Gauge :size="19" /> {{ l('总览','Overview') }}</button><button :class="{ active: activeView === 'compass' }" @click="selectView('compass')"><Sparkles :size="19" /> {{ l('罗盘','Compass') }}</button><button :class="{ active: activeView === 'journey' }" @click="selectView('journey')"><Route :size="19" /> {{ l('旅程','Journey') }}</button><button :class="{ active: activeView === 'friends' }" @click="selectView('friends')"><Users :size="19" /> {{ l('好友','Friends') }}<span>{{ onlineFriends }}</span></button><button :class="{ active: activeView === 'discovery' }" @click="selectView('discovery')"><Compass :size="19" /> {{ l('发现','Discover') }}</button><button :class="{ active: activeView === 'network' }" @click="selectView('network')"><Network :size="19" /> {{ l('关系网','Network') }}</button><button :class="{ active: activeView === 'worlds' }" @click="selectView('worlds')"><Globe2 :size="19" /> {{ l('世界','Worlds') }}</button><button :class="{ active: activeView === 'groups' }" @click="selectView('groups')"><Building2 :size="19" /> {{ l('群组','Groups') }}</button><button :class="{ active: activeView === 'avatars' }" @click="selectView('avatars')"><Box :size="19" /> {{ l('头像','Avatars') }}</button><button :class="{ active: activeView === 'activity' }" @click="selectView('activity')"><History :size="19" /> {{ l('历史','History') }}</button><button :class="{ active: activeView === 'notifications' }" @click="selectView('notifications')"><Bell :size="19" /> {{ l('通知','Notifications') }}<span v-if="unreadNotifications">{{ unreadNotifications }}</span></button></nav>
        <div class="sidebar-bottom"><button @click="toggleTheme"><Sun v-if="theme === 'dark'" :size="18" /><Moon v-else :size="18" /> {{ theme === 'dark' ? l('浅色模式','Light theme') : l('深色模式','Dark theme') }}</button><button @click="toggleLocale"><Languages :size="18" /> {{ locale === 'zh-CN' ? 'English' : '简体中文' }}</button><button @click="displaySettingsOpen = true"><ALargeSmall :size="18" /> {{ l('字体大小','Text size') }} <span>{{ Math.round(uiScale*100) }}%</span></button><button @click="settingsOpen = true"><Settings2 :size="18" /> {{ l('网络与缓存','Network & cache') }}</button><div class="sidebar-user"><button class="sidebar-user-main" :title="l('查看和编辑我的资料','View and edit my profile')" @click="openSelfProfile"><img v-if="userImage()" :src="userImage()" alt="" /><span v-else class="mini-avatar">{{ session.user?.displayName?.slice(0, 1) }}</span><span><strong>{{ session.user?.displayName }}</strong><small>{{ realtimeLabel }}</small></span></button><button :title="l('退出','Sign out')" @click="logout"><LogOut :size="17" /></button></div></div>
      </aside>
      <main class="console-main">
        <header class="console-header"><div><span class="panel-kicker">{{ viewCopy[0] }}</span><h1>{{ viewCopy[1] }}</h1><p>{{ viewCopy[2] }}</p></div><div class="console-actions"><GlobalSearch :friends="friends?.items ?? []" :worlds="worlds?.items ?? []" :annotations="annotations" @select-friend="openFriend" @select-world="selectSearchWorld" @resolve-user="resolveSearchUser" @resolve-world="resolveSearchWorld" /><button class="theme-button" :title="theme === 'dark' ? '切换浅色模式' : '切换深色模式'" @click="toggleTheme"><Sun v-if="theme === 'dark'" :size="17" /><Moon v-else :size="17" /></button><span class="route-pill"><PlugZap :size="14" /> {{ network.label }}</span><button class="icon-button" title="刷新" @click="refreshDiagnostics"><RefreshCw :size="17" /></button></div></header>
        <div v-if="error" class="error-banner"><CircleAlert :size="18" /> {{ error }}</div>
        <section class="console-grid" :class="{ loading: dataLoading }">
          <TodayView v-if="activeView === 'overview'" :friends="friends?.items ?? []" :worlds="worlds?.items ?? []" :notifications="notifications" :events="activityEvents" :favorite-count="Object.keys(upstreamWorldFavorites).length" :realtime-label="realtimeLabel" :route-label="network.label" :media-url="api.mediaUrl.bind(api)" @open-friend="openFriend" @select-view="selectView" />
          <CompassView v-if="activeView === 'compass'" :friends="friends?.items ?? []" :worlds="visibleWorlds" :events="activityEvents" :network="friendNetwork" :favorite-ids="Object.keys(upstreamWorldFavorites)" :storage-key="session.user?.id ?? 'default'" :realtime-label="realtimeLabel" :route-label="network.label" @open-friend="openNetworkFriend" @open-world="openCompassWorld" @load-network="loadFriendNetwork" />
          <JourneyView v-if="activeView === 'journey'" :friends="friends?.items ?? []" :worlds="visibleWorlds" :events="activityEvents" :network="friendNetwork" :diagnostics="diagnostics" :realtime="realtime" :network-state="network" :cache="cacheStats" :storage-key="session.user?.id ?? 'default'" :media-url="api.mediaUrl.bind(api)" :locale="locale" @open-friend="openNetworkFriend" @open-world="openCompassWorld" @load-network="loadFriendNetwork" @refresh-system="refreshSystemCenter" @hydrate-worlds="hydrateVisibleWorlds" />
          <RecentAccess v-if="activeView === 'overview'" :items="recentAccess" :media-url="api.mediaUrl.bind(api)" @open="openRecent" />
          <FriendNetworkView v-if="activeView === 'network'" :network="friendNetwork" :scanning="networkScanning" :scan-processed="networkScanProcessed" :scan-total="networkScanTotal" :scan-estimate="networkScanEstimate" :scan-message="networkScanMessage" :media-url="api.mediaUrl.bind(api)" :layout-key="session.user?.id ?? 'default'" :annotations="annotations" @start-scan="startNetworkScan" @scan-friends="startNetworkScan" @stop-scan="stopNetworkScan" @open-friend="openNetworkFriend" />
          <FriendListView v-if="activeView === 'friends'" :friends="friends?.items ?? []" :annotations="annotations" :media-url="api.mediaUrl.bind(api)" :storage-key="session.user?.id ?? 'default'" :source="friends?.source ?? 'live'" :message="friends?.message" @open-friend="openFriend" />
          <DiscoveryView v-if="activeView === 'discovery'" :friends="friends?.items ?? []" :events="activityEvents" :results="discoveryResults" :statuses="discoveryStatuses" :loading="discoveryLoading" :acting-id="friendRequestActingID" :media-url="api.mediaUrl.bind(api)" @search="searchDiscoveryUsers" @open="openDiscoveryUser" @request="sendUserFriendRequest" />
          <ActivityView v-if="activeView === 'activity'" class="wide-view" :events="activityEvents" :insights="activityInsights" :friends="friends?.items ?? []" :loading="activityLoading" :media-url="api.mediaUrl.bind(api)" @refresh="loadActivity" @clear="clearActivity" @select-user="selectActivityUser" />
          <NotificationCenter v-if="activeView === 'notifications'" class="wide-view" :items="notifications" :loading="notificationsLoading" :acting-id="notificationActingID" @refresh="loadNotifications" @action="actOnNotification" @open-world="openNotificationWorld" />
          <GroupCenter v-if="activeView === 'groups'" :user-id="session.user?.id ?? ''" :media-url="api.mediaUrl.bind(api)" @open-world="openWorldDetail" />
          <AvatarCabinet v-if="activeView === 'avatars'" :storage-key="session.user?.id ?? 'default'" :media-url="api.mediaUrl.bind(api)" />
          <WorldMemory v-if="activeView === 'worlds'" :events="activityEvents" :worlds="visibleWorlds" :favorite-ids="Object.keys(upstreamWorldFavorites)" @open-world="openWorldDetail" />
          <article v-if="showFriends" class="panel friends-panel" :class="{ wide: activeView === 'friends' }">
            <div class="panel-heading compact">
              <div><span class="panel-kicker">好友</span><h2>好友状态</h2></div>
              <div class="data-heading-actions"><span class="source-chip" :data-source="friends?.source ?? 'live'">{{ friends?.source === 'cache' ? '本地快照' : '实时数据' }}</span><span class="realtime-chip" :data-state="realtime.state"><Radio :size="13" /> {{ realtimeLabel }}</span></div>
            </div>
            <p v-if="friends?.message" class="snapshot-note">{{ friends.message }}</p>
            <div class="friend-grid">
              <button v-for="friend in visibleFriends" :key="friend.id" type="button" class="friend-card" :class="{ offline: !friend.online }" :style="annotations[friend.id]?.color ? { borderLeftColor: annotations[friend.id].color, borderLeftWidth: '3px' } : undefined" @click="openFriend(friend)">
                <img v-if="friendImage(friend)" :src="friendImage(friend)" alt="" :loading="activeView === 'overview' ? 'eager' : 'lazy'" decoding="async" />
                <span v-else class="friend-fallback">{{ friend.displayName.slice(0, 1) }}</span>
                <span class="presence-dot"></span>
                <span class="friend-copy"><strong>{{ friend.displayName }}</strong><small>{{ annotations[friend.id]?.group || friendLocation(friend) }} · {{ annotations[friend.id]?.tags.join(' · ') || friend.platform || friend.lastPlatform || '未知平台' }}</small><span v-if="annotations[friend.id]?.note || friend.statusDescription" class="friend-description">{{ annotations[friend.id]?.note || friend.statusDescription }}</span></span>
              </button>
              <div v-if="!friends?.items.length" class="empty-state"><Users :size="19" /> 暂无好友快照</div>
            </div>
            <button v-if="activeView === 'overview'" class="section-link" @click="selectView('friends')">查看全部好友 <span>→</span></button>
          </article>
          <article v-if="showWorlds" class="panel worlds-panel" :class="{ wide: activeView === 'worlds' }"><div class="panel-heading compact world-heading"><div><span class="panel-kicker">世界</span><h2>世界与收藏</h2></div><span class="source-chip" :data-source="worlds?.source ?? 'live'">{{ Object.keys(upstreamWorldFavorites).length }} 个 VRChat 收藏</span></div><form class="world-search" @submit.prevent="searchWorlds"><Search :size="17" /><input v-model="worldSearch" placeholder="搜索世界名称" aria-label="世界名称" /><button type="submit" :disabled="dataLoading">搜索</button></form><div class="world-grid"><button v-for="world in visibleWorlds" :key="world.id" type="button" class="world-card" @click="openWorldDetail(world)"><div class="world-image"><img v-if="worldImage(world)" :src="worldImage(world)" alt="" :loading="activeView === 'overview' ? 'eager' : 'lazy'" decoding="async" /><Globe2 v-else :size="28" /><span v-if="world.occupants !== undefined">{{ world.occupants }} 在线</span></div><div class="world-copy"><strong>{{ world.name }}</strong><small>{{ world.authorName || '未知作者' }} · 容量 {{ world.recommendedCapacity || world.capacity || '—' }}</small><div class="world-badges"><span v-if="upstreamWorldFavorites[world.id]">VRChat 收藏</span><span v-if="worldFavorites[world.id]">本机收藏</span></div><p>{{ worldFavorites[world.id]?.note || world.description || '暂无世界简介' }}</p></div></button><div v-if="!visibleWorlds.length" class="empty-state"><Globe2 :size="19" /> 没有找到匹配世界</div></div><button v-if="activeView === 'overview'" class="section-link" @click="selectView('worlds')">探索更多世界 <span>→</span></button></article>
        </section>
      </main>
    </template>

    <FriendDetailDrawer
      v-if="selectedFriend"
      :friend="selectedFriend"
      :profile="friendProfile"
      :world="friendWorld"
      :mutuals="mutualFriends"
      :loading="friendDetailLoading"
      :error="friendDetailError"
      :copy-message="copyMessage"
      :media-url="api.mediaUrl.bind(api)"
      :annotation="selectedAnnotation"
      :annotation-saving="annotationSaving"
      :insights="friendActivityInsights"
      :friend-status="selectedFriendStatus"
      :friend-request-acting="friendRequestActingID === selectedFriend.id"
      :boop-acting="boopActingID === selectedFriend.id"
      :boop-message="boopMessage"
      @close="closeFriendDetail"
      @copy-id="copyFriendID"
      @select-friend="openMutualFriend"
      @open-world="openWorldDetail"
      @save-annotation="saveFriendAnnotation"
      @send-friend-request="sendUserFriendRequest"
      @send-boop="sendFriendBoop"
    />

    <WorldDetailDrawer
      v-if="selectedWorld"
      :world="selectedWorld"
      :favorite="selectedWorldFavorite"
      :upstream-favorite="selectedUpstreamFavorite"
      :favorite-groups="favoriteGroups"
      :friends="friends?.items ?? []"
      :instance="selectedInstance"
      :instance-location="selectedInstanceLocation"
      :instance-loading="worldDetailLoading"
      :saving="worldFavoriteSaving"
      :upstream-saving="worldUpstreamSaving"
      :invite-sending="inviteSending"
      :image-url="worldImage(selectedWorld)"
      @close="closeWorldDetail"
      @save-favorite="saveWorldFavorite"
      @remove-favorite="removeWorldFavorite"
      @select-instance="selectPublicInstance"
      @add-upstream-favorite="addUpstreamFavorite"
      @remove-upstream-favorite="removeUpstreamFavorite"
      @send-invite="sendWorldInvite"
    />

    <SelfProfileDialog v-if="selfProfileOpen" :profile="selfProfile" :loading="selfProfileLoading" :saving="selfProfileSaving" :error="selfProfileError" :locale="locale" :media-url="api.mediaUrl.bind(api)" @close="selfProfileOpen = false" @save="saveSelfProfile" />

    <div v-if="settingsOpen" class="modal-backdrop" @click.self="settingsOpen = false"><section class="settings-modal"><div class="modal-heading"><div><span class="panel-kicker">本机设置</span><h2>网络、日志与更新</h2></div><button class="icon-button" @click="settingsOpen = false"><X :size="18" /></button></div><p>网络配置同时用于 VRChat REST、Pipeline 和图片缓存。建议优先使用“跟随系统”。</p><SystemCenter :diagnostics="diagnostics" :realtime="realtime" :network="network" :cache="cacheStats" :loading="systemCenterLoading" @refresh="refreshSystemCenter" @clear-media="clearMediaCache" /><div class="runtime-strip"><span><Gamepad2 :size="16" /><b>游戏日志</b>{{ gameLogStatus.state === 'watching' ? '监视中' : gameLogStatus.state === 'waiting' ? '等待 VRChat' : gameLogStatus.state }} · {{ gameLogStatus.events }} 条</span><span><RefreshCw :size="16" /><b>自动更新</b>{{ updateStatus.message || updateStatus.state }}<button :disabled="updateLoading || updateStatus.state === 'unconfigured'" @click="runUpdateAction(updateStatus.state === 'available' ? 'download' : updateStatus.state === 'ready' ? 'apply' : 'check')">{{ updateStatus.state === 'available' ? '下载' : updateStatus.state === 'ready' ? '安装并重启' : '检查' }}</button></span></div><div class="mode-grid"><button v-for="mode in ([['system','跟随系统','读取 HTTP_PROXY / HTTPS_PROXY'],['direct','强制直连','忽略所有代理环境变量'],['http','HTTP 代理','适合 Clash HTTP 端口'],['socks5','SOCKS5','适合本机 SOCKS5 端口']] as const)" :key="mode[0]" :class="{ active: networkMode === mode[0] }" @click="networkMode = mode[0]"><span>{{ mode[1] }}</span><small>{{ mode[2] }}</small></button></div><label v-if="networkMode === 'http' || networkMode === 'socks5'" class="proxy-field">代理地址<input v-model="proxyUrl" :placeholder="networkMode === 'http' ? 'http://127.0.0.1:7890' : 'socks5://127.0.0.1:7891'" /><small>仅支持本机或自有代理；URL 中不保存用户名和密码。</small></label><div class="modal-note"><GlobeLock :size="18" /><span>应用不依赖公共前端 CDN；API、实时连接和图片代理统一由本机网关管理。</span></div><button class="primary-button full" :disabled="savingNetwork" @click="saveNetwork"><LoaderCircle v-if="savingNetwork" class="spin" :size="18" /><PlugZap v-else :size="18" />应用网络配置并重新检测</button></section></div>
    <div v-if="displaySettingsOpen" class="modal-backdrop" @click.self="displaySettingsOpen = false"><section class="settings-modal display-settings-modal"><div class="modal-heading"><div><span class="panel-kicker">{{ l('显示设置','Display') }}</span><h2>{{ l('字体与语言','Text and language') }}</h2></div><button class="icon-button" @click="displaySettingsOpen = false"><X :size="18" /></button></div><div class="font-scale-preview"><ALargeSmall :size="22"/><div><strong>{{ l('界面大小','Interface size') }} {{ Math.round(uiScale*100) }}%</strong><span>{{ l('会同时放大文字、头像和按钮，设置只保存在当前浏览器。','Scales text, avatars and controls. Saved only in this browser.') }}</span></div></div><label class="font-scale-control"><span>90%</span><input type="range" min="90" max="140" step="5" :value="Math.round(uiScale*100)" @input="setUIScale"/><span>140%</span></label><div class="language-setting"><Languages :size="19"/><div><strong>{{ l('界面语言','Language') }}</strong><span>{{ locale === 'zh-CN' ? '简体中文' : 'English' }}</span></div><button @click="toggleLocale">{{ locale === 'zh-CN' ? 'Switch to English' : '切换到简体中文' }}</button></div></section></div>
  </div>
</template>
