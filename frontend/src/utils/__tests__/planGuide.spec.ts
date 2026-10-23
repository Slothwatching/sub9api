import { describe, expect, it } from 'vitest'
import { recommendPlan } from '../planGuide'
import type { SubscriptionPlan } from '@/types/payment'
const plan = (id: number, extra: Partial<SubscriptionPlan> = {}): SubscriptionPlan => ({ id, group_id: id, name: `Plan ${id}`, group_platform: 'openai', price: 63, currency: 'CNY', validity_days: 30, validity_unit: 'days', for_sale: true, daily_limit_usd: 5, ...extra } as SubscriptionPlan)
describe('plan selection', () => {
  it('compares normalized term prices within the chosen currency and all quota windows', () => {
    const plans = [plan(1), plan(2, { validity_unit: 'months', validity_days: 3, price: 170 }), plan(3, { price: 1, weekly_limit_usd: 10 }), plan(4, { price: 1, monthly_limit_usd: 10 }), plan(5, { price: 1, currency: 'USD' }), plan(6, { price: 1, for_sale: false })]
    expect(recommendPlan(plans, 'daily', 5, 'CNY')?.id).toBe(2)
    expect(recommendPlan(plans, 'daily', 6, 'CNY')).toBeUndefined()
  })
  it('recommends a monthly pool only when there are no daily or weekly caps', () => {
    const plans = [plan(1, { monthly_limit_usd: 5000 }), plan(2, { daily_limit_usd: null, monthly_limit_usd: 4000 }), plan(3, { daily_limit_usd: null, weekly_limit_usd: 300, monthly_limit_usd: 4000 })]
    expect(recommendPlan(plans, 'monthly', 4000, 'CNY')?.id).toBe(2)
  })
  it('never recommends unknown products, invalid demand or invalid durations', () => {
    const plans = [plan(1, { group_platform: 'anthropic' }), plan(2, { validity_days: 0 }), plan(3, { price: NaN })]
    expect(recommendPlan(plans, 'daily', 1, 'CNY')).toBeUndefined()
    expect(recommendPlan([plan(4)], 'daily', NaN, 'CNY')).toBeUndefined()
  })
})
