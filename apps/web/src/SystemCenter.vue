<script setup lang="ts">
import { Database, HardDrive, Image, Radio, RefreshCw, Trash2 } from '@lucide/vue'
import type { CacheStats, Diagnostics, NetworkState, RealtimeStatus } from './api'

defineProps<{
  diagnostics: Diagnostics | null
  realtime: RealtimeStatus
  network: NetworkState
  cache: CacheStats | null
  loading: boolean
}>()
const emit = defineEmits<{ refresh: []; clearMedia: []; clearEntities: [] }>()

function bytes(value = 0) {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`
  return `${(value / 1024 / 1024).toFixed(1)} MiB`
}
</script>

<template>
  <section class="system-center">
    <div class="system-title"><div><span class="panel-kicker">分层状态</span><h3>系统与缓存中心</h3></div><button :disabled="loading" @click="emit('refresh')"><RefreshCw :size="15" :class="{ spin: loading }" />重新检测</button></div>
    <div class="layer-grid">
      <div><Database :size="17" /><span>本地数据库</span><strong :data-state="diagnostics?.database.ready ? 'ok' : 'error'">{{ diagnostics?.database.ready ? '可用' : '异常' }}</strong></div>
      <div><Radio :size="17" /><span>实时好友动态</span><strong :data-state="realtime.state === 'connected' ? 'ok' : 'degraded'">{{ realtime.state === 'connected' ? '已连接' : realtime.state === 'connecting' ? '连接中' : '未连接' }}</strong></div>
      <div><HardDrive :size="17" /><span>网络出口</span><strong>{{ network.label }}</strong></div>
      <div><Image :size="17" /><span>图片缓存</span><strong>{{ cache?.mediaFiles ?? 0 }} 个 · {{ bytes(cache?.mediaBytes) }}</strong></div>
    </div>
    <div class="probe-list">
      <div v-for="check in diagnostics?.checks" :key="check.name"><span :data-state="check.state"></span><strong>{{ check.name }}</strong><small>{{ check.detail }}</small><em>{{ check.latencyMs }}ms</em></div>
    </div>
    <div class="cache-row">
      <div><strong>{{ bytes(cache?.databaseBytes) }}</strong><small>SQLite 数据库 · {{ cache?.annotationCount ?? 0 }} 条本机备注</small></div>
      <div><strong>{{ bytes(cache?.entityBytes) }}</strong><small>{{ cache?.entityEntries ?? 0 }} 个实体快照</small></div>
      <button :disabled="!cache?.mediaFiles || loading" @click="emit('clearMedia')"><Trash2 :size="14" />清理图片缓存</button>
    </div>
    <div class="cache-breakdown"><span>世界 {{ cache?.worldEntries ?? 0 }}</span><span>群组 {{ cache?.groupEntries ?? 0 }}</span><span>头像 {{ cache?.avatarEntries ?? 0 }}</span></div>
    <p>清理不会影响登录、备注、标签和关系网。</p>
  </section>
</template>

<style scoped>
.system-center{padding:14px;border:1px solid var(--line);border-radius:10px;background:var(--surface-muted)}.system-title{display:flex;align-items:center;justify-content:space-between}.system-title h3{margin:3px 0 0;font-size:14px}.system-title button,.cache-row button{display:flex;align-items:center;gap:6px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--ink-soft);padding:7px 9px;font-size:10px}.layer-grid{display:grid;grid-template-columns:1fr 1fr;gap:7px;margin-top:12px}.layer-grid>div{padding:9px;background:var(--surface);border:1px solid var(--line);border-radius:8px;display:grid;grid-template-columns:22px 1fr;align-items:center}.layer-grid svg{grid-row:1/3;color:var(--accent)}.layer-grid span{color:var(--muted);font-size:8px}.layer-grid strong{margin-top:2px;font-size:9px}.layer-grid strong[data-state=ok]{color:var(--success)}.layer-grid strong[data-state=degraded],.probe-list span[data-state=degraded]{color:var(--warning)}.layer-grid strong[data-state=error]{color:var(--danger)}.probe-list{display:grid;gap:5px;margin-top:9px}.probe-list>div{display:grid;grid-template-columns:7px 1fr minmax(0,2fr) auto;align-items:center;gap:7px;font-size:8px}.probe-list span{width:7px;height:7px;border-radius:50%;background:var(--muted)}.probe-list span[data-state=ok]{background:var(--success)}.probe-list span[data-state=degraded]{background:var(--warning)}.probe-list span[data-state=error]{background:var(--danger)}.probe-list small{color:var(--muted);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.probe-list em{color:var(--muted);font-style:normal}.cache-row{display:grid;grid-template-columns:1fr 1fr auto;gap:8px;align-items:center;margin-top:12px;padding-top:12px;border-top:1px solid var(--line)}.cache-row strong,.cache-row small{display:block}.cache-row strong{font-size:11px}.cache-row small{margin-top:2px;color:var(--muted);font-size:8px}.system-center>p{margin:9px 0 0;color:var(--muted);font-size:8px;line-height:1.5}@media(max-width:560px){.layer-grid{grid-template-columns:1fr}.cache-row{grid-template-columns:1fr 1fr}.cache-row button{grid-column:1/-1;justify-content:center}}
.cache-breakdown{display:flex;align-items:center;gap:6px;margin-top:8px}.cache-breakdown span{padding:4px 6px;border-radius:4px;background:var(--surface);color:var(--muted);font-size:8px}.cache-breakdown button{margin-left:auto;border:0;background:transparent;color:var(--danger);font-size:8px}
</style>
