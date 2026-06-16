import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import PasswordInput from '@/components/PasswordInput'
import { useModelStore } from '@/stores/modelStore'

export interface ModelFormData {
  name: string
  provider: string
  model: string
  category: string
  api_key: string
  base_url: string
  description: string
  think_level: string
  multimodal: boolean
  is_default: boolean
  is_base: boolean
}

export function emptyModelForm(): ModelFormData {
  return {
    name: '', provider: '', model: '', category: 'llm',
    api_key: '', base_url: '', description: '',
    think_level: 'off', multimodal: false,
    is_default: true, is_base: true,
  }
}

const API_TYPES = ['openai', 'claude', 'native'] as const

interface Props {
  form: ModelFormData
  onChange: (form: ModelFormData) => void
  readOnly?: boolean
}

export default function ModelForm({ form, onChange, readOnly }: Props) {
  const { t } = useTranslation()
  const store = useModelStore()
  const [apiType, setApiType] = useState('openai')

  useEffect(() => {
    store.fetchProviderDefaults()
  }, [])

  const providerList = useMemo(() => {
    const defaults = store.providerDefaults[apiType]
    return defaults ? Object.keys(defaults) : []
  }, [store.providerDefaults, apiType])

  const fillProvider = (p: string) => {
    const d = store.providerDefaults[apiType]?.[p]
    if (d) {
      onChange({
        ...form,
        provider: p,
        name: d.model || form.name || '',
        model: d.model || '',
        base_url: d.baseUrl || '',
      })
    } else {
      onChange({ ...form, provider: p })
    }
  }

  const apiTypeLabel = (key: string) => {
    if (key === 'openai') return t('model.openaiCompat')
    if (key === 'claude') return t('model.claudeCompat')
    return t('model.native')
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '8px 10px', borderRadius: 8, fontSize: 13,
    border: '0.5px solid rgba(16,24,40,0.15)', outline: 'none', background: '#fcfcfd', color: '#101828',
  }
  const apiTypeBtnGroup: React.CSSProperties = { display: 'flex', gap: 10 }
  const apiTypeBtn = (active: boolean): React.CSSProperties => ({
    flex: 1, padding: '10px 0', fontSize: 14, fontWeight: active ? 600 : 500, cursor: 'pointer',
    borderRadius: 8, border: active ? '2px solid #155aef' : '0.5px solid rgba(16,24,40,0.15)',
    outline: 'none',
    background: active ? '#eef4ff' : '#fcfcfd',
    color: active ? '#155aef' : '#354052',
    transition: 'all 0.15s',
  })
  const labelStyle: React.CSSProperties = { fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }
  const fieldGap: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 2 }
  const formGrid: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 10 }

  return (
    <div style={formGrid}>
      <div style={fieldGap}>
        <label style={labelStyle}>{t('model.apiType')}</label>
        <div style={apiTypeBtnGroup}>
          {API_TYPES.map(at => (
            <button key={at} type="button" onClick={() => { setApiType(at); onChange({ ...form, provider: '' }) }} style={apiTypeBtn(at === apiType)}>
              {apiTypeLabel(at)}
            </button>
          ))}
        </div>
      </div>
      <div style={fieldGap}>
        <label style={labelStyle}>{t('model.provider')}</label>
        <select value={form.provider} onChange={e => fillProvider(e.target.value)} style={inputStyle}>
          <option value="">{t('common.select')}...</option>
          {providerList.map(p => <option key={p} value={p}>{p}</option>)}
        </select>
      </div>
      <div style={fieldGap}>
        <label style={labelStyle}>{t('model.modelId')}</label>
        <input value={form.model} onChange={e => onChange({ ...form, model: e.target.value })} style={inputStyle} placeholder="gpt-4o" />
      </div>
        <div style={fieldGap}>
          <label style={labelStyle}>{t('model.apiKey')}</label>
          <PasswordInput value={form.api_key} onChange={v => onChange({ ...form, api_key: v })} placeholder="sk-..." />
        </div>
      <div style={fieldGap}>
        <label style={labelStyle}>{t('model.baseUrl')}</label>
        <input value={form.base_url} onChange={e => onChange({ ...form, base_url: e.target.value })} style={inputStyle} placeholder="https://api.openai.com/v1" />
      </div>
      <div style={fieldGap}>
        <label style={labelStyle}>{t('model.displayName')} ({t('common.optional')})</label>
        <input value={form.name} onChange={e => onChange({ ...form, name: e.target.value })} style={inputStyle} placeholder="GPT-4o" />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <input type="checkbox" checked={form.multimodal} onChange={e => onChange({ ...form, multimodal: e.target.checked })} id="mf-multimodal" />
        <label htmlFor="mf-multimodal" style={{ fontSize: 13, color: '#354052', cursor: 'pointer' }}>{t('model.multimodalHint')}</label>
      </div>
    </div>
  )
}
