import type { ConnectDownload, ConnectDownloadOS, ConnectResources, ConnectVideo, ConnectVideoOS } from '@/types'

export const connectVideoSystems: ConnectVideoOS[] = ['macos', 'windows', 'linux']
export const connectDownloadSystems: ConnectDownloadOS[] = ['macos', 'windows', 'linux', 'all']
export const connectSystemNames: Record<ConnectDownloadOS, string> = { macos: 'macOS', windows: 'Windows', linux: 'Linux', all: '' }

/** Mirrors the backend rule: absolute HTTPS without credentials. */
export function connectURL(value: string): string {
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && url.hostname && !url.username && !url.password ? url.href : ''
  } catch {
    return ''
  }
}

export function detectConnectOS(userAgent = navigator.userAgent): ConnectVideoOS {
  if (/Windows/i.test(userAgent)) return 'windows'
  if (/Linux/i.test(userAgent) && !/Android/i.test(userAgent)) return 'linux'
  return 'macos'
}

export function emptyConnectResources(): ConnectResources {
  return { videos: [], downloads: [] }
}

export function newConnectVideo(os: ConnectVideoOS): ConnectVideo {
  return { id: crypto.randomUUID(), os, title: '', title_en: '', url: '', poster_url: '', enabled: false }
}

export function newConnectDownload(os: ConnectDownloadOS): ConnectDownload {
  return { id: crypto.randomUUID(), os, name: '', name_en: '', version: '', note: '', note_en: '', url: '', sha256: '', enabled: false }
}

export function connectResourcesValidationError(resources: ConnectResources): string | null {
  for (const video of resources.videos) {
    if (video.enabled && !video.url.trim()) return 'commercial.tutorials.admin.videoRequired'
    if ((video.url && !connectURL(video.url)) || (video.poster_url && !connectURL(video.poster_url))) return 'commercial.tutorials.admin.linkError'
  }
  for (const download of resources.downloads) {
    if (download.enabled && (!download.name.trim() || !download.url.trim())) return 'commercial.tutorials.admin.downloadRequired'
    if (download.url && !connectURL(download.url)) return 'commercial.tutorials.admin.linkError'
    if (download.sha256 && !/^[0-9a-f]{64}$/.test(download.sha256)) return 'commercial.tutorials.admin.shaError'
  }
  return null
}
