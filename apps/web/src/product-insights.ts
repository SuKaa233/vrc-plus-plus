import type { ActivityEvent, Friend, FriendNetwork, VrcNotification, World } from './api'
import { buildFriendMotionOverview } from './journey'

export type DailyBriefKind = 'request' | 'gathering' | 'intersection' | 'reunion' | 'memory'
export interface DailyBriefItem {
  id: string
  kind: DailyBriefKind
  title: string
  summary: string
  score: number
  userIds: string[]
  worldId?: string
  observedAt?: string
}

export interface ReplayEntry {
  id: string
  title: string
  detail: string
  observedAt: string
  worldId?: string
  source: 'gameLog' | 'pipeline'
}

export interface MovementChain {
  id: string
  fromWorldId: string
  toWorldId: string
  userIds: string[]
  observedAt: string
}

export interface IntersectionChange {
  id: string
  title: string
  current: number
  previous: number
  delta: number
  userIds: string[]
}

export interface FriendCoverage {
  userId: string
  eventCount: number
  lastObservedAt?: string
  scanned: boolean
  visibleNow: boolean
  level: '充分' | '有限' | '稀少'
}

export interface WorldMemoryEntry {
  worldId: string
  name: string
  firstSeenAt: string
  lastSeenAt: string
  eventCount: number
  visitDays: number
  companionIds: string[]
}

export interface NetworkSnapshot {
  date: string
  nodeIds: string[]
  edges: string[]
  scannedCount: number
}

export interface NetworkDelta {
  addedNodes: number
  removedNodes: number
  addedEdges: number
  missingEdges: number
  scannedDelta: number
}

function time(value: string) {
  const parsed = new Date(value).getTime()
  return Number.isNaN(parsed) ? 0 : parsed
}

function worldIdFromLocation(location?: string) {
  return location?.startsWith('wrld_') ? location.split(':')[0] : ''
}

export function buildDailyBrief(
  friends: Friend[],
  worlds: World[],
  notifications: VrcNotification[],
  events: ActivityEvent[],
  network: FriendNetwork | null,
  now = new Date(),
): DailyBriefItem[] {
  const candidates: DailyBriefItem[] = []
  const unread = notifications.filter((item) => !item.seen)
  if (unread.length) candidates.push({
    id: 'requests', kind: 'request', title: `${unread.length} 条通知等待处理`,
    summary: '逐项查看后再决定是否接受、隐藏或标记已读。', score: 120 + unread.length, userIds: unread.map((item) => item.senderUserId).filter((value): value is string => !!value),
  })

  const worldById = new Map(worlds.map((world) => [world.id, world]))
  const gatherings = new Map<string, Friend[]>()
  friends.filter((friend) => friend.online).forEach((friend) => {
    const worldId = worldIdFromLocation(friend.location)
    if (!worldId) return
    const value = gatherings.get(worldId) ?? []
    value.push(friend)
    gatherings.set(worldId, value)
  })
  for (const [worldId, members] of gatherings) {
    if (members.length < 2) continue
    candidates.push({
      id: `gathering:${worldId}`, kind: 'gathering', title: `${members.length} 位好友正在同一世界`,
      summary: `${worldById.get(worldId)?.name || '世界资料待补'} · ${members.slice(0, 3).map((item) => item.displayName).join('、')}`,
      score: 110 + members.length * 4, userIds: members.map((item) => item.id), worldId,
    })
  }

  const motion = buildFriendMotionOverview(friends, worlds, events, network)
  const intersection = motion.liveScenes.find((scene) => scene.communityIds.length > 1)
  if (intersection) candidates.push({
    id: `intersection:${intersection.id}`, kind: 'intersection', title: `${intersection.communityIds.length} 个朋友圈正在交汇`,
    summary: `${intersection.title} · ${intersection.userIds.length} 位好友 · ${intersection.crossEdges} 条已观察跨圈连线`,
    score: 105 + intersection.communityIds.length * 5, userIds: intersection.userIds, worldId: intersection.worldId,
  })

  const lastByUser = new Map<string, ActivityEvent>()
  for (const event of events) if (event.userId && (!lastByUser.has(event.userId) || time(event.observedAt) > time(lastByUser.get(event.userId)!.observedAt))) lastByUser.set(event.userId, event)
  const reunionThreshold = now.getTime() - 14 * 86400000
  for (const friend of friends) {
    const previous = lastByUser.get(friend.id)
    if (!friend.online || !previous || time(previous.observedAt) >= reunionThreshold) continue
    const days = Math.floor((now.getTime() - time(previous.observedAt)) / 86400000)
    candidates.push({
      id: `reunion:${friend.id}`, kind: 'reunion', title: `${friend.displayName} 已经 ${days} 天没在本机记录中出现`,
      summary: '好友当前在线；位置隐私保持原样，打开资料后再决定是否联系。', score: 80 + Math.min(days, 60), userIds: [friend.id], observedAt: previous.observedAt,
    })
  }

  const recentCutoff = now.getTime() - 86400000
  const recentScene = motion.recentScenes.find((scene) => time(scene.observedAt) >= recentCutoff && scene.userIds.length >= 2)
  if (recentScene) candidates.push({
    id: `memory:${recentScene.id}`, kind: 'memory', title: '最近出现了一次多人同场',
    summary: `${recentScene.title} · ${recentScene.userIds.length} 位好友 · ${recentScene.eventCount} 条本机观测`,
    score: 70 + recentScene.userIds.length, userIds: recentScene.userIds, worldId: recentScene.worldId, observedAt: recentScene.observedAt,
  })
  return candidates.sort((left, right) => right.score - left.score || left.id.localeCompare(right.id)).slice(0, 5)
}

