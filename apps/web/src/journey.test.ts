import { describe, expect, it } from 'vitest'
import type { ActivityEvent, FriendNetwork, World } from './api'
import { buildFriendMotionOverview, buildReadinessChecks, buildSocialTides, buildWorldPassports, readinessScore } from './journey'

const network: FriendNetwork = {
  nodes: [
    { id: 'a', displayName: 'Alpha', online: true, scanned: true, optedOut: false },
    { id: 'b', displayName: 'Beta', online: true, scanned: true, optedOut: false },
    { id: 'c', displayName: 'Gamma', online: false, scanned: true, optedOut: false },
  ], edges: [{ source: 'a', target: 'b' }], totalFriends: 3, scannedCount: 3, optedOutCount: 0, generatedAt: '2026-08-15T00:00:00Z',
}
const events: ActivityEvent[] = [
  { id: '1', type: 'join', userId: 'a', worldId: 'wrld_a', summary: 'join', observedAt: '2026-08-14T12:00:00Z' },
  { id: '2', type: 'join', userId: 'b', worldId: 'wrld_a', summary: 'join', observedAt: '2026-08-15T12:00:00Z' },
]
const worlds: World[] = [{ id: 'wrld_a', name: 'Cafe' }]

describe('journey workspace', () => {
  it('builds evidence-backed community tides and excludes isolated nodes', () => {
    const tides = buildSocialTides(network, events)
    expect(tides).toHaveLength(1)
    expect(tides[0].memberIds).toEqual(expect.arrayContaining(['a', 'b']))
    expect(tides[0].coverageDays).toBe(2)
  })
  it('builds a world passport with companions', () => {
    const passports = buildWorldPassports(events, worlds)
    expect(passports[0]).toMatchObject({ worldId: 'wrld_a', name: 'Cafe', eventCount: 2 })
    expect(passports[0].companionIds).toEqual(['a', 'b'])
  })
  it('keeps unknown checks out of the denominator', () => {
    const checks = buildReadinessChecks(null, { state: 'disabled', reconnects: 0 }, { mode: 'system', label: '系统', description: '系统线路' }, null)
    expect(readinessScore(checks)).toBe(100)
  })
  it('merges friend activity into scenes and exposes community intersections', () => {
    const intertwined: FriendNetwork = {
      nodes: ['a', 'b', 'c', 'd', 'e', 'f'].map((id) => ({ id, displayName: id.toUpperCase(), online: true, scanned: true, optedOut: false })),
      edges: [
        { source: 'a', target: 'b' }, { source: 'a', target: 'c' }, { source: 'b', target: 'c' },
        { source: 'd', target: 'e' }, { source: 'd', target: 'f' }, { source: 'e', target: 'f' },
        { source: 'c', target: 'd' },
      ],
      totalFriends: 6, scannedCount: 6, optedOutCount: 0, generatedAt: '2026-08-15T00:00:00Z',
    }
    const motion = buildFriendMotionOverview(
      [
        { id: 'c', displayName: 'C', online: true, location: 'wrld_a:1' },
        { id: 'd', displayName: 'D', online: true, location: 'wrld_a:1' },
        { id: 'e', displayName: 'E', online: true, location: 'private' },
        { id: 'f', displayName: 'F', online: false },
      ],
      worlds,
      [
        { id: 'm1', type: 'friend-location', userId: 'c', worldId: 'wrld_a', summary: 'move', observedAt: '2026-08-15T12:01:00Z' },
        { id: 'm2', type: 'friend-location', userId: 'd', worldId: 'wrld_a', summary: 'move', observedAt: '2026-08-15T12:08:00Z' },
      ],
      intertwined,
    )
    expect(motion).toMatchObject({ onlineCount: 3, visibleCount: 2, privateCount: 1, offlineCount: 1 })
    expect(motion.liveScenes).toHaveLength(1)
    expect(motion.liveScenes[0]).toMatchObject({ title: 'Cafe', userIds: ['c', 'd'], crossEdges: 1 })
    expect(motion.recentScenes).toHaveLength(1)
    expect(motion.recentScenes[0].eventCount).toBe(2)
    expect(motion.intersections[0]).toMatchObject({ sceneCount: 2 })
    expect(motion.intersections[0].bridgeIds).toEqual(expect.arrayContaining(['c', 'd']))
  })
})
