<template>
  <aside class="rounded-2xl border border-primary-200 bg-primary-50 p-4 text-sm leading-6 text-primary-900 dark:border-primary-800 dark:bg-primary-950 dark:text-primary-100" :aria-label="t('experience.paymentPolicy.title')">
    <h3 class="font-semibold">{{ t('experience.paymentPolicy.title') }}</h3>
    <p class="mt-1">{{ t(`experience.paymentPolicy.${context}`) }}</p>
    <div class="mt-2 flex flex-wrap gap-x-5 gap-y-1">
      <RouterLink :to="{ path: '/help', query: { topic: 'refund' }, hash: '#refund' }" class="underline underline-offset-4">{{ t('experience.paymentPolicy.refundLink') }} →</RouterLink>
      <RouterLink v-if="context !== 'orders'" :to="{ path: '/help', query: { topic: 'recharge' }, hash: '#recharge' }" class="underline underline-offset-4">{{ t('experience.paymentPolicy.rechargeLink') }} →</RouterLink>
    </div>
    <!-- Local manual-upgrade guidance; no automatic billing or key migration. -->
    <section v-if="context === 'subscription'" class="mt-4 space-y-2 border-t border-primary-200 pt-4 dark:border-primary-800">
      <h3 class="font-semibold">{{ t('experience.paymentPolicy.upgradeTitle') }}</h3>
      <p>{{ t('experience.paymentPolicy.upgrade') }}</p>
      <p>{{ t('experience.paymentPolicy.upgradeActivation') }}</p>
      <p class="font-medium">{{ t('experience.paymentPolicy.upgradeWarning') }}</p>
      <RouterLink :to="{ path: '/help', query: { topic: 'upgrade' }, hash: '#upgrade' }" class="inline-block underline underline-offset-4">{{ t('experience.paymentPolicy.upgradeLink') }} →</RouterLink>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

// Local informational copy only; payment and refund behavior remain upstream-controlled.
defineProps<{ context: 'recharge' | 'subscription' | 'orders' }>()
const { t } = useI18n()
</script>
