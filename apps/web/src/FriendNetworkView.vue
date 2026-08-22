<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  Check,
  CircleStop,
  Eye,
  EyeOff,
  History,
  Maximize2,
  Minimize2,
  Minus,
  Move,
  Network,
  Play,
  Plus,
  RotateCcw,
  Search,
  ScanSearch,
  ShieldCheck,
  UserPlus,
  X,
} from '@lucide/vue'
import type { FriendAnnotation, FriendNetwork, FriendNetworkNode } from './api'
import { applyManualNetworkPositions, buildNetworkFocusIndex, compareCommunityMembers, compareNetworkSnapshots, detectNetworkCommunities, expandNetworkAvatarBudget, findShortestNetworkPath, layoutFriendNetwork, networkEdgeKey, rankBridgeNodes, seedFriendNetworkLayout, selectCommunityTheme, selectNetworkRenderEdges, shouldUseNetworkLayoutWorker, toggleElementFullscreen, zoomAroundPoint, type FriendNetworkSnapshot, type PositionedNetworkNode } from './friend-network'
import { preferredFriendAvatar } from './media'

const props = defineProps<{
  network: FriendNetwork | null
  scanning: boolean
  scanProcessed: number
  scanTotal: number
  scanEstimate: number
  scanMessage: string
  mediaUrl: (remoteUrl?: string) => string
  layoutKey: string
  annotations: Record<string, FriendAnnotation>
}>()

const emit = defineEmits<{
  startScan: []
  scanAll: []
  scanFriends: [userIDs: string[]]
  stopScan: []
  openFriend: [userID: string]
}>()

const query = ref('')
const onlineOnly = ref(false)
const coreOnly = ref(false)
const hideUnconnected = ref(true)
const selectedCommunity = ref<number | null>(null)
const collapsedCommunities = ref<Set<number>>(new Set())
const tagFilter = ref('')
const crossCommunityOnly = ref(false)
const edgeRenderMode = ref<'smart' | 'all'>('smart')
const evolutionOpen = ref(false)
const snapshots = ref<FriendNetworkSnapshot[]>([])
const snapshotIndex = ref(0)
const compareCommunityLeft = ref<number | null>(null)
const compareCommunityRight = ref<number | null>(null)
const pathStart = ref('')
const pathEnd = ref('')
const path = ref<string[]>([])
const pathMessage = ref('')
const hoveredID = ref('')
const selectedID = ref('')
const pickerOpen = ref(false)
const pickerQuery = ref('')
const pickedIDs = ref<Set<string>>(new Set())
const pinnedIDs = ref<Set<string>>(new Set())
const graphShellRef = ref<HTMLElement | null>(null)
const svgRef = ref<SVGSVGElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const fullscreen = ref(false)
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)
const manualPositions = ref<Record<string, { x: number; y: number }>>({})
const avatarFailures = ref<Set<string>>(new Set())
const avatarBudget = ref(72)
const graphFPS = ref(60)
const graphLongTasks = ref(0)
const interaction = ref<null | {
  mode: 'pan' | 'node'
  pointerID: number
  startClientX: number
  startClientY: number
  startPanX: number
  startPanY: number
  nodeID?: string
  nodeX?: number
  nodeY?: number
  moved: boolean
}>(null)

let suppressClickID = ''
let layoutSaveTimer: number | undefined
let snapshotSaveTimer: number | undefined
let hoverClearTimer: number | undefined
let hoverSwitchTimer: number | undefined
let hoverCandidateID = ''
let pointerFrame: number | undefined
let avatarExpansionTimer: number | undefined
let canvasFrame: number | undefined
let nodeClickTimer: number | undefined
let layoutWorker: Worker | undefined
let layoutRequestID = 0
let layoutFallbackTimer: number | undefined
let fpsFrame: number | undefined
let fpsStarted = 0
let fpsFrames = 0
let longTaskObserver: PerformanceObserver | undefined
let pendingPointer: { clientX: number; clientY: number } | null = null
const edgeElements = new Map<string, SVGLineElement>()
let highlightedEdgeKeys: string[] = []
let renderedPathEdgeKeys: string[] = []
const nodeElements = new Map<string, SVGGElement>()
let highlightedNodeIDs: string[] = []
let labelledNodeIDs: string[] = []
const layoutLoaded = ref(false)
const palette = ['#a37755', '#6f806d', '#a58a61', '#83717b', '#a16562', '#668084']

const degrees = computed(() => {
  const result = new Map<string, number>()
  props.network?.edges.forEach((edge) => {
    result.set(edge.source, (result.get(edge.source) ?? 0) + 1)
    result.set(edge.target, (result.get(edge.target) ?? 0) + 1)
  })
  return result
})

const isolatedCount = computed(() => (props.network?.nodes ?? []).filter((node) => !degrees.value.get(node.id)).length)
const connectedCount = computed(() => Math.max(0, (props.network?.nodes.length ?? 0) - isolatedCount.value))
const coreThreshold = computed(() => Math.max(3, Math.round(((props.network?.edges.length ?? 0) * 2) / Math.max(1, connectedCount.value))))
const communityByID = computed(() => detectNetworkCommunities(props.network?.nodes ?? [], props.network?.edges ?? []))
const communities = computed(() => {
  const groups = new Map<number, FriendNetworkNode[]>()
  for (const node of props.network?.nodes ?? []) {
    const community = communityByID.value.get(node.id) ?? -1
    const members = groups.get(community) ?? []
    members.push(node)
    groups.set(community, members)
  }
  return [...groups.entries()].map(([id, members]) => {
    const theme = selectCommunityTheme(members, props.network?.edges ?? [])!
    return {
      id,
      theme: theme.node,
      themeDegree: theme.degree,
      edgeCount: theme.edgeCount,
      members: members.sort((left, right) => (theme.internalDegrees.get(right.id) ?? 0) - (theme.internalDegrees.get(left.id) ?? 0)),
    }
  }).filter((community) => community.members.some((node) => (degrees.value.get(node.id) ?? 0) > 0))
    .sort((left, right) => right.members.length - left.members.length)
})
const tagOptions = computed(() => [...new Set(Object.values(props.annotations).flatMap((annotation) => annotation.tags ?? []))].sort((left, right) => left.localeCompare(right)))
const bridgeRanking = computed(() => {
  const names = new Map((props.network?.nodes ?? []).map((node) => [node.id, node.displayName]))
  return rankBridgeNodes(props.network?.nodes ?? [], props.network?.edges ?? [], communityByID.value).slice(0, 8)
    .map((item) => ({ ...item, displayName: names.get(item.id) ?? item.id }))
})
const currentSnapshot = computed<FriendNetworkSnapshot>(() => ({
  capturedAt: props.network?.generatedAt || new Date().toISOString(),
  nodeIDs: (props.network?.nodes ?? []).filter((node) => (degrees.value.get(node.id) ?? 0) > 0).map((node) => node.id).sort(),
  edges: [...(props.network?.edges ?? [])].map((edge) => ({ source: edge.source, target: edge.target })),
}))
const selectedSnapshot = computed(() => snapshots.value[snapshotIndex.value] ?? null)
const evolutionDelta = computed(() => compareNetworkSnapshots(selectedSnapshot.value, currentSnapshot.value))
const communityComparison = computed(() => {
  const left = communities.value.find((item) => item.id === compareCommunityLeft.value)
  const right = communities.value.find((item) => item.id === compareCommunityRight.value)
  if (!left || !right || left.id === right.id) return null
  const observedCircle = (themeID: string) => {
    const ids = new Set([themeID])
    for (const edge of props.network?.edges ?? []) {
      if (edge.source === themeID) ids.add(edge.target)
      if (edge.target === themeID) ids.add(edge.source)
    }
    return [...ids]
  }
  return compareCommunityMembers(observedCircle(left.theme.id), observedCircle(right.theme.id))
})

const visibleNodes = computed(() => {
  const text = query.value.trim().toLocaleLowerCase()
  return (props.network?.nodes ?? []).filter((node) => {
    const community = communityByID.value.get(node.id) ?? -1
    if (onlineOnly.value && !node.online) return false
    if (hideUnconnected.value && !degrees.value.get(node.id) && !pinnedIDs.value.has(node.id)) return false
    if (coreOnly.value && (degrees.value.get(node.id) ?? 0) < coreThreshold.value && !pinnedIDs.value.has(node.id)) return false
    if (collapsedCommunities.value.has(community)) return false
    if (selectedCommunity.value !== null && selectedCommunity.value !== community) return false
    if (tagFilter.value && !props.annotations[node.id]?.tags?.includes(tagFilter.value)) return false
    return !text || node.displayName.toLocaleLowerCase().includes(text) || node.id.toLocaleLowerCase().includes(text)
  })
})

