import { describe, expect, it } from 'vitest'
import type { ActivityEvent, Friend, FriendNetwork, World } from './api'
import { buildCoverageMap, buildDailyBrief, buildFriendReplay, buildMovementChains, buildWorldMemories, compareNetworkSnapshots, createNetworkSnapshot } from './product-insights'

const now = new Date('2026-08-16T12:00:00Z')
const friends: Friend[] = [
  { id: 'usr_a', displayName: 'Alpha', online: true, location: 'wrld_a:1' },
  { id: 'usr_b', displayName: 'Beta', online: true, location: 'wrld_a:1' },
]
const worlds: World[] = [{ id: 'wrld_a', name: 'Cafe' }, { id: 'wrld_b', name: 'Gallery' }]
const events: ActivityEvent[] = [
  { id: '1', type: 'game.player-joined', userId: 'usr_a', worldId: 'wrld_a', summary: 'Alpha 加入', observedAt: '2026-08-16T10:00:00Z' },
  { id: '2', type: 'game.location', userId: 'usr_a', worldId: 'wrld_b', summary: 'Alpha 移动', observedAt: '2026-08-16T10:20:00Z' },
  { id: '3', type: 'vrc.friend-location', userId: 'usr_b', worldId: 'wrld_a', summary: 'Beta 加入', observedAt: '2026-08-16T10:02:00Z' },
  { id: '4', type: 'vrc.friend-location', userId: 'usr_b', worldId: 'wrld_b', summary: 'Beta 移动', observedAt: '2026-08-16T10:25:00Z' },
]
const network: FriendNetwork = { nodes: friends.map((friend) => ({ ...friend, scanned: true, optedOut: false })), edges: [{ source: 'usr_a', target: 'usr_b' }], totalFriends: 2, scannedCount: 2, optedOutCount: 0, generatedAt: now.toISOString() }

describe('product insights', () => {
  it('limits the daily brief and prioritizes a current gathering', () => {
    const result = buildDailyBrief(friends, worlds, [], events, network, now)
    expect(result.length).toBeLessThanOrEqual(5)
    expect(result[0]).toMatchObject({ kind: 'gathering', worldId: 'wrld_a' })
    expect(result[0].evidence.length).toBeGreaterThan(0)
    expect(result[0].confidence).toBe('高')
  })
  it('filters a friend replay by period and preserves sources', () => {
    expect(buildFriendReplay(events, 'usr_a', worlds, 1, now)).toEqual(expect.arrayContaining([expect.objectContaining({ detail: 'Gallery', source: 'gameLog' })]))
  })
  it('detects only shared world transitions', () => {
    expect(buildMovementChains(events, 1, now)[0]).toMatchObject({ fromWorldId: 'wrld_a', toWorldId: 'wrld_b', userIds: expect.arrayContaining(['usr_a', 'usr_b']) })
  })
  it('builds coverage and world memory without calling event counts visits', () => {
    expect(buildCoverageMap(friends, events, network).find(item=>item.userId==='usr_b')).toMatchObject({ scanned:true, observationDays:1, pipelineEvents:2, gameLogEvents:0 })
    expect(buildWorldMemories(events, worlds)[0]).toMatchObject({ name: 'Gallery', visitDays: 1, eventCount: 2 })
  })
  it('compares local network snapshots without claiming missing edges were removed upstream', () => {
    const previous = createNetworkSnapshot({ ...network, edges: [] }, '2026-08-15')
    const current = createNetworkSnapshot(network, '2026-08-16')
    expect(compareNetworkSnapshots(previous, current)).toMatchObject({ addedEdges: 1, missingEdges: 0 })
  })
})
