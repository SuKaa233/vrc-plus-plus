<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Activity, CheckCircle2, ChevronRight, CircleAlert, Compass, Database, Globe2, NotebookTabs, Radar, RefreshCw, Route, Save, Users, Wifi } from '@lucide/vue'
import type { ActivityEvent, CacheStats, Diagnostics, Friend, FriendNetwork, NetworkState, RealtimeStatus, World } from './api'
import { buildReadinessChecks, buildSocialTides, buildWorldPassports, readinessScore } from './journey'
import FriendMotion from './FriendMotion.vue'
import type { InterfaceLocale } from './locale'

type Tab = 'motion' | 'tides' | 'passport' | 'readiness'
interface PassportNote { mood: string; tags: string; note: string }

const props = defineProps<{
  friends: Friend[]
  worlds: World[]
  events: ActivityEvent[]
  network: FriendNetwork | null
  diagnostics: Diagnostics | null
  realtime: RealtimeStatus
  networkState: NetworkState
  cache: CacheStats | null
  storageKey: string
  mediaUrl: (value?: string) => string
  locale: InterfaceLocale
}>()
const emit = defineEmits<{ openFriend: [userId: string]; openWorld: [worldId: string, location: string]; loadNetwork: []; refreshSystem: []; hydrateWorlds: [worldIds: string[]] }>()
const tab = ref<Tab>('motion')
const selectedPassportID = ref('')
const passportNotes = ref<Record<string, PassportNote>>({})

const tides = computed(() => buildSocialTides(props.network, props.events))
const passports = computed(() => buildWorldPassports(props.events, props.worlds))
const selectedPassport = computed(() => passports.value.find((item) => item.worldId === selectedPassportID.value) ?? passports.value[0] ?? null)
const selectedNote = computed({
  get: () => selectedPassport.value ? passportNotes.value[selectedPassport.value.worldId] ?? { mood: '', tags: '', note: '' } : { mood: '', tags: '', note: '' },
  set: (value) => { if (selectedPassport.value) passportNotes.value = { ...passportNotes.value, [selectedPassport.value.worldId]: value } },
})
const checks = computed(() => buildReadinessChecks(props.diagnostics, props.realtime, props.networkState, props.cache, '', passports.value))
const score = computed(() => readinessScore(checks.value))
const friendByID = computed(() => new Map(props.friends.map((friend) => [friend.id, friend])))
const maxTide = computed(() => Math.max(1, ...tides.value.flatMap((item) => item.buckets.flat())))
const hourLabels = ['00–04', '04–08', '08–12', '12–16', '16–20', '20–24']
const dayLabels = ['日', '一', '二', '三', '四', '五', '六']
const l = (zh: string, en: string) => props.locale === 'en' ? en : zh

function notesKey() { return `vrc-harbor-passports:${props.storageKey}` }
function persistNotes() { localStorage.setItem(notesKey(), JSON.stringify(passportNotes.value)) }
function formatTime(value: string) { return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) }
function setTab(next: Tab) { tab.value = next; if ((next === 'motion' || next === 'tides') && !props.network) emit('loadNetwork') }

onMounted(() => {
  try { passportNotes.value = JSON.parse(localStorage.getItem(notesKey()) || '{}') } catch { passportNotes.value = {} }
  if (!props.network) emit('loadNetwork')
})
watch(passportNotes, persistNotes, { deep: true })
</script>