const visibleIDs = computed(() => new Set(visibleNodes.value.map((node) => node.id)))
const structuralEdges = computed(() => (props.network?.edges ?? []).filter((edge) => visibleIDs.value.has(edge.source) && visibleIDs.value.has(edge.target)))
const visibleEdges = computed(() => structuralEdges.value.filter((edge) => !crossCommunityOnly.value || communityByID.value.get(edge.source) !== communityByID.value.get(edge.target)))
const largeGraph = computed(() => visibleNodes.value.length > 220 || visibleEdges.value.length > 1800)
const edgeBudget = computed(() => Math.max(700, Math.min(1400, visibleNodes.value.length * 2)))
const graphSize = computed(() => {
  const count = visibleNodes.value.length
  const width = Math.max(1080, Math.min(1800, 900 + count * 4.8))
  return { width, height: Math.max(680, Math.round(width * .62)) }
})

// Large force layouts run outside the UI thread. Pointer movement only changes
// the viewport and never restarts layout work.
const automaticPositions = ref<PositionedNetworkNode[]>([])
function requestAutomaticLayout() {
  const id=++layoutRequestID,nodes=visibleNodes.value,edges=structuralEdges.value,size=graphSize.value
  window.clearTimeout(layoutFallbackTimer)
  if(!shouldUseNetworkLayoutWorker(nodes.length)){automaticPositions.value=layoutFriendNetwork(nodes,edges,size.width,size.height);return}
  // Never leave the graph blank while WebView2 starts (or blocks) the module Worker.
  // This O(n+e) seed renders immediately; the Worker replaces it with the refined layout.
  automaticPositions.value=seedFriendNetworkLayout(nodes,edges,size.width,size.height)
  try {
    if(!layoutWorker){layoutWorker=new Worker(new URL('./friend-network.worker.ts',import.meta.url),{type:'module'});layoutWorker.onmessage=(event:MessageEvent<{id:number;positions:PositionedNetworkNode[]}>)=>{if(event.data.id===layoutRequestID&&event.data.positions.length){window.clearTimeout(layoutFallbackTimer);automaticPositions.value=event.data.positions}};layoutWorker.onerror=()=>{layoutWorker?.terminate();layoutWorker=undefined}}
    layoutWorker.postMessage({id,nodes,edges,width:size.width,height:size.height})
  } catch {
    layoutWorker?.terminate()
    layoutWorker=undefined
  }
  layoutFallbackTimer=window.setTimeout(()=>{if(id===layoutRequestID&&automaticPositions.value.length===0)automaticPositions.value=seedFriendNetworkLayout(nodes,edges,size.width,size.height)},2500)
}
watch([visibleNodes,structuralEdges,graphSize],requestAutomaticLayout,{immediate:true})

const rawPositioned = computed(() => automaticPositions.value.map((node) => {
  const manual = manualPositions.value[node.id]
  if (!manual) return node
  return {
    ...node,
    x: manual.x,
    y: manual.y,
  }
}))
const positioned = computed(() => interaction.value?.mode === 'node'
  ? rawPositioned.value
  : applyManualNetworkPositions(
      automaticPositions.value,
      manualPositions.value,
      graphSize.value.width,
      graphSize.value.height,
      Math.min(26, 18 + Math.sqrt(visibleNodes.value.length) * .45),
    ))

const positions = computed(() => new Map(positioned.value.map((node) => [node.id, node])))
// Hover previews another friend temporarily; click explicitly locks a focus until
// the user clears it, so a persistent dimmed state is always visible and escapable.
const focusedID = computed(() => hoveredID.value || selectedID.value)
const lockedFocusNode = computed(() => selectedID.value ? positions.value.get(selectedID.value) : null)
const focusIndex = computed(() => buildNetworkFocusIndex(visibleEdges.value))
const pathIDs = computed(() => new Set(path.value))
const pathEdgeKeys = computed(() => new Set(path.value.slice(1).map((id, index) => {
  const source = path.value[index]
  return networkEdgeKey({ source, target: id })
})))
const renderedNodes = computed(() => {
  if(!largeGraph.value || zoom.value>=1.15)return positioned.value
  const budget=zoom.value<.62?220:360,result=new Set<string>([...pinnedIDs.value,...pathIDs.value])
  if(focusedID.value){result.add(focusedID.value);for(const id of focusIndex.value.get(focusedID.value)?.neighbors??[])result.add(id)}
  ;[...positioned.value].sort((a,b)=>(degrees.value.get(b.id)??0)-(degrees.value.get(a.id)??0)||a.id.localeCompare(b.id)).slice(0,budget).forEach(node=>result.add(node.id))
  return positioned.value.filter(node=>result.has(node.id))
})
const renderedEdges = computed(() => {
  if (!largeGraph.value || edgeRenderMode.value === 'all') return visibleEdges.value
  return selectNetworkRenderEdges(visibleEdges.value, edgeBudget.value, focusedID.value, pathEdgeKeys.value)
})
const avatarEligibleIDs = computed(() => {
  if (!largeGraph.value) return visibleIDs.value
  const result = new Set<string>([...pinnedIDs.value, ...pathIDs.value])
  if (focusedID.value) {
    result.add(focusedID.value)
    const neighbors = [...(focusIndex.value.get(focusedID.value)?.neighbors ?? [])]
      .sort((left, right) => (degrees.value.get(right) ?? 0) - (degrees.value.get(left) ?? 0))
      .slice(0, 24)
    neighbors.forEach((id) => result.add(id))
  }
  ;[...visibleNodes.value]
    .sort((left, right) => (degrees.value.get(right.id) ?? 0) - (degrees.value.get(left.id) ?? 0) || left.id.localeCompare(right.id))
    .slice(0, avatarBudget.value)
    .forEach((node) => result.add(node.id))
  return result
})
const ambientLabelIDs = computed(() => {
  const count = visibleNodes.value.length <= 24 ? visibleNodes.value.length : visibleNodes.value.length <= 80 ? 16 : zoom.value >= 1.6 ? 24 : 10
  return new Set([...visibleNodes.value]
    .sort((left, right) => (degrees.value.get(right.id) ?? 0) - (degrees.value.get(left.id) ?? 0))
    .slice(0, count)
    .map((node) => node.id))
})
const coverage = computed(() => props.network?.totalFriends ? Math.round((props.network.scannedCount / props.network.totalFriends) * 100) : 0)
const pickerCandidates = computed(() => {
  const text = pickerQuery.value.trim().toLocaleLowerCase()
  return [...(props.network?.nodes ?? [])]
    .filter((node) => !text || node.displayName.toLocaleLowerCase().includes(text) || node.id.toLocaleLowerCase().includes(text))
    .sort((left, right) => {
      if (pickedIDs.value.has(left.id) !== pickedIDs.value.has(right.id)) return pickedIDs.value.has(left.id) ? -1 : 1
      if (left.online !== right.online) return left.online ? -1 : 1
      return left.displayName.localeCompare(right.displayName)
    })
    .slice(0, 80)
})

watch(() => props.layoutKey, (key) => {
  layoutLoaded.value = false
  manualPositions.value = {}
  pinnedIDs.value = new Set()
  collapsedCommunities.value = new Set()
  selectedCommunity.value = null
  selectedID.value = ''
  clearTransientFocus()
  tagFilter.value = ''
  crossCommunityOnly.value = false
  edgeRenderMode.value = 'smart'
  avatarBudget.value = 72
  zoom.value = 1
  panX.value = 0
  panY.value = 0
  snapshots.value = []
  snapshotIndex.value = 0
  if (key) {
    try {
      const saved = JSON.parse(localStorage.getItem(`vrc-harbor-graph-layout:${key}`) || 'null')
      if (saved?.version === 2) {
        manualPositions.value = saved.positions ?? {}
        pinnedIDs.value = new Set(Array.isArray(saved.pinnedIDs) ? saved.pinnedIDs : [])
        hideUnconnected.value = saved.hideUnconnected !== false
        coreOnly.value = saved.coreOnly === true
        collapsedCommunities.value = new Set(Array.isArray(saved.collapsedCommunities) ? saved.collapsedCommunities : [])
        tagFilter.value = typeof saved.tagFilter === 'string' ? saved.tagFilter : ''
        crossCommunityOnly.value = saved.crossCommunityOnly === true
        zoom.value = Math.max(.35, Math.min(3.5, Number(saved.zoom) || 1))
        panX.value = Number(saved.panX) || 0
        panY.value = Number(saved.panY) || 0
      }
    } catch { /* use automatic layout */ }
    try {
      const savedHistory = JSON.parse(localStorage.getItem(`vrc-harbor-graph-history:${key}`) || '[]')
      if (Array.isArray(savedHistory)) snapshots.value = savedHistory.slice(0, 30)
    } catch { /* graph history is optional */ }
  }
  layoutLoaded.value = true
}, { immediate: true })

