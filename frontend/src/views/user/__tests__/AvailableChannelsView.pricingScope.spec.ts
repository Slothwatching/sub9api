import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AvailableChannelsView from '../AvailableChannelsView.vue'

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@/api/channels', () => ({ default: { getAvailable: vi.fn().mockResolvedValue([]) } }))
vi.mock('@/api/groups', () => ({ default: { getUserGroupRates: vi.fn().mockResolvedValue({}) } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))

describe('available channel pricing', () => {
  it('shows the pricing scope alongside the model prices across groups', () => {
    const wrapper = mount(AvailableChannelsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          Icon: true,
          AvailableChannelsTable: { template: '<div data-channel-table />' },
        },
      },
    })

    expect(wrapper.text()).toContain('availableChannels.pricingScope')
    expect(wrapper.find('[data-channel-table]').exists()).toBe(true)
  })
})
