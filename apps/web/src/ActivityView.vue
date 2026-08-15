<script setup lang="ts">
import { computed } from 'vue'
import { Activity, Clock3, Gamepad2, RefreshCw, Trash2, UserRound } from '@lucide/vue'
import type { ActivityEvent, ActivityInsights, Friend } from './api'
import { preferredFriendAvatar } from './media'

const props = defineProps<{
  events: ActivityEvent[]
  insights: ActivityInsights | null
  friends: Friend[]
  loading: boolean
  mediaUrl: (value?: string) => string
}>()
const emit = defineEmits<{ refresh: []; clear: []; selectUser: [userId: string] }>()

const weekdays = ['日', '一', '二', '三', '四', '五', '六']
const friendsByID = computed(() => new Map(props.friends.map((friend) => [friend.id, friend])))
const bucketMap = computed(() => new Map((props.insights?.heatmap ?? []).map((item) => [`${item.weekday}-${item.hour}`, item.count])))
const peak = computed(() => Math.max(1, ...(props.insights?.heatmap ?? []).map((item) => item.count)))
const gameEvents = computed(() => props.events.filter((event) => event.type.startsWith('game.')).length)

function friendFor(userId?: string) { return userId ? friendsByID.value.get(userId) : undefined }
function avatarFor(userId?: string) {
  const friend = friendFor(userId)
  return props.mediaUrl(preferredFriendAvatar(friend))
}
function nameFor(userId?: string, fallback?: string) { return friendFor(userId)?.displayName || fallback || (userId ? '未知好友' : 'VRChat') }
function titleFor(event: ActivityEvent) {
  const name = nameFor(event.userId, event.displayName)
  if (event.type.includes('friend-online')) return `${name} 上线`
  if (event.type.includes('friend-offline')) return `${name} 离线`
  if (event.type.includes('friend-location')) return `${name} 切换了世界`
  if (event.type.includes('friend-update')) return `${name} 更新了资料`
  if (event.type.includes('player-joined')) return `${name} 进入了当前实例`
  if (event.type.includes('player-left')) return `${name} 离开了当前实例`
  return event.summary
}
function opacity(day: number, hour: number) {
  const count = bucketMap.value.get(`${day}-${hour}`) ?? 0
  return count ? .22 + (count / peak.value) * .78 : .055
}
function time(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}
</script>

<template>
  <section class="activity-view">
    <header class="activity-toolbar">
      <div>
        <span class="panel-kicker">本机记录</span>
        <h2>活动与游戏日志</h2>
        <p>好友动态与游戏日志。</p>
      </div>
      <div class="toolbar-actions">
        <button :disabled="loading" @click="emit('refresh')"><RefreshCw :size="15" :class="{ spin: loading }" />刷新</button>
        <button class="danger" :disabled="!events.length" @click="emit('clear')"><Trash2 :size="15" />清空</button>
      </div>
    </header>

    <div class="summary-strip">
      <span><Activity :size="16" /><b>{{ insights?.totalEvents ?? 0 }}</b> 条事件</span>
      <span><Clock3 :size="16" /><b>{{ insights?.coverageDays ?? 0 }}</b> 个活跃日</span>
      <span><UserRound :size="16" /><b>{{ insights?.topUsers.length ?? 0 }}</b> 位常见好友</span>
      <span><Gamepad2 :size="16" /><b>{{ gameEvents }}</b> 条游戏日志</span>
    </div>

    <div class="activity-layout">
      <article class="timeline-card">
        <div class="card-title"><Clock3 :size="17" /><strong>最近事件</strong><span>最新 {{ events.length }} 条</span></div>
        <div class="timeline">
          <button v-for="event in events" :key="event.id" :disabled="!event.userId" @click="event.userId && emit('selectUser', event.userId)">
            <img v-if="avatarFor(event.userId)" :src="avatarFor(event.userId)" alt="" loading="lazy" />
            <span v-else class="event-icon"><Gamepad2 v-if="event.type.startsWith('game.')" :size="15" /><Activity v-else :size="15" /></span>
            <span class="event-copy"><strong>{{ titleFor(event) }}</strong><small>{{ event.worldId ? `${event.worldId} · ` : '' }}{{ time(event.observedAt) }}</small></span>
            <span class="event-source" :data-source="event.type.startsWith('game.') ? 'game' : 'pipeline'">{{ event.type.startsWith('game.') ? '游戏' : '实时' }}</span>
          </button>
          <div v-if="!events.length" class="activity-empty"><Clock3 :size="28" /><strong>还没有活动记录</strong><p>保持应用运行并启动 VRChat 后，这里会自动出现好友和游戏事件。</p></div>
        </div>
      </article>

      <aside class="activity-side">
        <article class="top-card">
          <div class="card-title"><UserRound :size="17" /><strong>常见好友</strong><span>按事件次数</span></div>
          <button v-for="item in insights?.topUsers" :key="item.userId" @click="emit('selectUser', item.userId)">
            <img v-if="avatarFor(item.userId)" :src="avatarFor(item.userId)" alt="" loading="lazy" />
            <span v-else class="avatar-fallback">{{ nameFor(item.userId, item.displayName).slice(0, 1) }}</span>
            <span><strong>{{ nameFor(item.userId, item.displayName) }}</strong><small>{{ item.userId }}</small></span>
            <b>{{ item.count }}</b>
          </button>
          <p v-if="!insights?.topUsers.length">好友事件累积后会在这里形成排行。</p>
        </article>

        <article class="heatmap-card">
          <div class="card-title"><Activity :size="17" /><strong>活跃分布</strong><span>星期 × 小时</span></div>
          <div class="heatmap">
            <div class="hour-labels"><i></i><i v-for="hour in 24" :key="hour">{{ (hour - 1) % 6 === 0 ? hour - 1 : '' }}</i></div>
            <div v-for="day in 7" :key="day" class="heat-row"><b>{{ weekdays[day - 1] }}</b><i v-for="hour in 24" :key="hour" :title="`周${weekdays[day - 1]} ${hour - 1}:00 · ${bucketMap.get(`${day - 1}-${hour - 1}`) ?? 0} 条`" :style="{ opacity: opacity(day - 1, hour - 1) }"></i></div>
          </div>
        </article>
      </aside>
    </div>
  </section>
