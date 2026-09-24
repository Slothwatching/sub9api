<template>
  <component :is="auth.isAuthenticated ? AppLayout : 'div'" :class="auth.isAuthenticated ? '' : 'min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white'">
    <PlazaNavBar v-if="!auth.isAuthenticated" login-redirect="/connect" />
    <div class="mx-auto max-w-5xl space-y-6" :class="!auth.isAuthenticated && 'px-4 py-8'">
      <header v-if="!auth.isAuthenticated"><h1 class="text-3xl font-semibold">{{ t('commercial.connect.title') }}</h1></header>
      <p class="text-gray-500 dark:text-dark-300">{{ t('commercial.connect.subtitle') }}</p>
      <ConnectTutorials />
      <CcSwitchGuide />
      <header id="manual-configuration" class="border-t border-gray-200 pt-8 dark:border-dark-700"><h2 class="text-xl font-semibold">{{ t('commercial.ccSwitch.otherMethods') }}</h2><p class="mt-2 text-sm text-gray-500 dark:text-dark-300">{{ t('commercial.ccSwitch.otherMethodsHint') }}</p></header>
      <div v-if="loadFailed" role="alert" class="card space-y-2 p-4"><p>{{ t('commercial.connect.failed') }}</p><button class="btn btn-secondary" @click="load">{{ t('common.retry') }}</button></div>
      <div class="card space-y-4 p-5 sm:p-6">
        <div class="grid gap-4 sm:grid-cols-2">
          <label v-if="auth.isAuthenticated" class="min-w-0 space-y-2 text-sm"><span>{{ t('commercial.connect.key') }}</span><select v-model="keyID" class="input"><option value="">{{ t('commercial.connect.generic') }}</option><option v-for="key in usableKeys" :key="key.id" :value="String(key.id)">{{ key.name }}</option></select></label>
          <label v-if="!selectedKey" class="space-y-2 text-sm"><span>{{ t('commercial.connect.platform') }}</span><select v-model="genericPlatform" class="input"><option value="anthropic">Anthropic</option><option value="openai">OpenAI</option><option value="gemini">Gemini</option></select></label>
          <label class="min-w-0 space-y-2 text-sm"><span>{{ t('commercial.connect.endpoint') }}</span><select v-model="endpoint" class="input"><option v-for="ep in endpoints" :key="ep.endpoint" :value="ep.endpoint">{{ ep.name || ep.endpoint }}</option></select></label>
        </div>
        <p v-if="!selectedKey" class="text-sm text-gray-500 dark:text-dark-300">{{ t('commercial.connect.preview') }}</p>
        <p v-if="auth.isAuthenticated && !loading && !loadFailed && !usableKeys.length" class="text-sm">{{ t('commercial.connect.noKeys') }} <RouterLink to="/keys" class="underline">{{ t('commercial.connect.keys') }}</RouterLink></p>
        <ol class="list-inside list-decimal space-y-2 text-sm text-gray-600 dark:text-dark-300"><li v-for="n in 3" :key="n">{{ t(`commercial.connect.step${n}`) }}</li></ol>
      </div>
      <div class="card min-w-0 overflow-hidden p-5 sm:p-6"><UseKeyContent :show="true" :api-key="selectedKey?.key || 'YOUR_API_KEY'" :base-url="endpoint" :platform="platform" :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch" /></div>
      <BillingExplanation />
      <section class="space-y-3"><h2 class="text-xl font-semibold">{{ t('commercial.connect.faq') }}</h2><details v-for="topic in ['auth', 'quota', 'model', 'rate']" :key="topic" class="card p-4"><summary class="cursor-pointer font-medium">{{ t(`commercial.connect.${topic}Q`) }}</summary><p class="mt-3 text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t(`commercial.connect.${topic}A`) }}</p></details></section>
      <CommunitySection />
    </div>
  </component>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import * as keysAPI from '@/api/keys'
import type { ApiKey, GroupPlatform } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import UseKeyContent from '@/components/keys/UseKeyContent.vue'
import CcSwitchGuide from '@/components/keys/CcSwitchGuide.vue'
import ConnectTutorials from '@/components/connect/ConnectTutorials.vue'
import CommunitySection from '@/components/community/CommunitySection.vue'
import BillingExplanation from '@/components/user/BillingExplanation.vue'
const { t } = useI18n()
const auth = useAuthStore(); const app = useAppStore()
const keys = ref<ApiKey[]>([]); const keyID = ref(''); const genericPlatform = ref<GroupPlatform>('openai')
const loading = ref(false); const loadFailed = ref(false)
const endpoints = computed(() => [{ name: app.siteName, endpoint: app.cachedPublicSettings?.api_base_url || window.location.origin }, ...(app.cachedPublicSettings?.custom_endpoints ?? [])])
const endpoint = ref(endpoints.value[0].endpoint)
watch(endpoints, next => { if (!next.some(ep => ep.endpoint === endpoint.value)) endpoint.value = next[0].endpoint })
const usableKeys = computed(() => keys.value.filter(key => key.status === 'active' && (!key.expires_at || Date.parse(key.expires_at) > Date.now()) && (!key.quota || key.quota_used < key.quota) && key.group))
const selectedKey = computed(() => usableKeys.value.find(key => String(key.id) === keyID.value))
const platform = computed(() => selectedKey.value?.group?.platform ?? genericPlatform.value)
let loadVersion = 0
async function load() {
  const version = ++loadVersion
  keys.value = []; keyID.value = ''; loadFailed.value = false
  if (!auth.isAuthenticated) { loading.value = false; return }
  loading.value = true
  try {
    const result: ApiKey[] = []
    let page = 1
    let total = 0
    do {
      const res = await keysAPI.list(page++, 100)
      if (version !== loadVersion || !auth.isAuthenticated) return
      result.push(...res.items)
      total = res.total
      if (!res.items.length) break
    } while (result.length < total)
    keys.value = result
  } catch {
    if (version === loadVersion) loadFailed.value = true
  } finally {
    if (version === loadVersion) loading.value = false
  }
}
onBeforeUnmount(() => { loadVersion++; keys.value = [] })
watch(() => auth.isAuthenticated, load)
onMounted(() => { void app.fetchPublicSettings(); void load() })
</script>
