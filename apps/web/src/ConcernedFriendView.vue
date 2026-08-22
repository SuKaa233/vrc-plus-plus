<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Activity, AlertTriangle, Eye, Globe2, History, LoaderCircle, MapPin, Network, RefreshCw, ScanSearch, Search, ShieldCheck, Sparkles, UserRound, Users } from '@lucide/vue'
import { LocalApi, type ActivityEvent, type Friend, type FriendActivityInsights, type FriendNetwork, type MutualFriend, type UserProfile, type World } from './api'
import { optimizedVrcImageUrl, preferredFriendAvatar } from './media'
import { buildConcernedInferences } from './concerned-insights'

const props = defineProps<{
  friends: Friend[]
  network: FriendNetwork | null
  events: ActivityEvent[]
  worlds: World[]
  storageKey: string
  mediaUrl: (value?: string) => string
}>()

const emit = defineEmits<{
  openFriend: [userID: string]
  openWorld: [worldID: string]
  networkUpdated: []
}>()

const api = new LocalApi()
const selectedID = ref('')
const profile = ref<UserProfile | null>(null)
const insights = ref<FriendActivityInsights | null>(null)
const mutuals = ref<MutualFriend[]>([])
const loading = ref(false)
const scanning = ref(false)
const error = ref('')
const friendQuery = ref('')
const timelineFilter = ref<'all' | 'location' | 'private' | 'presence'>('all')
let requestSerial = 0

