import type { FriendNetworkEdge, FriendNetworkNode } from './api'

export interface PositionedNetworkNode extends FriendNetworkNode {
  x: number
  y: number
  radius: number
  degree: number
  component: number
}

export interface NetworkFocusEntry {
  neighbors: Set<string>
  edgeKeys: string[]
}

export interface FriendNetworkSnapshot {
  capturedAt: string
  nodeIDs: string[]
  edges: FriendNetworkEdge[]
}

export function networkEdgeKey(edge: FriendNetworkEdge) {
  return [edge.source, edge.target].sort().join('|')
}

/** Build on graph changes so hover work scales with degree instead of total edges. */
export function buildNetworkFocusIndex(edges: FriendNetworkEdge[]) {
  const result = new Map<string, NetworkFocusEntry>()
  const entry = (userID: string) => {
    let value = result.get(userID)
    if (!value) {
      value = { neighbors: new Set<string>(), edgeKeys: [] }
      result.set(userID, value)
    }
    return value
  }
  for (const edge of edges) {
    const key = networkEdgeKey(edge)
    const source = entry(edge.source)
    const target = entry(edge.target)
    source.neighbors.add(edge.target)
    target.neighbors.add(edge.source)
    source.edgeKeys.push(key)
    target.edgeKeys.push(key)
  }
  return result
}

/**
 * Keep the full graph available for search and analysis while selecting a small,
 * deterministic visual backbone for SVG rendering and force layout.
 */
export function selectNetworkRenderEdges(
  edges: FriendNetworkEdge[], maxEdges: number, focusedID = '', requiredKeys = new Set<string>(),
) {
  const unique = new Map<string, FriendNetworkEdge>()
  const degrees = new Map<string, number>()
  for (const edge of edges) {
    if (!edge.source || !edge.target || edge.source === edge.target) continue
    const key = networkEdgeKey(edge)
    if (unique.has(key)) continue
    unique.set(key, edge)
    degrees.set(edge.source, (degrees.get(edge.source) ?? 0) + 1)
    degrees.set(edge.target, (degrees.get(edge.target) ?? 0) + 1)
  }
  const all = [...unique.entries()]
  if (all.length <= maxEdges && !focusedID && !requiredKeys.size) return all.map(([, edge]) => edge)

  const score = ([key, edge]: [string, FriendNetworkEdge]) => ({
    key,
    edge,
    score: (degrees.get(edge.source) ?? 0) + (degrees.get(edge.target) ?? 0),
  })
  const ordered = all.map(score).sort((left, right) => right.score - left.score || left.key.localeCompare(right.key))
  const selected = new Map<string, FriendNetworkEdge>()
  for (const item of ordered) {
    if (requiredKeys.has(item.key) || (focusedID && (item.edge.source === focusedID || item.edge.target === focusedID))) {
      selected.set(item.key, item.edge)
    }
  }

  // Add a maximum spanning forest first so low-degree branches do not disappear.
  const parent = new Map<string, string>()
  const find = (id: string): string => {
    const current = parent.get(id)
    if (!current) { parent.set(id, id); return id }
    if (current === id) return id
    const root = find(current)
    parent.set(id, root)
    return root
  }
  for (const item of ordered) {
    if (selected.size >= maxEdges) break
    const sourceRoot = find(item.edge.source)
    const targetRoot = find(item.edge.target)
    if (sourceRoot === targetRoot) continue
    parent.set(targetRoot, sourceRoot)
    selected.set(item.key, item.edge)
  }
  for (const item of ordered) {
    if (selected.size >= maxEdges) break
    selected.set(item.key, item.edge)
  }
  return [...selected.values()]
}

export function expandNetworkAvatarBudget(current: number, total: number, initial = 72, batch = 16) {
  if (total <= 0) return 0
  return Math.min(total, Math.max(initial, current) + Math.max(1, batch))
}

