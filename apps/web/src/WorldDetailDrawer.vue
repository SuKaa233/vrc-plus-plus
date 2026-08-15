<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Copy, ExternalLink, Globe2, Heart, LoaderCircle, Play, Send, Server, Star, Users, X } from '@lucide/vue'
import type { FavoriteGroup, Friend, Instance, UpstreamWorldFavorite, World, WorldFavorite } from './api'

const props = defineProps<{
  world: World
  favorite: WorldFavorite | null
  upstreamFavorite: UpstreamWorldFavorite | null
  favoriteGroups: FavoriteGroup[]
  friends: Friend[]
  instance: Instance | null
  instanceLocation: string
  instanceLoading: boolean
  saving: boolean
  upstreamSaving: boolean
  inviteSending: boolean
  imageUrl: string
}>()
const emit = defineEmits<{
  close: []
  saveFavorite: [note: string]
  removeFavorite: []
  selectInstance: [location: string]
  addUpstreamFavorite: [group: string]
  removeUpstreamFavorite: []
  sendInvite: [receiverUserId: string]
}>()

const note = ref('')
const copied = ref('')
const selectedGroup = ref('')
const inviteFriendID = ref('')
watch(() => props.favorite, (value) => { note.value = value?.note ?? '' }, { immediate: true })
watch(() => props.favoriteGroups, (value) => { if (!selectedGroup.value) selectedGroup.value = value[0]?.name ?? '' }, { immediate: true })

const officialUrl = computed(() => `https://vrchat.com/home/world/${encodeURIComponent(props.world.id)}`)
const inviteFriends = computed(() => [...props.friends].sort((a, b) => Number(b.online) - Number(a.online) || a.displayName.localeCompare(b.displayName)))
const instanceKind = computed(() => {
  const value = props.instance?.type || props.instance?.groupAccessType || props.instanceLocation
  if (value.includes('private')) return '私人'
  if (value.includes('friends+')) return '好友+'
  if (value.includes('friends')) return '好友'
  if (value.includes('group')) return '群组'
  if (value.includes('hidden')) return '邀请+'
  return props.instanceLocation ? '公开' : '未选择'
})

function regionLabel(value?: string) { return ({ us: '美国', usw: '美国西部', use: '美国东部', eu: '欧洲', jp: '日本' } as Record<string, string>)[value ?? ''] || value || '自动' }
async function copy(value: string, label: string) { try { await navigator.clipboard.writeText(value); copied.value = label; window.setTimeout(() => { copied.value = '' }, 1600) } catch { copied.value = '复制失败' } }
function copyLocation() { if (!props.instanceLocation || !window.confirm('实例位置可能包含私人标记。只发送给你信任的人，仍要复制吗？')) return; void copy(props.instanceLocation, '实例位置已复制') }
function launch() { if (!props.instanceLocation || !window.confirm(`即将由 VRChat 启动并加入“${props.world.name}”。继续吗？`)) return; window.location.href = `vrchat://launch?id=${encodeURIComponent(props.instanceLocation)}` }
</script>

