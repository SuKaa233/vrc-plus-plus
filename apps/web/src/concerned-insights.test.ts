import { describe, expect, it } from 'vitest'
import { buildConcernedInferences } from './concerned-insights'

describe('buildConcernedInferences', () => {
  it('keeps relation changes and private-instance analysis evidence based', () => {
    const result = buildConcernedInferences({
      insights: {
        userId: 'usr_target', totalEvents: 12, coverageDays: 8, togetherMinutes: 0, togetherSessions: 0,
        distinctWorlds: 2, sourceCounts: {}, locationKinds: { private: 8, public: 2 }, privateVisits: 4,
        activeHours: [{ hour: 22, count: 6 }], commonWorlds: [], timeline: [], generatedAt: '2026-08-16T00:00:00Z',
        relationChanges: [
          { peerId: 'usr_a', state: 'newly_observed', observedAt: '2026-08-16T00:00:00Z' },
          { peerId: 'usr_b', state: 'not_observed', observedAt: '2026-08-15T00:00:00Z' },
        ],
      },
      events: [], relationPeerCount: 8, allNetworkDegrees: [1, 2, 3, 5, 8], coObservedCount: 0,
      coObservedRelationOverlap: 0, visitedWorlds: [{ count: 5, privateCount: 3 }, { count: 2, privateCount: 1 }],
    })
    expect(result.find((item) => item.id === 'relation-trend')?.summary).toContain('不等同于确认')
    expect(result.find((item) => item.id === 'instance-style')?.title).toContain('私人空间')
    expect(result.find((item) => item.id === 'network-role')?.evidence).toContain('8 条当前连线')
  })

  it('returns no invented cards when there is no evidence', () => {
    expect(buildConcernedInferences({ insights: null, events: [], relationPeerCount: 0, allNetworkDegrees: [], coObservedCount: 0, coObservedRelationOverlap: 0, visitedWorlds: [] })).toEqual([])
  })
})
