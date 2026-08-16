<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { GitCompare, History, ScanSearch } from '@lucide/vue'
import type { FriendNetwork } from './api'
import { compareNetworkSnapshots, createNetworkSnapshot, type NetworkSnapshot } from './product-insights'

const props = defineProps<{ network: FriendNetwork | null; storageKey: string }>()
const key = computed(() => `vrc-harbor-network-history:${props.storageKey}`)
const snapshots = ref<NetworkSnapshot[]>([])

function read(): NetworkSnapshot[] {
  try {
    const value = JSON.parse(localStorage.getItem(key.value) || '[]')
    return Array.isArray(value) ? value : []
  } catch { return [] }
}

watch(() => [props.network, key.value] as const, ([network]) => {
  if (!network?.nodes.length) { snapshots.value = read(); return }
  const current = createNetworkSnapshot(network)
  const values = read().filter((item) => item.date !== current.date)
  values.push(current)
  snapshots.value = values.slice(-30)
  try { localStorage.setItem(key.value, JSON.stringify(snapshots.value)) } catch { /* in-memory comparison still works */ }
}, { immediate: true, deep: true })

const delta = computed(() => snapshots.value.length > 1 ? compareNetworkSnapshots(snapshots.value.at(-2)!, snapshots.value.at(-1)!) : null)
</script>

<template>
  <section v-if="network" class="network-evolution panel wide-view">
    <header><div><span class="panel-kicker">本机快照</span><h2>朋友圈演变</h2></div><span>{{ snapshots.length }} 个自然日 · 最多保留 30 天</span></header>
    <div class="evolution-grid">
      <div><History :size="16"/><span>当前关系图<strong>{{ network.nodes.length }} 人 · {{ network.edges.length }} 条已观察连线</strong></span></div>
      <div><ScanSearch :size="16"/><span>扫描覆盖<strong>{{ network.scannedCount }} / {{ network.totalFriends }}</strong></span></div>
      <div v-if="delta"><GitCompare :size="16"/><span>相对上一快照<strong>+{{ delta.addedNodes }} 人 / +{{ delta.addedEdges }} 连线</strong><small v-if="delta.removedNodes || delta.missingEdges">{{ delta.removedNodes }} 人、{{ delta.missingEdges }} 条连线本次未观察到，不等于关系消失</small></span></div>
      <div v-else><GitCompare :size="16"/><span>等待对比<strong>下一个自然日生成变化摘要</strong></span></div>
    </div>
  </section>
</template>

<style scoped>
.network-evolution{padding:0;overflow:hidden}.network-evolution>header{display:flex;align-items:flex-start;justify-content:space-between;padding:13px 16px;border-bottom:1px solid var(--line)}header h2{margin:3px 0 0;font-size:14px}header>span{color:var(--muted);font-size:8px}.evolution-grid{display:grid;grid-template-columns:repeat(3,1fr)}.evolution-grid>div{min-width:0;padding:12px 14px;border-right:1px solid var(--line);display:grid;grid-template-columns:22px 1fr;gap:7px;align-items:start}.evolution-grid>div:last-child{border-right:0}.evolution-grid svg{color:var(--accent)}.evolution-grid span,.evolution-grid strong,.evolution-grid small{display:block}.evolution-grid span{color:var(--muted);font-size:8px}.evolution-grid strong{margin-top:4px;color:var(--ink);font-size:10px}.evolution-grid small{margin-top:4px;line-height:1.4}@media(max-width:760px){.evolution-grid{grid-template-columns:1fr}.evolution-grid>div{border-right:0;border-top:1px solid var(--line)}.evolution-grid>div:first-child{border-top:0}}
</style>
