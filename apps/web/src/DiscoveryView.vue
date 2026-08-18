<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Bookmark, BookmarkCheck, Check, Clock3, ExternalLink, Link2, LoaderCircle, Search, ShieldCheck, UserPlus, Users } from '@lucide/vue'
import type { ActivityEvent, Friend, FriendStatus, Group, GroupMember, UserProfile } from './api'
import { optimizedVrcImageUrl, preferredFriendAvatar } from './media'

const props = defineProps<{
  friends: Friend[]; events: ActivityEvent[]; results: UserProfile[]; statuses: Record<string, FriendStatus>
  groups: Group[]; groupMembers: GroupMember[]; expandedUser?: UserProfile | null; selectedGroupId?: string
  loading: boolean; expanding: boolean; actingId: string; clueError?: string; storageKey: string
  mediaUrl: (value?: string) => string
}>()
const emit = defineEmits<{
  search: [query: string]; open: [user: UserProfile]; request: [user: UserProfile]
  expandUser: [user: UserProfile]; loadGroup: [group: Group]
}>()

type SavedPerson = { id: string; displayName: string; imageUrl?: string; savedAt: string }
const query = ref('')
const searched = ref(false)
const saved = ref<SavedPerson[]>([])
const friendIDs = computed(() => new Set(props.friends.map(item => item.id)))
const selectedGroup = computed(() => props.groups.find(item => item.id === props.selectedGroupId))
const storageName = computed(() => `vrcpp-strangers:${props.storageKey}`)
const recentUsers = computed(() => {
  const byID = new Map<string, { user: UserProfile; count: number; latest: string }>()
  for (const event of props.events) {
    if (!event.userId || friendIDs.value.has(event.userId)) continue
    const current = byID.get(event.userId)
    if (current) { current.count += 1; if (event.observedAt > current.latest) current.latest = event.observedAt; continue }
    byID.set(event.userId, { user: { id: event.userId, displayName: event.displayName || event.userId, isFriend: false, allowAvatarCopying: false, trustLevel: 'visitor' }, count: 1, latest: event.observedAt })
  }
  return [...byID.values()].sort((a, b) => b.latest.localeCompare(a.latest)).slice(0, 24)
})
const publicCandidates = computed(() => props.groupMembers
  .filter(item => !friendIDs.value.has(item.userId) && item.userId !== props.expandedUser?.id)
  .map(item => ({ id: item.userId, displayName: item.displayName, status: item.status, statusDescription: item.statusDescription,
    imageUrl: item.imageUrl || item.thumbnailUrl, iconUrl: item.iconUrl, profilePicOverride: item.profilePicOverride, profilePicOverrideThumbnail: item.profilePicOverrideThumbnail,
    currentAvatarThumbnailImageUrl: item.currentAvatarThumbnailImageUrl, isFriend: false, allowAvatarCopying: false, trustLevel: 'visitor' } satisfies UserProfile)))

function loadSaved() { try { saved.value = JSON.parse(localStorage.getItem(storageName.value) || '[]') } catch { saved.value = [] } }
function persistSaved() { localStorage.setItem(storageName.value, JSON.stringify(saved.value.slice(0, 200))) }
function isSaved(id: string) { return saved.value.some(item => item.id === id) }
function toggleSaved(user: UserProfile) {
  if (isSaved(user.id)) saved.value = saved.value.filter(item => item.id !== user.id)
  else saved.value = [{ id: user.id, displayName: user.displayName, imageUrl: user.profilePicOverrideThumbnail || optimizedVrcImageUrl(user.profilePicOverride) || preferredFriendAvatar(user), savedAt: new Date().toISOString() }, ...saved.value]
  persistSaved()
}
function savedProfile(item: SavedPerson): UserProfile { return { id: item.id, displayName: item.displayName, imageUrl: item.imageUrl, isFriend: false, allowAvatarCopying: false, trustLevel: 'visitor' } }
function submit() { const value = query.value.trim(); if (value.length >= 2) { searched.value = true; emit('search', value) } }
function avatar(user: UserProfile) { return props.mediaUrl(user.profilePicOverrideThumbnail || optimizedVrcImageUrl(user.profilePicOverride) || preferredFriendAvatar(user)) }
function relationLabel(user: UserProfile) {
  const status = props.statuses[user.id]
  if (user.isFriend || status?.isFriend || friendIDs.value.has(user.id)) return '已是好友'
  if (status?.incomingRequest) return '对方已申请'
  if (status?.outgoingRequest) return '等待接受'
  return '加好友'
}
function evidence(user: UserProfile) {
  const parts: string[] = []
  if (user.mutualFriendCount) parts.push(`${user.mutualFriendCount} 个共同好友`)
  if (user.mutualGroupCount) parts.push(`${user.mutualGroupCount} 个共同群组`)
  if (user.representedGroup?.name) parts.push(`代表 ${user.representedGroup.name}`)
  return parts.join(' · ') || user.statusDescription || '点开可补全公开资料与关系计数'
}
function dateText(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '本机记录' : date.toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }) }
onMounted(loadSaved)
watch(storageName, loadSaved)
</script>

