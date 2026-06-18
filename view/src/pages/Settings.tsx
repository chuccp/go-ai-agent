import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import i18n from '@/i18n'
import { useSetupStore } from '@/stores/setupStore'

export default function Settings() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { clearDatabase, clearAllData, checkSetup } = useSetupStore()
  const [confirmType, setConfirmType] = useState<null | 'db' | 'all'>(null)
  const [clearing, setClearing] = useState(false)
  const [message, setMessage] = useState('')

  const handleClear = async () => {
    setClearing(true)
    setMessage('')
    try {
      const ok = confirmType === 'db' ? await clearDatabase() : await clearAllData()
      if (ok) {
        setMessage(t('settings.clearSuccess'))
        // Update the store so App.tsx re-renders and redirects to /setup
        await checkSetup()
        // Navigate directly to setup wizard
        navigate('/setup', { replace: true })
        // Force a full reload as a fallback to ensure clean state
        setTimeout(() => window.location.reload(), 500)
      } else {
        setMessage(t('common.error'))
        setClearing(false)
      }
    } catch {
      setMessage(t('common.error'))
      setClearing(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', background: '#f8f9fa' }}>
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '16px 24px',
        background: '#fff', borderBottom: '1px solid #e8eaed',
      }}>
        <button
          onClick={() => navigate('/')}
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            width: 36, height: 36, borderRadius: 18, border: '1px solid #dadce0',
            background: '#fff', cursor: 'pointer', color: '#3c4043',
          }}
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M19 12H5M12 19l-7-7 7-7" />
          </svg>
        </button>
        <div style={{ fontSize: 18, fontWeight: 600, color: '#202124' }}>{t('common.settings')}</div>
      </div>

      <div style={{ maxWidth: 640, margin: '0 auto', padding: '24px 16px' }}>
        {/* General Section */}
        <div style={{
          background: '#fff', borderRadius: 16, marginBottom: 24, overflow: 'hidden',
          border: '1px solid #e8eaed',
        }}>
          <div style={{
            padding: '16px 20px', borderBottom: '1px solid #f1f3f4',
            fontSize: 12, fontWeight: 600, color: '#5f6368', textTransform: 'uppercase', letterSpacing: 0.5,
          }}>
            {t('settings.general')}
          </div>
          <div style={{ padding: '16px 20px' }}>
            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            }}>
              <div>
                <div style={{ fontSize: 14, fontWeight: 500, color: '#202124' }}>{t('settings.language')}</div>
                <div style={{ fontSize: 12, color: '#5f6368', marginTop: 2 }}>{t('settings.languageDesc')}</div>
              </div>
              <select
                value={i18n.language}
                onChange={e => i18n.changeLanguage(e.target.value)}
                style={{
                  padding: '8px 12px', borderRadius: 12, border: '1px solid #dadce0',
                  fontSize: 14, outline: 'none', background: '#fff', color: '#202124',
                  cursor: 'pointer', minWidth: 140,
                }}
              >
                <option value="en">English</option>
                <option value="zh">简体中文</option>
                <option value="zh-TW">繁體中文</option>
                <option value="ja">日本語</option>
              </select>
            </div>
          </div>
        </div>

        {/* Danger Zone */}
        <div style={{
          background: '#fff', borderRadius: 16, overflow: 'hidden',
          border: '1px solid #fdc7c4',
        }}>
          <div style={{
            padding: '16px 20px', borderBottom: '1px solid #f1f3f4',
            fontSize: 12, fontWeight: 600, color: '#d93025', textTransform: 'uppercase', letterSpacing: 0.5,
          }}>
            {t('settings.dangerZone')}
          </div>
          <div style={{ padding: '8px 20px' }}>
            {/* Clear Database */}
            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '16px 0', borderBottom: '1px solid #f1f3f4',
            }}>
              <div style={{ flex: 1, paddingRight: 16 }}>
                <div style={{ fontSize: 14, fontWeight: 500, color: '#202124' }}>{t('settings.clearDatabase')}</div>
                <div style={{ fontSize: 12, color: '#5f6368', marginTop: 4 }}>{t('settings.clearDbDesc')}</div>
              </div>
              <button
                onClick={() => setConfirmType('db')}
                disabled={clearing}
                style={{
                  padding: '8px 20px', borderRadius: 20, border: '1px solid #d93025',
                  background: '#fff', color: '#d93025', fontSize: 13, fontWeight: 500,
                  cursor: clearing ? 'not-allowed' : 'pointer', whiteSpace: 'nowrap',
                  opacity: clearing ? 0.6 : 1,
                }}
              >
                {t('settings.clearDatabase')}
              </button>
            </div>

            {/* Clear All Data */}
            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '16px 0',
            }}>
              <div style={{ flex: 1, paddingRight: 16 }}>
                <div style={{ fontSize: 14, fontWeight: 500, color: '#202124' }}>{t('settings.clearAllData')}</div>
                <div style={{ fontSize: 12, color: '#5f6368', marginTop: 4 }}>{t('settings.clearAllDesc')}</div>
              </div>
              <button
                onClick={() => setConfirmType('all')}
                disabled={clearing}
                style={{
                  padding: '8px 20px', borderRadius: 20, border: 'none',
                  background: '#d93025', color: '#fff', fontSize: 13, fontWeight: 500,
                  cursor: clearing ? 'not-allowed' : 'pointer', whiteSpace: 'nowrap',
                  opacity: clearing ? 0.6 : 1,
                }}
              >
                {t('settings.clearAllData')}
              </button>
            </div>
          </div>
        </div>

        {/* Status message */}
        {message && (
          <div style={{
            marginTop: 16, padding: '12px 20px', borderRadius: 12,
            background: message === t('common.error') ? '#fce8e6' : '#e6f4ea',
            color: message === t('common.error') ? '#d93025' : '#137333',
            fontSize: 14, textAlign: 'center',
          }}>
            {message}
          </div>
        )}
      </div>

      {/* Confirmation Dialog */}
      {confirmType && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
        }} onClick={() => !clearing && setConfirmType(null)}>
          <div style={{
            background: '#fff', borderRadius: 20, padding: 24, maxWidth: 420,
            boxShadow: '0 8px 32px rgba(0,0,0,0.12)',
          }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 16, fontWeight: 600, color: '#202124', marginBottom: 12 }}>
              {confirmType === 'db' ? t('settings.clearDatabase') : t('settings.clearAllData')}
            </div>
            <div style={{ fontSize: 14, color: '#5f6368', marginBottom: 20, lineHeight: 1.5 }}>
              {confirmType === 'db' ? t('settings.clearDbConfirm') : t('settings.clearAllConfirm')}
            </div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button
                onClick={() => setConfirmType(null)}
                disabled={clearing}
                style={{
                  padding: '10px 24px', borderRadius: 20, border: '1px solid #dadce0',
                  background: '#fff', color: '#3c4043', fontSize: 14, fontWeight: 500,
                  cursor: clearing ? 'not-allowed' : 'pointer',
                }}
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleClear}
                disabled={clearing}
                style={{
                  padding: '10px 24px', borderRadius: 20, border: 'none',
                  background: '#d93025', color: '#fff', fontSize: 14, fontWeight: 500,
                  cursor: clearing ? 'not-allowed' : 'pointer',
                }}
              >
                {clearing ? t('common.loading') : t('common.confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
