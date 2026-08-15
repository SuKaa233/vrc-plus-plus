<script setup lang="ts">
import { computed, ref } from 'vue'
import { Globe2, Search, UserRound } from '@lucide/vue'
import type { Friend, FriendAnnotation, World } from './api'

const props = defineProps<{
  friends: Friend[]
  worlds: World[]
  annotations: Record<string, FriendAnnotation>
}>()
const emit = defineEmits<{
  selectFriend: [friend: Friend]
  selectWorld: [world: World]
  resolveUser: [userID: string]
  resolveWorld: [worldID: string]
}>()

const query = ref('')
const open = ref(false)
const normalized = computed(() => query.value.trim().toLocaleLowerCase())
const extractedID = computed(() => query.value.match(/(?:^|\/)(usr_[a-zA-Z0-9-]+|wrld_[a-zA-Z0-9-]+)/)?.[1] ?? '')
const friendResults = computed(() => {
  if (!normalized.value) return []
  return props.friends.filter((friend) => {
    const annotation = props.annotations[friend.id]
    return [friend.displayName, friend.id, annotation?.note, annotation?.group, ...(annotation?.tags ?? [])]
      .some((value) => value?.toLocaleLowerCase().includes(normalized.value))
  }).slice(0, 6)
})
const worldResults = computed(() => {
  if (!normalized.value) return []
  return props.worlds.filter((world) => [world.name, world.id, world.authorName]
    .some((value) => value?.toLocaleLowerCase().includes(normalized.value))).slice(0, 4)
})
const hasResults = computed(() => friendResults.value.length || worldResults.value.length || extractedID.value)

function chooseFriend(friend: Friend) {
  open.value = false
  query.value = ''
  emit('selectFriend', friend)
}
function chooseWorld(world: World) {
  open.value = false
  query.value = ''
  emit('selectWorld', world)
}
function resolveID() {
  const id = extractedID.value
  open.value = false
  query.value = ''
  if (id.startsWith('usr_')) emit('resolveUser', id)
  else if (id.startsWith('wrld_')) emit('resolveWorld', id)
}
function closeLater() {
  window.setTimeout(() => { open.value = false }, 120)
}
</script>

<template>
  <div class="global-search" @focusin="open = true" @focusout="closeLater">
    <label><Search :size="16" /><input v-model="query" placeholder="搜索好友、标签、用户或世界 ID" @keydown.esc="open = false" /></label>
    <div v-if="open && query.trim()" class="search-results">
      <button v-if="extractedID && !friendResults.some(item => item.id === extractedID) && !worldResults.some(item => item.id === extractedID)" @click="resolveID">
        <UserRound v-if="extractedID.startsWith('usr_')" :size="17" /><Globe2 v-else :size="17" />
        <span><strong>读取 {{ extractedID.startsWith('usr_') ? '用户' : '世界' }} ID</strong><small>{{ extractedID }}</small></span>
      </button>
      <button v-for="friend in friendResults" :key="friend.id" @click="chooseFriend(friend)">
        <UserRound :size="17" /><span><strong>{{ friend.displayName }}</strong><small>{{ annotations[friend.id]?.group || annotations[friend.id]?.tags.join(' · ') || (friend.online ? '在线好友' : '离线好友') }}</small></span>
      </button>
      <button v-for="world in worldResults" :key="world.id" @click="chooseWorld(world)">
        <Globe2 :size="17" /><span><strong>{{ world.name }}</strong><small>{{ world.authorName || world.id }}</small></span>
      </button>
      <p v-if="!hasResults">本地快照中没有匹配结果</p>
    </div>
  </div>
</template>

<style scoped>
.global-search{position:relative;width:min(380px,32vw)}.global-search label{height:38px;padding:0 11px;border:1px solid var(--line);border-radius:9px;background:var(--surface);display:flex;align-items:center;gap:8px;color:var(--muted)}.global-search input{min-width:0;width:100%;border:0;outline:0;background:transparent;color:var(--ink);font-size:12px}.search-results{position:absolute;z-index:20;top:44px;left:0;width:100%;padding:6px;background:var(--surface);border:1px solid var(--line);border-radius:10px;box-shadow:var(--shadow)}.search-results button{width:100%;padding:9px;border:0;border-radius:7px;background:transparent;color:var(--ink);display:flex;align-items:center;gap:9px;text-align:left}.search-results button:hover{background:var(--surface-hover)}.search-results button>svg{flex:none;color:var(--accent)}.search-results span,.search-results strong,.search-results small{min-width:0;display:block}.search-results span{flex:1}.search-results strong,.search-results small{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.search-results strong{font-size:11px}.search-results small{margin-top:2px;color:var(--muted);font-size:9px}.search-results p{margin:8px;color:var(--muted);font-size:10px}@media(max-width:760px){.global-search{width:100%}.search-results{position:fixed;top:62px;left:14px;width:calc(100% - 28px)}}
</style>
