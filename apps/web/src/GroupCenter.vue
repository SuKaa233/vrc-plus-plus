<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Building2, CalendarDays, Clock3, Globe2, RefreshCw, Search, Users } from '@lucide/vue'
import { LocalApi, type CalendarEvent, type DataEnvelope, type Group, type GroupInstance, type GroupPost, type World } from './api'

const props = defineProps<{ userId: string; mediaUrl: (value?: string) => string }>()
const emit = defineEmits<{ openWorld: [world: World, location: string] }>()
const api = new LocalApi()
const groups = ref<DataEnvelope<Group> | null>(null)
const selected = ref<Group | null>(null)
const posts = ref<DataEnvelope<GroupPost> | null>(null)
const instances = ref<DataEnvelope<GroupInstance> | null>(null)
const calendar = ref<DataEnvelope<CalendarEvent> | null>(null)
const query = ref('')
const loading = ref(false)
const error = ref('')
const filtered = computed(() => {
  const value = query.value.trim().toLocaleLowerCase()
  return (groups.value?.items ?? []).filter((group) => !value || [group.name, group.shortCode, group.description].some((item) => item?.toLocaleLowerCase().includes(value)))
})

async function load(refresh = false) {
  if (!props.userId) return
  loading.value = true; error.value = ''
  try {
    groups.value = await api.groups(props.userId, refresh)
    if (!selected.value || !groups.value.items.some((item) => item.id === selected.value?.id)) await selectGroup(groups.value.items[0] ?? null, refresh)
  } catch (cause) { error.value = cause instanceof Error ? cause.message : '群组读取失败' }
  finally { loading.value = false }
}
async function selectGroup(group: Group | null, refresh = false) {
  selected.value = group; posts.value = null; instances.value = null; calendar.value = null
  if (!group) return
  loading.value = true
  const [postResult, instanceResult, calendarResult] = await Promise.allSettled([api.groupPosts(group.id, refresh), api.groupInstances(group.id, refresh), api.groupCalendar(group.id, new Date().toISOString().slice(0,7), refresh)])
  if (postResult.status === 'fulfilled') posts.value = postResult.value
  if (instanceResult.status === 'fulfilled') instances.value = instanceResult.value
  if (calendarResult.status === 'fulfilled') calendar.value = calendarResult.value
  if (postResult.status === 'rejected' && instanceResult.status === 'rejected' && calendarResult.status === 'rejected') error.value = '该群组公告、日历与实例暂时不可用'
  loading.value = false
}
function date(value?: string) { if (!value) return '时间未知'; return new Intl.DateTimeFormat('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'}).format(new Date(value)) }
function upcoming(items: CalendarEvent[] = []) { const cutoff=Date.now()-6*3600_000; return items.filter((item)=>new Date(item.startsAt).getTime()>=cutoff).slice(0,8) }
watch(() => props.userId, () => void load(), { immediate: true })
</script>