export function detectNetworkCommunities(nodes: Array<{ id: string }>, edges: FriendNetworkEdge[]) {
  const ids = new Set(nodes.map((node) => node.id))
  const adjacency = new Map<string, string[]>(nodes.map((node) => [node.id, []]))
  for (const edge of edges) {
    if (!ids.has(edge.source) || !ids.has(edge.target) || edge.source === edge.target) continue
    adjacency.get(edge.source)!.push(edge.target)
    adjacency.get(edge.target)!.push(edge.source)
  }
  const community = new Map(nodes.map((node) => [node.id, node.id]))
  const totals = new Map(nodes.map((node) => [node.id, adjacency.get(node.id)!.length]))
  const twiceEdges = Math.max(1, [...adjacency.values()].reduce((sum, neighbors) => sum + neighbors.length, 0))
  const ordered = [...nodes].sort((left, right) => {
    const degreeDiff = adjacency.get(right.id)!.length - adjacency.get(left.id)!.length
    return degreeDiff || left.id.localeCompare(right.id)
  })
  for (let pass = 0; pass < 24; pass += 1) {
    let moved = false
    for (const node of ordered) {
      const degree = adjacency.get(node.id)!.length
      if (!degree) continue
      const current = community.get(node.id)!
      totals.set(current, (totals.get(current) ?? 0) - degree)
      const weights = new Map<string, number>()
      for (const neighbor of adjacency.get(node.id)!) {
        const label = community.get(neighbor)!
        weights.set(label, (weights.get(label) ?? 0) + 1)
      }
      let best = current
      let bestScore = (weights.get(current) ?? 0) - degree * (totals.get(current) ?? 0) / twiceEdges
      for (const [candidate, weight] of weights) {
        const score = weight - degree * (totals.get(candidate) ?? 0) / twiceEdges
        if (score > bestScore + 1e-9 || (Math.abs(score - bestScore) <= 1e-9 && candidate < best)) {
          best = candidate
          bestScore = score
        }
      }
      community.set(node.id, best)
      totals.set(best, (totals.get(best) ?? 0) + degree)
      moved ||= best !== current
    }
    if (!moved) break
  }
  const groups = new Map<string, string[]>()
  for (const node of nodes) {
    const label = community.get(node.id)!
    const members = groups.get(label) ?? []
    members.push(node.id)
    groups.set(label, members)
  }
  const orderedGroups = [...groups.values()].sort((left, right) => right.length - left.length || left[0].localeCompare(right[0]))
  const result = new Map<string, number>()
  orderedGroups.forEach((members, index) => members.forEach((id) => result.set(id, index)))
  return result
}

export function selectCommunityTheme(nodes: FriendNetworkNode[], edges: FriendNetworkEdge[]) {
  if (!nodes.length) return null
  const ids = new Set(nodes.map((node) => node.id))
  const internalDegrees = new Map(nodes.map((node) => [node.id, 0]))
  let edgeCount = 0
  for (const edge of edges) {
    if (edge.source === edge.target || !ids.has(edge.source) || !ids.has(edge.target)) continue
    internalDegrees.set(edge.source, (internalDegrees.get(edge.source) ?? 0) + 1)
    internalDegrees.set(edge.target, (internalDegrees.get(edge.target) ?? 0) + 1)
    edgeCount += 1
  }
  const node = [...nodes].sort((left, right) => {
    const degreeDifference = (internalDegrees.get(right.id) ?? 0) - (internalDegrees.get(left.id) ?? 0)
    if (degreeDifference) return degreeDifference
    const nameDifference = left.displayName.localeCompare(right.displayName)
    return nameDifference || left.id.localeCompare(right.id)
  })[0]
  return { node, degree: internalDegrees.get(node.id) ?? 0, edgeCount, internalDegrees }
}

export function findShortestNetworkPath(edges: FriendNetworkEdge[], start: string, end: string) {
  if (!start || !end) return []
  if (start === end) return [start]
  const adjacency = new Map<string, string[]>()
  for (const edge of edges) {
    const source = adjacency.get(edge.source) ?? []
    const target = adjacency.get(edge.target) ?? []
    source.push(edge.target)
    target.push(edge.source)
    adjacency.set(edge.source, source)
    adjacency.set(edge.target, target)
  }
  const previous = new Map<string, string | null>([[start, null]])
  const queue = [start]
  for (let cursor = 0; cursor < queue.length; cursor += 1) {
    const current = queue[cursor]
    for (const neighbor of adjacency.get(current) ?? []) {
      if (previous.has(neighbor)) continue
      previous.set(neighbor, current)
      if (neighbor === end) {
        const path = [end]
        let step: string | null = current
        while (step) {
          path.push(step)
          step = previous.get(step) ?? null
        }
        return path.reverse()
      }
      queue.push(neighbor)
    }
  }
  return []
}

