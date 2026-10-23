import type { SubscriptionPlan } from '@/types/payment'

export function planDays(plan: SubscriptionPlan): number {
  const unit = plan.validity_unit.replace(/s$/, '')
  if (!['day', 'week', 'month'].includes(unit) || !Number.isFinite(plan.validity_days) || plan.validity_days <= 0) return 0
  return plan.validity_days * (unit === 'month' ? 30 : unit === 'week' ? 7 : 1)
}

// D95: compare configured products in the same currency, never invented tiers.
export function recommendPlan(plans: SubscriptionPlan[], kind: 'daily' | 'monthly', spend: number, currency: string): SubscriptionPlan | undefined {
  if (!Number.isFinite(spend) || spend <= 0) return undefined
  return plans.filter(p => p.for_sale !== false && p.group_platform === 'openai' && (p.currency || '') === currency &&
    Number.isFinite(p.price) && p.price >= 0 && planDays(p) > 0 &&
    (kind === 'daily'
      ? p.daily_limit_usd != null && p.daily_limit_usd >= spend && (p.weekly_limit_usd == null || p.weekly_limit_usd >= spend * 7) && (p.monthly_limit_usd == null || p.monthly_limit_usd >= spend * 30)
      : p.daily_limit_usd == null && p.weekly_limit_usd == null && p.monthly_limit_usd != null && p.monthly_limit_usd >= spend))
    .sort((a, b) => a.price / planDays(a) - b.price / planDays(b) || a.id - b.id)[0]
}
