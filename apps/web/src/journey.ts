import type { ActivityEvent, CacheStats, Diagnostics, Friend, FriendNetwork, NetworkState, RealtimeStatus, World } from './api'
import { detectNetworkCommunities, selectCommunityTheme } from './friend-network'

export interface FriendMotionScene {
  id: string
  kind: 'live' | 'recent'
  title: string
  worldId?: string
  location?: string
  observedAt: string
  userIds: string[]
  eventCount: number
  communityIds: number[]
  communityNames: string[]
  crossEdges: number
  bridgeIds: string[]
}

export interface CommunityIntersection {
  id: string
  leftId: number
  rightId: number
  leftName: string
  rightName: string
  sceneCount: number
  crossEdges: number
  userIds: string[]
  worldIds: string[]
  bridgeIds: string[]
}

export interface FriendMotionOverview {
  onlineCount: number
  visibleCount: number
  privateCount: number
  offlineCount: number
  coverageDays: number
  liveScenes: FriendMotionScene[]
  recentScenes: FriendMotionScene[]
  intersections: CommunityIntersection[]
}

interface CommunityContext {
  byUser: Map<string, number>
  names: Map<number, string>
  edges: FriendNetwork['edges']
}

function buildCommunityContext(network: FriendNetwork | null): CommunityContext {
  if (!network?.nodes.length) return { byUser: new Map(), names: new Map(), edges: [] }
  const detected = detectNetworkCommunities(network.nodes, network.edges)
  const grouped = new Map<number, typeof network.nodes>()
  for (const node of network.nodes) {
    const communityId = detected.get(node.id)
    if (communityId === undefined) continue
    const members = grouped.get(communityId) ?? []
    members.push(node)
    grouped.set(communityId, members)
  }
  const byUser = new Map<string, number>()
  const names = new Map<number, string>()
  for (const [communityId, nodes] of grouped) {
    const connected = nodes.length > 1 || network.edges.some((edge) => edge.source === nodes[0].id || edge.target === nodes[0].id)
    if (!connected) continue
    nodes.forEach((node) => byUser.set(node.id, communityId))
    const theme = selectCommunityTheme(nodes, network.edges)
    names.set(communityId, `${theme?.node.displayName ?? `朋友圈 ${communityId + 1}`}的朋友圈`)
  }
  return { byUser, names, edges: network.edges }
}

function analyzeScene(
  base: Omit<FriendMotionScene, 'communityIds' | 'communityNames' | 'crossEdges' | 'bridgeIds'>,
  communities: CommunityContext,
): FriendMotionScene {
  const participants = new Set(base.userIds)
  const communityIds = [...new Set(base.userIds.map((id) => communities.byUser.get(id)).filter((id): id is number => id !== undefined))]
    .sort((left, right) => left - right)
  const bridgeScores = new Map<string, number>()
  let crossEdges = 0
  for (const edge of communities.edges) {
    if (!participants.has(edge.source) || !participants.has(edge.target)) continue
    const sourceCommunity = communities.byUser.get(edge.source)
    const targetCommunity = communities.byUser.get(edge.target)
    if (sourceCommunity === undefined || targetCommunity === undefined || sourceCommunity === targetCommunity) continue
    crossEdges += 1
    bridgeScores.set(edge.source, (bridgeScores.get(edge.source) ?? 0) + 1)
    bridgeScores.set(edge.target, (bridgeScores.get(edge.target) ?? 0) + 1)
  }
  return {
    ...base,
    communityIds,
    communityNames: communityIds.map((id) => communities.names.get(id) ?? `朋友圈 ${id + 1}`),
    crossEdges,
    bridgeIds: [...bridgeScores].sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0])).slice(0, 3).map(([id]) => id),
  }
}

function worldIdFromLocation(location?: string) {
  return location?.startsWith('wrld_') ? location.split(':')[0] : ''
}

function motionEventGroup(type: string) {
  const normalized = type.toLowerCase()
  if (normalized.includes('offline') || normalized.includes('left')) return '离线波次'
  if (normalized.includes('online') || normalized.includes('joined')) return '上线波次'
  return '状态变化'
}

