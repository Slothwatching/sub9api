import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AdminPaymentPlansView from '../AdminPaymentPlansView.vue'

const { getPlans, getConfig, getGroups, updateConfig } = vi.hoisted(() => ({
  updateConfig: vi.fn(),
  getPlans: vi.fn(),
  getConfig: vi.fn(),
  getGroups: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getPlans,
    getConfig,
    updateConfig,
  },
}))

vi.mock('@/api/admin', () => ({
  default: {
    groups: {
      getAll: getGroups,
    },
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-price" :value="row.price" :row="row" />
      </div>
    </div>
  `,
}

describe('AdminPaymentPlansView', () => {
  beforeEach(() => {
    getGroups.mockResolvedValue([])
    getConfig.mockResolvedValue({ data: {} })
    getPlans.mockResolvedValue({
      data: [
        {
          id: 1,
          name: 'CNY plan',
          group_id: 1,
          price: 499,
          original_price: 599,
          currency: 'CNY',
          validity_days: 30,
          validity_unit: 'day',
          sort_order: 0,
          for_sale: true,
          features: [],
        },
        {
          id: 2,
          name: 'Legacy plan',
          group_id: 1,
          price: 10,
          original_price: 0,
          currency: '',
          validity_days: 30,
          validity_unit: 'day',
          sort_order: 0,
          for_sale: true,
          features: [],
        },
      ],
    })
  })

  it('uses the configured currency symbol and keeps legacy prices in USD', async () => {
    const wrapper = mount(AdminPaymentPlansView, {
      global: {
        plugins: [createPinia()],
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          DataTable: DataTableStub,
          ConfirmDialog: true,
          GroupBadge: true,
          Icon: true,
          PlanEditDialog: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('¥499.00CNY')
    expect(wrapper.text()).toContain('¥599.00')
    expect(wrapper.text()).toContain('$10.00')
  })
})


it('saves an explicit legacy/current policy without changing plan prices or entitlements', async () => {
  getGroups.mockResolvedValue([])
  getConfig.mockResolvedValue({data:{}})
  getPlans.mockResolvedValue({data:[{id:1,name:'Old',group_id:7,price:60,validity_days:30,features:[]},{id:9,name:'New',group_id:16,price:63,validity_days:30,features:[]}]})
  updateConfig.mockResolvedValue({data:{}})
  const wrapper = mount(AdminPaymentPlansView, {global:{plugins:[createPinia()],stubs:{AppLayout:{template:'<div><slot /></div>'},DataTable:DataTableStub,ConfirmDialog:true,GroupBadge:true,Icon:true,PlanEditDialog:true,BaseDialog:{props:['show'],template:'<div v-if="show"><slot /><slot name="footer" /></div>'}}}})
  await flushPromises()
  await wrapper.findAll('button').find(button => button.text() === 'renewal.title')!.trigger('click')
  const inputs = wrapper.findAll('input[type="checkbox"]')
  await inputs[0]!.setValue(true)
  await inputs[1]!.setValue(true)
  await inputs[4]!.setValue(true)
  await wrapper.find('#renewal-policy-form').trigger('submit')
  await flushPromises()
  expect(updateConfig).toHaveBeenCalledWith({subscription_renewal_policy:{enabled:true,legacy_plan_ids:[1],replacement_plan_ids:[9]}})
  wrapper.unmount()
})
