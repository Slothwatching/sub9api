<template>
  <section id="cc-switch-guide" aria-labelledby="cc-switch-title" class="min-w-0">
    <header class="space-y-3 border-b border-gray-200 pb-6 dark:border-dark-700">
      <span class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-700 dark:text-primary-300">
        <Icon name="checkCircle" size="sm" />{{ t('commercial.ccSwitch.recommended') }}
      </span>
      <h2 id="cc-switch-title" class="text-2xl font-semibold">{{ t('commercial.ccSwitch.title') }}</h2>
      <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('commercial.ccSwitch.intro') }}</p>
      <nav :aria-label="t('commercial.connect.steps')" class="flex flex-wrap gap-x-5 gap-y-2 text-sm">
        <a v-for="(step, index) in steps" :key="step" :href="`#cc-switch-${step}`" class="inline-flex items-center gap-2 py-1 text-gray-600 underline-offset-4 hover:text-primary-700 hover:underline dark:text-dark-300 dark:hover:text-primary-300">
          <span class="font-mono text-xs text-primary-700 dark:text-primary-300">0{{ index + 1 }}</span>{{ t(`commercial.ccSwitch.${step}.short`) }}
        </a>
      </nav>
      <a href="#manual-configuration" class="inline-flex items-center gap-1.5 text-sm text-gray-500 underline underline-offset-4 dark:text-dark-400">{{ t('commercial.ccSwitch.otherMethods') }}<Icon name="arrowDown" size="sm" /></a>
    </header>

    <ol class="divide-y divide-gray-200 dark:divide-dark-700">
      <li v-for="(step, index) in steps" :id="`cc-switch-${step}`" :key="step" class="scroll-mt-24 py-7 sm:py-8">
        <div class="flex items-start gap-3 sm:gap-4">
          <span aria-hidden="true" class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-100 font-mono text-sm font-medium text-primary-800 dark:bg-primary-900/40 dark:text-primary-200">{{ index + 1 }}</span>
          <div class="min-w-0 flex-1 space-y-3">
            <h3 class="pt-1 text-lg font-semibold">{{ t(`commercial.ccSwitch.${step}.title`) }}</h3>
            <p class="max-w-3xl text-sm leading-7 text-gray-600 dark:text-dark-300">{{ t(`commercial.ccSwitch.${step}.body`) }}</p>

            <template v-if="step === 'download'">
              <div role="group" :aria-label="t('commercial.ccSwitch.os')" class="inline-flex max-w-full flex-wrap gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
                <button v-for="system in systems" :key="system" type="button" :aria-pressed="os === system" class="min-h-9 rounded-md px-4 text-sm transition-colors" :class="os === system ? 'bg-white font-medium text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'" @click="os = system">{{ systemNames[system] }}</button>
              </div>
              <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t(`commercial.ccSwitch.download.${os}`) }}</p>
              <a :href="downloadUrl" target="_blank" rel="noopener noreferrer" class="btn btn-primary gap-2"><Icon name="download" size="sm" />{{ t('commercial.ccSwitch.download.action') }}<Icon name="externalLink" size="sm" /></a>
            </template>

            <template v-if="step === 'create'">
              <RouterLink to="/keys" class="btn btn-secondary gap-2"><Icon name="key" size="sm" />{{ t('commercial.ccSwitch.create.action') }}<Icon name="arrowRight" size="sm" /></RouterLink>
              <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('commercial.ccSwitch.create.note') }}</p>
            </template>

            <figure v-if="pictures[step]" class="min-w-0 pt-2" :class="step === 'create' ? 'max-w-sm' : step === 'import' ? 'max-w-lg' : ''">
              <button type="button" class="group relative block w-full overflow-hidden rounded-lg border border-gray-200 bg-white text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:border-dark-600" :aria-label="t('commercial.ccSwitch.enlarge', { title: t(`commercial.ccSwitch.${step}.title`) })" @click="openPicture(step)">
                <img :src="pictures[step]!.src" :width="pictures[step]!.width" :height="pictures[step]!.height" :alt="t(`commercial.ccSwitch.${step}.imageAlt`)" loading="lazy" class="block h-auto w-full" />
                <span aria-hidden="true" class="absolute top-2 right-2 flex h-8 w-8 items-center justify-center rounded-md bg-white/95 text-gray-700 shadow-sm"><Icon name="search" size="sm" /></span>
              </button>
              <figcaption class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t(`commercial.ccSwitch.${step}.caption`) }}</figcaption>
            </figure>

            <p v-if="step === 'enable'" class="break-words text-sm leading-6 text-gray-600 dark:text-dark-300">
              <span class="font-medium text-gray-900 dark:text-white">{{ app.siteName }}</span>
              <span class="mx-2 text-gray-400">/</span>{{ baseUrl }}
            </p>

            <template v-if="step === 'verify'">
              <blockquote class="border-l-2 border-primary-500 py-2 pl-4 text-sm text-gray-900 dark:text-white">{{ t('commercial.ccSwitch.verify.prompt') }}</blockquote>
              <RouterLink :to="auth.isSimpleMode ? '/keys' : '/usage'" class="btn btn-secondary gap-2"><Icon name="chart" size="sm" />{{ t(auth.isSimpleMode ? 'commercial.ccSwitch.verify.keyUsage' : 'commercial.ccSwitch.verify.action') }}<Icon name="arrowRight" size="sm" /></RouterLink>
              <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('commercial.ccSwitch.verify.note') }}</p>
            </template>
          </div>
        </div>
      </li>
    </ol>

    <div class="space-y-2 border-t border-gray-200 pt-5 dark:border-dark-700">
      <details v-for="topic in ['open', 'group', 'switch']" :key="topic" class="border-b border-gray-200 py-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm font-medium">{{ t(`commercial.ccSwitch.help.${topic}Q`) }}</summary>
        <p class="mt-3 max-w-3xl text-sm leading-7 text-gray-600 dark:text-dark-300">{{ t(`commercial.ccSwitch.help.${topic}A`) }}</p>
      </details>
      <p class="pt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('commercial.ccSwitch.reference') }} <a :href="manualUrl" target="_blank" rel="noopener noreferrer" class="underline underline-offset-2">{{ t('commercial.ccSwitch.manual') }}</a></p>
    </div>

    <BaseDialog :show="!!enlarged" :title="enlarged ? t(`commercial.ccSwitch.${enlarged}.title`) : ''" width="extra-wide" close-on-click-outside @close="enlarged = null">
      <div v-if="enlarged" tabindex="0" :aria-label="t(`commercial.ccSwitch.${enlarged}.imageAlt`)" class="max-h-[75vh] overflow-auto">
        <img :src="pictures[enlarged]!.src" :alt="t(`commercial.ccSwitch.${enlarged}.imageAlt`)" :width="pictures[enlarged]!.width" :height="pictures[enlarged]!.height" class="mx-auto h-auto" :class="enlarged === 'enable' ? 'w-[1200px] max-w-none' : 'max-w-none sm:max-w-full'" />
      </div>
    </BaseDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'

