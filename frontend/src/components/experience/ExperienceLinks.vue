<template>
  <nav :aria-label="t('experience.links')" class="flex flex-wrap gap-x-5 gap-y-2 text-sm font-medium text-primary-700 dark:text-primary-300">
    <RouterLink to="/help" class="underline underline-offset-4">{{ t('experience.help.title') }}</RouterLink>
    <RouterLink to="/connect" class="underline underline-offset-4">{{ t('commercial.connect.title') }}</RouterLink>
    <RouterLink v-if="models" to="/model-plaza" class="underline underline-offset-4">{{ t('nav.modelPlaza') }}</RouterLink>
    <RouterLink to="/billing" class="underline underline-offset-4">{{ t('experience.billing.title') }}</RouterLink>
    <RouterLink v-if="plans" to="/subscribe" class="underline underline-offset-4">{{ t('experience.plans.title') }}</RouterLink>
  </nav>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
const { t } = useI18n(); const app = useAppStore(); const auth = useAuthStore()
const models = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza))
const plans = computed(() => !auth.isSimpleMode && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription) && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment))
</script>