<template>
  <div class="world-detail-backdrop" @click.self="emit('close')">
    <aside class="world-detail" role="dialog" aria-modal="true" :aria-label="`${world.name} 世界详情`">
      <header><div><span class="panel-kicker">世界详情</span><h2>{{ world.name }}</h2><small>{{ world.authorName || '未知作者' }}</small></div><button @click="emit('close')"><X :size="18" /></button></header>
      <div class="world-hero" :class="{ empty: !imageUrl }"><img v-if="imageUrl" :src="imageUrl" alt="" /><Globe2 v-else :size="42" /><div class="hero-stats"><span><Users :size="13" />{{ world.occupants ?? 0 }} 在线</span><span>{{ world.visits ?? 0 }} 次访问</span></div></div>
      <div class="world-detail-body">
        <section class="world-description"><p>{{ world.description || '这个世界暂时没有公开简介。' }}</p><div><span>容量 {{ world.recommendedCapacity || world.capacity || '—' }}</span><span>{{ world.favorites ?? 0 }} 收藏</span><span>{{ world.releaseStatus || 'public' }}</span></div></section>

        <section class="instances-panel">
          <div class="section-heading"><Server :size="16" /><strong>公共实例</strong><span>{{ world.publicInstances?.length ?? 0 }} 个可见</span></div>
          <div v-if="world.publicInstances?.length" class="instance-list">
            <button v-for="item in world.publicInstances" :key="item.location" :class="{ active: instanceLocation === item.location }" @click="emit('selectInstance', item.location)">
              <span><b>#{{ item.instanceId.split('~')[0] }}</b><small>{{ item.type === 'public' ? '公开' : item.type }} · {{ regionLabel(item.region) }}</small></span><strong>{{ item.userCount }} 人</strong>
            </button>
          </div>
          <p v-else>此世界当前没有通过详情接口公开的实例；从好友位置打开时仍可查看对应实例。</p>
          <div v-if="instanceLoading" class="instance-loading"><LoaderCircle class="spin" :size="17" />正在读取实例详情</div>
          <div v-else-if="instance" class="instance-summary"><span><small>人数</small><strong>{{ instance.userCount ?? 0 }} / {{ instance.capacity || '—' }}</strong></span><span><small>区域</small><strong>{{ regionLabel(instance.region) }}</strong></span><span><small>队列</small><strong>{{ instance.queueEnabled ? `${instance.queueSize ?? 0} 人` : '未启用' }}</strong></span><span><small>权限</small><strong>{{ instanceKind }}</strong></span></div>
          <div v-if="instanceLocation" class="instance-actions"><button @click="copyLocation"><Copy :size="15" />{{ copied || '复制位置' }}</button><button class="join" @click="launch"><Play :size="15" />启动并加入</button></div>
        </section>

        <section class="invite-panel">
          <div class="section-heading"><Send :size="16" /><strong>发送邀请</strong><span>每次仅发送给 1 位好友</span></div>
          <div class="inline-form"><select v-model="inviteFriendID" :disabled="!instanceLocation"><option value="">选择好友</option><option v-for="friend in inviteFriends" :key="friend.id" :value="friend.id">{{ friend.online ? '在线 · ' : '离线 · ' }}{{ friend.displayName }}</option></select><button :disabled="!inviteFriendID || !instanceLocation || inviteSending" @click="emit('sendInvite', inviteFriendID)"><LoaderCircle v-if="inviteSending" class="spin" :size="15" /><Send v-else :size="15" />发送</button></div>
          <p v-if="!instanceLocation">请先选择一个公共实例，或从好友当前位置打开这个世界。</p>
        </section>

        <div class="favorite-grid">
          <section class="world-local"><div class="section-heading"><Heart :size="16" /><strong>本机收藏</strong><span>仅这台电脑</span></div><textarea v-model="note" maxlength="300" placeholder="记录玩法、朋友或注意事项"></textarea><div><button v-if="favorite" :disabled="saving" @click="emit('removeFavorite')"><Heart :size="15" fill="currentColor" />取消</button><button v-else :disabled="saving" @click="emit('saveFavorite', note)"><Heart :size="15" />收藏</button><button v-if="favorite" :disabled="saving" @click="emit('saveFavorite', note)">保存备注</button></div></section>
          <section class="world-upstream"><div class="section-heading"><Star :size="16" /><strong>VRChat 收藏</strong><span>同步到账户</span></div><template v-if="upstreamFavorite"><p>这个世界已在 VRChat 上游收藏中。</p><button :disabled="upstreamSaving" @click="emit('removeUpstreamFavorite')"><Star :size="15" fill="currentColor" />从 VRChat 移除</button></template><template v-else><select v-model="selectedGroup"><option v-for="group in favoriteGroups" :key="group.name" :value="group.name">{{ group.displayName || group.name }}</option></select><button :disabled="upstreamSaving || !selectedGroup" @click="emit('addUpstreamFavorite', selectedGroup)"><Star :size="15" />加入 VRChat 收藏</button><p v-if="!favoriteGroups.length">未读取到可用的世界收藏分组。</p></template></section>
        </div>

        <section v-if="world.tags?.length" class="world-tags"><span v-for="tag in world.tags.slice(0, 10)" :key="tag">{{ tag.replace(/^author_tag_/, '') }}</span></section>
      </div>
      <footer><button @click="copy(world.id, '世界 ID 已复制')"><Copy :size="15" />{{ copied || '复制世界 ID' }}</button><a :href="officialUrl" target="_blank" rel="noreferrer"><ExternalLink :size="15" />VRChat 官方页面</a></footer>
    </aside>
  </div>
</template>

