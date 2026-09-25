<template>
  <section class="card p-5 sm:p-6">
    <div class="flex flex-wrap items-start justify-between gap-3"><div><h2 class="text-lg font-semibold">{{ t('commercial.onboarding.title') }}</h2><p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('commercial.onboarding.intro') }}</p></div><button class="btn btn-secondary btn-sm" :aria-expanded="!collapsed" @click="collapsed = !collapsed">{{ t(collapsed ? 'commercial.onboarding.expand' : 'commercial.onboarding.collapse') }}</button></div>
    <div v-if="!collapsed" class="mt-5 space-y-4">
      <p v-if="loading" role="status">{{ t('common.loading') }}</p>
      <div v-else-if="failed" role="alert"><p>{{ t('commercial.onboarding.failed') }}</p><button class="btn btn-secondary btn-sm mt-2" @click="load">{{ t('common.retry') }}</button></div>
      <template v-else>
        <p v-if="!groups.length" class="text-sm">{{ t('commercial.onboarding.noGroups') }}</p>
        <p v-else-if="!hasKey" class="text-sm">{{ t('commercial.onboarding.noKeys') }}</p>
        <p v-if="typeof auth.user?.balance === 'number' && auth.user.balance <= 0 && !subscriptions.length" class="text-sm">{{ t('commercial.onboarding.noFunding') }} <RouterLink v-if="paymentEnabled" :to="purchaseLink" class="underline">{{ t('commercial.onboarding.purchase') }}</RouterLink></p>
        <ol class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"><li v-for="(step, index) in steps" :key="step.label" class="rounded-xl border border-gray-200 p-4 dark:border-dark-600"><span class="text-xs text-gray-500 dark:text-dark-300">0{{ index + 1 }} · {{ t(step.done ? 'commercial.onboarding.done' : 'commercial.onboarding.next') }}</span><RouterLink :to="step.to" class="mt-2 block font-medium underline-offset-4 hover:underline">{{ t(`commercial.onboarding.${step.label}`) }}</RouterLink></li></ol>
        <label class="flex items-center gap-2 text-sm"><input v-model="configured" type="checkbox" />{{ t('commercial.onboarding.configured') }}</label><p class="text-xs text-gray-500 dark:text-dark-300">{{ t('commercial.onboarding.hint') }}</p>
      </template>
      <RouterLink to="/connect" class="text-sm underline">{{ t('commercial.connect.title') }}</RouterLink>
      <a v-if="hasCommunities" href="#community" class="ml-4 text-sm underline">{{ t('commercial.community.title') }}</a>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { getAvailable } from '@/api/groups'
import { list } from '@/api/keys'
import { getActiveSubscriptions } from '@/api/subscriptions'
import { usageAPI } from '@/api/usage'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { Group, UserSubscription } from '@/types'
const { t } = useI18n(); const auth = useAuthStore(); const app = useAppStore()
const collapsed = ref(false); const configured = ref(false); const loading = ref(true); const failed = ref(false)
const groups = ref<Group[]>([]); const subscriptions = ref<UserSubscription[]>([]); const hasKey = ref(false); const hasRequest = ref(false)
const paymentEnabled = computed(() => !auth.isSimpleMode && isFeatureFlagEnabled(FeatureFlags.payment))
const purchaseLink = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription) ? { path: '/purchase', query: { tab: 'subscription' } } : '/purchase')
const hasCommunities = computed(() => app.cachedPublicSettings?.community_links?.some(item => item.enabled))
const serviceStep = computed(() => {
  if (isFeatureFlagEnabled(FeatureFlags.availableChannels)) return { label: 'channels', to: '/available-channels' }
  if (isFeatureFlagEnabled(FeatureFlags.modelPlaza)) return { label: 'services', to: '/model-plaza?embedded=1' }
  return { label: 'accessGuide', to: '/connect' }
})
const steps = computed(() => [{ ...serviceStep.value, done: groups.value.length > 0 }, { label: 'key', to: '/keys', done: hasKey.value }, { label: 'configure', to: '/connect', done: configured.value }, { label: 'request', to: auth.isSimpleMode ? '/keys' : '/usage', done: hasRequest.value }])
async function load() {
  loading.value = true; failed.value = false
  try {
    const [available, keyList, active, stats] = await Promise.all([getAvailable(), list(1, 100, { status: 'active' }), getActiveSubscriptions(), usageAPI.getDashboardStats()])
    groups.value = available; subscriptions.value = active
    hasKey.value = keyList.items.some(key => (!key.expires_at || Date.parse(key.expires_at) > Date.now()) && (!key.quota || key.quota_used < key.quota) && !!key.group_id)
    hasRequest.value = stats.total_requests > 0
  } catch { failed.value = true } finally { loading.value = false }
}
onMounted(load)
</script>
