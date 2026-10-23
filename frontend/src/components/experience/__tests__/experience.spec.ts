import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, RouterLinkStub } from '@vue/test-utils'
import { reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import ModelGuide from '../ModelGuide.vue'
import CostCalculator from '../CostCalculator.vue'
import HelpView from '@/views/user/HelpView.vue'
import BillingGuideView from '@/views/user/BillingGuideView.vue'
import PlanGuide from '../PlanGuide.vue'
import type { SubscriptionPlan } from '@/types/payment'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'
const mocks = vi.hoisted(() => ({ list: vi.fn(), plaza: vi.fn(), settings: vi.fn(), plans: vi.fn() }))
const auth = reactive({ isAuthenticated: false, isSimpleMode: false })
const app = reactive({ cachedPublicSettings: { model_plaza_enabled: true, model_plaza_require_auth: false, channel_monitor_enabled: true, payment_enabled: true, subscription_enabled: true }, fetchPublicSettings: mocks.settings })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/api/channelMonitor', () => ({ list: mocks.list }))
vi.mock('@/api/modelPlaza', () => ({ getModelPlaza: mocks.plaza }))
vi.mock('@/api/payment', () => ({ paymentAPI: { getCheckoutInfo: mocks.plans } }))
vi.mock('vue-router', async () => ({ ...await vi.importActual('vue-router'), useRoute: () => ({ query: { topic: 'auth' } }) }))
const groups = [{ id: 1, name: 'GPT', rate_multiplier: 1, models: [{ name: 'configured-gpt', platform: 'openai', catalog_capabilities: { supports_vision: false }, pricing: { billing_mode: 'token', input_price: 1e-6, output_price: 2e-6, cache_read_price: 0.1e-6, cache_write_price: null, intervals: [] } }] }] as ModelPlazaGroup[]
// The test runtime omits the i18n string compiler. Supply message functions
// while exercising actual translated content and search behavior.
function compiledMessages(value: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, typeof item === 'string'
    ? (ctx: { named: (name: string) => unknown }) => item.replace(/\{(\w+)\}/g, (_, name: string) => String(ctx.named(name)))
    : compiledMessages(item as Record<string, unknown>)]))
}
function options(locale = 'zh') { return { global: { plugins: [createI18n({ legacy: false, locale, messages: { zh: compiledMessages(zh), en: compiledMessages(en) } })], stubs: { RouterLink: RouterLinkStub, PlazaNavBar: true, AppLayout: { template: '<div><slot /></div>' } } } } }
beforeEach(() => { vi.clearAllMocks(); auth.isAuthenticated = false; app.cachedPublicSettings.model_plaza_enabled = true; app.cachedPublicSettings.model_plaza_require_auth = false; auth.isSimpleMode = false; mocks.settings.mockResolvedValue(undefined); mocks.list.mockResolvedValue({ items: [] }) })
describe('GPT customer experience', () => {
  it('loads sale quotas only for signed-in billing and discards an old plan response after logout', async () => {
    mocks.plaza.mockResolvedValue({ groups })
    const anonymous = mount(BillingGuideView, options()); await flushPromises()
    expect(mocks.plans).not.toHaveBeenCalled()
    anonymous.unmount()
    auth.isAuthenticated = true
    let resolvePlans!: (data: unknown) => void
    mocks.plans.mockReturnValueOnce(new Promise(resolve => { resolvePlans = resolve }))
    const wrapper = mount(BillingGuideView, options()); await flushPromises()
    expect(mocks.plans).toHaveBeenCalledTimes(1)
    auth.isAuthenticated = false; await flushPromises()
    resolvePlans({ data: { plans: [{ id: 1, group_id: 1, monthly_limit_usd: 165 }] } }); await flushPromises()
    expect(wrapper.findComponent(CostCalculator).props('subscriptionPlans')).toEqual([])
    wrapper.unmount()
  })
  it('uses the checkout-info quota contract to display a real signed-in plan forecast', async () => {
    auth.isAuthenticated = true
    mocks.plaza.mockResolvedValue({ groups })
    mocks.plans.mockResolvedValue({ data: { plans: [{ id: 16, group_id: 1, name: 'Current Starter', for_sale: true, weekly_limit_usd: 49.5, monthly_limit_usd: 165 }] } })
    const wrapper = mount(BillingGuideView, options()); await flushPromises()
    expect(wrapper.find('[data-testid=quota-forecast]').text()).toContain('$49.5')
    expect(wrapper.find('[data-testid=quota-forecast]').text()).toContain('$165')
    expect(wrapper.findComponent(CostCalculator).props('subscriptionPlans')[0].id).toBe(16)
    wrapper.unmount()
  })
  it.each(['zh', 'en'])('uses monthly and busiest-week estimates to select a real product in %s', async locale => {
    const offered = [
      { id: 1, group_id: 1, group_platform: 'openai', name: 'Starter', price: 63, currency: 'CNY', validity_days: 30, validity_unit: 'days', daily_limit_usd: null, weekly_limit_usd: 49.5, monthly_limit_usd: 165 },
      { id: 2, group_id: 2, group_platform: 'openai', name: 'Advanced', price: 120, currency: 'CNY', validity_days: 30, validity_unit: 'days', daily_limit_usd: null, weekly_limit_usd: 99, monthly_limit_usd: 330 },
    ] as SubscriptionPlan[]
    const wrapper = mount(PlanGuide, { props: { plans: offered }, ...options(locale) }); await flushPromises()
    expect(wrapper.text()).toContain('Starter')
    await wrapper.findAll('input[type=number]')[1]!.setValue(50)
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('select')).toEqual([[2]])
    wrapper.unmount()
  })
  it('projects the selected real pricing group only, and clears projections when pricing becomes invalid', async () => {
    const offered = [
      { id: 1, group_id: 1, name: 'Matching', weekly_limit_usd: 49.5, monthly_limit_usd: 165 },
      { id: 2, group_id: 99, name: 'Other group', monthly_limit_usd: 999999 },
      { id: 3, group_id: 1, name: 'Retired', for_sale: false, monthly_limit_usd: 999999 },
    ] as SubscriptionPlan[]
    const wrapper = mount(CostCalculator, { props: { groups, subscriptionPlans: offered }, ...options() }); await flushPromises()
    const forecast = wrapper.find('[data-testid=quota-forecast]')
    expect(forecast.text()).toContain('$49.5')
    expect(forecast.text()).toContain('$165')
    expect(forecast.findAll('option').map(o => o.text())).toEqual(['Matching'])
    await wrapper.findAll('input[type=number]')[3]!.setValue(50)
    expect(wrapper.find('[data-testid=quota-forecast]').exists()).toBe(false)
    wrapper.unmount()
  })
  it('keeps unknown capabilities distinct from explicitly unsupported and makes no anonymous monitor request', async () => {
    const wrapper = mount(ModelGuide, { props: { groups }, ...options() }); await flushPromises()
    expect(wrapper.text()).toContain('目录声明不支持')
    expect(wrapper.text()).toContain('未验证')
    expect(wrapper.text()).toContain('/v1/responses')
    expect(mocks.list).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('shows only matching monitor observations, marks old results and clears them on logout', async () => {
    auth.isAuthenticated = true
    mocks.list.mockResolvedValue({ items: [{ id: 1, name: 'Correct', provider: 'openai', group_name: 'GPT', primary_model: 'configured-gpt', timeline: [{ status: 'operational', checked_at: '2020-01-01T00:00:00Z' }] }, { id: 2, name: 'Wrong group', provider: 'openai', group_name: 'Private', primary_model: 'configured-gpt', timeline: [] }] })
    const wrapper = mount(ModelGuide, { props: { groups }, ...options() }); await flushPromises()
    expect(wrapper.text()).toContain('Correct')
    expect(wrapper.text()).toContain('较旧观测')
    expect(wrapper.text()).not.toContain('Wrong group')
    auth.isAuthenticated = false; await flushPromises()
    expect(wrapper.text()).not.toContain('Correct')
    wrapper.unmount()
  })
  it('calculates from selected real prices and rejects missing cache-write prices', async () => {
    const wrapper = mount(CostCalculator, { props: { groups }, ...options() }); await flushPromises()
    expect(wrapper.text()).toContain('$0.01400000')
    await wrapper.findAll('input[type=number]')[3]!.setValue(50)
    expect(wrapper.text()).toContain('价项或档位缺失')
  })
  it.each(['zh', 'en'])('searches translated help content and supports deep-linked articles in %s', async locale => {
    const wrapper = mount(HelpView, options(locale)); await flushPromises()
    expect(wrapper.findAll('details')).toHaveLength(14)
    expect(wrapper.find('#auth').attributes()).toHaveProperty('open')
    await wrapper.find('input[type=search]').setValue('429')
    expect(wrapper.findAll('details')).toHaveLength(1)
    expect(wrapper.find('details').attributes('id')).toBe('rate')
  })
  it('does not fetch or expose price data while the model directory feature is disabled', async () => {
    app.cachedPublicSettings.model_plaza_enabled = false
    app.cachedPublicSettings.model_plaza_require_auth = true
    const wrapper = mount(BillingGuideView, options()); await flushPromises()
    expect(mocks.plaza).not.toHaveBeenCalled()
    expect(mocks.plans).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('暂无可用于试算')
    wrapper.unmount()
  })
  it('discards a pending authenticated estimate response after logout', async () => {
    auth.isAuthenticated = true
    let resolve!: (data: unknown) => void
    mocks.plaza.mockReturnValueOnce(new Promise(yes => { resolve = yes })).mockResolvedValueOnce({ groups: [] })
    const wrapper = mount(BillingGuideView, options()); await flushPromises()
    auth.isAuthenticated = false; await flushPromises()
    resolve({ groups }); await flushPromises()
    expect(wrapper.findComponent(CostCalculator).props('groups')).toEqual([])
    wrapper.unmount()
  })
})

it('keeps public billing guidance readable when prices require authentication', async () => {
  app.cachedPublicSettings.model_plaza_require_auth = true
  const wrapper = mount(BillingGuideView, options()); await flushPromises()
  expect(mocks.plaza).not.toHaveBeenCalled()
  expect(wrapper.text()).toContain('费用说明仍可阅读')
  expect(wrapper.findAllComponents(RouterLinkStub).some(link => JSON.stringify(link.props('to')) === JSON.stringify({ path: '/login', query: { redirect: '/billing' } }))).toBe(true)
  wrapper.unmount()
})
it('retains the existing simple-mode usage destination and hides purchase discovery', async () => {
  auth.isSimpleMode = true
  mocks.plaza.mockResolvedValue({ groups: [] })
  const wrapper = mount(BillingGuideView, options()); await flushPromises()
  const targets = wrapper.findAllComponents(RouterLinkStub).map(link => link.props('to'))
  expect(targets).toContain('/keys')
  expect(targets).not.toContain('/usage')
  expect(targets).not.toContain('/subscribe')
  wrapper.unmount()
})
