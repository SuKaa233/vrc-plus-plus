<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Bookmark, Search, SlidersHorizontal, Tags, Trash2, Users } from '@lucide/vue'
import type { Friend, FriendAnnotation } from './api'
import { preferredFriendAvatar } from './media'

const props = defineProps<{
  friends: Friend[]
  annotations: Record<string, FriendAnnotation>
  mediaUrl: (value?: string) => string
  storageKey: string
  source: string
  message?: string
}>()
const emit = defineEmits<{ openFriend: [friend: Friend] }>()

const query = ref('')
const statusFilter = ref('all')
const platformFilter = ref('all')
const tagFilter = ref('')
const sortBy = ref('status')
const scrollTop = ref(0)
const viewportHeight = ref(620)
const columnCount = ref(2)
const viewportRef = ref<HTMLElement | null>(null)
const avatarFailures = ref<Set<string>>(new Set())
type FilterPreset = { name: string; query: string; statusFilter: string; platformFilter: string; tagFilter: string; sortBy: string }
const presets = ref<FilterPreset[]>([])
const selectedPreset = ref('')
let resizeObserver: ResizeObserver | undefined
let saveTimer: number | undefined
const rowHeight = 70
const overscan = 6

const tags = computed(() => [...new Set(Object.values(props.annotations).flatMap((annotation) => annotation.tags ?? []))].sort((left, right) => left.localeCompare(right)))
const platforms = computed(() => [...new Set(props.friends.map((friend) => friend.platform || friend.lastPlatform).filter(Boolean) as string[])].sort())

function isJoinable(friend: Friend) {
  const location = friend.location ?? ''
  return friend.online && location.startsWith('wrld_') && !location.includes('~private(')
}

function matchesStatus(friend: Friend) {
  if (statusFilter.value === 'online') return friend.online
  if (statusFilter.value === 'joinable') return isJoinable(friend)
  if (statusFilter.value === 'private') return friend.online && (!friend.location || friend.location === 'private' || friend.location.includes('~private('))
  if (statusFilter.value === 'offline') return !friend.online
  return true
}

const filtered = computed(() => {
  const text = query.value.trim().toLocaleLowerCase()
  const result = props.friends.filter((friend) => {
    const annotation = props.annotations[friend.id]
    if (!matchesStatus(friend)) return false
    if (platformFilter.value !== 'all' && (friend.platform || friend.lastPlatform) !== platformFilter.value) return false
    if (tagFilter.value && !annotation?.tags?.includes(tagFilter.value)) return false
    if (!text) return true
    return [friend.displayName, friend.id, friend.bio, friend.statusDescription, friend.lastPlatform, annotation?.group, annotation?.note, ...(annotation?.tags ?? [])]
      .some((value) => value?.toLocaleLowerCase().includes(text))
  })
  return result.sort((left, right) => {
    if (sortBy.value === 'name') return left.displayName.localeCompare(right.displayName)
    if (sortBy.value === 'platform') return (left.platform || left.lastPlatform || '').localeCompare(right.platform || right.lastPlatform || '') || left.displayName.localeCompare(right.displayName)
    if (sortBy.value === 'activity') return Date.parse(right.lastActivity || right.lastLogin || '1970-01-01') - Date.parse(left.lastActivity || left.lastLogin || '1970-01-01') || left.displayName.localeCompare(right.displayName)
    if (sortBy.value === 'tagged') return Number(Boolean(props.annotations[right.id])) - Number(Boolean(props.annotations[left.id])) || left.displayName.localeCompare(right.displayName)
    if (sortBy.value === 'default') return 0
    return Number(right.online) - Number(left.online) || Number(isJoinable(right)) - Number(isJoinable(left)) || left.displayName.localeCompare(right.displayName)
  })
})

const startRow = computed(() => Math.max(0, Math.floor(scrollTop.value / rowHeight) - overscan))
const endRow = computed(() => Math.ceil((scrollTop.value + viewportHeight.value) / rowHeight) + overscan)
const virtualItems = computed(() => {
  const start = startRow.value * columnCount.value
  const end = Math.min(filtered.value.length, endRow.value * columnCount.value)
  return filtered.value.slice(start, end).map((friend, offset) => {
    const index = start + offset
    return { friend, index, row: Math.floor(index / columnCount.value), column: index % columnCount.value }
  })
})
const virtualHeight = computed(() => Math.ceil(filtered.value.length / columnCount.value) * rowHeight)

