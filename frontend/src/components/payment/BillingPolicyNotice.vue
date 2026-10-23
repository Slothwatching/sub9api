<template>
  <aside class="rounded-2xl border border-primary-200 bg-primary-50 p-4 text-sm leading-6 text-primary-900 dark:border-primary-800 dark:bg-primary-950 dark:text-primary-100" :aria-label="t('experience.paymentPolicy.title')">
    <h3 class="font-semibold">{{ t('experience.paymentPolicy.title') }}</h3>
    <p class="mt-1">{{ t(`experience.paymentPolicy.${context}`) }}</p>
    <div class="mt-2 flex flex-wrap gap-x-5 gap-y-1">
      <RouterLink :to="{ path: '/help', query: { topic: 'refund' }, hash: '#refund' }" class="underline underline-offset-4">{{ t('experience.paymentPolicy.refundLink') }} →</RouterLink>
      <RouterLink v-if="context !== 'orders'" :to="{ path: '/help', query: { topic: 'recharge' }, hash: '#recharge' }" class="underline underline-offset-4">{{ t('experience.paymentPolicy.rechargeLink') }} →</RouterLink>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

// Local informational copy only; payment and refund behavior remain upstream-controlled.
defineProps<{ context: 'recharge' | 'subscription' | 'orders' }>()
const { t } = useI18n()
</script>
