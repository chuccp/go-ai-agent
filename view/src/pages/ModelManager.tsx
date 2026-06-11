import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useModelStore, AIModel } from '@/stores/modelStore'
import { THINK_LEVELS } from '@/constants'

const CATEGORIES = ['all', 'llm', 'image', 'voice', 'video'] as const
type Category = typeof CATEGORIES[number]

const API_TYPES = ['openai', 'claude', 'native'] as const

interface ModelForm {
  name: string
  provider: string
  model_id: string
  category: string
  api_key: string
  base_url: string
  description: string
  think_level: string
  multimodal: boolean
  is_default: boolean
  is_base: boolean
}

const emptyForm: ModelForm = {
  name: '',
  provider: '',
  model_id: '',
  category: 'llm',
  api_key: '',
  base_url: '',
  description: '',
  think_level: 'off',
  multimodal: false,
  is_default: false,
  is_base: false,
}

export default function ModelManager() {
  const { t } = useTranslation()
  const store = useModelStore()
  const [category, setCategory] = useState<Category>('all')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [form, setForm] = useState<ModelForm>({ ...emptyForm })
  const [apiType, setApiType] = useState<string>('openai')
  const [step, setStep] = useState(1)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    store.fetchModels()
    store.fetchProviderDefaults()
  }, [])

  const filteredModels = useMemo(() => {
    if (category === 'all') return store.models
    return store.models.filter(m => m.category === category)
  }, [store.models, category])

  const providerList = useMemo(() => {
    const defaults = store.providerDefaults[apiType]
    return defaults ? Object.keys(defaults) : []
  }, [store.providerDefaults, apiType])

  const maskKey = (key: string) => {
    if (!key) return ''
    if (key.length <= 8) return '****'
    return key.slice(0, 4) + '****' + key.slice(-4)
  }

  const openAdd = () => {
    setEditId(null)
    setForm({ ...emptyForm })
    setApiType('openai')
    setStep(1)
    setDialogOpen(true)
  }

  const openEdit = (m: AIModel) => {
    setEditId(m.id)
    setForm({
      name: m.name,
      provider: m.provider,
      model_id: m.model_id,
      category: m.category,
      api_key: m.api_key,
      base_url: m.base_url,
      description: m.description,
      think_level: m.think_level || 'off',
      multimodal: m.multimodal,
      is_default: m.is_default,
      is_base: m.is_base,
    })
    // Determine apiType from provider
    if (m.provider.startsWith('openai.')) setApiType('openai')
    else if (m.provider.startsWith('claude.')) setApiType('claude')
    else setApiType('native')
    setStep(1)
    setDialogOpen(true)
  }

  const fillProvider = (p: string) => {
    const d = store.providerDefaults[apiType]?.[p]
    if (d) {
      setForm(f => ({
        ...f,
        provider: p,
        name: f.name || d.model || '',
        model_id: f.model_id || d.model || '',
        base_url: f.base_url || d.baseUrl || '',
      }))
    } else {
      setForm(f => ({ ...f, provider: p }))
    }
  }

  const handleSave = async () => {
    if (!form.name.trim() || !form.model_id.trim()) return
    setSaving(true)
    const payload: Partial<AIModel> = { ...form }
    let ok: boolean
    if (editId) {
      ok = await store.updateModel(editId, payload)
    } else {
      ok = await store.createModel(payload)
    }
    if (ok) setDialogOpen(false)
    setSaving(false)
  }

  const handleDelete = async (m: AIModel) => {
    if (!window.confirm(t('model.confirmDelete', { name: m.name }))) return
    await store.deleteModel(m.id)
  }

  const handleSetDefault = async (id: number) => {
    await store.setDefault(id)
  }

  const handleSetBase = async (id: number) => {
    await store.setBase(id)
  }

  const apiTypeLabel = (key: string) => {
    if (key === 'openai') return t('model.openaiCompat')
    if (key === 'claude') return t('model.claudeCompat')
    return t('model.native')
  }

  // Styles
  const pageBg: React.CSSProperties = { background: '#f2f4f7', minHeight: '100vh', padding: '0' }
  const toolbar: React.CSSProperties = {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '16px 24px', background: '#fcfcfd',
    borderBottom: '0.5px solid rgba(16,24,40,0.08)',
  }
  const toolbarTitle: React.CSSProperties = { fontSize: 18, fontWeight: 700, color: '#101828' }
  const backLink: React.CSSProperties = {
    fontSize: 13, color: '#155aef', textDecoration: 'none', marginRight: 16,
  }
  const primaryBtn: React.CSSProperties = {
    padding: '8px 20px', borderRadius: 8, border: 'none',
    background: '#155aef', color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer',
  }
  const secondaryBtn: React.CSSProperties = {
    padding: '8px 20px', borderRadius: 8,
    border: '0.5px solid rgba(16,24,40,0.08)', background: '#fcfcfd', color: '#101828',
    fontSize: 13, fontWeight: 500, cursor: 'pointer',
  }
  const filterBar: React.CSSProperties = {
    display: 'flex', gap: 4, padding: '12px 24px', background: '#fcfcfd',
    borderBottom: '0.5px solid rgba(16,24,40,0.08)',
  }
  const tab = (active: boolean): React.CSSProperties => ({
    padding: '6px 16px', borderRadius: 8, border: 'none',
    background: active ? 'rgba(21,90,239,0.08)' : 'transparent',
    color: active ? '#155aef' : '#676f83', fontSize: 13, fontWeight: active ? 600 : 400,
    cursor: 'pointer',
  })
  const content: React.CSSProperties = { padding: '24px' }
  const card: React.CSSProperties = {
    background: '#fcfcfd', borderRadius: 12,
    border: '0.5px solid rgba(16,24,40,0.08)', overflow: 'hidden',
  }
  const tableHeader: React.CSSProperties = {
    display: 'grid', gridTemplateColumns: '1.5fr 1fr 1.2fr 0.7fr 1fr 1.2fr 0.7fr 0.7fr 1.2fr',
    padding: '10px 16px', background: '#f9fafb', borderBottom: '0.5px solid rgba(16,24,40,0.08)',
    fontSize: 12, fontWeight: 600, color: '#676f83', textTransform: 'uppercase', letterSpacing: '0.02em',
  }
  const tableRow: React.CSSProperties = {
    display: 'grid', gridTemplateColumns: '1.5fr 1fr 1.2fr 0.7fr 1fr 1.2fr 0.7fr 0.7fr 1.2fr',
    padding: '12px 16px', borderBottom: '0.5px solid rgba(16,24,40,0.06)',
    fontSize: 13, color: '#101828', alignItems: 'center',
  }
  const badge = (color: string, bg: string): React.CSSProperties => ({
    display: 'inline-block', padding: '2px 8px', borderRadius: 6,
    fontSize: 11, fontWeight: 600, color, background: bg,
  })
  const actionBtn: React.CSSProperties = {
    padding: '4px 10px', borderRadius: 6, border: '0.5px solid rgba(16,24,40,0.08)',
    background: '#fcfcfd', color: '#354052', fontSize: 12, cursor: 'pointer', marginRight: 4,
  }
  const overlay: React.CSSProperties = {
    position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
    display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
  }
  const dialog: React.CSSProperties = {
    background: '#fcfcfd', borderRadius: 16, width: '100%', maxWidth: 560, maxHeight: '90vh',
    overflow: 'auto', boxShadow: '0 8px 32px rgba(16,24,40,0.12)',
  }
  const dialogHeader: React.CSSProperties = {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '20px 24px 12px', borderBottom: '0.5px solid rgba(16,24,40,0.08)',
  }
  const dialogBody: React.CSSProperties = { padding: '20px 24px' }
  const dialogFooter: React.CSSProperties = {
    display: 'flex', justifyContent: 'flex-end', gap: 8,
    padding: '16px 24px', borderTop: '0.5px solid rgba(16,24,40,0.08)',
  }
  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '8px 10px', borderRadius: 8, fontSize: 13,
    border: '0.5px solid rgba(16,24,40,0.15)', outline: 'none', background: '#fcfcfd', color: '#101828',
  }
  const labelStyle: React.CSSProperties = {
    fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block',
  }
  const fieldGap: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 2 }
  const formGrid: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 14 }
  const emptyState: React.CSSProperties = {
    display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
    padding: '64px 24px', color: '#676f83', fontSize: 14,
  }
  const stepIndicator: React.CSSProperties = {
    display: 'flex', gap: 12, marginBottom: 20, padding: '0 24px',
  }
  const stepDot = (active: boolean): React.CSSProperties => ({
    width: 8, height: 8, borderRadius: '50%',
    background: active ? '#155aef' : '#d0d5dd',
  })

  return (
    <div style={pageBg}>
      {/* Toolbar */}
      <div style={toolbar}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <a href="#/" style={backLink}>{'< ' + t('common.back')}</a>
          <span style={toolbarTitle}>{t('model.title')}</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <button onClick={openAdd} style={primaryBtn}>{t('model.addModel')}</button>
        </div>
      </div>

      {/* Filter Bar */}
      <div style={filterBar}>
        {CATEGORIES.map(cat => (
          <button
            key={cat}
            onClick={() => setCategory(cat)}
            style={tab(category === cat)}
          >
            {cat === 'all' ? t('common.category') : t(`model.categories.${cat}`)}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={content}>
        {store.loading ? (
          <div style={emptyState}>{t('common.loading')}</div>
        ) : filteredModels.length === 0 ? (
          <div style={emptyState}>{t('model.noModels')}</div>
        ) : (
          <div style={card}>
            <div style={tableHeader}>
              <div>{t('common.name')}</div>
              <div>{t('model.provider')}</div>
              <div>{t('model.modelId')}</div>
              <div>{t('common.category')}</div>
              <div>{t('model.apiKey')}</div>
              <div>{t('model.baseUrl')}</div>
              <div>{t('model.thinkingLevel')}</div>
              <div>{t('common.default')}</div>
              <div>{t('common.actions')}</div>
            </div>
            {filteredModels.map(m => (
              <div key={m.id} style={tableRow}>
                <div style={{ fontWeight: 500 }}>{m.name}</div>
                <div style={{ color: '#676f83' }}>{m.provider}</div>
                <div style={{ fontFamily: 'monospace', fontSize: 12, color: '#354052' }}>{m.model_id}</div>
                <div>
                  <span style={badge('#155aef', 'rgba(21,90,239,0.08)')}>
                    {t(`model.categories.${m.category}`)}
                  </span>
                </div>
                <div style={{ fontFamily: 'monospace', fontSize: 12, color: '#676f83' }}>{maskKey(m.api_key)}</div>
                <div style={{ fontSize: 12, color: '#676f83', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {m.base_url || '-'}
                </div>
                <div style={{ fontSize: 12, color: '#676f83' }}>{t(`think.${m.think_level || 'off'}`)}</div>
                <div style={{ display: 'flex', gap: 4 }}>
                  {m.is_default && <span style={badge('#039855', 'ECFDF3')}>{t('common.default')}</span>}
                  {m.is_base && <span style={badge('#7A5AF8', 'F4F3FF')}>{t('common.base')}</span>}
                </div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                  <button onClick={() => openEdit(m)} style={actionBtn} title={t('common.edit')}>{t('common.edit')}</button>
                  <button onClick={() => handleDelete(m)} style={{ ...actionBtn, color: '#d92d20' }} title={t('common.delete')}>{t('common.delete')}</button>
                  <button onClick={() => handleSetDefault(m.id)} style={actionBtn} title={t('model.setDefault')}>{t('common.default')}</button>
                  <button onClick={() => handleSetBase(m.id)} style={actionBtn} title={t('model.baseModel')}>{t('common.base')}</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Editor Dialog */}
      {dialogOpen && (
        <div style={overlay} onClick={e => { if (e.target === e.currentTarget) setDialogOpen(false) }}>
          <div style={dialog}>
            <div style={dialogHeader}>
              <span style={{ fontSize: 16, fontWeight: 600, color: '#101828' }}>
                {editId ? t('model.editModel') : t('model.addModel')}
              </span>
              <button onClick={() => setDialogOpen(false)} style={{ background: 'none', border: 'none', fontSize: 18, cursor: 'pointer', color: '#676f83' }}>x</button>
            </div>

            {/* Step Indicator */}
            <div style={stepIndicator}>
              <div style={stepDot(step === 1)} />
              <div style={stepDot(step === 2)} />
            </div>

            <div style={dialogBody}>
              {step === 1 ? (
                <div style={formGrid}>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.apiType')}</label>
                    <select
                      value={apiType}
                      onChange={e => { setApiType(e.target.value); setForm(f => ({ ...f, provider: '' })) }}
                      style={inputStyle}
                    >
                      {API_TYPES.map(at => (
                        <option key={at} value={at}>{apiTypeLabel(at)}</option>
                      ))}
                    </select>
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.provider')}</label>
                    <select
                      value={form.provider}
                      onChange={e => fillProvider(e.target.value)}
                      style={inputStyle}
                    >
                      <option value="">{t('common.select')}...</option>
                      {providerList.map(p => <option key={p} value={p}>{p}</option>)}
                    </select>
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.selectCategory')}</label>
                    <select
                      value={form.category}
                      onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
                      style={inputStyle}
                    >
                      {CATEGORIES.filter(c => c !== 'all').map(cat => (
                        <option key={cat} value={cat}>{t(`model.categories.${cat}`)}</option>
                      ))}
                    </select>
                  </div>
                </div>
              ) : (
                <div style={formGrid}>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.displayName')}</label>
                    <input
                      value={form.name}
                      onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                      style={inputStyle}
                      placeholder="e.g. GPT-4o"
                    />
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.modelId')}</label>
                    <input
                      value={form.model_id}
                      onChange={e => setForm(f => ({ ...f, model_id: e.target.value }))}
                      style={inputStyle}
                      placeholder="e.g. gpt-4o"
                    />
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.apiKey')}</label>
                    <input
                      type="password"
                      value={form.api_key}
                      onChange={e => setForm(f => ({ ...f, api_key: e.target.value }))}
                      style={inputStyle}
                      placeholder="sk-..."
                    />
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.baseUrl')}</label>
                    <input
                      value={form.base_url}
                      onChange={e => setForm(f => ({ ...f, base_url: e.target.value }))}
                      style={inputStyle}
                      placeholder="https://api.openai.com/v1"
                    />
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.description')}</label>
                    <input
                      value={form.description}
                      onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                      style={inputStyle}
                    />
                  </div>
                  <div style={fieldGap}>
                    <label style={labelStyle}>{t('model.thinkingLevel')}</label>
                    <select
                      value={form.think_level}
                      onChange={e => setForm(f => ({ ...f, think_level: e.target.value }))}
                      style={inputStyle}
                    >
                      {THINK_LEVELS.map(l => (
                        <option key={l} value={l}>{t(`think.${l}`)}</option>
                      ))}
                    </select>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <input
                      type="checkbox"
                      checked={form.multimodal}
                      onChange={e => setForm(f => ({ ...f, multimodal: e.target.checked }))}
                      id="multimodal"
                    />
                    <label htmlFor="multimodal" style={{ fontSize: 13, color: '#354052', cursor: 'pointer' }}>
                      {t('model.multimodal')} - {t('model.multimodalHint')}
                    </label>
                  </div>
                  <div style={{ display: 'flex', gap: 16 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <input
                        type="checkbox"
                        checked={form.is_default}
                        onChange={e => setForm(f => ({ ...f, is_default: e.target.checked }))}
                        id="is_default"
                      />
                      <label htmlFor="is_default" style={{ fontSize: 13, color: '#354052', cursor: 'pointer' }}>
                        {t('model.defaultModel')}
                      </label>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <input
                        type="checkbox"
                        checked={form.is_base}
                        onChange={e => setForm(f => ({ ...f, is_base: e.target.checked }))}
                        id="is_base"
                      />
                      <label htmlFor="is_base" style={{ fontSize: 13, color: '#354052', cursor: 'pointer' }}>
                        {t('model.baseModel')} - {t('model.baseModelHint')}
                      </label>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <div style={dialogFooter}>
              <button onClick={() => setDialogOpen(false)} style={secondaryBtn}>{t('common.cancel')}</button>
              {step === 2 && <button onClick={() => setStep(1)} style={secondaryBtn}>{t('common.prev')}</button>}
              {step === 1 ? (
                <button onClick={() => setStep(2)} style={primaryBtn}>{t('common.next')}</button>
              ) : (
                <button
                  onClick={handleSave}
                  disabled={saving || !form.name.trim() || !form.model_id.trim()}
                  style={{ ...primaryBtn, opacity: saving || !form.name.trim() || !form.model_id.trim() ? 0.6 : 1 }}
                >
                  {saving ? t('common.loading') : t('common.save')}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