function avatar(friend: Friend) {
  if (avatarFailures.value.has(friend.id)) return ''
  return props.mediaUrl(preferredFriendAvatar(friend))
}

function markAvatarFailed(userID: string) {
  avatarFailures.value = new Set([...avatarFailures.value, userID])
}

function locationLabel(friend: Friend) {
  if (!friend.online) return '离线'
  if (isJoinable(friend)) return '可加入实例'
  if (!friend.location) return '在线 · 位置受限'
  if (friend.location === 'private' || friend.location.includes('~private(')) return '私人实例'
  return friend.location === 'traveling' ? '切换世界中' : '在线'
}

function platformLabel(value?: string) {
  return ({ standalonewindows: 'PC', android: 'Quest / Android', ios: 'iOS', web: 'Web' } as Record<string, string>)[value ?? ''] || value || '未知平台'
}

function activityLabel(friend: Friend) {
  const value = friend.lastActivity || friend.lastLogin || friend.lastMobile
  if (!value) return friend.statusDescription || '暂无公开活动时间'
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return value
  const days = Math.max(0, Math.floor((Date.now() - time.getTime()) / 86400000))
  return days === 0 ? `最近活动 ${time.toLocaleTimeString('zh-CN', { hour:'2-digit', minute:'2-digit' })}` : `最近活动 ${days} 天前`
}

function itemPosition(item: { row: number; column: number }) {
  if (columnCount.value === 1) return { transform: `translateY(${item.row * rowHeight}px)`, left: '12px', right: '12px' }
  return item.column === 0
    ? { transform: `translateY(${item.row * rowHeight}px)`, left: '12px', right: 'calc(50% + 3px)' }
    : { transform: `translateY(${item.row * rowHeight}px)`, left: 'calc(50% + 3px)', right: '12px' }
}

function saveState() {
  window.clearTimeout(saveTimer)
  saveTimer = window.setTimeout(() => {
    localStorage.setItem(`vrc-harbor-friends:${props.storageKey}`, JSON.stringify({
      query: query.value, statusFilter: statusFilter.value, platformFilter: platformFilter.value,
      tagFilter: tagFilter.value, sortBy: sortBy.value, scrollTop: scrollTop.value,
    }))
  }, 180)
}

function presetKey() { return `vrc-harbor-friend-presets:${props.storageKey}` }
function persistPresets() { localStorage.setItem(presetKey(), JSON.stringify(presets.value)) }
function savePreset() {
  const name = window.prompt('给当前筛选方案命名（最多 24 个字符）')?.trim().slice(0, 24)
  if (!name) return
  const preset = { name, query: query.value, statusFilter: statusFilter.value, platformFilter: platformFilter.value, tagFilter: tagFilter.value, sortBy: sortBy.value }
  presets.value = [...presets.value.filter((item) => item.name !== name), preset].slice(-12)
  selectedPreset.value = name
  persistPresets()
}
function applyPreset() {
  const preset = presets.value.find((item) => item.name === selectedPreset.value)
  if (!preset) return
  query.value = preset.query; statusFilter.value = preset.statusFilter; platformFilter.value = preset.platformFilter; tagFilter.value = preset.tagFilter; sortBy.value = preset.sortBy
}
function deletePreset() {
  if (!selectedPreset.value) return
  presets.value = presets.value.filter((item) => item.name !== selectedPreset.value)
  selectedPreset.value = ''
  persistPresets()
}

watch([query, statusFilter, platformFilter, tagFilter, sortBy], () => {
  scrollTop.value = 0
  if (viewportRef.value) viewportRef.value.scrollTop = 0
  saveState()
})
watch(scrollTop, saveState)

