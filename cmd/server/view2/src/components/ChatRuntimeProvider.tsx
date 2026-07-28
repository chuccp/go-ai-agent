import { type ReactNode, useMemo, useRef, useEffect } from 'react'
import { AssistantRuntimeProvider, useLocalRuntime } from '@assistant-ui/react'
import { createSimpleWebSocketAdapter } from './WebSocketAdapter'

interface Props { children: ReactNode }

export function ChatRuntimeProvider({ children }: Props) {
  const wsRef = useRef<WebSocket | null>(null)

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

  const adapter = useMemo(() => createSimpleWebSocketAdapter(getWs), [])

  const runtime = useLocalRuntime(adapter)

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      {children}
    </AssistantRuntimeProvider>
  )
}
