<template>
  <section v-if="!auth.isAuthenticated && available" id="video-tutorials" aria-labelledby="video-tutorials-title" class="card flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
    <div class="flex min-w-0 items-start gap-3">
      <span aria-hidden="true" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-800 dark:bg-primary-900/40 dark:text-primary-200"><Icon name="play" size="sm" /></span>
      <div class="min-w-0">
        <h2 id="video-tutorials-title" class="font-semibold">{{ t('commercial.tutorials.title') }}</h2>
        <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('commercial.tutorials.loginPrompt') }}</p>
      </div>
    </div>
    <RouterLink :to="{ path: '/login', query: { redirect: '/connect' } }" class="btn btn-primary shrink-0">{{ t('commercial.tutorials.login') }}</RouterLink>
  </section>

  <section v-else-if="auth.isAuthenticated && (hasContent || failed)" id="video-tutorials" aria-labelledby="video-tutorials-title" class="min-w-0 space-y-5">
    <header class="space-y-2">
      <h2 id="video-tutorials-title" class="text-2xl font-semibold">{{ t('commercial.tutorials.title') }}</h2>
      <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('commercial.tutorials.subtitle') }}</p>
    </header>

    <div v-if="failed" role="alert" class="card flex flex-wrap items-center justify-between gap-3 p-4 text-sm">
      <span>{{ t('commercial.tutorials.loadFailed') }}</span>
      <button type="button" class="btn btn-secondary btn-sm" @click="load">{{ t('common.retry') }}</button>
    </div>

    <template v-else>
      <div v-if="systems.length > 1" role="group" :aria-label="t('commercial.tutorials.os')" class="inline-flex max-w-full flex-wrap gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
        <button v-for="system in systems" :key="system" type="button" :aria-pressed="os === system" class="min-h-9 rounded-md px-4 text-sm transition-colors" :class="os === system ? 'bg-white font-medium text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'" @click="os = system">{{ connectSystemNames[system] }}</button>
      </div>

      <div v-for="video in videos" :key="video.id" class="space-y-2">
        <h3 v-if="videoTitle(video)" class="font-medium">{{ videoTitle(video) }}</h3>
        <div class="overflow-hidden rounded-xl border border-gray-200 bg-black dark:border-dark-700">
          <video :key="video.url" :src="video.url" :poster="video.poster_url || undefined" controls playsinline preload="metadata" class="block aspect-video w-full" :aria-label="videoTitle(video) || t('commercial.tutorials.title')">
            <a :href="video.url" target="_blank" rel="noopener noreferrer">{{ t('commercial.tutorials.openVideo') }}</a>
          </video>
        </div>
      </div>

      <div v-if="downloads.length" class="space-y-3">
        <h3 class="font-semibold">{{ t('commercial.tutorials.downloads') }}</h3>
        <ul class="divide-y divide-gray-200 rounded-xl border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
          <li v-for="item in downloads" :key="item.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="min-w-0 space-y-1">
              <p class="flex flex-wrap items-center gap-2">
                <span class="break-words font-medium">{{ downloadName(item) }}</span>
                <span v-if="item.version" class="rounded-full bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-200">{{ item.version }}</span>
                <span class="text-xs text-gray-500 dark:text-dark-400">{{ item.os === 'all' ? t('commercial.tutorials.allSystems') : connectSystemNames[item.os] }}</span>
              </p>
              <p v-if="downloadNote(item)" class="break-words text-sm text-gray-600 dark:text-dark-300">{{ downloadNote(item) }}</p>
              <button v-if="item.sha256" type="button" class="max-w-full text-left font-mono text-xs text-gray-500 underline-offset-4 hover:underline dark:text-dark-400" :title="item.sha256" :aria-label="t('commercial.tutorials.copySha', { name: downloadName(item) })" @click="copyToClipboard(item.sha256)">SHA-256 {{ item.sha256.slice(0, 16) }}… · {{ t('common.copy') }}</button>
            </div>
            <a :href="item.url" rel="noopener noreferrer" class="btn btn-secondary shrink-0 gap-2" download><Icon name="download" size="sm" />{{ t('commercial.tutorials.download') }}</a>
          </li>
        </ul>
      </div>
    </template>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { getConnectResources } from '@/api/user'
import Icon from '@/components/icons/Icon.vue'
import type { ConnectDownload, ConnectResources, ConnectVideo, ConnectVideoOS } from '@/types'
import { connectSystemNames, connectURL, connectVideoSystems, detectConnectOS, emptyConnectResources } from '@/utils/connectResources'

const { t, locale } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const { copyToClipboard } = useClipboard()

const available = computed(() => app.cachedPublicSettings?.connect_resources_available === true)
const resources = ref<ConnectResources>(emptyConnectResources())
const failed = ref(false)
const preferred = detectConnectOS()
const os = ref<ConnectVideoOS>(preferred)

// Defence in depth: the API already filters, but never render a non-HTTPS link.
const safeVideos = computed(() => resources.value.videos.filter(v => connectURL(v.url) && (!v.poster_url || connectURL(v.poster_url))))
const safeDownloads = computed(() => resources.value.downloads.filter(d => connectURL(d.url)))
const hasContent = computed(() => safeVideos.value.length > 0 || safeDownloads.value.length > 0)
const systems = computed(() => connectVideoSystems.filter(system => safeVideos.value.some(v => v.os === system) || safeDownloads.value.some(d => d.os === system)))
const videos = computed(() => safeVideos.value.filter(v => v.os === os.value))
const downloads = computed(() => safeDownloads.value.filter(d => d.os === 'all' || d.os === os.value || !systems.value.length))
watch(systems, next => { if (next.length && !next.includes(os.value)) os.value = next.includes(preferred) ? preferred : next[0] })

const english = computed(() => locale.value.startsWith('en'))
const videoTitle = (video: ConnectVideo) => (english.value && video.title_en.trim()) || video.title
const downloadName = (item: ConnectDownload) => (english.value && item.name_en.trim()) || item.name
const downloadNote = (item: ConnectDownload) => (english.value && item.note_en.trim()) || item.note

let loadVersion = 0
async function load() {
  const version = ++loadVersion
  failed.value = false
  resources.value = emptyConnectResources()
  if (!auth.isAuthenticated) return
  try {
    const result = await getConnectResources()
    if (version === loadVersion) resources.value = { videos: result.videos ?? [], downloads: result.downloads ?? [] }
  } catch {
    // Only surface the error when the administrator has actually published something.
    if (version === loadVersion) failed.value = available.value
  }
}
watch(() => auth.isAuthenticated, load)
onMounted(load)
onBeforeUnmount(() => { loadVersion++ })
</script>
