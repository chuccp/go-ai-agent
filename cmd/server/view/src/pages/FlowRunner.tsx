import { useEffect, useRef, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { API_BASE } from '@/constants'
import FlowForm from '@/components/FlowForm'
import { isIconFilename } from '@/types/flow'
import type { FlowDetail, FlowEvent as IFlowEvent, FormSchema } from '@/types/flow'

interface FlowEvent {
  type: string
  execution_id?: number
  node_label?: string
  node_type?: string
  content?: string
  message?: string
  status?: string
  form_schema?: FormSchema
}

// WebSocket-only mode.

export default function FlowRunner() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const nav = useNavigate()
  const flowId = parseInt(id || '0', 10)

  const [flow, setFlow] = useState<FlowDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [events, setEvents] = useState<FlowEvent[]>([])
  const [executionId, setExecutionId] = useState<number | null>(null)
  const [waitingUser, setWaitingUser] = useState(false)
  const [promptMessage, setPromptMessage] = useState('')
  const [response, setResponse] = useState('')
  const [running, setRunning] = useState(false)
  const [finished, setFinished] = useState(false)

  const wsRef = useRef<WebSocket | null>(null)
  const eventsEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    eventsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [events])

  useEffect(() => {
    if (!flowId) return
    fetch(`${API_BASE}/api/flows/${flowId}`)
      .then(r => r.json())
      .then(data => {
        setFlow(data.data || null)
        setLoading(false)
      })
      .catch(() => {
        setError(t('flow.failedToLoad'))
        setLoading(false)
      })
  }, [flowId])

  const connectWS = useCallback(() => {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const isDev = location.port === '5173'
    // In Vite dev mode connect directly to the Go backend (port 19009).
    // The Go server has CORS enabled for this.
    const wsHost = isDev ? `${location.hostname}:19009` : location.host
    const wsUrl = `${proto}://${wsHost}/ws/chat`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws
    return ws
  }, [])

  const pushEvent = useCallback((evt: FlowEvent) => {
    setEvents(prev => [...prev, evt])
    if (evt.execution_id) {
      setExecutionId(evt.execution_id)
    }
    if (evt.type === 'flow_waiting_user') {
      setWaitingUser(true)
      setPromptMessage(evt.message || 'Please provide input:')
      setResponse('')
    }
    if (evt.type === 'flow_complete' || evt.type === 'flow_error') {
      setRunning(false)
      setFinished(true)
      setWaitingUser(false)
    }
  }, [])

  const startFlowWS = useCallback((formValues?: Record<string, any>) => {
    setRunning(true)
    setFinished(false)
    setEvents([])

    const ws = connectWS()
    ws.onopen = () => {
      const payload: any = {
        type: 'flow_start',
        options: { flow_id: flowId },
      }
      if (formValues && Object.keys(formValues).length > 0) {
        payload.options.form_values = formValues
      }
      ws.send(JSON.stringify(payload))
    }

    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data) as FlowEvent
        pushEvent(msg)
        if (msg.type === 'flow_complete' || msg.type === 'flow_error') {
          ws.close()
        }
      } catch (e) { console.error('FlowRunner ws message parse error:', e) }
    }

    ws.onerror = () => {
      setError(t('flow.wsError'))
      setRunning(false)
    }

    ws.onclose = () => {
      if (!finished) setRunning(false)
    }
  }, [connectWS, flowId, finished, pushEvent])

  const startFlow = useCallback((formValues?: Record<string, any>) => {
    if (running) return
    startFlowWS(formValues)
  }, [running, startFlowWS])

  const sendUserResponse = useCallback(() => {
    if (!executionId || !wsRef.current) return
    wsRef.current.send(JSON.stringify({
      type: 'flow_user_response',
      options: { execution_id: executionId, response: response },
    }))
    setWaitingUser(false)
    setResponse('')
  }, [executionId, response])

  useEffect(() => {
    return () => {
      wsRef.current?.close()
    }
  }, [])

  if (loading) {
    return <div style={{ padding: 40, textAlign: 'center' }}>{t('common.loading')}</div>
  }
  if (error) {
    return <div style={{ padding: 40, textAlign: 'center', color: '#f04438' }}>{error}</div>
  }
  if (!flow) {
    return <div style={{ padding: 40, textAlign: 'center' }}>{t('flow.notFound')}</div>
  }

  const iconStr = flow.settings?.icon || flow.icon || '⚡'
  const title = flow.name || 'Untitled Flow'
  const hasForm = flow.form_schema && flow.form_schema.fields && flow.form_schema.fields.length > 0

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: '#f2f4f7' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 20px', background: '#fff', borderBottom: '0.5px solid rgba(16,24,40,0.08)' }}>
        <button onClick={() => nav('/designer')} style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#676f83', fontSize: 13 }}>← {t('common.back')}</button>
        <div style={{ fontSize: 20 }}>
          {isIconFilename(iconStr) ? (
            <img src={`${API_BASE}/api/flows/${flowId}/icon`} alt="icon" style={{ width: 28, height: 28, borderRadius: 6, objectFit: 'cover' }} />
          ) : iconStr}
        </div>
        <div style={{ fontSize: 16, fontWeight: 600, color: '#101828' }}>{title}</div>
        {flow.category && <span style={{ fontSize: 11, color: '#155aef', background: '#eff4ff', padding: '2px 8px', borderRadius: 10 }}>{flow.category}</span>}
      </div>

      {/* Main */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Left: form / run info */}
        <div style={{ width: 380, minWidth: 380, background: '#fff', borderRight: '0.5px solid rgba(16,24,40,0.08)', padding: 20, overflowY: 'auto' }}>
          {flow.description && (
            <div style={{ fontSize: 13, color: '#676f83', marginBottom: 16, lineHeight: 1.6 }}>{flow.description}</div>
          )}
          {!running && !finished && (
            hasForm ? (
              <FlowForm schema={flow.form_schema!} onSubmit={startFlow} disabled={running} />
            ) : (
              <button
                onClick={() => startFlow()}
                style={{
                  width: '100%', padding: '10px 20px', borderRadius: 8, border: 'none',
                  background: '#155aef', color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer',
                }}
              >
                {t('flow.runFlow')}
              </button>
            )
          )}
          {waitingUser && (
            <div style={{ marginTop: 16, padding: 14, borderRadius: 10, background: '#f9fafb', border: '1px solid #e2e8f0' }}>
              <div style={{ fontSize: 13, color: '#354052', marginBottom: 8 }}>{promptMessage}</div>
              <input
                value={response}
                onChange={e => setResponse(e.target.value)}
                style={{ width: '100%', padding: '8px 10px', border: '1px solid #d0d5dd', borderRadius: 8, fontSize: 13, marginBottom: 8 }}
                placeholder={t('flow.yourResponse')}
              />
              <button onClick={sendUserResponse} style={{ padding: '8px 16px', borderRadius: 8, border: 'none', background: '#155aef', color: '#fff', fontSize: 13, cursor: 'pointer' }}>{t('flow.sendResponse')}</button>
            </div>
          )}
        </div>

        {/* Right: execution log */}
        <div style={{ flex: 1, padding: 20, overflowY: 'auto' }}>
          <div style={{ maxWidth: 720, margin: '0 auto' }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: '#101828', marginBottom: 12 }}>{t('flow.executionLog')}</div>
            {events.length === 0 && (
              <div style={{ padding: 40, textAlign: 'center', color: '#98a2b3', fontSize: 13 }}>{t('flow.runToSeeProgress')}</div>
            )}
            {events.map((e, i) => (
              <div key={i} style={{ padding: '10px 12px', borderRadius: 8, background: '#fff', marginBottom: 8, border: '0.5px solid rgba(16,24,40,0.06)' }}>
                <div style={{ fontSize: 11, color: '#98a2b3', textTransform: 'uppercase', letterSpacing: 0.5 }}>{e.type}</div>
                {e.node_label && <div style={{ fontSize: 12, color: '#676f83', marginTop: 2 }}>{e.node_label} {e.node_type && `(${e.node_type})`}</div>}
                {e.content && <div style={{ fontSize: 13, color: '#101828', marginTop: 4 }}>{e.content}</div>}
                {e.message && <div style={{ fontSize: 13, color: '#354052', marginTop: 4 }}>{e.message}</div>}
              </div>
            ))}
            <div ref={eventsEndRef} />
          </div>
        </div>
      </div>
    </div>
  )
}
