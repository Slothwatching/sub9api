<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-7">
      <section class="relative overflow-hidden rounded-3xl bg-primary-950 px-6 py-8 text-white sm:px-9 sm:py-10">
        <div aria-hidden="true" class="pointer-events-none absolute -right-20 -top-32 h-96 w-96 rounded-full border-[48px] border-primary-800/50" />
        <div class="relative flex flex-wrap items-end justify-between gap-6">
          <div class="max-w-xl">
            <p class="mb-3 text-xs font-semibold tracking-[0.2em] text-primary-200">GPT / CODEX</p>
            <h2 class="text-3xl font-semibold tracking-tight sm:text-4xl">{{ t('storefront.title') }}</h2>
            <p class="mt-3 text-sm leading-6 text-primary-100">{{ t('storefront.subtitle') }}</p>
          </div>
          <div class="rounded-2xl border border-white/15 bg-white/10 px-5 py-4">
            <p class="text-xs text-primary-200">{{ t('payment.currentBalance') }}</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">${{ (authStore.user?.balance ?? 0).toFixed(2) }}</p>
            <RouterLink to="/purchase?tab=recharge" class="mt-2 inline-block text-xs text-primary-100 underline underline-offset-4">{{ t('payment.tabTopUp') }} →</RouterLink>
          </div>
        </div>
      </section>

      <div v-if="loading" role="status" class="py-20 text-center text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
      <div v-else-if="failed" role="alert" class="card space-y-4 p-8 text-center">
        <p>{{ t('storefront.loadError') }}</p>
        <button class="btn btn-primary" @click="loadPlans">{{ t('storefront.retry') }}</button>
      </div>
      <template v-else>
        <SubscriptionRenewalNotice :info="renewalInfo" />
        <BillingPolicyNotice context="subscription" />
        <PlanGuide v-if="plans.length" :plans="plans" @select="choosePlan" />
        <nav :aria-label="t('storefront.choose')" class="flex flex-wrap gap-2">
          <button v-for="item in filters" :key="item.key" type="button" :aria-pressed="filter === item.key"
            class="rounded-xl px-5 py-2.5 text-sm font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500"
            :class="filter === item.key ? 'bg-primary-600 text-white dark:bg-primary-500' : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700'"
            @click="filter = item.key">{{ t(item.label) }}</button>
        </nav>
        <p class="rounded-2xl border border-primary-200 bg-primary-50 px-5 py-4 text-sm leading-6 text-primary-800 dark:border-primary-800 dark:bg-primary-950 dark:text-primary-200">{{ t('storefront.notice') }}</p>

        <div v-if="visibleGroups.length" class="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-4">
          <div v-for="group in visibleGroups" :key="group.id" :data-plan-group="group.id"
            class="min-w-0 rounded-2xl transition-shadow" :class="selected?.group_id === group.id ? 'ring-2 ring-primary-500 ring-offset-2 dark:ring-offset-dark-900' : ''">
            <SubscriptionPlanCard class="h-full" :plan="group.plans[0]!" :action-label="t('storefront.choose')" :active-subscriptions="subscriptionStore.activeSubscriptions" @select="chooseGroup(group.id)" />
          </div>
        </div>
        <p v-else class="card p-12 text-center text-gray-500 dark:text-gray-400">{{ t('storefront.empty') }}</p>

        <section v-if="selected" ref="summary" tabindex="-1" :aria-label="t('storefront.selected')" class="card grid gap-6 p-6 outline-none sm:p-8 lg:grid-cols-[1.4fr_1fr]">
          <div class="min-w-0">
            <p class="text-xs font-semibold uppercase tracking-widest text-primary-600 dark:text-primary-400">{{ t('storefront.selected') }}</p>
            <h3 class="mt-2 break-words text-xl font-semibold text-gray-900 dark:text-white">{{ selected.name }}</h3>
            <p class="mt-5 text-sm font-medium text-gray-600 dark:text-gray-300">{{ t('storefront.duration') }}</p>
            <div class="mt-3 flex flex-wrap gap-2">
              <button v-for="plan in selectedGroup?.plans" :key="plan.id" type="button" :aria-pressed="selected.id === plan.id"
                class="rounded-xl border px-4 py-3 text-sm transition-colors"
                :class="selected.id === plan.id ? 'border-primary-600 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-950 dark:text-primary-200' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-gray-300'"
                @click="selectedId = plan.id">{{ planValiditySuffix(plan, t) }} · {{ planPrice(plan) }}</button>
            </div>
            <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('storefront.periods') }}</p>
          </div>
          <div class="flex flex-col justify-center border-t border-gray-200 pt-6 dark:border-dark-700 lg:border-l lg:border-t-0 lg:pl-8 lg:pt-0">
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('storefront.price') }}</p>
            <p class="mt-2 break-words text-3xl font-semibold tracking-tight text-primary-700 dark:text-primary-300">{{ planPrice(selected) }}</p>
            <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('storefront.checkoutNote') }}</p>
            <RouterLink :to="{path: '/purchase', query: {tab: 'subscription', plan_id: String(selected.id)}}" class="btn btn-primary mt-5 w-full text-center">{{ t('storefront.continue') }} →</RouterLink>
          </div>
        </section>
        <RouterLink to="/purchase?tab=subscription" class="inline-block text-sm text-gray-500 underline underline-offset-4 dark:text-gray-400">← {{ t('storefront.back') }}</RouterLink>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAPI } from '@/api/payment'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import AppLayout from '@/components/layout/AppLayout.vue'
