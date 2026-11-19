<template>
  <section class="border-t border-gray-100 dark:border-dark-700/60" data-image-pricing-table>
    <div class="flex flex-wrap items-center justify-between gap-3 px-5 pb-2 pt-5">
      <div>
        <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('modelPlaza.table.imageModels') }}</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.unitPerMillion') }}</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.imagePaidRate') }}: {{ rate }}x</p>
      </div>
      <div class="flex rounded-lg border border-gray-200 p-1 dark:border-dark-700" role="group" :aria-label="t('modelPlaza.table.imagePriceType')">
        <button v-for="option in options" :key="option.value" type="button" class="rounded px-3 py-1 text-xs" :class="mode === option.value ? 'bg-primary-50 font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'" :aria-pressed="mode === option.value" @click="mode = option.value">{{ t(option.label) }}</button>
      </div>
    </div>
    <div class="overflow-x-auto">
      <table class="w-full min-w-[850px] border-collapse text-sm tabular-nums">
        <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800/50 dark:text-dark-400">
          <tr><th v-for="heading in headings" :key="heading" class="px-5 py-3 font-medium">{{ t(heading) }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="model in models" :key="model.platform + ':' + model.name" class="border-t border-gray-100 dark:border-dark-700/60">
            <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">{{ model.name }}</td>
            <td v-for="(price, index) in prices(model)" :key="index" class="whitespace-nowrap px-5 py-3 font-mono text-gray-700 dark:text-dark-200">{{ format(price) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="px-5 py-4 text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.imageTokenScope') }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatScaled } from '@/utils/pricing'
import type { PlazaModel } from '@/api/modelPlaza'

const props = defineProps<{ models: PlazaModel[]; rate: number }>()
const { t } = useI18n()
const mode = ref<'paid' | 'reference'>('paid')
const options = [
  { value: 'paid' as const, label: 'modelPlaza.table.paidPrice' },
  { value: 'reference' as const, label: 'modelPlaza.table.imageReferencePrice' }
]
const headings = ['modelPlaza.table.model', 'modelPlaza.table.textInput', 'modelPlaza.table.textOutput', 'modelPlaza.table.textCacheInput', 'modelPlaza.table.imageInput', 'modelPlaza.table.imageCacheInput', 'modelPlaza.table.imageOutput']
function prices(model: PlazaModel): (number | null | undefined)[] {
  const text = mode.value === 'paid' ? model.pricing : model.official_pricing
  const image = mode.value === 'paid' ? model.image_token_tiers?.[0] : model.official_pricing?.image_token_tiers?.[0]
  return [text?.input_price, text?.output_price, text?.cache_read_price, image?.input, image?.cache_read, image?.output]
}
function format(value: number | null | undefined): string {
  return value == null ? '-' : formatScaled(value * (mode.value === 'paid' ? props.rate : 1), 1_000_000, 2)
}
</script>
