import { create } from 'zustand'
import { API_BASE } from '@/constants'

export interface Skill {
  id: number
  skill_id: string
  name: string
  version: string
  description: string
  icon?: string
  inputs?: string
  outputs?: string
  default_model?: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface SkillPrompt {
  id?: number
  skill_id?: number
  name: string
  content: string
}

export interface SkillDetail extends Skill {
  prompts: SkillPrompt[]
  resources: any[]
}

interface SkillState {
  skills: Skill[]
  loading: boolean
  fetchSkills: () => Promise<void>
  getSkill: (id: number) => Promise<SkillDetail | null>
  saveSkill: (skill: Partial<Skill> & { prompts?: SkillPrompt[] }) => Promise<Skill | null>
  deleteSkill: (id: number) => Promise<void>
  executeSkill: (id: number, inputs: Record<string, any>, model?: string) => Promise<string>
}

export const useSkillStore = create<SkillState>((set, get) => ({
  skills: [],
  loading: false,
  fetchSkills: async () => {
    set({ loading: true })
    try {
      const res = await fetch(`${API_BASE}/api/skills`)
      const data = await res.json()
      set({ skills: Array.isArray(data.data) ? data.data : [] })
    } catch (e) {
      console.error(e)
    } finally {
      set({ loading: false })
    }
  },
  getSkill: async (id) => {
    try {
      const res = await fetch(`${API_BASE}/api/skills/${id}`)
      const data = await res.json()
      return data.data as SkillDetail
    } catch (e) {
      console.error(e)
      return null
    }
  },
  saveSkill: async (skill) => {
    const isUpdate = !!skill.id
    const url = `${API_BASE}/api/skills${isUpdate ? `/${skill.id}` : ''}`
    const body = {
      skill_id: skill.skill_id,
      name: skill.name,
      version: skill.version || '1.0.0',
      description: skill.description || '',
      icon: skill.icon || '',
      inputs: skill.inputs || '',
      outputs: skill.outputs || '',
      default_model: skill.default_model || '',
      enabled: skill.enabled !== false,
      prompts: skill.prompts || [],
    }
    try {
      const res = await fetch(url, {
        method: isUpdate ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (!res.ok) {
        alert(data?.msg || 'Save failed')
        return null
      }
      get().fetchSkills()
      return data.data as Skill
    } catch (e) {
      console.error(e)
      return null
    }
  },
  deleteSkill: async (id) => {
    try {
      await fetch(`${API_BASE}/api/skills/${id}`, { method: 'DELETE' })
      get().fetchSkills()
    } catch (e) {
      console.error(e)
    }
  },
  executeSkill: async (id, inputs, model) => {
    const res = await fetch(`${API_BASE}/api/skills/${id}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ inputs, model }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data?.msg || 'Execution failed')
    return data.data?.output || ''
  },
}))
