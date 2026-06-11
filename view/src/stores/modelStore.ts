import { create } from 'zustand'
import { API_BASE } from '@/constants'

export interface AIModel {
  id: number
  name: string
  provider: string
  model_id: string
  category: string
  api_key: string
  base_url: string
  description: string
  is_default: boolean
  is_base: boolean
  think_level: string
  multimodal: boolean
  created_at: string
  updated_at: string
}

interface ModelState {
  models: AIModel[]
  loading: boolean
  providerDefaults: Record<string, any>
  fetchModels: (category?: string) => Promise<void>
  fetchProviderDefaults: () => Promise<void>
  createModel: (data: Partial<AIModel>) => Promise<boolean>
  updateModel: (id: number, data: Partial<AIModel>) => Promise<boolean>
  deleteModel: (id: number) => Promise<boolean>
  setDefault: (id: number) => Promise<boolean>
  setBase: (id: number) => Promise<boolean>
}

export const useModelStore = create<ModelState>((set, get) => ({
  models: [],
  loading: false,
  providerDefaults: {},

  async fetchModels(category?: string) {
    set({ loading: true })
    try {
      const params = category ? `?category=${category}` : ''
      const res = await fetch(`${API_BASE}/api/ai-models${params}`)
      const data = await res.json()
      set({ models: data.data || [] })
    } catch (e) { console.error('fetchModels failed', e) }
    finally { set({ loading: false }) }
  },

  async fetchProviderDefaults() {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/providers`)
      const data = await res.json()
      set({ providerDefaults: data.data || {} })
    } catch (e) { console.error('fetchProviderDefaults failed', e) }
  },

  async createModel(d) {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(d),
      })
      if (res.ok) { await get().fetchModels(); return true }
      return false
    } catch { return false }
  },

  async updateModel(id, d) {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/${id}`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(d),
      })
      if (res.ok) { await get().fetchModels(); return true }
      return false
    } catch { return false }
  },

  async deleteModel(id) {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/${id}`, { method: 'DELETE' })
      if (res.ok) { set({ models: get().models.filter(m => m.id !== id) }); return true }
      return false
    } catch { return false }
  },

  async setDefault(id) {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/${id}/default`, { method: 'PUT' })
      if (res.ok) { await get().fetchModels(); return true }
      return false
    } catch { return false }
  },

  async setBase(id) {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/${id}/base`, { method: 'PUT' })
      if (res.ok) { await get().fetchModels(); return true }
      return false
    } catch { return false }
  },
}))
