import type { SubscriptionPlan } from '@/types/payment'

export function planDays(plan: SubscriptionPlan): number {
  const unit = plan.validity_unit.replace(/s$/, '')
  if (!['day', 'week', 'month'].includes(unit) || !Number.isFinite(plan.validity_days) || plan.validity_days <= 0) return 0
  return plan.validity_days * (unit === 'month' ? 30 : unit === 'week' ? 7 : 1)
}

// D103: independent full-window projections, never a sum or remaining balance.
export function quotaBatches(plan: SubscriptionPlan, batchCost: number) {
  if (!Number.isFinite(batchCost) || batchCost <= 0) return []
  return (['daily', 'weekly', 'monthly'] as const).flatMap(window => {
    const cap = plan[`${window}_limit_usd`]
    if (cap == null || !Number.isFinite(cap) || cap <= 0) return []
    const count = Math.floor(cap / batchCost)
    return Number.isSafeInteger(count) ? [{ window, cap, count }] : []
  })
}

// D95: compare configured products in the same currency, never invented tiers.
export function recommendPlan(plans: SubscriptionPlan[], kind: 'daily' | 'monthly', spend: number, currency: string, weeklySpend = spend): SubscriptionPlan | undefined {
  if (!Number.isFinite(spend) || spend <= 0) return undefined
  if (kind === 'monthly' && (!Number.isFinite(weeklySpend) || weeklySpend <= 0 || weeklySpend > spend)) return undefined
  // D103: an absent daily cap must not exclude weekly/monthly subscriptions.
  // Bursty demand supplies its busiest week explicitly; never infer it from /4.
  const covers = (limit: number | null | undefined, demand: number) => limit == null || Number.isFinite(limit) && limit > 0 && limit >= demand
  return plans.filter(p => p.for_sale !== false && p.group_platform === 'openai' && (p.currency || '') === currency &&
    Number.isFinite(p.price) && p.price >= 0 && planDays(p) > 0 &&
    (kind === 'daily'
      ? covers(p.daily_limit_usd, spend) && covers(p.weekly_limit_usd, spend * 7) && covers(p.monthly_limit_usd, spend * 30)
      : p.daily_limit_usd == null && p.monthly_limit_usd != null && covers(p.monthly_limit_usd, spend) && covers(p.weekly_limit_usd, weeklySpend)))
    .sort((a, b) => a.price / planDays(a) - b.price / planDays(b) || a.id - b.id)[0]
}