const snapshotSignature = computed(() => `${currentSnapshot.value.nodeIDs.join(',')}::${currentSnapshot.value.edges.map(networkEdgeKey).sort().join(',')}`)
watch(snapshotSignature, () => {
  if (!props.layoutKey || !props.network?.nodes.length) return
  window.clearTimeout(snapshotSaveTimer)
  snapshotSaveTimer = window.setTimeout(() => {
    const current = currentSnapshot.value
    const latest = snapshots.value[0]
    const latestSignature = latest ? `${latest.nodeIDs.join(',')}::${latest.edges.map(networkEdgeKey).sort().join(',')}` : ''
    if (latestSignature === snapshotSignature.value) return
    snapshots.value = [{ ...current, capturedAt: new Date().toISOString() }, ...snapshots.value].slice(0, 30)
    snapshotIndex.value = snapshots.value.length > 1 ? 1 : 0
    try { localStorage.setItem(`vrc-harbor-graph-history:${props.layoutKey}`, JSON.stringify(snapshots.value)) } catch { /* optional */ }
  }, 1800)
}, { immediate: true })

watch([manualPositions, pinnedIDs, hideUnconnected, coreOnly, collapsedCommunities, tagFilter, crossCommunityOnly, zoom, panX, panY], () => {
  if (!layoutLoaded.value || !props.layoutKey) return
  window.clearTimeout(layoutSaveTimer)
  window.clearTimeout(snapshotSaveTimer)
  layoutSaveTimer = window.setTimeout(() => {
    try {
      localStorage.setItem(`vrc-harbor-graph-layout:${props.layoutKey}`, JSON.stringify({
        version: 2,
        positions: manualPositions.value,
        pinnedIDs: [...pinnedIDs.value],
        hideUnconnected: hideUnconnected.value,
        coreOnly: coreOnly.value,
        collapsedCommunities: [...collapsedCommunities.value],
        tagFilter: tagFilter.value,
        crossCommunityOnly: crossCommunityOnly.value,
        zoom: zoom.value,
        panX: panX.value,
        panY: panY.value,
      }))
    } catch { /* layout persistence is optional */ }
  }, 400)
}, { deep: true })

function syncFullscreen() {
  fullscreen.value = document.fullscreenElement === graphShellRef.value
}

function applyEdgeHighlights(userID: string) {
  for (const key of highlightedEdgeKeys) edgeElements.get(key)?.classList.remove('highlighted')
  highlightedEdgeKeys = userID ? (focusIndex.value.get(userID)?.edgeKeys ?? []) : []
  for (const key of highlightedEdgeKeys) edgeElements.get(key)?.classList.add('highlighted')
  scheduleCanvasDraw()
}

function applyNodeHighlights(userID: string) {
  for (const id of highlightedNodeIDs) nodeElements.get(id)?.classList.remove('related', 'hovered')
  for (const id of labelledNodeIDs) nodeElements.get(id)?.classList.remove('related-label')
  highlightedNodeIDs = []
  labelledNodeIDs = []
  if (!userID) return

  const related = [userID, ...(focusIndex.value.get(userID)?.neighbors ?? [])]
  highlightedNodeIDs = related
  for (const id of related) nodeElements.get(id)?.classList.add('related')
  nodeElements.get(userID)?.classList.add('hovered')

  labelledNodeIDs = [userID, ...related
    .filter((id) => id !== userID)
    .sort((left, right) => (degrees.value.get(right) ?? 0) - (degrees.value.get(left) ?? 0))
    .slice(0, 8)]
  for (const id of labelledNodeIDs) nodeElements.get(id)?.classList.add('related-label')
}

function applyPathHighlights() {
  for (const key of renderedPathEdgeKeys) edgeElements.get(key)?.classList.remove('path-edge')
  renderedPathEdgeKeys = [...pathEdgeKeys.value]
  for (const key of renderedPathEdgeKeys) edgeElements.get(key)?.classList.add('path-edge')
  scheduleCanvasDraw()
}

function scheduleCanvasDraw() {
  if (!largeGraph.value || canvasFrame !== undefined) return
  canvasFrame = window.requestAnimationFrame(() => { canvasFrame = undefined; drawCanvasEdges() })
}

function sampleGraphFPS(now:number){if(!fpsStarted)fpsStarted=now;fpsFrames+=1;if(now-fpsStarted>=2000){graphFPS.value=Math.round(fpsFrames*1000/(now-fpsStarted));fpsFrames=0;fpsStarted=now}fpsFrame=window.requestAnimationFrame(sampleGraphFPS)}

function drawCanvasEdges() {
  const canvas=canvasRef.value;if(!canvas||!largeGraph.value)return
  const dpr=Math.min(window.devicePixelRatio||1,2),width=graphSize.value.width,height=graphSize.value.height
  const pixelWidth=Math.round(width*dpr),pixelHeight=Math.round(height*dpr);if(canvas.width!==pixelWidth)canvas.width=pixelWidth;if(canvas.height!==pixelHeight)canvas.height=pixelHeight
  const context=canvas.getContext('2d');if(!context)return;context.setTransform(dpr,0,0,dpr,0,0);context.clearRect(0,0,width,height);context.translate(panX.value,panY.value);context.scale(zoom.value,zoom.value)
  const styles=getComputedStyle(canvas),base=styles.getPropertyValue('--line-strong').trim()||'#777',accent=styles.getPropertyValue('--accent').trim()||'#b7895f',warning=styles.getPropertyValue('--warning').trim()||'#d59a45',focusKeys=new Set(highlightedEdgeKeys)
  const paint=(edges:typeof renderedEdges.value,color:string,alpha:number,lineWidth:number)=>{context.beginPath();context.strokeStyle=color;context.globalAlpha=alpha;context.lineWidth=lineWidth/zoom.value;for(const edge of edges){const a=positions.value.get(edge.source),b=positions.value.get(edge.target);if(!a||!b)continue;context.moveTo(a.x,a.y);context.lineTo(b.x,b.y)}context.stroke()}
  paint(renderedEdges.value.filter(edge=>!focusKeys.has(networkEdgeKey(edge))&&!pathEdgeKeys.value.has(networkEdgeKey(edge))),base,focusedID.value?.06:.19,1.05)
  if(focusKeys.size)paint(renderedEdges.value.filter(edge=>focusKeys.has(networkEdgeKey(edge))),accent,.92,2)
  if(pathEdgeKeys.value.size)paint(renderedEdges.value.filter(edge=>pathEdgeKeys.value.has(networkEdgeKey(edge))),warning,1,2.6)
  context.globalAlpha=1
}

function rebuildEdgeElementIndex() {
  edgeElements.clear()
  svgRef.value?.querySelectorAll<SVGLineElement>('.network-edge').forEach((element) => {
    if (element.dataset.edgeKey) edgeElements.set(element.dataset.edgeKey, element)
  })
  nodeElements.clear()
  svgRef.value?.querySelectorAll<SVGGElement>('.graph-node').forEach((element) => {
    if (element.dataset.nodeId) nodeElements.set(element.dataset.nodeId, element)
  })
  applyEdgeHighlights(focusedID.value)
  applyNodeHighlights(focusedID.value)
  applyPathHighlights()
}

watch([renderedEdges, visibleNodes], () => { void nextTick(()=>{rebuildEdgeElementIndex();scheduleCanvasDraw()}) })
watch([positions, zoom, panX, panY, largeGraph], scheduleCanvasDraw, { deep:true, flush:'post' })
watch([visibleNodes, largeGraph, () => props.layoutKey], scheduleAvatarExpansion, { immediate: true, flush: 'post' })
watch(visibleIDs, (ids) => {
  if (hoveredID.value && !ids.has(hoveredID.value)) clearTransientFocus()
  if (selectedID.value && !ids.has(selectedID.value)) selectedID.value = ''
})
watch(focusedID, (userID) => {
  applyEdgeHighlights(userID)
  applyNodeHighlights(userID)
}, { flush: 'post' })
watch(pathEdgeKeys, () => { void nextTick(applyPathHighlights) })

