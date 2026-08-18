<script setup lang="ts">
import { computed, ref } from 'vue'
import { AlertTriangle, Focus, Network, RotateCcw, Sparkles, X, ZoomIn, ZoomOut } from '@lucide/vue'
import type { ActivityEvent, Group, GroupMember, MutualFriend, UserProfile } from './api'
import { buildStrangerGraph, relationshipInsights, type StrangerGraphNode } from './stranger-network'

const props = defineProps<{
  self:{ id:string; displayName:string; imageUrl?:string }
  target:UserProfile
  mutuals:MutualFriend[]
  groups:Group[]
  members:GroupMember[]
  events:ActivityEvent[]
  mediaUrl:(value?:string)=>string
}>()
const emit = defineEmits<{ openUser:[userId:string] }>()
const hoveredID = ref('')
const selectedID = ref('')
const zoom = ref(1)
const showCandidates = ref(true)
const graph = computed(() => buildStrangerGraph(props))
const insights = computed(() => relationshipInsights(props))
const visibleNodes = computed(() => showCandidates.value ? graph.value.nodes : graph.value.nodes.filter(item=>item.kind!=='candidate'))
const visibleIDs = computed(() => new Set(visibleNodes.value.map(item=>item.id)))
const visibleEdges = computed(() => graph.value.edges.filter(item=>visibleIDs.value.has(item.source)&&visibleIDs.value.has(item.target)))
const focusedID = computed(() => hoveredID.value || selectedID.value)
const focusedNode = computed(() => graph.value.nodes.find(item=>item.id===focusedID.value))
const neighborIDs = computed(() => {
  const ids = new Set<string>(focusedID.value ? [focusedID.value] : [])
  for (const edge of visibleEdges.value) if (edge.source===focusedID.value) ids.add(edge.target); else if (edge.target===focusedID.value) ids.add(edge.source)
  return ids
})
function nodeRadius(node:StrangerGraphNode) { return node.kind==='target'?31:node.kind==='self'?27:node.kind==='group'?23:18 }
function nodeClass(node:StrangerGraphNode) { return { selected:selectedID.value===node.id, dimmed:focusedID.value&&!neighborIDs.value.has(node.id), [node.kind]:true } }
function edgeClass(edge:{source:string;target:string;kind:string}) { return { active:focusedID.value&&(edge.source===focusedID.value||edge.target===focusedID.value), dimmed:focusedID.value&&edge.source!==focusedID.value&&edge.target!==focusedID.value, [edge.kind]:true } }
function selectNode(node:StrangerGraphNode) { selectedID.value=node.id; if (node.kind==='target'||node.kind==='mutual'||node.kind==='candidate') emit('openUser',node.id) }
function reset() { selectedID.value=''; hoveredID.value=''; zoom.value=1 }
function toneLabel(value:string) { return value==='strong'?'关系信号':value==='caution'?'边界提醒':'参考判断' }
</script>

