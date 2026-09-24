<template>
  <section class="space-y-5 border-t border-gray-200 pt-6 dark:border-dark-600" data-testid="connect-resource-settings">
    <div class="space-y-1">
      <h3 class="font-semibold">{{ t('commercial.tutorials.admin.title') }}</h3>
      <p class="text-sm text-gray-500 dark:text-dark-300">{{ t('commercial.tutorials.admin.notice') }}</p>
    </div>

    <div class="space-y-3">
      <h4 class="text-sm font-medium">{{ t('commercial.tutorials.admin.videos') }}</h4>
      <div v-for="(video, index) in value.videos" :key="video.id" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <label class="flex items-center gap-2"><input v-model="video.enabled" type="checkbox" />{{ t('commercial.tutorials.admin.enabled') }}</label>
          <div class="flex gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="index === 0" :aria-label="t('commercial.community.up')" @click="move('videos', index, -1)">↑</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="index === value.videos.length - 1" :aria-label="t('commercial.community.down')" @click="move('videos', index, 1)">↓</button>
            <button type="button" class="btn btn-secondary btn-sm" @click="remove('videos', index)">{{ t('common.delete') }}</button>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.os') }}</span>
            <select v-model="video.os" class="input"><option v-for="system in connectVideoSystems" :key="system" :value="system">{{ connectSystemNames[system] }}</option></select>
          </label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.videoTitle') }}</span><input v-model="video.title" class="input" maxlength="80" /></label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.videoTitleEn') }}</span><input v-model="video.title_en" class="input" maxlength="80" /></label>
          <label class="min-w-0 space-y-1 text-sm sm:col-span-2"><span>{{ t('commercial.tutorials.admin.videoUrl') }}</span><input v-model.trim="video.url" class="input" type="url" maxlength="2048" placeholder="https://" /></label>
          <label class="min-w-0 space-y-1 text-sm sm:col-span-2"><span>{{ t('commercial.tutorials.admin.posterUrl') }}</span><input v-model.trim="video.poster_url" class="input" type="url" maxlength="2048" placeholder="https://" /></label>
        </div>
        <video v-if="connectURL(video.url)" :src="video.url" controls preload="metadata" playsinline class="aspect-video w-full max-w-md rounded-lg bg-black" />
      </div>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="value.videos.length >= 12" @click="addVideo">{{ t('commercial.tutorials.admin.addVideo') }}</button>
    </div>

    <div class="space-y-3">
      <h4 class="text-sm font-medium">{{ t('commercial.tutorials.admin.downloads') }}</h4>
      <div v-for="(item, index) in value.downloads" :key="item.id" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <label class="flex items-center gap-2"><input v-model="item.enabled" type="checkbox" />{{ t('commercial.tutorials.admin.enabled') }}</label>
          <div class="flex gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="index === 0" :aria-label="t('commercial.community.up')" @click="move('downloads', index, -1)">↑</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="index === value.downloads.length - 1" :aria-label="t('commercial.community.down')" @click="move('downloads', index, 1)">↓</button>
            <button type="button" class="btn btn-secondary btn-sm" @click="remove('downloads', index)">{{ t('common.delete') }}</button>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.os') }}</span>
            <select v-model="item.os" class="input"><option v-for="system in connectDownloadSystems" :key="system" :value="system">{{ system === 'all' ? t('commercial.tutorials.allSystems') : connectSystemNames[system] }}</option></select>
          </label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.version') }}</span><input v-model.trim="item.version" class="input" maxlength="40" /></label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.name') }}</span><input v-model="item.name" class="input" maxlength="80" /></label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.nameEn') }}</span><input v-model="item.name_en" class="input" maxlength="80" /></label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.note') }}</span><input v-model="item.note" class="input" maxlength="200" /></label>
          <label class="min-w-0 space-y-1 text-sm"><span>{{ t('commercial.tutorials.admin.noteEn') }}</span><input v-model="item.note_en" class="input" maxlength="200" /></label>
          <label class="min-w-0 space-y-1 text-sm sm:col-span-2"><span>{{ t('commercial.tutorials.admin.downloadUrl') }}</span><input v-model.trim="item.url" class="input" type="url" maxlength="2048" placeholder="https://" /></label>
          <label class="min-w-0 space-y-1 text-sm sm:col-span-2"><span>SHA-256</span><input :value="item.sha256" class="input font-mono" maxlength="64" spellcheck="false" @change="item.sha256 = ($event.target as HTMLInputElement).value.trim().toLowerCase()" /></label>
        </div>
      </div>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="value.downloads.length >= 30" @click="addDownload">{{ t('commercial.tutorials.admin.addDownload') }}</button>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ConnectResources } from '@/types'
import { connectDownloadSystems, connectSystemNames, connectURL, connectVideoSystems, emptyConnectResources, newConnectDownload, newConnectVideo } from '@/utils/connectResources'

const props = defineProps<{ modelValue?: ConnectResources }>()
const emit = defineEmits<{ 'update:modelValue': [ConnectResources] }>()
const { t } = useI18n()
const value = computed(() => ({ ...emptyConnectResources(), ...props.modelValue }))

function update(next: Partial<ConnectResources>) { emit('update:modelValue', { ...value.value, ...next }) }
function addVideo() {
  const used = new Set(value.value.videos.map(v => v.os))
  update({ videos: [...value.value.videos, newConnectVideo(connectVideoSystems.find(os => !used.has(os)) ?? 'macos')] })
}
function addDownload() { update({ downloads: [...value.value.downloads, newConnectDownload('macos')] }) }
function remove(list: 'videos' | 'downloads', index: number) { update({ [list]: value.value[list].filter((_, i) => i !== index) }) }
function move(list: 'videos' | 'downloads', index: number, delta: number) {
  const next = [...value.value[list]] as unknown[]
  ;[next[index], next[index + delta]] = [next[index + delta], next[index]]
  update({ [list]: next })
}
</script>
