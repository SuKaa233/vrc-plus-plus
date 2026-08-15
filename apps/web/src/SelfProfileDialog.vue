<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { CircleAlert, ExternalLink, LoaderCircle, Save, UserRoundPen, X } from '@lucide/vue'
import type { SelfProfileUpdate, UserProfile } from './api'
import type { InterfaceLocale } from './locale'
import { preferredFriendAvatar } from './media'

const props = defineProps<{
  profile: UserProfile | null
  loading: boolean
  saving: boolean
  error: string
  locale: InterfaceLocale
  mediaUrl: (value?: string) => string
}>()
const emit = defineEmits<{ close: []; save: [value: SelfProfileUpdate] }>()
const form = reactive<SelfProfileUpdate>({ status: 'active', statusDescription: '', pronouns: '', bio: '', bioLinks: [], allowAvatarCopying: false })
const l = (zh: string, en: string) => props.locale === 'en' ? en : zh
const avatar = computed(() => props.mediaUrl(preferredFriendAvatar(props.profile ?? undefined)))

watch(() => props.profile, (profile) => {
  if (!profile) return
  form.status = profile.status || 'active'
  form.statusDescription = profile.statusDescription || ''
  form.pronouns = profile.pronouns || ''
  form.bio = profile.bio || ''
  form.bioLinks = [...(profile.bioLinks || [])]
  while (form.bioLinks.length < 3) form.bioLinks.push('')
  form.allowAvatarCopying = profile.allowAvatarCopying
}, { immediate: true })

function submit() {
  emit('save', { ...form, bioLinks: form.bioLinks.map((item) => item.trim()).filter(Boolean) })
}
</script>

<template>
  <div class="profile-backdrop" @click.self="emit('close')">
    <section class="profile-dialog" role="dialog" aria-modal="true" :aria-label="l('编辑我的资料','Edit my profile')">
      <header>
        <div class="profile-identity">
          <img v-if="avatar" :src="avatar" alt=""/>
          <span v-else>{{ profile?.displayName?.slice(0,1) }}</span>
          <div><small>{{ l('我的 VRChat 资料','My VRChat profile') }}</small><h2>{{ profile?.displayName || l('读取中','Loading') }}</h2><p>{{ profile?.id }}</p></div>
        </div>
        <button class="profile-close" :title="l('关闭','Close')" @click="emit('close')"><X :size="18"/></button>
      </header>

      <div v-if="loading" class="profile-loading"><LoaderCircle class="spin" :size="24"/>{{ l('正在读取最新资料','Loading latest profile') }}</div>
      <form v-else-if="profile" @submit.prevent="submit">
        <div class="profile-grid">
          <label><span>{{ l('在线状态','Status') }}</span><select v-model="form.status"><option value="join me">Join Me</option><option value="active">Online</option><option value="ask me">Ask Me</option><option value="busy">Busy</option><option value="offline">Offline</option></select></label>
          <label><span>{{ l('称谓','Pronouns') }} <em>{{ form.pronouns.length }}/32</em></span><input v-model="form.pronouns" maxlength="32" :placeholder="l('例如：TA / They','e.g. They / Them')"/></label>
        </div>
        <label><span>{{ l('状态签名','Status message') }} <em>{{ form.statusDescription.length }}/32</em></span><input v-model="form.statusDescription" maxlength="32"/></label>
        <label><span>{{ l('个人简介','Bio') }} <em>{{ form.bio.length }}/512</em></span><textarea v-model="form.bio" maxlength="512" rows="5"/></label>
        <fieldset><legend>{{ l('资料链接（最多 3 条）','Profile links (up to 3)') }}</legend><input v-for="(_,index) in form.bioLinks" :key="index" v-model="form.bioLinks[index]" type="url" placeholder="https://"/></fieldset>
        <label class="copy-setting"><input v-model="form.allowAvatarCopying" type="checkbox"/><span><strong>{{ l('允许好友复制当前公开头像','Allow friends to clone my public avatar') }}</strong><small>{{ l('是否真正可复制仍受头像自身公开状态影响。','The avatar must also be public and clonable.') }}</small></span></label>
        <div v-if="error" class="profile-error"><CircleAlert :size="17"/>{{ error }}</div>
        <footer><a href="https://vrchat.com/home/profile" target="_blank" rel="noreferrer">{{ l('在 VRChat 网页查看','Open on VRChat') }}<ExternalLink :size="13"/></a><button type="submit" :disabled="saving"><LoaderCircle v-if="saving" class="spin" :size="16"/><Save v-else :size="16"/>{{ saving ? l('保存中','Saving') : l('保存到 VRChat','Save to VRChat') }}</button></footer>
      </form>
      <div v-else class="profile-loading"><UserRoundPen :size="24"/>{{ error || l('暂时无法读取资料','Profile unavailable') }}</div>
    </section>
  </div>
