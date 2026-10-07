import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true)
  })
}))

vi.mock('file-saver', () => ({
  saveAs: vi.fn()
}))

import UseKeyContent from '../UseKeyContent.vue'

// Fork-local: /connect renders this guide with a YOUR_API_KEY placeholder for
// visitors. The Codex catalog can only be fetched with a real key, so its box
// stays hidden there, while the generated config still references the remote
// catalog that upstream b5efbe3f4 enables by default.
describe('UseKeyContent', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('hides the Codex catalog box for the /connect placeholder key but keeps the remote catalog URL', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(UseKeyContent, {
      props: {
        show: true,
        apiKey: 'YOUR_API_KEY',
        baseUrl: 'https://gateway.example.com',
        platform: 'openai'
      },
      global: {
        stubs: {
          Icon: { template: '<span />' }
        }
      }
    })
    const transports = ['keys.useKeyModal.cliTabs.codexCli', 'keys.useKeyModal.cliTabs.codexCliWs']
    const selectTransport = (transport: string) =>
      wrapper.findAll('button').find((button) => button.text().trim() === transport)!.trigger('click')
    const openAIConfig = () => wrapper.findAll('pre code')
      .map((code) => code.text())
      .find((content) => content.includes('model_provider = "OpenAI"'))

    for (const transport of transports) {
      await selectTransport(transport)
      expect(wrapper.find('[data-testid="codex-model-catalog"]').exists(), transport).toBe(false)
      expect(openAIConfig(), transport).toContain(
        'base_url = "https://gateway.example.com/v1"\nmodel_catalog_url = "https://gateway.example.com/v1/models"'
      )
      expect(openAIConfig(), transport).not.toContain('model_catalog_json')
    }

    // A signed-in user's selected key shows the catalog on both transports.
    await wrapper.setProps({ apiKey: 'test-only-key' })
    for (const transport of transports) {
      await selectTransport(transport)
      expect(wrapper.find('[data-testid="codex-model-catalog"]').exists(), transport).toBe(true)
    }
    expect(fetchMock).not.toHaveBeenCalled()
  })

  // Fork-local: upstream d1ba57977 added the TypeSafe System One guide to the
  // monolithic UseKeyModal; this fork renders guides from UseKeyContent, so the
  // port is covered here.
  it('offers only the System One curl guide for TypeSafe groups', async () => {
    const wrapper = mount(UseKeyContent, {
      props: {
        show: true,
        apiKey: 'test-only-key',
        baseUrl: 'https://gateway.example.com/v1',
        platform: 'typesafe'
      },
      global: {
        stubs: {
          Icon: { template: '<span />' }
        }
      }
    })
    const labels = wrapper.findAll('button').map((button) => button.text().trim())
    expect(labels).toContain('keys.useKeyModal.cliTabs.systemOne')
    expect(labels).not.toContain('keys.useKeyModal.cliTabs.claudeCode')
    expect(labels).not.toContain('keys.useKeyModal.cliTabs.codexCli')
    expect(labels).not.toContain('keys.useKeyModal.cliTabs.opencode')
    expect(wrapper.text()).toContain('keys.useKeyModal.typesafe.description')

    const config = () => wrapper.findAll('pre code').map((code) => code.text()).join('\n')
    expect(config()).toContain('curl -X POST "https://gateway.example.com/v1/systemone"')
    expect(config()).toContain('"model": "jev-latest"')

    await wrapper.findAll('button').find((button) => button.text().trim() === 'PowerShell')!.trigger('click')
    expect(config()).toContain('Invoke-RestMethod -Method Post -Uri "https://gateway.example.com/v1/systemone"')
  })

  // Fork-local: upstream bbba01dae enables api_key_model_discovery whenever the
  // generated Codex config points at the remote model catalog. It changed only
  // the monolithic UseKeyModal; this fork generates configs in UseKeyContent.
  it('enables API-key model discovery only with the remote Codex catalog', async () => {
    vi.stubGlobal('fetch', vi.fn())
    const mountFor = (platform: string) => mount(UseKeyContent, {
      props: {
        show: true,
        apiKey: 'test-only-key',
        baseUrl: 'https://gateway.example.com',
        platform
      },
      global: {
        stubs: {
          Icon: { template: '<span />' }
        }
      }
    })
    const configText = (wrapper: ReturnType<typeof mountFor>) =>
      wrapper.findAll('pre code').map((code) => code.text()).join('\n')
    const selectTab = (wrapper: ReturnType<typeof mountFor>, label: string) =>
      wrapper.findAll('button').find((button) => button.text().trim() === label)!.trigger('click')

    const openai = mountFor('openai')
    for (const transport of ['keys.useKeyModal.cliTabs.codexCli', 'keys.useKeyModal.cliTabs.codexCliWs']) {
      await selectTab(openai, transport)
      await openai.find('[data-testid="codex-model-catalog-mode"]').setValue('remote')
      expect(configText(openai), transport).toContain('[features]\napi_key_model_discovery = true\n')
      await openai.find('[data-testid="codex-model-catalog-mode"]').setValue('file')
      expect(configText(openai), transport).not.toContain('api_key_model_discovery')
    }

    for (const platform of ['grok', 'deepseek']) {
      const wrapper = mountFor(platform)
      await selectTab(wrapper, 'keys.useKeyModal.cliTabs.codexCli')
      expect(configText(wrapper), platform).toContain('model_catalog_url = "https://gateway.example.com/v1/models"')
      expect(configText(wrapper), platform).toContain('[features]\napi_key_model_discovery = true')
    }
  })
})
