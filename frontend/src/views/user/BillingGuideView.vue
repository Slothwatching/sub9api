<template>
  <component :is="auth.isAuthenticated ? AppLayout : 'div'" :class="auth.isAuthenticated ? '' : 'min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white'">
    <PlazaNavBar v-if="!auth.isAuthenticated" login-redirect="/billing" />
    <main class="mx-auto max-w-5xl space-y-6" :class="!auth.isAuthenticated && 'px-4 py-8'">
      <header class="space-y-3"><h1 class="text-3xl font-semibold">{{ t('experience.billing.title') }}</h1><p class="text-gray-600 dark:text-dark-300">{{ t('experience.billing.subtitle') }}</p><ExperienceLinks /></header>
      <section class="grid gap-4 sm:grid-cols-3"><article v-for="topic in ['tokens', 'quota', 'payment']" :key="topic" class="card space-y-3 p-5"><h2 class="font-semibold">{{ t(`experience.billing.${topic}Title`) }}</h2><p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t(`experience.billing.${topic}Text`) }}</p></article></section>
      <p v-if="loading" role="status">{{ t('common.loading') }}</p>
      <div v-else-if="failed" role="alert" class="card space-y-3 p-5"><p>{{ t('experience.calculator.failed') }}</p><button type="button" class="btn btn-secondary" @click="load">{{ t('common.retry') }}</button></div>
      <div v-else-if="authRequired" class="card space-y-3 p-5 text-sm"><p>{{ t('experience.calculator.loginRequired') }}</p><RouterLink :to="{ path: '/login', query: { redirect: '/billing' } }" class="btn btn-primary">{{ t('modelPlaza.nav.login') }}</RouterLink></div>
      <CostCalculator v-else :groups="groups" :subscription-plans="subscriptionPlans" />
      <section class="card space-y-4 p-5 sm:p-6"><h2 class="text-xl font-semibold">{{ t('experience.billing.auditTitle') }}</h2><ol class="list-decimal space-y-3 pl-5 text-sm leading-6 text-gray-600 dark:text-dark-300"><li v-for="step in ['audit1', 'audit2', 'audit3', 'audit4']" :key="step">{{ t(`experience.billing.${step}`) }}</li></ol><RouterLink :to="auth.isSimpleMode ? '/keys' : '/usage'" class="inline-block text-sm text-primary-700 underline dark:text-primary-300">{{ t('experience.help.actions.usage') }} →</RouterLink></section>
    </main>
  </component>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import ExperienceLinks from '@/components/experience/ExperienceLinks.vue'
import CostCalculator from '@/components/experience/CostCalculator.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { paymentAPI } from '@/api/payment'
import type { SubscriptionPlan } from '@/types/payment'
const { t } = useI18n(); const auth = useAuthStore(); const app = useAppStore()
const loading = ref(true); const failed = ref(false); const data = ref<ModelPlazaResponse | null>(null)
const subscriptionPlans = ref<SubscriptionPlan[]>([])
const groups = computed(() => data.value?.groups.filter(g => g.models.some(m => m.platform === 'openai' && m.pricing?.billing_mode === 'token')) ?? [])
const authRequired = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza) && app.cachedPublicSettings?.model_plaza_require_auth === true && !auth.isAuthenticated)
let version = 0
async function load() {
  const current = ++version; data.value = null; subscriptionPlans.value = []; failed.value = false; loading.value = true
  try {
    await app.fetchPublicSettings()
    if (current !== version) return
    if (!resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza)) return
    if (authRequired.value || app.backendModeEnabled && !auth.isAdmin) return
    const result = await getModelPlaza()
    if (current === version) data.value = result
    // D103: quota projections use actual sale products in the selected price group.
    // Public price access must not make an authenticated payment API request.
    if (current === version && auth.isAuthenticated && !auth.isSimpleMode && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription) && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment)) {
      try {
        // The plain /plans response omits quota fields; checkout-info includes them.
        const offered = await paymentAPI.getCheckoutInfo()
        if (current === version) subscriptionPlans.value = offered.data.plans.filter(p => p.for_sale !== false)
      } catch { /* Model cost estimates remain available without plan projections. */ }
    }
  } catch { if (current === version) failed.value = true }
  finally { if (current === version) loading.value = false }
}
watch([() => auth.isAuthenticated, () => auth.user?.id, () => auth.isSimpleMode, () => app.cachedPublicSettings?.model_plaza_enabled, () => app.cachedPublicSettings?.model_plaza_require_auth, () => app.cachedPublicSettings?.payment_enabled, () => app.cachedPublicSettings?.subscription_enabled], load)
onMounted(load)
onBeforeUnmount(() => { version++; data.value = null; subscriptionPlans.value = [] })
</script>