<template>
  <section class="group-center panel wide-view">
    <header><div><span class="panel-kicker">VRChat 群组</span><h2>群组中心</h2><p>活动、公告与实例。</p></div><button :disabled="loading" @click="load(true)"><RefreshCw :size="15" :class="{ spin: loading }" />刷新</button></header>
    <p v-if="error" class="group-error">{{ error }}</p>
    <div class="group-layout">
      <aside><label><Search :size="15" /><input v-model="query" placeholder="搜索群组" /></label><div class="group-list"><button v-for="group in filtered" :key="group.id" :class="{ active: selected?.id === group.id }" @click="selectGroup(group)"><img v-if="group.iconUrl" :src="mediaUrl(group.iconUrl)" alt="" /><span v-else><Building2 :size="18" /></span><div><strong>{{ group.name }}</strong><small>{{ group.shortCode || '无短码' }} · {{ group.memberCount ?? 0 }} 人</small></div><i v-if="group.isRepresenting">代表</i></button><p v-if="!filtered.length">没有匹配群组</p></div></aside>
      <main v-if="selected">
        <div class="group-profile"><img v-if="selected.bannerUrl" :src="mediaUrl(selected.bannerUrl)" alt="" /><div><b>{{ selected.shortCode }}</b><h3>{{ selected.name }}</h3><p>{{ selected.description || '该群组没有公开简介。' }}</p><span><Users :size="13" />{{ selected.memberCount ?? 0 }} 位成员 · {{ selected.privacy || '公开信息' }}</span></div></div>
        <div class="group-columns">
          <section><h4><Globe2 :size="15" />活跃实例 <span>{{ instances?.items.length ?? 0 }}</span></h4><button v-for="item in instances?.items" :key="item.location" class="instance-row" @click="emit('openWorld', item.world, item.location)"><img v-if="item.world.thumbnailImageUrl" :src="mediaUrl(item.world.thumbnailImageUrl)" alt="" /><span v-else><Globe2 :size="18" /></span><div><strong>{{ item.world.name || item.world.id }}</strong><small>{{ item.memberCount }} 位群组成员 · 点击查看实例</small></div></button><p v-if="!instances?.items.length" class="empty-copy">当前没有可见的群组实例。</p></section>
          <section><h4><CalendarDays :size="15" />本月活动 <span>{{ upcoming(calendar?.items).length }}</span></h4><article v-for="event in upcoming(calendar?.items)" :key="event.id" class="calendar-row"><div><strong>{{ event.title || '群组活动' }}</strong><time>{{ date(event.startsAt) }}</time></div><p>{{ event.description || `${event.category || '活动'} · ${event.interestedUserCount ?? 0} 人感兴趣` }}</p><small><Clock3 :size="11" />{{ event.occurrenceKind === 'recurring' ? '重复活动' : '单次活动' }}<i v-if="event.following">已关注</i></small></article><p v-if="!upcoming(calendar?.items).length" class="empty-copy">本月没有即将开始的可见活动。</p></section>
          <section><h4><CalendarDays :size="15" />最近公告 <span>{{ posts?.items.length ?? 0 }}</span></h4><article v-for="post in posts?.items" :key="post.id"><div><strong>{{ post.title || '群组公告' }}</strong><time>{{ date(post.createdAt) }}</time></div><p>{{ post.text || '仅包含图片的公告' }}</p><img v-if="post.imageUrl" :src="mediaUrl(post.imageUrl)" alt="" /></article><p v-if="!posts?.items.length" class="empty-copy">没有可见公告。</p></section>
        </div>
      </main>
      <div v-else class="group-empty"><Building2 :size="28" /><strong>尚未读取到群组</strong><span>部分隐私群组可能不会出现在公开成员接口中。</span></div>
    </div>
  </section>
</template>

