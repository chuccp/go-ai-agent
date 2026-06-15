import { create } from 'zustand'
import { API_BASE } from '@/constants'

export interface PackageItem {
  id: number
  package_id: string
  name: string
  version: string
  description: string
  icon?: string
  kind: string
  created_at: string
  updated_at: string
}

interface PackageState {
  packages: PackageItem[]
  loading: boolean
  fetchPackages: () => Promise<void>
  importPackage: (file: File) => Promise<PackageItem | null>
  exportPackage: (id: number, name: string) => Promise<void>
  deletePackage: (id: number) => Promise<void>
}

export const usePackageStore = create<PackageState>((set, get) => ({
  packages: [],
  loading: false,
  fetchPackages: async () => {
    set({ loading: true })
    try {
      const res = await fetch(`${API_BASE}/api/packages`)
      const data = await res.json()
      set({ packages: Array.isArray(data.data) ? data.data : [] })
    } catch (e) {
      console.error(e)
    } finally {
      set({ loading: false })
    }
  },
  importPackage: async (file) => {
    const form = new FormData()
    form.append('file', file)
    try {
      const res = await fetch(`${API_BASE}/api/packages/import`, { method: 'POST', body: form })
      const data = await res.json()
      if (!res.ok) {
        alert(data?.msg || 'Import failed')
        return null
      }
      get().fetchPackages()
      return data.data as PackageItem
    } catch (e) {
      console.error(e)
      return null
    }
  },
  exportPackage: async (id, name) => {
    const res = await fetch(`${API_BASE}/api/packages/${id}/export`)
    if (!res.ok) { alert('Export failed'); return }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${name || 'package'}.zip`
    a.click()
    URL.revokeObjectURL(url)
  },
  deletePackage: async (id) => {
    try {
      await fetch(`${API_BASE}/api/packages/${id}`, { method: 'DELETE' })
      get().fetchPackages()
    } catch (e) {
      console.error(e)
    }
  },
}))