<template>
  <article class="relationship-lab">
    <header><div><Network :size="17" /><span><strong>复杂关系网</strong><small>实线为接口确认关系，虚线为本机观察线索</small></span></div><div class="graph-actions"><label><input v-model="showCandidates" type="checkbox" />群组候选</label><button @click="zoom=Math.max(.7,zoom-.1)"><ZoomOut :size="14" /></button><span>{{ Math.round(zoom*100) }}%</span><button @click="zoom=Math.min(1.5,zoom+.1)"><ZoomIn :size="14" /></button><button @click="reset"><RotateCcw :size="14" />重置</button></div></header>
    <div class="lab-grid">
      <div class="graph-canvas" @click.self="selectedID=''">
        <div v-if="selectedID && focusedNode" class="focus-chip"><Focus :size="13" /><span>已锁定 <b>{{ focusedNode.label }}</b></span><button @click="selectedID=''"><X :size="12" /></button></div>
        <svg viewBox="0 0 1000 580" role="img" aria-label="陌生人与自己的复杂关系图">
          <defs><clipPath v-for="node in visibleNodes" :id="`stranger-clip-${node.id}`" :key="`clip-${node.id}`"><circle :cx="node.x" :cy="node.y" :r="nodeRadius(node)-3" /></clipPath></defs>
          <g :transform="`translate(${500-500*zoom} ${290-290*zoom}) scale(${zoom})`">
            <g class="edges"><line v-for="(edge,index) in visibleEdges" :key="`${edge.source}-${edge.target}-${index}`" :x1="graph.nodes.find(item=>item.id===edge.source)?.x" :y1="graph.nodes.find(item=>item.id===edge.source)?.y" :x2="graph.nodes.find(item=>item.id===edge.target)?.x" :y2="graph.nodes.find(item=>item.id===edge.target)?.y" :class="edgeClass(edge)" /></g>
            <g v-for="node in visibleNodes" :key="node.id" class="relation-node" :class="nodeClass(node)" @mouseenter="hoveredID=node.id" @mouseleave="hoveredID=''" @click.stop="selectNode(node)">
              <circle class="node-halo" :cx="node.x" :cy="node.y" :r="nodeRadius(node)+4" /><circle class="node-body" :cx="node.x" :cy="node.y" :r="nodeRadius(node)" />
              <image v-if="node.imageUrl" :href="mediaUrl(node.imageUrl)" :x="node.x-nodeRadius(node)+3" :y="node.y-nodeRadius(node)+3" :width="nodeRadius(node)*2-6" :height="nodeRadius(node)*2-6" preserveAspectRatio="xMidYMid slice" :clip-path="`url(#stranger-clip-${node.id})`" />
              <text v-else :x="node.x" :y="node.y+4" text-anchor="middle">{{ node.kind==='group'?'群':node.label.slice(0,1) }}</text><text class="node-label" :x="node.x" :y="node.y+nodeRadius(node)+15" text-anchor="middle">{{ node.label }}</text>
            </g>
          </g>
        </svg>
        <div class="graph-legend"><span><i class="self"></i>自己</span><span><i class="target"></i>目标</span><span><i class="mutual"></i>共同好友</span><span><i class="group"></i>公开群组</span><span><i class="candidate"></i>圈外候选</span></div>
        <div v-if="graph.hiddenNodes" class="hidden-note">为保证流畅，图中省略 {{ graph.hiddenNodes }} 个低优先级节点</div>
      </div>
      <aside class="node-inspector"><template v-if="focusedNode"><span class="kind-label">{{ focusedNode.kind }}</span><h3>{{ focusedNode.label }}</h3><p>{{ focusedNode.detail }}</p><dl><div><dt>直接连线</dt><dd>{{ visibleEdges.filter(edge=>edge.source===focusedNode?.id||edge.target===focusedNode?.id).length }}</dd></div><div><dt>节点类型</dt><dd>{{ focusedNode.kind }}</dd></div><div><dt>证据性质</dt><dd>{{ visibleEdges.some(edge=>(edge.source===focusedNode?.id||edge.target===focusedNode?.id)&&edge.evidence==='observed')?'含本机观察':'接口确认' }}</dd></div></dl></template><template v-else><Network :size="24" /><h3>悬停预览，点击锁定</h3><p>选择节点后只突出它的一跳关系；点击人物节点可以继续打开完整档案。</p></template></aside>
    </div>
    <section class="inference-section"><header><div><Sparkles :size="16" /><strong>基于现有证据的关系推测</strong></div><span>推测仅供参考，不判断亲疏、敌友或现实身份</span></header><div class="inference-grid"><article v-for="item in insights" :key="item.id" :data-tone="item.tone"><div><span>{{ toneLabel(item.tone) }}</span><b>可信度 {{ item.confidence }}</b></div><h4>{{ item.title }}</h4><p>{{ item.summary }}</p><ul><li v-for="evidence in item.evidence" :key="evidence">{{ evidence }}</li></ul></article></div><div class="inference-warning"><AlertTriangle :size="14" />图中“群组成员”只说明成员列表可见；除共同好友接口明确返回的节点外，不推断两个人互为好友。</div></section>
  </article>
</template>

