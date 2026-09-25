import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import SubscriptionsView from '../SubscriptionsView.vue'

const { app, auth } = vi.hoisted(() => ({
  app: { cachedPublicSettings: {} as Record<string, unknown>, showError: () => {} },
  auth: { isSimpleMode: false },
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/api/subscriptions', () => ({ default: { getMySubscriptions: vi.fn().mockResolvedValue([]) } }))

const stubs = { AppLayout: { template: '<div><slot /></div>' }, BillingExplanation: true, Icon: true, RouterLink: RouterLinkStub }

beforeEach(() => {
  app.cachedPublicSettings = {}
  auth.isSimpleMode = false
})

describe('SubscriptionsView empty state', () => {
  it('links to the subscription tab when online purchase is available', async () => {
    app.cachedPublicSettings = { payment_enabled: true, subscription_enabled: true }
    const wrapper = mount(SubscriptionsView, { global: { stubs } })
    await flushPromises()
    const link = wrapper.findComponent(RouterLinkStub)
    expect(link.exists()).toBe(true)
    expect(link.props('to')).toEqual({ path: '/purchase', query: { tab: 'subscription' } })
    expect(wrapper.text()).toContain('userSubscriptions.noActiveSubscriptionsPurchaseDesc')
    wrapper.unmount()
  })

  it.each([
    ['payment disabled', { payment_enabled: false, subscription_enabled: true }, false],
    ['subscriptions disabled', { payment_enabled: true, subscription_enabled: false }, false],
    ['simple mode', { payment_enabled: true, subscription_enabled: true }, true],
  ] as const)('keeps the administrator guidance when %s', async (_label, settings, simple) => {
    app.cachedPublicSettings = { ...settings }
    auth.isSimpleMode = simple
    const wrapper = mount(SubscriptionsView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.findComponent(RouterLinkStub).exists()).toBe(false)
    expect(wrapper.text()).toContain('userSubscriptions.noActiveSubscriptionsDesc')
    wrapper.unmount()
  })
})
