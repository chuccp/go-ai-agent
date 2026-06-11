// ── Thinking levels ──
export const THINK_LEVELS = [
  { value: 'off', label: '关' },
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'max', label: '最高' },
] as const

export function thinkLabel(v: string): string {
  return THINK_LEVELS.find(t => t.value === v)?.label || '关'
}

// ── File helpers ──
export function fileIcon(mime: string): string {
  if (mime.startsWith('image/')) return '🖼'
  if (mime.startsWith('text/')) return '📄'
  if (mime.includes('pdf')) return '📕'
  if (mime.includes('doc')) return '📝'
  return '📎'
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + 'KB'
  return (bytes / 1048576).toFixed(1) + 'MB'
}

// ── API bases ──
export const API_BASE = ''

// ── Flow designer ──
export const GRID_SIZE = 20
export const NODE_WIDTH = 170
export const NODE_SPACING_X = 220
export const NODE_SPACING_Y = 140
export const NODE_COLS = 3
