<template>
  <aside v-if="info?.activation_at || info?.scheduled?.length" class="rounded-2xl border border-primary-200 bg-primary-50 p-4 text-sm leading-6 text-primary-800 dark:border-primary-800 dark:bg-primary-950 dark:text-primary-200" role="note">
    <p v-if="info.activation_at">{{ t('renewal.notice', { date: formatDate(info.activation_at) }) }}</p>
    <p v-for="item in info.scheduled" :key="item.order_id" class="mt-2"><RouterLink :to="'/orders'" class="underline underline-offset-4">{{ t('renewal.waiting', { id: item.order_id, date: formatDate(item.activation_at) }) }}</RouterLink></p>
  </aside>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SubscriptionRenewalInfo } from '@/types/payment'
defineProps<{ info?: SubscriptionRenewalInfo }>()
const {t, locale} = useI18n()
function formatDate(value: string) { return new Date(value).toLocaleString(locale.value) }
</script>