export function summarizeNetworkDelta(before: { nodes: FriendNetworkNode[]; edges: FriendNetworkEdge[] } | null, after: { nodes: FriendNetworkNode[]; edges: FriendNetworkEdge[] } | null) {
  const beforeScanned = new Set((before?.nodes ?? []).filter((node) => node.scanned).map((node) => node.id))
  const beforeConnected = new Set((before?.edges ?? []).flatMap((edge) => [edge.source, edge.target]))
  const beforeEdges = new Set((before?.edges ?? []).map((edge) => [edge.source, edge.target].sort().join('|')))
  const afterConnected = new Set((after?.edges ?? []).flatMap((edge) => [edge.source, edge.target]))
  return {
    addedScanned: (after?.nodes ?? []).filter((node) => node.scanned && !beforeScanned.has(node.id)).length,
    addedConnected: [...afterConnected].filter((id) => !beforeConnected.has(id)).length,
    addedEdges: (after?.edges ?? []).filter((edge) => !beforeEdges.has([edge.source, edge.target].sort().join('|'))).length,
  }
}

export function compareNetworkSnapshots(before: FriendNetworkSnapshot | null, after: FriendNetworkSnapshot | null) {
  const beforeNodes = new Set(before?.nodeIDs ?? [])
  const afterNodes = new Set(after?.nodeIDs ?? [])
  const beforeEdges = new Set((before?.edges ?? []).map(networkEdgeKey))
  const afterEdges = new Set((after?.edges ?? []).map(networkEdgeKey))
  return {
    addedNodes: [...afterNodes].filter((id) => !beforeNodes.has(id)),
    removedNodes: [...beforeNodes].filter((id) => !afterNodes.has(id)),
    addedEdges: [...afterEdges].filter((key) => !beforeEdges.has(key)),
    removedEdges: [...beforeEdges].filter((key) => !afterEdges.has(key)),
  }
}

export function rankBridgeNodes(nodes: Array<{ id: string }>, edges: FriendNetworkEdge[], communities: Map<string, number>) {
  const scores = new Map(nodes.map((node) => [node.id, { crossEdges: 0, communities: new Set<number>() }]))
  for (const edge of edges) {
    const sourceCommunity = communities.get(edge.source)
    const targetCommunity = communities.get(edge.target)
    if (sourceCommunity === undefined || targetCommunity === undefined || sourceCommunity === targetCommunity) continue
    const source = scores.get(edge.source)
    const target = scores.get(edge.target)
    if (source) { source.crossEdges += 1; source.communities.add(targetCommunity) }
    if (target) { target.crossEdges += 1; target.communities.add(sourceCommunity) }
  }
  return [...scores.entries()].map(([id, score]) => ({ id, crossEdges: score.crossEdges, communityCount: score.communities.size }))
    .filter((item) => item.crossEdges > 0)
    .sort((left, right) => right.crossEdges - left.crossEdges || right.communityCount - left.communityCount || left.id.localeCompare(right.id))
}

export function compareCommunityMembers(left: string[], right: string[]) {
  const leftSet = new Set(left)
  const rightSet = new Set(right)
  const overlap = left.filter((id) => rightSet.has(id))
  return { overlap, leftOnly: left.filter((id) => !rightSet.has(id)), rightOnly: right.filter((id) => !leftSet.has(id)) }
}

export async function toggleElementFullscreen(
  element: { requestFullscreen: () => Promise<void> },
  activeElement: unknown,
  exitFullscreen: () => Promise<void>,
) {
  if (activeElement === element) await exitFullscreen()
  else await element.requestFullscreen()
}

export function zoomAroundPoint(currentZoom: number, nextZoom: number, panX: number, panY: number, anchorX: number, anchorY: number) {
  const graphX = (anchorX - panX) / currentZoom
  const graphY = (anchorY - panY) / currentZoom
  return {
    zoom: nextZoom,
    panX: anchorX - graphX * nextZoom,
    panY: anchorY - graphY * nextZoom,
  }
}

