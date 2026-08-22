<script setup lang="ts">
import { computed } from 'vue'
import { Bell, ChevronRight, Clock3, Globe2, ListChecks, Radio, Users } from '@lucide/vue'
import type { ActivityEvent, Friend, FriendNetwork, VrcNotification, World } from './api'
import { preferredFriendAvatar } from './media'
import { buildDailyBrief } from './product-insights'

const props = defineProps<{
  friends: Friend[]
  worlds: World[]
  notifications: VrcNotification[]
  events: ActivityEvent[]
  network: FriendNetwork | null
  favoriteCount: number
  realtimeLabel: string
  routeLabel: string
  mediaUrl: (value?: string) => string
}>()
const emit = defineEmits<{ openFriend: [friend: Friend]; openWorld: [worldId: string]; selectView: [view: 'friends' | 'worlds' | 'activity' | 'notifications'] }>()

const joinable = computed(() => props.friends.filter((item) => item.online && item.location?.startsWith('wrld_')).slice(0, 8))
const worldClusters = computed(() => {
  const groups = new Map<string, Friend[]>()
  for (const friend of props.friends) {
    if (!friend.location?.startsWith('wrld_')) continue
    const world = friend.location.split(':')[0]
    const items = groups.get(world) ?? []
    items.push(friend); groups.set(world, items)
  }
  const names = new Map(props.worlds.map((world) => [world.id, world.name]))
  return [...groups.entries()].map(([worldId, friends]) => ({ worldId, worldName: names.get(worldId), friends }))
    .sort((a, b) => b.friends.length - a.friends.length || a.worldId.localeCompare(b.worldId)).slice(0, 5)
})
const pending = computed(() => props.notifications.filter((item) => !item.seen).slice(0, 5))
const recent = computed(() => props.events.slice(0, 6))
const brief = computed(() => buildDailyBrief(props.friends, props.worlds, props.notifications, props.events, props.network))
const friendById = computed(() => new Map(props.friends.map((friend) => [friend.id, friend])))
function openBrief(item: ReturnType<typeof buildDailyBrief>[number]) {
  if (item.kind === 'request') return emit('selectView', 'notifications')
  if (item.worldId) return emit('openWorld', item.worldId)
  const friend = item.userIds.map((id) => friendById.value.get(id)).find(Boolean)
  if (friend) emit('openFriend', friend)
  else emit('selectView', 'activity')
}
function avatar(friend: Friend) { return props.mediaUrl(preferredFriendAvatar(friend)) }
function shortTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('zh-CN',{hour:'2-digit',minute:'2-digit'}).format(date) }
</script>

<template>
  <section class="today-view wide-view">
    <div class="today-status">
      <div><span>现在可加入</span><strong>{{ joinable.length }}</strong><small>位好友公开了可见位置</small></div>
      <div><span>好友所在世界</span><strong>{{ worldClusters.length }}</strong><small>个当前可观察世界</small></div>
      <div><span>待处理</span><strong>{{ pending.length }}</strong><small>条未读通知与请求</small></div>
      <div><span>收藏世界</span><strong>{{ favoriteCount }}</strong><small>{{ realtimeLabel }} · {{ routeLabel }}</small></div>
    </div>
    <div class="today-columns">
      <article class="today-panel daily-brief"><header><div><ListChecks :size="17" /><strong>今日简报</strong></div><span>最多 5 条 · 本机生成 · 均可查证</span></header><button v-for="item in brief" :key="item.id" class="brief-row" @click="openBrief(item)"><span :data-kind="item.kind"></span><div><strong>{{ item.title }} <i>{{ item.confidence }}可信</i></strong><small>{{ item.summary }}</small><em>{{ item.evidence.join(' · ') }}</em></div><ChevronRight :size="14" /></button><p v-if="!brief.length" class="today-empty">当前没有需要优先处理的场景；继续积累本机记录后会自动生成。</p></article>
      <article class="today-panel primary-feed"><header><div><Users :size="17" /><strong>可以一起玩</strong></div><button @click="emit('selectView','friends')">全部好友 <ChevronRight :size="14" /></button></header><div v-if="joinable.length" class="joinable-list"><button v-for="friend in joinable" :key="friend.id" @click="emit('openFriend',friend)"><span><img v-if="avatar(friend)" :src="avatar(friend)" alt="" /><b v-else>{{ friend.displayName.slice(0,1) }}</b><i></i></span><div><strong>{{ friend.displayName }}</strong><small>{{ friend.statusDescription || '当前在线' }}</small></div><ChevronRight :size="15" /></button></div><p v-else class="today-empty">目前没有公开可见位置的在线好友。</p></article>
      <div class="today-side">
        <article class="today-panel"><header><div><Globe2 :size="17" /><strong>好友所在世界</strong></div><button @click="emit('selectView','worlds')">世界 <ChevronRight :size="14" /></button></header><button v-for="group in worldClusters" :key="group.worldId" class="compact-row" @click="emit('openFriend',group.friends[0])"><span><strong>{{ group.worldName || (group.friends.length > 1 ? `${group.friends.length} 位好友同世界` : `${group.friends[0].displayName} 所在世界`) }}</strong><small>{{ group.friends.length }} 位 · {{ group.friends.map(item=>item.displayName).slice(0,3).join('、') }}</small></span><ChevronRight :size="14" /></button><p v-if="!worldClusters.length" class="today-empty">还没有观察到好友所在的可见世界。</p></article>
        <article class="today-panel"><header><div><Bell :size="17" /><strong>等待处理</strong></div><button @click="emit('selectView','notifications')">通知 <ChevronRight :size="14" /></button></header><button v-for="item in pending" :key="item.id" class="compact-row" @click="emit('selectView','notifications')"><span><strong>{{ item.senderUsername || item.type }}</strong><small>{{ item.message || '等待你查看的 VRChat 通知' }}</small></span><ChevronRight :size="14" /></button><p v-if="!pending.length" class="today-empty">没有待处理通知。</p></article>
      </div>
      <article class="today-panel recent-panel"><header><div><Clock3 :size="17" /><strong>刚刚发生</strong></div><button @click="emit('selectView','activity')">历史 <ChevronRight :size="14" /></button></header><div v-for="event in recent" :key="event.id" class="event-row"><i></i><span><strong>{{ event.summary }}</strong><small>{{ shortTime(event.observedAt) }} · {{ event.type.startsWith('game.') ? '游戏日志' : '实时观察' }}</small></span></div><p v-if="!recent.length" class="today-empty">本机还没有活动记录。</p></article>
    </div>
    <div class="today-foot"><Radio :size="14" />数据来自当前好友快照、Pipeline 与本机游戏日志；缺失时段不会被推断。</div>
  </section>
