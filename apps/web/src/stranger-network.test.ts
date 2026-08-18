import { describe, expect, it } from 'vitest'
import { buildStrangerGraph, relationshipInsights } from './stranger-network'

const input = {
  self:{ id:'usr_me', displayName:'Me' },
  target:{ id:'usr_target', displayName:'Target', isFriend:false, allowAvatarCopying:false, trustLevel:'user' as const, mutualFriendCount:2, mutualGroupCount:2, profileSources:['user','mutuals'] },
  mutuals:[{ id:'usr_bridge', displayName:'Bridge' }],
  groups:[{ id:'grp_a', name:'A', isRepresenting:false },{ id:'grp_b', name:'B', isRepresenting:false }],
  members:[{ userId:'usr_candidate', groupId:'grp_a', displayName:'Candidate', isRepresenting:false },{ userId:'usr_candidate', groupId:'grp_b', displayName:'Candidate', isRepresenting:false },{ userId:'usr_bridge', groupId:'grp_a', displayName:'Bridge', isRepresenting:false }],
  events:[
    { id:'evt_target', type:'presence', summary:'Target joined', observedAt:'2026-08-19T01:00:00Z', userId:'usr_target', location:'wrld_demo:1' },
    { id:'evt_bridge', type:'presence', summary:'Bridge joined', observedAt:'2026-08-19T01:10:00Z', userId:'usr_bridge', location:'wrld_demo:1' },
  ],
}

describe('stranger relationship graph', () => {
  it('creates confirmed mutual and group paths while deduplicating candidates', () => {
    const graph = buildStrangerGraph(input)
    expect(graph.nodes.filter(item=>item.id==='usr_candidate')).toHaveLength(1)
    expect(graph.edges.filter(item=>item.target==='usr_candidate')).toHaveLength(2)
    expect(graph.edges.some(item=>item.source==='usr_bridge' && item.target==='usr_target')).toBe(true)
    expect(graph.edges.some(item=>item.kind==='co_presence' && item.target==='usr_bridge')).toBe(true)
  })
  it('keeps inferences evidence-labelled and includes coverage caveat', () => {
    const items = relationshipInsights(input)
    expect(items.some(item=>item.id==='mutual-route' && item.confidence==='中')).toBe(true)
    expect(items.some(item=>item.id==='friend-slice')).toBe(true)
    expect(items.some(item=>item.id==='mutual-cross-source' && item.confidence==='高')).toBe(true)
    expect(items.some(item=>item.id==='mutual-co-presence')).toBe(true)
    expect(items.some(item=>item.id==='network-shape')).toBe(true)
    expect(items.some(item=>item.id==='circle-concentration')).toBe(true)
    expect(items.some(item=>item.id==='social-visibility')).toBe(true)
    expect(items.at(-1)?.id).toBe('coverage')
  })
})