onMounted(() => {
  try { presets.value = JSON.parse(localStorage.getItem(presetKey()) || '[]') } catch { presets.value = [] }
  try {
    const saved = JSON.parse(localStorage.getItem(`vrc-harbor-friends:${props.storageKey}`) || 'null')
    if (saved) {
      query.value = saved.query ?? ''
      statusFilter.value = saved.statusFilter ?? 'all'
      platformFilter.value = saved.platformFilter ?? 'all'
      tagFilter.value = saved.tagFilter ?? ''
      sortBy.value = saved.sortBy ?? 'status'
      scrollTop.value = Math.max(0, Number(saved.scrollTop) || 0)
    }
  } catch { /* use defaults */ }
  resizeObserver = new ResizeObserver(([entry]) => {
    viewportHeight.value = entry.contentRect.height
    columnCount.value = entry.contentRect.width >= 900 ? 2 : 1
  })
  if (viewportRef.value) resizeObserver.observe(viewportRef.value)
  void nextTick(() => { if (viewportRef.value) viewportRef.value.scrollTop = scrollTop.value })
})
onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  window.clearTimeout(saveTimer)
})
</script>

<template>
  <article class="friend-workbench panel wide-view">
    <header class="workbench-heading">
      <div><span>好友工作台</span><h2>{{ filtered.length }} / {{ friends.length }} 位好友</h2><p>筛选、排序和滚动位置保存在当前浏览器。</p></div>
      <span class="source-badge">{{ source === 'cache' ? '本地快照' : '实时数据' }}</span>
    </header>
    <p v-if="message" class="workbench-note">{{ message }}</p>
    <div class="friend-filters">
      <label class="friend-query"><Search :size="16" /><input v-model="query" placeholder="搜索昵称、ID、备注、分组或标签" /></label>
      <label><SlidersHorizontal :size="14" /><select v-model="statusFilter"><option value="all">全部状态</option><option value="online">仅在线</option><option value="joinable">可加入</option><option value="private">私人实例</option><option value="offline">离线</option></select></label>
      <label><Users :size="14" /><select v-model="platformFilter"><option value="all">全部平台</option><option v-for="platform in platforms" :key="platform" :value="platform">{{ platformLabel(platform) }}</option></select></label>
      <label><Tags :size="14" /><select v-model="tagFilter"><option value="">全部标签</option><option v-for="tag in tags" :key="tag" :value="tag"># {{ tag }}</option></select></label>
      <label>排序<select v-model="sortBy"><option value="status">在线与可加入优先</option><option value="activity">最近活动</option><option value="name">名称</option><option value="platform">平台</option><option value="tagged">已整理优先</option><option value="default">VRChat 默认顺序</option></select></label>
      <label><Bookmark :size="14" /><select v-model="selectedPreset" @change="applyPreset"><option value="">筛选方案</option><option v-for="preset in presets" :key="preset.name" :value="preset.name">{{ preset.name }}</option></select></label>
      <button class="filter-action" title="保存当前筛选方案" @click="savePreset"><Bookmark :size="14" />保存</button>
      <button v-if="selectedPreset" class="filter-action danger" title="删除当前方案" @click="deletePreset"><Trash2 :size="14" /></button>
    </div>
    <div ref="viewportRef" class="virtual-friend-list" @scroll="scrollTop = ($event.target as HTMLElement).scrollTop">
      <div class="virtual-spacer" :style="{ height: `${virtualHeight}px` }">
        <button v-for="item in virtualItems" :key="item.friend.id" type="button" class="virtual-friend" :class="{ offline: !item.friend.online }" :style="{ ...itemPosition(item), borderLeftColor: annotations[item.friend.id]?.color || 'transparent' }" @click="emit('openFriend', item.friend)">
          <img v-if="avatar(item.friend)" :src="avatar(item.friend)" alt="" loading="lazy" decoding="async" @error="markAvatarFailed(item.friend.id)" />
          <span v-else class="virtual-avatar-fallback">{{ item.friend.displayName.slice(0, 1) }}</span>
          <span class="virtual-presence" :class="{ online: item.friend.online }"></span>
          <span class="virtual-copy"><strong>{{ item.friend.displayName }}</strong><small>{{ locationLabel(item.friend) }} · {{ platformLabel(item.friend.platform || item.friend.lastPlatform) }}</small><em>{{ annotations[item.friend.id]?.group || annotations[item.friend.id]?.note || activityLabel(item.friend) }}</em></span>
          <span class="virtual-tags"><i v-for="tag in annotations[item.friend.id]?.tags?.slice(0, 4)" :key="tag">#{{ tag }}</i></span>
        </button>
      </div>
      <div v-if="!filtered.length" class="friend-list-empty"><Users :size="24" /><strong>没有匹配好友</strong><span>调整状态、平台或标签筛选条件。</span></div>
    </div>
  </article>
