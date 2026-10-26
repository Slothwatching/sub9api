// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import html from './guide.html?raw'
import script from './guide.js?raw'
import review from '../../../public/guides/thesis-example/data/review.json'

async function start(options: { failure?: boolean; clipboardFailure?: boolean } = {}) {
  document.open()
  document.write(html.replace(/<script[\s\S]*?<\/script>/g, ''))
  document.close()
  vi.stubGlobal('matchMedia', () => ({ matches: true }))
  vi.stubGlobal('IntersectionObserver', class { observe() {} unobserve() {} disconnect() {} })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(new Proxy({}, { get: () => () => {} }) as never)
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('fetch', options.failure
    ? vi.fn().mockRejectedValue(new Error('offline'))
    : vi.fn().mockResolvedValue({ ok: true, json: async () => structuredClone(review) }))
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } })
  Object.defineProperty(document, 'execCommand', { configurable: true, value: vi.fn(() => !options.clipboardFailure) })
  window.eval(script)
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
}

beforeEach(() => { vi.useFakeTimers(); localStorage.clear() })
afterEach(() => { vi.clearAllTimers(); vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals() })

describe('D108 worked thesis walkthrough', () => {
  it('renders all five real records, fixed review date and unknown counts without invented bars', async () => {
    await start()
    expect(document.querySelectorAll('.paper')).toHaveLength(5)
    for (const paper of review.papers) expect(document.querySelector('#papers')?.textContent).toContain(paper.doi)
    expect(document.querySelector('#bDate')?.textContent).toBe('2026-10-26')
    expect(document.querySelector('#legend')?.textContent).toContain('免费全文可获取 2')
    expect(document.querySelector('#legend')?.textContent).toContain('订阅获取 2')
    expect(document.querySelectorAll('.paper .bars')).toHaveLength(0)
    expect(document.querySelector('#papers')?.textContent).toContain('支持 未获取')
    expect(document.querySelector('#papers')?.textContent).toContain('LGF21H020006')
    expect(document.querySelector('#papers')?.textContent).not.toContain('10.xxxx')
  })

  it('filters to exactly the three recorded free texts and restores all five with keyboard', async () => {
    await start()
    const toggle = document.querySelector<HTMLElement>('#accToggle')!
    toggle.click()
    expect(document.querySelectorAll('.paper:not([hidden])')).toHaveLength(3)
    expect(document.querySelector('#boardCount')?.textContent).toContain('3 / 5')
    expect(document.querySelector('#legend')?.textContent).toContain('订阅获取 2')
    toggle.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true }))
    expect(document.querySelectorAll('.paper:not([hidden])')).toHaveLength(5)
    expect(toggle.getAttribute('aria-checked')).toBe('false')
  })

  it('keeps the nine prompt tabs and twelve stages, with consistent schema and plugin fallbacks', async () => {
    await start()
    const tabs = [...document.querySelectorAll<HTMLButtonElement>('#genTabs button')]
    expect(tabs).toHaveLength(9)
    expect(document.querySelectorAll('#levels button')).toHaveLength(12)
    document.querySelectorAll<HTMLButtonElement>('#genTabs button')[0]!.click()
    expect(document.querySelector('#genPre')?.textContent).toContain('没有任何联网工具')
    document.querySelectorAll<HTMLButtonElement>('#genTabs button')[1]!.click()
    expect(document.querySelector('#genPre')?.textContent).toContain('"papers":[]')
    document.querySelectorAll<HTMLButtonElement>('#genTabs button')[4]!.click()
    expect(document.querySelector('#genPre')?.textContent).toContain('"papers":[]')
    document.querySelectorAll<HTMLButtonElement>('#genTabs button')[3]!.click()
    expect(document.querySelector('#genPre')?.textContent).toContain('GenAI 使用权')
    expect(document.querySelector('#genPre')?.textContent).toContain('未核实')
    expect(html).toContain('/guides/thesis-example.zip')
  })

  it('reports clipboard failure rather than claiming the prompt was copied', async () => {
    await start({ clipboardFailure: true })
    document.querySelector<HTMLButtonElement>('#genCopy')!.click()
    await Promise.resolve()
    await Promise.resolve()
    expect(document.querySelector('#toast')?.textContent).toContain('复制未成功')
    expect(document.querySelector('#genCopy')?.textContent).toBe('复制')
  })

  it('shows a recovery message rather than sample evidence when data cannot load', async () => {
    await start({ failure: true })
    expect(document.querySelectorAll('.paper')).toHaveLength(0)
    expect(document.querySelector('#papers')?.textContent).toContain('下载完整项目')
    expect(document.querySelector('#boardCount')?.textContent).toContain('数据未加载')
    expect(document.querySelector('#bDate')?.textContent).toBe('未获取')
  })
})