<template>
  <section class="stranger-view wide-view">
    <article class="stranger-hero">
      <div class="hero-copy"><span>陌生人资料库</span><h2>从公开线索找到好友圈外的人</h2><p>搜索、同房日志、共同好友计数和可见群组成员统一整理。</p></div>
      <form @submit.prevent="submit"><Search :size="17" /><input v-model="query" placeholder="显示名称或完整 usr_ ID" aria-label="搜索陌生人" /><button :disabled="loading || query.trim().length < 2"><LoaderCircle v-if="loading" class="spin" :size="16" /><Search v-else :size="16" />搜索</button></form>
      <div class="boundary"><ShieldCheck :size="16" /><span><strong>能力边界</strong> VRChat 不提供任意用户的完整好友列表；这里用共同关系、公开群组和本机同房记录扩展，不绕过隐私设置。</span></div>
    </article>
    <div class="source-stats">
      <div><Search :size="16" /><span><strong>{{ results.length }}</strong>搜索结果</span></div><div><Clock3 :size="16" /><span><strong>{{ recentUsers.length }}</strong>最近遇见</span></div>
      <div><Users :size="16" /><span><strong>{{ publicCandidates.length }}</strong>群组候选</span></div><div><Bookmark :size="16" /><span><strong>{{ saved.length }}</strong>本地收藏</span></div>
    </div>

    <article v-if="results.length" class="stranger-section">
      <header><div><Search :size="17" /><strong>搜索结果</strong></div><span>先展开关系线索，再决定是否收藏或加好友</span></header>
      <div class="person-grid"><div v-for="user in results" :key="user.id" class="person-card">
        <button class="person-main" @click="emit('open', user)"><span class="person-avatar"><img v-if="avatar(user)" :src="avatar(user)" alt="" loading="lazy" /><b v-else>{{ user.displayName.slice(0, 1) }}</b></span><span><strong>{{ user.displayName }}</strong><small>{{ evidence(user) }}</small></span></button>
        <div class="person-actions"><button title="查看关系线索" @click="emit('expandUser', user)"><Link2 :size="14" />关系线索</button><button :class="{ saved: isSaved(user.id) }" title="仅保存在本机" @click="toggleSaved(user)"><BookmarkCheck v-if="isSaved(user.id)" :size="14" /><Bookmark v-else :size="14" />{{ isSaved(user.id) ? '已收藏' : '收藏' }}</button><button :disabled="actingId === user.id || relationLabel(user) !== '加好友'" @click="emit('request', user)"><LoaderCircle v-if="actingId === user.id" class="spin" :size="14" /><Check v-else-if="relationLabel(user) !== '加好友'" :size="14" /><UserPlus v-else :size="14" />{{ relationLabel(user) }}</button></div>
      </div></div>
    </article>
    <article v-else-if="searched && !loading" class="stranger-section empty"><Search :size="18" /><strong>没有找到匹配用户</strong><span>可直接输入完整 usr_ 用户 ID。</span></article>

    <article v-if="expandedUser || expanding || clueError" class="stranger-section clue-section">
      <header><div><Link2 :size="17" /><strong>{{ expandedUser?.displayName || '关系线索' }}</strong></div><span>公开群组是关系线索，不代表好友关系</span></header>
      <div v-if="expanding" class="empty"><LoaderCircle class="spin" :size="18" />正在读取可见群组…</div><div v-else-if="clueError" class="clue-error">{{ clueError }}</div>
      <template v-else><div class="group-strip"><button v-for="group in groups" :key="group.id" :class="{ active: selectedGroupId === group.id }" @click="emit('loadGroup', group)"><span>{{ group.name }}</span><small>{{ group.memberCount ? `${group.memberCount} 人` : group.shortCode || '公开群组' }}</small></button><span v-if="!groups.length">该用户没有对你公开群组列表</span></div>
        <div v-if="selectedGroupId" class="group-result-title"><span>{{ selectedGroup?.name }}</span><small>最多读取 100 位有权查看的成员，已排除你的好友</small></div>
        <div v-if="publicCandidates.length" class="candidate-grid"><button v-for="user in publicCandidates" :key="user.id" @click="emit('open', user)"><span class="mini-avatar"><img v-if="avatar(user)" :src="avatar(user)" alt="" loading="lazy" /><b v-else>{{ user.displayName.slice(0, 1) }}</b></span><span><strong>{{ user.displayName }}</strong><small>{{ user.statusDescription || '来自可见群组成员' }}</small></span><ExternalLink :size="13" /></button></div>
      </template>
    </article>

    <div class="lower-grid">
      <article class="stranger-section"><header><div><Clock3 :size="17" /><strong>最近遇见</strong></div><span>本机 VRChat 日志证据</span></header><div v-if="recentUsers.length" class="compact-list"><button v-for="item in recentUsers" :key="item.user.id" @click="emit('open', item.user)"><span><strong>{{ item.user.displayName }}</strong><small>{{ item.count }} 条记录 · 最近 {{ dateText(item.latest) }}</small></span><ExternalLink :size="13" /></button></div><div v-else class="empty"><Users :size="18" />暂无可识别的非好友同房记录</div></article>
      <article class="stranger-section"><header><div><Bookmark :size="17" /><strong>本地收藏</strong></div><span>最多 200 位，仅保存在本机</span></header><div v-if="saved.length" class="compact-list"><button v-for="item in saved" :key="item.id" @click="emit('open', savedProfile(item))"><span><strong>{{ item.displayName }}</strong><small>{{ item.id }}</small></span><ExternalLink :size="13" /></button></div><div v-else class="empty"><Bookmark :size="18" />尚未收藏陌生人</div></article>
    </div>
  </section>
