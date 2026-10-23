import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import UserOrdersView from '../UserOrdersView.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'

const api = vi.hoisted(() => ({ getMyOrders: vi.fn(), getRefundEligibleProviders: vi.fn() }))
vi.mock('@/api/payment', () => ({ paymentAPI: api }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))
vi.mock('vue-router', async () => ({ ...await vi.importActual('vue-router'), useRouter: () => ({ push: vi.fn() }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key })
}))
enableAutoUnmount(afterEach)

beforeEach(() => {
  vi.clearAllMocks()
  api.getMyOrders.mockResolvedValue({ data: { items: [], total: 100 } })
  api.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: [] } })
})

async function openOrders() {
  const wrapper = mount(UserOrdersView, {
    global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      OrderTable: true, BaseDialog: true, Icon: true, Pagination: true, teleport: true, RouterLink: RouterLinkStub
    } }
  })
  await flushPromises()
  wrapper.getComponent(Pagination).vm.$emit('update:page', 4)
  await flushPromises()
  return wrapper
}

describe('order status filtering', () => {
  // Local refund guidance must match the backend's balance-only request contract.
  it('offers self-service refunds only for completed balance orders on eligible providers', async () => {
    api.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: ['eligible'] } })
    api.getMyOrders.mockResolvedValue({ data: { total: 6, items: [
      { id: 1, order_type: 'balance', status: 'COMPLETED', provider_instance_id: 'eligible' },
      { id: 2, order_type: 'subscription', status: 'COMPLETED', provider_instance_id: 'eligible' },
      { id: 3, order_type: 'subscription', status: 'SCHEDULED', provider_instance_id: 'eligible' },
      { id: 4, order_type: 'balance', status: 'COMPLETED', provider_instance_id: 'disabled' },
      { id: 5, order_type: 'balance', status: 'COMPLETED', provider_instance_id: null },
      { id: 6, order_type: 'balance', status: 'REFUNDED', provider_instance_id: 'eligible' },
    ] } })
    const wrapper = mount(UserOrdersView, { global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      OrderTable: { props: ['orders'], template: '<div><div v-for="row in orders" :key="row.id" :data-order="row.id"><slot name="actions" :row="row" /></div></div>' },
      BaseDialog: true, Icon: true, Pagination: true, RouterLink: RouterLinkStub,
    } } })
    await flushPromises()
    expect(wrapper.get('[data-order="1"]').text()).toContain('payment.orders.requestRefund')
    for (const id of [2, 3, 4, 5, 6]) {
      expect(wrapper.get(`[data-order="${id}"]`).text()).not.toContain('payment.orders.requestRefund')
    }
    expect(wrapper.findComponent(RouterLinkStub).props('to')).toEqual({ path: '/help', query: { topic: 'refund' }, hash: '#refund' })
  })
  it('loads the first page with the selected status', async () => {
    const wrapper = await openOrders()
    const select = wrapper.getComponent(Select)
    await select.get('button').trigger('click')
    await select.findAll('[role="option"]').find(option => option.text() === 'payment.status.pending')!.trigger('click')
    await flushPromises()
    expect(api.getMyOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 20, status: 'PENDING' })
    expect(wrapper.getComponent(Pagination).props('page')).toBe(1)
    expect(api.getMyOrders).toHaveBeenCalledTimes(3)
  })

  it('keeps the current page on manual refresh', async () => {
    const wrapper = await openOrders()
    await wrapper.get('[title="common.refresh"]').trigger('click')
    await flushPromises()
    expect(api.getMyOrders).toHaveBeenLastCalledWith({ page: 4, page_size: 20, status: undefined })
  })
})
