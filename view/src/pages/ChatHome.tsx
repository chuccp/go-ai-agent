import { useState, useEffect, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import i18n from '@/i18n'
import { API_BASE, IS_DESKTOP } from '@/constants'
import { useFlowStore } from '@/stores/flowStore'
import { MyRuntimeProvider, type ThreadMessageLike, type PendingFlow } from '@/components/assistant-ui/MyRuntimeProvider'
import { Thread } from '@/components/assistant-ui/Thread'

interface Session { id: number; title: string; flow_id?: number | null; created_at: string }
interface LLMModel { id: string; name: string; provider: string; model: string; category: string; is_default: boolean }

export default function ChatHome() {
  const { t } = useTranslation()
  const { flows, fetchFlows } = useFlowStore()

  // State
  const [sessions, setSessions] = useState<Session[]>([])
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null)
  const [sessionMessages, setSessionMessages] = useState<readonly ThreadMessageLike[]>([])
  const [sessionKey, setSessionKey] = useState(0)
  const [models, setModels] = useState<LLMModel[]>([])
  const [selectedModelId, setSelectedModelId] = useState('')
  const [thinkLevel, setThinkLevel] = useState('off')
  const [selectedFlowId, setSelectedFlowId] = useState<number | null>(null)
  const [showNewSession, setShowNewSession] = useState(false)
  const [newSessionFlowId, setNewSessionFlowId] = useState<number | null>(null)
  const [sessionToDelete, setSessionToDelete] = useState<number | null>(null)
  const [showSettings, setShowSettings] = useState(false)
  const [pendingFlow, setPendingFlow] = useState<PendingFlow | null>(null)

  // WebSocket
  const wsRef = useRef<WebSocket | null>(null)

  // Fetch data
  const fetchSessions = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/sessions`)
      const data = await res.json()
      setSessions(Array.isArray(data.data) ? data.data : [])
    } catch {}
  }, [])

  const fetchModels = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/models?category=llm`)
      const data = await res.json()
      const m = Array.isArray(data.data?.models) ? data.data.models : (Array.isArray(data.data) ? data.data : [])
      setModels(m)
      const def = m.find((x: LLMModel) => x.is_default)
      if (def) setSelectedModelId(def.id)
    } catch {}
  }, [])

  useEffect(() => { fetchSessions(); fetchModels(); fetchFlows() }, [fetchSessions, fetchModels, fetchFlows])

  // WebSocket connection (web mode only; desktop uses IPC)
  useEffect(() => {
    if (IS_DESKTOP) return
    let ws: WebSocket | null = null
    let reconnecting = false
    const connect = () => {
      let wsUrl: string
      if (API_BASE) {
        // Wails dev mode: API_BASE is a full URL like http://localhost:19009
        wsUrl = API_BASE.replace(/^http/, 'ws') + '/ws/chat'
      } else {
        const proto = location.protocol === 'https:' ? 'wss' : 'ws'
        const isDev = location.port === '5173'
        const wsHost = isDev ? `${location.hostname}:19009` : location.host
        wsUrl = `${proto}://${wsHost}/ws/chat`
      }
      ws = new WebSocket(wsUrl)
      wsRef.current = ws
      ws.onclose = () => {
        if (!reconnecting) {
          reconnecting = true
          setTimeout(() => { reconnecting = false; connect() }, 2000)
        }
      }
      ws.onerror = () => {}
    }
    connect()
    return () => { ws?.close() }
  }, [])

  // Session management
  const createSession = useCallback(async (flowId?: number | null) => {
    try {
      const res = await fetch(`${API_BASE}/api/sessions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: flowId ? 'Flow Session' : 'New Chat', flow_id: flowId || undefined }),
      })
      const data = await res.json()
      if (data.data) {
        const s = data.data as Session
        setSessions(prev => [s, ...prev])
        setActiveSessionId(s.id)
        setSelectedFlowId(flowId || null)
      }
    } catch {}
    setShowNewSession(false)
    setNewSessionFlowId(null)
  }, [])

  const deleteSession = useCallback(async (id: number) => {
    try {
      await fetch(`${API_BASE}/api/sessions/${id}`, { method: 'DELETE' })
      setSessions(prev => prev.filter(s => s.id !== id))
      if (activeSessionId === id) setActiveSessionId(null)
    } catch {}
    setSessionToDelete(null)
  }, [activeSessionId])

  const selectSession = useCallback(async (id: number) => {
    setActiveSessionId(id)
    setSelectedFlowId(null)
    setSessionMessages([])
    // Fetch message history for this session
    try {
      const res = await fetch(`${API_BASE}/api/sessions/${id}/messages`)
      const data = await res.json()
      const msgs = Array.isArray(data.data) ? data.data : []
      setSessionMessages(msgs.map((m: any) => ({ role: m.role, content: [{ type: 'text' as const, text: String(m.content ?? '') }] })))
    } catch {
      setSessionMessages([])
    }
    setSessionKey(k => k + 1)
  }, [])

  // Adapter getters
  const getWs = useCallback(() => wsRef.current, [])
  const getSessionId = useCallback(() => activeSessionId, [activeSessionId])
  const getModelId = useCallback(() => selectedModelId, [selectedModelId])
  const getThinkLevel = useCallback(() => thinkLevel, [thinkLevel])
  const getFlowId = useCallback(() => selectedFlowId, [selectedFlowId])
  const getPendingFlow = useCallback(() => pendingFlow, [pendingFlow])

  const handleSessionCreated = useCallback((sessionId: number) => {
    setActiveSessionId(sessionId)
    setSessions(prev => {
      if (prev.some(s => s.id === sessionId)) return prev
      return [{ id: sessionId, title: 'New Chat', created_at: new Date().toISOString() }, ...prev]
    })
  }, [])

  const handleFlowWaiting = useCallback((executionId: number, question: string) => {
    setPendingFlow({ executionId, question })
  }, [])

  const handleFlowResponseSent = useCallback(() => {
    setPendingFlow(null)
  }, [])

  const handleFlowEnded = useCallback(() => {
    setPendingFlow(null)
  }, [])

  return (
    <MyRuntimeProvider
      key={sessionKey}
      getWs={getWs}
      sessionId={getSessionId}
      modelId={getModelId}
      thinkLevel={getThinkLevel}
      flowId={getFlowId}
      onSessionCreated={handleSessionCreated}
      pendingFlow={getPendingFlow}
      onFlowResponseSent={handleFlowResponseSent}
      onFlowWaiting={handleFlowWaiting}
      onFlowEnded={handleFlowEnded}
      initialMessages={sessionMessages}
    >
      <div style={{ display: 'flex', height: '100vh', overflow: 'hidden', background: '#fff' }}>
        {/* ── Sidebar ── */}
        <div style={{ width: 260, minWidth: 260, background: '#f8f9fa', borderRight: '1px solid #e8eaed', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <div style={{ padding: '16px 16px 8px' }}>
            <button onClick={() => setShowNewSession(true)} style={{ width: '100%', padding: '10px 16px', borderRadius: 24, border: '1px solid #dadce0', background: '#fff', color: '#1a73e8', fontSize: 14, fontWeight: 500, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M12 4v16M4 12h16" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/></svg>
              {t('chat.newChat')}
            </button>
          </div>
          <div style={{ flex: 1, overflowY: 'auto', padding: '4px 8px' }}>
            {sessions.map(session => (
              <div key={session.id} onClick={() => selectSession(session.id)} style={{ padding: '10px 14px', borderRadius: 20, cursor: 'pointer', marginBottom: 2, background: session.id === activeSessionId ? '#e8f0fe' : 'transparent', color: session.id === activeSessionId ? '#1a73e8' : '#3c4043', fontSize: 14, fontWeight: session.id === activeSessionId ? 500 : 400, display: 'flex', alignItems: 'center', justifyContent: 'space-between', transition: 'background 0.15s' }}>
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{session.title}</span>
                <button onClick={e => { e.stopPropagation(); session.id === activeSessionId ? setSessionToDelete(session.id) : deleteSession(session.id) }} style={{ background: 'none', border: 'none', color: '#80868b', cursor: 'pointer', fontSize: 16, padding: '0 4px', lineHeight: 1, flexShrink: 0, opacity: 0.6 }}>×</button>
              </div>
            ))}
            {sessions.length === 0 && <div style={{ padding: '24px 14px', textAlign: 'center', fontSize: 13, color: '#80868b' }}>{t('common.noData')}</div>}
          </div>
          <div style={{ padding: '8px', borderTop: '1px solid #e8eaed' }}>
            <a href="#/models" style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderRadius: 20, color: '#3c4043', fontSize: 14, textDecoration: 'none' }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 2a4 4 0 014 4c0 .73-.2 1.41-.54 2A4 4 0 0118 12c0 1.1-.45 2.1-1.17 2.83A4 4 0 0116 22h-1a3 3 0 01-3-3v-1m0 0V8m0 10a3 3 0 01-3 3H8a4 4 0 01-.83-7.17A4 4 0 016 12a4 4 0 012.54-3.72A4 4 0 018 6a4 4 0 014-4"/>
              </svg>
              {t('nav.modelManager')}
            </a>
            <a href="#/designer" style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderRadius: 20, color: '#3c4043', fontSize: 14, textDecoration: 'none' }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="3" width="6" height="6" rx="1"/><rect x="15" y="3" width="6" height="6" rx="1"/><rect x="9" y="15" width="6" height="6" rx="1"/>
                <path d="M6 9v3a3 3 0 003 3h6a3 3 0 003-3V9M12 12v3"/>
              </svg>
              {t('nav.flowDesigner')}
            </a>
            <button onClick={() => setShowSettings(true)} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderRadius: 20, color: '#3c4043', fontSize: 14, background: 'none', border: 'none', cursor: 'pointer', width: '100%', textAlign: 'left' }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"/>
              </svg>
              {t('common.settings')}
            </button>
          </div>
        </div>

        {/* ── Main Area with Assistant UI Thread ── */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <Thread
            models={models}
            selectedModelId={selectedModelId}
            thinkLevel={thinkLevel}
            onModelChange={setSelectedModelId}
            onThinkChange={setThinkLevel}
          />
        </div>
      </div>

      {/* ── New Session Modal ── */}
      {showNewSession && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }} onClick={() => { setShowNewSession(false); setNewSessionFlowId(null) }}>
          <div style={{ background: '#fff', borderRadius: 20, padding: 24, width: 420, maxWidth: '90vw', boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 16, color: '#202124' }}>{t('chat.newSession')}</div>
            <div style={{ fontSize: 13, color: '#5f6368', marginBottom: 12 }}>{t('chat.selectMode')}</div>
            <div onClick={() => setNewSessionFlowId(null)} style={{ padding: '14px 16px', borderRadius: 14, cursor: 'pointer', marginBottom: 8, border: newSessionFlowId === null ? '2px solid #1a73e8' : '1px solid #dadce0', background: newSessionFlowId === null ? '#e8f0fe' : '#fff', transition: 'all 0.15s' }}>
              <div style={{ fontSize: 14, fontWeight: 600, color: '#202124' }}>💬 {t('chat.freeChat')}</div>
              <div style={{ fontSize: 12, color: '#5f6368', marginTop: 2 }}>{t('chat.freeChatDesc')}</div>
            </div>
            {flows.map(flow => (
              <div key={flow.id} onClick={() => setNewSessionFlowId(flow.id)} style={{ padding: '14px 16px', borderRadius: 14, cursor: 'pointer', marginBottom: 8, border: newSessionFlowId === flow.id ? '2px solid #1a73e8' : '1px solid #dadce0', background: newSessionFlowId === flow.id ? '#e8f0fe' : '#fff', transition: 'all 0.15s' }}>
                <div style={{ fontSize: 14, fontWeight: 600, color: '#202124' }}>⚡ {flow.name}</div>
                {flow.description && <div style={{ fontSize: 12, color: '#5f6368', marginTop: 2 }}>{flow.description}</div>}
              </div>
            ))}
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 16 }}>
              <button onClick={() => { setShowNewSession(false); setNewSessionFlowId(null) }} style={{ padding: '10px 24px', borderRadius: 20, border: '1px solid #dadce0', background: '#fff', color: '#3c4043', fontSize: 14, fontWeight: 500, cursor: 'pointer' }}>{t('common.cancel')}</button>
              <button onClick={() => createSession(newSessionFlowId)} style={{ padding: '10px 24px', borderRadius: 20, border: 'none', background: '#1a73e8', color: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer' }}>{t('common.confirm')}</button>
            </div>
          </div>
        </div>
      )}

      {/* ── Settings Modal ── */}
      {showSettings && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }} onClick={() => setShowSettings(false)}>
          <div style={{ background: '#fff', borderRadius: 20, padding: 0, width: 480, maxWidth: '90vw', boxShadow: '0 8px 32px rgba(0,0,0,0.12)', overflow: 'hidden' }} onClick={e => e.stopPropagation()}>
            <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid #e8eaed' }}>
              <div style={{ fontSize: 18, fontWeight: 600, color: '#202124' }}>{t('common.settings')}</div>
            </div>
            <div style={{ padding: '16px 24px 24px' }}>
              <div style={{ marginBottom: 20 }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#5f6368', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 12 }}>General</div>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 0', borderBottom: '1px solid #f1f3f4' }}>
                  <div>
                    <div style={{ fontSize: 14, fontWeight: 500, color: '#202124' }}>Language</div>
                    <div style={{ fontSize: 12, color: '#5f6368', marginTop: 2 }}>Display language for the interface</div>
                  </div>
                  <select value={i18n.language} onChange={e => i18n.changeLanguage(e.target.value)} style={{ padding: '8px 12px', borderRadius: 12, border: '1px solid #dadce0', fontSize: 14, outline: 'none', background: '#fff', color: '#202124', cursor: 'pointer', minWidth: 140 }}>
                    <option value="en">English</option>
                    <option value="zh">简体中文</option>
                    <option value="zh-TW">繁體中文</option>
                    <option value="ja">日本語</option>
                  </select>
                </div>
              </div>
            </div>
            <div style={{ padding: '12px 24px', borderTop: '1px solid #e8eaed', display: 'flex', justifyContent: 'flex-end' }}>
              <button onClick={() => setShowSettings(false)} style={{ padding: '10px 24px', borderRadius: 20, border: 'none', background: '#1a73e8', color: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer' }}>Done</button>
            </div>
          </div>
        </div>
      )}

      {/* ── Delete Confirmation Modal ── */}
      {sessionToDelete !== null && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }} onClick={() => setSessionToDelete(null)}>
          <div style={{ background: '#fff', borderRadius: 20, padding: 24, maxWidth: 340, boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 14, color: '#202124', marginBottom: 20 }}>{t('chat.confirmDeleteSession')}</div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button onClick={() => setSessionToDelete(null)} style={{ padding: '10px 24px', borderRadius: 20, border: '1px solid #dadce0', background: '#fff', color: '#3c4043', fontSize: 14, fontWeight: 500, cursor: 'pointer' }}>{t('common.cancel')}</button>
              <button onClick={() => { if (sessionToDelete !== null) deleteSession(sessionToDelete) }} style={{ padding: '10px 24px', borderRadius: 20, border: 'none', background: '#d93025', color: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer' }}>{t('common.delete')}</button>
            </div>
          </div>
        </div>
      )}
    </MyRuntimeProvider>
  )
}
