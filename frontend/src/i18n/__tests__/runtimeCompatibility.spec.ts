import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

// Keep locale loading, translation and title resolution real; only isolate the
// router/store side effects that otherwise require bootstrapping the whole app.
vi.mock('@/router', () => ({
  default: {
    currentRoute: {
      value: { name: 'Home', params: {}, meta: { titleKey: 'home.pageTitle' } }
    }
  }
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ siteName: 'Example', cachedPublicSettings: null })
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isAdmin: false })
}))
vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] })
}))

describe('application i18n runtime compatibility', () => {
  const originalLocale = localStorage.getItem('sub2api_locale')
  const originalLang = document.documentElement.getAttribute('lang')
  const originalTitle = document.title

  afterEach(() => {
    vi.restoreAllMocks()
    if (originalLocale === null) localStorage.removeItem('sub2api_locale')
    else localStorage.setItem('sub2api_locale', originalLocale)
    if (originalLang === null) document.documentElement.removeAttribute('lang')
    else document.documentElement.setAttribute('lang', originalLang)
    document.title = originalTitle
  })

  it('renders raw locale strings and named parameters, then persists a reactive language switch', async () => {
    localStorage.setItem('sub2api_locale', 'en')
    vi.resetModules()
    const { i18n, initI18n, setLocale, getLocale } = await import('@/i18n')
    const { useI18n } = await import('vue-i18n')
    await initI18n()
    const wrapper = mount(defineComponent({
      setup() {
        const { t } = useI18n()
        return () => h('div', [
          h('span', t('common.save')),
          h('span', t('common.selectedCount', { count: 3 }))
        ])
      }
    }), { global: { plugins: [i18n] } })

    try {
      expect(document.documentElement.lang).toBe('en')
      expect(wrapper.text()).toBe('Save(3 selected)')
      await setLocale('zh')
      await nextTick()
      expect(wrapper.text()).toBe('保存（已选 3 个）')
      expect(getLocale()).toBe('zh')
      expect(localStorage.getItem('sub2api_locale')).toBe('zh')
      expect(document.documentElement.lang).toBe('zh')
      expect(document.title).toBe('首页 - Example')

      await setLocale('en')
      await nextTick()
      expect(wrapper.text()).toBe('Save(3 selected)')
      expect(localStorage.getItem('sub2api_locale')).toBe('en')
      expect(document.documentElement.lang).toBe('en')
      expect(document.title).toBe('Home - Example')
    } finally {
      wrapper.unmount()
    }
  })

  it('compiles previously unused raw messages without unsafe-eval', async () => {
    localStorage.setItem('sub2api_locale', 'en')
    vi.resetModules()
    const { i18n, initI18n, loadLocaleMessages } = await import('@/i18n')
    await initI18n()
    await loadLocaleMessages('zh')

    // Scope the CSP simulation to message compilation so Vitest and jsdom can
    // still use their own constructors outside application translation calls.
    const functionConstructor = vi.spyOn(globalThis, 'Function').mockImplementation(() => {
      throw new Error('unsafe-eval is disabled')
    })
    try {
      expect(i18n.global.t('common.selectedFile', { name: 'report.csv' })).toBe('Selected: report.csv')
      i18n.global.locale.value = 'zh'
      expect(i18n.global.t('common.selectedFile', { name: 'report.csv' })).toBe('已选：report.csv')
      expect(functionConstructor).not.toHaveBeenCalled()
    } finally {
      functionConstructor.mockRestore()
    }
  })
})
