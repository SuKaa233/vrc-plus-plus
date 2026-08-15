<script setup lang="ts">
import { computed, watch } from 'vue'
import { Clock3, Eye, EyeOff, GitMerge, MapPin, Radio, UsersRound } from '@lucide/vue'
import type { ActivityEvent, Friend, FriendNetwork, World } from './api'
import { buildFriendMotionOverview, type FriendMotionScene } from './journey'
import { preferredFriendAvatar } from './media'
import type { InterfaceLocale } from './locale'

const props = defineProps<{
  friends: Friend[]
  worlds: World[]
  events: ActivityEvent[]
  network: FriendNetwork | null
  mediaUrl: (value?: string) => string
  locale: InterfaceLocale
}>()
const emit = defineEmits<{ openFriend: [userId: string]; openWorld: [worldId: string, location: string]; hydrateWorlds: [worldIds: string[]] }>()

const motion = computed(() => buildFriendMotionOverview(props.friends, props.worlds, props.events, props.network))
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
</script>

<template>
  <section class="motion-view">
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
    </div>
  </section>
</template>

<style scoped>
.motion-view{display:grid;gap:12px}.motion-metrics{display:flex;align-items:center;gap:7px;padding:8px}.motion-metrics>div{display:flex;align-items:center;gap:9px;min-width:138px;padding:10px 12px;border-radius:7px;background:var(--surface-muted);color:var(--muted)}.motion-metrics>div svg{color:var(--accent)}.motion-metrics span{display:flex;align-items:baseline;gap:7px;font-size:11px}.motion-metrics strong{color:var(--ink);font-size:16px}.motion-metrics>p{margin-left:auto;padding:0 8px;color:var(--muted);font-size:10px}.motion-layout{display:grid;grid-template-columns:minmax(0,1.45fr) minmax(400px,.85fr);gap:12px}.motion-panel{min-width:0;padding:14px}.motion-panel>header{display:flex;align-items:flex-end;justify-content:space-between;margin-bottom:12px;padding:0 2px}.motion-panel h3{margin:2px 0 0;font-size:16px}.motion-panel>header>span{color:var(--muted);font-size:11px}.live-panel{grid-column:1}.intersection-panel{grid-column:2;grid-row:1 / span 2}.recent-panel{grid-column:1}.scene-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:9px}.scene-card{display:grid;grid-template-columns:112px minmax(0,1fr);min-height:128px;padding:0;overflow:hidden;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted);color:var(--ink);text-align:left;cursor:pointer}.scene-card>img,.scene-placeholder{width:112px;height:100%;min-height:128px;object-fit:cover}.scene-placeholder{display:grid;place-items:center;background:var(--surface-strong);color:var(--muted)}.scene-body{min-width:0;padding:12px}.scene-title{display:flex;align-items:center;gap:8px}.scene-title strong{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:13px}.scene-title span{margin-left:auto;padding:3px 6px;border-radius:4px;background:var(--surface);color:var(--muted);font-size:10px}.scene-body>p{margin:6px 0 9px;color:var(--muted);font-size:11px}.avatar-row{display:flex;align-items:center;padding-left:4px}.avatar-row>span{position:relative;width:30px;height:30px;margin-left:-4px;overflow:hidden;border:2px solid var(--surface-muted);border-radius:50%;background:var(--surface-strong)}.avatar-row img,.avatar-row em{width:100%;height:100%;object-fit:cover}.avatar-row em{display:grid;place-items:center;color:var(--ink-soft);font-size:10px;font-style:normal}.avatar-row small{margin-left:6px;color:var(--muted);font-size:10px}.scene-body footer{display:flex;align-items:center;gap:5px;margin-top:8px;color:var(--accent);font-size:10px}.intersection-list{display:grid;gap:8px}.intersection-item{padding:13px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.intersection-names{display:grid;grid-template-columns:minmax(0,1fr) 20px minmax(0,1fr);align-items:center;gap:7px}.intersection-names strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:13px;line-height:1.35}.intersection-names svg{color:var(--accent)}.intersection-item>p{margin:8px 0;color:var(--muted);font-size:10px}.bridge-row{display:flex;align-items:center;flex-wrap:wrap;gap:6px}.bridge-row>span{color:var(--muted);font-size:10px}.bridge-row button,.recent-scene footer button{max-width:128px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;padding:4px 7px;border:1px solid var(--line);border-radius:5px;background:var(--surface);color:var(--ink-soft);font-size:10px;cursor:pointer}.recent-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:9px}.recent-scene{padding:12px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.recent-head{display:flex;align-items:center;gap:8px}.recent-head>span{color:var(--muted);font-size:10px}.recent-head>button,.recent-head>strong{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;border:0;background:transparent;color:var(--ink-soft);font-size:11px}.recent-head>button{padding:0;text-decoration:underline;text-decoration-color:var(--line);cursor:pointer}.recent-main{display:flex;align-items:center;justify-content:space-between;gap:9px;margin-top:9px}.recent-main>div:first-child{min-width:0}.recent-main strong{font-size:13px}.recent-main p{margin:4px 0 0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--muted);font-size:10px}.mini-avatars{display:flex;padding-left:4px}.mini-avatars button{width:28px;height:28px;margin-left:-4px;padding:0;overflow:hidden;border:2px solid var(--surface-muted);border-radius:50%;background:var(--surface-strong);color:var(--ink-soft);font-size:9px}.mini-avatars img{width:100%;height:100%;object-fit:cover}.recent-scene footer{display:flex;align-items:center;gap:5px;margin-top:9px;color:var(--muted);font-size:10px}.motion-empty{min-height:140px;display:flex;flex-direction:column;align-items:center;justify-content:center;color:var(--muted)}.motion-empty.compact{min-height:220px}.motion-empty strong{margin:8px 0 4px;color:var(--ink);font-size:13px}.motion-empty span{font-size:10px}@media(max-width:1180px){.motion-layout{grid-template-columns:1fr}.live-panel,.recent-panel,.intersection-panel{grid-column:1;grid-row:auto}.intersection-list{grid-template-columns:repeat(2,minmax(0,1fr))}.motion-metrics{flex-wrap:wrap}.motion-metrics>p{width:100%;margin:2px 0}}@media(max-width:680px){.scene-grid,.recent-grid,.intersection-list{grid-template-columns:1fr}.motion-metrics>div{flex:1;min-width:140px}.scene-card{grid-template-columns:84px minmax(0,1fr)}.scene-card>img,.scene-placeholder{width:84px}.motion-metrics>p{line-height:1.5}}
</style>
