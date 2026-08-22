<script setup lang="ts">
import { computed } from 'vue'
import { Activity, Clock3, ExternalLink, MapPin, RotateCcw, ShieldCheck, Users, X } from '@lucide/vue'
import type { GuardianSession, GuardianStatus } from './api'

const props = defineProps<{ status:GuardianStatus|null; acting:boolean; message?:string }>()
const emit = defineEmits<{ resume:[]; dismiss:[]; refresh:[] }>()
const session = computed<GuardianSession|undefined>(() => props.status?.current || props.status?.last)
const stateLabel = computed(() => ({ protecting:'守护中', recovery:'等待救援', ready:'可以续玩', idle:'等待游戏' })[props.status?.state || 'idle'])
function dateTime(value?:string){return value?new Date(value).toLocaleString('zh-CN',{month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit'}):'—'}
function kindLabel(value?:string){return ({public:'公开',friends:'好友',friends_plus:'好友+',invite:'邀请',invite_plus:'邀请+',group:'群组'} as Record<string,string>)[value||'']||'实例'}
</script>

<template>
  <section class="guardian-shell">
    <article class="guardian-hero panel" :data-state="status?.state || 'idle'">
      <div class="guardian-orb"><ShieldCheck :size="34"/></div>
      <div class="guardian-title"><span>VRChat 守护与续玩</span><h2>{{ stateLabel }}</h2><p>{{ status?.message || '正在读取本机会话状态' }}</p></div>
      <div class="guardian-live"><i :data-live="status?.gameRunning"></i>{{ status?.gameRunning?'VRChat 正在运行':'VRChat 未运行' }}</div>
    </article>
    <article v-if="session" class="guardian-session panel">
      <header><div><span>{{ status?.current?'当前现场':'最近现场' }}</span><h3>{{ session.worldName || session.worldId }}</h3></div><b>{{ kindLabel(session.locationKind) }}</b></header>
      <div class="guardian-facts">
        <span><MapPin :size="17"/><small>实例</small><strong>{{ session.instanceId }}</strong></span>
        <span><Users :size="17"/><small>日志观测人数</small><strong>{{ session.participantCount }}</strong></span>
        <span><Clock3 :size="17"/><small>{{ status?.current?'进入时间':'退出时间' }}</small><strong>{{ dateTime(status?.current?session.joinedAt:status?.exitAt) }}</strong></span>
        <span><Activity :size="17"/><small>区域</small><strong>{{ session.region?.toUpperCase() || '未知' }}</strong></span>
      </div>
      <div class="guardian-details">
        <label><small>完整实例地址</small><code>{{ session.location }}</code></label>
        <label v-if="session.accessOwnerId"><small>实例房主 / 访问主体</small><code>{{ session.accessOwnerId }}</code></label>
        <label v-if="session.groupId"><small>群组</small><code>{{ session.groupId }}</code></label>
        <label><small>最后观测</small><strong>{{ dateTime(session.lastObservedAt) }}</strong></label>
      </div>
      <div v-if="session.participants?.length" class="guardian-people"><header><strong>{{ status?.current?'当前日志仍在场的玩家':'退出前仍在房间的玩家' }}</strong><span>{{ session.participants.length }} 人</span></header><div><span v-for="person in session.participants" :key="person.userId||person.displayName"><b>{{ person.displayName }}</b><small v-if="person.userId">{{ person.userId }}</small></span></div></div>
      <div v-if="status?.state==='recovery'" class="guardian-warning"><strong>检测到意外退出</strong><span>这里展示完整本机记录；状态文件仍使用 Windows DPAPI 加密保存，不上传服务器。</span></div>
      <footer>
        <button class="quiet" @click="emit('refresh')"><RotateCcw :size="16"/>刷新状态</button>
        <button v-if="status?.state==='recovery'&&!status.dismissed" class="quiet" @click="emit('dismiss')"><X :size="16"/>暂不处理</button>
        <button v-if="status?.canResume&&!status?.current" class="primary" :disabled="acting" @click="emit('resume')"><ExternalLink :size="16"/>{{ acting?'正在打开…':'返回最后实例' }}</button>
      </footer>
      <p v-if="message" class="guardian-message">{{ message }}</p>
    </article>
    <article v-else class="guardian-empty panel"><ShieldCheck :size="42"/><h3>守护服务已经待命</h3><p>进入任意 VRChat 世界后，这里会保存最近实例。正常退出会保留“继续上次”，意外退出会弹出救援通知。</p></article>
    <div class="guardian-grid">
      <article class="panel"><span>已交付</span><h3>异常退出识别</h3><p>结合 VRChat 进程状态和游戏正常退出日志，区分正常关闭与疑似崩溃。</p></article>
      <article class="panel"><span>已交付</span><h3>加密现场快照</h3><p>世界、实例、区域和人数在本机持续更新；敏感 nonce 不进入普通历史。</p></article>
      <article class="panel"><span>下一阶段</span><h3>满房蹲位</h3><p>需要在真实账号上验证实例人数、队列和限流行为后开放，不会用高频请求抢位。</p></article>
      <article class="panel"><span>下一阶段</span><h3>好友迁移追踪</h3><p>只对用户刚才同房且当前仍公开可见的好友进行短时核对。</p></article>
    </div>
  </section>
</template>
