import { type ReactNode, useMemo, useRef, useEffect } from 'react'
import { AssistantRuntimeProvider, useLocalRuntime, useAssistantRuntime } from '@assistant-ui/react'
import { createSimpleWebSocketAdapter } from './WebSocketAdapter'

interface Props {
  children: ReactNode
  sessionId: number
}

export function ChatRuntimeProvider({ children, sessionId }: Props) {
  const wsRef = useRef<WebSocket | null>(null)
  const sessionIdRef = useRef(sessionId)
  sessionIdRef.current = sessionId

  // Connect WebSocket on mount
  useEffect(() => {
    let ws: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout>
    let mounted = true

    const connect = () => {
      if (!mounted) return
      const proto = location.protocol === 'https:' ? 'wss' : 'ws'
      const wsUrl = `${proto}://${location.hostname}:19009/ws/chat`
      ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onclose = () => {
        if (mounted) {
          reconnectTimer = setTimeout(connect, 2000)
        }
      }
      ws.onerror = () => {}
    }

    connect()

    return () => {
      mounted = false
      clearTimeout(reconnectTimer)
      ws?.close()
    }
  }, [])

  const getWs = () => wsRef.current
  const getSessionId = () => sessionIdRef.current

  const adapter = useMemo(() => createSimpleWebSocketAdapter(getWs, getSessionId), [])

  const runtime = useLocalRuntime(adapter)

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <SessionResetter sessionId={sessionId} />
      {children}
    </AssistantRuntimeProvider>
  )
}

/** Resets the thread when sessionId changes. */
function SessionResetter({ sessionId }: { sessionId: number }) {
  const prevRef = useRef(sessionId)
  const runtime = useAssistantRuntime()

  useEffect(() => {
    if (sessionId !== prevRef.current) {
      prevRef.current = sessionId
      runtime.threads.main.reset()
    }
  }, [sessionId, runtime])

  return null
}
