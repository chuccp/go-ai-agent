import { useTranslation } from 'react-i18next'
import { useState, useRef, useEffect } from 'react'

const LANGUAGES = [
  { code: 'zh', label: '简体中文' },
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'en', label: 'English' },
  { code: 'ja', label: '日本語' },
]

export default function LanguageSwitcher() {
  const { i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const current = LANGUAGES.find(l => l.code === i18n.language)?.label || '中文'

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen(!open)}
        style={{
          background: 'none', border: '0.5px solid rgba(16,24,40,0.08)', borderRadius: 6,
          padding: '4px 8px', cursor: 'pointer', fontSize: 12, color: '#676f83',
        }}
      >
        {current}
      </button>
      {open && (
        <div style={{
          position: 'absolute', right: 0, top: '100%', marginTop: 4, zIndex: 100,
          background: '#fff', borderRadius: 8, border: '0.5px solid rgba(16,24,40,0.08)',
          boxShadow: '0 4px 12px rgba(16,24,40,0.08)', minWidth: 120, overflow: 'hidden',
        }}>
          {LANGUAGES.map(lang => (
            <button
              key={lang.code}
              onClick={() => { i18n.changeLanguage(lang.code); setOpen(false) }}
              style={{
                display: 'block', width: '100%', padding: '8px 12px', border: 'none',
                background: lang.code === i18n.language ? 'rgba(21,90,239,0.08)' : 'transparent',
                color: lang.code === i18n.language ? '#155aef' : '#354052',
                fontSize: 13, cursor: 'pointer', textAlign: 'left',
              }}
            >
              {lang.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
