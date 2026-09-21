import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  APP_CONFIG_ELEMENT_ID,
  hydrateInjectedConfig,
  readInjectedConfig
} from '@/utils/injectedConfig'

function injectBlock(content: string): void {
  const el = document.createElement('script')
  el.type = 'application/json'
  el.id = APP_CONFIG_ELEMENT_ID
  el.textContent = content
  document.head.appendChild(el)
}

describe('injectedConfig', () => {
  beforeEach(() => {
    document.getElementById(APP_CONFIG_ELEMENT_ID)?.remove()
    delete (window as any).__APP_CONFIG__
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('页面没有注入数据块时返回 null', () => {
    expect(readInjectedConfig()).toBeNull()
    expect(hydrateInjectedConfig()).toBeNull()
    expect((window as any).__APP_CONFIG__).toBeUndefined()
  })

  it('解析注入的 JSON 数据块并挂到 window 上', () => {
    injectBlock('{"site_name":"Sloth Code","site_logo":"/logo.svg"}')

    const config = hydrateInjectedConfig()

    expect(config?.site_name).toBe('Sloth Code')
    expect((window as any).__APP_CONFIG__.site_name).toBe('Sloth Code')
  })

  it('已有 window.__APP_CONFIG__ 时不覆盖', () => {
    const win = window as any
    win.__APP_CONFIG__ = { site_name: 'Existing' }
    injectBlock('{"site_name":"Injected"}')

    expect(hydrateInjectedConfig()?.site_name).toBe('Existing')
    expect((window as any).__APP_CONFIG__.site_name).toBe('Existing')
  })

  it('忽略 DOM 命名访问泄漏到 window 上的元素', () => {
    // id 一旦叫 __APP_CONFIG__，window.__APP_CONFIG__ 会指向 <script> 元素本身。
    const leaked = document.createElement('script')
    ;(window as unknown as Record<string, unknown>).__APP_CONFIG__ = leaked
    injectBlock('{"site_name":"Sloth Code"}')

    expect(hydrateInjectedConfig()?.site_name).toBe('Sloth Code')
  })

  it('数据块内容损坏时降级为 null 而不抛错', () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    injectBlock('{not json')

    expect(hydrateInjectedConfig()).toBeNull()
    expect((window as any).__APP_CONFIG__).toBeUndefined()
    expect(errorSpy).toHaveBeenCalled()
  })
})