</template>

<style scoped>
.friend-workbench{padding:0;overflow:hidden}.workbench-heading{display:flex;align-items:flex-start;justify-content:space-between;padding:18px 20px;border-bottom:1px solid var(--line)}.workbench-heading span{color:var(--muted);font-size:9px}.workbench-heading h2{margin:4px 0 0;font-size:18px}.workbench-heading p{margin:4px 0 0;color:var(--muted);font-size:9px}.source-badge{padding:5px 8px;border-radius:99px;color:var(--success)!important;background:var(--success-soft)}.workbench-note{margin:10px 20px 0;padding-left:8px;border-left:2px solid var(--warning);color:var(--warning);font-size:9px}.friend-filters{display:flex;align-items:center;gap:7px;padding:12px 20px;border-bottom:1px solid var(--line);background:var(--surface-muted)}.friend-filters label{height:34px;display:flex;align-items:center;gap:6px;padding:0 8px;border:1px solid var(--line);border-radius:6px;color:var(--muted);background:var(--surface);font-size:9px}.friend-filters select,.friend-filters input{height:30px;border:0;outline:0;color:var(--ink-soft);background:transparent;font-size:10px}.friend-query{flex:1;min-width:220px}.friend-query input{width:100%}.virtual-friend-list{height:min(72vh,760px);min-height:520px;overflow:auto;position:relative;background:var(--surface-muted)}.virtual-spacer{position:relative}.virtual-friend{position:absolute;left:12px;right:12px;top:0;height:64px;display:grid;grid-template-columns:44px 1fr auto;align-items:center;gap:10px;padding:7px 12px;border:1px solid var(--line);border-left-width:3px;border-radius:7px;color:var(--ink);background:var(--surface);text-align:left;cursor:pointer}.virtual-friend:hover{border-color:var(--line-strong);background:var(--surface-hover)}.virtual-friend.offline{opacity:.62}.virtual-friend img,.virtual-avatar-fallback{width:42px;height:42px;border-radius:8px;object-fit:cover}.virtual-avatar-fallback{display:grid;place-items:center;color:var(--accent);background:var(--accent-soft);font-weight:750}.virtual-presence{width:9px;height:9px;position:absolute;left:47px;bottom:8px;border:2px solid var(--surface);border-radius:50%;background:var(--muted)}.virtual-presence.online{background:var(--success)}.virtual-copy{min-width:0}.virtual-copy strong,.virtual-copy small,.virtual-copy em{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.virtual-copy strong{font-size:12px}.virtual-copy small{margin-top:2px;color:var(--muted);font-size:9px}.virtual-copy em{margin-top:3px;color:var(--muted);font-size:9px;font-style:normal}.virtual-tags{display:flex;justify-content:flex-end;gap:4px;max-width:300px}.virtual-tags i{padding:3px 5px;border-radius:4px;color:var(--accent);background:var(--accent-soft);font-size:8px;font-style:normal}.friend-list-empty{height:100%;display:flex;flex-direction:column;align-items:center;justify-content:center;color:var(--muted)}.friend-list-empty strong{margin-top:9px;color:var(--ink)}.friend-list-empty span{margin-top:4px;font-size:9px}@media(max-width:1000px){.friend-filters{flex-wrap:wrap}.friend-query{width:100%;flex-basis:100%}.virtual-tags{display:none}}@media(max-width:620px){.workbench-heading,.friend-filters{padding-inline:12px}.friend-filters label{flex:1}.virtual-friend{left:7px;right:7px}.virtual-friend-list{min-height:480px}}
.filter-action{height:34px;display:flex;align-items:center;gap:5px;padding:0 8px;border:1px solid var(--line);border-radius:6px;background:var(--surface);color:var(--ink-soft);font-size:9px}.filter-action.danger{color:var(--danger)}
</style>
