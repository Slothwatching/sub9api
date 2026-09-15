import { describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'

const state = reactive({
  cachedPublicSettings: { subscription_enabled: true, payment_balance_disabled: false },
})
vi.mock('@/stores', () => ({ useAppStore: () => state, useAuthStore: () => ({ isAdmin: false }) }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({ customMenuItems: [] }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ name: 'PurchaseSubscription', meta: { titleKey: 'nav.buySubscription', descriptionKey: 'purchase.description' } }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
import { usePageHeading } from '../usePageHeading'

describe('workspace billing headings', () => {
  it('updates the visible workspace heading when the administrator changes billing mode', () => {
    const { pageTitle, pageDescription } = usePageHeading()
    expect(pageTitle.value).toBe('nav.buySubscription')
    state.cachedPublicSettings.subscription_enabled = false
    expect(pageTitle.value).toBe('nav.recharge')
    expect(pageDescription.value).toBe('purchase.rechargeDescription')
    state.cachedPublicSettings.subscription_enabled = true
    state.cachedPublicSettings.payment_balance_disabled = true
    expect(pageTitle.value).toBe('nav.subscribe')
    expect(pageDescription.value).toBe('purchase.subscriptionDescription')
  })
})