const storageName = computed(() => `vrc-plus-plus:concerned-friend:${props.storageKey}`)
const friend = computed(() => props.friends.find((item) => item.id === selectedID.value) ?? null)
const friendSearchResults = computed(() => {
  const query = friendQuery.value.trim().toLocaleLowerCase()
  if (!query) return []
  return [...props.friends].filter((item) => [item.displayName, item.id, item.statusDescription, item.platform, item.lastPlatform]
    .some((value) => value?.toLocaleLowerCase().includes(query)))
    .sort((left, right) => {
      const leftExact = left.displayName.toLocaleLowerCase() === query
      const rightExact = right.displayName.toLocaleLowerCase() === query
      if (leftExact !== rightExact) return leftExact ? -1 : 1
      if (left.online !== right.online) return left.online ? -1 : 1
      return left.displayName.localeCompare(right.displayName)
    }).slice(0, 30)
})
const worldNames = computed(() => new Map(props.worlds.map((item) => [item.id, item.name])))
const networkNodes = computed(() => new Map((props.network?.nodes ?? []).map((node) => [node.id, node])))
const relationPeers = computed(() => {
  if (!selectedID.value) return []
  const ids = new Set<string>()
  for (const edge of props.network?.edges ?? []) {
    if (edge.source === selectedID.value) ids.add(edge.target)
    if (edge.target === selectedID.value) ids.add(edge.source)
  }
  return [...ids].map((id) => ({ id, name: networkNodes.value.get(id)?.displayName || id }))
    .sort((left, right) => left.name.localeCompare(right.name))
})
const avatar = computed(() => props.mediaUrl(
  profile.value?.profilePicOverrideThumbnail
  || optimizedVrcImageUrl(profile.value?.profilePicOverride)
  || preferredFriendAvatar(profile.value)
  || preferredFriendAvatar(friend.value ?? undefined),
))
const targetEvents = computed(() => insights.value?.timeline ?? props.events.filter((event) => event.userId === selectedID.value))
const filteredTimeline = computed(() => targetEvents.value.filter((event) => {
  if (timelineFilter.value === 'location') return event.type.includes('location') || !!event.worldId
  if (timelineFilter.value === 'private') return event.locationKind === 'private'
  if (timelineFilter.value === 'presence') return /online|offline|active|player-joined|player-left/.test(event.type)
  return true
}))
const visitedWorlds = computed(() => {
  const result = new Map<string, { id: string; name: string; count: number; lastSeenAt: string; privateCount: number }>()
  for (const event of targetEvents.value) {
    if (!event.worldId) continue
    const item = result.get(event.worldId) ?? { id: event.worldId, name: worldNames.value.get(event.worldId) || event.worldId, count: 0, lastSeenAt: event.observedAt, privateCount: 0 }
    item.count += 1
    if (event.locationKind === 'private') item.privateCount += 1
    if (event.observedAt > item.lastSeenAt) item.lastSeenAt = event.observedAt
    result.set(event.worldId, item)
  }
  return [...result.values()].sort((left, right) => right.lastSeenAt.localeCompare(left.lastSeenAt))
})
const coObservedPeers = computed(() => {
  const target = targetEvents.value.filter((event) => /player-joined|player-left/.test(event.type))
  if (!target.length) return []
  const counts = new Map<string, { id: string; name: string; count: number; lastSeenAt: string }>()
  for (const event of props.events) {
    if (!event.userId || event.userId === selectedID.value || !/player-joined|player-left/.test(event.type)) continue
    const observed = new Date(event.observedAt).getTime()
    if (!target.some((item) => Math.abs(new Date(item.observedAt).getTime() - observed) <= 45 * 60 * 1000 && (!item.worldId || !event.worldId || item.worldId === event.worldId))) continue
    const item = counts.get(event.userId) ?? { id: event.userId, name: event.displayName || event.userId, count: 0, lastSeenAt: event.observedAt }
    item.count += 1
    if (event.observedAt > item.lastSeenAt) item.lastSeenAt = event.observedAt
    counts.set(event.userId, item)
  }
  return [...counts.values()].sort((left, right) => right.count - left.count).slice(0, 30)
})
const networkDegrees = computed(() => {
  const degrees = new Map<string, number>()
  for (const edge of props.network?.edges ?? []) {
    degrees.set(edge.source, (degrees.get(edge.source) ?? 0) + 1)
    degrees.set(edge.target, (degrees.get(edge.target) ?? 0) + 1)
  }
  return degrees
})
const inferredInsights = computed(() => {
  const relationIDs = new Set(relationPeers.value.map((item) => item.id))
  return buildConcernedInferences({
    insights: insights.value,
    events: targetEvents.value,
    relationPeerCount: relationPeers.value.length,
    allNetworkDegrees: [...networkDegrees.value.values()],
    coObservedCount: coObservedPeers.value.length,
    coObservedRelationOverlap: coObservedPeers.value.filter((item) => relationIDs.has(item.id)).length,
    visitedWorlds: visitedWorlds.value,
  })
})
const currentLocation = computed(() => profile.value?.location || friend.value?.location || '')
const privateRoomStatus = computed(() => {
  const location = currentLocation.value
  if (!location || location === 'offline') return '当前没有可见的在线位置'
  if (location === 'private') return '当前位于不可见私人实例；无法确认是谁创建'
  if (location.includes('~private(')) {
    return location.includes(`~private(${selectedID.value})`) ? '当前可见位置显示：由该好友持有私人实例' : '当前位于私人实例；可见信息未表明由该好友创建'
  }
  return '当前没有观察到私人实例状态'
})
const coverageText = computed(() => `${insights.value?.coverageDays ?? 0} 个自然日 / ${insights.value?.totalEvents ?? 0} 条事件`)

watch(() => props.storageKey, restoreSelection, { immediate: true })
watch(() => props.friends, () => {
  if (!selectedID.value || !props.friends.some((item) => item.id === selectedID.value)) restoreSelection()
}, { deep: true })
watch(selectedID, (value) => {
  if (!value) return
  localStorage.setItem(storageName.value, value)
  void loadDossier()
})

onMounted(() => { if (selectedID.value) void loadDossier() })

function restoreSelection() {
  const saved = localStorage.getItem(storageName.value) || ''
  selectedID.value = props.friends.some((item) => item.id === saved)
    ? saved
    : (props.friends.find((item) => item.online)?.id || props.friends[0]?.id || '')
}

function selectConcernedFriend(item: Friend) {
  selectedID.value = item.id
  friendQuery.value = ''
}

function searchFriendImage(item: Friend) {
  return props.mediaUrl(preferredFriendAvatar(item))
}

