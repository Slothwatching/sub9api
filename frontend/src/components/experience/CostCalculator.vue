<template>
  <section class="card space-y-5 p-5 sm:p-6" aria-labelledby="cost-calculator-title">
    <div><h2 id="cost-calculator-title" class="text-xl font-semibold">{{ t('experience.calculator.title') }}</h2><p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-300">{{ t('experience.calculator.intro') }}</p></div>
    <p v-if="!groups.length" class="text-sm">{{ t('experience.calculator.empty') }}</p>
    <template v-else>
      <div class="grid gap-4 sm:grid-cols-2"><label class="min-w-0 space-y-2 text-sm"><span>{{ t('experience.calculator.group') }}</span><select v-model.number="groupId" class="input"><option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option></select></label><label class="min-w-0 space-y-2 text-sm"><span>{{ t('experience.calculator.model') }}</span><select v-model="modelName" class="input"><option v-for="m in models" :key="m.name" :value="m.name">{{ m.name }}</option></select></label></div>
      <p v-if="group" class="text-sm text-gray-500 dark:text-dark-400">{{ t('experience.calculator.rateHint', { rate: group.user_rate_multiplier ?? group.rate_multiplier }) }}</p>
      <p class="rounded-xl bg-gray-50 p-4 text-sm leading-6 dark:bg-dark-900">{{ t('experience.calculator.counting') }}</p>
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <label v-for="field in fields" :key="field" class="space-y-2 text-sm"><span>{{ t(`experience.calculator.${field}`) }}</span><input v-model.number="values[field]" type="number" :min="field === 'requests' ? 1 : 0" :max="field === 'requests' ? 1000000 : 1000000000" step="1" class="input" /></label>
        <label class="space-y-2 text-sm"><span>{{ t('experience.calculator.writeDuration') }}</span><select v-model="values.cacheWrite1h" class="input"><option :value="false">5m</option><option :value="true">1h</option></select></label>
        <label v-if="efforts.length" class="space-y-2 text-sm"><span>{{ t('experience.calculator.effort') }}</span><select v-model="values.effort" class="input"><option value="">{{ t('experience.calculator.baseEffort') }}</option><option v-for="effort in efforts" :key="effort" :value="effort">{{ effort }} ×{{ model?.pricing?.reasoning_effort_multipliers?.[effort] }}</option></select></label>
        <label v-if="model?.time_pricing?.periods.length" class="space-y-2 text-sm"><span>{{ t('experience.calculator.period') }} ({{ model.time_pricing.timezone }})</span><select v-model.number="values.period" class="input"><option :value="-1">{{ t('experience.calculator.standard') }}</option><option v-for="(period, index) in model.time_pricing.periods" :key="index" :value="index">{{ period.start_time }}–{{ period.end_time }} ×{{ period.multiplier }}</option></select><span v-if="model.time_pricing.weekdays_only" class="block text-xs">{{ t('experience.calculator.weekdays') }}</span></label>
      </div>
      <label v-if="group && hasPeakRate(group)" class="flex items-start gap-2 text-sm"><input v-model="values.peak" type="checkbox" class="mt-1" /><span>{{ t('experience.calculator.peak') }}: {{ formatPeakRateWindow(group, serverTimezoneLabel(app.cachedPublicSettings?.server_utc_offset)) }}</span></label>
      <div aria-live="polite" class="rounded-2xl bg-primary-50 p-5 dark:bg-primary-950/50">
        <p v-if="!result || 'error' in result" class="text-sm">{{ t(`experience.calculator.errors.${result && 'error' in result ? result.error : 'unsupported'}`) }}</p>
        <template v-else><p class="text-sm text-primary-800 dark:text-primary-200">{{ t('experience.calculator.estimated') }}</p><p class="mt-2 break-all text-3xl font-semibold tabular-nums text-primary-800 dark:text-primary-100">${{ result.total.toFixed(8) }}</p><dl class="mt-4 grid gap-3 text-sm sm:grid-cols-2"><div v-for="field in costFields" :key="field" class="flex justify-between gap-2"><dt>{{ t(`experience.calculator.${field}Cost`) }}</dt><dd class="font-mono">${{ result[field].toFixed(8) }}</dd></div><div class="flex justify-between gap-2"><dt>{{ t('experience.calculator.multiplier') }}</dt><dd>×{{ Number(result.multiplier.toPrecision(10)) }}</dd></div><div class="flex justify-between gap-2"><dt>{{ t('experience.calculator.requests') }}</dt><dd>{{ values.requests }}</dd></div></dl><p v-if="result.tier" class="mt-4 text-sm">{{ t('experience.calculator.tier', { tier: result.tier }) }}</p><p class="mt-3 text-xs leading-5">{{ t('experience.calculator.formula') }}</p></template>
      </div>
      <p class="text-xs leading-6 text-gray-500 dark:text-dark-400">{{ t('experience.calculator.limitations') }}</p>
    </template>
  </section>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import { estimateCost, type EstimateInput } from '@/utils/costEstimate'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { useAppStore } from '@/stores/app'
const props = defineProps<{ groups: ModelPlazaGroup[] }>()
const { t } = useI18n(); const app = useAppStore()
const groupId = ref<number>(); const modelName = ref('')
const group = computed(() => props.groups.find(g => g.id === groupId.value))
const models = computed(() => group.value?.models.filter(m => m.platform === 'openai' && m.pricing?.billing_mode === 'token') ?? [])
const model = computed(() => models.value.find(m => m.name === modelName.value))
const fields = ['input', 'output', 'cacheRead', 'cacheWrite', 'requests'] as const
const costFields = ['input', 'output', 'cacheRead', 'cacheWrite'] as const
const values = reactive<EstimateInput>({ input: 10000, output: 2000, cacheRead: 0, cacheWrite: 0, cacheWrite1h: false, requests: 1, effort: '', period: -1, peak: false })
watch(() => props.groups, next => { if (!next.some(g => g.id === groupId.value)) groupId.value = next[0]?.id }, { immediate: true })
watch(models, next => { if (!next.some(m => m.name === modelName.value)) modelName.value = next[0]?.name ?? '' }, { immediate: true })
watch([groupId, modelName], () => { values.period = -1; values.effort = ''; values.peak = false })
const efforts = computed(() => Object.keys(model.value?.pricing?.reasoning_effort_multipliers ?? {}))
const result = computed(() => group.value && model.value ? estimateCost(group.value, model.value, values) : null)
</script>
