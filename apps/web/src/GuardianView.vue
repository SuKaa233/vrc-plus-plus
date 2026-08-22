<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Activity, BellRing, Clock3, ExternalLink, MapPin, RotateCcw, Route, ShieldCheck, Square, Users, X } from '@lucide/vue'
import type { GuardianSession, GuardianStatus } from './api'

const props = defineProps<{ status:GuardianStatus|null; acting:boolean; message?:string }>()
const emit = defineEmits<{ resume:[]; dismiss:[]; refresh:[]; startSlotWatch:[location:string]; stopSlotWatch:[]; startMigrationWatch:[]; stopMigrationWatch:[]; openLocation:[location:string] }>()
const session = computed<GuardianSession|undefined>(() => props.status?.current || props.status?.last)
const locationInput = ref('')
watch(session, value => { if (!locationInput.value && value?.location) locationInput.value = value.location }, { immediate:true })
const stateLabel = computed(() => ({ protecting:'守护中', recovery:'等待救援', ready:'可以续玩', idle:'等待游戏' })[props.status?.state || 'idle'])
function dateTime(value?:string){return value?new Date(value).toLocaleString('zh-CN',{month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit'}):'—'}
function kindLabel(value?:string){return ({public:'公开',friends:'好友',friends_plus:'好友+',invite:'邀请',invite_plus:'邀请+',group:'群组'} as Record<string,string>)[value||'']||'实例'}
function peopleText(value:Array<{displayName:string}>){ return value.map(person=>person.displayName).join('、') }
</script>

<template>
  <section class="guardian-shell">
    <article class="guardian-hero panel" :data-state="status?.state || 'idle'">
      <div class="guardian-orb"><ShieldCheck :size="34"/></div>
      <div class="guardian-title"><span>VRChat 守护与续玩</span><h2>{{ stateLabel }}</h2><p>{{ status?.message || '正在读取会话状态' }}</p></div>
      <div class="guardian-live"><i :data-live="status?.gameRunning"></i>{{ status?.gameRunning?'VRChat 正在运行':'VRChat 未运行' }}</div>
    </article>
    <article v-if="session" class="guardian-session panel">
      <header><div><span>{{ status?.current?'当前现场':'最近现场' }}</span><h3>{{ session.worldName || session.worldId }}</h3></div><b>{{ kindLabel(session.locationKind) }}</b></header>
      <div class="guardian-facts">
        <span><MapPin :size="17"/><small>实例</small><strong>{{ session.instanceId }}</strong></span>
        <span><Users :size="17"/><small>同房记录</small><strong>{{ session.participantCount }} 人</strong></span>
        <span><Clock3 :size="17"/><small>{{ status?.current?'进入时间':'退出时间' }}</small><strong>{{ dateTime(status?.current?session.joinedAt:status?.exitAt) }}</strong></span>
        <span><Activity :size="17"/><small>区域</small><strong>{{ session.region?.toUpperCase() || '未知' }}</strong></span>
      </div>
      <div class="guardian-details"><label><small>实例地址</small><code>{{ session.location }}</code></label><label v-if="session.accessOwnerId"><small>房主 / 访问主体</small><code>{{ session.accessOwnerId }}</code></label><label v-if="session.groupId"><small>群组</small><code>{{ session.groupId }}</code></label><label><small>最后更新</small><strong>{{ dateTime(session.lastObservedAt) }}</strong></label></div>
      <div v-if="session.participants?.length" class="guardian-people"><header><strong>{{ status?.current?'当前仍在场':'离开前仍在场' }}</strong><span>{{ session.participants.length }} 人</span></header><div><span v-for="person in session.participants" :key="person.userId||person.displayName"><b>{{ person.displayName }}</b><small v-if="person.userId">{{ person.userId }}</small></span></div></div>
      <div v-if="status?.state==='recovery'" class="guardian-warning"><strong>检测到意外退出</strong><span>最近现场已保存在这台电脑上，可以尝试直接返回。</span></div>
      <footer><button class="quiet" @click="emit('refresh')"><RotateCcw :size="16"/>刷新状态</button><button v-if="status?.state==='recovery'&&!status.dismissed" class="quiet" @click="emit('dismiss')"><X :size="16"/>暂不处理</button><button v-if="status?.canResume&&!status?.current" class="primary" :disabled="acting" @click="emit('resume')"><ExternalLink :size="16"/>{{ acting?'正在打开…':'返回最后实例' }}</button></footer>
    </article>
    <article v-else class="guardian-empty panel"><ShieldCheck :size="42"/><h3>守护服务已经待命</h3><p>进入任意 VRChat 世界后，这里会记住最近现场；意外退出时可以快速返回。</p></article>

    <div class="guardian-tools">
      <article class="guardian-tool panel">
        <header><div class="guardian-tool-icon"><BellRing :size="21"/></div><div><h3>满房空位提醒</h3><p>房间满员时每分钟检查一次，发现空位立即托盘提醒；不会自动抢位。</p></div></header>
        <template v-if="status?.slotWatch">
          <div class="guardian-tool-state" :data-state="status.slotWatch.state"><strong>{{ status.slotWatch.state==='available'?'有空位了':status.slotWatch.state==='expired'?'提醒已结束':'等待空位' }}</strong><span v-if="status.slotWatch.capacity">{{ status.slotWatch.userCount }} / {{ status.slotWatch.capacity }} 人<span v-if="status.slotWatch.queueSize"> · 队列 {{ status.slotWatch.queueSize }} 人</span></span><span>{{ status.slotWatch.message }}</span></div>
          <div class="guardian-tool-meta"><span>上次检查 {{ dateTime(status.slotWatch.lastCheckedAt) }}</span><span>结束时间 {{ dateTime(status.slotWatch.expiresAt) }}</span></div>
          <div class="guardian-tool-actions"><button v-if="status.slotWatch.state==='available'" class="primary" :disabled="acting" @click="emit('openLocation',status.slotWatch.location)"><ExternalLink :size="15"/>进入该实例</button><button class="quiet guardian-stop" @click="emit('stopSlotWatch')"><Square :size="15"/>停止提醒</button></div>
        </template>
        <template v-else><label class="guardian-location"><span>实例地址</span><input v-model.trim="locationInput" placeholder="wrld_xxx:12345~region(jp)"/></label><button class="primary guardian-action" :disabled="acting||!locationInput" @click="emit('startSlotWatch',locationInput)"><BellRing :size="16"/>开启两小时提醒</button></template>
      </article>

      <article class="guardian-tool panel">
        <header><div class="guardian-tool-icon"><Route :size="21"/></div><div><h3>好友去向追踪</h3><p>离开房间后，短时核对刚才同房好友当前可见的位置，帮助找到大家的新房间。</p></div></header>
        <template v-if="status?.migration">
          <div class="guardian-tool-state" :data-state="status.migration.state"><strong>{{ status.migration.state==='expired'?'追踪已结束':'正在追踪' }}</strong><span>{{ status.migration.tracked.length }} 位好友在核对范围内</span><span>{{ status.migration.message }}</span></div>
          <div v-if="status.migration.destinations?.length" class="guardian-destinations"><button v-for="destination in status.migration.destinations" :key="destination.location" @click="emit('openLocation',destination.location)"><b>{{ destination.people.length >= 2 ? `${destination.people.length} 位好友出现在同一新实例` : `${destination.people[0]?.displayName} 的可见去向` }}</b><span>{{ peopleText(destination.people) }}</span><code>{{ destination.worldId }}<template v-if="destination.region"> · {{ destination.region.toUpperCase() }}</template></code><ExternalLink :size="15"/></button></div>
          <div class="guardian-tool-meta"><span>上次检查 {{ dateTime(status.migration.lastCheckedAt) }}</span><span>结束时间 {{ dateTime(status.migration.expiresAt) }}</span></div>
          <button class="quiet guardian-stop" @click="emit('stopMigrationWatch')"><Square :size="15"/>停止追踪</button>
        </template>
        <template v-else><div class="guardian-tool-ready"><Users :size="18"/><span v-if="session?.participants?.length">将核对最近同房记录中的 {{ session.participants.length }} 人</span><span v-else>进入过房间并记录到玩家后即可使用</span></div><button class="primary guardian-action" :disabled="acting||!session?.participants?.some(person=>person.userId)" @click="emit('startMigrationWatch')"><Route :size="16"/>追踪 30 分钟</button></template>
      </article>
    </div>
    <p v-if="message" class="guardian-message">{{ message }}</p>
  </section>
</template>