async function loadDossier() {
  if (!selectedID.value) return
  const serial = ++requestSerial
  loading.value = true
  error.value = ''
  const [profileResult, insightResult, mutualResult] = await Promise.allSettled([
    api.user(selectedID.value),
    api.friendActivityInsights(selectedID.value, 30),
    api.mutualFriends(selectedID.value),
  ])
  if (serial !== requestSerial) return
  profile.value = profileResult.status === 'fulfilled' ? profileResult.value.items[0] ?? null : null
  insights.value = insightResult.status === 'fulfilled' ? insightResult.value : null
  mutuals.value = mutualResult.status === 'fulfilled' ? mutualResult.value.items : []
  const failed = [profileResult, insightResult, mutualResult].filter((item) => item.status === 'rejected').length
  if (failed === 3) error.value = '暂时无法读取该好友档案，请检查登录状态和网络。'
  loading.value = false
}

async function rescanRelations() {
  if (!selectedID.value || scanning.value) return
  scanning.value = true
  error.value = ''
  try {
    const result = await api.mutualFriends(selectedID.value, true)
    mutuals.value = result.items
    insights.value = await api.friendActivityInsights(selectedID.value, 30)
    emit('networkUpdated')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '关系扫描失败'
  } finally {
    scanning.value = false
  }
}

function locationLabel(kind?: string) {
  return ({ private: '私人实例', friends_plus: '好友+ 实例', friends: '好友实例', invite_plus: '邀请+ 实例', group: '群组实例', public: '公开实例', traveling: '切换世界', offline: '离线', unknown: '位置未知', unavailable: '位置受限' } as Record<string, string>)[kind || 'unknown'] || '位置未知'
}

function eventLabel(event: ActivityEvent) {
  const world = event.worldId ? worldNames.value.get(event.worldId) || event.worldId : ''
  return [event.summary, world, locationLabel(event.locationKind)].filter(Boolean).join(' · ')
}

function relationStateLabel(state: string) {
  if (state === 'newly_observed') return '本次扫描新观察到关系'
  if (state === 'not_observed') return '本次扫描未再次观察到'
  return '首次关系扫描基线'
}

function dateTime(value?: string) {
  if (!value) return '暂无记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(date)
}
</script>

