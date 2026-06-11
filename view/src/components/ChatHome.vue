<template>
  <div class="app">
    <SessionList
      :sessions="sessions"
      :active-session-id="activeSessionId"
      @select="selectSession"
      @new="createNewSession"
      @delete="deleteSession"
    />
    <div class="main-area">
      <!-- Flow bar -->
      <div class="flow-bar" v-if="selectedFlowId">
        <span class="flow-badge">⚡ {{ selectedFlowName }}</span>
        <button class="flow-action" @click="selectedFlowId = null">退出流程</button>
        <button v-if="isFlowRunning" class="flow-stop" @click="stopFlow">⏹ 停止</button>
      </div>

      <FlowProgress :events="flowEvents" :is-running="isFlowRunning" />

      <ChatPanel
        :messages="messages"
        :is-streaming="isStreaming"
        @send="sendMessage"
        @upload="onFilesSelected"
        @model-change="onModelChange"
        @think-change="onThinkChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { API_BASE } from '@/constants'
import { useFlowStore } from '@/stores/flow'
import { useChatStream } from '@/composables/useChatStream'
import type { FlowEvent } from '@/types/flow'
import SessionList from './SessionList.vue'
import ChatPanel from './ChatPanel.vue'
import FlowProgress from './FlowProgress.vue'

interface Session { id: number; title: string; created_at: string; updated_at: string }
interface Message { id?: number; role: string; content: string; reasoning?: string; flowId?: number; attachments?: Attachment[]; status?: 'thinking' | 'streaming' }
interface Attachment { id: string; name: string; type: string; size: number; path: string }

const sessions = ref<Session[]>([])
const messages = ref<Message[]>([])
const activeSessionId = ref<number | null>(null)
const isStreaming = ref(false)
const isFlowRunning = ref(false)
const selectedModelId = ref('')
const thinkLevel = ref('off')
const selectedFlowId = ref<number | null>(null)
const flowEvents = ref<FlowEvent[]>([])
const currentExecutionId = ref<number | null>(null)
const ws = ref<WebSocket | null>(null)
const flowStore = useFlowStore()
const pendingAttachments = ref<Attachment[]>([])

const selectedFlowName = computed(() => {
  if (!selectedFlowId.value) return ''
  const f = flowStore.flows.find(fl => fl.id === selectedFlowId.value)
  return f ? f.name : ''
})

// Session management
async function fetchSessions() {
  try { const r = await fetch(`${API_BASE}/api/sessions`); const d = await r.json(); sessions.value = d.data || [] } catch {}
}
async function fetchMessages(sessionId: number) {
  try { const r = await fetch(`${API_BASE}/api/sessions/${sessionId}/messages`); const d = await r.json(); messages.value = d.data || [] } catch {}
}
async function createNewSession(flowId: number | null) {
  try {
    const r = await fetch(`${API_BASE}/api/sessions`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title: 'New Chat' }) })
    const d = await r.json(); const s = d.data as Session
    sessions.value.unshift(s); activeSessionId.value = s.id; messages.value = []
    selectedFlowId.value = flowId
  } catch {}
}
async function deleteSession(id: number) {
  try {
    await fetch(`${API_BASE}/api/sessions/${id}`, { method: 'DELETE' })
    sessions.value = sessions.value.filter(s => s.id !== id)
    if (activeSessionId.value === id) { activeSessionId.value = null; messages.value = [] }
  } catch {}
}
function selectSession(id: number) { activeSessionId.value = id; fetchMessages(id) }
function onModelChange(id: string) { selectedModelId.value = id }
function onThinkChange(level: string) { thinkLevel.value = level }

// ── Streaming composable ──
const { beginStream, addReasoning, acceptContent, appendDelta, setThinkingText, endStream } = useChatStream(messages, isStreaming, isFlowRunning)

// ── WebSocket ──
function connectWebSocket() {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws.value = new WebSocket(`${protocol}//${location.host}/ws/chat`)
  ws.value.onmessage = (event: MessageEvent) => {
    const data = JSON.parse(event.data)
    if (data.type && data.type.startsWith('flow_')) { handleFlowEvent(data as FlowEvent); return }
    if (data.type === 'pong') return
    if (data.type === 'session_created') { activeSessionId.value = data.session_id; fetchSessions(); return }

    if (data.type === 'chunk') {
      if (data.done) { endStream(); return }

      if (data.reasoning && data.content) {
        addReasoning(data.content)
      } else if (data.content && !data.reasoning) {
        const m = findStream()
        if (m && m.status === 'thinking') acceptContent(data.content)
        else appendDelta(data.content)
      }
      return
    }

    if (data.type === 'tool_call') {
      setThinkingText(data.message || '处理中...')
      return
    }

    if (data.type === 'tool_result') {
      popStale()
      let flowId: number | undefined
      const m = data.message || ''
      const fm = m.match(/ID:\s*(\d+)/)
      if (fm && (m.includes('流程创建成功') || m.includes('流程更新成功'))) {
        flowId = parseInt(fm[1]); flowStore.fetchFlows()
      }
      messages.value.push({ role: 'tool', content: m, flowId })
      beginStream()
      return
    }

    if (data.type === 'error') {
      endStream()
      messages.value.push({ role: 'tool', content: '❌ ' + data.message })
    }
  }
  ws.value.onclose = () => setTimeout(connectWebSocket, 2000)
}