<style scoped>
.relationship-lab{margin-top:12px;border:1px solid var(--line);border-radius:9px;background:var(--surface);overflow:hidden}.relationship-lab>header,.inference-section>header{display:flex;align-items:center;justify-content:space-between;gap:10px;padding:10px 12px;border-bottom:1px solid var(--line)}.relationship-lab>header>div:first-child,.inference-section>header>div{display:flex;align-items:center;gap:7px}.relationship-lab>header svg,.inference-section>header svg{color:var(--accent)}.relationship-lab>header span span,.relationship-lab>header strong,.relationship-lab>header small{display:block}.relationship-lab>header strong{font-size:11px}.relationship-lab>header small{margin-top:2px;color:var(--muted);font-size:8px}.graph-actions{display:flex;align-items:center;gap:4px}.graph-actions label,.graph-actions button,.graph-actions>span{height:28px;display:flex;align-items:center;gap:4px;padding:0 7px;border:1px solid var(--line);border-radius:5px;background:var(--surface-muted);color:var(--ink-soft);font-size:8px}.graph-actions button{cursor:pointer}.graph-actions>span{min-width:42px;justify-content:center}.lab-grid{display:grid;grid-template-columns:minmax(0,1fr) 190px}.graph-canvas{position:relative;min-height:540px;overflow:hidden;background:radial-gradient(circle at 47% 50%,color-mix(in srgb,var(--accent) 7%,transparent),transparent 40%),var(--surface-muted)}.graph-canvas svg{width:100%;height:540px}.edges line{stroke:var(--line-strong);stroke-width:1;opacity:.58;transition:opacity .15s,stroke-width .15s}.edges line.membership{stroke:color-mix(in srgb,#9a6bda 55%,var(--line))}.edges line.candidate{stroke:color-mix(in srgb,#c38b45 45%,var(--line))}.edges line.co_presence{stroke:var(--accent);stroke-dasharray:5 5}.edges line.active{stroke:var(--accent);stroke-width:2.2;opacity:1}.edges line.dimmed{opacity:.06}.relation-node{cursor:pointer;transition:opacity .15s}.relation-node.dimmed{opacity:.12}.node-halo{fill:var(--surface);stroke:var(--line);stroke-width:2}.node-body{fill:#78848f}.relation-node.self .node-halo{stroke:#4f8bc9}.relation-node.target .node-halo{stroke:var(--accent);stroke-width:4}.relation-node.mutual .node-body{fill:#5f8b72}.relation-node.group .node-body{fill:#8569a9}.relation-node.candidate .node-body{fill:#a77b50}.relation-node.selected .node-halo{stroke:var(--accent);stroke-width:5}.relation-node text{fill:#fff;font-size:10px;font-weight:700;pointer-events:none}.relation-node .node-label{fill:var(--ink-soft);font-size:8px;font-weight:650;paint-order:stroke;stroke:var(--surface-muted);stroke-width:4px}.focus-chip{position:absolute;z-index:2;top:10px;left:10px;display:flex;align-items:center;gap:6px;padding:6px 7px;border:1px solid var(--line);border-radius:6px;background:color-mix(in srgb,var(--surface) 94%,transparent);font-size:8px}.focus-chip svg,.focus-chip b{color:var(--accent)}.focus-chip button{display:grid;place-items:center;padding:0;border:0;background:transparent;color:var(--muted);cursor:pointer}.graph-legend{position:absolute;left:10px;bottom:10px;display:flex;flex-wrap:wrap;gap:8px;padding:6px 8px;border:1px solid var(--line);border-radius:6px;background:color-mix(in srgb,var(--surface) 94%,transparent)}.graph-legend span{display:flex;align-items:center;gap:4px;color:var(--muted);font-size:7px}.graph-legend i{width:8px;height:8px;border-radius:50%;background:#78848f}.graph-legend i.self{background:#4f8bc9}.graph-legend i.target{background:var(--accent)}.graph-legend i.mutual{background:#5f8b72}.graph-legend i.group{background:#8569a9}.graph-legend i.candidate{background:#a77b50}.hidden-note{position:absolute;right:10px;bottom:10px;color:var(--muted);font-size:7px}.node-inspector{padding:14px;border-left:1px solid var(--line);background:var(--surface)}.node-inspector>svg{color:var(--muted)}.node-inspector .kind-label{color:var(--accent);font-size:8px;text-transform:uppercase}.node-inspector h3{margin:6px 0;font-size:14px}.node-inspector p{color:var(--muted);font-size:9px;line-height:1.55}.node-inspector dl{margin:12px 0}.node-inspector dl>div{display:flex;justify-content:space-between;padding:6px 0;border-top:1px dashed var(--line)}.node-inspector dt,.node-inspector dd{margin:0;font-size:8px}.node-inspector dt{color:var(--muted)}.inference-section{border-top:1px solid var(--line)}.inference-section>header>span{color:var(--muted);font-size:8px}.inference-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:7px;padding:10px}.inference-grid article{padding:10px;border:1px solid var(--line);border-left:3px solid #7d8790;border-radius:7px;background:var(--surface-muted)}.inference-grid article[data-tone=strong]{border-left-color:var(--accent)}.inference-grid article[data-tone=caution]{border-left-color:#c28a43}.inference-grid article>div{display:flex;justify-content:space-between}.inference-grid article>div span,.inference-grid article>div b{font-size:7px}.inference-grid article>div span{color:var(--muted)}.inference-grid article>div b{color:var(--accent)}.inference-grid h4{margin:7px 0 4px;font-size:10px}.inference-grid p{margin:0;color:var(--ink-soft);font-size:8px;line-height:1.55}.inference-grid ul{margin:7px 0 0;padding-left:14px;color:var(--muted);font-size:7px;line-height:1.5}.inference-warning{display:flex;align-items:flex-start;gap:6px;margin:0 10px 10px;padding:8px;border-radius:6px;background:color-mix(in srgb,#c28a43 9%,var(--surface));color:var(--muted);font-size:8px;line-height:1.5}.inference-warning svg{flex:none;color:#c28a43}@media(max-width:1000px){.lab-grid{grid-template-columns:1fr}.node-inspector{border-left:0;border-top:1px solid var(--line)}.inference-grid{grid-template-columns:repeat(2,1fr)}}@media(max-width:700px){.relationship-lab>header{align-items:flex-start;flex-direction:column}.graph-actions{flex-wrap:wrap}.inference-grid{grid-template-columns:1fr}}
</style>
