// In Wails dev mode, the webview is served via wails:// but API calls go through
// the Wails dev server (wails.localhost:34115) which does NOT proxy /api to the
// Go backend. We detect this and point to the Vite dev server (which has /api
// proxy configured) instead.
export const API_BASE = (() => {
  if (typeof window !== 'undefined' && window.location.hostname === 'wails.localhost') {
    return 'http://localhost:5173'
  }
  return ''
})()

export const THINK_LEVELS = ['off', 'low', 'medium', 'high', 'max'] as const
export type ThinkLevel = typeof THINK_LEVELS[number]

export function thinkLabelKey(level: string): string {
  return `think.${level}`
}

export function fileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext)) return '🖼'
  if (['mp3', 'wav', 'ogg', 'flac'].includes(ext)) return '🎵'
  if (['mp4', 'avi', 'mov', 'mkv'].includes(ext)) return '🎬'
  if (['pdf'].includes(ext)) return '📄'
  if (['doc', 'docx'].includes(ext)) return '📝'
  if (['xls', 'xlsx'].includes(ext)) return '📊'
  if (['zip', 'rar', '7z', 'tar'].includes(ext)) return '📦'
  return '📎'
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
