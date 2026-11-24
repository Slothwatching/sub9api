<template>
  <section v-if="agreements.length || page > 1" class="card space-y-3 p-4">
    <h2 class="font-semibold">{{ t('payment.wechatRenewal.title') }}</h2>
    <p v-if="cancelError" class="text-sm text-red-600" role="alert">{{ t('payment.wechatRenewal.cancelError') }}</p>
    <ul class="divide-y divide-gray-200 dark:divide-dark-600">
      <li v-for="agreement in agreements" :key="agreement.id" class="flex flex-wrap items-center justify-between gap-2 py-3 text-sm">
        <div>
          <div class="font-medium">{{ t('payment.wechatRenewal.agreement', { id: agreement.id, plan: agreement.plan_id }) }}</div>
          <div v-if="admin">{{ t('payment.wechatRenewal.account', { id: agreement.user_id }) }}</div>
          <div>{{ t(`payment.wechatRenewal.${agreement.state}`) }}</div>
          <div v-if="agreement.processing">{{ t('payment.wechatRenewal.processing') }}</div>
          <div v-if="agreement.next_debit_at && !agreement.processing">
            {{ t('payment.wechatRenewal.nextDebit', { time: new Date(agreement.next_debit_at).toLocaleString() }) }}
          </div>
          <div v-if="admin && agreement.failure_code" class="text-red-600">{{ agreement.failure_code }}</div>
        </div>
        <button v-if="agreement.state !== 'failed'" type="button" class="btn btn-secondary" :disabled="busy" @click="cancel(agreement.id)">
          {{ t('payment.wechatRenewal.cancel') }}
        </button>
      </li>
    </ul>
    <div v-if="page > 1 || agreements.length === 50" class="flex gap-2">
      <button type="button" class="btn btn-secondary" :disabled="busy || page === 1" @click="load(page - 1)">{{ t('payment.wechatRenewal.previous') }}</button>
      <button type="button" class="btn btn-secondary" :disabled="busy || agreements.length < 50" @click="load(page + 1)">{{ t('payment.wechatRenewal.next') }}</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import type { WeChatRenewalAgreement } from '@/types/payment'

const props = defineProps<{ admin?: boolean }>()
const { t } = useI18n()
const agreements = ref<WeChatRenewalAgreement[]>([])
const page = ref(1)
const busy = ref(false)
const cancelError = ref(false)
const endpoint = props.admin ? '/admin/payment/wechat-renewals' : '/payment/wechat-renewals'

async function load(target: number) {
  busy.value = true
  try {
    const response = await apiClient.get<WeChatRenewalAgreement[]>(endpoint, { params: { page: target } })
    agreements.value = response.data
    page.value = target
  } catch {
    // Keep the previous page if refreshing fails.
  } finally {
    busy.value = false
  }
}

async function cancel(id: number) {
  if (!window.confirm(t('payment.wechatRenewal.askCancel'))) return
  busy.value = true
  cancelError.value = false
  try {
    await apiClient.post(`${endpoint}/${id}/cancel`)
  } catch {
    cancelError.value = true
  } finally {
    await load(page.value)
  }
}

onMounted(() => { void load(1) })
</script>