export function buildFriendMotionOverview(
  friends: Friend[],
  worlds: World[],
  events: ActivityEvent[],
  network: FriendNetwork | null,
): FriendMotionOverview {
  const communities = buildCommunityContext(network)
  const worldById = new Map(worlds.map((world) => [world.id, world]))
  const onlineFriends = friends.filter((friend) => friend.online)
  const visibleFriends = onlineFriends.filter((friend) => worldIdFromLocation(friend.location))
  const liveGroups = new Map<string, Friend[]>()
  for (const friend of visibleFriends) {
    const location = friend.location!
    const group = liveGroups.get(location) ?? []
    group.push(friend)
    liveGroups.set(location, group)
  }
  const liveScenes = [...liveGroups].map(([location, members]) => {
    const worldId = worldIdFromLocation(location)
    return analyzeScene({
      id: `live:${location}`,
      kind: 'live',
      title: worldById.get(worldId)?.name || '可见世界',
      worldId,
      location,
      observedAt: new Date().toISOString(),
      userIds: members.map((friend) => friend.id),
      eventCount: members.length,
    }, communities)
  }).sort((left, right) => right.userIds.length - left.userIds.length || right.communityIds.length - left.communityIds.length || left.title.localeCompare(right.title))

  const recentGroups = new Map<string, { title: string; worldId?: string; observedAt: string; userIds: Set<string>; eventCount: number }>()
  const bucketSize = 20 * 60 * 1000
  for (const event of events) {
    if (!event.userId) continue
    const timestamp = new Date(event.observedAt).getTime()
    if (Number.isNaN(timestamp)) continue
    const bucket = Math.floor(timestamp / bucketSize)
    const title = event.worldId ? worldById.get(event.worldId)?.name || '世界动向' : motionEventGroup(event.type)
    const key = `${bucket}:${event.worldId || motionEventGroup(event.type)}`
    const group = recentGroups.get(key) ?? { title, worldId: event.worldId, observedAt: event.observedAt, userIds: new Set<string>(), eventCount: 0 }
    group.userIds.add(event.userId)
    group.eventCount += 1
    if (event.observedAt > group.observedAt) group.observedAt = event.observedAt
    recentGroups.set(key, group)
  }
  const recentScenes = [...recentGroups].map(([id, group]) => analyzeScene({
    id: `recent:${id}`,
    kind: 'recent',
    title: group.title,
    worldId: group.worldId,
    observedAt: group.observedAt,
    userIds: [...group.userIds],
    eventCount: group.eventCount,
  }, communities)).sort((left, right) => right.observedAt.localeCompare(left.observedAt))

  const intersectionMap = new Map<string, {
    leftId: number; rightId: number; sceneCount: number; crossEdges: number; userIds: Set<string>; worldIds: Set<string>; bridgeScores: Map<string, number>
  }>()
  for (const scene of [...liveScenes, ...recentScenes]) {
    for (let left = 0; left < scene.communityIds.length; left += 1) for (let right = left + 1; right < scene.communityIds.length; right += 1) {
      const leftId = scene.communityIds[left]
      const rightId = scene.communityIds[right]
      const id = `${leftId}:${rightId}`
      const value = intersectionMap.get(id) ?? { leftId, rightId, sceneCount: 0, crossEdges: 0, userIds: new Set<string>(), worldIds: new Set<string>(), bridgeScores: new Map<string, number>() }
      value.sceneCount += 1
      const participants = new Set(scene.userIds)
      for (const edge of communities.edges) {
        if (!participants.has(edge.source) || !participants.has(edge.target)) continue
        const sourceCommunity = communities.byUser.get(edge.source)
        const targetCommunity = communities.byUser.get(edge.target)
        if (!((sourceCommunity === leftId && targetCommunity === rightId) || (sourceCommunity === rightId && targetCommunity === leftId))) continue
        value.crossEdges += 1
        value.bridgeScores.set(edge.source, (value.bridgeScores.get(edge.source) ?? 0) + 1)
        value.bridgeScores.set(edge.target, (value.bridgeScores.get(edge.target) ?? 0) + 1)
      }
      scene.userIds.filter((userId) => {
        const communityId = communities.byUser.get(userId)
        return communityId === leftId || communityId === rightId
      }).forEach((userId) => value.userIds.add(userId))
      if (scene.worldId) value.worldIds.add(scene.worldId)
      intersectionMap.set(id, value)
    }
  }
  const intersections = [...intersectionMap].map(([id, value]) => ({
    id,
    leftId: value.leftId,
    rightId: value.rightId,
    leftName: communities.names.get(value.leftId) ?? `朋友圈 ${value.leftId + 1}`,
    rightName: communities.names.get(value.rightId) ?? `朋友圈 ${value.rightId + 1}`,
    sceneCount: value.sceneCount,
    crossEdges: value.crossEdges,
    userIds: [...value.userIds],
    worldIds: [...value.worldIds],
    bridgeIds: [...value.bridgeScores].sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0])).slice(0, 3).map(([id]) => id),
  })).sort((left, right) => right.sceneCount - left.sceneCount || right.crossEdges - left.crossEdges || right.userIds.length - left.userIds.length)

  return {
    onlineCount: onlineFriends.length,
    visibleCount: visibleFriends.length,
    privateCount: onlineFriends.length - visibleFriends.length,
    offlineCount: friends.length - onlineFriends.length,
    coverageDays: new Set(events.map((event) => event.observedAt.slice(0, 10)).filter(Boolean)).size,
    liveScenes,
    recentScenes,
    intersections,
  }
}

