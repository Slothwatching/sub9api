<template>
  <section class="card space-y-5 p-5 sm:p-6" aria-labelledby="plan-guide-title">
    <div><h2 id="plan-guide-title" class="text-xl font-semibold">{{ t('experience.plans.title') }}</h2><p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-300">{{ t('experience.plans.intro') }}</p></div>
    <div class="grid gap-4 md:grid-cols-3">
      <div v-for="kind in ['start', 'daily', 'flexible']" :key="kind" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900"><h3 class="font-medium">{{ t(`experience.plans.${kind}Title`) }}</h3><p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t(`experience.plans.${kind}Text`) }}</p></div>
    </div>
    <div v-if="currencies.length" class="space-y-4 rounded-xl border border-primary-200 p-4 dark:border-primary-800">
      <h3 class="font-medium">{{ t('experience.plans.find') }}</h3>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <label class="space-y-1 text-sm"><span>{{ t('experience.plans.pattern') }}</span><select v-model="kind" class="input"><option value="daily">{{ t('experience.plans.steady') }}</option><option value="monthly">{{ t('experience.plans.bursty') }}</option></select></label>
        <label class="space-y-1 text-sm"><span>{{ t(kind === 'daily' ? 'experience.plans.dailySpend' : 'experience.plans.monthlySpend') }}</span><input v-model.number="spend" type="number" min="0.01" step="1" class="input" /></label>
        <label v-if="kind === 'monthly'" class="space-y-1 text-sm"><span>{{ t('experience.plans.weeklySpend') }}</span><input v-model.number="weeklySpend" type="number" min="0.01" step="0.5" class="input" /></label>
        <label class="space-y-1 text-sm"><span>{{ t('experience.plans.currency') }}</span><select v-model="currency" class="input"><option v-for="item in currencies" :key="item" :value="item">{{ item || t('experience.plans.unlabelled') }}</option></select></label>
      </div>
      <div v-if="recommended" class="flex flex-wrap items-center justify-between gap-3" aria-live="polite"><p class="text-sm">{{ t('experience.plans.result', { name: recommended.name }) }} <span class="font-semibold">{{ formatPrice(recommended) }} / {{ planValiditySuffix(recommended, t) }}</span></p><button type="button" class="btn btn-primary" @click="$emit('select', recommended.id)">{{ t('storefront.choose') }}</button></div>
      <p v-else class="text-sm" aria-live="polite">{{ t('experience.plans.noMatch') }}</p>
      <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('experience.plans.method') }}</p>
    </div>
    <details><summary class="cursor-pointer font-medium">{{ t('experience.plans.compare') }}</summary>
      <div class="mt-4 max-w-full overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700" role="region" tabindex="0" :aria-label="t('experience.plans.compare')">
        <table class="w-full min-w-[720px] text-left text-sm"><caption class="sr-only">{{ t('experience.plans.compare') }}</caption><thead class="bg-gray-50 dark:bg-dark-900"><tr><th v-for="column in ['name', 'price', 'duration', 'daily', 'weekly', 'monthly', 'rate', 'action']" :key="column" scope="col" class="whitespace-nowrap p-3">{{ t(`experience.plans.columns.${column}`) }}</th></tr></thead>
          <tbody><tr v-for="plan in plans" :key="plan.id" class="border-t border-gray-100 dark:border-dark-700"><th scope="row" class="p-3 font-medium">{{ plan.name }}</th><td class="whitespace-nowrap p-3">{{ formatPrice(plan) }}</td><td class="whitespace-nowrap p-3">{{ planValiditySuffix(plan, t) }}</td><td v-for="(value, index) in [plan.daily_limit_usd, plan.weekly_limit_usd, plan.monthly_limit_usd]" :key="index" class="p-3">{{ value == null ? '—' : `$${value}` }}</td><td class="p-3">×{{ plan.rate_multiplier ?? 1 }}<p v-if="hasPeakRate(plan)" class="mt-1 text-xs">{{ formatPeakRateWindow(plan, serverTimezoneLabel(app.cachedPublicSettings?.server_utc_offset)) }}</p></td><td class="p-3"><button class="text-primary-700 underline dark:text-primary-300" type="button" @click="$emit('select', plan.id)">{{ t('storefront.choose') }}</button></td></tr></tbody>
        </table>
      </div>
      <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('experience.plans.tableNote') }}</p>
    </details>
    <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('experience.plans.renewal') }}</p>
    <div class="flex flex-wrap gap-4 text-sm text-primary-700 dark:text-primary-300"><RouterLink to="/billing" class="underline">{{ t('experience.billing.title') }} →</RouterLink><RouterLink to="/help?topic=renewal" class="underline">{{ t('experience.plans.rules') }} →</RouterLink></div>
  </section>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import { useAppStore } from '@/stores/app'
import { recommendPlan } from '@/utils/planGuide'
import { planValiditySuffix } from '@/components/payment/validity'
import { currencySymbol } from '@/components/payment/currency'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
const props = defineProps<{ plans: SubscriptionPlan[] }>()
defineEmits<{ select: [planId: number] }>()
const { t } = useI18n(); const app = useAppStore()
const kind = ref<'daily' | 'monthly'>('monthly'); const spend = ref(165); const weeklySpend = ref(49.5); const currency = ref('')
const currencies = computed(() => [...new Set(props.plans.filter(p => p.group_platform === 'openai').map(p => p.currency || ''))])
watch(currencies, next => { if (!next.includes(currency.value)) currency.value = next[0] ?? '' }, { immediate: true })
const recommended = computed(() => recommendPlan(props.plans, kind.value, spend.value, currency.value, weeklySpend.value))
const formatPrice = (p: SubscriptionPlan) => `${currencySymbol(p.currency)}${p.price.toFixed(2)} ${p.currency || ''}`
</script>