const { t, locale } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const steps = ['download', 'create', 'import', 'enable', 'verify'] as const
type Step = typeof steps[number]
const systems = ['mac', 'windows', 'linux'] as const
const systemNames = { mac: 'macOS', windows: 'Windows', linux: 'Linux' }
const os = ref<typeof systems[number]>(/Windows/i.test(navigator.userAgent) ? 'windows' : /Linux/i.test(navigator.userAgent) && !/Android/i.test(navigator.userAgent) ? 'linux' : 'mac')
const downloadUrl = 'https://ccswitch.io/'
const manualUrl = computed(() => `https://github.com/farion1231/cc-switch/blob/main/${locale.value === 'zh' ? 'README_ZH.md' : 'README.md'}`)
const baseUrl = computed(() => app.cachedPublicSettings?.api_base_url || window.location.origin)
const pictures: Partial<Record<Step, { src: string; width: number; height: number }>> = {
  create: { src: '/guides/cc-switch/create-key.webp', width: 688, height: 817 },
  import: { src: '/guides/cc-switch/import-key.webp', width: 417, height: 185 },
  enable: { src: '/guides/cc-switch/enable-provider.webp', width: 1875, height: 312 }
}
const enlarged = ref<Step | null>(null)
function openPicture(step: Step) {
  if (pictures[step]) enlarged.value = step
}
</script>
