import { describe, expect, it } from 'vitest'
import { recommendPlan, quotaBatches } from '../planGuide'
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
  it('matches the seven new tiers against both busiest-week and monthly demand', () => {
    const monthly = [165, 330, 660, 1155, 1650, 2805, 4400]
    const weekly = [49.5, 99, 198, 346.5, 495, 841.5, null]
    const prices = [63, 120, 230, 375, 520, 860, 1768]
    const plans = monthly.map((cap, i) => plan(i + 1, { daily_limit_usd: null, weekly_limit_usd: weekly[i], monthly_limit_usd: cap, price: prices[i] }))
    for (let i = 0; i < plans.length; i++) {
      expect(recommendPlan(plans, 'monthly', monthly[i]!, 'CNY', weekly[i] ?? monthly[i])?.id).toBe(i + 1)
    }
    expect(recommendPlan(plans, 'monthly', 165, 'CNY', 50)?.id).toBe(2)
    expect(recommendPlan(plans, 'monthly', 165, 'CNY', 165)?.id).toBe(3)
    expect(recommendPlan(plans, 'daily', 5.5, 'CNY')?.id).toBe(1)
    expect(recommendPlan(plans, 'monthly', 4401, 'CNY', 100)).toBeUndefined()
    expect(recommendPlan(plans, 'monthly', 165, 'CNY', 166)).toBeUndefined()
    expect(recommendPlan(plans, 'monthly', 165, 'CNY', NaN)).toBeUndefined()
  })
  it('projects full-window identical batches without adding weekly quota to monthly quota', () => {
    const offered = plan(1, { daily_limit_usd: null, weekly_limit_usd: 49.5, monthly_limit_usd: 165 })
    expect(quotaBatches(offered, 10)).toEqual([{ window: 'weekly', cap: 49.5, count: 4 }, { window: 'monthly', cap: 165, count: 16 }])
    expect(quotaBatches(plan(7, { daily_limit_usd: null, weekly_limit_usd: null, monthly_limit_usd: 4400 }), 10)).toEqual([{ window: 'monthly', cap: 4400, count: 440 }])
    expect(quotaBatches(offered, 0)).toEqual([])
    expect(quotaBatches(offered, NaN)).toEqual([])
    expect(quotaBatches(offered, -1)).toEqual([])
  })
})