<template>
  <article class="concerned-view">
    <header class="focus-header panel">
      <div><span class="panel-kicker">特别关心</span><h2>好友全景档案</h2><p>把公开资料、本机日志、世界轨迹和关系网证据放在同一页。</p></div>
      <div class="focus-search">
        <label><span>搜索并切换关注好友</span><span class="focus-search-input"><Search :size="15" /><input v-model="friendQuery" autocomplete="off" placeholder="输入好友昵称或 usr_ ID" /></span></label>
        <div v-if="friendQuery.trim()" class="focus-search-results">
          <button v-for="item in friendSearchResults" :key="item.id" type="button" :class="{ selected: item.id === selectedID }" @click="selectConcernedFriend(item)"><img v-if="searchFriendImage(item)" :src="searchFriendImage(item)" alt="" /><span v-else>{{ item.displayName.slice(0,1) }}</span><strong>{{ item.displayName }}</strong><small>{{ item.online ? '在线' : '离线' }} · {{ item.statusDescription || item.id }}</small></button>
          <p v-if="!friendSearchResults.length">没有找到匹配好友</p>
        </div>
      </div>
      <button :disabled="loading || !selectedID" @click="loadDossier"><LoaderCircle v-if="loading" class="spin" :size="16" /><RefreshCw v-else :size="16" />刷新档案</button>
    </header>

    <div v-if="!friend" class="panel focus-empty"><UserRound :size="28" /><strong>还没有可关注的好友</strong><span>登录并读取好友列表后即可建立特别关心档案。</span></div>
    <template v-else>
      <section class="focus-identity panel">
        <div class="focus-avatar"><img v-if="avatar" :src="avatar" alt="" /><UserRound v-else :size="34" /></div>
        <div class="focus-name"><span>特别关心对象</span><h2>{{ profile?.displayName || friend.displayName }}</h2><p>{{ profile?.statusDescription || friend.statusDescription || profile?.bio || '暂无公开状态说明' }}</p></div>
        <div class="presence-card"><MapPin :size="17" /><span>当前位置</span><strong>{{ privateRoomStatus }}</strong></div>
        <button class="profile-button" @click="emit('openFriend', friend.id)"><Eye :size="16" />打开基础资料</button>
      </section>

      <p v-if="error" class="focus-error"><AlertTriangle :size="15" />{{ error }}</p>
      <section class="evidence-boundary panel"><ShieldCheck :size="18" /><p><strong>信息范围：</strong>页面只汇总当前账号可见的 VRChat 公开资料、实时好友动态与游戏记录。私人实例通常只能识别“处于私人位置”；关系变化表示两次读取间新看到的共同好友，不能证明具体加好友时间，也不能读取对方私聊内容。</p></section>

      <section class="focus-metrics">
        <div class="panel"><Activity :size="18" /><span>本机日志覆盖</span><strong>{{ coverageText }}</strong></div>
        <div class="panel"><Globe2 :size="18" /><span>观察到的世界</span><strong>{{ visitedWorlds.length }} 个</strong></div>
        <div class="panel"><ShieldCheck :size="18" /><span>私人位置变化</span><strong>{{ insights?.privateVisits ?? 0 }} 次</strong></div>
        <div class="panel"><Network :size="18" /><span>当前关系线索</span><strong>{{ relationPeers.length }} 位</strong></div>
        <div class="panel"><Users :size="18" /><span>公开共同好友</span><strong>{{ mutuals.length }} 位</strong></div>
      </section>

      <section class="panel inference-section">
        <header><div><Sparkles :size="18" /><span><strong>基于当前证据的参考推测</strong><small>每条结论均附证据和置信度，不代表对方完整或真实社交行为。</small></span></div><em>{{ inferredInsights.length }} 条</em></header>
        <div class="inference-grid">
          <article v-for="item in inferredInsights" :key="item.id" :data-tone="item.tone"><div><strong>{{ item.title }}</strong><span>置信度 {{ item.confidence }}</span></div><p>{{ item.summary }}</p><small>{{ item.evidence }}</small></article>
          <p v-if="!inferredInsights.length" class="inference-empty">当前日志或关系网证据不足。运行应用一段时间并扫描该好友关系后，这里会逐步生成参考结论。</p>
        </div>
      </section>

      <section class="focus-grid">
        <article class="panel focus-block relationship-block">
          <header><div><Network :size="17" /><strong>关系与新增好友线索</strong></div><button :disabled="scanning" @click="rescanRelations"><LoaderCircle v-if="scanning" class="spin" :size="14" /><ScanSearch v-else :size="14" />重新扫描此人</button></header>
          <p class="block-note">当前连线来自共同好友公开结果；“新观察到”仅表示相邻两次扫描结果发生变化。</p>
          <div class="relation-list">
            <button v-for="peer in relationPeers" :key="peer.id" @click="emit('openFriend', peer.id)"><span>{{ peer.name.slice(0,1) }}</span><strong>{{ peer.name }}</strong><small>当前已观察关系</small></button>
            <p v-if="!relationPeers.length">关系网尚未扫描到该好友的连接。</p>
          </div>
          <div class="change-list">
            <div v-for="item in insights?.relationChanges ?? []" :key="`${item.peerId}-${item.observedAt}-${item.state}`"><i :data-state="item.state"></i><span><strong>{{ item.displayName || item.peerId }}</strong><small>{{ relationStateLabel(item.state) }} · {{ dateTime(item.observedAt) }}</small></span></div>
            <p v-if="!insights?.relationChanges?.length">从下一次关系扫描开始记录变化；首次扫描会建立基线。</p>
          </div>
        </article>

        <article class="panel focus-block">
          <header><div><Users :size="17" /><strong>同场与交互线索</strong></div><span>{{ coObservedPeers.length }} 位</span></header>
          <p class="block-note">仅统计该好友出现前后 45 分钟内，本机游戏日志也出现的人；这是同场时段线索，不代表私聊或直接互动。</p>
          <div class="peer-table"><button v-for="peer in coObservedPeers" :key="peer.id" @click="emit('openFriend', peer.id)"><strong>{{ peer.name }}</strong><span>{{ peer.count }} 条邻近日志</span><small>{{ dateTime(peer.lastSeenAt) }}</small></button><p v-if="!coObservedPeers.length">尚无足够的同场日志。</p></div>
        </article>

        <article class="panel focus-block world-block">
          <header><div><Globe2 :size="17" /><strong>世界轨迹</strong></div><span>{{ visitedWorlds.length }} 个</span></header>
          <div class="world-history"><button v-for="world in visitedWorlds" :key="world.id" @click="emit('openWorld', world.id)"><Globe2 :size="15" /><span><strong>{{ world.name }}</strong><small>{{ world.count }} 条记录 · 最近 {{ dateTime(world.lastSeenAt) }}</small></span><em v-if="world.privateCount">{{ world.privateCount }} 条私人位置</em></button><p v-if="!visitedWorlds.length">尚未在本机记录中识别到世界。</p></div>
        </article>

        <article class="panel focus-block timeline-block">
          <header><div><History :size="17" /><strong>完整本机日志</strong></div><span>{{ filteredTimeline.length }} / {{ targetEvents.length }}</span></header>
          <div class="timeline-filters"><button v-for="item in ([['all','全部'],['location','世界'],['private','私人'],['presence','上下线 / 同场']] as const)" :key="item[0]" :class="{ active: timelineFilter === item[0] }" @click="timelineFilter = item[0]">{{ item[1] }}</button></div>
          <div class="focus-timeline"><div v-for="event in filteredTimeline" :key="event.id"><i :data-private="event.locationKind === 'private'"></i><span><strong>{{ eventLabel(event) }}</strong><small>{{ dateTime(event.observedAt) }} · {{ event.type.startsWith('game.') ? '游戏记录' : '实时动态' }}</small></span></div><p v-if="!filteredTimeline.length">当前筛选范围暂无记录。没有记录不代表没有发生。</p></div>
        </article>
      </section>
    </template>
  </article>
