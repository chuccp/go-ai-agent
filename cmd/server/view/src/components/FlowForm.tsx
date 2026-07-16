import { useState } from 'react'
import type { FormSchema, FormField } from '@/types/flow'

interface FlowFormProps {
  schema: FormSchema
  onSubmit: (values: Record<string, any>) => void
  disabled?: boolean
}

export default function FlowForm({ schema, onSubmit, disabled }: FlowFormProps) {
  const [values, setValues] = useState<Record<string, any>>(() => {
    const init: Record<string, any> = {}
    for (const f of schema.fields) {
      init[f.name] = f.default ?? (f.type === 'checkbox' ? false : '')
    }
    return init
  })

  const handleChange = (field: FormField, value: any) => {
    setValues(prev => ({ ...prev, [field.name]: value }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(values)
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 10px',
    border: '1px solid #d0d5dd',
    borderRadius: 8,
    fontSize: 13,
    outline: 'none',
    background: '#fff',
  }

  const labelStyle: React.CSSProperties = {
    fontSize: 12,
    color: '#354052',
    fontWeight: 500,
    marginBottom: 4,
    display: 'block',
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {schema.fields.map(field => (
        <div key={field.name}>
          <label style={labelStyle}>
            {field.label}
            {field.required && <span style={{ color: '#f04438', marginLeft: 2 }}>*</span>}
          </label>
          {field.type === 'textarea' && (
            <textarea
              value={values[field.name] || ''}
              onChange={e => handleChange(field, e.target.value)}
              required={field.required}
              disabled={disabled}
              rows={4}
              style={{ ...inputStyle, resize: 'vertical' }}
            />
          )}
          {field.type === 'select' && (
            <select
              value={values[field.name] || ''}
              onChange={e => handleChange(field, e.target.value)}
              required={field.required}
              disabled={disabled}
              style={inputStyle}
            >
              {(field.options || []).map(opt => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          )}
          {field.type === 'radio' && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, marginTop: 6 }}>
              {(field.options || []).map(opt => (
                <label key={opt} style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 13, color: '#354052', cursor: 'pointer' }}>
                  <input
                    type="radio"
                    name={field.name}
                    value={opt}
                    checked={values[field.name] === opt}
                    onChange={() => handleChange(field, opt)}
                    disabled={disabled}
                  />
                  {opt}
                </label>
              ))}
            </div>
          )}
          {field.type === 'checkbox' && (
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: '#354052', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={!!values[field.name]}
                onChange={e => handleChange(field, e.target.checked)}
                disabled={disabled}
              />
              {field.label}
            </label>
          )}
          {(field.type === 'text' || field.type === 'number' || field.type === 'file') && (
            <input
              type={field.type === 'number' ? 'number' : field.type === 'file' ? 'file' : 'text'}
              value={field.type === 'file' ? undefined : (values[field.name] || '')}
              onChange={e => handleChange(field, field.type === 'file' ? e.target.files?.[0] : e.target.value)}
              required={field.required}
              disabled={disabled}
              style={inputStyle}
            />
          )}
        </div>
      ))}
      <button
        type="submit"
        disabled={disabled}
        style={{
          padding: '10px 20px',
          borderRadius: 8,
          border: 'none',
          background: '#155aef',
          color: '#fff',
          fontSize: 14,
          fontWeight: 600,
          cursor: disabled ? 'not-allowed' : 'pointer',
          opacity: disabled ? 0.6 : 1,
        }}
      >
        {disabled ? 'Running...' : 'Run Flow'}
      </button>
    </form>
  )
}
