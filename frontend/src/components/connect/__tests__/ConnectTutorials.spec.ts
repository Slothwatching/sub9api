import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import type { ConnectDownload, ConnectResources, ConnectVideo } from '@/types'

const { app, auth, locale, copy, getConnectResources } = vi.hoisted(() => ({
  app: { cachedPublicSettings: { connect_resources_available: false } as Record<string, unknown> },
  auth: { isAuthenticated: false },
  locale: { value: 'zh' },
  copy: vi.fn(),
  getConnectResources: vi.fn()
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: copy }) }))
vi.mock('@/api/user', () => ({ getConnectResources }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale }) }))

import ConnectTutorials from '../ConnectTutorials.vue'
import ConnectResourceSettings from '../ConnectResourceSettings.vue'
import { connectResourcesValidationError, detectConnectOS } from '@/utils/connectResources'

const video = (over: Partial<ConnectVideo> = {}): ConnectVideo => ({ id: 'mac', os: 'macos', title: 'Mac 教程', title_en: 'Mac guide', url: 'https://media.example.com/mac.mp4', poster_url: '', enabled: true, ...over })
const download = (over: Partial<ConnectDownload> = {}): ConnectDownload => ({ id: 'ccs', os: 'macos', name: 'CC Switch', name_en: '', version: '3.20.2', note: '', note_en: '', url: 'https://media.example.com/ccs.dmg', sha256: 'a'.repeat(64), enabled: true, ...over })
const options = { global: { stubs: { RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' }, Icon: true } } }
const macUA = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)'
const winUA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'

function setUA(ua: string) { Object.defineProperty(window.navigator, 'userAgent', { value: ua, configurable: true }) }

beforeEach(() => {
  app.cachedPublicSettings = { connect_resources_available: false }
  auth.isAuthenticated = false
  locale.value = 'zh'
  copy.mockClear()
  getConnectResources.mockReset()
  setUA(macUA)
})

describe('ConnectTutorials', () => {
  it('shows nothing to visitors when no resources are published', () => {
    const w = mount(ConnectTutorials, options)
    expect(w.find('section').exists()).toBe(false)
    expect(getConnectResources).not.toHaveBeenCalled()
  })

  it('asks visitors to sign in without requesting media addresses', () => {
    app.cachedPublicSettings = { connect_resources_available: true }
    const w = mount(ConnectTutorials, options)
    expect(w.text()).toContain('commercial.tutorials.loginPrompt')
    expect(w.get('a').attributes('data-to')).toContain('/connect')
    expect(w.find('video').exists()).toBe(false)
    expect(getConnectResources).not.toHaveBeenCalled()
  })

  it('defaults to the visitor system and switches between systems', async () => {
    setUA(winUA)
    auth.isAuthenticated = true
    getConnectResources.mockResolvedValue({ videos: [video(), video({ id: 'win', os: 'windows', url: 'https://media.example.com/win.mp4' })], downloads: [download(), download({ id: 'node', os: 'all', name: 'Node.js' })] } satisfies ConnectResources)
    const w = mount(ConnectTutorials, options)
    await flushPromises()
    expect(w.get('video').attributes('src')).toBe('https://media.example.com/win.mp4')
    expect(w.text()).toContain('Node.js')
    expect(w.text()).not.toContain('CC Switch')
    const mac = w.findAll('button').find(b => b.text() === 'macOS')!
    await mac.trigger('click')
    expect(w.get('video').attributes('src')).toBe('https://media.example.com/mac.mp4')
    expect(w.text()).toContain('CC Switch')
    expect(w.text()).toContain('Node.js')
    // Official store pages open beside /connect instead of replacing it.
    const link = w.findAll('a').find(a => a.attributes('href') === 'https://media.example.com/ccs.dmg')!
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toBe('noopener noreferrer')
  })

  it('shows one all-systems video under every system tab', async () => {
    setUA(winUA)
    auth.isAuthenticated = true
    getConnectResources.mockResolvedValue({ videos: [video({ id: 'shared', os: 'all', url: 'https://media.example.com/shared.mp4' })], downloads: [download(), download({ id: 'ccs-win', os: 'windows', name: 'CC Switch Windows' })] })
    const w = mount(ConnectTutorials, options)
    await flushPromises()
    expect(w.findAll('video').map(v => v.attributes('src'))).toEqual(['https://media.example.com/shared.mp4'])
    await w.findAll('button').find(b => b.text() === 'macOS')!.trigger('click')
    expect(w.findAll('video').map(v => v.attributes('src'))).toEqual(['https://media.example.com/shared.mp4'])
  })

  it('needs no system tabs when the only content is an all-systems video', async () => {
    auth.isAuthenticated = true
    getConnectResources.mockResolvedValue({ videos: [video({ os: 'all' })], downloads: [] })
    const w = mount(ConnectTutorials, options)
    await flushPromises()
    expect(w.find('[role="group"]').exists()).toBe(false)
    expect(w.get('video').attributes('src')).toBe('https://media.example.com/mac.mp4')
  })

  it('falls back to an available system and uses English text when present', async () => {
    setUA(winUA)
    locale.value = 'en'
    auth.isAuthenticated = true
    getConnectResources.mockResolvedValue({ videos: [video()], downloads: [] })
    const w = mount(ConnectTutorials, options)
    await flushPromises()
    expect(w.get('video').attributes('src')).toBe('https://media.example.com/mac.mp4')
    expect(w.text()).toContain('Mac guide')
    expect(w.findAll('[role="group"]')).toHaveLength(0)
  })

  it('never renders non-HTTPS links and copies checksums', async () => {
    auth.isAuthenticated = true
    getConnectResources.mockResolvedValue({ videos: [video({ url: 'javascript:alert(1)' })], downloads: [download(), download({ id: 'bad', name: 'Bad', url: 'http://media.example.com/x.exe' })] })
    const w = mount(ConnectTutorials, options)
    await flushPromises()
    expect(w.find('video').exists()).toBe(false)
    expect(w.text()).not.toContain('Bad')
    await w.findAll('button').find(b => b.text().startsWith('SHA-256'))!.trigger('click')
    expect(copy).toHaveBeenCalledWith('a'.repeat(64))
  })

  it('stays hidden when signed-in users have nothing to see, and reports failures only when published', async () => {
    auth.isAuthenticated = true
    getConnectResources.mockResolvedValue({ videos: [], downloads: [] })
    const empty = mount(ConnectTutorials, options)
    await flushPromises()
    expect(empty.find('section').exists()).toBe(false)

    getConnectResources.mockRejectedValue(new Error('offline'))
    const hidden = mount(ConnectTutorials, options)
    await flushPromises()
    expect(hidden.find('[role="alert"]').exists()).toBe(false)

    app.cachedPublicSettings = { connect_resources_available: true }
    const failed = mount(ConnectTutorials, options)
    await flushPromises()
    expect(failed.get('[role="alert"]').text()).toContain('commercial.tutorials.loadFailed')
  })
})

describe('connect resource settings', () => {
  it('validates enabled entries, HTTPS links and checksums', () => {
    expect(connectResourcesValidationError({ videos: [video()], downloads: [download()] })).toBeNull()
    expect(connectResourcesValidationError({ videos: [video({ url: '' })], downloads: [] })).toBe('commercial.tutorials.admin.videoRequired')
    expect(connectResourcesValidationError({ videos: [video({ enabled: false, url: '' })], downloads: [] })).toBeNull()
    expect(connectResourcesValidationError({ videos: [video({ url: 'http://media.example.com/a.mp4' })], downloads: [] })).toBe('commercial.tutorials.admin.linkError')
    expect(connectResourcesValidationError({ videos: [], downloads: [download({ url: 'https://u:p@media.example.com/a' })] })).toBe('commercial.tutorials.admin.linkError')
    expect(connectResourcesValidationError({ videos: [], downloads: [download({ name: ' ' })] })).toBe('commercial.tutorials.admin.downloadRequired')
    expect(connectResourcesValidationError({ videos: [], downloads: [download({ sha256: 'xyz' })] })).toBe('commercial.tutorials.admin.shaError')
  })

  it('detects desktop systems', () => {
    expect(detectConnectOS(winUA)).toBe('windows')
    expect(detectConnectOS('Mozilla/5.0 (X11; Linux x86_64)')).toBe('linux')
    expect(detectConnectOS('Mozilla/5.0 (Linux; Android 14)')).toBe('macos')
    expect(detectConnectOS(macUA)).toBe('macos')
  })

  it('defaults a first video to all systems, then to the next unused system', async () => {
    const empty = mount(ConnectResourceSettings, { props: { modelValue: { videos: [], downloads: [] } } })
    await empty.findAll('button').find(b => b.text() === 'commercial.tutorials.admin.addVideo')!.trigger('click')
    expect((empty.emitted('update:modelValue')![0][0] as ConnectResources).videos.map(v => v.os)).toEqual(['all'])
    expect(empty.findAll('select').length).toBe(0)
  })

  it('adds a video for the next unused system and reorders downloads', async () => {
    const value = reactive<ConnectResources>({ videos: [video({ os: 'all' })], downloads: [download(), download({ id: 'two' })] })
    const w = mount(ConnectResourceSettings, { props: { modelValue: value } })
    expect(w.get('select').findAll('option').map(o => o.attributes('value'))).toContain('all')
    await w.findAll('button').find(b => b.text() === 'commercial.tutorials.admin.addVideo')!.trigger('click')
    const added = w.emitted('update:modelValue')![0][0] as ConnectResources
    expect(added.videos.map(v => v.os)).toEqual(['all', 'macos'])
    expect(added.videos[1].enabled).toBe(false)
    const up = w.findAll('button').filter(b => b.attributes('aria-label') === 'commercial.community.up')
    await up[up.length - 1].trigger('click')
    expect((w.emitted('update:modelValue')![1][0] as ConnectResources).downloads.map(d => d.id)).toEqual(['two', 'ccs'])
  })
})