</template>

<style scoped>
.concerned-view{grid-column:1/-1;display:grid;gap:12px}.panel{min-width:0}.focus-header{padding:18px 20px;display:grid;grid-template-columns:1fr minmax(260px,360px) auto;align-items:end;gap:14px}.focus-header h2{margin:3px 0 4px;font-size:19px}.focus-header p{margin:0;color:var(--muted);font-size:10px}.focus-header label{display:grid;gap:5px;color:var(--muted);font-size:9px}.focus-header input,.focus-header>button,.profile-button,.focus-block header button{min-height:38px;padding:0 11px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted);color:var(--ink);font:inherit;font-size:10px}.focus-header>button,.profile-button,.focus-block header button{display:flex;align-items:center;justify-content:center;gap:6px;cursor:pointer}.focus-search{position:relative}.focus-search-input{display:grid;grid-template-columns:28px 1fr;align-items:center;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.focus-search-input svg{margin-left:9px;color:var(--muted)}.focus-search-input input{width:100%;min-width:0;border:0;background:transparent;outline:0}.focus-search-results{z-index:20;width:100%;max-height:330px;padding:5px;background:var(--surface);border:1px solid var(--line);border-radius:9px;box-shadow:var(--shadow);overflow:auto;position:absolute;top:calc(100% + 5px);left:0}.focus-search-results button{width:100%;min-height:48px;padding:6px;border:0;border-radius:7px;background:transparent;color:var(--ink);display:grid;grid-template-columns:34px 1fr;align-items:center;gap:2px 8px;text-align:left}.focus-search-results button:hover,.focus-search-results button.selected{background:var(--accent-soft)}.focus-search-results img,.focus-search-results button>span{width:34px;height:34px;grid-row:1/3;border-radius:8px;object-fit:cover;background:var(--surface-muted);display:grid;place-items:center}.focus-search-results strong,.focus-search-results small{min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.focus-search-results strong{font-size:9px}.focus-search-results small{color:var(--muted);font-size:8px}.focus-search-results p{margin:0;padding:16px;text-align:center;color:var(--muted);font-size:9px}.focus-identity{padding:18px 20px;display:grid;grid-template-columns:72px 1fr minmax(240px,auto) auto;align-items:center;gap:15px}.focus-avatar{width:72px;height:72px;border-radius:18px;background:var(--accent-soft);display:grid;place-items:center;overflow:hidden;color:var(--accent)}.focus-avatar img{width:100%;height:100%;object-fit:cover}.focus-name span,.presence-card span{color:var(--muted);font-size:8px}.focus-name h2{margin:3px 0;font-size:21px}.focus-name p{margin:0;color:var(--muted);font-size:9px}.presence-card{padding:11px 12px;border-radius:9px;background:var(--surface-muted);display:grid;grid-template-columns:20px 1fr;gap:3px 7px}.presence-card svg{grid-row:1/3;color:var(--accent)}.presence-card strong{font-size:9px}.profile-button{background:var(--accent);border-color:var(--accent);color:#fff}.focus-error,.evidence-boundary{margin:0;padding:11px 14px;display:flex;align-items:flex-start;gap:9px;font-size:9px;line-height:1.6}.focus-error{color:var(--danger);background:var(--danger-soft);border:1px solid color-mix(in srgb,var(--danger) 24%,var(--line));border-radius:8px}.evidence-boundary{color:var(--muted)}.evidence-boundary svg{flex:none;color:var(--success)}.evidence-boundary p{margin:0}.evidence-boundary strong{color:var(--ink)}.focus-metrics{display:grid;grid-template-columns:repeat(5,1fr);gap:10px}.focus-metrics>div{padding:12px;display:grid;grid-template-columns:23px 1fr;gap:3px}.focus-metrics svg{grid-row:1/3;color:var(--accent)}.focus-metrics span{color:var(--muted);font-size:8px}.focus-metrics strong{font-size:10px}.inference-section{padding:0;overflow:hidden}.inference-section>header{min-height:54px;padding:0 15px;border-bottom:1px solid var(--line);display:flex;align-items:center;justify-content:space-between}.inference-section>header>div{display:flex;align-items:center;gap:9px}.inference-section>header svg{color:var(--accent)}.inference-section>header strong,.inference-section>header small{display:block}.inference-section>header strong{font-size:11px}.inference-section>header small{margin-top:3px;color:var(--muted);font-size:8px}.inference-section>header em{padding:4px 7px;border-radius:99px;background:var(--accent-soft);color:var(--accent);font-size:8px;font-style:normal}.inference-grid{padding:12px;display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px}.inference-grid article{padding:11px;border:1px solid var(--line);border-top:3px solid var(--muted);border-radius:8px;background:var(--surface-muted)}.inference-grid article[data-tone=accent]{border-top-color:var(--accent)}.inference-grid article[data-tone=success]{border-top-color:var(--success)}.inference-grid article[data-tone=warning]{border-top-color:var(--warning)}.inference-grid article>div{display:flex;align-items:flex-start;justify-content:space-between;gap:8px}.inference-grid strong{font-size:9px}.inference-grid article>div span{flex:none;color:var(--muted);font-size:7px}.inference-grid p{margin:7px 0;color:var(--ink-soft);font-size:8px;line-height:1.6}.inference-grid small{color:var(--muted);font-size:7px}.inference-empty{grid-column:1/-1;margin:0!important;padding:20px;text-align:center;color:var(--muted)!important}.focus-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}.focus-block{padding:0;overflow:hidden}.focus-block>header{min-height:48px;padding:0 14px;border-bottom:1px solid var(--line);display:flex;align-items:center;justify-content:space-between;gap:10px}.focus-block>header>div{display:flex;align-items:center;gap:7px}.focus-block>header svg{color:var(--accent)}.focus-block>header strong{font-size:11px}.focus-block>header>span{color:var(--muted);font-size:9px}.focus-block header button{min-height:30px;padding-inline:8px}.block-note{margin:0;padding:10px 14px;color:var(--muted);background:var(--surface-muted);font-size:8px;line-height:1.55}.relation-list{padding:12px;display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:6px}.relation-list button{min-width:0;padding:7px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--ink);display:grid;grid-template-columns:28px 1fr;gap:2px 7px;text-align:left}.relation-list button>span{grid-row:1/3;width:28px;height:28px;border-radius:7px;background:var(--accent-soft);display:grid;place-items:center;color:var(--accent)}.relation-list strong,.relation-list small{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.relation-list strong{font-size:8px}.relation-list small{color:var(--muted);font-size:7px}.relation-list p,.change-list p,.peer-table p,.world-history p,.focus-timeline p{padding:10px;color:var(--muted);font-size:9px}.change-list{max-height:230px;overflow:auto;border-top:1px solid var(--line)}.change-list>div{padding:8px 14px;display:grid;grid-template-columns:8px 1fr;align-items:center;gap:8px;border-top:1px solid var(--line)}.change-list>div:first-child{border-top:0}.change-list i{width:7px;height:7px;border-radius:50%;background:var(--muted)}.change-list i[data-state=newly_observed]{background:var(--success)}.change-list i[data-state=not_observed]{background:var(--danger)}.change-list strong,.change-list small{display:block}.change-list strong{font-size:8px}.change-list small{margin-top:2px;color:var(--muted);font-size:7px}.peer-table,.world-history{display:grid;max-height:350px;overflow:auto}.peer-table button,.world-history button{padding:10px 14px;border:0;border-top:1px solid var(--line);background:transparent;color:var(--ink);display:grid;align-items:center;text-align:left}.peer-table button{grid-template-columns:1fr auto auto;gap:10px}.peer-table strong{font-size:9px}.peer-table span,.peer-table small{color:var(--muted);font-size:8px}.world-history button{grid-template-columns:20px 1fr auto;gap:9px}.world-history button>svg{color:var(--accent)}.world-history strong,.world-history small{display:block}.world-history strong{font-size:9px}.world-history small{margin-top:3px;color:var(--muted);font-size:8px}.world-history em{padding:3px 5px;border-radius:99px;background:var(--danger-soft);color:var(--danger);font-size:7px;font-style:normal}.timeline-block{grid-column:1/-1}.timeline-filters{padding:9px 14px;display:flex;gap:5px;border-bottom:1px solid var(--line)}.timeline-filters button{padding:5px 8px;border:1px solid var(--line);border-radius:6px;background:var(--surface);color:var(--muted);font-size:8px}.timeline-filters button.active{border-color:var(--accent);background:var(--accent-soft);color:var(--accent)}.focus-timeline{max-height:540px;overflow:auto;padding:0 14px}.focus-timeline>div{padding:9px 2px;display:grid;grid-template-columns:8px 1fr;align-items:center;gap:9px;border-top:1px solid var(--line)}.focus-timeline>div:first-child{border-top:0}.focus-timeline i{width:7px;height:7px;border-radius:50%;background:var(--accent)}.focus-timeline i[data-private=true]{background:var(--danger)}.focus-timeline strong,.focus-timeline small{display:block}.focus-timeline strong{font-size:9px}.focus-timeline small{margin-top:3px;color:var(--muted);font-size:8px}.focus-empty{min-height:320px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:8px;color:var(--muted)}.focus-empty strong{color:var(--ink)}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:1050px){.focus-metrics{grid-template-columns:repeat(3,1fr)}.inference-grid{grid-template-columns:1fr 1fr}.focus-identity{grid-template-columns:64px 1fr}.presence-card,.profile-button{grid-column:2}.focus-grid{grid-template-columns:1fr}.timeline-block{grid-column:auto}}
@media(max-width:700px){.focus-header{grid-template-columns:1fr}.focus-identity{grid-template-columns:56px 1fr;padding:14px}.focus-avatar{width:56px;height:56px}.presence-card,.profile-button{grid-column:1/-1}.focus-metrics{grid-template-columns:1fr 1fr}.inference-grid{grid-template-columns:1fr}.relation-list{grid-template-columns:1fr 1fr}.peer-table button{grid-template-columns:1fr auto}.peer-table small{grid-column:1/-1}.focus-header>button{width:100%}}
</style>
