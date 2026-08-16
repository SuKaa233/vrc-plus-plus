<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  CalendarDays,
  BarChart3,
  Check,
  Clock3,
  Copy,
  ExternalLink,
  Globe2,
  Link2,
  LoaderCircle,
  MapPin,
  Timer,
  Monitor,
  Save,
  Send,
  ShieldCheck,
  Smartphone,
  UserRound,
  UserPlus,
  Users,
  X,
} from '@lucide/vue'
import type { Friend, FriendActivityInsights, FriendAnnotation, FriendStatus, MutualFriend, UserProfile, World } from './api'
import { optimizedVrcImageUrl, preferredFriendAvatar } from './media'
import { buildFriendReplay } from './product-insights'

const props = defineProps<{
  friend: Friend
  profile: UserProfile | null
  world: World | null
  worlds: World[]
  mutuals: MutualFriend[]
  loading: boolean
  error: string
  copyMessage: string
  mediaUrl: (value?: string) => string
  annotation: FriendAnnotation | null
  annotationSaving: boolean
  insights: FriendActivityInsights | null
  friendStatus: FriendStatus | null
  friendRequestActing: boolean
  boopActing: boolean
  boopMessage: string
}>()

const emit = defineEmits<{
  close: []
  copyId: []
  selectFriend: [friend: MutualFriend]
  openWorld: [world: World, location: string]
  saveAnnotation: [value: { note: string; group: string; color: string; tags: string[] }]
  sendFriendRequest: [profile: UserProfile]
  sendBoop: [emojiId: string]
}>()

const localNote = ref('')
const localGroup = ref('')
const localColor = ref('#4f6ef7')
const localTags = ref('')
const selectedBoop = ref('default_hand_wave')
const replayDays = ref<1 | 7 | 30>(7)
const boopOptions = [
  { id: 'default_hand_wave', label: '挥手', emoji: '👋' },
  { id: 'default_smile', label: '微笑', emoji: '😊' },
  { id: 'default_heart', label: '爱心', emoji: '❤️' },
  { id: 'default_thumbs_up', label: '点赞', emoji: '👍' },
  { id: 'default_laugh', label: '大笑', emoji: '😄' },
  { id: 'default_thinking', label: '思考', emoji: '🤔' },
]
watch(() => [props.friend.id, props.annotation] as const, () => {
  localNote.value = props.annotation?.note ?? ''
  localGroup.value = props.annotation?.group ?? ''
  localColor.value = props.annotation?.color || '#4f6ef7'
  localTags.value = props.annotation?.tags.join('，') ?? ''
}, { immediate: true })

function saveLocalAnnotation() {
  const tags = localTags.value.split(/[,，]/).map((value) => value.trim()).filter(Boolean)
  emit('saveAnnotation', { note: localNote.value.trim(), group: localGroup.value.trim(), color: localColor.value, tags })
}

let previousOverflow = ''
function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}
onMounted(() => {
  previousOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  window.addEventListener('keydown', handleKeydown)
})
onBeforeUnmount(() => {
  document.body.style.overflow = previousOverflow
  window.removeEventListener('keydown', handleKeydown)
})

const displayName = computed(() => props.profile?.displayName || props.friend.displayName)
const location = computed(() => props.profile?.location || props.friend.location || '')
const avatar = computed(() => props.mediaUrl(
  props.profile?.profilePicOverrideThumbnail
  || optimizedVrcImageUrl(props.profile?.profilePicOverride)
  || preferredFriendAvatar(props.profile)
  || preferredFriendAvatar(props.friend),
))
const banner = computed(() => props.mediaUrl(props.profile?.bannerUrl))
const safeLinks = computed(() => (props.profile?.bioLinks ?? []).filter((value) => /^https?:\/\//i.test(value)))
const knownWorlds = computed(() => new Map(props.worlds.map((item) => [item.id, item.name])))
const replay = computed(() => buildFriendReplay(props.insights?.timeline ?? [], props.friend.id, props.worlds, replayDays.value))

function statusLabel(value?: string) {
  return ({ active: '可加入', join_me: '欢迎加入', ask_me: '加入前询问', busy: '请勿打扰', offline: '离线' } as Record<string, string>)[value ?? ''] || '状态未知'
}

function platformLabel(value?: string) {
  return ({ standalonewindows: 'PC', android: 'Quest / Android', ios: 'iOS' } as Record<string, string>)[value ?? ''] || value || '未知平台'
}

function trustLabel(value?: UserProfile['trustLevel']) {
  return ({ visitor: '访客', new: '新用户', user: '用户', known: '知名用户', trusted: '受信任用户' } as Record<string, string>)[value ?? 'visitor']
}

function dateLabel(value?: string) {
  if (!value) return '暂无记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium' }).format(date)
}

