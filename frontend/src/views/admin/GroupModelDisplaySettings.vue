<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  hydrateModelAllowlistState, buildModelAllowlistConfig, moveModelAllowlistItem,
  selectAllModelAllowlistItems, invertModelAllowlistSelection, addCustomModelAllowlistItem,
  type ModelAllowlistConfig,
} from './groupModelAllowlist'

const props = defineProps<{ modelValue: ModelAllowlistConfig; candidates: string[]; loading: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: ModelAllowlistConfig] }>()
const { t } = useI18n()
function hydrate() {
  const next = hydrateModelAllowlistState(props.modelValue, props.candidates)
  if (props.modelValue.models.length === 0) next.items.forEach(item => { item.selected = false })
  return next
}
const state = reactive(hydrate())
const entry = ref('')
const error = ref('')
watch(() => [props.modelValue, props.candidates], () => {
  Object.assign(state, hydrate())
})
function save() { emit('update:modelValue', buildModelAllowlistConfig(state)) }
function move(from: number, to: number) { moveModelAllowlistItem(state, from, to); save() }
function selectAll() { selectAllModelAllowlistItems(state); save() }
function invert() { invertModelAllowlistSelection(state); save() }
function add() {
  const result = addCustomModelAllowlistItem(state, entry.value)
  error.value = result ?? ''
  if (!result) { entry.value = ''; save() }
}
</script>

<template>
  <div class="border-t pt-4" data-testid="model-display-settings">
    <div class="mb-3 flex items-center justify-between gap-3">
      <div>
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.groups.modelsList.title') }}</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groups.modelsList.hint') }}</p>
      </div>
      <Toggle v-model="state.enabled" :aria-label="t('admin.groups.modelsList.title')" @update:model-value="save" />
    </div>
    <div v-if="state.enabled">
      <div class="mb-2 flex flex-wrap items-center justify-between gap-2 text-xs">
        <span>{{ t('admin.groups.modelAllowlist.selectedSummary', { selected: state.items.filter(item => item.selected).length, total: state.items.length }) }}</span>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary" @click="selectAll">{{ t('admin.groups.modelAllowlist.selectAll') }}</button>
          <button type="button" class="btn btn-secondary" @click="invert">{{ t('admin.groups.modelAllowlist.invertSelection') }}</button>
        </div>
      </div>
      <p v-if="loading" class="text-xs">{{ t('admin.groups.modelAllowlist.loading') }}</p>
      <div class="max-h-64 space-y-2 overflow-y-auto">
        <div v-for="(item, index) in state.items" :key="item.id" class="flex items-center gap-2 py-1">
          <label class="flex min-w-0 flex-1 items-center gap-2 text-sm">
            <input v-model="item.selected" type="checkbox" @change="save" />
            <span class="break-all">{{ item.id }}</span>
          </label>
          <button type="button" class="h-8 w-8 shrink-0 rounded p-1 disabled:opacity-40" :title="t('admin.groups.modelsList.moveUp')" :aria-label="t('admin.groups.modelsList.moveUp')" :disabled="index === 0" @click="move(index, index - 1)"><Icon name="arrowUp" size="sm" /></button>
          <button type="button" class="h-8 w-8 shrink-0 rounded p-1 disabled:opacity-40" :title="t('admin.groups.modelsList.moveDown')" :aria-label="t('admin.groups.modelsList.moveDown')" :disabled="index === state.items.length - 1" @click="move(index, index + 1)"><Icon name="arrowDown" size="sm" /></button>
        </div>
      </div>
      <div class="mt-2 flex gap-2">
        <input v-model="entry" class="input min-w-0 flex-1" :aria-label="t('admin.groups.modelsList.modelId')" :placeholder="t('admin.groups.modelsList.modelId')" @keydown.enter.prevent="add" />
        <button type="button" class="btn btn-primary" @click="add">{{ t('admin.groups.modelAllowlist.addCustom') }}</button>
      </div>
      <p v-if="error" class="mt-1 text-xs text-red-500">{{ t(`admin.groups.modelAllowlist.errors.${error}`) }}</p>
    </div>
  </div>
</template>
