import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { UserAvailableChannel } from '@/api/channels'

// Curated display only. This does not control routing, access or billing.
const openAIModels = new Set([
  'gpt-6.1-sol', 'gpt-6-astra', 'gpt-6.1-terra', 'gpt-6-sol',
  'gpt-6-luna', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna'
])

export function isDisplayModel(model: { name: string; platform: string }): boolean {
  return model.platform.toLowerCase() !== 'openai' || openAIModels.has(model.name.toLowerCase())
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
