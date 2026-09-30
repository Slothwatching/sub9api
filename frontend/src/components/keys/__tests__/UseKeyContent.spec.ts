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
})
