import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const { getPublicSettingsMock, pushMock, sendCodeMock, loginCodeMock, loginMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  pushMock: vi.fn(),
  sendCodeMock: vi.fn(),
  loginCodeMock: vi.fn(),
  loginMock: vi.fn()
}))

const publicSettings = {
  registration_enabled: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: '',
  aliyun_captcha_enabled: false,
  aliyun_captcha_scene_id: '',
  aliyun_captcha_prefix: '',
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  backend_mode_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  password_reset_enabled: false,
  passkey_enabled: false,
  login_agreement_enabled: false,
  login_agreement_documents: []
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
    currentRoute: { value: { query: {} } }
  })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: loginMock,
    loginWithEmailCode: loginCodeMock,
    loginWithPasskey: vi.fn(),
    login2FA: vi.fn()
  }),
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/api/auth', () => ({
  buildOAuthLoginStartURL: vi.fn(),
  getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
  isTotp2FARequired: vi.fn(() => false),
  isWeChatWebOAuthEnabled: vi.fn(() => false),
  startOAuthLogin: vi.fn(),
  sendEmailLoginCode: (...args: unknown[]) => sendCodeMock(...args)
}))

function mountLogin() {
  return mount(LoginView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        DingTalkOAuthSection: true,
        EmailOAuthButtons: true,
        Icon: true,
        LinuxDoOAuthSection: true,
        LoginAgreementPrompt: true,
        OidcOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TotpLoginModal: true,
        TurnstileWidget: true,
        WechatOAuthSection: true,
        transition: false
      }
    }
  })
}

describe('LoginView registration entry', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    pushMock.mockReset()
    sendCodeMock.mockReset()
    loginCodeMock.mockReset()
    loginMock.mockReset()
    getPublicSettingsMock.mockResolvedValue(publicSettings)
  })

  it('shows the registration entry when registration is enabled', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain('auth.signUp')
  })

  it('hides the registration entry when registration is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      registration_enabled: false
    })

    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).not.toContain('auth.signUp')
  })
})

// Fork-local D85: feature switch and passwordless sign-in regressions.
describe('LoginView email code login', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getPublicSettingsMock.mockResolvedValue({ ...publicSettings, email_code_login_enabled: true })
    sendCodeMock.mockResolvedValue({ countdown: 60 })
    loginCodeMock.mockResolvedValue({ access_token: 'test-token', user: {} })
  })
  it('keeps password login by default and hides the new method without its switch', async () => {
    getPublicSettingsMock.mockResolvedValueOnce(publicSettings)
    const wrapper = mountLogin()
    await flushPromises()
    expect(wrapper.find('#password').exists()).toBe(true)
    expect(wrapper.find('[data-testid="email-code-tab"]').exists()).toBe(false)
    wrapper.unmount()
  })
  it('sends with cooldown and logs in without a password', async () => {
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('[data-testid="email-code-tab"]').trigger('click')
    expect(wrapper.find('#password').exists()).toBe(false)
    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('[data-testid="send-login-code"]').trigger('click')
    await flushPromises()
    expect(sendCodeMock).toHaveBeenCalledWith(expect.objectContaining({ email: 'user@example.com' }))
    expect(wrapper.get('[data-testid="send-login-code"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('auth.emailCodeSent')
    await wrapper.get('#login-code').setValue('123456')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(loginCodeMock).toHaveBeenCalledWith(expect.objectContaining({ email: 'user@example.com', code: '123456' }))
    expect(loginMock).not.toHaveBeenCalled()
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
    wrapper.unmount()
  })
  it('blocks actions until login terms are accepted', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, email_code_login_enabled: true,
      login_agreement_enabled: true, login_agreement_mode: 'checkbox',
      login_agreement_revision: 'email-login-test', login_agreement_documents: [{ id: 'terms', title: 'Terms' }] })
    const wrapper = mountLogin()
    await flushPromises()
    expect(wrapper.get('[data-testid="email-code-tab"]').attributes('disabled')).toBeDefined()
    expect(sendCodeMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
