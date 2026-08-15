<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, Clock3, LoaderCircle, Search, Send, UserPlus, Users } from '@lucide/vue'
import type { ActivityEvent, Friend, FriendStatus, UserProfile } from './api'
import { optimizedVrcImageUrl, preferredFriendAvatar } from './media'

const props = defineProps<{
  friends: Friend[]
  events: ActivityEvent[]
  results: UserProfile[]
  statuses: Record<string, FriendStatus>
  loading: boolean
  actingId: string
  mediaUrl: (value?: string) => string
}>()

const emit = defineEmits<{
  search: [query: string]
  open: [user: UserProfile]
  request: [user: UserProfile]
}>()

const query = ref('')
const searched = ref(false)
const friendIDs = computed(() => new Set(props.friends.map((item) => item.id)))
const recentUsers = computed(() => {
  const seen = new Set<string>()
  const items: UserProfile[] = []
  for (const event of props.events) {
    if (!event.userId || friendIDs.value.has(event.userId) || seen.has(event.userId)) continue
    seen.add(event.userId)
    items.push({ id: event.userId, displayName: event.displayName || event.userId, isFriend: false, allowAvatarCopying: false, trustLevel: 'visitor' })
    if (items.length >= 12) break
  }
  return items
})

function submit() {
  const value = query.value.trim()
  if (value.length >= 2) {
    searched.value = true
    emit('search', value)
  }
}

function avatar(user: UserProfile) {
  return props.mediaUrl(user.profilePicOverrideThumbnail || optimizedVrcImageUrl(user.profilePicOverride) || preferredFriendAvatar(user))
}

function relationLabel(user: UserProfile) {
  const status = props.statuses[user.id]
  if (user.isFriend || status?.isFriend || friendIDs.value.has(user.id)) return '已是好友'
  if (status?.incomingRequest) return '对方已申请'
  if (status?.outgoingRequest) return '等待接受'
  return '加好友'
}
</script>

<template>
  <section class="discovery-view wide-view">
    <article class="discovery-hero">
      <div><span>发现</span><h2>找到想认识的人</h2><p>搜索用户，或查看最近遇见。</p></div>
      <form @submit.prevent="submit"><Search :size="17" /><input v-model="query" placeholder="输入显示名称或用户 ID" aria-label="搜索用户" /><button :disabled="loading || query.trim().length < 2"><LoaderCircle v-if="loading" class="spin" :size="16" /><Search v-else :size="16" />搜索</button></form>
    </article>

    <article v-if="results.length" class="discovery-section">
      <header><div><Search :size="17" /><strong>搜索结果</strong></div><span>{{ results.length }} 位</span></header>
      <div class="discovery-grid">
        <div v-for="user in results" :key="user.id" class="person-row">
          <button class="person-main" @click="emit('open', user)"><span class="person-avatar"><img v-if="avatar(user)" :src="avatar(user)" alt="" loading="lazy" /><b v-else>{{ user.displayName.slice(0, 1) }}</b></span><span><strong>{{ user.displayName }}</strong><small>{{ user.statusDescription || user.id }}</small></span></button>
          <button class="request-button" :disabled="actingId === user.id || relationLabel(user) !== '加好友'" @click="emit('request', user)"><LoaderCircle v-if="actingId === user.id" class="spin" :size="14" /><Check v-else-if="relationLabel(user) !== '加好友'" :size="14" /><UserPlus v-else :size="14" />{{ relationLabel(user) }}</button>
        </div>
      </div>
    </article>

    <article v-else-if="searched && !loading" class="discovery-section discovery-search-empty"><Search :size="18" /><strong>没有找到匹配用户</strong><span>检查显示名称，或直接输入完整的 usr_ 用户 ID。</span></article>

    <article class="discovery-section">
      <header><div><Clock3 :size="17" /><strong>最近遇到</strong></div><span>来自本机日志，不代表好友关系</span></header>
      <div v-if="recentUsers.length" class="discovery-grid">
        <div v-for="user in recentUsers" :key="user.id" class="person-row">
          <button class="person-main" @click="emit('open', user)"><span class="person-avatar"><Users :size="17" /></span><span><strong>{{ user.displayName }}</strong><small>{{ user.id }}</small></span></button>
          <button class="request-button" @click="emit('open', user)"><Send :size="14" />查看后决定</button>
        </div>
      </div>
      <div v-else class="discovery-empty"><Users :size="18" />本机日志中暂时没有可识别的非好友用户</div>
    </article>
  </section>
</template>

<style scoped>
.discovery-view{display:grid;gap:12px}.discovery-hero,.discovery-section{border:1px solid var(--line);border-radius:8px;background:var(--surface)}.discovery-hero{display:grid;grid-template-columns:minmax(260px,.8fr) minmax(360px,1.2fr);align-items:end;gap:26px;padding:22px}.discovery-hero>div>span{color:var(--muted);font-size:10px}.discovery-hero h2{margin:5px 0 7px;font-size:21px}.discovery-hero p{max-width:620px;margin:0;color:var(--muted);font-size:11px;line-height:1.65}.discovery-hero form{height:42px;display:flex;align-items:center;gap:8px;padding-left:12px;border:1px solid var(--line-strong);border-radius:7px;background:var(--surface-muted);color:var(--muted)}.discovery-hero input{min-width:0;flex:1;border:0;outline:0;background:transparent;color:var(--ink);font-size:12px}.discovery-hero form button,.request-button{height:32px;display:inline-flex;align-items:center;justify-content:center;gap:6px;border:1px solid var(--line);border-radius:6px;background:var(--surface);color:var(--ink-soft);font-size:10px;font-weight:650;cursor:pointer}.discovery-hero form button{height:100%;padding:0 15px;border-width:0 0 0 1px;border-radius:0 6px 6px 0;color:#fff;background:var(--accent)}.discovery-hero button:disabled,.request-button:disabled{opacity:.55;cursor:default}.discovery-section{padding:14px}.discovery-section>header{display:flex;align-items:center;justify-content:space-between;margin-bottom:10px}.discovery-section>header div{display:flex;align-items:center;gap:7px}.discovery-section>header svg{color:var(--muted)}.discovery-section>header strong{font-size:12px}.discovery-section>header>span{color:var(--muted);font-size:9px}.discovery-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:7px}.person-row{min-width:0;display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:center;gap:8px;padding:7px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted)}.person-main{min-width:0;display:grid;grid-template-columns:40px minmax(0,1fr);align-items:center;gap:9px;border:0;background:transparent;color:var(--ink);text-align:left;cursor:pointer}.person-avatar,.person-avatar img{width:40px;height:40px;border-radius:8px}.person-avatar{display:grid;place-items:center;overflow:hidden;color:var(--muted);background:var(--surface)}.person-avatar img{object-fit:cover}.person-main strong,.person-main small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.person-main strong{font-size:12px}.person-main small{margin-top:4px;color:var(--muted);font-size:9px}.request-button{min-width:92px;padding:0 9px}.discovery-empty{height:90px;display:flex;align-items:center;justify-content:center;gap:8px;color:var(--muted);font-size:10px}.discovery-search-empty{min-height:86px;display:flex;align-items:center;justify-content:center;gap:8px;color:var(--muted)}.discovery-search-empty strong{color:var(--ink);font-size:11px}.discovery-search-empty span{font-size:9px}@media(max-width:800px){.discovery-hero{grid-template-columns:1fr}.discovery-grid{grid-template-columns:1fr}}@media(max-width:520px){.discovery-hero{padding:16px}.person-row{grid-template-columns:1fr}.request-button{width:100%}}
</style>