export function resolveNodeCollisions<T extends { id: string; x: number; y: number; radius: number }>(
  nodes: T[], width = 1000, height = 620, gap = 7, iterations = 18, boundary = 18, fixedIDs = new Set<string>(),
) {
  const result = nodes.map((node) => ({ ...node }))
  for (let pass = 0; pass < iterations; pass += 1) {
    let adjusted = false
    const maximumRadius = result.reduce((maximum, node) => Math.max(maximum, node.radius), 1)
    const cellSize = Math.max(1, maximumRadius * 2 + gap)
    const buckets = new Map<string, number[]>()
    result.forEach((node, index) => {
      const cellX = Math.floor(node.x / cellSize)
      const cellY = Math.floor(node.y / cellSize)
      const key = `${cellX}:${cellY}`
      const bucket = buckets.get(key) ?? []
      bucket.push(index)
      buckets.set(key, bucket)
    })
    for (let left = 0; left < result.length; left += 1) {
      const a = result[left]
      const cellX = Math.floor(a.x / cellSize)
      const cellY = Math.floor(a.y / cellSize)
      for (let offsetX = -1; offsetX <= 1; offsetX += 1) {
        for (let offsetY = -1; offsetY <= 1; offsetY += 1) {
          for (const right of buckets.get(`${cellX + offsetX}:${cellY + offsetY}`) ?? []) {
            if (right <= left) continue
            const b = result[right]
            let dx = b.x - a.x
            let dy = b.y - a.y
            let distance = Math.sqrt(dx * dx + dy * dy)
            const minimum = a.radius + b.radius + gap
            if (distance >= minimum) continue
            if (distance < .001) {
              const angle = ((hash(a.id + b.id) % 360) / 180) * Math.PI
              dx = Math.cos(angle)
              dy = Math.sin(angle)
              distance = 1
            }
            const correction = (minimum - distance) * 1.02
            const unitX = dx / distance
            const unitY = dy / distance
            const aFixed = fixedIDs.has(a.id)
            const bFixed = fixedIDs.has(b.id)
            if (aFixed && bFixed) continue
            const aShare = aFixed ? 0 : bFixed ? 1 : .5
            const bShare = bFixed ? 0 : aFixed ? 1 : .5
            a.x -= unitX * correction * aShare
            a.y -= unitY * correction * aShare
            b.x += unitX * correction * bShare
            b.y += unitY * correction * bShare
            adjusted = true
          }
        }
      }
    }
    for (const node of result) {
      node.x = Math.max(boundary + node.radius, Math.min(width - boundary - node.radius, node.x))
      node.y = Math.max(boundary + node.radius, Math.min(height - boundary - node.radius, node.y))
    }
    if (!adjusted) break
  }
  return result
}

export function applyManualNetworkPositions<T extends { id: string; x: number; y: number; radius: number }>(
  nodes: T[], manual: Record<string, { x: number; y: number }>, width: number, height: number, gap = 18,
) {
  const fixedIDs = new Set<string>()
  const merged = nodes.map((node) => {
    const position = manual[node.id]
    if (!position) {
      fixedIDs.add(node.id)
      return { ...node }
    }
    return {
      ...node,
      x: Math.max(node.radius + 18, Math.min(width - node.radius - 18, position.x)),
      y: Math.max(node.radius + 18, Math.min(height - node.radius - 18, position.y)),
    }
  })
  if (fixedIDs.size === nodes.length) return merged
  return resolveNodeCollisions(merged, width, height, gap, nodes.length > 220 ? 12 : 42, 18, fixedIDs)
}

function hash(value: string) {
  let result = 2166136261
  for (let index = 0; index < value.length; index += 1) {
    result ^= value.charCodeAt(index)
    result = Math.imul(result, 16777619)
  }
  return result >>> 0
}

export function seedFriendNetworkLayout(nodes: FriendNetworkNode[], edges: FriendNetworkEdge[], width = 1000, height = 620) {
  if (!nodes.length) return [] as PositionedNetworkNode[]
  const degrees = new Map(nodes.map((node) => [node.id, 0]))
  for (const edge of edges) {
    if (edge.source === edge.target || !degrees.has(edge.source) || !degrees.has(edge.target)) continue
    degrees.set(edge.source, (degrees.get(edge.source) ?? 0) + 1)
    degrees.set(edge.target, (degrees.get(edge.target) ?? 0) + 1)
  }
  const padding = 56
  const centerX = width / 2
  const centerY = height / 2
  const orbitX = (width - padding * 2) * .46
  const orbitY = (height - padding * 2) * .46
  return nodes.map((node, index) => {
    const angle = index * 2.399963229728653 + (hash(node.id) % 1000) / 1800
    const radial = Math.sqrt((index + .65) / nodes.length)
    const degree = degrees.get(node.id) ?? 0
    return {
      ...node,
      x: centerX + Math.cos(angle) * orbitX * radial,
      y: centerY + Math.sin(angle) * orbitY * radial,
      radius: Math.min(19, 8 + Math.sqrt(degree) * 2.6),
      degree,
      component: 0,
    }
  })
}

export function shouldUseNetworkLayoutWorker(nodeCount: number) {
  return nodeCount > 220
}