export interface SocialTide {
  id: number
  name: string
  memberIds: string[]
  eventCount: number
  coverageDays: number
  peak: string
  buckets: number[][]
}

export interface WorldPassportEntry {
  worldId: string
  name: string
  imageUrl?: string
  firstSeenAt: string
  lastSeenAt: string
  eventCount: number
  companionIds: string[]
}

export type ReadinessState = 'ok' | 'warn' | 'error' | 'unknown'
export interface ReadinessCheck { id: string; label: string; state: ReadinessState; detail: string }

const weekdayNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

export function buildSocialTides(network: FriendNetwork | null, events: ActivityEvent[]): SocialTide[] {
  if (!network?.nodes.length) return []
  const communities = detectNetworkCommunities(network.nodes, network.edges)
  const grouped = new Map<number, typeof network.nodes>()
  for (const node of network.nodes) {
    const id = communities.get(node.id)
    if (id === undefined) continue
    const members = grouped.get(id) ?? []
    members.push(node)
    grouped.set(id, members)
  }
  const result: SocialTide[] = []
  for (const [id, nodes] of grouped) {
    if (nodes.length < 2) continue
    const memberIds = new Set(nodes.map((node) => node.id))
    const relevant = events.filter((event) => event.userId && memberIds.has(event.userId))
    const buckets = Array.from({ length: 7 }, () => Array(6).fill(0) as number[])
    for (const event of relevant) {
      const date = new Date(event.observedAt)
      if (Number.isNaN(date.getTime())) continue
      buckets[date.getDay()][Math.floor(date.getHours() / 4)] += 1
    }
    let peakDay = 0
    let peakBucket = 0
    for (let day = 0; day < 7; day += 1) for (let bucket = 0; bucket < 6; bucket += 1) {
      if (buckets[day][bucket] > buckets[peakDay][peakBucket]) { peakDay = day; peakBucket = bucket }
    }
    const theme = selectCommunityTheme(nodes, network.edges)
    result.push({
      id,
      name: `${theme?.node.displayName ?? `朋友圈 ${id + 1}`}的朋友圈`,
      memberIds: [...memberIds],
      eventCount: relevant.length,
      coverageDays: new Set(relevant.map((event) => event.observedAt.slice(0, 10))).size,
      peak: relevant.length ? `${weekdayNames[peakDay]} ${String(peakBucket * 4).padStart(2, '0')}:00–${String(peakBucket * 4 + 4).padStart(2, '0')}:00` : '尚无活动记录',
      buckets,
    })
  }
  return result.sort((left, right) => right.eventCount - left.eventCount || right.memberIds.length - left.memberIds.length)
}

