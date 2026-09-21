/**
 * Server-injected public settings.
 *
 * The backend embeds the public settings into `index.html` so the first paint
 * already carries the administrator's branding (see `backend/internal/web/embed_on.go`).
 * The payload rides in a **non-executable** `<script type="application/json">`
 * block rather than an inline script: `index.html` is served with an ETag, so a
 * browser replays its cached body after a `304` while the response carries a
 * freshly generated CSP nonce. An inline script would then be blocked by CSP
 * ("Executing inline script violates the following Content Security Policy
 * directive"), the SPA would fall back to the async settings API, and the
 * default site name would flash before the configured one on slow connections.
 *
 * Data blocks are never executed, so CSP leaves them alone and the HTML stays
 * cacheable.
 */

import type { PublicSettings } from '@/types'

/**
 * Matches `AppConfigElementID` in `backend/internal/web/embed_on.go`.
 *
 * Never name it `__APP_CONFIG__`: DOM named access would expose the `<script>`
 * element itself as `window.__APP_CONFIG__` and shadow the parsed settings.
 */
export const APP_CONFIG_ELEMENT_ID = 'app-config'

/** Guards against DOM named access leaking an element into `window.__APP_CONFIG__`. */
function isSettingsObject(value: unknown): value is PublicSettings {
  return (
    typeof value === 'object' &&
    value !== null &&
    !(typeof Node !== 'undefined' && value instanceof Node)
  )
}

/**
 * Parse the injected JSON data block, if the page carries one.
 * Returns null in dev (Vite serves the raw template) or when parsing fails.
 */
export function readInjectedConfig(): PublicSettings | null {
  if (typeof document === 'undefined') return null

  const raw = document.getElementById(APP_CONFIG_ELEMENT_ID)?.textContent?.trim()
  if (!raw) return null

  try {
    return JSON.parse(raw) as PublicSettings
  } catch (error) {
    console.error('Failed to parse injected app config:', error)
    return null
  }
}

/**
 * Publish the injected settings on `window.__APP_CONFIG__` before anything reads
 * them (the app store, feature flags, table preferences). Idempotent: an already
 * populated `window.__APP_CONFIG__` always wins.
 */
export function hydrateInjectedConfig(): PublicSettings | null {
  if (typeof window === 'undefined') return null
  if (isSettingsObject(window.__APP_CONFIG__)) return window.__APP_CONFIG__

  const config = readInjectedConfig()
  if (config) {
    window.__APP_CONFIG__ = config
  }
  return config
}
