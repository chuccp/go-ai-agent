import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useSetupStore } from '@/stores/setupStore'
import ModelForm, { emptyModelForm, ModelFormData } from '@/components/ModelForm'

export default function SetupWizard() {
  const { t } = useTranslation()
  const nav = useNavigate()
  const store = useSetupStore()
  const [step, setStep] = useState(0)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const [adminExists, setAdminExists] = useState(false)
  const [isDesktop, setIsDesktop] = useState(false)
  const [dbConfigured, setDbConfigured] = useState(false)
  const [adminConfigured, setAdminConfigured] = useState(false)

  // DB form
  const [dbType, setDbType] = useState('sqlite')
  const [dbForm, setDbForm] = useState({ host: '', port: '3306', name: '', user: '', password: '', charset: 'utf8mb4', ssl_mode: 'disable', file_path: './data/go-ai-agent.db' })

  // Admin form
  const [adminForm, setAdminForm] = useState({ username: 'admin', password: '', confirm: '' })

  // Model form
  const [modelForm, setModelForm] = useState<ModelFormData>(emptyModelForm())

  useEffect(() => {
    store.getSetupStatus().then((s: any) => {
      setDbConfigured(s.db_configured)
      setAdminConfigured(s.admin_configured)
      setIsDesktop(s.mode === 'desktop')
      if (s.admin_configured) setAdminExists(true)
      // Desktop mode: skip DB and admin steps entirely
      if (s.db_configured && s.admin_configured) setStep(2)
      else if (s.db_configured) setStep(1)
    })
  }, [])

  const handleTest = async () => {
    setTesting(true); setError('')
    const r = await store.testConnection({ type: dbType, ...dbForm })
    if (!r.ok) setError(r.msg || t('common.error'))
    setTesting(false)
  }

  const handleDbNext = async () => {
    setSaving(true); setError('')
    const r = await store.initDatabase({ type: dbType, ...dbForm })
    if (r.ok) setStep(1)
    else setError(r.msg || t('common.error'))
    setSaving(false)
  }

  const handleAdminNext = async () => {
    if (!adminForm.username || !adminForm.password) { setError(t('setup.validation.userPassRequired')); return }
    if (adminForm.password.length < 6) { setError(t('setup.validation.passTooShort')); return }
    if (adminForm.password !== adminForm.confirm) { setError(t('setup.validation.passMismatch')); return }
    setSaving(true); setError('')
    const r = await store.initAdmin({ username: adminForm.username, password: adminForm.password })
    if (r.ok) setStep(2)
    else setError(r.msg || t('common.error'))
    setSaving(false)
  }

  const handleModelComplete = async () => {
    if (!modelForm.model_id) { setError(t('setup.validation.modelIdRequired')); return }
    if (!modelForm.api_key) { setError(t('setup.validation.apiKeyRequired')); return }
    if (!modelForm.base_url) { setError(t('setup.validation.baseUrlRequired')); return }
    setSaving(true); setError('')
    const r1 = await store.initBaseModel(modelForm)
    if (!r1.ok) { setError(r1.msg || t('common.error')); setSaving(false); return }
    const r2 = await store.completeSetup()
    if (r2.ok) { setDone(true); setTimeout(() => nav('/'), 1500) }
    else setError(r2.msg || t('common.error'))
    setSaving(false)
  }

  if (done) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: 12 }}>
        <div style={{ fontSize: 32 }}>✅</div>
        <div style={{ fontSize: 18, fontWeight: 600 }}>{t('setup.setupComplete')}</div>
        <div style={{ fontSize: 14, color: '#676f83' }}>{t('setup.redirecting')}</div>
      </div>
    )
  }

  const card: React.CSSProperties = {
    background: '#fcfcfd', borderRadius: 16, border: '0.5px solid rgba(16,24,40,0.08)',
    padding: 24, width: '100%', maxWidth: 480, boxShadow: '0 4px 12px rgba(16,24,40,0.06)',
  }
  const inputStyle: React.CSSProperties = { padding: '8px 10px', border: '1px solid #d0d5dd', borderRadius: 8, fontSize: 13, outline: 'none', width: '100%' }
  const labelStyle: React.CSSProperties = { fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }
  const btnStyle = (primary = false): React.CSSProperties => ({
    padding: '8px 20px', borderRadius: 8, border: primary ? 'none' : '1px solid #d0d5dd',
    background: primary ? '#155aef' : '#fcfcfd', color: primary ? '#fff' : '#354052',
    fontSize: 13, fontWeight: 500, cursor: 'pointer',
  })
  const stepDot = (active: boolean): React.CSSProperties => ({
    width: 28, height: 28, borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center',
    fontSize: 12, fontWeight: 600,
    background: active ? '#155aef' : '#e2e8f0', color: active ? '#fff' : '#676f83',
  })
  const stepLabel: React.CSSProperties = { fontSize: 13, fontWeight: 500, color: '#354052' }
  const stepDone: React.CSSProperties = { ...stepLabel, color: '#039855', textDecoration: 'line-through' }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', padding: 20, background: '#f2f4f7' }}>
      <div style={{ marginBottom: 24, textAlign: 'center' }}>
        <div style={{ fontSize: 24, fontWeight: 700 }}>⚡ {t('setup.title')}</div>
        <div style={{ fontSize: 14, color: '#676f83', marginTop: 4 }}>{t('setup.subtitle')}</div>
      </div>

      {/* Steps indicator */}
      <div style={{ display: 'flex', gap: 32, marginBottom: 24, alignItems: 'center' }}>
        {['database', 'admin', 'model'].map((s, i) => {
          const done = (i === 0 && dbConfigured) || (i === 1 && adminConfigured)
          const active = i === step
          return (
            <div key={s} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={stepDot(active)}>{done ? '✓' : i + 1}</div>
              <span style={done ? stepDone : stepLabel}>{t(`setup.steps.${s}`)}</span>
              {i < 2 && <span style={{ color: '#d0d5dd' }}>—</span>}
            </div>
          )
        })}
      </div>

      <div style={card}>
        {error && <div style={{ background: '#fef3f2', color: '#d92d20', padding: '8px 12px', borderRadius: 8, fontSize: 13, marginBottom: 12 }}>{error}</div>}

        {/* Step 0: Database */}
        {step === 0 && !isDesktop && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div><label style={labelStyle}>{t('setup.db.dbType')}</label>
              <select value={dbType} onChange={e => setDbType(e.target.value)} style={inputStyle}>
                <option value="sqlite">{t('setup.db.sqlite')}</option>
                <option value="mysql">MySQL</option>
                <option value="postgres">PostgreSQL</option>
              </select>
            </div>
            {dbType === 'sqlite' ? (
              <div><label style={labelStyle}>{t('setup.db.filePath')}</label>
                <input value={dbForm.file_path} onChange={e => setDbForm(f => ({ ...f, file_path: e.target.value }))} style={inputStyle} />
              </div>
            ) : (
              <>
                <div style={{ display: 'flex', gap: 8 }}>
                  <div style={{ flex: 3 }}><label style={labelStyle}>{t('setup.db.host')}</label>
                    <input value={dbForm.host} onChange={e => setDbForm(f => ({ ...f, host: e.target.value }))} style={inputStyle} placeholder="localhost" />
                  </div>
                  <div style={{ flex: 1 }}><label style={labelStyle}>{t('setup.db.port')}</label>
                    <input value={dbForm.port} onChange={e => setDbForm(f => ({ ...f, port: e.target.value }))} style={inputStyle} />
                  </div>
                </div>
                <div><label style={labelStyle}>{t('setup.db.dbName')}</label>
                  <input value={dbForm.name} onChange={e => setDbForm(f => ({ ...f, name: e.target.value }))} style={inputStyle} />
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <div style={{ flex: 1 }}><label style={labelStyle}>{t('setup.db.username')}</label>
                    <input value={dbForm.user} onChange={e => setDbForm(f => ({ ...f, user: e.target.value }))} style={inputStyle} />
                  </div>
                  <div style={{ flex: 1 }}><label style={labelStyle}>{t('setup.db.password')}</label>
                    <input type="password" value={dbForm.password} onChange={e => setDbForm(f => ({ ...f, password: e.target.value }))} style={inputStyle} />
                  </div>
                </div>
              </>
            )}
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 8 }}>
              {dbType !== 'sqlite' && <button onClick={handleTest} disabled={testing} style={btnStyle()}>{testing ? t('setup.db.testing') : t('setup.db.testConnection')}</button>}
              <button onClick={handleDbNext} disabled={saving} style={btnStyle(true)}>{saving ? t('setup.initializing') : t('common.next')}</button>
            </div>
          </div>
        )}

        {/* Step 1: Admin */}
        {step === 1 && !isDesktop && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div style={{ fontSize: 14, fontWeight: 600 }}>{t('setup.admin.createAdmin')}</div>
            <div style={{ fontSize: 13, color: '#676f83' }}>{adminExists ? t('setup.admin.existsHint') : t('setup.admin.adminHint')}</div>
            <div><label style={labelStyle}>{t('setup.db.username')}</label>
              <input value={adminForm.username} onChange={e => setAdminForm(f => ({ ...f, username: e.target.value }))} style={inputStyle} />
            </div>
            <div><label style={labelStyle}>{t('setup.db.password')} ({t('setup.admin.minLength')})</label>
              <input type="password" value={adminForm.password} onChange={e => setAdminForm(f => ({ ...f, password: e.target.value }))} style={inputStyle} />
            </div>
            <div><label style={labelStyle}>{t('setup.admin.confirmPassword')}</label>
              <input type="password" value={adminForm.confirm} onChange={e => setAdminForm(f => ({ ...f, confirm: e.target.value }))} style={inputStyle} />
            </div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 8 }}>
              <button onClick={() => setStep(0)} style={btnStyle()}>{t('common.prev')}</button>
              <button onClick={handleAdminNext} disabled={saving} style={btnStyle(true)}>{saving ? t('setup.initializing') : t('common.next')}</button>
            </div>
          </div>
        )}

        {/* Step 2: Model — shared with ModelManager */}
        {step === 2 && (
          <div>
            <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>{t('setup.model.configBase')}</div>
            {isDesktop && (
              <div style={{ fontSize: 12, color: '#155aef', background: '#e8f0fe', padding: '8px 12px', borderRadius: 8, marginBottom: 14 }}>
                SQLite + {t('setup.admin.createAdmin')} {t('common.success')} — {t('setup.model.configBase')} {t('setup.subtitle')}
              </div>
            )}
            <ModelForm form={modelForm} onChange={setModelForm} compact />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 16 }}>
              {!isDesktop && <button onClick={() => setStep(1)} style={btnStyle()}>{t('common.prev')}</button>}
              <button onClick={() => { store.completeSetup().then(() => nav('/')) }} style={btnStyle()}>{t('setup.skip')}</button>
              <button onClick={handleModelComplete} disabled={saving} style={btnStyle(true)}>
                {saving ? t('setup.initializing') : t('setup.completeSetup')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