</template>

<style scoped>
.today-view{display:grid;gap:10px}.today-status{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));border:1px solid var(--line);border-radius:8px;background:var(--surface);overflow:hidden}.today-status>div{min-width:0;padding:16px;border-right:1px solid var(--line)}.today-status>div:last-child{border-right:0}.today-status span,.today-status strong,.today-status small{display:block}.today-status span{color:var(--muted);font-size:9px}.today-status strong{margin:7px 0 4px;font-size:19px}.today-status small{color:var(--muted);font-size:9px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.today-columns{display:grid;grid-template-columns:1.15fr .85fr;gap:10px}.today-side{display:grid;gap:10px}.today-panel{border:1px solid var(--line);border-radius:8px;background:var(--surface);padding:13px}.daily-brief{grid-column:1/-1}.daily-brief>header>span{color:var(--muted);font-size:8px}.brief-row{width:100%;display:grid;grid-template-columns:8px 1fr auto;align-items:center;gap:9px;padding:9px 5px;border:0;border-top:1px solid var(--line);background:transparent;color:var(--ink);text-align:left;cursor:pointer}.brief-row>span{width:7px;height:7px;border-radius:50%;background:var(--accent)}.brief-row>span[data-kind=request]{background:var(--warning)}.brief-row>span[data-kind=gathering],.brief-row>span[data-kind=intersection]{background:var(--success)}.brief-row div{min-width:0}.brief-row strong,.brief-row small{display:block}.brief-row strong{font-size:10px}.brief-row small{margin-top:3px;color:var(--muted);font-size:8px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.today-panel>header{min-height:26px;display:flex;align-items:center;justify-content:space-between;margin-bottom:8px}.today-panel>header>div{display:flex;align-items:center;gap:7px}.today-panel header svg{color:var(--muted)}.today-panel header strong{font-size:11px}.today-panel header button{display:flex;align-items:center;gap:2px;border:0;background:transparent;color:var(--muted);font-size:9px;cursor:pointer}.joinable-list{display:grid;grid-template-columns:1fr 1fr;gap:6px}.joinable-list>button{min-width:0;display:grid;grid-template-columns:38px 1fr auto;align-items:center;gap:8px;padding:7px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted);color:var(--ink);text-align:left;cursor:pointer}.joinable-list>button>span,.joinable-list img,.joinable-list b{width:38px;height:38px;border-radius:8px}.joinable-list>button>span{position:relative}.joinable-list img{object-fit:cover}.joinable-list b{display:grid;place-items:center;background:var(--accent-soft)}.joinable-list i{width:9px;height:9px;position:absolute;right:-2px;bottom:-2px;border:2px solid var(--surface);border-radius:50%;background:var(--success)}.joinable-list div{min-width:0}.joinable-list strong,.joinable-list small,.compact-row strong,.compact-row small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.joinable-list strong,.compact-row strong{font-size:10px}.joinable-list small,.compact-row small{margin-top:3px;color:var(--muted);font-size:8px}.compact-row{width:100%;display:grid;grid-template-columns:1fr auto;align-items:center;gap:8px;padding:8px 4px;border:0;border-top:1px solid var(--line);background:transparent;color:var(--ink);text-align:left;cursor:pointer}.compact-row:first-of-type{border-top:0}.compact-row>span{min-width:0}.recent-panel{grid-column:1/-1}.event-row{display:grid;grid-template-columns:8px 1fr;align-items:center;gap:8px;padding:7px 3px;border-top:1px solid var(--line)}.event-row:first-of-type{border-top:0}.event-row i{width:6px;height:6px;border-radius:50%;background:var(--accent)}.event-row strong,.event-row small{display:block}.event-row strong{font-size:10px}.event-row small{margin-top:3px;color:var(--muted);font-size:8px}.today-empty{margin:18px 0;color:var(--muted);text-align:center;font-size:9px}.today-foot{display:flex;align-items:center;gap:6px;color:var(--muted);font-size:9px}@media(max-width:900px){.today-status{grid-template-columns:1fr 1fr}.today-status>div:nth-child(2){border-right:0}.today-status>div:nth-child(-n+2){border-bottom:1px solid var(--line)}.today-columns{grid-template-columns:1fr}}@media(max-width:560px){.joinable-list{grid-template-columns:1fr}.today-status>div{padding:12px}.today-status small{white-space:normal}}
.brief-row strong i{margin-left:5px;color:var(--accent);font-size:8px;font-style:normal}.brief-row em{display:block;margin-top:3px;color:var(--muted);font-size:8px;font-style:normal;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
</style>
