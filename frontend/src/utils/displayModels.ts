import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { UserAvailableChannel } from '@/api/channels'

// Curated display only. This does not control routing, access or billing.
const openAIModels = new Set([
  'gpt-6.1-sol', 'gpt-6-astra', 'gpt-6.1-terra', 'gpt-6-sol',
  'gpt-6-luna', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna'
])

// ChatGPT-signed-in Codex retirements, checked 2026-11-10.
// https://learn.chatgpt.com/docs/models#deprecated-codex-models
// Exact IDs avoid hiding models with only an API retirement notice.
const retiredOpenAIModels = new Set([
  'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.3-codex-spark'
])

export function isRetiredDisplayModel(name: string, platform = 'openai'): boolean {
  return platform.toLowerCase() === 'openai' && retiredOpenAIModels.has(name.toLowerCase())
}

export function isDisplayModel(model: { name: string; platform: string }): boolean {
  const name = model.name.toLowerCase()
  if (isRetiredDisplayModel(name, model.platform)) return false
  return model.platform.toLowerCase() !== 'openai' ||
    /^(gpt-image-|chatgpt-image-|dall-e-|gpt-audio|gpt-realtime|gpt-live|gpt-transcribe|whisper|tts-)/.test(name) ||
    /^gpt-4o.*(?:audio|realtime|transcribe|tts)/.test(name) || openAIModels.has(name)
}

export function displayPlazaGroups(groups: ModelPlazaGroup[]): ModelPlazaGroup[] {
  return groups.map(group => ({ ...group, models: group.models.filter(isDisplayModel) }))
    .filter(group => group.models.length > 0)
}

export function displayChannels(channels: UserAvailableChannel[]): UserAvailableChannel[] {
  return channels.map(channel => ({
    ...channel,
    platforms: channel.platforms.map(section => ({
      ...section, supported_models: section.supported_models.filter(isDisplayModel)
    })).filter((section, index) => section.supported_models.length > 0 || channel.platforms[index].supported_models.length === 0)
  })).filter(channel => channel.platforms.length > 0)
}