// ── Send ──
async function sendMessage(content: string) {
  if (isStreaming.value) return
  if (!content.trim() && pendingAttachments.value.length === 0) return
  if (!activeSessionId.value) {
    try {
      const title = content.trim() ? content.substring(0, 30) : 'New Chat'
      const r = await fetch(`${API_BASE}/api/sessions`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title }) })
      const d = await r.json(); const s = d.data as Session; activeSessionId.value = s.id; sessions.value.unshift(s)
    } catch { return }
  }

  const atts = [...pendingAttachments.value]
  pendingAttachments.value = []

  const wsMsgs = [...messages.value.map(m => ({ role: m.role, content: m.content })), { role: 'user', content }]
  messages.value.push({ role: 'user', content, attachments: atts })
  isStreaming.value = true

  if (selectedFlowId.value) {
    // Flow mode: flow events provide their own progress indicators
    if (isFlowRunning.value && currentExecutionId.value) {
      ws.value!.send(JSON.stringify({ type: 'flow_user_response', session_id: activeSessionId.value, options: { execution_id: currentExecutionId.value, response: content } }))
    } else {
      isFlowRunning.value = true; flowEvents.value = []; flowStore.clearFlowEvents()
      const er = await fetch(`${API_BASE}/api/flows/${selectedFlowId.value}/execute`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ session_id: activeSessionId.value }) })
      const ed = await er.json(); currentExecutionId.value = ed.data?.execution_id
      ws.value!.send(JSON.stringify({ type: 'flow_start', session_id: activeSessionId.value, model: selectedModelId.value, messages: wsMsgs, stream: true, options: { flow_id: selectedFlowId.value, execution_id: currentExecutionId.value, thinking_level: thinkLevel.value }, attachments: atts }))
    }
  } else {
    // Agent/chat mode: create thinking placeholder, transitioned to content on first chunk
    beginStream()
    const msgType = atts.length > 0 ? 'chat' : 'agent'
    ws.value!.send(JSON.stringify({ type: msgType, session_id: activeSessionId.value, model: selectedModelId.value, messages: wsMsgs, stream: true, options: { thinking_level: thinkLevel.value }, attachments: atts }))
  }
}

function stopFlow() {
  if (ws.value && currentExecutionId.value) ws.value.send(JSON.stringify({ type: 'flow_stop', session_id: activeSessionId.value, options: { execution_id: currentExecutionId.value } }))
  isFlowRunning.value = false; isStreaming.value = false; currentExecutionId.value = null; messages.value.push({ role: 'tool', content: '⏹ 流程已停止' })
}

// File upload handling
async function onFilesSelected(files: File[]) {
  for (const file of files) {
    try {
      const formData = new FormData()
      formData.append('file', file)
      const r = await fetch(`${API_BASE}/api/upload`, { method: 'POST', body: formData })
      const d = await r.json()
      if (d.data) {
        pendingAttachments.value.push(d.data as Attachment)
      }
    } catch (e) { console.error('upload failed', e) }
  }
}

// Flow events
function handleFlowEvent(event: FlowEvent) {
  flowEvents.value.push(event); flowStore.addFlowEvent(event)
  switch (event.type) {
    case 'flow_node_start': isFlowRunning.value = true; messages.value.push({ role: 'tool', content: `📌 [${event.node_label}] 开始...` }); break
    case 'flow_node_chunk': if (event.content) { const l = messages.value[messages.value.length - 1]; if (l && l.role === 'assistant') l.content += event.content; else messages.value.push({ role: 'assistant', content: event.content }) }; break
    case 'flow_node_done': messages.value.push({ role: 'tool', content: `✅ [${event.node_label}] 完成${event.content ? ' (' + event.content + ')' : ''}` }); break
    case 'flow_waiting_user': messages.value.push({ role: 'tool', content: `⏳ ${event.message || '请确认继续...'}` }); isStreaming.value = false; break
    case 'flow_complete': currentExecutionId.value = null; isFlowRunning.value = false; isStreaming.value = false; messages.value.push({ role: 'tool', content: '🎉 流程执行完成！' }); break
    case 'flow_error': currentExecutionId.value = null; isFlowRunning.value = false; isStreaming.value = false; messages.value.push({ role: 'tool', content: '❌ ' + (event.message || 'Flow error') }); break
  }
}

onMounted(() => { fetchSessions(); connectWebSocket(); flowStore.fetchFlows() })
</script>

<style scoped>
* { margin: 0; padding: 0; box-sizing: border-box; }
.app { display: flex; height: 100vh; background: #fff; }
.main-area { flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden; }

/* Flow bar */
.flow-bar {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 20px; background: #eef2ff; border-bottom: 1px solid #c7d2fe;
  flex-shrink: 0;
}
.flow-badge { font-size: 13px; color: #4f46e5; font-weight: 600; }
.flow-action { font-size: 12px; color: #64748b; background: none; border: none; cursor: pointer; }
.flow-action:hover { color: #ef4444; }
.flow-stop {
  background: #ef4444; color: #fff; border: none;
  padding: 4px 12px; border-radius: 6px; cursor: pointer;
  font-size: 12px; margin-left: auto;
}
.flow-stop:hover { background: #dc2626; }
</style>