onMounted(() => {
  document.addEventListener('fullscreenchange', syncFullscreen)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('blur', clearTransientFocus)
  window.addEventListener('keydown', handleGlobalKeydown)
  void nextTick(rebuildEdgeElementIndex)
  void nextTick(scheduleCanvasDraw)
  fpsFrame=window.requestAnimationFrame(sampleGraphFPS)
  if('PerformanceObserver'in window){try{longTaskObserver=new PerformanceObserver(list=>{graphLongTasks.value+=list.getEntries().length});longTaskObserver.observe({entryTypes:['longtask']})}catch{/* Chromium may disable long-task entries */}}
})
onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', syncFullscreen)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('blur', clearTransientFocus)
  window.removeEventListener('keydown', handleGlobalKeydown)
  window.clearTimeout(layoutSaveTimer)
  window.clearTimeout(hoverClearTimer)
  window.clearTimeout(hoverSwitchTimer)
  window.clearTimeout(avatarExpansionTimer)
  if (pointerFrame !== undefined) window.cancelAnimationFrame(pointerFrame)
  if (canvasFrame !== undefined) window.cancelAnimationFrame(canvasFrame)
  window.clearTimeout(nodeClickTimer)
  layoutWorker?.terminate()
  window.clearTimeout(layoutFallbackTimer)
  if(fpsFrame!==undefined)window.cancelAnimationFrame(fpsFrame)
  longTaskObserver?.disconnect()
  edgeElements.clear()
  nodeElements.clear()
})

function nodeColor(component: number) {
  return palette[component % palette.length]
}

function scheduleAvatarExpansion() {
  window.clearTimeout(avatarExpansionTimer)
  if (!largeGraph.value || avatarBudget.value >= visibleNodes.value.length) return
  avatarExpansionTimer = window.setTimeout(() => {
    avatarBudget.value = expandNetworkAvatarBudget(avatarBudget.value, visibleNodes.value.length)
    scheduleAvatarExpansion()
  }, 500)
}