export function buildWorldPassports(events: ActivityEvent[], worlds: World[]): WorldPassportEntry[] {
  const known = new Map(worlds.map((world) => [world.id, world]))
  const grouped = new Map<string, { events: ActivityEvent[]; companions: Set<string> }>()
  for (const event of events) {
    if (!event.worldId) continue
    const value = grouped.get(event.worldId) ?? { events: [], companions: new Set<string>() }
    value.events.push(event)
    if (event.userId) value.companions.add(event.userId)
    grouped.set(event.worldId, value)
  }
  return [...grouped].map(([worldId, value]) => {
    const ordered = [...value.events].sort((left, right) => left.observedAt.localeCompare(right.observedAt))
    const world = known.get(worldId)
    return {
      worldId,
      name: world?.name || '未缓存世界',
      imageUrl: world?.thumbnailImageUrl || world?.imageUrl,
      firstSeenAt: ordered[0].observedAt,
      lastSeenAt: ordered.at(-1)!.observedAt,
      eventCount: ordered.length,
      companionIds: [...value.companions],
    }
  }).sort((left, right) => right.lastSeenAt.localeCompare(left.lastSeenAt))
}

export function buildReadinessChecks(
  diagnostics: Diagnostics | null,
  realtime: RealtimeStatus,
  route: NetworkState,
  cache: CacheStats | null,
  selectedWorldId = '',
  passports: WorldPassportEntry[] = [],
): ReadinessCheck[] {
  const rest = diagnostics?.checks.find((check) => /api|vrchat|rest/i.test(check.name))
  const hasWorldEvidence = !selectedWorldId || passports.some((entry) => entry.worldId === selectedWorldId)
  return [
    { id: 'database', label: '本机数据库', state: diagnostics?.database.ready ? 'ok' : diagnostics ? 'error' : 'unknown', detail: diagnostics?.database.ready ? '可读取历史、计划与缓存索引' : '尚未确认本机数据库状态' },
    { id: 'rest', label: 'VRChat REST', state: rest?.state === 'ok' ? 'ok' : rest?.state === 'error' ? 'error' : rest ? 'warn' : 'unknown', detail: rest?.detail || '等待线路诊断结果' },
    { id: 'pipeline', label: '实时事件', state: realtime.state === 'connected' ? 'ok' : realtime.state === 'connecting' ? 'warn' : realtime.state === 'disabled' ? 'unknown' : 'error', detail: realtime.state === 'connected' ? 'Pipeline 已连接' : realtime.message || '当前没有实时事件连接' },
    { id: 'route', label: '中国网络线路', state: route.mode === 'direct' ? 'warn' : 'ok', detail: route.mode === 'direct' ? '直连可用性取决于当前网络；遇阻时可切换系统或自有代理' : `${route.label}：${route.description}` },
    { id: 'media', label: '图片离线余量', state: cache ? (cache.mediaFiles > 0 ? 'ok' : 'warn') : 'unknown', detail: cache ? `本机已有 ${cache.mediaFiles} 个媒体文件` : '尚未读取缓存统计' },
    { id: 'world', label: '目标世界线索', state: hasWorldEvidence ? 'ok' : 'warn', detail: !selectedWorldId ? '尚未在活动计划中指定世界' : hasWorldEvidence ? '本机历史中已有该世界记录' : '本机没有该世界历史；起航前建议打开详情确认' },
  ]
}

export function readinessScore(checks: ReadinessCheck[]) {
  const known = checks.filter((check) => check.state !== 'unknown')
  if (!known.length) return 0
  return Math.round(known.reduce((sum, check) => sum + (check.state === 'ok' ? 1 : check.state === 'warn' ? .5 : 0), 0) / known.length * 100)
}
