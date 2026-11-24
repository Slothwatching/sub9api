import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import WeChatRenewalList from '../WeChatRenewalList.vue'
import type { WeChatRenewalAgreement } from '@/types/payment'

const { apiGet, apiPost } = vi.hoisted(() => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: apiGet,
    post: apiPost,
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

describe('WeChatRenewalList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  it('shows cancel button only for pending and active agreements', async () => {
    const agreements: WeChatRenewalAgreement[] = [
      { id: 1, user_id: 10, plan_id: 2, state: 'pending', processing: false },
      { id: 2, user_id: 10, plan_id: 2, state: 'active', processing: false, next_debit_at: '2026-12-24T00:00:00Z' },
      { id: 3, user_id: 10, plan_id: 2, state: 'cancelled', processing: false },
      { id: 4, user_id: 10, plan_id: 2, state: 'failed', failure_code: 'NOTENOUGH', processing: false },
    ]
    apiGet.mockResolvedValueOnce({ data: agreements })

    const wrapper = mount(WeChatRenewalList)
    await flushPromises()

    expect(apiGet).toHaveBeenCalledWith('/payment/wechat-renewals', { params: { page: 1 } })
    const items = wrapper.findAll('li')
    expect(items).toHaveLength(4)
    expect(items[0].find('button').exists()).toBe(true)
    expect(items[1].find('button').exists()).toBe(true)
    expect(items[1].text()).toContain('payment.wechatRenewal.nextDebit')
    expect(items[2].find('button').exists()).toBe(false)
    expect(items[3].find('button').exists()).toBe(false)
    expect(items[3].text()).not.toContain('NOTENOUGH')
  })

  it('renders admin fields and handles cancel success and failure', async () => {
    const agreements: WeChatRenewalAgreement[] = [
      { id: 7, user_id: 42, plan_id: 3, state: 'active', processing: true },
      { id: 8, user_id: 43, plan_id: 3, state: 'failed', failure_code: 'PAY_FAIL', processing: false },
    ]
    apiGet.mockResolvedValueOnce({ data: agreements })
    apiPost.mockRejectedValueOnce(new Error('network'))
    apiGet.mockResolvedValueOnce({ data: agreements })

    const wrapper = mount(WeChatRenewalList, { props: { admin: true } })
    await flushPromises()

    expect(apiGet).toHaveBeenCalledWith('/admin/payment/wechat-renewals', { params: { page: 1 } })
    expect(wrapper.text()).toContain('payment.wechatRenewal.account:{"id":42}')
    expect(wrapper.text()).toContain('payment.wechatRenewal.processing')
    expect(wrapper.text()).toContain('PAY_FAIL')

    await wrapper.find('li button').trigger('click')
    await flushPromises()

    expect(apiPost).toHaveBeenCalledWith('/admin/payment/wechat-renewals/7/cancel')
    expect(wrapper.find('[role="alert"]').text()).toContain('payment.wechatRenewal.cancelError')
  })
})
