import { describe, expect, it } from 'vitest'
import { displayChannels, displayPlazaGroups, isRetiredDisplayModel } from '../displayModels'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { UserAvailableChannel } from '@/api/channels'

describe('curated model display', () => {
  it('shows the verified GPT-6.2 models in the plaza and channels without exposing lookalikes', () => {
    const models = ['gpt-6.2-sol', 'gpt-6.2-astra', 'gpt-6.2-sol-preview']
      .map(name => ({ name, platform: 'openai' }))
    const groups = [{ id: 1, models }] as unknown as ModelPlazaGroup[]
    const channels = [{ name: 'OpenAI', description: '', platforms: [
      { platform: 'openai', supported_models: models, groups: [] }
    ] }] as unknown as UserAvailableChannel[]
    const expected = ['gpt-6.2-sol', 'gpt-6.2-astra']
    expect(displayPlazaGroups(groups)[0].models.map(m => m.name)).toEqual(expected)
    expect(displayChannels(channels)[0].platforms[0].supported_models.map(m => m.name)).toEqual(expected)
  })

  it('hides retired and similarly named OpenAI models without changing prices or other providers', () => {
    const price = { billing_mode: 'token', input_price: 42 }
    const groups = [{ id: 1, models: [
      { name: 'gpt-4', platform: 'openai' },
      { name: 'gpt-6.1-sol-fake', platform: 'openai' },
      { name: 'gpt-6.1-sol', platform: 'openai', pricing: price },
      { name: 'gpt-image-2.6-flare', platform: 'openai' },
      { name: 'gpt-image-2.6-sunburst', platform: 'openai' },
      { name: 'gpt-image-1', platform: 'openai' },
      { name: 'chatgpt-image-latest', platform: 'openai' },
      { name: 'dall-e-3', platform: 'openai' },
      { name: 'gpt-4o-mini-transcribe', platform: 'openai' },
      { name: 'gpt-5.5', platform: 'openai' },
      { name: 'other-model', platform: 'anthropic' }
    ] }, { id: 2, models: [{ name: 'gpt-4', platform: 'openai' }] }] as unknown as ModelPlazaGroup[]
    const visible = displayPlazaGroups(groups)
    expect(visible).toHaveLength(1)
    expect(visible[0].models.map(m => m.name)).toEqual(['gpt-6.1-sol', 'gpt-image-2.6-flare', 'gpt-image-2.6-sunburst', 'gpt-image-1', 'chatgpt-image-latest', 'dall-e-3', 'gpt-4o-mini-transcribe', 'other-model'])
    expect(visible[0].models[0].pricing).toBe(price)
    expect(groups[0].models).toHaveLength(11)
  })

  it('hides exact retired IDs while preserving future and other-provider models', () => {
    expect(isRetiredDisplayModel('gpt-5.5')).toBe(true)
    expect(isRetiredDisplayModel('gpt-5.3-codex-spark')).toBe(true)
    expect(isRetiredDisplayModel('gpt-image-1')).toBe(false)
    expect(isRetiredDisplayModel('gpt-image-1-mini')).toBe(false)
    expect(isRetiredDisplayModel('gpt-4o-audio-preview-2025-06-03')).toBe(false)
    expect(isRetiredDisplayModel('gpt-image-1', 'other')).toBe(false)
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