function locationLabel(value: string) {
  if (!value || value === 'offline') return '当前离线'
  if (value === 'private') return '位于私人实例'
  if (value === 'traveling') return '正在切换世界'
  if (!value.startsWith('wrld_')) return '在线位置不可见'
  const instance = value.split(':')[1] || ''
  if (instance.includes('~private(')) return '私人实例'
  if (instance.includes('~friends+')) return '好友+ 实例'
  if (instance.includes('~friends(')) return '好友实例'
  if (instance.includes('~group(')) return '群组实例'
  if (instance.includes('~hidden(')) return '邀请+ 实例'
  return '公开实例'
}

function mutualImage(friend: MutualFriend) {
  return props.mediaUrl(optimizedVrcImageUrl(friend.profilePicOverride) || preferredFriendAvatar(friend))
}

const relationLabel = computed(() => {
  if (props.friendStatus?.isFriend || props.profile?.isFriend) return '已是好友'
  if (props.friendStatus?.incomingRequest) return '对方已发送请求'
  if (props.friendStatus?.outgoingRequest) return '等待对方接受'
  return '发送好友请求'
})
const canBoop = computed(() => props.friendStatus?.isFriend || props.profile?.isFriend)

function dateTimeLabel(value?: string) {
  if (!value) return '暂无同场记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

function durationLabel(minutes: number) {
  if (minutes < 60) return `${minutes} 分钟`
  return `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分`
}
</script>

<template>
  <div class="detail-backdrop" @click.self="$emit('close')">
    <aside class="friend-detail" role="dialog" aria-modal="true" :aria-label="`${displayName} 的好友详情`">
      <header class="detail-toolbar">
        <div><span>好友资料</span><small>{{ profile ? '来自 VRChat 用户资料' : '正在读取完整资料' }}</small></div>
        <button type="button" title="关闭" @click="$emit('close')"><X :size="19" /></button>
      </header>

      <div class="profile-cover" :class="{ empty: !banner }">
        <img v-if="banner" :src="banner" alt="" />
      </div>
      <section class="profile-summary">
        <div class="detail-avatar">
          <img v-if="avatar" :src="avatar" alt="" />
          <UserRound v-else :size="30" />
          <span :class="{ online: friend.online }"></span>
        </div>
        <div class="profile-copy">
          <h2>{{ displayName }}</h2>
          <p>{{ profile?.pronouns || statusLabel(profile?.status || friend.status) }}</p>
          <div class="profile-badges">
            <span><ShieldCheck :size="13" /> {{ trustLabel(profile?.trustLevel) }}</span>
            <span><Monitor v-if="(profile?.platform || friend.platform) === 'standalonewindows'" :size="13" /><Smartphone v-else :size="13" /> {{ platformLabel(profile?.platform || friend.platform || profile?.lastPlatform || friend.lastPlatform) }}</span>
            <span v-if="profile?.isFriend"><Check :size="13" /> 已是好友</span>
          </div>
        </div>
      </section>

      <div v-if="error" class="detail-error">{{ error }}</div>
      <div v-if="loading" class="detail-loading"><LoaderCircle class="spin" :size="21" /> 正在读取好友资料</div>

      <div class="detail-content">
        <section class="detail-section location-section">
          <div class="section-title"><MapPin :size="16" /><strong>当前位置</strong><span>{{ locationLabel(location) }}</span></div>
          <div v-if="world" class="detail-world">
            <img v-if="mediaUrl(world.thumbnailImageUrl || world.imageUrl)" :src="mediaUrl(world.thumbnailImageUrl || world.imageUrl)" alt="" />
            <div><strong>{{ world.name }}</strong><small>{{ world.authorName || '未知作者' }} · {{ world.occupants ?? 0 }} 人在线</small><p>{{ world.description || '暂无世界简介' }}</p></div>
            <button type="button" @click="emit('openWorld', world, location)">{{ world.recommendedCapacity || world.capacity || '—' }} 人 · 查看实例</button>
          </div>
          <p v-else class="muted-copy">{{ locationLabel(location) }}，没有可公开读取的世界信息。</p>
        </section>

        <section class="detail-section">
          <div class="section-title"><UserRound :size="16" /><strong>个人简介</strong></div>
          <p class="bio-copy">{{ profile?.bio || '这位好友还没有填写公开简介。' }}</p>
          <div v-if="safeLinks.length" class="bio-links">
            <a v-for="link in safeLinks" :key="link" :href="link" target="_blank" rel="noreferrer"><Link2 :size="13" />{{ link.replace(/^https?:\/\//, '') }}<ExternalLink :size="12" /></a>
          </div>
        </section>

        <section class="detail-section meta-grid">
          <div><CalendarDays :size="15" /><span>加入 VRChat</span><strong>{{ dateLabel(profile?.dateJoined) }}</strong></div>
          <div><CalendarDays :size="15" /><span>最近活动</span><strong>{{ dateLabel(profile?.lastActivity || profile?.lastLogin) }}</strong></div>
          <div><Globe2 :size="15" /><span>头像克隆</span><strong>{{ profile?.allowAvatarCopying ? '允许' : '不允许' }}</strong></div>
          <div><Monitor :size="15" /><span>在线状态</span><strong>{{ statusLabel(profile?.status || friend.status) }}</strong></div>
        </section>

        <section v-if="canBoop" class="detail-section boop-section">
          <div class="section-title"><Send :size="16" /><strong>戳一戳</strong><span>{{ friend.online ? '对方在线' : '当前显示离线' }}</span></div>
          <div class="boop-controls">
            <select v-model="selectedBoop" aria-label="选择戳一戳表情">
              <option v-for="option in boopOptions" :key="option.id" :value="option.id">{{ option.emoji }} {{ option.label }}</option>
            </select>
            <button type="button" :disabled="boopActing" @click="$emit('sendBoop', selectedBoop)">
              <LoaderCircle v-if="boopActing" class="spin" :size="15" /><Send v-else :size="15" />{{ boopActing ? '发送中' : '发送' }}
            </button>
          </div>
          <p v-if="boopMessage" class="boop-message">{{ boopMessage }}</p>
        </section>

        <section class="detail-section local-insights">
          <div class="section-title"><BarChart3 :size="16" /><strong>你们的本机记录</strong><span>近 30 天 · 基于本机观测</span></div>
          <div class="insight-grid">
            <div><Timer :size="15" /><span>已配对同场时长</span><strong>{{ durationLabel(insights?.togetherMinutes ?? 0) }}</strong></div>
            <div><Users :size="15" /><span>已配对共同会话</span><strong>{{ insights?.togetherSessions ?? 0 }} 次</strong></div>
            <div><CalendarDays :size="15" /><span>最近同场</span><strong>{{ dateTimeLabel(insights?.lastMetAt) }}</strong></div>
            <div><BarChart3 :size="15" /><span>观察事件</span><strong>{{ insights?.totalEvents ?? 0 }} 条 / {{ insights?.coverageDays ?? 0 }} 天</strong></div>
            <div><Clock3 :size="15" /><span>常见活跃时段</span><strong>{{ insights?.activeHours?.length ? insights.activeHours.map(item => `${item.hour}:00`).join('、') : '数据不足' }}</strong></div>
            <div><Globe2 :size="15" /><span>涉及世界</span><strong>{{ insights?.distinctWorlds ?? 0 }} 个</strong></div>
          </div>
          <div v-if="insights?.commonWorlds?.length" class="common-worlds"><span v-for="item in insights.commonWorlds" :key="item.worldId"><Globe2 :size="12" />{{ knownWorlds.get(item.worldId) || item.worldId }} · {{ item.count }} 条观测</span></div>
          <div class="evidence-strip"><span>游戏日志 {{ insights?.sourceCounts?.gameLog ?? 0 }}</span><span>Pipeline {{ insights?.sourceCounts?.pipeline ?? 0 }}</span><span>首次观测 {{ dateLabel(insights?.firstObservedAt) }}</span></div>
          <div class="replay-toolbar"><strong>好友聚焦回放</strong><div><button v-for="days in ([1,7,30] as const)" :key="days" :class="{ active: replayDays === days }" @click="replayDays = days">{{ days === 1 ? '24 小时' : `${days} 天` }}</button></div></div>
          <div v-if="replay.length" class="friend-timeline"><div v-for="event in replay.slice(0, 12)" :key="event.id"><i></i><span><strong>{{ event.title }}</strong><small>{{ event.detail }} · {{ dateTimeLabel(event.observedAt) }} · {{ event.source === 'gameLog' ? '游戏日志' : '实时观察' }}</small></span></div></div>
          <p v-else class="muted-copy">暂无足够的 Pipeline 或游戏日志记录。未观测到不代表你们没有共同活动。</p>
        </section>

        <section v-if="profile?.note" class="detail-section">
          <div class="section-title"><UserRound :size="16" /><strong>好友备注</strong></div>
          <p class="bio-copy">{{ profile.note }}</p>
        </section>

        <section class="detail-section local-organizer">
          <div class="section-title"><UserRound :size="16" /><strong>本机整理</strong><span>不会同步到 VRChat</span></div>
          <div class="organizer-grid"><label>分组<input v-model="localGroup" maxlength="32" placeholder="例如：摄影、常玩" /></label><label>标记颜色<input v-model="localColor" type="color" /></label></div>
          <label>标签<input v-model="localTags" placeholder="使用逗号分隔，最多 12 个" /></label>
          <label>本机备注<textarea v-model="localNote" maxlength="500" rows="3" placeholder="只保存在这台设备"></textarea></label>
          <button type="button" :disabled="annotationSaving" @click="saveLocalAnnotation"><LoaderCircle v-if="annotationSaving" class="spin" :size="15" /><Save v-else :size="15" />{{ annotationSaving ? '保存中' : '保存本机整理' }}</button>
        </section>

        <section class="detail-section">
          <div class="section-title"><Users :size="16" /><strong>共同好友</strong><span>{{ mutuals.length }} 位</span></div>
          <div v-if="mutuals.length" class="mutual-grid">
            <button v-for="mutual in mutuals" :key="mutual.id" type="button" @click="$emit('selectFriend', mutual)">
              <img v-if="mutualImage(mutual)" :src="mutualImage(mutual)" alt="" />
              <span v-else>{{ mutual.displayName.slice(0, 1) }}</span>
              <div><strong>{{ mutual.displayName }}</strong><small>{{ statusLabel(mutual.status) }}</small></div>
            </button>
          </div>
          <p v-else class="muted-copy">没有公开的共同好友，或对方关闭了共享关系。</p>
        </section>
      </div>

      <footer class="detail-actions">
        <button type="button" @click="$emit('copyId')"><Check v-if="copyMessage" :size="16" /><Copy v-else :size="16" />{{ copyMessage || '复制用户 ID' }}</button>
        <button v-if="profile && relationLabel !== '已是好友'" type="button" :disabled="friendRequestActing || relationLabel !== '发送好友请求'" @click="$emit('sendFriendRequest', profile)"><LoaderCircle v-if="friendRequestActing" class="spin" :size="15" /><UserPlus v-else :size="15" />{{ relationLabel }}</button>
        <a :href="`https://vrchat.com/home/user/${friend.id}`" target="_blank" rel="noreferrer">打开 VRChat 页面 <ExternalLink :size="15" /></a>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.detail-backdrop { z-index: 30; background: var(--overlay); display: flex; justify-content: flex-end; position: fixed; inset: 0; }
.friend-detail { width: min(520px, 100%); height: 100%; color: var(--ink); background: var(--bg); border-left: 1px solid var(--line); box-shadow: -18px 0 56px #0002; overflow-y: auto; }
.detail-toolbar { z-index: 2; height: 58px; padding: 0 18px; background: var(--surface); border-bottom: 1px solid var(--line); display: flex; align-items: center; justify-content: space-between; position: sticky; top: 0; }
.detail-toolbar span, .detail-toolbar small { display: block; }.detail-toolbar span { font-size: 12px; font-weight: 700; }.detail-toolbar small { color: var(--muted); margin-top: 2px; font-size: 9px; }
.detail-toolbar button { width: 34px; height: 34px; color: var(--muted); background: var(--surface); border: 1px solid var(--line); border-radius: 8px; display: grid; place-items: center; cursor: pointer; }
.profile-cover { height: 128px; background: var(--accent-soft); overflow: hidden; }.profile-cover.empty { background: var(--surface-muted); }.profile-cover img { width: 100%; height: 100%; object-fit: cover; }
.profile-summary { min-height: 104px; margin-top: -30px; padding: 0 20px 18px; display: flex; align-items: flex-end; gap: 14px; position: relative; }
.detail-avatar { width: 84px; height: 84px; color: var(--accent); background: var(--surface); border: 4px solid var(--surface); border-radius: 20px; display: grid; place-items: center; position: relative; box-shadow: var(--shadow); overflow: visible; }.detail-avatar img { width: 100%; height: 100%; border-radius: 16px; object-fit: cover; }.detail-avatar > span { width: 13px; height: 13px; background: var(--muted); border: 3px solid var(--surface); border-radius: 50%; position: absolute; right: -1px; bottom: -1px; }.detail-avatar > span.online { background: var(--success); }
.profile-copy { min-width: 0; padding-bottom: 2px; }.profile-copy h2 { margin: 0; font-size: 22px; letter-spacing: -.03em; overflow-wrap: anywhere; }.profile-copy > p { color: var(--muted); margin: 4px 0 8px; font-size: 10px; }
.profile-badges { display: flex; flex-wrap: wrap; gap: 5px; }.profile-badges span { min-height: 22px; color: var(--ink-soft); background: var(--surface); border: 1px solid var(--line); border-radius: 99px; padding: 0 7px; font-size: 8px; display: inline-flex; align-items: center; gap: 4px; }
.detail-loading, .detail-error { margin: 0 20px 12px; padding: 10px 12px; border-radius: 8px; font-size: 10px; display: flex; align-items: center; gap: 8px; }.detail-loading { color: var(--muted); background: var(--surface-muted); }.detail-error { color: var(--danger); background: var(--danger-soft); border: 1px solid color-mix(in srgb, var(--danger) 25%, var(--line)); }
.detail-content { padding: 0 20px 86px; display: grid; gap: 10px; }.detail-section { padding: 15px; background: var(--surface); border: 1px solid var(--line); border-radius: 10px; box-shadow: var(--shadow); }
.section-title { min-height: 24px; color: var(--muted); display: flex; align-items: center; gap: 7px; }.section-title strong { color: var(--ink); font-size: 11px; }.section-title > span { margin-left: auto; font-size: 9px; }
.detail-world { margin-top: 10px; display: grid; grid-template-columns: 90px 1fr auto; align-items: center; gap: 11px; }.detail-world > img { width: 90px; height: 62px; border-radius: 8px; object-fit: cover; }.detail-world div { min-width: 0; }.detail-world strong, .detail-world small { display: block; }.detail-world strong { font-size: 11px; }.detail-world small { color: var(--muted); margin-top: 4px; font-size: 8px; }.detail-world p { color: var(--muted); margin: 6px 0 0; font-size: 8px; white-space: nowrap; text-overflow: ellipsis; overflow: hidden; }.detail-world > button { padding: 7px 8px; border: 1px solid var(--line); border-radius: 7px; background: var(--surface); color: var(--accent); font-size: 8px; }
.muted-copy, .bio-copy { color: var(--muted); margin: 9px 0 0; font-size: 10px; line-height: 1.7; white-space: pre-wrap; }.bio-copy { color: var(--ink-soft); }
.bio-links { margin-top: 11px; display: grid; gap: 6px; }.bio-links a { min-width: 0; color: var(--accent); text-decoration: none; font-size: 9px; display: flex; align-items: center; gap: 6px; }.bio-links a svg:last-child { margin-left: auto; }.bio-links a { overflow-wrap: anywhere; }
.meta-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }.meta-grid > div { min-width: 0; padding: 10px; background: var(--surface-muted); border-radius: 8px; display: grid; grid-template-columns: 18px 1fr; align-items: center; }.meta-grid svg { color: var(--accent); grid-row: 1 / 3; }.meta-grid span { color: var(--muted); font-size: 8px; }.meta-grid strong { margin-top: 3px; font-size: 9px; }
.local-insights{display:grid;gap:10px}.insight-grid{display:grid;grid-template-columns:1fr 1fr;gap:7px}.insight-grid>div{min-width:0;padding:9px;background:var(--surface-muted);border-radius:7px;display:grid;grid-template-columns:18px 1fr;align-items:center}.insight-grid svg{grid-row:1/3;color:var(--muted)}.insight-grid span{color:var(--muted);font-size:8px}.insight-grid strong{margin-top:3px;font-size:9px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.common-worlds,.evidence-strip{display:flex;flex-wrap:wrap;gap:5px}.common-worlds span,.evidence-strip span{display:inline-flex;align-items:center;gap:4px;padding:4px 6px;border:1px solid var(--line);border-radius:5px;color:var(--muted);font-size:8px}.evidence-strip span{background:var(--surface-muted)}.replay-toolbar{display:flex;align-items:center;justify-content:space-between;gap:8px;padding-top:3px}.replay-toolbar>strong{font-size:9px}.replay-toolbar>div{display:flex;gap:4px}.replay-toolbar button{padding:4px 6px;border:1px solid var(--line);border-radius:6px;background:var(--surface);color:var(--muted);font-size:8px}.replay-toolbar button.active{border-color:var(--accent);color:var(--accent);background:var(--accent-soft)}.friend-timeline{display:grid}.friend-timeline>div{display:grid;grid-template-columns:8px 1fr;align-items:center;gap:7px;padding:6px 2px;border-top:1px solid var(--line)}.friend-timeline>div:first-child{border-top:0}.friend-timeline i{width:6px;height:6px;border-radius:50%;background:var(--accent)}.friend-timeline strong,.friend-timeline small{display:block}.friend-timeline strong{font-size:9px}.friend-timeline small{margin-top:2px;color:var(--muted);font-size:8px}
.boop-section{display:grid;gap:10px}.boop-controls{display:grid;grid-template-columns:1fr auto;gap:8px}.boop-controls select{min-width:0;padding:9px 10px;color:var(--ink);background:var(--surface-muted);border:1px solid var(--line);border-radius:8px;font:inherit;font-size:10px}.boop-controls button{min-width:92px;padding:9px 13px;color:#fff;background:var(--accent);border:1px solid var(--accent);border-radius:8px;font-size:10px;font-weight:650;display:flex;align-items:center;justify-content:center;gap:6px;cursor:pointer}.boop-controls button:disabled{opacity:.6;cursor:default}.boop-message{margin:0;color:var(--success);font-size:9px}
.mutual-grid { margin-top: 10px; display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }.mutual-grid button { min-width: 0; padding: 7px; color: var(--ink); background: var(--surface-muted); border: 1px solid transparent; border-radius: 8px; display: grid; grid-template-columns: 34px 1fr; align-items: center; gap: 8px; text-align: left; cursor: pointer; }.mutual-grid button:hover { border-color: var(--line-strong); }.mutual-grid img, .mutual-grid button > span { width: 34px; height: 34px; object-fit: cover; background: var(--accent-soft); border-radius: 8px; display: grid; place-items: center; }.mutual-grid div { min-width: 0; }.mutual-grid strong, .mutual-grid small { display: block; white-space: nowrap; text-overflow: ellipsis; overflow: hidden; }.mutual-grid strong { font-size: 9px; }.mutual-grid small { color: var(--muted); margin-top: 3px; font-size: 8px; }
.local-organizer{display:grid;gap:9px}.local-organizer .section-title{margin-bottom:1px}.local-organizer label{display:grid;gap:5px;color:var(--muted);font-size:8px}.local-organizer input,.local-organizer textarea{width:100%;padding:8px 9px;border:1px solid var(--line);border-radius:7px;background:var(--surface-muted);color:var(--ink);font:inherit;resize:vertical}.local-organizer input[type=color]{height:34px;padding:3px}.organizer-grid{display:grid;grid-template-columns:1fr 86px;gap:8px}.local-organizer>button{justify-self:end;display:flex;align-items:center;gap:6px;padding:8px 10px;border:0;border-radius:7px;background:var(--accent);color:#fff;font-size:9px;font-weight:650}
.detail-actions { width: min(520px, 100%); padding: 12px 20px; background: var(--surface); border-top: 1px solid var(--line); display: flex; gap: 8px; position: fixed; right: 0; bottom: 0; }.detail-actions button, .detail-actions a { min-height: 38px; flex:1; border-radius: 8px; font-size: 10px; font-weight: 600; display: flex; align-items: center; justify-content: center; gap: 7px; cursor: pointer; text-decoration: none; }.detail-actions button { color: var(--ink-soft); background: var(--surface); border: 1px solid var(--line); }.detail-actions button:disabled{opacity:.58;cursor:default}.detail-actions a { color: #fff; background: var(--accent); border: 1px solid var(--accent); }
.spin { animation: spin 1s linear infinite; } @keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 560px) { .friend-detail { border-left: 0; }.profile-cover { height: 108px; }.profile-summary { padding-inline: 14px; }.detail-content { padding-inline: 14px; }.detail-loading, .detail-error { margin-inline: 14px; }.detail-world { grid-template-columns: 72px 1fr; }.detail-world > img { width: 72px; height: 52px; }.detail-world > button { grid-column: 1 / -1; }.mutual-grid,.insight-grid { grid-template-columns: 1fr; }.detail-actions { padding-inline: 14px; }.detail-actions a{display:none} }
</style>