<style scoped>
.group-center{padding:0;overflow:hidden}.group-center>header{display:flex;justify-content:space-between;align-items:flex-start;padding:18px 20px;border-bottom:1px solid var(--line)}header h2{margin:3px 0;font-size:18px}header p,.empty-copy,.group-empty span{margin:0;color:var(--muted);font-size:9px}header button{display:flex;align-items:center;gap:6px;padding:7px 10px;border:1px solid var(--line);border-radius:7px;color:var(--ink-soft);background:var(--surface)}.group-error{margin:0;padding:9px 20px;color:var(--danger);background:var(--danger-soft);font-size:9px}.group-layout{display:grid;grid-template-columns:290px 1fr;min-height:650px}.group-layout>aside{padding:12px;border-right:1px solid var(--line);background:var(--surface-muted)}aside>label{height:34px;display:flex;align-items:center;gap:7px;padding:0 9px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--muted)}aside input{width:100%;border:0;outline:0;background:transparent;color:var(--ink);font-size:10px}.group-list{display:grid;gap:5px;margin-top:9px}.group-list button{display:grid;grid-template-columns:38px 1fr auto;align-items:center;gap:9px;padding:7px;border:1px solid transparent;border-radius:7px;background:transparent;color:var(--ink);text-align:left}.group-list button:hover,.group-list button.active{border-color:var(--line);background:var(--surface)}.group-list img,.group-list button>span{width:38px;height:38px;border-radius:8px;object-fit:cover}.group-list button>span{display:grid;place-items:center;background:var(--accent-soft);color:var(--accent)}.group-list strong,.group-list small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.group-list strong{font-size:11px}.group-list small{margin-top:3px;color:var(--muted);font-size:8px}.group-list i{padding:3px 5px;border-radius:4px;color:var(--accent);background:var(--accent-soft);font-size:8px;font-style:normal}.group-layout>main{min-width:0;padding:16px}.group-profile{position:relative;min-height:118px;overflow:hidden;border:1px solid var(--line);border-radius:9px;background:var(--surface-muted)}.group-profile>img{position:absolute;inset:0;width:100%;height:100%;object-fit:cover;opacity:.16}.group-profile>div{position:relative;padding:16px}.group-profile b{color:var(--accent);font-size:9px}.group-profile h3{margin:4px 0;font-size:20px}.group-profile p{max-width:720px;margin:0;color:var(--ink-soft);font-size:10px;line-height:1.55}.group-profile span{display:flex;align-items:center;gap:5px;margin-top:11px;color:var(--muted);font-size:9px}.group-columns{display:grid;grid-template-columns:minmax(260px,.8fr) minmax(320px,1.2fr);gap:12px;margin-top:12px}.group-columns>section{padding:12px;border:1px solid var(--line);border-radius:9px;background:var(--surface-muted)}h4{display:flex;align-items:center;gap:6px;margin:0 0 10px;font-size:11px}h4 span{margin-left:auto;color:var(--muted);font-size:9px}.instance-row{width:100%;display:grid;grid-template-columns:42px 1fr;align-items:center;gap:9px;margin-top:6px;padding:6px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--ink);text-align:left}.instance-row img,.instance-row>span{width:42px;height:34px;border-radius:6px;object-fit:cover}.instance-row>span{display:grid;place-items:center;color:var(--accent);background:var(--accent-soft)}.instance-row strong,.instance-row small{display:block}.instance-row strong{font-size:10px}.instance-row small{margin-top:3px;color:var(--muted);font-size:8px}.group-columns article{padding:9px 0;border-top:1px solid var(--line)}.group-columns article:first-of-type{border-top:0}.group-columns article>div{display:flex;justify-content:space-between;gap:10px}.group-columns article strong{font-size:10px}.group-columns time{color:var(--muted);font-size:8px}.group-columns article p{margin:5px 0 0;color:var(--ink-soft);font-size:9px;line-height:1.55;white-space:pre-wrap}.group-columns article img{max-width:180px;max-height:100px;margin-top:7px;border-radius:6px}.group-empty{display:flex;flex-direction:column;align-items:center;justify-content:center;color:var(--muted)}.group-empty strong{margin:9px 0 4px;color:var(--ink)}@media(max-width:900px){.group-layout{grid-template-columns:1fr}.group-layout>aside{border-right:0;border-bottom:1px solid var(--line)}.group-list{grid-template-columns:repeat(2,1fr)}.group-columns{grid-template-columns:1fr}}@media(max-width:560px){.group-list{grid-template-columns:1fr}.group-layout>main{padding:10px}}
.group-columns{grid-template-columns:repeat(3,minmax(0,1fr))}.calendar-row>small{display:flex;align-items:center;gap:4px;margin-top:6px;color:var(--muted);font-size:8px}.calendar-row>small i{margin-left:auto;color:var(--accent);font-style:normal}@media(max-width:1100px){.group-columns{grid-template-columns:1fr}}
</style>
