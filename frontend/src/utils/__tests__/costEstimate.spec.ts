import { describe, expect, it } from 'vitest'
import { estimateCost, type EstimateInput } from '../costEstimate'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
const group = { rate_multiplier: 2, user_rate_multiplier: 0.5, peak_rate_multiplier: 3 } as ModelPlazaGroup
const model = { name: 'configured-gpt', platform: 'openai', pricing: {
  billing_mode: 'token', input_price: 1e-6, output_price: 10e-6, cache_read_price: 0.1e-6, cache_write_price: 1.25e-6,
  cache_write_1h_price: 2e-6, reasoning_effort_multipliers: { high: 2 }, intervals: [
    { min_tokens: 0, max_tokens: 1000, input_price: 1e-6, output_price: 10e-6, cache_read_price: 0.1e-6 },
    { min_tokens: 1000, max_tokens: null, input_price: 2e-6, output_price: 15e-6, cache_read_price: 0.2e-6 },
  ],
}, time_pricing: { timezone: 'UTC', periods: [{ start_time: '01:00', end_time: '02:00', multiplier: 1.5 }] }, long_context_basis: 'whole_request' } as PlazaModel
const values: EstimateInput = { input: 200, cacheRead: 800, cacheWrite: 0, output: 10, requests: 1, effort: '', period: -1, peak: false, cacheWrite1h: false }
describe('cost estimates from configured price schedules', () => {
  it('counts cached input towards the context boundary without charging it twice', () => {
    const at = estimateCost(group, model, values)
    expect(at).not.toHaveProperty('error')
    if ('error' in at) throw new Error(at.error)
    expect(at.total).toBeCloseTo((200e-6 + 80e-6 + 100e-6) * 0.5, 10)
    const above = estimateCost(group, model, { ...values, input: 201 })
    if ('error' in above) throw new Error(above.error)
    expect(above.total).toBeCloseTo((201 * 2e-6 + 800 * 0.2e-6 + 10 * 15e-6) * 0.5, 10)
  })
  it('applies user, reasoning, time and peak multipliers to every identical request', () => {
    const result = estimateCost(group, model, { ...values, requests: 4, effort: 'high', period: 0, peak: true })
    if ('error' in result) throw new Error(result.error)
    expect(result.total).toBeCloseTo(0.00038 * 0.5 * 2 * 1.5 * 3 * 4, 10)
  })
  it('handles explicit tier multipliers and cache-write lifetimes', () => {
    const tiered = { ...model, pricing: { ...model.pricing!, intervals: [{ min_tokens: 0, max_tokens: null, input_price: null, output_price: null, cache_write_price: null, cache_read_price: null, per_request_price: null, input_multiplier: 2, cache_write_multiplier: 3 }] } }
    const result = estimateCost({ ...group, user_rate_multiplier: 1 }, tiered, { ...values, input: 10, cacheRead: 0, cacheWrite: 100, output: 0, cacheWrite1h: true })
    if ('error' in result) throw new Error(result.error)
    expect(result.total).toBeCloseTo(10 * 2e-6 + 100 * 2e-6 * 3, 10)
  })
  it('refuses to guess missing prices or unsupported billing modes', () => {
    expect(estimateCost(group, { ...model, pricing: null }, values)).toEqual({ error: 'unsupported' })
    expect(estimateCost(group, { ...model, pricing: { ...model.pricing!, billing_mode: 'image' } }, values)).toEqual({ error: 'unsupported' })
    expect(estimateCost(group, { ...model, long_context_basis: 'marginal' }, values)).toEqual({ error: 'unsupported' })
    expect(estimateCost(group, { ...model, pricing: { ...model.pricing!, intervals: [], cache_read_price: null } }, values)).toEqual({ error: 'missingPrice' })
    const missing = { ...model, pricing: { ...model.pricing!, intervals: [], input_price: null } }
    expect(estimateCost(group, missing, { ...values, input: 0 })).not.toHaveProperty('error')
  })
  it.each([-1, NaN, Infinity, 0.5, 1_000_000_001])('rejects invalid token input %s', input => {
    expect(estimateCost(group, model, { ...values, input })).toEqual({ error: 'invalid' })
  })
  it('rejects invalid counts and stale multiplier selections', () => {
    expect(estimateCost(group, model, { ...values, requests: 0 })).toEqual({ error: 'invalid' })
    expect(estimateCost(group, model, { ...values, period: 50 })).toEqual({ error: 'invalid' })
    expect(estimateCost(group, model, { ...values, effort: 'unknown' })).toEqual({ error: 'invalid' })
  })
})
