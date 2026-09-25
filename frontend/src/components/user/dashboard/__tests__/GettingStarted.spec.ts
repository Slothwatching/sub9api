import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import GettingStarted from '../GettingStarted.vue'

const { app } = vi.hoisted(() => ({
  app: { cachedPublicSettings: { available_channels_enabled: false, model_plaza_enabled: false } },
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isSimpleMode: false, user: { balance: 0 } }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/groups', () => ({ getAvailable: vi.fn().mockResolvedValue([]) }))
vi.mock('@/api/keys', () => ({ list: vi.fn().mockResolvedValue({ items: [] }) }))
vi.mock('@/api/subscriptions', () => ({ getActiveSubscriptions: vi.fn().mockResolvedValue([]) }))
vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardStats: vi.fn().mockResolvedValue({ total_requests: 0 }) } }))

beforeEach(() => { app.cachedPublicSettings = { available_channels_enabled: false, model_plaza_enabled: false } })

describe('GettingStarted service destination', () => {
  it.each([
    [true, true, '/available-channels', 'channels'],
    [true, false, '/available-channels', 'channels'],
    [false, true, '/model-plaza?embedded=1', 'services'],
    [false, false, '/connect', 'accessGuide'],
  ] as const)('matches its label to the enabled destination (channels=%s, plaza=%s)', async (channels, plaza, destination, label) => {
    app.cachedPublicSettings = { available_channels_enabled: channels, model_plaza_enabled: plaza }
    const wrapper = mount(GettingStarted, { global: { stubs: { RouterLink: RouterLinkStub } } })
    await flushPromises()
    const links = wrapper.get('ol').findAllComponents(RouterLinkStub)
    expect(links[0].props('to')).toBe(destination)
    expect(links[0].text()).toBe(`commercial.onboarding.${label}`)
    expect(links[1].props('to')).toBe('/keys')
    expect(links[2].props('to')).toBe('/connect')
    expect(links[3].props('to')).toBe('/usage')
    wrapper.unmount()
  })
})

describe('GettingStarted purchase link', () => {
  it.each([
    [true, { path: '/purchase', query: { tab: 'subscription' } }],
    [false, '/purchase'],
  ] as const)('opens the subscription tab only when subscriptions are enabled (subscription=%s)', async (subscription, destination) => {
    app.cachedPublicSettings = { available_channels_enabled: false, model_plaza_enabled: false, payment_enabled: true, subscription_enabled: subscription } as typeof app.cachedPublicSettings
    const wrapper = mount(GettingStarted, { global: { stubs: { RouterLink: RouterLinkStub } } })
    await flushPromises()
    const purchase = wrapper.findAllComponents(RouterLinkStub).find(link => link.text() === 'commercial.onboarding.purchase')
    expect(purchase?.props('to')).toEqual(destination)
    wrapper.unmount()
  })
})