</template>

<style scoped>
.activity-view{display:grid;gap:12px}.activity-toolbar{display:flex;justify-content:space-between;gap:20px;align-items:flex-end}.activity-toolbar h2{margin:3px 0 0;font-size:20px}.activity-toolbar p{margin:5px 0 0;color:var(--muted);font-size:10px}.toolbar-actions{display:flex;gap:7px}.activity-view button{border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--ink);cursor:pointer}.toolbar-actions button{height:36px;padding:0 10px;display:flex;align-items:center;gap:6px;font-size:10px}.activity-view button.danger{color:var(--danger)}.summary-strip{min-height:46px;padding:8px 12px;border:1px solid var(--line);border-radius:10px;background:var(--surface);display:flex;align-items:center;gap:8px;box-shadow:var(--shadow)}.summary-strip span{display:flex;align-items:center;gap:6px;padding:6px 11px;border-right:1px solid var(--line);color:var(--muted);font-size:10px}.summary-strip span:last-child{border-right:0}.summary-strip svg{color:var(--accent)}.summary-strip b{color:var(--ink);font-size:13px}.activity-layout{display:grid;grid-template-columns:minmax(0,1.7fr) minmax(300px,.8fr);gap:12px;align-items:start}.timeline-card,.top-card,.heatmap-card{border:1px solid var(--line);border-radius:11px;background:var(--surface);box-shadow:var(--shadow)}.timeline-card{padding:14px}.activity-side{display:grid;gap:12px}.top-card,.heatmap-card{padding:13px}.card-title{display:flex;align-items:center;gap:7px}.card-title>span{margin-left:auto;color:var(--muted);font-size:8px}.timeline{display:grid;gap:5px;margin-top:11px}.timeline>button{min-height:54px;padding:7px 9px;display:flex;align-items:center;gap:9px;text-align:left}.timeline img,.event-icon{width:36px;height:36px;border-radius:9px;flex:0 0 auto}.timeline img{object-fit:cover}.event-icon{display:grid;place-items:center;background:var(--surface-muted);color:var(--accent)}.event-copy{min-width:0;flex:1}.event-copy strong,.event-copy small{display:block;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.event-copy strong{font-size:10px}.event-copy small{margin-top:3px;color:var(--muted);font-size:8px}.event-source{padding:3px 6px;border-radius:999px;background:var(--surface-muted);color:var(--muted);font-size:8px}.event-source[data-source="game"]{color:#16a3c7;background:color-mix(in srgb,#16a3c7 12%,transparent)}.top-card>button{width:100%;margin-top:6px;padding:7px;display:grid;grid-template-columns:34px minmax(0,1fr) auto;align-items:center;gap:8px;text-align:left}.top-card img,.avatar-fallback{width:34px;height:34px;border-radius:9px;object-fit:cover}.avatar-fallback{display:grid;place-items:center;background:var(--surface-muted);color:var(--accent);font-weight:700}.top-card button span:nth-child(2){min-width:0}.top-card button strong,.top-card button small{display:block;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.top-card button strong{font-size:10px}.top-card button small{margin-top:2px;color:var(--muted);font-size:7px}.top-card button>b{color:var(--accent)}.top-card>p{color:var(--muted);font-size:9px}.heatmap{margin-top:12px;overflow:auto}.hour-labels,.heat-row{min-width:330px;display:grid;grid-template-columns:15px repeat(24,1fr);gap:2px}.hour-labels i{color:var(--muted);font-size:6px;font-style:normal;text-align:center}.heat-row{margin-top:2px}.heat-row b{font-size:7px;color:var(--muted);font-weight:500}.heat-row i{height:10px;border-radius:2px;background:var(--accent);opacity:.055}.activity-empty{padding:52px 20px;text-align:center;color:var(--muted)}.activity-empty strong{display:block;margin-top:10px;color:var(--ink)}.activity-empty p{font-size:9px}@media(max-width:980px){.activity-layout{grid-template-columns:1fr}.activity-side{grid-template-columns:1fr 1fr}}@media(max-width:650px){.activity-toolbar{display:block}.toolbar-actions{margin-top:10px}.summary-strip{overflow:auto}.summary-strip span{white-space:nowrap}.activity-side{grid-template-columns:1fr}}
</style>
