import { create } from 'zustand'
import { API_BASE } from '@/constants'

interface SetupStatus {
  initialized: boolean
  db_configured: boolean
  admin_configured: boolean
  mode: string
}

async function fetchSetupStatus(): Promise<SetupStatus> {
  try {
    const res = await fetch(`${API_BASE}/api/setup/status`)
    const data = await res.json()
    const s = data.data || {}
    return {
      initialized: s.initialized ?? false,
      db_configured: s.db_configured ?? false,
      admin_configured: s.admin_configured ?? false,
      mode: s.mode ?? 'web',
    }
  } catch {
    return { initialized: false, db_configured: false, admin_configured: false, mode: 'web' }
  }
}

interface SetupState {
  initialized: boolean | null
  checkSetup: () => Promise<boolean>
  getSetupStatus: () => Promise<SetupStatus>
  testConnection: (data: any) => Promise<{ ok: boolean; msg: string }>
  initDatabase: (data: any) => Promise<{ ok: boolean; msg: string }>
  checkAdminExists: () => Promise<boolean>
  initAdmin: (data: any) => Promise<{ ok: boolean; msg: string }>
  initBaseModel: (data: any) => Promise<{ ok: boolean; msg: string }>
  completeSetup: () => Promise<{ ok: boolean; msg: string }>
  fetchProviderDefaults: () => Promise<Record<string, any>>
  clearDatabase: () => Promise<boolean>
  clearAllData: () => Promise<boolean>
}

export const useSetupStore = create<SetupState>((set) => ({
  initialized: null,

  getSetupStatus: async () => {
    const s = await fetchSetupStatus()
    set({ initialized: s.initialized })
    return s
  },

  checkSetup: async () => {
    const s = await fetchSetupStatus()
    set({ initialized: s.initialized })
    return s.initialized
  },

  async testConnection(data) {
    try {
      const res = await fetch(`${API_BASE}/api/setup/db/test`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data),
      })
      const r = await res.json()
      return { ok: res.ok && r.code === 200, msg: r.msg || '' }
    } catch (e: any) { return { ok: false, msg: e.message } }
  },

  async initDatabase(data) {
    try {
      const res = await fetch(`${API_BASE}/api/setup/db`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data),
      })
      const r = await res.json()
      return { ok: res.ok && r.code === 200, msg: r.msg || '' }
    } catch (e: any) { return { ok: false, msg: e.message } }
  },

  async checkAdminExists() {
    try {
      const res = await fetch(`${API_BASE}/api/setup/admin/exists`)
      const data = await res.json()
      return data.data?.exists ?? false
    } catch { return false }
  },

  async initAdmin(data) {
    try {
      const res = await fetch(`${API_BASE}/api/setup/admin`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data),
      })
      const r = await res.json()
      return { ok: res.ok && r.code === 200, msg: r.msg || '' }
    } catch (e: any) { return { ok: false, msg: e.message } }
  },

  async initBaseModel(data) {
    try {
      const res = await fetch(`${API_BASE}/api/setup/model`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data),
      })
      const r = await res.json()
      return { ok: res.ok && r.code === 200, msg: r.msg || '' }
    } catch (e: any) { return { ok: false, msg: e.message } }
  },

  async completeSetup() {
    try {
      const res = await fetch(`${API_BASE}/api/setup/complete`, { method: 'POST' })
      const r = await res.json()
      if (res.ok) set({ initialized: true })
      return { ok: res.ok, msg: r.msg || '' }
    } catch (e: any) { return { ok: false, msg: e.message } }
  },

  async fetchProviderDefaults() {
    try {
      const res = await fetch(`${API_BASE}/api/setup/providers`)
      const data = await res.json()
      return data.data || {}
    } catch { return {} }
  },

  async clearDatabase() {
    try {
      const res = await fetch(`${API_BASE}/api/system/clear-db`, { method: 'POST' })
      return res.ok
    } catch { return false }
  },

  async clearAllData() {
    try {
      const res = await fetch(`${API_BASE}/api/system/clear-all`, { method: 'POST' })
      return res.ok
    } catch { return false }
  },
}))
