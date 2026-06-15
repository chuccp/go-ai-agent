import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useSkillStore, type Skill, type SkillPrompt } from '@/stores/skillStore'

export default function SkillManager() {
  const { t } = useTranslation()
  const { skills, loading, fetchSkills, saveSkill, deleteSkill } = useSkillStore()
  const [editing, setEditing] = useState<Partial<Skill> & { prompts?: SkillPrompt[] } | null>(null)

  useEffect(() => { fetchSkills() }, [fetchSkills])

  const handleSave = async () => {
    if (!editing?.skill_id || !editing.name) return
    await saveSkill(editing)
    setEditing(null)
  }

  const addPrompt = () => {
    if (!editing) return
    setEditing({
      ...editing,
      prompts: [...(editing.prompts || []), { name: `prompt-${(editing.prompts?.length || 0) + 1}`, content: '' }],
    })
  }

  const updatePrompt = (idx: number, p: SkillPrompt) => {
    if (!editing) return
    const prompts = [...(editing.prompts || [])]
    prompts[idx] = p
    setEditing({ ...editing, prompts })
  }

  const removePrompt = (idx: number) => {
    if (!editing) return
    const prompts = [...(editing.prompts || [])]
    prompts.splice(idx, 1)
    setEditing({ ...editing, prompts })
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '8px 10px', border: '1px solid #d0d5dd', borderRadius: 8, fontSize: 13, outline: 'none',
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: '#f2f4f7' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px', background: '#fff', borderBottom: '0.5px solid rgba(16,24,40,0.08)' }}>
        <a href="#/" style={{ display: 'flex', alignItems: 'center', color: '#676f83', textDecoration: 'none', fontSize: 13, gap: 4 }}>
          ← {t('common.back')}
        </a>
        <div style={{ flex: 1 }} />
        <button onClick={() => setEditing({ skill_id: '', name: '', version: '1.0.0', description: '', prompts: [], enabled: true })} style={{ padding: '7px 16px', borderRadius: 8, border: 'none', background: '#155aef', color: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer' }}>
          + {t('skill.newSkill') || 'New Skill'}
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
        {loading && <div style={{ color: '#676f83' }}>{t('common.loading')}</div>}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 14 }}>
          {skills.map(s => (
            <div key={s.id} style={{ background: '#fff', borderRadius: 12, border: '0.5px solid rgba(16,24,40,0.08)', padding: 16, display: 'flex', flexDirection: 'column', gap: 8 }}>
              <div style={{ fontSize: 15, fontWeight: 600, color: '#101828' }}>{s.icon || '🧩'} {s.name}</div>
              <div style={{ fontSize: 12, color: '#676f83' }}>{s.skill_id} · v{s.version}</div>
              <div style={{ fontSize: 12, color: '#98a2b3' }}>{s.description}</div>
              <div style={{ display: 'flex', gap: 8, marginTop: 'auto' }}>
                <button onClick={() => setEditing({ ...s, prompts: [] })} style={{ ...inputStyle, width: 'auto', padding: '6px 14px', cursor: 'pointer' }}>{t('common.edit')}</button>
                <button onClick={() => { if (window.confirm(t('skill.confirmDelete') || 'Delete?')) deleteSkill(s.id) }} style={{ ...inputStyle, width: 'auto', padding: '6px 14px', color: '#f04438', borderColor: '#f0443833', cursor: 'pointer' }}>{t('common.delete')}</button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {editing && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 }} onClick={() => setEditing(null)}>
          <div style={{ background: '#fff', borderRadius: 16, padding: 24, width: 520, maxHeight: '90vh', overflow: 'auto', boxShadow: '0 12px 32px rgba(0,0,0,0.15)' }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 16 }}>{editing.id ? t('skill.editSkill') || 'Edit Skill' : t('skill.newSkill') || 'New Skill'}</div>
            <div style={{ display: 'grid', gap: 12 }}>
              <div>
                <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>Skill ID</label>
                <input value={editing.skill_id || ''} onChange={e => setEditing({ ...editing, skill_id: e.target.value })} style={inputStyle} placeholder="unique-id" />
              </div>
              <div>
                <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>{t('common.name')}</label>
                <input value={editing.name || ''} onChange={e => setEditing({ ...editing, name: e.target.value })} style={inputStyle} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>{t('common.version')}</label>
                <input value={editing.version || ''} onChange={e => setEditing({ ...editing, version: e.target.value })} style={inputStyle} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>{t('common.description')}</label>
                <textarea value={editing.description || ''} onChange={e => setEditing({ ...editing, description: e.target.value })} rows={3} style={{ ...inputStyle, resize: 'vertical' }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>{t('skill.defaultModel') || 'Default Model'}</label>
                <input value={editing.default_model || ''} onChange={e => setEditing({ ...editing, default_model: e.target.value })} style={inputStyle} placeholder="1.default" />
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <input type="checkbox" checked={editing.enabled !== false} onChange={e => setEditing({ ...editing, enabled: e.target.checked })} id="skillEnabled" />
                <label htmlFor="skillEnabled" style={{ fontSize: 13, color: '#354052', cursor: 'pointer' }}>{t('common.enabled') || 'Enabled'}</label>
              </div>

              <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginTop: 8 }}>{t('skill.prompts') || 'Prompts'}</div>
              {editing.prompts?.map((p, idx) => (
                <div key={idx} style={{ border: '1px solid #e2e8f0', borderRadius: 8, padding: 10, display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <input value={p.name} onChange={e => updatePrompt(idx, { ...p, name: e.target.value })} style={inputStyle} placeholder="Prompt name" />
                  <textarea value={p.content} onChange={e => updatePrompt(idx, { ...p, content: e.target.value })} rows={4} style={{ ...inputStyle, resize: 'vertical' }} placeholder="Prompt template (supports {{.inputName}})" />
                  <button onClick={() => removePrompt(idx)} style={{ alignSelf: 'flex-end', fontSize: 12, color: '#f04438', background: 'none', border: 'none', cursor: 'pointer' }}>{t('common.remove')}</button>
                </div>
              ))}
              <button onClick={addPrompt} style={{ padding: '6px 12px', borderRadius: 8, border: '1px dashed #d0d5dd', background: '#fff', color: '#354052', fontSize: 13, cursor: 'pointer' }}>+ {t('skill.addPrompt') || 'Add Prompt'}</button>
            </div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 20 }}>
              <button onClick={() => setEditing(null)} style={{ padding: '8px 20px', borderRadius: 8, border: '1px solid #d0d5dd', background: '#fff', color: '#354052', fontSize: 13, cursor: 'pointer' }}>{t('common.cancel')}</button>
              <button onClick={handleSave} style={{ padding: '8px 20px', borderRadius: 8, border: 'none', background: '#155aef', color: '#fff', fontSize: 13, cursor: 'pointer' }}>{t('common.save')}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
