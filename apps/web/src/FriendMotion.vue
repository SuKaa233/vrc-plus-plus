<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Clock3, Eye, EyeOff, GitCompare, GitMerge, MapPin, MoveRight, Radio, UsersRound } from '@lucide/vue'
import type { ActivityEvent, Friend, FriendNetwork, World } from './api'
import { buildFriendMotionOverview, type FriendMotionScene } from './journey'
import { preferredFriendAvatar } from './media'
import type { InterfaceLocale } from './locale'
import { buildCoverageMap, buildIntersectionChanges, buildMovementChains } from './product-insights'

const props = defineProps<{
  friends: Friend[]
  worlds: World[]
  events: ActivityEvent[]
  network: FriendNetwork | null
  mediaUrl: (value?: string) => string
  locale: InterfaceLocale
}>()
const emit = defineEmits<{ openFriend: [userId: string]; openWorld: [worldId: string, location: string]; hydrateWorlds: [worldIds: string[]] }>()

const replayDays = ref<1 | 7 | 30>(7)
const focusedUserId = ref('')
const filteredEvents = computed(() => {
  const cutoff = Date.now() - replayDays.value * 86400000
  return props.events.filter((event) => new Date(event.observedAt).getTime() >= cutoff && (!focusedUserId.value || event.userId === focusedUserId.value))
})
const motion = computed(() => buildFriendMotionOverview(props.friends, props.worlds, filteredEvents.value, props.network))
const friendByID = computed(() => {
  const values = new Map<string, Friend | NonNullable<typeof props.network>['nodes'][number]>()
  ;(props.network?.nodes ?? []).forEach((friend) => values.set(friend.id, friend))
  props.friends.forEach((friend) => values.set(friend.id, friend))
  return values
})
const eventNameByID = computed(() => {
  const values = new Map<string, string>()
  props.events.forEach((event) => { if (event.userId && event.displayName) values.set(event.userId, event.displayName) })
  return values
})
const worldByID = computed(() => new Map(props.worlds.map((world) => [world.id, world])))
const visibleRecentScenes = computed(() => motion.value.recentScenes.slice(0, 12))
const visibleIntersections = computed(() => motion.value.intersections.slice(0, 8))
const missingLiveWorldIDs = computed(() => [...new Set(motion.value.liveScenes.map((scene) => scene.worldId).filter((id): id is string => !!id && !worldByID.value.has(id)))])
const intersectionChanges = computed(() => buildIntersectionChanges(props.events, props.worlds, props.network, replayDays.value))
const movementChains = computed(() => buildMovementChains(props.events, replayDays.value))
const coverage = computed(() => buildCoverageMap(props.friends, props.events, props.network).filter((item) => item.level !== '充分').slice(0, 8))

watch(missingLiveWorldIDs, (worldIDs) => { if (worldIDs.length) emit('hydrateWorlds', worldIDs) }, { immediate: true })

function friendName(id: string) { return friendByID.value.get(id)?.displayName || eventNameByID.value.get(id) || `好友 ${id.slice(-5)}` }
function l(zh: string, en: string) { return props.locale === 'en' ? en : zh }
function circleName(value: string) { return props.locale === 'en' ? value.replace(/的朋友圈$/, "'s circle") : value }
function sceneTitle(scene: FriendMotionScene) { return scene.title === '可见世界' ? l('可见世界', 'Visible world') : scene.title }
function friendAvatar(id: string) { return props.mediaUrl(preferredFriendAvatar(friendByID.value.get(id))) }
function worldImage(scene: FriendMotionScene) {
  const world = scene.worldId ? worldByID.value.get(scene.worldId) : undefined
  return props.mediaUrl(world?.thumbnailImageUrl || world?.imageUrl)
}
function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--'
  return new Intl.DateTimeFormat(props.locale, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}
function sceneSummary(scene: FriendMotionScene) {
  if (scene.communityIds.length > 1) return l(`${scene.communityIds.length} 个朋友圈在这里交汇`, `${scene.communityIds.length} circles meet here`)
  if (scene.communityIds.length === 1) return circleName(scene.communityNames[0])
  return l('尚未归入已知朋友圈', 'Not yet assigned to a known circle')
}
function worldName(id: string) { return worldByID.value.get(id)?.name || id }
</script>

