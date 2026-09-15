import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
const mocks = vi.hoisted(() => ({ list: vi.fn(), error: vi.fn() }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.error }) }))
vi.mock('@/api/channelMonitor', () => ({ list: mocks.list, status: vi.fn() }))
vi.mock('@/composables/useAutoRefresh', () => ({ useAutoRefresh: () => ({ countdown: { value: 60 }, setEnabled: vi.fn(), resetCountdown: vi.fn() }) }))
import View from '../ChannelStatusV1View.vue'
const global = { stubs: { AppLayout: { template: '<main><slot /></main>' }, MonitorHero: { props: ['overallStatus'], template: '<button data-testid="status" @click="$emit(\'refresh\')">{{ overallStatus }}</button>' }, MonitorCardGrid: { props: ['items'], template: '<div data-testid="rows">{{ items.length }}</div>' }, MonitorDetailDialog: true } }
beforeEach(() => vi.clearAllMocks())
describe('monitor evidence states', () => {
 it('does not claim operational status for an empty list', async () => { mocks.list.mockResolvedValue({ items: [] }); const w = mount(View, { global }); expect(w.get('[data-testid="status"]').text()).toBe('loading'); await flushPromises(); expect(w.get('[data-testid="status"]').text()).toBe('empty'); w.unmount() })
 it('shows refresh failure while retaining the last successful data', async () => { mocks.list.mockResolvedValueOnce({ items: [{ id: 1, primary_status: 'operational', timeline: [] }] }).mockRejectedValueOnce(new Error('offline')); const w = mount(View, { global }); await flushPromises(); expect(w.get('[data-testid="status"]').text()).toBe('operational'); await w.get('[data-testid="status"]').trigger('click'); await flushPromises(); expect(w.get('[data-testid="status"]').text()).toBe('error'); expect(w.get('[data-testid="rows"]').text()).toBe('1'); w.unmount() })
})
