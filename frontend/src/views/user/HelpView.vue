<template>
  <component :is="auth.isAuthenticated ? AppLayout : 'div'" :class="auth.isAuthenticated ? '' : 'min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white'">
    <PlazaNavBar v-if="!auth.isAuthenticated" login-redirect="/help" />
    <main class="mx-auto max-w-5xl space-y-6" :class="!auth.isAuthenticated && 'px-4 py-8'">
      <header class="space-y-3"><p class="text-xs font-semibold tracking-widest text-primary-700 dark:text-primary-300">GPT / CODEX</p><h1 class="text-3xl font-semibold">{{ t('experience.help.title') }}</h1><p class="text-gray-600 dark:text-dark-300">{{ t('experience.help.subtitle') }}</p><ExperienceLinks /></header>
      <section class="card space-y-4 p-5 sm:p-6">
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t('experience.help.search') }}</span><input v-model="search" type="search" class="input" :placeholder="t('experience.help.placeholder')" /></label>
        <div class="flex flex-wrap gap-2" role="group" :aria-label="t('experience.help.category')"><button v-for="item in categories" :key="item" type="button" :aria-pressed="category === item" class="rounded-lg px-3 py-2 text-sm" :class="category === item ? 'bg-primary-600 text-white' : 'bg-gray-100 dark:bg-dark-700'" @click="category = item">{{ t(`experience.help.categories.${item}`) }}</button></div>
        <p class="text-sm text-gray-500 dark:text-dark-400" role="status">{{ t('experience.help.results', { count: filtered.length }) }}</p>
      </section>
      <div class="space-y-3">
        <details v-for="article in filtered" :id="article.id" :key="article.id" :open="route.query.topic === article.id || !!search.trim()" class="card scroll-mt-24 p-5 sm:p-6">
          <summary class="cursor-pointer text-lg font-medium">{{ t(`experience.articles.${article.id}.title`) }}</summary>
          <div class="mt-4 space-y-4 text-sm leading-7 text-gray-600 dark:text-dark-300"><p>{{ t(`experience.articles.${article.id}.lead`) }}</p><ol class="list-decimal space-y-2 pl-5"><li v-for="step in ['step1', 'step2', 'step3']" :key="step">{{ t(`experience.articles.${article.id}.${step}`) }}</li></ol>
            <RouterLink v-if="actionEnabled(article.action)" :to="article.action === 'usage' && auth.isSimpleMode ? '/keys' : article.to" class="inline-block font-medium text-primary-700 underline dark:text-primary-300">{{ t(`experience.help.actions.${article.action}`) }} →</RouterLink>
            <RouterLink :to="{ path: '/help', query: { topic: article.id }, hash: `#${article.id}` }" class="ml-5 inline-block text-gray-500 underline dark:text-dark-400" @click="search = ''; category = 'all'">{{ t('experience.help.permalink') }}</RouterLink>
          </div>
        </details>
        <p v-if="!filtered.length" class="card p-8 text-center text-gray-500">{{ t('experience.help.noResults') }}</p>
      </div>
      <section class="card space-y-3 p-5 text-sm leading-6 sm:p-6"><h2 class="font-semibold">{{ t('experience.help.sources') }}</h2><p class="text-gray-500 dark:text-dark-400">{{ t('experience.help.sourceNote') }}</p><div class="flex flex-wrap gap-4 text-primary-700 dark:text-primary-300"><a href="https://learn.chatgpt.com/docs/config-file/config-reference" target="_blank" rel="noopener noreferrer" class="underline">{{ t('experience.help.codexReference') }} ↗</a><a href="https://developers.openai.com/api/docs/guides/prompt-caching" target="_blank" rel="noopener noreferrer" class="underline">{{ t('experience.help.cacheReference') }} ↗</a></div></section>
    </main>
  </component>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import ExperienceLinks from '@/components/experience/ExperienceLinks.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
const { t } = useI18n(); const auth = useAuthStore(); const app = useAppStore(); const route = useRoute()
function actionEnabled(action: string): boolean {
  if (action === 'orders') return !auth.isSimpleMode && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment)
  if (action === 'subscriptions') return !auth.isSimpleMode && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription)
  return true
}
const search = ref(''); const category = ref('all')
const categories = ['all', 'start', 'trouble', 'billing']
const articles = [
  { id: 'quickstart', category: 'start', to: '/connect', action: 'connect' },
  { id: 'codex', category: 'start', to: '/connect#manual-configuration', action: 'configure' },
  { id: 'chat', category: 'start', to: '/connect#manual-configuration', action: 'configure' },
  { id: 'protocol', category: 'start', to: '/connect', action: 'connect' },
  { id: 'auth', category: 'trouble', to: '/connect', action: 'connect' },
  { id: 'quota', category: 'trouble', to: '/subscriptions', action: 'subscriptions' },
  { id: 'model', category: 'trouble', to: '/connect', action: 'configure' },
  { id: 'rate', category: 'trouble', to: '/usage', action: 'usage' },
  { id: 'network', category: 'trouble', to: '/usage', action: 'usage' },
  { id: 'billing', category: 'billing', to: '/billing', action: 'estimate' },
  { id: 'cache', category: 'billing', to: '/billing', action: 'estimate' },
  { id: 'renewal', category: 'billing', to: '/orders', action: 'orders' },
]
const filtered = computed(() => articles.filter(a => (category.value === 'all' || a.category === category.value) &&
  (!search.value.trim() || ['title', 'lead', 'step1', 'step2', 'step3'].some(field => t(`experience.articles.${a.id}.${field}`).toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())))))
onMounted(() => { void app.fetchPublicSettings() })
</script>
