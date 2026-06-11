import { API_BASE } from '@/constants'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { FlowDefinition, FlowDetail, FlowEvent } from '@/types/flow'


export const useFlowStore = defineStore('flow', () => {
  const flows = ref<FlowDefinition[]>([])
  const flowEvents = ref<FlowEvent[]>([])
  const activeExecutionId = ref<number | null>(null)

  async function fetchFlows(category?: string) {
    try {
      const params = category ? `?category=${category}` : ''
      const res = await fetch(`${API_BASE}/api/flows${params}`)
      const data = await res.json()
      flows.value = data.data || []
    } catch (e) { console.error('fetchFlows failed', e) }
  }

  async function fetchFlow(id: number): Promise<FlowDetail | null> {
    try {
      const res = await fetch(`${API_BASE}/api/flows/${id}`)
      const data = await res.json()
      return data.data || null
    } catch (e) { console.error('fetchFlow failed', e); return null }
  }

  async function saveFlow(payload: {
    name: string; description?: string; category?: string
    nodes?: any[]; edges?: any[]
  }, flowId?: number) {
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
        if (flowId) {
          const idx = flows.value.findIndex(f => f.id === flowId)
          if (idx !== -1) flows.value[idx] = saved
        } else {
          flows.value.push(saved)
        }
      }
      return saved
    } catch (e) { console.error('saveFlow failed', e); return null }
  }

  async function deleteFlow(id: number) {
    try {
      await fetch(`${API_BASE}/api/flows/${id}`, { method: 'DELETE' })
      flows.value = flows.value.filter(f => f.id !== id)
    } catch (e) { console.error('deleteFlow failed', e) }
  }

  function addFlowEvent(event: FlowEvent) {
    flowEvents.value.push(event)
    if (event.execution_id) activeExecutionId.value = event.execution_id
  }

  function clearFlowEvents() {
    flowEvents.value = []
    activeExecutionId.value = null
  }

  return { flows, flowEvents, activeExecutionId, fetchFlows, fetchFlow, saveFlow, deleteFlow, addFlowEvent, clearFlowEvents }
})