</template>

<style scoped>
.profile-backdrop{position:fixed;inset:0;z-index:24;display:grid;place-items:center;padding:20px;background:var(--overlay)}.profile-dialog{width:min(680px,100%);max-height:calc(100vh - 40px);overflow:auto;border:1px solid var(--line);border-radius:12px;background:var(--surface);box-shadow:0 24px 70px rgb(0 0 0/.24)}.profile-dialog>header{position:sticky;top:0;z-index:1;display:flex;align-items:center;justify-content:space-between;padding:18px 20px;border-bottom:1px solid var(--line);background:var(--surface)}.profile-identity{min-width:0;display:flex;align-items:center;gap:12px}.profile-identity>img,.profile-identity>span{width:54px;height:54px;display:grid;place-items:center;flex:0 0 auto;border-radius:12px;object-fit:cover;background:var(--surface-muted);font-size:20px;font-weight:700}.profile-identity div{min-width:0}.profile-identity small{color:var(--muted);font-size:11px}.profile-identity h2{margin:2px 0;font-size:20px}.profile-identity p{margin:0;overflow:hidden;color:var(--muted);font:10px ui-monospace,monospace;text-overflow:ellipsis}.profile-close{width:34px;height:34px;display:grid;place-items:center;border:1px solid var(--line);border-radius:6px;background:transparent;color:var(--muted)}.profile-dialog form{display:grid;gap:14px;padding:20px}.profile-dialog label{display:grid;gap:6px;color:var(--ink-soft);font-size:12px;font-weight:600}.profile-dialog label>span:first-child{display:flex;justify-content:space-between}.profile-dialog em{color:var(--muted);font-size:10px;font-style:normal;font-weight:400}.profile-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}.profile-dialog input,.profile-dialog select,.profile-dialog textarea{width:100%;padding:10px 11px;border:1px solid var(--line-strong);border-radius:7px;outline:0;background:var(--surface-muted);color:var(--ink);font:inherit;resize:vertical}.profile-dialog input:focus,.profile-dialog select:focus,.profile-dialog textarea:focus{border-color:var(--accent)}.profile-dialog fieldset{display:grid;gap:7px;margin:0;padding:12px;border:1px solid var(--line);border-radius:8px}.profile-dialog legend{padding:0 5px;color:var(--ink-soft);font-size:12px;font-weight:600}.copy-setting{grid-template-columns:18px 1fr!important;align-items:start;padding:12px;border:1px solid var(--line);border-radius:8px;background:var(--surface-muted)}.copy-setting input{width:16px;height:16px;margin-top:2px}.copy-setting span,.copy-setting strong,.copy-setting small{display:block}.copy-setting strong{font-size:12px}.copy-setting small{margin-top:3px;color:var(--muted);font-size:10px;font-weight:400}.profile-error{display:flex;align-items:center;gap:7px;padding:10px;border:1px solid color-mix(in srgb,var(--danger) 30%,var(--line));border-radius:7px;color:var(--danger);background:color-mix(in srgb,var(--danger) 7%,var(--surface));font-size:11px}.profile-dialog footer{display:flex;align-items:center;justify-content:space-between;gap:12px;padding-top:2px}.profile-dialog footer a,.profile-dialog footer button{display:inline-flex;align-items:center;gap:5px}.profile-dialog footer a{color:var(--muted);font-size:11px}.profile-dialog footer button{padding:10px 14px;border:0;border-radius:7px;background:var(--accent);color:white;font-size:12px;font-weight:650}.profile-dialog footer button:disabled{opacity:.55}.profile-loading{min-height:220px;display:flex;align-items:center;justify-content:center;gap:9px;color:var(--muted);font-size:12px}@media(max-width:600px){.profile-grid{grid-template-columns:1fr}.profile-dialog footer{align-items:stretch;flex-direction:column}.profile-dialog footer button{justify-content:center}.profile-identity p{max-width:220px}}
</style>