import PlanGuide from '@/components/experience/PlanGuide.vue'
import SubscriptionRenewalNotice from '@/components/payment/SubscriptionRenewalNotice.vue'
import BillingPolicyNotice from '@/components/payment/BillingPolicyNotice.vue'
import SubscriptionPlanCard from '@/components/payment/SubscriptionPlanCard.vue'
import { planValiditySuffix } from '@/components/payment/validity'
import { currencySymbol } from '@/components/payment/currency'
import type { SubscriptionPlan, SubscriptionRenewalInfo } from '@/types/payment'

const { t } = useI18n()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const renewalInfo = ref<SubscriptionRenewalInfo>()
const loading = ref(true)
const failed = ref(false)
const plans = ref<SubscriptionPlan[]>([])
const selectedId = ref<number | null>(null)
const summary = ref<HTMLElement>()
const filter = ref('all')
const filterOptions = [
  { key: 'all', label: 'storefront.all' },
  { key: 'daily', label: 'storefront.daily' },
  { key: 'monthly', label: 'storefront.monthly' },
  { key: 'combined', label: 'storefront.combined' },
  { key: 'weekly', label: 'storefront.weekly' },
]

// D92: group configured products by their actual quota group. Durations and prices
// come from the server; the catalog never invents discounted purchasable terms.
function durationDays(plan: SubscriptionPlan) {
  const unit = plan.validity_unit.replace(/s$/, '')
  return plan.validity_days * (unit === 'month' ? 30 : unit === 'week' ? 7 : 1)
}
const groups = computed(() => {
  const byGroup = new Map<number, SubscriptionPlan[]>()
  for (const plan of plans.value) {
    const list = byGroup.get(plan.group_id) ?? []
    list.push(plan)
    byGroup.set(plan.group_id, list)
  }
  return [...byGroup].map(([id, items]) => ({
    id,
    plans: items.sort((a, b) => durationDays(a) - durationDays(b) || a.sort_order - b.sort_order || a.id - b.id),
  }))
})
function matchesFilter(group: typeof groups.value[number], key: string) {
  const plan = group.plans[0]!
  if (key === 'daily') return plan.daily_limit_usd != null && plan.weekly_limit_usd == null && plan.monthly_limit_usd == null
  if (key === 'monthly') return plan.monthly_limit_usd != null && plan.daily_limit_usd == null && plan.weekly_limit_usd == null
  if (key === 'weekly') return plan.weekly_limit_usd != null && plan.daily_limit_usd == null && plan.monthly_limit_usd == null
  if (key === 'combined') return [plan.daily_limit_usd, plan.weekly_limit_usd, plan.monthly_limit_usd].filter(value => value != null).length > 1
  return true
}
// D103: retain support for every quota type, but do not offer empty daily tabs.
const filters = computed(() => filterOptions.filter(item => item.key === 'all' || groups.value.some(g => matchesFilter(g, item.key))))
const visibleGroups = computed(() => groups.value.filter(group => matchesFilter(group, filter.value)))
const selected = computed(() => plans.value.find(plan => plan.id === selectedId.value))
const selectedGroup = computed(() => groups.value.find(group => group.id === selected.value?.group_id))
function planPrice(plan: SubscriptionPlan) {
  return `${currencySymbol(plan.currency)}${plan.price.toFixed(2)}${plan.currency ? ` ${plan.currency}` : ''}`
}
function chooseGroup(groupId: number) {
  const id = groups.value.find(group => group.id === groupId)?.plans[0]?.id
  if (id != null) void choosePlan(id)
}
async function choosePlan(planId: number) {
  if (!plans.value.some(plan => plan.id === planId)) return
  selectedId.value = planId
  await nextTick()
  summary.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  summary.value?.focus({ preventScroll: true })
}
async function loadPlans() {
  loading.value = true
  failed.value = false
  try {
    const response = await paymentAPI.getCheckoutInfo()
    renewalInfo.value = response.data.subscription_renewal
    plans.value = response.data.plans.filter(plan => plan.for_sale !== false)
    selectedId.value = null
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(() => {
  void loadPlans()
  void subscriptionStore.fetchActiveSubscriptions().catch(() => {})
})
</script>
