import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, RouterLinkStub } from '@vue/test-utils'
import { reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import ModelGuide from '../ModelGuide.vue'
import CostCalculator from '../CostCalculator.vue'
import HelpView from '@/views/user/HelpView.vue'
import BillingGuideView from '@/views/user/BillingGuideView.vue'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'
const mocks = vi.hoisted(() => ({ list: vi.fn(), plaza: vi.fn(), settings: vi.fn() }))
const auth = reactive({ isAuthenticated: false })
const app = reactive({ cachedPublicSettings: { model_plaza_enabled: true, channel_monitor_enabled: true, payment_enabled: true, subscription_enabled: true }, fetchPublicSettings: mocks.settings })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/api/channelMonitor', () => ({ list: mocks.list }))
vi.mock('@/api/modelPlaza', () => ({ getModelPlaza: mocks.plaza }))
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
beforeEach(() => { vi.clearAllMocks(); auth.isAuthenticated = false; app.cachedPublicSettings.model_plaza_enabled = true; mocks.settings.mockResolvedValue(undefined); mocks.list.mockResolvedValue({ items: [] }) })
describe('GPT customer experience', () => {
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
    expect(wrapper.findAll('details')).toHaveLength(12)
    expect(wrapper.find('#auth').attributes()).toHaveProperty('open')
    await wrapper.find('input[type=search]').setValue('429')
    expect(wrapper.findAll('details')).toHaveLength(1)
    expect(wrapper.find('details').attributes('id')).toBe('rate')
  })
  it('does not fetch or expose price data while the model directory feature is disabled', async () => {
    app.cachedPublicSettings.model_plaza_enabled = false
    const wrapper = mount(BillingGuideView, options()); await flushPromises()
    expect(mocks.plaza).not.toHaveBeenCalled()
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