export function buildFriendReplay(events: ActivityEvent[], userId: string, worlds: World[], days: 1 | 7 | 30, now = new Date()): ReplayEntry[] {
  const cutoff = now.getTime() - days * 86400000
  const names = new Map(worlds.map((world) => [world.id, world.name]))
  return events.filter((event) => event.userId === userId && time(event.observedAt) >= cutoff)
    .sort((left, right) => right.observedAt.localeCompare(left.observedAt))
    .map((event) => ({
      id: event.id,
      title: event.summary,
      detail: event.worldId ? names.get(event.worldId) || event.worldId : event.type.includes('offline') || event.type.includes('left') ? '离开或离线' : '状态变化',
      observedAt: event.observedAt,
      worldId: event.worldId,
      source: event.type.startsWith('game.') ? 'gameLog' : 'pipeline',
    }))
}

function intersectionsFor(events: ActivityEvent[], worlds: World[], network: FriendNetwork | null) {
  return new Map(buildFriendMotionOverview([], worlds, events, network).intersections.map((item) => [item.id, item]))
}

export function buildIntersectionChanges(events: ActivityEvent[], worlds: World[], network: FriendNetwork | null, days: 1 | 7 | 30, now = new Date()): IntersectionChange[] {
  const width = days * 86400000
  const end = now.getTime()
  const current = intersectionsFor(events.filter((event) => { const value = time(event.observedAt); return value >= end - width && value <= end }), worlds, network)
  const previous = intersectionsFor(events.filter((event) => { const value = time(event.observedAt); return value >= end - width * 2 && value < end - width }), worlds, network)
  const ids = new Set([...current.keys(), ...previous.keys()])
  return [...ids].map((id) => {
    const next = current.get(id)
    const before = previous.get(id)
    const currentCount = next?.sceneCount ?? 0
    const previousCount = before?.sceneCount ?? 0
    return {
      id,
      title: next ? `${next.leftName} × ${next.rightName}` : before ? `${before.leftName} × ${before.rightName}` : id,
      current: currentCount,
      previous: previousCount,
      delta: currentCount - previousCount,
      userIds: next?.userIds ?? before?.userIds ?? [],
    }
  }).filter((item) => item.delta !== 0).sort((left, right) => Math.abs(right.delta) - Math.abs(left.delta)).slice(0, 6)
}

export function buildMovementChains(events: ActivityEvent[], days: 1 | 7 | 30, now = new Date()): MovementChain[] {
  const cutoff = now.getTime() - days * 86400000
  const byUser = new Map<string, ActivityEvent[]>()
  events.filter((event) => event.userId && event.worldId && time(event.observedAt) >= cutoff).forEach((event) => {
    const value = byUser.get(event.userId!) ?? []
    value.push(event)
    byUser.set(event.userId!, value)
  })
  const groups = new Map<string, { fromWorldId: string; toWorldId: string; userIds: Set<string>; observedAt: string }>()
  for (const [userId, values] of byUser) {
    const ordered = values.sort((left, right) => left.observedAt.localeCompare(right.observedAt))
    for (let index = 1; index < ordered.length; index += 1) {
      const before = ordered[index - 1]
      const after = ordered[index]
      const gap = time(after.observedAt) - time(before.observedAt)
      if (before.worldId === after.worldId || gap <= 0 || gap > 90 * 60000) continue
      const bucket = Math.floor(time(after.observedAt) / (30 * 60000))
      const id = `${before.worldId}:${after.worldId}:${bucket}`
      const group = groups.get(id) ?? { fromWorldId: before.worldId!, toWorldId: after.worldId!, userIds: new Set<string>(), observedAt: after.observedAt }
      group.userIds.add(userId)
      if (after.observedAt > group.observedAt) group.observedAt = after.observedAt
      groups.set(id, group)
    }
  }
  return [...groups].filter(([, value]) => value.userIds.size >= 2).map(([id, value]) => ({ ...value, id, userIds: [...value.userIds] }))
    .sort((left, right) => right.userIds.length - left.userIds.length || right.observedAt.localeCompare(left.observedAt)).slice(0, 6)
}