function snapshotLabel(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

function toggleCommunity(communityID: number) {
  selectedCommunity.value = selectedCommunity.value === communityID ? null : communityID
  path.value = []
  pathMessage.value = ''
}

function toggleCommunityCollapsed(communityID: number) {
  const next = new Set(collapsedCommunities.value)
  if (next.has(communityID)) next.delete(communityID)
  else next.add(communityID)
  collapsedCommunities.value = next
  if (selectedCommunity.value === communityID) selectedCommunity.value = null
}

function calculatePath() {
  path.value = findShortestNetworkPath(structuralEdges.value, pathStart.value, pathEnd.value)
  if (path.value.length) pathMessage.value = `${path.value.length - 1} 跳 · ${path.value.map((id) => positions.value.get(id)?.displayName ?? id).join(' → ')}`
  else pathMessage.value = '当前筛选范围内没有观察到连接路径'
}

function clearPath() {
  path.value = []
  pathMessage.value = ''
}

function toggleEdgeRenderMode() {
  if (edgeRenderMode.value === 'smart' && visibleEdges.value.length > 5000
    && !window.confirm(`将同时绘制 ${visibleEdges.value.length} 条连线，可能降低操作流畅度。仍要继续吗？`)) return
  edgeRenderMode.value = edgeRenderMode.value === 'smart' ? 'all' : 'smart'
}

function selectNode(userID: string) {
  if (suppressClickID === userID) return
  window.clearTimeout(nodeClickTimer)
  nodeClickTimer = window.setTimeout(() => { selectedID.value = selectedID.value === userID ? '' : userID }, 220)
}

function openNode(userID: string) {
  window.clearTimeout(nodeClickTimer)
  emit('openFriend', userID)
}

function avatarURL(node: FriendNetworkNode) {
  if (avatarFailures.value.has(node.id)) return ''
  return props.mediaUrl(preferredFriendAvatar(node))
}

function graphAvatarURL(node: FriendNetworkNode) {
  return avatarEligibleIDs.value.has(node.id) ? avatarURL(node) : ''
}

function enterNode(userID: string) {
  window.clearTimeout(hoverClearTimer)
  window.clearTimeout(hoverSwitchTimer)
  hoverCandidateID = userID
  if (!hoveredID.value || hoveredID.value === userID) {
    hoveredID.value = userID
    return
  }
  hoverSwitchTimer = window.setTimeout(() => {
    if (hoverCandidateID === userID) hoveredID.value = userID
  }, 45)
}

function clearTransientFocus() {
  window.clearTimeout(hoverClearTimer)
  window.clearTimeout(hoverSwitchTimer)
  hoverCandidateID = ''
  hoveredID.value = ''
}

function handleVisibilityChange() {
  if (document.hidden) clearTransientFocus()
}

function clearLockedFocus() {
  selectedID.value = ''
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') clearLockedFocus()
}

function leaveNode(userID: string) {
  window.clearTimeout(hoverClearTimer)
  if (hoverCandidateID === userID) {
    hoverCandidateID = ''
    window.clearTimeout(hoverSwitchTimer)
  }
  hoverClearTimer = window.setTimeout(() => {
    if (hoveredID.value === userID) hoveredID.value = ''
  }, 80)
}

function markAvatarFailed(userID: string) {
  avatarFailures.value = new Set([...avatarFailures.value, userID])
}

function setZoom(nextZoom: number, anchorX = graphSize.value.width / 2, anchorY = graphSize.value.height / 2) {
  const result = zoomAroundPoint(zoom.value, Math.max(.35, Math.min(3.5, nextZoom)), panX.value, panY.value, anchorX, anchorY)
  zoom.value = result.zoom
  panX.value = result.panX
  panY.value = result.panY
}

function handleWheel(event: WheelEvent) {
  clearTransientFocus()
  const rect = svgRef.value?.getBoundingClientRect()
  if (!rect) return
  const anchorX = (event.clientX - rect.left) / rect.width * graphSize.value.width
  const anchorY = (event.clientY - rect.top) / rect.height * graphSize.value.height
  setZoom(zoom.value * (event.deltaY > 0 ? .9 : 1.11), anchorX, anchorY)
}

function resetView() {
  zoom.value = 1
  panX.value = 0
  panY.value = 0
  manualPositions.value = {}
}

async function toggleFullscreen() {
  try {
    if (graphShellRef.value) await toggleElementFullscreen(graphShellRef.value, document.fullscreenElement, () => document.exitFullscreen())
  } catch { /* browser may block fullscreen outside a user gesture */ }
}

function beginPan(event: PointerEvent) {
  if (event.button !== 0 || (event.target as Element).closest('.graph-node, .graph-controls, .graph-focus-lock, .friend-picker')) return
  clearTransientFocus()
  svgRef.value?.setPointerCapture(event.pointerId)
  interaction.value = {
    mode: 'pan', pointerID: event.pointerId, startClientX: event.clientX, startClientY: event.clientY,
    startPanX: panX.value, startPanY: panY.value, moved: false,
  }
}

function beginNodeDrag(event: PointerEvent, userID: string) {
  if (event.button !== 0) return
  event.stopPropagation()
  const node = positions.value.get(userID)
  if (!node) return
  svgRef.value?.setPointerCapture(event.pointerId)
  interaction.value = {
    mode: 'node', pointerID: event.pointerId, startClientX: event.clientX, startClientY: event.clientY,
    startPanX: panX.value, startPanY: panY.value, nodeID: userID, nodeX: node.x, nodeY: node.y, moved: false,
  }
}

function applyPointerPosition(clientX: number, clientY: number) {
  const current = interaction.value
  const rect = svgRef.value?.getBoundingClientRect()
  if (!current || !rect) return
  const dx = (clientX - current.startClientX) / rect.width * graphSize.value.width
  const dy = (clientY - current.startClientY) / rect.height * graphSize.value.height
  current.moved ||= Math.abs(dx) + Math.abs(dy) > 3
  if (current.mode === 'pan') {
    panX.value = current.startPanX + dx
    panY.value = current.startPanY + dy
  } else if (current.nodeID && current.nodeX !== undefined && current.nodeY !== undefined) {
    manualPositions.value[current.nodeID] = {
      x: current.nodeX + dx / zoom.value,
      y: current.nodeY + dy / zoom.value,
    }
  }
}

function movePointer(event: PointerEvent) {
  const current = interaction.value
  if (!current || current.pointerID !== event.pointerId) return
  pendingPointer = { clientX: event.clientX, clientY: event.clientY }
  if (pointerFrame !== undefined) return
  pointerFrame = window.requestAnimationFrame(() => {
    pointerFrame = undefined
    if (pendingPointer) applyPointerPosition(pendingPointer.clientX, pendingPointer.clientY)
    pendingPointer = null
  })
}

function endPointer(event: PointerEvent) {
  const current = interaction.value
  if (!current || current.pointerID !== event.pointerId) return
  if (pendingPointer) applyPointerPosition(pendingPointer.clientX, pendingPointer.clientY)
  pendingPointer = null
  if (pointerFrame !== undefined) window.cancelAnimationFrame(pointerFrame)
  pointerFrame = undefined
  if (current.mode === 'node' && current.moved && current.nodeID) {
    suppressClickID = current.nodeID
    window.setTimeout(() => { suppressClickID = '' }, 0)
  }
  if (current.mode === 'pan' && !current.moved) clearLockedFocus()
  if (svgRef.value?.hasPointerCapture(event.pointerId)) svgRef.value.releasePointerCapture(event.pointerId)
  interaction.value = null
}

function togglePicked(userID: string) {
  const next = new Set(pickedIDs.value)
  if (next.has(userID)) next.delete(userID)
  else if (next.size < 20) next.add(userID)
  pickedIDs.value = next
}

function addPicked(scan: boolean) {
  if (!pickedIDs.value.size) return
  pinnedIDs.value = new Set([...pinnedIDs.value, ...pickedIDs.value])
  if (scan) emit('scanFriends', [...pickedIDs.value])
  pickedIDs.value = new Set()
  pickerQuery.value = ''
  pickerOpen.value = false
}
</script>

<template>
  <article ref="graphShellRef" class="network-shell panel">
    <header class="network-header">
      <div class="network-title">
        <div class="network-icon"><Network :size="19" /></div>
        <div>
          <h2>好友关系网</h2>
          <p>看看朋友圈和共同连接。</p>
        </div>
      </div>
      <div class="network-actions">
        <button class="secondary-action" type="button" @click="pickerOpen = !pickerOpen">
          <UserPlus :size="16" />添加好友
        </button>
        <button v-if="scanning" class="scan-button stop" type="button" @click="emit('stopScan')">
          <CircleStop :size="16" />停止扫描
        </button>
        <button v-else class="scan-button" type="button" :disabled="scanEstimate === 0" @click="emit('startScan')">
          <Play :size="16" />默认扫描<span v-if="scanEstimate">{{ scanEstimate }} 位</span>
        </button>
        <button v-if="!scanning" class="scan-all-button" type="button" :disabled="!(network?.totalFriends)" @click="emit('scanAll')"><ScanSearch :size="16" />一键扫描全部</button>
      </div>
    </header>

    <div class="network-summary-row">
      <span><b>{{ connectedCount }}</b> 位好友</span>
      <span><b>{{ network?.edges.length ?? 0 }}</b> 条连接</span>
      <span>覆盖 <b>{{ coverage }}%</b></span>
      <span v-if="scanEstimate" class="scan-estimate">下次扫描 {{ scanEstimate }} 位</span>
      <span v-else class="scan-estimate">已是最新</span>
    </div>
    <div class="network-coverage-tip"><ShieldCheck :size="15" /><span>默认扫描最多 100 位好友；好友不足 100 位时会扫描全部。想体验完整好友关系、朋友圈和变化追踪，建议使用“一键扫描全部”。</span></div>

    <div v-if="scanning || scanMessage" class="scan-progress" :class="{ active: scanning }">
      <span>{{ scanMessage || '正在读取共同好友关系' }}</span>
      <strong v-if="scanTotal">{{ scanProcessed }}/{{ scanTotal }}</strong>
      <div v-if="scanTotal"><i :style="{ width: `${Math.round(scanProcessed / scanTotal * 100)}%` }"></i></div>
    </div>

    <section class="community-strip">
      <div class="community-strip-heading"><strong>朋友圈</strong><span>{{ communities.length }} 个</span></div>
      <div class="community-list">
        <div v-for="community in communities" :key="community.id" class="community-item" :class="{ active: selectedCommunity === community.id, collapsed: collapsedCommunities.has(community.id) }">
          <i :style="{ background: nodeColor(community.id) }"></i>
          <button class="community-main" type="button" :aria-label="`查看你与 ${community.theme.displayName} 的朋友圈`" @click="toggleCommunity(community.id)">
            <span class="community-avatar">
              <img v-if="avatarURL(community.theme)" :src="avatarURL(community.theme)" alt="" loading="lazy" decoding="async" @error="markAvatarFailed(community.theme.id)" />
              <b v-else>{{ community.theme.displayName.slice(0, 1).toUpperCase() }}</b>
            </span>
            <span class="community-copy">
              <strong>{{ community.theme.displayName }} 的朋友圈</strong>
              <small>{{ community.themeDegree }} 条圈内连接</small>
            </span>
            <em>{{ community.members.length }} 人</em>
          </button>
          <button class="community-collapse" type="button" @click="toggleCommunityCollapsed(community.id)">{{ collapsedCommunities.has(community.id) ? '展开' : '折叠' }}</button>
        </div>
      </div>
    </section>

    <div class="network-toolbar">
      <label class="network-search"><Search :size="16" /><input v-model="query" placeholder="搜索好友或 ID" /></label>
      <button :class="{ active: onlineOnly }" :aria-pressed="onlineOnly" @click="onlineOnly = !onlineOnly">只看在线</button>
      <button :class="{ active: coreOnly }" :aria-pressed="coreOnly" @click="coreOnly = !coreOnly">核心好友</button>
      <button class="isolate-toggle" :class="{ active: hideUnconnected }" :aria-pressed="hideUnconnected" @click="hideUnconnected = !hideUnconnected">
        <EyeOff v-if="hideUnconnected" :size="15" /><Eye v-else :size="15" />
        {{ hideUnconnected ? `已隐藏 ${isolatedCount} 位孤立好友` : '隐藏孤立好友' }}
      </button>
      <select v-model="tagFilter" aria-label="按本机标签筛选"><option value="">全部标签</option><option v-for="tag in tagOptions" :key="tag" :value="tag"># {{ tag }}</option></select>
      <button :class="{ active: crossCommunityOnly }" :aria-pressed="crossCommunityOnly" @click="crossCommunityOnly = !crossCommunityOnly">跨圈连接</button>
      <button v-if="largeGraph" :class="{ active: edgeRenderMode === 'all' }" :aria-pressed="edgeRenderMode === 'all'" @click="toggleEdgeRenderMode">
        {{ edgeRenderMode === 'smart' ? '显示全部连线' : '恢复流畅模式' }}
      </button>
      <button :class="{ active: evolutionOpen }" :aria-pressed="evolutionOpen" @click="evolutionOpen = !evolutionOpen"><History :size="15" />变化</button>
      <span class="visible-count">{{ visibleNodes.length }} / {{ network?.nodes.length ?? 0 }}</span>
    </div>

    <div v-if="largeGraph && edgeRenderMode === 'smart'" class="graph-performance-note">
      <ShieldCheck :size="14" />
      <span><b>大图流畅模式</b>：完整保留 {{ visibleEdges.length }} 条关系用于搜索、寻路和统计；后台 Worker 布局、显示 {{ renderedEdges.length }} 条关键连线、语义缩放 {{ renderedNodes.length }}/{{ visibleNodes.length }} 个节点。当前约 {{ graphFPS }} FPS，本次打开记录 {{ graphLongTasks }} 个长任务。</span>
    </div>

    <section v-if="evolutionOpen" class="evolution-panel">
      <div class="evolution-block">
        <header><strong>与历史快照比较</strong><span>本机最多保存 30 个变化快照</span></header>
        <select v-model.number="snapshotIndex"><option v-for="(snapshot,index) in snapshots" :key="`${snapshot.capturedAt}-${index}`" :value="index">{{ snapshotLabel(snapshot.capturedAt) }} · {{ snapshot.nodeIDs.length }} 人 / {{ snapshot.edges.length }} 线</option></select>
        <div class="delta-grid"><span><b>+{{ evolutionDelta.addedNodes.length }}</b> 新进入连接区</span><span><b>-{{ evolutionDelta.removedNodes.length }}</b> 未在当前观察到</span><span><b>+{{ evolutionDelta.addedEdges.length }}</b> 新观察连线</span><span><b>-{{ evolutionDelta.removedEdges.length }}</b> 未在当前观察到</span></div>
      </div>
      <div class="evolution-block">
        <header><strong>跨圈关键好友</strong><span>按跨圈连线数排序</span></header>
        <div class="bridge-list"><button v-for="item in bridgeRanking" :key="item.id" @click="emit('openFriend',item.id)"><strong>{{ item.displayName }}</strong><small>{{ item.crossEdges }} 条跨圈连接 · 连接 {{ item.communityCount }} 个朋友圈</small></button><p v-if="!bridgeRanking.length">当前没有观察到跨圈连接。</p></div>
      </div>
      <div class="evolution-block circle-compare">
        <header><strong>朋友圈对比</strong><span>比较两位主题好友的已观察邻居</span></header>
        <div><select v-model="compareCommunityLeft"><option :value="null">选择朋友圈</option><option v-for="item in communities" :key="`left-${item.id}`" :value="item.id">你与 {{ item.theme.displayName }}</option></select><span>和</span><select v-model="compareCommunityRight"><option :value="null">选择朋友圈</option><option v-for="item in communities" :key="`right-${item.id}`" :value="item.id">你与 {{ item.theme.displayName }}</option></select></div>
        <p v-if="communityComparison">共同邻居 {{ communityComparison.overlap.length }} 位 · 左侧独有 {{ communityComparison.leftOnly.length }} 位 · 右侧独有 {{ communityComparison.rightOnly.length }} 位</p><p v-else>选择两个不同的朋友圈进行已观察邻居交集比较。</p>
      </div>
      <small class="evolution-note">“减少”只表示当前本机快照没有再次观察到，不代表 VRChat 好友关系已经消失。</small>
    </section>

    <div class="path-toolbar">
      <strong>找关系路径</strong>
      <select v-model="pathStart"><option value="">选择起点好友</option><option v-for="node in visibleNodes" :key="`start-${node.id}`" :value="node.id">{{ node.displayName }}</option></select>
      <span>到</span>
      <select v-model="pathEnd"><option value="">选择终点好友</option><option v-for="node in visibleNodes" :key="`end-${node.id}`" :value="node.id">{{ node.displayName }}</option></select>
      <button type="button" :disabled="!pathStart || !pathEnd" @click="calculatePath">查找</button>
      <button v-if="path.length" type="button" @click="clearPath">清除</button>
      <small v-if="pathMessage">{{ pathMessage }}</small>
    </div>

    <div class="network-workspace">
      <div v-if="positioned.length" class="graph-stage" :class="{ dragging: interaction, focused: focusedID }" @pointerleave="clearTransientFocus">
        <div v-if="lockedFocusNode" class="graph-focus-lock">
          <span>已锁定 <b>{{ lockedFocusNode.displayName }}</b> 的直接关系</span>
          <button type="button" title="解除关系锁定" @click.stop="clearLockedFocus"><X :size="14" />解除</button>
        </div>
        <div class="graph-controls" aria-label="关系图视图控制">
          <button type="button" title="缩小" aria-label="缩小关系网" @click="setZoom(zoom / 1.25)"><Minus :size="16" /></button>
          <span>{{ Math.round(zoom * 100) }}%</span>
          <button type="button" title="放大" aria-label="放大关系网" @click="setZoom(zoom * 1.25)"><Plus :size="16" /></button>
          <button type="button" title="恢复自动布局" aria-label="恢复自动布局" @click="resetView"><RotateCcw :size="16" /></button>
          <button type="button" :title="fullscreen ? '退出全屏' : '全屏查看'" :aria-label="fullscreen ? '退出关系网全屏' : '全屏查看关系网'" @click="toggleFullscreen">
            <Minimize2 v-if="fullscreen" :size="16" /><Maximize2 v-else :size="16" />
          </button>
        </div>
        <div class="graph-hint"><Move :size="14" />拖动 · 滚轮缩放 · 单击锁定 · 双击档案</div>
        <svg ref="svgRef" :viewBox="`0 0 ${graphSize.width} ${graphSize.height}`" role="img" aria-label="共同好友关系图"
          @wheel.prevent="handleWheel" @pointerdown="beginPan" @pointermove="movePointer"
          @pointerup="endPointer" @pointercancel="endPointer">
          <g class="graph-viewport" :transform="`translate(${panX} ${panY}) scale(${zoom})`">
            <line v-for="edge in renderedEdges" :key="`${edge.source}-${edge.target}`" class="network-edge" :data-edge-key="networkEdgeKey(edge)"
              :x1="positions.get(edge.source)?.x" :y1="positions.get(edge.source)?.y"
              :x2="positions.get(edge.target)?.x" :y2="positions.get(edge.target)?.y"
            />
            <g v-for="node in renderedNodes" :key="node.id" class="graph-node"
              :data-node-id="node.id"
              :class="{
                'ambient-label': ambientLabelIDs.has(node.id),
                selected: selectedID === node.id,
                unscanned: !node.scanned,
                opted: node.optedOut,
                pinned: pinnedIDs.has(node.id),
                'path-node': pathIDs.has(node.id),
              }"
              :transform="`translate(${node.x} ${node.y})`" tabindex="0"
              @pointerdown="beginNodeDrag($event, node.id)" @pointerenter="enterNode(node.id)" @pointerleave="leaveNode(node.id)"
              @focus="enterNode(node.id)" @blur="leaveNode(node.id)" @click="selectNode(node.id)" @dblclick.stop="openNode(node.id)" @keydown.enter="selectNode(node.id)">
              <title>{{ node.displayName }} · {{ node.degree }} 条连接</title>
              <circle class="node-hit-target" :r="node.radius + 7" />
              <circle class="node-focus-halo" :r="node.radius + 9" />
              <circle class="node-ring" :r="node.radius + 5" />
              <circle class="node-body" :r="node.radius" :fill="node.scanned ? nodeColor(communityByID.get(node.id) ?? node.component) : 'var(--surface)'" />
              <image v-if="graphAvatarURL(node)" class="node-avatar" :href="graphAvatarURL(node)"
                :x="-node.radius + 1" :y="-node.radius + 1" :width="(node.radius - 1) * 2" :height="(node.radius - 1) * 2"
                preserveAspectRatio="xMidYMid slice" @error="markAvatarFailed(node.id)" />
              <text v-else class="node-initial" y="1">{{ node.displayName.slice(0, 1).toUpperCase() }}</text>
              <circle v-if="node.online" class="online-mark" :cx="node.radius * .72" :cy="node.radius * .72" r="4.5" />
              <text class="node-label" :y="-node.radius - 11">{{ node.displayName }}</text>
            </g>
          </g>
        </svg>
      </div>

      <div v-else class="network-empty">
        <Network :size="28" />
        <strong>当前筛选下没有可显示的连接</strong>
        <p v-if="hideUnconnected">孤立点默认隐藏。你可以添加指定好友，或临时开启“显示孤立点”。</p>
        <p v-else>先添加好友或扫描一批，关系数据返回后会在这里形成连线。</p>
      </div>

      <aside v-if="pickerOpen" class="friend-picker" aria-label="添加好友到关系网">
        <div class="picker-heading">
          <div><strong>添加到关系网</strong><small>最多选择 20 位；只保存在当前电脑</small></div>
          <button title="关闭" @click="pickerOpen = false"><X :size="16" /></button>
        </div>
        <label class="picker-search"><Search :size="15" /><input v-model="pickerQuery" autofocus placeholder="输入好友名称或 usr_ ID" /></label>
        <div class="picker-list">
          <button v-for="node in pickerCandidates" :key="node.id" class="picker-item" :class="{ selected: pickedIDs.has(node.id) }" @click="togglePicked(node.id)">
            <span class="picker-avatar">
              <img v-if="avatarURL(node)" :src="avatarURL(node)" alt="" @error="markAvatarFailed(node.id)" />
              <i v-else>{{ node.displayName.slice(0, 1).toUpperCase() }}</i>
            </span>
            <span><strong>{{ node.displayName }}</strong><small>{{ node.scanned ? `${degrees.get(node.id) ?? 0} 条连接` : '尚未扫描' }}</small></span>
            <i class="picker-check"><Check v-if="pickedIDs.has(node.id)" :size="14" /></i>
          </button>
        </div>
        <div class="picker-footer">
          <span>已选择 {{ pickedIDs.size }}/20</span>
          <button :disabled="!pickedIDs.size" @click="addPicked(false)">只加入画布</button>
          <button class="primary" :disabled="!pickedIDs.size || scanning" @click="addPicked(true)">加入并扫描</button>
        </div>
      </aside>
    </div>

    <footer class="network-boundary">
      <ShieldCheck :size="16" />
      <span>关系快照和变化记录只保存在本机 SQLite 中，不上传、不公开。扫描串行执行且可随时停止；VRChat 限流时会提前结束。</span>
    </footer>
  </article>
