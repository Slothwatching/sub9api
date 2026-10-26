<template>
  <div class="thesis-shell">
    <header class="thesis-topbar">
      <RouterLink :to="backTarget" class="thesis-brand" :aria-label="t('thesis.back')">
        <img :src="siteLogo || '/brand-mark.svg'" :alt="app.siteName" />
        <span>{{ app.siteName }}</span>
        <span class="back-label">← {{ t('thesis.back') }}</span>
      </RouterLink>
      <div class="thesis-service-links">
        <span class="guide-label">{{ t('thesis.original') }}</span>
        <RouterLink to="/connect">{{ t('thesis.connect') }}</RouterLink>
        <RouterLink v-if="plansEnabled" to="/subscribe" class="thesis-plans">{{ t('thesis.plans') }} ↗</RouterLink>
      </div>
    </header>
    <nav class="thesis-chapters" :aria-label="t('thesis.chapters')">
      <button v-for="section in sections" :key="section" type="button" :aria-current="active === section ? 'location' : undefined" @click="navigate(section)">{{ t(`thesis.sections.${section}`) }}</button>
    </nav>
    <iframe ref="guide" class="thesis-document" :srcdoc="guideDocument" :title="t('thesis.title')" allow="clipboard-write" sandbox="allow-scripts allow-same-origin allow-downloads allow-popups" @load="navigate(active)" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'
import guideTemplate from '@/assets/thesis/guide.html?raw'
import guideScriptUrl from '@/assets/thesis/guide.js?url'

// D99: srcdoc avoids embedding a response with DENY/frame-ancestors headers.
// Same-origin external scripts honor the existing CSP without inline exceptions.
const guideDocument = guideTemplate.replace('__THESIS_SCRIPT_URL__', guideScriptUrl)
const { t } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const guide = ref<HTMLIFrameElement | null>(null)
const sections = ['overview', 'split', 'gen', 'sim', 'board', 'traps', 'cheat'] as const
type Section = typeof sections[number]
const active = ref<Section>('overview')
const siteLogo = computed(() => sanitizeUrl(app.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const backTarget = computed(() => !auth.isAuthenticated ? '/home' : auth.isAdmin ? '/admin/dashboard' : '/dashboard')
const plansEnabled = computed(() => !auth.isSimpleMode && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment) && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription))

function navigate(section: Section) {
  active.value = section
  guide.value?.contentWindow?.postMessage({ type: 'sloth-thesis:navigate', section }, window.location.origin)
}
function receive(event: MessageEvent) {
  if (event.origin !== window.location.origin || event.source !== guide.value?.contentWindow) return
  if (event.data?.type !== 'sloth-thesis:section' || !sections.includes(event.data.section)) return
  active.value = event.data.section
}
onMounted(() => { window.addEventListener('message', receive); void app.fetchPublicSettings() })
onBeforeUnmount(() => { window.removeEventListener('message', receive) })
</script>

<style scoped>
/* D98: this document keeps the source guide's art direction within its own route. */
.thesis-shell{height:100dvh;height:100vh;display:flex;flex-direction:column;overflow:hidden;background:#07080f;color:#eef0ff}
@supports(height:100dvh){.thesis-shell{height:100dvh}}
.thesis-topbar{display:flex;align-items:center;justify-content:space-between;gap:20px;flex-wrap:wrap;padding:11px 24px;flex-shrink:0;background:#0d1020;border-bottom:1px solid rgba(140,160,255,.16)}
.thesis-brand{display:flex;align-items:center;gap:10px;min-width:0;font-size:14px;font-weight:600;color:#eef0ff}.thesis-brand img{height:26px;width:26px;border-radius:7px;object-fit:contain}.back-label{margin-left:10px;font-size:12px;font-weight:400;color:#aab0d0}.thesis-service-links{display:flex;align-items:center;flex-wrap:wrap;gap:20px;font-size:12px}.guide-label{color:#aab0d0}.thesis-service-links a{color:#d9ddff}.thesis-service-links a.thesis-plans{background:linear-gradient(110deg,#3ee6ff,#9b6bff 60%,#ff5fa8);color:#07080f;padding:6px 12px;border-radius:7px;font-weight:600}
.thesis-chapters{display:flex;align-items:center;gap:4px;flex-shrink:0;overflow-x:auto;scrollbar-width:thin;padding:7px 24px;background:#0d1020;border-bottom:1px solid rgba(140,160,255,.16)}.thesis-chapters button{flex:none;border:1px solid transparent;border-radius:7px;background:transparent;color:#aab0d0;padding:7px 15px;font-size:13px;white-space:nowrap}.thesis-chapters button:hover{color:#eef0ff;background:rgba(255,255,255,.04)}.thesis-chapters button[aria-current=location]{color:#3ee6ff;background:rgba(62,230,255,.09);border-color:rgba(62,230,255,.18)}.thesis-shell :focus-visible{outline:2px solid #3ee6ff;outline-offset:2px}.thesis-document{flex:1;min-height:0;width:100%;border:0;background:#07080f;display:block}
@media(max-width:650px){.thesis-topbar{padding:10px 14px;gap:8px}.back-label{margin-left:3px}.thesis-service-links{gap:13px;font-size:11px}.guide-label{display:none}.thesis-chapters{padding:5px 9px}.thesis-chapters button{padding:9px 12px;font-size:12px}.thesis-service-links a{padding-block:8px}}
</style>