export function layoutFriendNetwork(nodes: FriendNetworkNode[], edges: FriendNetworkEdge[], width = 1000, height = 620) {
  if (!nodes.length) return [] as PositionedNetworkNode[]
  const ids = new Set(nodes.map((node) => node.id))
  const validEdges = edges.filter((edge) => ids.has(edge.source) && ids.has(edge.target) && edge.source !== edge.target)
  const adjacency = new Map<string, Set<string>>(nodes.map((node) => [node.id, new Set<string>()]))
  validEdges.forEach((edge) => {
    adjacency.get(edge.source)?.add(edge.target)
    adjacency.get(edge.target)?.add(edge.source)
  })

  const components = new Map<string, number>()
  let component = 0
  for (const node of nodes) {
    if (components.has(node.id)) continue
    const queue = [node.id]
    components.set(node.id, component)
    while (queue.length) {
      const current = queue.shift()!
      for (const neighbor of adjacency.get(current) ?? []) {
        if (!components.has(neighbor)) {
          components.set(neighbor, component)
          queue.push(neighbor)
        }
      }
    }
    component += 1
  }

  const padding = 56
  const centerX = width / 2
  const centerY = height / 2
  const orbitX = (width - padding * 2) * .46
  const orbitY = (height - padding * 2) * .46
  const positions = nodes.map((node, index) => {
    const angle = index * 2.399963229728653 + (hash(node.id) % 1000) / 1800
    const radial = Math.sqrt((index + .65) / nodes.length)
    const degree = adjacency.get(node.id)?.size ?? 0
    return {
      ...node,
      x: centerX + Math.cos(angle) * orbitX * radial,
      y: centerY + Math.sin(angle) * orbitY * radial,
      radius: Math.min(19, 8 + Math.sqrt(degree) * 2.6),
      degree,
      component: components.get(node.id) ?? 0,
      vx: 0,
      vy: 0,
    }
  })
  const byID = new Map(positions.map((node) => [node.id, node]))
  const forceEdges = selectNetworkRenderEdges(validEdges, Math.max(600, nodes.length * 6))
  const idealEdgeDistance = Math.min(240, 120 + Math.sqrt(nodes.length) * 8)
  const repulsionStrength = 5200 + Math.min(3600, nodes.length * 18)
  const forceIterations = nodes.length > 220 ? 28 : 120
  for (let iteration = 0; iteration < forceIterations; iteration += 1) {
    const cooling = 1 - iteration / (forceIterations * 1.1)
    for (let left = 0; left < positions.length; left += 1) {
      for (let right = left + 1; right < positions.length; right += 1) {
        const a = positions[left]
        const b = positions[right]
        const dx = b.x - a.x || 0.1
        const dy = b.y - a.y || 0.1
        const distance2 = Math.max(100, dx * dx + dy * dy)
        const force = (repulsionStrength / distance2) * cooling
        const distance = Math.sqrt(distance2)
        a.vx -= (dx / distance) * force
        a.vy -= (dy / distance) * force
        b.vx += (dx / distance) * force
        b.vy += (dy / distance) * force
      }
    }
    forceEdges.forEach((edge) => {
      const a = byID.get(edge.source)
      const b = byID.get(edge.target)
      if (!a || !b) return
      const dx = b.x - a.x
      const dy = b.y - a.y
      const distance = Math.max(1, Math.sqrt(dx * dx + dy * dy))
      const force = (distance - idealEdgeDistance) * 0.00145 * cooling
      a.vx += (dx / distance) * force
      a.vy += (dy / distance) * force
      b.vx -= (dx / distance) * force
      b.vy -= (dy / distance) * force
    })
    positions.forEach((node) => {
      node.vx += (centerX - node.x) * 0.00025
      node.vy += (centerY - node.y) * 0.00025
      node.vx *= 0.84
      node.vy *= 0.84
      node.x = Math.max(padding, Math.min(width - padding, node.x + node.vx))
      node.y = Math.max(padding, Math.min(height - padding, node.y + node.vy))
    })
  }
  const raw = positions.map(({ vx: _vx, vy: _vy, ...node }) => node)
  const minX = Math.min(...raw.map((node) => node.x))
  const maxX = Math.max(...raw.map((node) => node.x))
  const minY = Math.min(...raw.map((node) => node.y))
  const maxY = Math.max(...raw.map((node) => node.y))
  const rangeX = Math.max(1, maxX - minX)
  const rangeY = Math.max(1, maxY - minY)
  const normalized = raw.map((node) => ({
    ...node,
    x: padding + ((node.x - minX) / rangeX) * (width - padding * 2),
    y: padding + ((node.y - minY) / rangeY) * (height - padding * 2),
  }))
  const collisionGap = Math.min(26, 15 + Math.sqrt(nodes.length) * .55)
  return resolveNodeCollisions(normalized, width, height, collisionGap, nodes.length > 220 ? 24 : 48)
}