<template>
  <section class="motion-view">
    <div class="motion-filters panel"><label>{{ l('聚焦好友','Focus friend') }}<select v-model="focusedUserId"><option value="">{{ l('全部好友','All friends') }}</option><option v-for="friend in friends" :key="friend.id" :value="friend.id">{{ friend.displayName }}</option></select></label><div><span>{{ l('回放范围','Replay range') }}</span><button v-for="days in ([1,7,30] as const)" :key="days" :class="{active:replayDays===days}" @click="replayDays=days">{{ days===1?l('24 小时','24 hours'):`${days} ${l('天','days')}` }}</button></div><p>{{ l('筛选只作用于本机历史，不会增加上游请求。','Filters local history only and never adds upstream requests.') }}</p></div>
    <div class="motion-metrics panel">
      <div><Radio :size="16"/><span>{{ l('在线','Online') }}<strong>{{ motion.onlineCount }}</strong></span></div>
      <div><Eye :size="16"/><span>{{ l('位置可见','Visible') }}<strong>{{ motion.visibleCount }}</strong></span></div>
      <div><EyeOff :size="16"/><span>{{ l('位置未知','Private') }}<strong>{{ motion.privateCount }}</strong></span></div>
      <div><Clock3 :size="16"/><span>{{ l('历史覆盖','History') }}<strong>{{ motion.coverageDays }} {{ l('天','days') }}</strong></span></div>
      <p>{{ l('仅使用 VRChat 返回与本机观察到的信息。','Uses only data returned by VRChat or observed locally.') }}</p>
    </div>

    <div class="motion-layout">
      <article class="motion-panel live-panel panel">
        <header><div><span class="panel-kicker">{{ l('现在','Now') }}</span><h3>{{ l('好友在哪里','Where friends are') }}</h3></div><span>{{ motion.liveScenes.length }} {{ l('个可见地点','visible places') }}</span></header>
        <div v-if="motion.liveScenes.length" class="scene-grid">
          <button v-for="scene in motion.liveScenes" :key="scene.id" class="scene-card" @click="scene.worldId && emit('openWorld', scene.worldId, scene.location || '')">
            <img v-if="worldImage(scene)" :src="worldImage(scene)" alt=""/>
            <div v-else class="scene-placeholder"><MapPin :size="20"/></div>
            <div class="scene-body">
              <div class="scene-title"><strong>{{ sceneTitle(scene) }}</strong><span>{{ scene.userIds.length }} {{ l('人','people') }}</span></div>
              <p>{{ sceneSummary(scene) }}</p>
              <div class="avatar-row">
                <span v-for="id in scene.userIds.slice(0,7)" :key="id" :title="friendName(id)" @click.stop="emit('openFriend', id)">
                  <img v-if="friendAvatar(id)" :src="friendAvatar(id)" alt=""/><em v-else>{{ friendName(id).slice(0,1) }}</em>
                </span>
                <small v-if="scene.userIds.length>7">+{{ scene.userIds.length-7 }}</small>
              </div>
              <footer v-if="scene.communityIds.length>1"><GitMerge :size="12"/>{{ scene.crossEdges }} {{ l('条跨圈关系','cross-circle links') }}</footer>
            </div>
          </button>
        </div>
        <div v-else class="motion-empty"><MapPin :size="23"/><strong>{{ l('暂时没有可见地点','No visible places') }}</strong><span>{{ l('在线好友可能在私人位置或切换世界。','Friends may be private or traveling.') }}</span></div>
      </article>

      <article class="motion-panel intersection-panel panel">
        <header><div><span class="panel-kicker">{{ l('关系','Links') }}</span><h3>{{ l('朋友圈交汇','Circle intersections') }}</h3></div><span>{{ motion.intersections.length }} {{ l('组','groups') }}</span></header>
        <div v-if="visibleIntersections.length" class="intersection-list">
          <div v-for="item in visibleIntersections" :key="item.id" class="intersection-item">
            <div class="intersection-names"><strong :title="circleName(item.leftName)">{{ circleName(item.leftName) }}</strong><GitMerge :size="16"/><strong :title="circleName(item.rightName)">{{ circleName(item.rightName) }}</strong></div>
            <p>{{ item.sceneCount }} {{ l('次同场','co-visits') }} · {{ item.userIds.length }} {{ l('位好友','friends') }}<span v-if="item.crossEdges"> · {{ item.crossEdges }} {{ l('条跨圈关系','cross-circle links') }}</span></p>
            <div v-if="item.bridgeIds.length" class="bridge-row"><span>{{ l('连接人','Bridges') }}</span><button v-for="id in item.bridgeIds" :key="id" @click="emit('openFriend',id)">{{ friendName(id) }}</button></div>
          </div>
        </div>
        <div v-else class="motion-empty compact"><GitMerge :size="22"/><strong>{{ l('还没有跨圈交汇','No intersections yet') }}</strong><span>{{ l('积累更多位置或关系记录后会自动出现。','More location or relationship history will reveal them.') }}</span></div>
      </article>

      <article class="motion-panel recent-panel panel">
        <header><div><span class="panel-kicker">{{ l('最近','Recent') }}</span><h3>{{ l('聚合动向','Grouped motion') }}</h3></div><span>{{ motion.recentScenes.length }} {{ l('组场景','scenes') }}</span></header>
        <div v-if="visibleRecentScenes.length" class="recent-grid">
          <div v-for="scene in visibleRecentScenes" :key="scene.id" class="recent-scene">
            <div class="recent-head"><span>{{ formatTime(scene.observedAt) }}</span><button v-if="scene.worldId" @click="emit('openWorld',scene.worldId,'')">{{ sceneTitle(scene) }}</button><strong v-else>{{ sceneTitle(scene) }}</strong></div>
            <div class="recent-main"><div><strong>{{ scene.userIds.length }} {{ l('位好友','friends') }}</strong><p>{{ scene.eventCount }} {{ l('条变化已合并','events merged') }} · {{ sceneSummary(scene) }}</p></div><div class="mini-avatars"><button v-for="id in scene.userIds.slice(0,5)" :key="id" :title="friendName(id)" @click="emit('openFriend',id)"><img v-if="friendAvatar(id)" :src="friendAvatar(id)" alt=""/><span v-else>{{ friendName(id).slice(0,1) }}</span></button></div></div>
            <footer v-if="scene.bridgeIds.length"><GitMerge :size="11"/>{{ l('由','Linked by') }} <button v-for="id in scene.bridgeIds" :key="id" @click="emit('openFriend',id)">{{ friendName(id) }}</button> {{ l('串起','') }}</footer>
          </div>
        </div>
        <div v-else class="motion-empty"><UsersRound :size="23"/><strong>{{ l('还没有近期动向','No recent motion') }}</strong><span>{{ l('保持应用运行，事件会按时间和世界自动合并。','Keep the app running to group events by time and world.') }}</span></div>
      </article>

      <article class="motion-panel evolution-panel panel">
        <header><div><span class="panel-kicker">{{ l('变化','Changes') }}</span><h3>{{ l('交汇变化与共同移动','Intersection changes and shared movement') }}</h3></div><span>{{ replayDays }} {{ l('天窗口对比','day window') }}</span></header>
        <div class="evolution-columns">
          <section><h4><GitCompare :size="14"/>{{ l('交汇变化摘要','Intersection changes') }}</h4><div v-for="item in intersectionChanges" :key="item.id" class="change-row"><span :class="{up:item.delta>0}">{{ item.delta>0?'+':'' }}{{ item.delta }}</span><div><strong>{{ item.title }}</strong><small>{{ l('本期','Current') }} {{ item.current }} · {{ l('上期','Previous') }} {{ item.previous }} · {{ item.userIds.length }} {{ l('位好友','friends') }}</small></div></div><p v-if="!intersectionChanges.length">{{ l('当前窗口没有足够差异，或朋友圈数据尚未加载。','No supported change in this window, or circle data is not loaded.') }}</p></section>
          <section><h4><MoveRight :size="14"/>{{ l('共同移动链','Shared movement') }}</h4><button v-for="item in movementChains" :key="item.id" class="movement-row" @click="emit('openWorld',item.toWorldId,'')"><span>{{ worldName(item.fromWorldId) }}</span><MoveRight :size="13"/><strong>{{ worldName(item.toWorldId) }}</strong><small>{{ item.userIds.length }} {{ l('位好友在相近时段发生同向变化','friends changed in the same direction around the same time') }}</small></button><p v-if="!movementChains.length">{{ l('没有观察到至少两人的共同世界变化；不会推断谁带领了谁。','No shared transition for at least two friends; leadership is never inferred.') }}</p></section>
        </div>
      </article>

      <article class="motion-panel coverage-panel panel">
        <header><div><span class="panel-kicker">{{ l('证据','Evidence') }}</span><h3>{{ l('覆盖度地图','Coverage map') }}</h3></div><span>{{ l('优先展示证据不足项','Low-evidence items first') }}</span></header>
        <div class="coverage-list"><button v-for="item in coverage" :key="item.userId" @click="emit('openFriend',item.userId)"><span :data-level="item.level"></span><div><strong>{{ friendName(item.userId) }}</strong><small>{{ item.eventCount }} {{ l('条事件','events') }} · {{ item.scanned?l('关系已扫描','network scanned'):l('关系未扫描','network not scanned') }} · {{ item.visibleNow?l('位置当前可见','visible now'):l('位置当前不可见','not visible now') }}</small></div><b>{{ item.level }}</b></button><p v-if="!coverage.length">{{ l('当前好友均有较充分的本机证据。','All current friends have sufficient local evidence.') }}</p></div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.motion-view{display:grid;gap:12px}.motion-filters{display:flex;align-items:end;gap:10px;padding:10px 12px}.motion-filters label{display:grid;gap:4px;color:var(--muted);font-size:9px}.motion-filters select{min-width:180px;padding:7px 8px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--ink)}.motion-filters>div{display:flex;align-items:center;gap:5px}.motion-filters>div>span{margin-right:3px;color:var(--muted);font-size:9px}.motion-filters button{padding:7px 8px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--muted);font-size:9px}.motion-filters button.active{border-color:var(--accent);background:var(--accent-soft);color:var(--accent)}.motion-filters>p{margin:0 0 7px auto;color:var(--muted);font-size:9px}.motion-metrics{display:flex;align-items:center;gap:7px;padding:8px}.motion-metrics>div{display:flex;align-items:center;gap:9px;min-width:138px;padding:10px 12px;border-radius:7px;background:var(--surface-muted);color:var(--muted)}.motion-metrics>div svg{color:var(--accent)}.motion-metrics span{display:flex;align-items:baseline;gap:7px;font-size:11px}.motion-metrics strong{color:var(--ink);font-size:16px}.motion-metrics>p{margin-left:auto;padding:0 8px;color:var(--muted);font-size:10px}.motion-layout{display:grid;grid-template-columns:minmax(0,1.45fr) minmax(400px,.85fr);gap:12px}.motion-panel{min-width:0;padding:14px}.motion-panel>header{display:flex;align-items:flex-end;justify-content:space-between;margin-bottom:12px;padding:0 2px}.motion-panel h3{margin:2px 0 0;font-size:16px}.motion-panel>header>span{color:var(--muted);font-size:11px}.live-panel{grid-column:1}.intersection-panel{grid-column:2;grid-row:1 / span 2}.recent-panel{grid-column:1}.evolution-panel,.coverage-panel{grid-column:1/-1}.scene-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:9px}.scene-card{display:grid;grid-template-columns:112px minmax(0,1fr);min-height:128px;padding:0;overflow:hidden;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted);color:var(--ink);text-align:left;cursor:pointer}.scene-card>img,.scene-placeholder{width:112px;height:100%;min-height:128px;object-fit:cover}.scene-placeholder{display:grid;place-items:center;background:var(--surface-strong);color:var(--muted)}.scene-body{min-width:0;padding:12px}.scene-title{display:flex;align-items:center;gap:8px}.scene-title strong{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:13px}.scene-title span{margin-left:auto;padding:3px 6px;border-radius:4px;background:var(--surface);color:var(--muted);font-size:10px}.scene-body>p{margin:6px 0 9px;color:var(--muted);font-size:11px}.avatar-row{display:flex;align-items:center;padding-left:4px}.avatar-row>span{position:relative;width:30px;height:30px;margin-left:-4px;overflow:hidden;border:2px solid var(--surface-muted);border-radius:50%;background:var(--surface-strong)}.avatar-row img,.avatar-row em{width:100%;height:100%;object-fit:cover}.avatar-row em{display:grid;place-items:center;color:var(--ink-soft);font-size:10px;font-style:normal}.avatar-row small{margin-left:6px;color:var(--muted);font-size:10px}.scene-body footer{display:flex;align-items:center;gap:5px;margin-top:8px;color:var(--accent);font-size:10px}.intersection-list{display:grid;gap:8px}.intersection-item{padding:13px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.intersection-names{display:grid;grid-template-columns:minmax(0,1fr) 20px minmax(0,1fr);align-items:center;gap:7px}.intersection-names strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:13px;line-height:1.35}.intersection-names svg{color:var(--accent)}.intersection-item>p{margin:8px 0;color:var(--muted);font-size:10px}.bridge-row{display:flex;align-items:center;flex-wrap:wrap;gap:6px}.bridge-row>span{color:var(--muted);font-size:10px}.bridge-row button,.recent-scene footer button{max-width:128px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;padding:4px 7px;border:1px solid var(--line);border-radius:5px;background:var(--surface);color:var(--ink-soft);font-size:10px;cursor:pointer}.recent-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:9px}.recent-scene{padding:12px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.recent-head{display:flex;align-items:center;gap:8px}.recent-head>span{color:var(--muted);font-size:10px}.recent-head>button,.recent-head>strong{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;border:0;background:transparent;color:var(--ink-soft);font-size:11px}.recent-head>button{padding:0;text-decoration:underline;text-decoration-color:var(--line);cursor:pointer}.recent-main{display:flex;align-items:center;justify-content:space-between;gap:9px;margin-top:9px}.recent-main>div:first-child{min-width:0}.recent-main strong{font-size:13px}.recent-main p{margin:4px 0 0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--muted);font-size:10px}.mini-avatars{display:flex;padding-left:4px}.mini-avatars button{width:28px;height:28px;margin-left:-4px;padding:0;overflow:hidden;border:2px solid var(--surface-muted);border-radius:50%;background:var(--surface-strong);color:var(--ink-soft);font-size:9px}.mini-avatars img{width:100%;height:100%;object-fit:cover}.recent-scene footer{display:flex;align-items:center;gap:5px;margin-top:9px;color:var(--muted);font-size:10px}.evolution-columns{display:grid;grid-template-columns:1fr 1fr;gap:10px}.evolution-columns>section{padding:11px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.evolution-columns h4{display:flex;align-items:center;gap:6px;margin:0 0 8px;font-size:10px}.evolution-columns>section>p,.coverage-list>p{color:var(--muted);font-size:9px}.change-row{display:grid;grid-template-columns:34px 1fr;gap:7px;align-items:center;padding:7px 0;border-top:1px solid var(--line)}.change-row>span{padding:4px;border-radius:6px;background:var(--danger-soft);color:var(--danger);text-align:center;font-size:9px}.change-row>span.up{background:var(--success-soft);color:var(--success)}.change-row strong,.change-row small{display:block}.change-row strong{font-size:9px}.change-row small{margin-top:2px;color:var(--muted);font-size:8px}.movement-row{width:100%;display:grid;grid-template-columns:minmax(0,1fr) 18px minmax(0,1fr);align-items:center;gap:5px;padding:7px 0;border:0;border-top:1px solid var(--line);background:transparent;color:var(--ink);text-align:left}.movement-row>span,.movement-row>strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:9px}.movement-row>small{grid-column:1/-1;color:var(--muted);font-size:8px}.coverage-list{display:grid;grid-template-columns:repeat(2,1fr);gap:6px}.coverage-list button{display:grid;grid-template-columns:8px 1fr auto;align-items:center;gap:7px;padding:8px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted);color:var(--ink);text-align:left}.coverage-list button>span{width:7px;height:7px;border-radius:50%;background:var(--warning)}.coverage-list button>span[data-level='稀少']{background:var(--danger)}.coverage-list button div{min-width:0}.coverage-list strong,.coverage-list small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.coverage-list strong{font-size:9px}.coverage-list small{margin-top:2px;color:var(--muted);font-size:8px}.coverage-list b{font-size:8px;color:var(--muted)}.motion-empty{min-height:140px;display:flex;flex-direction:column;align-items:center;justify-content:center;color:var(--muted)}.motion-empty.compact{min-height:220px}.motion-empty strong{margin:8px 0 4px;color:var(--ink);font-size:13px}.motion-empty span{font-size:10px}@media(max-width:1180px){.motion-layout{grid-template-columns:1fr}.live-panel,.recent-panel,.intersection-panel{grid-column:1;grid-row:auto}.intersection-list{grid-template-columns:repeat(2,minmax(0,1fr))}.motion-metrics{flex-wrap:wrap}.motion-metrics>p{width:100%;margin:2px 0}}@media(max-width:760px){.motion-filters{align-items:stretch;flex-direction:column}.motion-filters>p{margin:0}.evolution-columns,.coverage-list{grid-template-columns:1fr}}@media(max-width:680px){.scene-grid,.recent-grid,.intersection-list{grid-template-columns:1fr}.motion-metrics>div{flex:1;min-width:140px}.scene-card{grid-template-columns:84px minmax(0,1fr)}.scene-card>img,.scene-placeholder{width:84px}.motion-metrics>p{line-height:1.5}}
</style>
