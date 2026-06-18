import { useState } from 'react'
import { useTranslation } from 'react-i18next'

interface Props {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  style?: React.CSSProperties
}

export default function PasswordInput({ value, onChange, placeholder, style }: Props) {
  const { t } = useTranslation()
  const [show, setShow] = useState(false)

  const wrapperStyle: React.CSSProperties = {
    position: 'relative',
    width: '100%',
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 36px 8px 10px',
    borderRadius: 8,
    fontSize: 13,
    border: '0.5px solid rgba(16,24,40,0.15)',
    outline: 'none',
    background: '#fcfcfd',
    color: '#101828',
    ...style,
    paddingRight: 36,
  }

  const toggleStyle: React.CSSProperties = {
    position: 'absolute',
    right: 8,
    top: '50%',
    transform: 'translateY(-50%)',
    background: 'none',
    border: 'none',
    cursor: 'pointer',
    fontSize: 13,
    color: '#676f83',
    padding: '2px 4px',
  }

  return (
    <div style={wrapperStyle}>
      <input
        type={show ? 'text' : 'password'}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        style={inputStyle}
      />
      <button
        type="button"
        onClick={() => setShow(s => !s)}
        style={toggleStyle}
        tabIndex={-1}
        title={show ? t('common.hidePassword') : t('common.showPassword')}
      >
        {show ? (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
            <line x1="1" y1="1" x2="23" y2="23" />
          </svg>
        ) : (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
            <circle cx="12" cy="12" r="3" />
          </svg>
        )}
      </button>
    </div>
  )
}
