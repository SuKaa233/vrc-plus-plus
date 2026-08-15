<script setup lang="ts">
import { Bell, Check, Eye, EyeOff, MapPin, RefreshCw, UserPlus } from '@lucide/vue'
import type { VrcNotification } from './api'

defineProps<{ items: VrcNotification[]; loading: boolean; actingId: string }>()
const emit = defineEmits<{ refresh: []; action: [item: VrcNotification, action: 'see' | 'hide' | 'accept']; openWorld: [worldId: string, location: string] }>()
function time(value?: string) {
  return value ? new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : '时间未知'
}
function label(type: string) {
  if (type === 'friendRequest') return '好友请求'
  if (type === 'invite') return '世界邀请'
  if (type === 'requestInvite') return '加入请求'
  return type || 'VRChat 通知'
}
</script>

<template>
<section class="notification-center">
  <header><div><span class="panel-kicker">VRChat 通知</span><h2>通知与请求</h2><p>由你决定如何处理。</p></div><button :disabled="loading" @click="emit('refresh')"><RefreshCw :size="15" :class="{ spin: loading }" />刷新</button></header>
  <div class="notification-list">
    <article v-for="item in items" :key="item.id" :class="{ unseen: !item.seen }"><div class="notification-icon"><UserPlus v-if="item.type === 'friendRequest'" :size="18" /><MapPin v-else-if="item.worldId" :size="18" /><Bell v-else :size="18" /></div><div class="notification-copy"><div><span>{{ label(item.type) }}</span><time>{{ time(item.createdAt) }}</time></div><strong>{{ item.senderUsername || item.senderUserId || 'VRChat' }}</strong><p>{{ item.message || (item.worldName ? `邀请你前往 ${item.worldName}` : '没有附加消息') }}</p><button v-if="item.worldId" class="world-target" @click="emit('openWorld', item.worldId, item.instanceId ? `${item.worldId}:${item.instanceId}` : '')"><MapPin :size="13" />{{ item.worldName || item.worldId }}</button></div><div class="notification-actions"><button v-if="item.type === 'friendRequest'" :disabled="actingId === item.id" class="accept" @click="emit('action', item, 'accept')"><Check :size="14" />接受</button><button v-if="!item.seen" :disabled="actingId === item.id" @click="emit('action', item, 'see')"><Eye :size="14" />已读</button><button :disabled="actingId === item.id" @click="emit('action', item, 'hide')"><EyeOff :size="14" />隐藏</button></div></article>
    <div v-if="!items.length" class="notification-empty"><Bell :size="24" /><strong>没有待处理通知</strong><p>这里读取 VRChat 通知，不会在后台自动接受或拒绝。</p></div>
  </div>
</section>
</template>

<style scoped>
.notification-center{display:grid;gap:14px}.notification-center>header{display:flex;justify-content:space-between;gap:20px}.notification-center h2{margin:3px 0;font-size:20px}.notification-center header p{margin:5px 0;color:var(--muted);font-size:9px}.notification-center header>button{height:36px;padding:0 10px;border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--ink);display:flex;align-items:center;gap:6px}.notification-list{display:grid;gap:8px}.notification-list article{position:relative;padding:13px;border:1px solid var(--line);border-radius:10px;background:var(--surface);display:grid;grid-template-columns:38px 1fr auto;gap:11px;box-shadow:var(--shadow)}.notification-list article.unseen{border-left:3px solid var(--accent)}.notification-icon{width:38px;height:38px;border-radius:9px;background:var(--accent-soft);color:var(--accent);display:grid;place-items:center}.notification-copy{min-width:0}.notification-copy>div{display:flex;gap:8px;align-items:center}.notification-copy span{color:var(--accent);font-size:8px;font-weight:600}.notification-copy time{color:var(--muted);font-size:8px}.notification-copy>strong{display:block;margin-top:3px;font-size:11px}.notification-copy p{margin:4px 0;color:var(--ink-soft);font-size:9px}.world-target{padding:4px 7px!important;color:var(--accent)!important}.notification-actions{display:flex;align-items:center;gap:5px}.notification-actions button,.world-target{border:1px solid var(--line);border-radius:7px;background:var(--surface-muted);color:var(--ink);display:flex;align-items:center;gap:4px;padding:7px;font-size:8px}.notification-actions .accept{border-color:var(--accent);background:var(--accent);color:white}.notification-empty{padding:54px;border:1px dashed var(--line);border-radius:10px;color:var(--muted);display:grid;place-items:center;text-align:center}.notification-empty strong{margin-top:8px;color:var(--ink)}.notification-empty p{font-size:9px}@media(max-width:680px){.notification-center>header{display:block}.notification-center header>button{margin-top:10px}.notification-list article{grid-template-columns:38px 1fr}.notification-actions{grid-column:1/-1;justify-content:flex-end}}
</style>
