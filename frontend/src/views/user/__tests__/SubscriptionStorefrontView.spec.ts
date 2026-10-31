import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { defineComponent } from 'vue'
import SubscriptionStorefrontView from '../SubscriptionStorefrontView.vue'
import type { SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key, locale: {value: 'zh'} }),
}))
const getCheckoutInfo = vi.hoisted(() => vi.fn())
vi.mock('@/api/payment', () => ({ paymentAPI: { getCheckoutInfo } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 16 } }) }))
vi.mock('@/stores/subscriptions', () => ({ useSubscriptionStore: () => ({ activeSubscriptions: [], fetchActiveSubscriptions: () => Promise.resolve() }) }))
const card = defineComponent({ props: ['plan'], emits: ['select'], template: '<button @click="$emit(\'select\', plan)">{{ plan.name }}</button>' })
function plan(id: number, overrides: Partial<SubscriptionPlan> = {}): SubscriptionPlan {
  return { id, group_id: id, name: `Plan ${id}`, description: '', price: 63, currency: 'CNY', validity_days: 30, validity_unit: 'days', daily_limit_usd: 5, weekly_limit_usd: null, monthly_limit_usd: null, features: [], for_sale: true, sort_order: id, ...overrides }
}
function render() {
  return mount(SubscriptionStorefrontView, { global: {
    stubs: { AppLayout: { template: '<div><slot /></div>' }, SubscriptionPlanCard: card, PlanGuide: true, RouterLink: RouterLinkStub },
  } })
}
beforeEach(() => {
  vi.clearAllMocks()
  Element.prototype.scrollIntoView = vi.fn()
})
describe('subscription storefront', () => {
  it('shows six weekly/monthly plans and one monthly-only premium with no empty daily filter', async () => {
    const monthly = [165, 330, 660, 1155, 1650, 2805, 4400]
    getCheckoutInfo.mockResolvedValue({ data: { plans: monthly.map((cap, i) => plan(i + 1, { daily_limit_usd: null, weekly_limit_usd: i === 6 ? null : cap * 0.3, monthly_limit_usd: cap })) } })
    const wrapper = render(); await flushPromises()
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(7)
    expect(wrapper.findAll('nav button').map(b => b.text())).toEqual(['storefront.all', 'storefront.monthly', 'storefront.combined'])
    await wrapper.findAll('nav button')[2]!.trigger('click')
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(6)
    await wrapper.findAll('nav button')[1]!.trigger('click')
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(1)
    expect(wrapper.find('[data-plan-group]').attributes('data-plan-group')).toBe('7')
    wrapper.unmount()
  })
  it('keeps all seven new and seven existing groups; filters real quotas and excludes unavailable products', async () => {
    const existing = Array.from({ length: 7 }, (_, i) => plan(i + 1, { weekly_limit_usd: 75, monthly_limit_usd: 150 }))
    const added = Array.from({ length: 7 }, (_, i) => plan(i + 101, i === 6 ? { daily_limit_usd: null, monthly_limit_usd: 4000 } : {}))
    getCheckoutInfo.mockResolvedValue({ data: { plans: [...existing, ...added, plan(999, { for_sale: false })] } })
    const wrapper = render(); await flushPromises()
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(14)
    await wrapper.findAll('nav button')[1]!.trigger('click')
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(6)
    await wrapper.findAll('nav button')[2]!.trigger('click')
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(1)
    await wrapper.findAll('nav button')[3]!.trigger('click')
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(7)
  })
  it('groups real durations, forwards the exact selected product, and does not invent terms or place orders', async () => {
    getCheckoutInfo.mockResolvedValue({ data: { plans: [plan(2, { group_id: 1, validity_days: 3, validity_unit: 'months', price: 170.1 }), plan(1)] } })
    const wrapper = render(); await flushPromises()
    expect(wrapper.findAll('[data-plan-group]')).toHaveLength(1)
    await wrapper.find('[data-plan-group] button').trigger('click'); await flushPromises()
    const summary = wrapper.find('section[tabindex]')
    expect(summary.findAll('button')).toHaveLength(2)
    expect(summary.text()).toContain('¥63.00 CNY')
    await summary.findAll('button')[1]!.trigger('click')
    expect(summary.findComponent(RouterLinkStub).props('to')).toEqual({ path: '/purchase', query: { tab: 'subscription', plan_id: '2' } })
    expect(summary.text()).toContain('¥170.10 CNY')
  })
  it('offers a retry after loading fails instead of presenting stale purchasable data', async () => {
    getCheckoutInfo.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ data: { plans: [] } })
    const wrapper = render(); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    await wrapper.find('[role="alert"] button').trigger('click'); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('storefront.empty')
  })
})

// Older checkout responses omit for_sale; they are already filtered server-side.
it('accepts the production checkout contract without for_sale and displays queued renewals', async () => {
  const product = plan(9)
  const offered = { ...product } as Partial<SubscriptionPlan>
  delete offered.for_sale
  getCheckoutInfo.mockResolvedValue({ data: { plans: [offered], subscription_renewal: { activation_at: '2026-11-23T00:00:00Z', legacy_group_ids: [7], scheduled: [{ order_id: 12, plan_id: 9, activation_at: '2026-11-23T00:00:00Z' }] } } })
  const wrapper = render(); await flushPromises()
  expect(wrapper.findAll('[data-plan-group]')).toHaveLength(1)
  expect(wrapper.find('[role="note"]').text()).toContain('renewal.notice')
  expect(wrapper.find('[role="note"]').text()).toContain('renewal.waiting')
})

it('keeps the exact product and duration selected by the comparison guide', async () => {
  getCheckoutInfo.mockResolvedValue({ data: { plans: [plan(1), plan(2, { group_id: 1, validity_days: 3, validity_unit: 'months', price: 170.1 })] } })
  const wrapper = render(); await flushPromises()
  wrapper.findComponent({ name: 'PlanGuide' }).vm.$emit('select', 2)
  await flushPromises()
  const summary = wrapper.find('section[tabindex]')
  expect(summary.findComponent(RouterLinkStub).props('to')).toEqual({ path: '/purchase', query: { tab: 'subscription', plan_id: '2' } })
  expect(summary.text()).toContain('¥170.10 CNY')
})


it('blocks subscription selection and checkout on invalid policy while retaining the top-up link', async () => {
  getCheckoutInfo.mockResolvedValue({ data: { plans: [plan(1)], methods: { wxpay: {} }, subscription_renewal_error: 'INVALID_RENEWAL_POLICY' } })
  const wrapper = render(); await flushPromises()
  expect(wrapper.get('[role="alert"]').text()).toBe('payment.errors.INVALID_RENEWAL_POLICY')
  expect(wrapper.findAll('[data-plan-group]')).toHaveLength(0)
  expect(wrapper.findComponent({ name: 'PlanGuide' }).exists()).toBe(false)
  expect(wrapper.find('section[tabindex]').exists()).toBe(false)
  expect(wrapper.findAllComponents(RouterLinkStub).map(link => link.props('to'))).toEqual(['/purchase?tab=recharge'])
  wrapper.unmount()
})
