<template>
  <section v-if="groups.length" class="card space-y-5 p-5 sm:p-6" aria-labelledby="model-guide-title">
    <div class="flex flex-wrap items-start justify-between gap-3"><div><h2 id="model-guide-title" class="text-xl font-semibold">{{ t('experience.models.title') }}</h2><p class="mt-2 max-w-3xl text-sm leading-6 text-gray-500 dark:text-dark-300">{{ t('experience.models.intro') }}</p></div><RouterLink to="/help?topic=protocol" class="text-sm text-primary-700 underline dark:text-primary-300">{{ t('experience.models.protocolHelp') }} →</RouterLink></div>
    <div class="grid gap-4 sm:grid-cols-2"><label class="min-w-0 space-y-2 text-sm"><span>{{ t('experience.calculator.group') }}</span><select v-model.number="groupId" class="input"><option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option></select></label><label class="min-w-0 space-y-2 text-sm"><span>{{ t('experience.calculator.model') }}</span><select v-model="modelName" class="input"><option v-for="m in models" :key="m.name" :value="m.name">{{ m.name }}</option></select></label></div>
    <template v-if="model">
      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-900"><h3 class="font-medium">{{ t('experience.models.capabilities') }}</h3><dl class="space-y-3 text-sm"><div v-for="field in ['max_input_tokens', 'max_output_tokens', 'supports_vision', 'supports_function_calling'] as const" :key="field" class="flex justify-between gap-4"><dt>{{ t(`experience.models.${field}`) }}</dt><dd class="text-right font-medium">{{ capability(model.catalog_capabilities?.[field]) }}</dd></div></dl><p class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('experience.models.catalogNote') }}</p></div>
        <div class="space-y-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-900"><h3 class="font-medium">{{ t('experience.models.protocols') }}</h3><template v-if="model.pricing?.billing_mode === 'token'"><p class="text-sm">Codex → Responses API</p><code class="block break-all text-xs">POST /v1/responses</code><p class="text-sm">{{ t('experience.models.chatClients') }} → Chat Completions</p><code class="block break-all text-xs">POST /v1/chat/completions</code><p class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('experience.models.protocolNote') }}</p></template><p v-else class="text-sm leading-6">{{ t('experience.models.specialized') }}</p><RouterLink to="/connect" class="inline-block text-sm text-primary-700 underline dark:text-primary-300">{{ t('experience.help.actions.configure') }} →</RouterLink></div>
      </div>
      <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700"><h3 class="font-medium">{{ t('experience.models.availability') }}</h3><p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('experience.models.listed') }}</p><p v-if="monitorFailed" role="status" class="mt-2 text-sm">{{ t('experience.models.monitorFailed') }} <button type="button" class="underline" @click="loadMonitors">{{ t('common.retry') }}</button></p><p v-else-if="monitorLoading" role="status" class="mt-2 text-sm">{{ t('common.loading') }}</p><p v-else-if="!observations.length" class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('experience.models.unknownStatus') }}</p><ul v-else class="mt-3 space-y-2"><li v-for="observation in observations" :key="observation.id" class="flex flex-wrap justify-between gap-2 text-sm"><span>{{ observation.name }} · {{ t(`experience.models.status.${observation.stale ? 'stale' : observation.status}`) }}</span><span class="text-gray-500 dark:text-dark-400">{{ observation.at ? new Date(observation.at).toLocaleString(locale) : t('experience.models.unknown') }}</span></li></ul><p class="mt-3 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('experience.models.monitorNote') }}</p></div>
    </template>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import { list, type UserMonitorView } from '@/api/channelMonitor'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
const props = defineProps<{ groups: ModelPlazaGroup[] }>()
const { t, locale } = useI18n(); const auth = useAuthStore(); const app = useAppStore()
const groups = computed(() => props.groups.filter(g => g.models.some(m => m.platform === 'openai')))
const groupId = ref<number>(); const modelName = ref('')
const group = computed(() => groups.value.find(g => g.id === groupId.value))
const models = computed(() => group.value?.models.filter(m => m.platform === 'openai') ?? [])
const model = computed(() => models.value.find(m => m.name === modelName.value))
watch(groups, next => { if (!next.some(g => g.id === groupId.value)) groupId.value = next[0]?.id }, { immediate: true })
watch(models, next => { if (!next.some(m => m.name === modelName.value)) modelName.value = next[0]?.name ?? '' }, { immediate: true })
function capability(value: number | boolean | null | undefined) {
  if (typeof value === 'boolean') return t(value ? 'experience.models.yes' : 'experience.models.no')
  return typeof value === 'number' && Number.isSafeInteger(value) && value > 0 ? value.toLocaleString(locale.value) : t('experience.models.unknown')
}
const now = ref(Date.now())
let ageTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => { ageTimer = setInterval(() => { now.value = Date.now() }, 30000) })
const monitors = ref<UserMonitorView[]>([]); const monitorFailed = ref(false); const monitorLoading = ref(false)
const enabled = computed(() => auth.isAuthenticated && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.channelMonitor) && groups.value.length > 0)
let version = 0
async function loadMonitors() {
  const current = ++version; monitors.value = []; monitorFailed.value = false; monitorLoading.value = false
  if (!enabled.value) return
  monitorLoading.value = true
  try { const response = await list(); if (current === version && enabled.value) monitors.value = response.items }
  catch { if (current === version) monitorFailed.value = true }
  finally { if (current === version) monitorLoading.value = false }
}
watch([enabled, () => auth.user?.id], loadMonitors, { immediate: true })
const observations = computed(() => monitors.value.filter(m => m.group_name === group.value?.name && m.provider === 'openai' && m.primary_model === modelName.value).map(m => {
  const last = [...m.timeline].sort((a, b) => Date.parse(b.checked_at) - Date.parse(a.checked_at))[0]
  return { id: m.id, name: m.name, at: last?.checked_at, status: last?.status || 'error', stale: !last || !Number.isFinite(Date.parse(last.checked_at)) || now.value - Date.parse(last.checked_at) > 15 * 60 * 1000 || Date.parse(last.checked_at) - now.value > 60000 }
}))
onBeforeUnmount(() => { version++; monitors.value = []; if (ageTimer) clearInterval(ageTimer) })
</script>