export function buildCoverageMap(friends: Friend[], events: ActivityEvent[], network: FriendNetwork | null): FriendCoverage[] {
  const observations = new Map<string, { count: number; last: string }>()
  for (const event of events) if (event.userId) {
    const value = observations.get(event.userId) ?? { count: 0, last: '' }
    value.count += 1
    if (event.observedAt > value.last) value.last = event.observedAt
    observations.set(event.userId, value)
  }
  const scanned = new Set((network?.nodes ?? []).filter((node) => node.scanned).map((node) => node.id))
  const levelOrder = { '稀少': 0, '有限': 1, '充分': 2 } as const
  return friends.map((friend) => {
    const value = observations.get(friend.id)
    const eventCount = value?.count ?? 0
    const evidence = eventCount + (scanned.has(friend.id) ? 3 : 0) + (worldIdFromLocation(friend.location) ? 2 : 0)
    const level: FriendCoverage['level'] = evidence >= 12 ? '充分' : evidence >= 4 ? '有限' : '稀少'
    return { userId: friend.id, eventCount, lastObservedAt: value?.last, scanned: scanned.has(friend.id), visibleNow: !!worldIdFromLocation(friend.location), level }
  }).sort((left, right) => levelOrder[left.level] - levelOrder[right.level] || right.eventCount - left.eventCount)
}

export function buildWorldMemories(events: ActivityEvent[], worlds: World[]): WorldMemoryEntry[] {
  const names = new Map(worlds.map((world) => [world.id, world.name]))
  const grouped = new Map<string, { events: ActivityEvent[]; companions: Set<string>; days: Set<string> }>()
  for (const event of events) {
    if (!event.worldId) continue
    const value = grouped.get(event.worldId) ?? { events: [], companions: new Set<string>(), days: new Set<string>() }
    value.events.push(event)
    if (event.userId) value.companions.add(event.userId)
    value.days.add(event.observedAt.slice(0, 10))
    grouped.set(event.worldId, value)
  }
  return [...grouped].map(([worldId, value]) => {
    const ordered = value.events.sort((left, right) => left.observedAt.localeCompare(right.observedAt))
    return { worldId, name: names.get(worldId) || worldId, firstSeenAt: ordered[0].observedAt, lastSeenAt: ordered.at(-1)!.observedAt, eventCount: ordered.length, visitDays: value.days.size, companionIds: [...value.companions] }
  }).sort((left, right) => right.lastSeenAt.localeCompare(left.lastSeenAt))
}

export function createNetworkSnapshot(network: FriendNetwork, date = new Date().toISOString().slice(0, 10)): NetworkSnapshot {
  const edges = network.edges.map((edge) => [edge.source, edge.target].sort().join('|')).sort()
  return { date, nodeIds: network.nodes.map((node) => node.id).sort(), edges: [...new Set(edges)], scannedCount: network.scannedCount }
}

export function compareNetworkSnapshots(previous: NetworkSnapshot, current: NetworkSnapshot): NetworkDelta {
  const beforeNodes = new Set(previous.nodeIds)
  const nextNodes = new Set(current.nodeIds)
  const beforeEdges = new Set(previous.edges)
  const nextEdges = new Set(current.edges)
  return {
    addedNodes: current.nodeIds.filter((id) => !beforeNodes.has(id)).length,
    removedNodes: previous.nodeIds.filter((id) => !nextNodes.has(id)).length,
    addedEdges: current.edges.filter((id) => !beforeEdges.has(id)).length,
    missingEdges: previous.edges.filter((id) => !nextEdges.has(id)).length,
    scannedDelta: current.scannedCount - previous.scannedCount,
  }
}