</template>

<style scoped>
.stranger-view{display:grid;gap:12px}.stranger-hero,.stranger-section,.source-stats{border:1px solid var(--line);border-radius:9px;background:var(--surface)}.stranger-hero{display:grid;grid-template-columns:minmax(280px,.85fr) minmax(360px,1.15fr);align-items:end;gap:18px;padding:20px}.hero-copy>span{color:var(--muted);font-size:10px}.hero-copy h2{margin:5px 0 6px;font-size:20px}.hero-copy p{margin:0;color:var(--muted);font-size:10px}.stranger-hero form{height:42px;display:flex;align-items:center;gap:8px;padding-left:12px;border:1px solid var(--line-strong);border-radius:7px;background:var(--surface-muted);color:var(--muted)}.stranger-hero input{min-width:0;flex:1;border:0;outline:0;background:transparent;color:var(--ink);font-size:11px}.stranger-hero form button{height:100%;padding:0 15px;border:0;border-left:1px solid var(--line);border-radius:0 6px 6px 0;color:#fff;background:var(--accent);cursor:pointer}.stranger-hero form button:disabled{opacity:.55}.boundary{grid-column:1/-1;display:flex;align-items:flex-start;gap:8px;padding:9px 11px;border-radius:7px;background:var(--surface-muted);color:var(--muted);font-size:9px;line-height:1.5}.boundary svg{flex:none;color:var(--accent)}.boundary strong{color:var(--ink-soft)}.source-stats{display:grid;grid-template-columns:repeat(4,1fr);padding:9px}.source-stats>div{display:flex;align-items:center;justify-content:center;gap:8px;min-height:36px;border-right:1px solid var(--line);color:var(--muted)}.source-stats>div:last-child{border:0}.source-stats svg{color:var(--accent)}.source-stats span{display:flex;align-items:baseline;gap:5px;font-size:9px}.source-stats strong{font-size:15px;color:var(--ink)}.stranger-section{padding:14px}.stranger-section>header{display:flex;align-items:center;justify-content:space-between;margin-bottom:10px}.stranger-section>header div{display:flex;align-items:center;gap:7px}.stranger-section>header svg{color:var(--accent)}.stranger-section>header strong{font-size:12px}.stranger-section>header>span{color:var(--muted);font-size:9px}.person-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}.person-card{min-width:0;padding:8px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.person-main{width:100%;min-width:0;display:grid;grid-template-columns:42px minmax(0,1fr);align-items:center;gap:9px;padding:0;border:0;background:transparent;color:var(--ink);text-align:left;cursor:pointer}.person-avatar,.person-avatar img{width:42px;height:42px;border-radius:9px}.person-avatar{display:grid;place-items:center;overflow:hidden;background:var(--surface);color:var(--muted)}.person-avatar img,.mini-avatar img{object-fit:cover}.person-main strong,.person-main small,.candidate-grid strong,.candidate-grid small,.compact-list strong,.compact-list small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.person-main strong{font-size:11px}.person-main small{margin-top:4px;color:var(--muted);font-size:9px}.person-actions{display:flex;justify-content:flex-end;gap:5px;margin-top:8px}.person-actions button{height:27px;display:inline-flex;align-items:center;gap:4px;padding:0 8px;border:1px solid var(--line);border-radius:6px;background:var(--surface);color:var(--ink-soft);font-size:9px;cursor:pointer}.person-actions button.saved{color:var(--accent)}.person-actions button:disabled{opacity:.5}.clue-section{border-color:color-mix(in srgb,var(--accent) 25%,var(--line))}.group-strip{display:flex;gap:6px;overflow:auto;padding-bottom:4px}.group-strip>button{min-width:145px;padding:8px 10px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted);color:var(--ink);text-align:left;cursor:pointer}.group-strip>button.active{border-color:var(--accent);box-shadow:inset 0 0 0 1px var(--accent)}.group-strip>span{padding:16px;color:var(--muted);font-size:9px}.group-strip button span,.group-strip button small{display:block}.group-strip button span{font-size:10px;font-weight:650}.group-strip button small{margin-top:3px;color:var(--muted);font-size:8px}.group-result-title{display:flex;justify-content:space-between;align-items:center;margin:11px 0 7px}.group-result-title span{font-size:10px;font-weight:650}.group-result-title small{color:var(--muted);font-size:8px}.candidate-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:6px}.candidate-grid>button,.compact-list>button{min-width:0;display:grid;grid-template-columns:auto minmax(0,1fr) auto;align-items:center;gap:7px;padding:7px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted);color:var(--ink);text-align:left;cursor:pointer}.mini-avatar,.mini-avatar img{width:30px;height:30px;border-radius:7px}.mini-avatar{display:grid;place-items:center;overflow:hidden;background:var(--surface);font-size:9px}.candidate-grid strong,.compact-list strong{font-size:9px}.candidate-grid small,.compact-list small{margin-top:3px;color:var(--muted);font-size:8px}.candidate-grid svg,.compact-list svg{color:var(--muted)}.lower-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}.compact-list{display:grid;gap:5px;max-height:280px;overflow:auto}.compact-list>button{grid-template-columns:minmax(0,1fr) auto}.empty{min-height:74px;display:flex;align-items:center;justify-content:center;gap:8px;color:var(--muted);font-size:9px}.empty strong{color:var(--ink)}.clue-error{padding:11px;border-radius:7px;background:var(--surface-muted);color:var(--muted);font-size:9px}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}@media(max-width:900px){.stranger-hero{grid-template-columns:1fr}.person-grid,.lower-grid{grid-template-columns:1fr}.candidate-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:620px){.source-stats{grid-template-columns:repeat(2,1fr)}.candidate-grid{grid-template-columns:1fr}.person-actions{flex-wrap:wrap}}
</style>