<style scoped>
.world-detail-backdrop{position:fixed;z-index:50;inset:0;background:rgba(0,0,0,.62);display:flex;justify-content:flex-end}.world-detail{width:min(620px,100%);height:100%;overflow:auto;background:var(--surface);border-left:1px solid var(--line);box-shadow:-24px 0 70px rgba(0,0,0,.25)}header{position:sticky;z-index:2;top:0;padding:16px 20px;background:color-mix(in srgb,var(--surface) 94%,transparent);backdrop-filter:blur(14px);border-bottom:1px solid var(--line);display:flex;justify-content:space-between;align-items:flex-start}header h2{margin:3px 0;font-size:20px}header small{color:var(--muted)}header button{width:34px;height:34px;border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--ink)}.world-hero{position:relative;height:220px;background:var(--surface-muted);display:grid;place-items:center;color:var(--muted);overflow:hidden}.world-hero img{width:100%;height:100%;object-fit:cover}.world-hero:after{content:"";position:absolute;inset:auto 0 0;height:45%;background:linear-gradient(transparent,rgba(8,12,18,.75))}.hero-stats{position:absolute;z-index:1;left:18px;bottom:14px;display:flex;gap:7px}.hero-stats span{display:flex;align-items:center;gap:5px;padding:5px 8px;border-radius:7px;background:rgba(8,12,18,.72);color:#fff;font-size:9px}.world-detail-body{display:grid;gap:11px;padding:14px 18px 86px}.world-description,.world-local,.world-upstream,.instances-panel,.invite-panel{padding:13px;border:1px solid var(--line);border-radius:10px;background:var(--surface-muted)}.world-description p,.instances-panel>p,.invite-panel>p,.world-upstream p{margin:0;color:var(--ink-soft);font-size:10px;line-height:1.65}.world-description>div{display:flex;gap:6px;flex-wrap:wrap;margin-top:10px}.world-description span,.world-tags span{padding:4px 7px;border:1px solid var(--line);border-radius:999px;color:var(--muted);font-size:8px}.section-heading{display:flex;align-items:center;gap:7px}.section-heading strong{font-size:11px}.section-heading span{margin-left:auto;color:var(--muted);font-size:8px}.instance-list{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-top:10px;max-height:196px;overflow:auto}.instance-list button{min-width:0;padding:8px;border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--ink);display:flex;align-items:center;text-align:left}.instance-list button.active{border-color:var(--accent);box-shadow:0 0 0 1px var(--accent)}.instance-list button span{min-width:0;flex:1}.instance-list b,.instance-list small{display:block;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.instance-list b{font-size:9px}.instance-list small{margin-top:3px;color:var(--muted);font-size:7px}.instance-list button>strong{color:var(--accent);font-size:10px}.instance-loading{padding:14px;color:var(--muted);display:flex;justify-content:center;gap:7px}.instance-summary{display:grid;grid-template-columns:repeat(4,1fr);gap:6px;margin-top:10px}.instance-summary span{padding:8px;border:1px solid var(--line);border-radius:8px;background:var(--surface)}.instance-summary small,.instance-summary strong{display:block}.instance-summary small{color:var(--muted);font-size:7px}.instance-summary strong{margin-top:3px;font-size:9px}.instance-actions,.inline-form{display:flex;gap:7px;margin-top:10px}.instance-actions button,.inline-form button,.world-local button,.world-upstream button,footer button,footer a{display:inline-flex;align-items:center;justify-content:center;gap:6px;padding:8px 10px;border:1px solid var(--line);border-radius:7px;background:var(--surface);color:var(--ink);font-size:9px;text-decoration:none}.instance-actions .join,.inline-form button{border-color:var(--accent);background:var(--accent);color:white}.inline-form select{min-width:0;flex:1}.favorite-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}.world-local textarea{width:100%;min-height:66px;margin:10px 0 7px;resize:vertical}.world-local button+button{margin-left:5px}.world-upstream select{width:100%;margin:10px 0 7px}.world-upstream p{margin:10px 0}.world-tags{display:flex;flex-wrap:wrap;gap:5px}footer{position:fixed;right:0;bottom:0;width:min(620px,100%);padding:11px 18px;background:var(--surface);border-top:1px solid var(--line);display:grid;grid-template-columns:1fr 1fr;gap:8px}@media(max-width:620px){.world-detail{width:100%}.world-hero{height:180px}.favorite-grid,.instance-list{grid-template-columns:1fr}.instance-summary{grid-template-columns:1fr 1fr}footer{width:100%}}
</style>
