import { describe, expect, it } from 'vitest'
import { applyManualNetworkPositions, buildNetworkFocusIndex, compareCommunityMembers, compareNetworkSnapshots, detectNetworkCommunities, findShortestNetworkPath, layoutFriendNetwork, rankBridgeNodes, resolveNodeCollisions, selectCommunityTheme, summarizeNetworkDelta, toggleElementFullscreen, zoomAroundPoint } from './friend-network'

describe('layoutFriendNetwork', () => {
  it('lays out nodes deterministically and calculates graph metadata', () => {
    const nodes = ['a', 'b', 'c'].map((id) => ({ id, displayName: id, online: id === 'a', scanned: true, optedOut: false }))
    const edges = [{ source: 'a', target: 'b' }, { source: 'missing', target: 'a' }]
    const first = layoutFriendNetwork(nodes, edges)
    const second = layoutFriendNetwork(nodes, edges)
    expect(first).toEqual(second)
    expect(first.every((node) => Number.isFinite(node.x) && Number.isFinite(node.y))).toBe(true)
    expect(first.find((node) => node.id === 'a')?.degree).toBe(1)
    expect(first.find((node) => node.id === 'c')?.degree).toBe(0)
  })

  it('keeps the pointer anchor stable while zooming', () => {
    const result = zoomAroundPoint(1, 2, 0, 0, 250, 120)
    expect(result).toEqual({ zoom: 2, panX: -250, panY: -120 })
    expect((250 - result.panX) / result.zoom).toBe(250)
    expect((120 - result.panY) / result.zoom).toBe(120)
  })

  it('separates overlapping avatars with a visible gap', () => {
    const crowded = Array.from({ length: 60 }, (_, index) => ({ id: `node-${index}`, x: 500, y: 310, radius: 8 }))
    const settled = resolveNodeCollisions(crowded, 1000, 620, 7, 50)
    let smallestGap = Number.POSITIVE_INFINITY
    for (let left = 0; left < settled.length; left += 1) {
      for (let right = left + 1; right < settled.length; right += 1) {
        const a = settled[left]
        const b = settled[right]
        const distance = Math.hypot(a.x - b.x, a.y - b.y)
        smallestGap = Math.min(smallestGap, distance - a.radius - b.radius)
      }
    }
    expect(smallestGap).toBeGreaterThan(6.5)
  })

  it('keeps saved manual positions without allowing them to cover automatic nodes', () => {
    const nodes = [
      { id: 'automatic', x: 500, y: 300, radius: 16 },
      { id: 'manual', x: 800, y: 500, radius: 16 },
    ]
    const result = applyManualNetworkPositions(nodes, { manual: { x: 500, y: 300 } }, 1000, 620, 20)
    const automatic = result.find((node) => node.id === 'automatic')!
    const manual = result.find((node) => node.id === 'manual')!
    expect(automatic).toMatchObject({ x: 500, y: 300 })
    expect(Math.hypot(automatic.x - manual.x, automatic.y - manual.y)).toBeGreaterThanOrEqual(51.5)
  })

  it('lays out a 300-node, 2000-edge graph without overlapping avatars', () => {
    const nodes = Array.from({ length: 300 }, (_, index) => ({
      id: `friend-${index}`,
      displayName: `Friend ${index}`,
      online: index % 7 === 0,
      scanned: true,
      optedOut: false,
    }))
    const edgeKeys = new Set<string>()
    for (let index = 0; edgeKeys.size < 2000; index += 1) {
      const source = index % nodes.length
      const target = (source + Math.floor(index / nodes.length) + 1) % nodes.length
      if (source !== target) edgeKeys.add(`${Math.min(source, target)}:${Math.max(source, target)}`)
    }
    const edges = [...edgeKeys].map((key) => {
      const [source, target] = key.split(':')
      return { source: `friend-${source}`, target: `friend-${target}` }
    })
    const started = performance.now()
    const result = layoutFriendNetwork(nodes, edges, 1800, 1120)
    expect(result).toHaveLength(300)
    expect(performance.now() - started).toBeLessThan(3000)
    for (let left = 0; left < result.length; left += 1) {
      for (let right = left + 1; right < result.length; right += 1) {
        const a = result[left]
        const b = result[right]
        expect(Math.hypot(a.x - b.x, a.y - b.y)).toBeGreaterThanOrEqual(a.radius + b.radius + 6)
      }
    }
  })

  it('reports the actual nodes and edges added by a scan', () => {
    const node = (id: string, scanned: boolean) => ({ id, displayName: id, online: false, scanned, optedOut: false })
    const result = summarizeNetworkDelta(
      { nodes: [node('a', true), node('b', false)], edges: [] },
      { nodes: [node('a', true), node('b', true), node('c', true)], edges: [{ source: 'a', target: 'b' }, { source: 'b', target: 'c' }] },
    )
    expect(result).toEqual({ addedScanned: 2, addedConnected: 3, addedEdges: 2 })
  })

  it('indexes hover neighbors and incident edges without rescanning the graph', () => {
    const index = buildNetworkFocusIndex([
      { source: 'a', target: 'b' },
      { source: 'a', target: 'c' },
      { source: 'd', target: 'c' },
    ])
    expect([...index.get('a')!.neighbors]).toEqual(['b', 'c'])
    expect(index.get('a')!.edgeKeys).toEqual(['a|b', 'a|c'])
    expect(index.get('d')!.edgeKeys).toEqual(['c|d'])
  })

  it('detects stable local communities separated by a sparse bridge', () => {
    const nodes = ['a', 'b', 'c', 'd', 'e', 'f'].map((id) => ({ id }))
    const edges = [
      { source: 'a', target: 'b' }, { source: 'a', target: 'c' }, { source: 'b', target: 'c' },
      { source: 'd', target: 'e' }, { source: 'd', target: 'f' }, { source: 'e', target: 'f' },
      { source: 'c', target: 'd' },
    ]
    const communities = detectNetworkCommunities(nodes, edges)
    expect(communities.get('a')).toBe(communities.get('c'))
    expect(communities.get('d')).toBe(communities.get('f'))
    expect(communities.get('a')).not.toBe(communities.get('d'))
  })

  it('uses the friend with the most links inside a community as its theme', () => {
    const nodes = ['a', 'b', 'c', 'd'].map((id) => ({ id, displayName: id.toUpperCase(), online: false, scanned: true, optedOut: false }))
    const result = selectCommunityTheme(nodes, [
      { source: 'a', target: 'b' }, { source: 'b', target: 'c' }, { source: 'b', target: 'd' },
      { source: 'a', target: 'outside' },
    ])
    expect(result?.node.id).toBe('b')
    expect(result?.degree).toBe(3)
    expect(result?.edgeCount).toBe(3)
  })

  it('finds the shortest observed friendship path', () => {
    const path = findShortestNetworkPath([
      { source: 'a', target: 'b' }, { source: 'b', target: 'c' },
      { source: 'a', target: 'd' }, { source: 'd', target: 'e' }, { source: 'e', target: 'c' },
    ], 'a', 'c')
    expect(path).toEqual(['a', 'b', 'c'])
    expect(findShortestNetworkPath([], 'a', 'z')).toEqual([])
  })

  it('compares local graph snapshots without treating missing edges as a definitive unfriend', () => {
    const result = compareNetworkSnapshots(
      { capturedAt: '2026-08-14T00:00:00Z', nodeIDs: ['a', 'b'], edges: [{ source: 'a', target: 'b' }] },
      { capturedAt: '2026-08-15T00:00:00Z', nodeIDs: ['b', 'c'], edges: [{ source: 'b', target: 'c' }] },
    )
    expect(result).toEqual({ addedNodes: ['c'], removedNodes: ['a'], addedEdges: ['b|c'], removedEdges: ['a|b'] })
  })

  it('ranks friends that bridge multiple local communities and compares circle members', () => {
    const communities = new Map([['a', 0], ['b', 0], ['c', 1], ['d', 2]])
    const ranking = rankBridgeNodes(['a', 'b', 'c', 'd'].map((id) => ({ id })), [
      { source: 'a', target: 'b' }, { source: 'a', target: 'c' }, { source: 'a', target: 'd' }, { source: 'b', target: 'c' },
    ], communities)
    expect(ranking[0]).toMatchObject({ id: 'a', crossEdges: 2, communityCount: 2 })
    expect(compareCommunityMembers(['a', 'b', 'c'], ['b', 'c', 'd'])).toEqual({ overlap: ['b', 'c'], leftOnly: ['a'], rightOnly: ['d'] })
  })

  it('enters and exits fullscreen for the graph shell', async () => {
    let entered = 0
    let exited = 0
    const element = { requestFullscreen: async () => { entered += 1 } }
    await toggleElementFullscreen(element, null, async () => { exited += 1 })
    await toggleElementFullscreen(element, element, async () => { exited += 1 })
    expect({ entered, exited }).toEqual({ entered: 1, exited: 1 })
  })
})
