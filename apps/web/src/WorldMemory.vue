<script setup lang="ts">
import { computed } from 'vue'
import { CalendarDays, Clock3, Compass, History, MapPin, Users } from '@lucide/vue'
import type { ActivityEvent, Friend, World } from './api'
import { buildWorldMemories } from './product-insights'

const props = defineProps<{ events: ActivityEvent[]; worlds: World[]; friends: Friend[]; favoriteIds: string[] }>()
const emit = defineEmits<{ openWorld: [world: World]; resolveWorld: [worldId: string] }>()
const worldById = computed(() => new Map(props.worlds.map((item) => [item.id,item])))
const memories = computed(() => buildWorldMemories(props.events, props.worlds).slice(0, 10))
const friendNames = computed(() => new Map(props.friends.map((friend) => [friend.id, friend.displayName])))
const recommendations = computed(() => {
  const visited=new Set(memories.value.map((item)=>item.worldId)); const favorite=new Set(props.favoriteIds)
  return props.worlds.filter((item)=>!visited.has(item.id)).sort((a,b)=>Number(favorite.has(b.id))-Number(favorite.has(a.id))||(b.occupants??0)-(a.occupants??0)).slice(0,5)
})
function date(value:string){return new Intl.DateTimeFormat('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'}).format(new Date(value))}
function companionLabel(ids:string[]){const names=ids.map((id)=>friendNames.value.get(id)).filter(Boolean).slice(0,3);return names.length?names.join('、'):'没有可识别同行好友'}
</script>
<template>
  <section class="world-memory panel wide-view"><header><div><span class="panel-kicker">本机观察</span><h2>世界回忆与推荐</h2></div><span>事件数不等于完整到访次数</span></header><div class="memory-grid"><section><h3><History :size="15" />最近留下记录的世界</h3><button v-for="item in memories" :key="item.worldId" @click="worldById.get(item.worldId) ? emit('openWorld',worldById.get(item.worldId)!) : emit('resolveWorld',item.worldId)"><MapPin :size="15" /><div><strong>{{ item.name }}</strong><small><Clock3 :size="11" />最近 {{ date(item.lastSeenAt) }} · {{ item.eventCount }} 条观测</small><small><CalendarDays :size="11" />{{ item.visitDays }} 个自然日 · 首次 {{ date(item.firstSeenAt) }}</small><small><Users :size="11" />{{ companionLabel(item.companionIds) }}</small></div></button><p v-if="!memories.length">游戏日志还没有记录到世界访问。</p></section><section><h3><Compass :size="15" />本机推荐</h3><button v-for="world in recommendations" :key="world.id" @click="emit('openWorld',world)"><Compass :size="15" /><div><strong>{{ world.name }}</strong><small>{{ favoriteIds.includes(world.id) ? '来自你的收藏' : `${world.occupants ?? 0} 人在线` }} · {{ world.authorName || '未知作者' }}</small></div></button><p v-if="!recommendations.length">继续浏览世界后会形成更准确的本机推荐。</p></section></div></section>
</template>
<style scoped>
.world-memory{padding:0;overflow:hidden}.world-memory>header{display:flex;align-items:flex-start;justify-content:space-between;padding:15px 18px;border-bottom:1px solid var(--line)}header h2{margin:3px 0 0;font-size:15px}header>span{color:var(--muted);font-size:8px}.memory-grid{display:grid;grid-template-columns:1fr 1fr;gap:0}.memory-grid>section{padding:12px 15px}.memory-grid>section+section{border-left:1px solid var(--line)}h3{display:flex;align-items:center;gap:6px;margin:0 0 7px;font-size:10px}.memory-grid button{width:100%;display:grid;grid-template-columns:20px 1fr;align-items:flex-start;gap:5px;padding:8px 5px;border:0;border-top:1px solid var(--line);background:transparent;color:var(--ink);text-align:left}.memory-grid button:not(:disabled):hover{background:var(--surface-hover)}.memory-grid strong,.memory-grid small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.memory-grid strong{font-size:9px}.memory-grid small{display:flex;align-items:center;gap:3px;margin-top:3px;color:var(--muted);font-size:8px}.memory-grid p{color:var(--muted);font-size:9px}@media(max-width:700px){.memory-grid{grid-template-columns:1fr}.memory-grid>section+section{border-left:0;border-top:1px solid var(--line)}}
</style>
