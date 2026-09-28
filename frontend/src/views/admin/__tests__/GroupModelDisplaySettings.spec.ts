import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import GroupModelDisplaySettings from '../GroupModelDisplaySettings.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('independent model display settings', () => {
  it('keeps an enabled empty legacy list empty after candidate refresh', async () => {
    const wrapper = mount(GroupModelDisplaySettings, { props: { modelValue: { enabled: true, models: [] }, candidates: ['gpt-5.5'], loading: false } })
    expect((wrapper.get('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    await wrapper.setProps({ candidates: ['gpt-5.5', 'gpt-5.4'] })
    expect(wrapper.findAll('input[type="checkbox"]').every(input => !(input.element as HTMLInputElement).checked)).toBe(true)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('round trips ordering, selection and toggle without emitting an admission policy', async () => {
    const wrapper = mount(GroupModelDisplaySettings, { props: { modelValue: { enabled: true, models: ['gpt-5.5', 'gpt-5.4'] }, candidates: ['gpt-5.4', 'gpt-5.5'], loading: false } })
    await wrapper.findAll('button[title="admin.groups.modelsList.moveDown"]')[0].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{ enabled: true, models: ['gpt-5.4', 'gpt-5.5'] }])
    await wrapper.get('input[type="checkbox"]').setValue(false)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{ enabled: true, models: ['gpt-5.5'] }])
    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{ enabled: false, models: ['gpt-5.5'] }])
  })

  it('accepts wildcards anywhere in custom entries and reports duplicates', async () => {
    const wrapper = mount(GroupModelDisplaySettings, { props: { modelValue: { enabled: true, models: ['gpt-5.5'] }, candidates: ['gpt-5.5'], loading: false } })
    const entry = wrapper.get('input[aria-label="admin.groups.modelsList.modelId"]')
    const add = wrapper.findAll('button').find(button => button.text() === 'admin.groups.modelAllowlist.addCustom')!
    await entry.setValue('gpt-*-codex')
    await add.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{ enabled: true, models: ['gpt-5.5', 'gpt-*-codex'] }])
    expect(wrapper.find('p.text-red-500').exists()).toBe(false)
    await entry.setValue('GPT-5.5')
    await add.trigger('click')
    expect(wrapper.get('p.text-red-500').text()).toBe('admin.groups.modelAllowlist.errors.duplicate')
  })
})