<template>
  <section class="journey-view wide-view">
    <article class="journey-bar panel">
      <div><span class="panel-kicker">{{ l('旅程工作区','Journey') }}</span><h2>{{ l('好友与足迹','Friends and trails') }}</h2><p>{{ l('看清动向、圈层和去过的地方。','See motion, circles and visited worlds.') }}</p></div>
      <nav :aria-label="l('旅程功能','Journey features')"><button :class="{active:tab==='motion'}" @click="setTab('motion')"><Radar :size="15"/>{{ l('好友动向','Motion') }}</button><button :class="{active:tab==='tides'}" @click="setTab('tides')"><Activity :size="15"/>{{ l('社交潮汐','Social tides') }}</button><button :class="{active:tab==='passport'}" @click="setTab('passport')"><NotebookTabs :size="15"/>{{ l('世界护照','World passport') }}</button><button :class="{active:tab==='readiness'}" @click="setTab('readiness')"><Route :size="15"/>{{ l('起航检查','Readiness') }}</button></nav>
    </article>

    <FriendMotion v-if="tab==='motion'" :friends="friends" :worlds="worlds" :events="events" :network="network" :media-url="mediaUrl" :locale="locale" @open-friend="emit('openFriend',$event)" @open-world="(worldId,location)=>emit('openWorld',worldId,location)" @hydrate-worlds="emit('hydrateWorlds',$event)"/>

    <template v-else-if="tab==='tides'">
      <div class="section-summary"><div><strong>{{ tides.length }}</strong><span>个朋友圈</span></div><div><strong>{{ events.length }}</strong><span>条活动记录</span></div><p>空白时段表示暂无记录。</p></div>
      <article v-if="!network" class="empty-panel panel"><RefreshCw :size="24"/><strong>正在读取关系网快照</strong><span>只读取已有 SQLite 数据，不会自动扫描好友。</span></article>
      <div v-else class="tide-list"><article v-for="item in tides.slice(0,8)" :key="item.id" class="tide-card panel"><header><div><strong>{{ item.name }}</strong><small>{{ item.memberIds.length }} 人 · {{ item.coverageDays }} 天覆盖</small></div><span>{{ item.peak }}</span></header><div class="heatmap"><div></div><small v-for="label in hourLabels" :key="label">{{ label }}</small><template v-for="(row,day) in item.buckets" :key="day"><em>{{ dayLabels[day] }}</em><i v-for="(value,hour) in row" :key="hour" :style="{opacity:String(.08 + value/maxTide*.92)}" :title="`${dayLabels[day]} ${hourLabels[hour]}：${value} 条`"></i></template></div><footer><span>{{ item.eventCount }} 条事件</span><button v-for="id in item.memberIds.slice(0,4)" :key="id" @click="emit('openFriend',id)">{{ friendByID.get(id)?.displayName || id }}</button></footer></article><article v-if="!tides.length" class="empty-panel panel"><Users :size="24"/><strong>还没有可统计的朋友圈</strong><span>关系网出现至少两位相连好友后，才会生成潮汐图。</span></article></div>
    </template>

    <div v-else-if="tab==='passport'" class="passport-layout">
      <aside class="passport-list panel"><header><strong>世界到访簿</strong><span>{{ passports.length }} 个世界</span></header><button v-for="item in passports" :key="item.worldId" :class="{active:selectedPassport?.worldId===item.worldId}" @click="selectedPassportID=item.worldId"><img v-if="item.imageUrl" :src="mediaUrl(item.imageUrl)" alt=""/><Globe2 v-else :size="20"/><span><strong>{{ item.name }}</strong><small>{{ formatTime(item.lastSeenAt) }} · {{ item.companionIds.length }} 位同行</small></span><ChevronRight :size="14"/></button><p v-if="!passports.length">活动历史里还没有世界 ID。</p></aside>
      <article v-if="selectedPassport" class="passport-detail panel"><header><div><span class="panel-kicker">世界护照</span><h2>{{ selectedPassport.name }}</h2><p>{{ formatTime(selectedPassport.firstSeenAt) }} 首次记录 · {{ selectedPassport.eventCount }} 条事件</p></div><button @click="emit('openWorld',selectedPassport.worldId,'')">打开世界详情</button></header><div class="passport-facts"><div><span>最近记录</span><strong>{{ formatTime(selectedPassport.lastSeenAt) }}</strong></div><div><span>同行好友</span><strong>{{ selectedPassport.companionIds.length }}</strong></div><div><span>世界 ID</span><strong>{{ selectedPassport.worldId }}</strong></div></div><label>印象<select v-model="selectedNote.mood"><option value="">未标记</option><option>想再去</option><option>适合聚会</option><option>适合拍照</option><option>保持观望</option></select></label><label>标签<input v-model="selectedNote.tags" placeholder="安静, 聚会, 风景"/></label><label>本机笔记<textarea v-model="selectedNote.note" rows="5" placeholder="记录路线、体验或下次要做的事"></textarea></label><div class="companion-strip"><span>同行记录</span><button v-for="id in selectedPassport.companionIds.slice(0,12)" :key="id" @click="emit('openFriend',id)">{{ friendByID.get(id)?.displayName || id }}</button></div><footer><Save :size="13"/>输入后自动保存在本机，不上传 VRChat。</footer></article>
    </div>

    <template v-else>
      <article class="readiness-hero panel"><div><span class="panel-kicker">起航检查</span><h2>{{ score }}<small>/ 100</small></h2><p>出发前检查连接和缓存。</p></div><button @click="emit('refreshSystem')"><RefreshCw :size="15"/>重新检测</button></article>
      <div class="check-grid"><article v-for="check in checks" :key="check.id" class="check-card panel" :data-state="check.state"><CheckCircle2 v-if="check.state==='ok'" :size="20"/><CircleAlert v-else-if="check.state==='error'" :size="20"/><Wifi v-else-if="check.id==='pipeline'||check.id==='route'" :size="20"/><Database v-else :size="20"/><div><strong>{{ check.label }}</strong><p>{{ check.detail }}</p></div><span>{{ check.state==='ok'?'正常':check.state==='warn'?'需留意':check.state==='error'?'异常':'待检测' }}</span></article></div>
      <aside class="readiness-note panel"><Compass :size="18"/><div><strong>线路说明</strong><p>不会自动切换代理。需要时请使用系统或自有代理。</p></div></aside>
    </template>
  </section>
