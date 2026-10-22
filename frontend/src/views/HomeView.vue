<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="compact-home flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/brand-mark.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <ThemeToggle />
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/brand-mark.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default branded home; custom and compact settings retain their precedence. -->
  <div v-else class="brand-home" data-testid="default-home">
    <header class="home-container">
      <nav class="home-nav" :aria-label="t('home.workspace.navigation')">
        <div class="home-brand">
          <img :src="siteLogo || '/brand-mark.svg'" :alt="siteName" />
          <span>{{ siteName }}</span>
        </div>
        <div class="home-nav-actions">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="btn btn-ghost">{{ t('home.docs') }}</a>
          <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="btn btn-ghost">{{ t('nav.modelPlaza') }}</router-link>
          <LocaleSwitcher />
          <ThemeToggle />
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn btn-primary">
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>
    <main class="home-container">
      <section class="home-hero">
        <div>
          <p class="home-eyebrow">{{ t('home.workspace.eyebrow') }}</p>
          <h1>{{ siteName }}</h1>
          <p class="home-lead">{{ siteSubtitle }}</p>
          <div class="home-cta">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn btn-primary px-6 py-3">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="sm" class="ml-2" />
            </router-link>
            <router-link v-if="!isAuthenticated && appStore.cachedPublicSettings?.registration_enabled" to="/register" class="btn btn-secondary px-6 py-3">
              {{ t('auth.createAccount') }}
            </router-link>
            <router-link to="/connect" class="btn btn-ghost gap-2 px-4 py-3"><Icon name="book" size="sm" />{{ t('home.workspace.setupGuide') }}</router-link>
          </div>
        </div>
        <div class="home-guide">
          <h2 class="home-guide-heading">{{ t('home.workspace.guideTitle') }}</h2>
          <div v-for="step in [1, 2, 3]" :key="step" class="home-step">
            <span class="home-step-number" aria-hidden="true">0{{ step }}</span>
            <div>
              <h3>{{ t(`home.workspace.step${step}Title`) }}</h3>
              <p>{{ t(`home.workspace.step${step}Description`) }}</p>
            </div>
          </div>
        </div>
      </section>
      <div class="home-tags">
        <span><Icon name="swap" size="sm" />{{ t('home.tags.subscriptionToApi') }}</span>
        <span><Icon name="shield" size="sm" />{{ t('home.tags.stickySession') }}</span>
        <span><Icon name="chart" size="sm" />{{ t('home.tags.realtimeBilling') }}</span>
      </div>
      <section class="home-features">
        <article v-for="(feature, index) in ['unifiedGateway', 'multiAccount', 'balanceQuota']" :key="feature">
          <span class="home-feature-index" aria-hidden="true">0{{ index + 1 }}</span>
          <h2>{{ t(`home.features.${feature}`) }}</h2>
          <p>{{ t(`home.features.${feature}Desc`) }}</p>
        </article>
      </section>
      <section class="home-support" aria-labelledby="home-support-title">
        <div>
          <p class="home-eyebrow">{{ t('home.workspace.supportEyebrow') }}</p>
          <h2 id="home-support-title">{{ t('home.workspace.supportTitle') }}</h2>
          <p>{{ t('home.workspace.supportDescription') }}</p>
        </div>
        <div class="home-support-actions">
          <router-link to="/connect" class="btn btn-primary">
            {{ t('home.workspace.supportPrimary') }}
            <Icon name="arrowRight" size="sm" class="ml-2" />
          </router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="btn btn-secondary">
            {{ t('home.workspace.supportDocs') }}
          </a>
        </div>
      </section>
      <section class="home-providers">
        <div class="home-providers-heading">
          <h2>{{ t('home.providers.title') }}</h2>
          <p>{{ t('home.workspace.providerNote') }}</p>
        </div>
        <div class="home-provider-list">
          <span>GPT</span>
          <span>Codex</span>
        </div>
      </section>
    </main>
    <footer class="home-container home-footer">
      <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
      <div class="home-footer-links">
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('home.docs') }}</a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import ThemeToggle from '@/components/common/ThemeToggle.vue'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sloth Code')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => {
  const configured = appStore.cachedPublicSettings?.site_subtitle?.trim()
  // Older backends return this built-in placeholder for an empty setting.
  return configured && configured !== 'Subscription to API Conversion Platform'
    ? configured
    : t('home.workspace.defaultSubtitle')
})
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})


// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
