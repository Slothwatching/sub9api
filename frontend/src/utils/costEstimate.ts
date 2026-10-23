import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { resolveIntervalPrices } from '@/utils/pricing'

export interface EstimateInput {
  input: number
  output: number
  cacheRead: number
  cacheWrite: number
  cacheWrite1h: boolean
  requests: number
  effort: string
  period: number
  peak: boolean
}
export type EstimateResult = { error: 'invalid' | 'unsupported' | 'missingPrice' } | {
  input: number; output: number; cacheRead: number; cacheWrite: number
  subtotal: number; multiplier: number; total: number; tier: string
}

// Uses the same effective per-token schedules as the model price table. No
// inference is sent and no balance/quota is changed. Unsupported modes fail closed.
export function estimateCost(group: ModelPlazaGroup, model: PlazaModel, values: EstimateInput): EstimateResult {
  const counts = [values.input, values.output, values.cacheRead, values.cacheWrite]
  if (counts.some(v => !Number.isSafeInteger(v) || v < 0 || v > 1_000_000_000) ||
    !Number.isSafeInteger(values.requests) || values.requests < 1 || values.requests > 1_000_000) return { error: 'invalid' }
  if (!Number.isInteger(values.period) || values.period < -1) return { error: 'invalid' }
  const base = model.pricing
  if (!base || base.billing_mode !== 'token' || model.long_context_basis === 'marginal') return { error: 'unsupported' }
  const context = values.input + values.cacheRead + values.cacheWrite
  const intervals = [...base.intervals].sort((a, b) => a.min_tokens - b.min_tokens)
  const interval = intervals.find(iv => (context > iv.min_tokens || context === 0 && iv.min_tokens === 0) && (iv.max_tokens == null || context <= iv.max_tokens))
  if (intervals.length && !interval) return { error: 'missingPrice' }
  const price = interval ? resolveIntervalPrices(interval, base) : base
  const prices = [price.input_price, price.output_price, price.cache_read_price,
    values.cacheWrite1h ? price.cache_write_1h_price : price.cache_write_price]
  if (prices.some((p, i) => counts[i] > 0 && (p == null || !Number.isFinite(p) || p < 0))) return { error: 'missingPrice' }
  const effort = values.effort ? base.reasoning_effort_multipliers?.[values.effort] : 1
  const period = values.period < 0 ? 1 : model.time_pricing?.periods[values.period]?.multiplier
  const peak = values.peak ? group.peak_rate_multiplier : 1
  const rate = group.user_rate_multiplier ?? group.rate_multiplier
  if ([effort, period, peak, rate].some(v => v == null || !Number.isFinite(v) || v < 0)) return { error: 'invalid' }
  const costs = counts.map((v, i) => v * (prices[i] ?? 0))
  const subtotal = costs.reduce((a, b) => a + b, 0)
  const multiplier = rate * effort! * period! * peak
  const total = subtotal * multiplier * values.requests
  if (!Number.isFinite(total)) return { error: 'invalid' }
  return { input: costs[0]!, output: costs[1]!, cacheRead: costs[2]!, cacheWrite: costs[3]!, subtotal, multiplier, total,
    tier: interval?.tier_label || (interval ? `${interval.min_tokens}–${interval.max_tokens ?? '∞'}` : '') }
}