</template>

<style scoped>
.network-shell{grid-column:1/-1;padding:0;min-width:0;overflow:hidden;border-radius:8px;box-shadow:none}
.network-header{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:18px 20px;border-bottom:1px solid var(--line)}
.network-title{display:flex;align-items:center;gap:12px;min-width:0}.network-icon{width:36px;height:36px;display:grid;place-items:center;flex:none;border:1px solid var(--line);border-radius:7px;color:var(--ink-soft);background:var(--surface-muted)}
.network-title h2{margin:0;font-size:18px;letter-spacing:-.01em}.network-title p{margin:4px 0 0;color:var(--muted);font-size:12px}
.network-actions{display:flex;align-items:center;gap:8px}.network-actions button{min-height:36px;display:inline-flex;align-items:center;gap:7px;padding:0 12px;border-radius:7px;font-size:12px;font-weight:600;white-space:nowrap;cursor:pointer}
.secondary-action{border:1px solid var(--line-strong);color:var(--ink-soft);background:var(--surface)}.secondary-action:hover{background:var(--surface-hover);color:var(--ink)}
.scan-button{border:1px solid var(--accent);color:#fff;background:var(--accent)}.scan-button span{padding-left:7px;border-left:1px solid rgba(255,255,255,.28);font-weight:500}.scan-button.stop{border-color:var(--danger);background:var(--danger)}.scan-button:disabled{opacity:.48;cursor:not-allowed}
.network-summary-row{min-height:42px;display:flex;align-items:center;gap:20px;padding:0 20px;border-bottom:1px solid var(--line);color:var(--muted);font-size:11px}.network-summary-row b{color:var(--ink);font-size:12px}.scan-estimate{margin-left:auto;text-align:right}
.network-coverage-tip{padding:9px 20px;border-bottom:1px solid var(--line);background:var(--accent-soft);color:var(--ink-soft);font-size:9px;line-height:1.55;display:flex;align-items:flex-start;gap:7px}.network-coverage-tip svg{flex:none;color:var(--accent)}.scan-all-button{color:var(--accent)!important;border-color:color-mix(in srgb,var(--accent) 35%,var(--line))!important}
.scan-progress{display:grid;grid-template-columns:1fr auto;gap:6px 14px;padding:9px 20px;border-bottom:1px solid var(--line);color:var(--ink-soft);background:var(--surface-muted);font-size:12px}.scan-progress>div{grid-column:1/-1;height:2px;overflow:hidden;background:var(--line)}.scan-progress i{display:block;height:100%;background:var(--accent);transition:width .2s}
.community-strip{padding:10px 12px;border-bottom:1px solid var(--line);background:var(--surface)}.community-strip-heading{display:flex;align-items:center;gap:9px;margin-bottom:8px}.community-strip-heading strong{font-size:11px}.community-strip-heading span{color:var(--muted);font-size:9px}.community-list{display:flex;gap:7px;overflow-x:auto;padding-bottom:3px}.community-item{flex:none;display:grid;grid-template-columns:4px minmax(250px,300px) auto;align-items:stretch;gap:5px;padding:4px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted)}.community-item.active{border-color:var(--accent);background:var(--accent-soft)}.community-item.collapsed{opacity:.58}.community-item>i{width:4px;border-radius:3px}.community-main{height:44px;display:grid;grid-template-columns:34px minmax(0,1fr) auto;align-items:center;gap:8px;padding:0 8px;border:0;border-radius:5px;color:var(--ink);background:transparent;text-align:left;cursor:pointer}.community-main:hover{background:var(--surface-hover)}.community-avatar,.community-avatar img{width:32px;height:32px;border-radius:50%}.community-avatar{display:grid;place-items:center;overflow:hidden;color:var(--accent);background:var(--accent-soft)}.community-avatar img{object-fit:cover}.community-avatar b{font-size:11px}.community-copy{min-width:0}.community-copy strong,.community-copy small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.community-copy strong{font-size:10px}.community-copy small{margin-top:3px;color:var(--muted);font-size:8px}.community-main em{padding:3px 5px;border-radius:4px;color:var(--muted);background:var(--surface);font-size:8px;font-style:normal}.community-collapse{align-self:center;height:28px;padding:0 7px;border:1px solid var(--line);border-radius:5px;color:var(--muted);background:var(--surface);font-size:9px;cursor:pointer}.community-collapse:hover{color:var(--ink);border-color:var(--line-strong)}
.network-toolbar{display:flex;align-items:center;gap:7px;padding:10px 12px;border-bottom:1px solid var(--line);background:var(--surface-muted)}
.network-search{width:min(360px,32vw);display:flex;align-items:center;gap:8px;padding:0 10px;border:1px solid var(--line-strong);border-radius:6px;color:var(--muted);background:var(--surface)}.network-search input{width:100%;height:34px;border:0;outline:0;color:var(--ink);background:transparent;font-size:12px}
.network-toolbar>button,.network-toolbar>select{height:34px;display:inline-flex;align-items:center;gap:6px;padding:0 10px;border:1px solid var(--line);border-radius:6px;color:var(--ink-soft);background:var(--surface);font-size:11px;cursor:pointer}.network-toolbar>button:hover{border-color:var(--line-strong);color:var(--ink)}.network-toolbar>button.active{border-color:color-mix(in srgb,var(--accent) 42%,var(--line));color:var(--accent);background:var(--accent-soft)}.visible-count{margin-left:auto;color:var(--muted);font-size:11px}.graph-performance-note{display:flex;align-items:flex-start;gap:7px;padding:8px 12px;border-bottom:1px solid var(--line);color:var(--muted);background:var(--accent-soft);font-size:9px;line-height:1.5}.graph-performance-note svg{flex:none;color:var(--accent)}.graph-performance-note b{color:var(--ink-soft)}
.evolution-panel{display:grid;grid-template-columns:1fr 1fr 1fr;gap:8px;padding:10px 12px;border-bottom:1px solid var(--line);background:var(--surface-muted)}.evolution-block{min-width:0;padding:10px;border:1px solid var(--line);border-radius:7px;background:var(--surface)}.evolution-block header{display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:8px}.evolution-block header strong{font-size:10px}.evolution-block header span{color:var(--muted);font-size:8px}.evolution-block>select,.circle-compare select{width:100%;height:30px;padding:0 7px;border:1px solid var(--line);border-radius:5px;background:var(--surface-muted);color:var(--ink-soft);font-size:9px}.delta-grid{display:grid;grid-template-columns:1fr 1fr;gap:5px;margin-top:7px}.delta-grid span{padding:6px;border-radius:5px;background:var(--surface-muted);color:var(--muted);font-size:8px}.delta-grid b{color:var(--ink);font-size:10px}.bridge-list{display:grid;grid-template-columns:1fr 1fr;gap:4px}.bridge-list button{min-width:0;padding:6px;border:0;border-radius:5px;background:var(--surface-muted);color:var(--ink);text-align:left;cursor:pointer}.bridge-list strong,.bridge-list small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.bridge-list strong{font-size:9px}.bridge-list small{margin-top:2px;color:var(--muted);font-size:7px}.bridge-list p,.circle-compare p{margin:9px 0 0;color:var(--muted);font-size:8px}.circle-compare>div{display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:5px}.circle-compare>div>span{color:var(--muted);font-size:8px}.evolution-note{grid-column:1/-1;color:var(--muted);font-size:8px}
.path-toolbar{display:flex;align-items:center;gap:7px;padding:8px 12px;border-bottom:1px solid var(--line);background:var(--surface)}.path-toolbar strong{font-size:10px}.path-toolbar span,.path-toolbar small{color:var(--muted);font-size:9px}.path-toolbar select{min-width:150px;height:30px;padding:0 7px;border:1px solid var(--line);border-radius:5px;color:var(--ink-soft);background:var(--surface-muted);font-size:10px}.path-toolbar button{height:30px;padding:0 8px;border:1px solid var(--line-strong);border-radius:5px;color:var(--ink-soft);background:var(--surface);font-size:10px;cursor:pointer}.path-toolbar button:disabled{opacity:.45}.path-toolbar small{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.network-workspace{position:relative}.graph-stage{position:relative;height:min(68vh,760px);min-height:560px;overflow:hidden;background:var(--surface-muted)}.graph-stage>svg{width:100%;height:100%;touch-action:none;cursor:grab;user-select:none}.graph-stage.dragging>svg{cursor:grabbing}
.graph-controls{position:absolute;z-index:3;top:12px;right:12px;display:flex;align-items:center;padding:3px;border:1px solid var(--line);border-radius:7px;background:color-mix(in srgb,var(--surface) 94%,transparent);box-shadow:0 2px 8px rgba(0,0,0,.08)}.graph-controls button{width:32px;height:32px;display:grid;place-items:center;padding:0;border:0;border-radius:5px;color:var(--ink-soft);background:transparent;cursor:pointer}.graph-controls button:hover{color:var(--ink);background:var(--surface-hover)}.graph-controls span{min-width:48px;text-align:center;color:var(--muted);font-size:11px}
.graph-focus-lock{position:absolute;z-index:3;top:12px;left:12px;display:flex;align-items:center;gap:8px;padding:6px 7px 6px 10px;border:1px solid color-mix(in srgb,var(--accent) 40%,var(--line));border-radius:7px;color:var(--ink-soft);background:color-mix(in srgb,var(--surface) 94%,transparent);box-shadow:0 2px 8px rgba(0,0,0,.08);font-size:10px}.graph-focus-lock b{color:var(--accent)}.graph-focus-lock button{height:25px;display:inline-flex;align-items:center;gap:4px;padding:0 7px;border:0;border-radius:5px;color:var(--ink-soft);background:var(--surface-hover);font-size:9px;cursor:pointer}.graph-focus-lock button:hover{color:var(--accent)}
.graph-hint{position:absolute;z-index:2;left:12px;bottom:12px;display:flex;align-items:center;gap:6px;padding:7px 9px;border:1px solid var(--line);border-radius:6px;color:var(--muted);background:color-mix(in srgb,var(--surface) 92%,transparent);font-size:11px;pointer-events:none}
line{stroke:var(--line-strong);stroke-width:1.1;opacity:.2;vector-effect:non-scaling-stroke}.network-edge{pointer-events:none}.graph-stage.focused line{opacity:.055}.graph-stage.focused line.highlighted{stroke:var(--accent);stroke-width:1.9;opacity:.95}.graph-stage line.path-edge{stroke:var(--warning);stroke-width:2.5;opacity:1}.graph-node{cursor:pointer;outline:none;touch-action:none}.node-hit-target{fill:transparent;pointer-events:all}.node-focus-halo{fill:var(--accent);opacity:0;pointer-events:none}.graph-node.related .node-focus-halo{opacity:.13}.graph-node.hovered .node-focus-halo,.graph-node.selected .node-focus-halo{opacity:.28}.node-ring{fill:var(--surface);stroke:var(--line-strong);stroke-width:1;vector-effect:non-scaling-stroke;pointer-events:none}.node-body{stroke:var(--surface);stroke-width:2;vector-effect:non-scaling-stroke;pointer-events:none}.graph-node.related .node-ring{stroke:color-mix(in srgb,var(--accent) 72%,var(--line-strong));stroke-width:2.4}.graph-node.path-node .node-ring{stroke:var(--warning);stroke-width:3}.graph-node.unscanned .node-ring{stroke-dasharray:3 3}.graph-node.opted .node-body{fill:var(--surface-muted)!important}.graph-node.hovered .node-ring,.graph-node.selected .node-ring{stroke:var(--accent);stroke-width:3.5}.graph-node.pinned .node-ring{stroke:var(--warning)}.online-mark{fill:var(--success);stroke:var(--surface);stroke-width:2;pointer-events:none}.graph-node text{fill:var(--ink);font-size:11px;font-weight:650;text-anchor:middle;paint-order:stroke;stroke:var(--surface-muted);stroke-width:4px;stroke-linejoin:round;pointer-events:none}.graph-node.hovered .node-label{fill:var(--accent);font-weight:800}.node-avatar,.node-label{pointer-events:none}.node-label{display:none}.graph-stage:not(.focused) .graph-node.ambient-label .node-label,.graph-node.related-label .node-label{display:block}.node-initial{fill:#fff!important;stroke:none!important;font-size:11px!important;font-weight:750;dominant-baseline:middle}.graph-node.unscanned .node-initial,.graph-node.opted .node-initial{fill:var(--ink-soft)!important}.graph-viewport{transform-box:view-box;transform-origin:0 0}
.node-avatar{clip-path:circle(50%)}
.network-empty{height:min(68vh,760px);min-height:560px;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;color:var(--muted);background:var(--surface-muted)}.network-empty strong{margin-top:12px;color:var(--ink)}.network-empty p{max-width:460px;margin:7px 20px 0;font-size:12px;line-height:1.6}
.friend-picker{position:absolute;z-index:5;top:12px;right:12px;width:min(380px,calc(100% - 24px));max-height:calc(100% - 24px);display:flex;flex-direction:column;overflow:hidden;border:1px solid var(--line-strong);border-radius:8px;background:var(--surface);box-shadow:0 12px 36px rgba(0,0,0,.2)}
.picker-heading{display:flex;align-items:flex-start;justify-content:space-between;padding:14px;border-bottom:1px solid var(--line)}.picker-heading strong,.picker-heading small{display:block}.picker-heading strong{font-size:13px}.picker-heading small{margin-top:3px;color:var(--muted);font-size:10px}.picker-heading button{width:28px;height:28px;display:grid;place-items:center;border:0;border-radius:5px;color:var(--muted);background:transparent;cursor:pointer}.picker-heading button:hover{background:var(--surface-hover);color:var(--ink)}
.picker-search{display:flex;align-items:center;gap:7px;margin:10px;padding:0 9px;border:1px solid var(--line-strong);border-radius:6px;color:var(--muted)}.picker-search input{width:100%;height:34px;border:0;outline:0;color:var(--ink);background:transparent;font-size:12px}.picker-list{min-height:0;overflow:auto;padding:0 7px 7px}.picker-item{width:100%;display:grid;grid-template-columns:34px 1fr 22px;align-items:center;gap:9px;padding:7px;border:1px solid transparent;border-radius:6px;color:var(--ink);background:transparent;text-align:left;cursor:pointer}.picker-item:hover{background:var(--surface-hover)}.picker-item.selected{border-color:color-mix(in srgb,var(--accent) 38%,var(--line));background:var(--accent-soft)}.picker-avatar,.picker-avatar img{width:34px;height:34px;border-radius:6px}.picker-avatar{display:grid;place-items:center;overflow:hidden;color:var(--ink-soft);background:var(--surface-muted)}.picker-avatar img{object-fit:cover}.picker-avatar i{font-style:normal;font-weight:700}.picker-item>span:nth-child(2){min-width:0}.picker-item strong,.picker-item small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.picker-item strong{font-size:12px}.picker-item small{margin-top:2px;color:var(--muted);font-size:10px}.picker-check{width:18px;height:18px;display:grid;place-items:center;border:1px solid var(--line-strong);border-radius:4px;color:#fff}.picker-item.selected .picker-check{border-color:var(--accent);background:var(--accent)}
.picker-footer{display:grid;grid-template-columns:1fr auto auto;align-items:center;gap:7px;padding:10px;border-top:1px solid var(--line);background:var(--surface-muted)}.picker-footer span{color:var(--muted);font-size:10px}.picker-footer button{height:32px;padding:0 9px;border:1px solid var(--line-strong);border-radius:6px;color:var(--ink-soft);background:var(--surface);font-size:10px;font-weight:600;cursor:pointer}.picker-footer button.primary{border-color:var(--accent);color:#fff;background:var(--accent)}.picker-footer button:disabled{opacity:.45;cursor:not-allowed}
.network-boundary{display:flex;align-items:flex-start;gap:8px;padding:11px 20px;border-top:1px solid var(--line);color:var(--muted);font-size:10px;line-height:1.5}.network-boundary svg{flex:none;color:var(--success)}
.network-shell:fullscreen{width:100vw;height:100vh;overflow:auto;border:0;border-radius:0;background:var(--surface)}.network-shell:fullscreen .graph-stage,.network-shell:fullscreen .network-empty{height:calc(100vh - 310px);min-height:520px;max-height:none}.network-shell:fullscreen .network-boundary{display:none}
@media(prefers-reduced-motion:reduce){line,.graph-node,.scan-progress i{transition:none}}
@media(max-width:900px){.network-header{align-items:flex-start}.network-title p{display:none}.network-summary-row{gap:10px;flex-wrap:wrap;padding-top:8px;padding-bottom:8px}.scan-estimate{width:100%;margin-left:0;text-align:left}.network-toolbar,.path-toolbar{flex-wrap:wrap}.network-search{width:100%}.path-toolbar small{width:100%}.visible-count{margin-left:0}.evolution-panel{grid-template-columns:1fr}.graph-stage,.network-empty{min-height:520px;height:62vh}}
@media(max-width:620px){.network-header{flex-direction:column;align-items:stretch}.network-actions button{flex:1;justify-content:center}.network-summary-row span:nth-child(2){display:none}.network-toolbar>button{flex:1;justify-content:center}.visible-count{width:100%}.graph-controls{top:8px;right:8px}.graph-hint{display:none}.picker-footer{grid-template-columns:1fr 1fr}.picker-footer span{grid-column:1/-1}.network-boundary{padding:10px 12px}}
.graph-stage>svg{position:relative;z-index:1}.graph-edge-canvas{position:absolute;z-index:0;inset:0;width:100%;height:100%;pointer-events:none}
</style>