</template>

<style scoped>
.journey-view{display:grid;gap:10px}.journey-bar{display:flex;align-items:flex-end;justify-content:space-between;padding:17px}.journey-bar h2{margin:3px 0;font-size:20px}.journey-bar p{margin:0;color:var(--muted);font-size:9px}.journey-bar nav{display:flex;gap:4px;padding:3px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted)}.journey-bar nav button{display:flex;align-items:center;gap:5px;padding:7px 9px;border:0;border-radius:5px;background:transparent;color:var(--muted);font-size:9px}.journey-bar nav button.active{background:var(--surface);color:var(--ink);box-shadow:0 1px 3px rgb(0 0 0/.12)}.section-summary{display:flex;align-items:center;gap:18px;padding:8px 13px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted)}.section-summary div{display:flex;align-items:baseline;gap:5px}.section-summary strong{font-size:14px}.section-summary span,.section-summary p{color:var(--muted);font-size:8px}.section-summary p{margin-left:auto}.tide-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.tide-card{padding:13px}.tide-card header{display:flex;justify-content:space-between}.tide-card header strong,.tide-card header small{display:block}.tide-card header strong{font-size:11px}.tide-card header small,.tide-card header span{margin-top:3px;color:var(--muted);font-size:8px}.heatmap{display:grid;grid-template-columns:18px repeat(6,1fr);gap:3px;margin:12px 0}.heatmap small,.heatmap em{color:var(--muted);font-size:7px;font-style:normal;text-align:center}.heatmap i{height:9px;border-radius:2px;background:var(--accent)}.tide-card footer{display:flex;align-items:center;gap:4px;border-top:1px solid var(--line);padding-top:8px}.tide-card footer span{margin-right:auto;color:var(--muted);font-size:8px}.tide-card footer button,.companion-strip button{max-width:100px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;padding:3px 5px;border:1px solid var(--line);border-radius:4px;background:transparent;color:var(--ink-soft);font-size:7px}.empty-panel{min-height:180px;display:flex;flex-direction:column;align-items:center;justify-content:center;color:var(--muted)}.empty-panel strong{margin:8px 0 3px;color:var(--ink)}.empty-panel span{font-size:8px}.passport-layout{display:grid;grid-template-columns:310px minmax(0,1fr);gap:10px}.passport-list{padding:9px}.passport-list>header{display:flex;align-items:center;justify-content:space-between;padding:6px 7px 10px}.passport-list header strong{font-size:11px}.passport-list header span{color:var(--muted);font-size:8px}.passport-list>button{width:100%;display:grid;grid-template-columns:38px 1fr 14px;align-items:center;gap:8px;padding:8px;border:0;border-radius:6px;background:transparent;color:var(--ink);text-align:left}.passport-list>button.active{background:var(--surface-muted)}.passport-list img{width:38px;height:30px;object-fit:cover;border-radius:4px}.passport-list>button span{min-width:0}.passport-list>button strong,.passport-list>button small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.passport-list>button strong{font-size:9px}.passport-list>button small{margin-top:3px;color:var(--muted);font-size:7px}.passport-list>p{margin:60px 10px;color:var(--muted);font-size:9px;text-align:center}.passport-detail{padding:17px}.passport-detail>header{display:flex;align-items:flex-start;justify-content:space-between}.passport-detail h2{margin:3px 0;font-size:18px}.passport-detail header p{margin:0;color:var(--muted);font-size:8px}.passport-detail header button{padding:7px 9px;border:1px solid var(--line);border-radius:5px;background:transparent;color:var(--ink);font-size:8px}.passport-facts{display:grid;grid-template-columns:repeat(3,1fr);gap:7px;margin:16px 0}.passport-facts div{min-width:0;padding:9px;border:1px solid var(--line);border-radius:6px;background:var(--surface-muted)}.passport-facts span,.passport-facts strong{display:block}.passport-facts span{color:var(--muted);font-size:7px}.passport-facts strong{overflow:hidden;text-overflow:ellipsis;margin-top:4px;font-size:9px;white-space:nowrap}.passport-detail>label{display:grid;gap:5px;margin-top:10px;color:var(--muted);font-size:8px}.passport-detail input,.passport-detail select,.passport-detail textarea{width:100%;padding:8px;border:1px solid var(--line);border-radius:5px;background:var(--surface-muted);color:var(--ink);font:inherit;resize:vertical}.companion-strip{display:flex;align-items:center;flex-wrap:wrap;gap:4px;margin-top:12px}.companion-strip>span{margin-right:4px;color:var(--muted);font-size:8px}.passport-detail>footer{display:flex;align-items:center;gap:5px;margin-top:12px;color:var(--muted);font-size:8px}.readiness-hero{display:flex;align-items:center;justify-content:space-between;padding:17px}.readiness-hero h2{margin:4px 0;font-size:30px}.readiness-hero h2 small{color:var(--muted);font-size:11px}.readiness-hero p{margin:0;color:var(--muted);font-size:9px}.readiness-hero button{display:flex;align-items:center;gap:5px;padding:8px 10px;border:1px solid var(--line);border-radius:5px;background:transparent;color:var(--ink);font-size:8px}.check-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px}.check-card{display:grid;grid-template-columns:24px 1fr auto;align-items:start;gap:8px;padding:14px}.check-card>svg{color:var(--muted)}.check-card[data-state=ok]>svg{color:var(--success)}.check-card[data-state=error]>svg{color:var(--danger)}.check-card strong{font-size:10px}.check-card p{margin:4px 0 0;color:var(--muted);font-size:8px;line-height:1.5}.check-card>span{padding:3px 5px;border-radius:4px;background:var(--surface-muted);color:var(--muted);font-size:7px}.readiness-note{display:flex;align-items:flex-start;gap:10px;padding:13px}.readiness-note strong{font-size:10px}.readiness-note p{margin:4px 0 0;color:var(--muted);font-size:8px;line-height:1.6}@media(max-width:1000px){.journey-bar{align-items:flex-start;flex-direction:column;gap:12px}.tide-list,.check-grid{grid-template-columns:1fr}.passport-layout{grid-template-columns:1fr}.passport-list{max-height:300px;overflow:auto}}@media(max-width:600px){.journey-bar nav{width:100%;overflow:auto}.journey-bar nav button{white-space:nowrap}.section-summary{align-items:flex-start;flex-wrap:wrap}.section-summary p{width:100%;margin:0}.passport-facts{grid-template-columns:1fr}.check-grid{grid-template-columns:1fr}}
.journey-bar h2{font-size:21px}.journey-bar p,.journey-bar nav button{font-size:11px}.journey-bar nav button{padding:8px 11px}.section-summary span,.section-summary p,.tide-card header small,.tide-card header span,.tide-card footer span,.passport-list header span,.passport-list>button small,.passport-detail header p,.passport-facts span,.passport-detail>label,.companion-strip>span,.passport-detail>footer,.readiness-hero p,.check-card p,.check-card>span,.readiness-note p{font-size:10px}.tide-card header strong,.passport-list header strong,.passport-detail header button,.passport-facts strong,.tide-card footer button,.companion-strip button,.readiness-hero button,.check-card strong,.readiness-note strong{font-size:11px}
</style>
