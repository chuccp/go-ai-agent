import { create } from 'zustand'
import { API_BASE } from '@/constants'
import type { FlowDefinition, FlowDetail, FlowEvent } from '@/types/flow'

interface FlowState {
  flows: FlowDefinition[]
  flowEvents: FlowEvent[]
  activeExecutionId: number | null
  fetchFlows: (category?: string) => Promise<void>
  fetchFlow: (id: number) => Promise<FlowDetail | null>
  saveFlow: (payload: any, flowId?: number) => Promise<any>
  deleteFlow: (id: number) => Promise<void>
  uploadIcon: (flowId: number, file: File) => Promise<boolean>
  getIconUrl: (flowId: number) => string
  addFlowEvent: (event: FlowEvent) => void
  clearFlowEvents: () => void
}

export const useFlowStore = create<FlowState>((set, get) => ({
  flows: [],
  flowEvents: [],
  activeExecutionId: null,

  async fetchFlows(category?: string) {
    try {
      const params = category ? `?category=${category}` : ''
      const res = await fetch(`${API_BASE}/api/flows${params}`)
      const data = await res.json()
      set({ flows: data.data || [] })
    } catch (e) { console.error('fetchFlows failed', e) }
  },

  async fetchFlow(id: number) {
    try {
      const res = await fetch(`${API_BASE}/api/flows/${id}`)
      const data = await res.json()
      return data.data || null
    } catch (e) { console.error('fetchFlow failed', e); return null }
  },

  async saveFlow(payload: any, flowId?: number) {
    try {
      const url = flowId ? `${API_BASE}/api/flows/${flowId}` : `${API_BASE}/api/flows`
      const res = await fetch(url, {
        method: flowId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      const data = await res.json()
      const saved = data.data
      if (saved) {
        const flows = get().flows
        if (flowId) {
          set({ flows: flows.map(f => f.id === flowId ? saved : f) })
        } else {
          set({ flows: [...flows, saved] })
        }
      }
      return saved
    } catch (e) { console.error('saveFlow failed', e); return null }
  },

  async deleteFlow(id: number) {
    try {
      await fetch(`${API_BASE}/api/flows/${id}`, { method: 'DELETE' })
      set({ flows: get().flows.filter(f => f.id !== id) })
    } catch (e) { console.error('deleteFlow failed', e) }
  },

  async uploadIcon(flowId: number, file: File) {
    try {
      const formData = new FormData()
      formData.append('icon', file)
      const res = await fetch(`${API_BASE}/api/flows/${flowId}/icon`, {
        method: 'POST',
        body: formData,
      })
      const data = await res.json()
      if (data.data) {
        const flows = get().flows
        set({ flows: flows.map(f => f.id === flowId ? { ...f, icon: data.data.icon } : f) })
      }
      return true
    } catch (e) { console.error('uploadIcon failed', e); return false }
  },

  getIconUrl(flowId: number) {
    return `${API_BASE}/api/flows/${flowId}/icon`
  },

  addFlowEvent(event: FlowEvent) {
    set(state => ({
      flowEvents: [...state.flowEvents, event],
      activeExecutionId: event.execution_id || state.activeExecutionId,
    }))
  },

  clearFlowEvents() {
    set({ flowEvents: [], activeExecutionId: null })
  },
}))
