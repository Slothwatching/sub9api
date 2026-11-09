import { describe, expect, it } from 'vitest'
import { displayChannels, displayPlazaGroups } from '../displayModels'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { UserAvailableChannel } from '@/api/channels'

describe('curated model display', () => {
  it('hides retired and similarly named OpenAI models without changing prices or other providers', () => {
    const price = { billing_mode: 'token', input_price: 42 }
    const groups = [{ id: 1, models: [
      { name: 'gpt-4', platform: 'openai' },
      { name: 'gpt-6.1-sol-fake', platform: 'openai' },
      { name: 'gpt-6.1-sol', platform: 'openai', pricing: price },
      { name: 'gpt-image-2.6-flare', platform: 'openai' },
      { name: 'gpt-image-2.6-sunburst', platform: 'openai' },
      { name: 'gpt-5.5', platform: 'openai' },
      { name: 'other-model', platform: 'anthropic' }
    ] }, { id: 2, models: [{ name: 'gpt-4', platform: 'openai' }] }] as unknown as ModelPlazaGroup[]
    const visible = displayPlazaGroups(groups)
    expect(visible).toHaveLength(1)
    expect(visible[0].models.map(m => m.name)).toEqual(['gpt-6.1-sol', 'gpt-image-2.6-flare', 'gpt-image-2.6-sunburst', 'other-model'])
    expect(visible[0].models[0].pricing).toBe(price)
    expect(groups[0].models).toHaveLength(7)
  })

  it('filters channels before searching while preserving originally empty and other-provider sections', () => {
    const channels = [{ name: 'OpenAI', description: '', platforms: [
      { platform: 'openai', supported_models: [{ name: 'gpt-4', platform: 'openai' }], groups: [] },
      { platform: 'anthropic', supported_models: [{ name: 'claude', platform: 'anthropic' }], groups: [] },
      { platform: 'empty', supported_models: [], groups: [] }
    ] }] as UserAvailableChannel[]
    const visible = displayChannels(channels)
    expect(visible[0].platforms.map(p => p.platform)).toEqual(['anthropic', 'empty'])
    expect(channels[0].platforms).toHaveLength(3)
  })
})
