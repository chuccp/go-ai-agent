import { type ReactNode, useMemo, useRef, useEffect, useState, useCallback, createContext, useContext } from 'react'
import { AssistantRuntimeProvider, useLocalRuntime, useAssistantRuntime } from '@assistant-ui/react'
import { createSimpleWebSocketAdapter, type QueueEvent } from './WebSocketAdapter'

// ── Message Queue Context ──

export interface QueuedMessage {
  id: number
  status: 'queued' | 'consumed'
}

interface MessageQueueState {
  queuedMessages: QueuedMessage[]
  queueCount: number
}

const MessageQueueContext = createContext<MessageQueueState>({ queuedMessages: [], queueCount: 0 })

export function useMessageQueue() {
  return useContext(MessageQueueContext)
}

interface Props {
  children: ReactNode
  sessionId: number
}

export function ChatRuntimeProvider({ children, sessionId }: Props) {
  const wsRef = useRef<WebSocket | null>(null)
  const sessionIdRef = useRef(sessionId)
  sessionIdRef.current = sessionId

  // 消息队列状态
  const [queuedMessages, setQueuedMessages] = useState<QueuedMessage[]>([])

  const handleQueueEvent = useCallback((evt: QueueEvent) => {
    switch (evt.type) {
      case 'message_queued':
        setQueuedMessages(prev => [...prev, { id: evt.message_id, status: 'queued' }])
        break
      case 'message_consumed':
        setQueuedMessages(prev =>
          prev.map(m => m.id === evt.message_id ? { ...m, status: 'consumed' as const } : m)
        )
        // 消费后短暂保留然后移除
        setTimeout(() => {
          setQueuedMessages(prev => prev.filter(m => m.id !== evt.message_id))
        }, 600)
        break
      case 'message_sent':
        // 消息已被立即处理，无需排队显示
        break
    }
  }, [])

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

  const adapter = useMemo(
    () => createSimpleWebSocketAdapter(getWs, getSessionId, handleQueueEvent),
    [handleQueueEvent],
  )

  const runtime = useLocalRuntime(adapter)

  const queueState = useMemo<MessageQueueState>(() => ({
    queuedMessages,
    queueCount: queuedMessages.filter(m => m.status === 'queued').length,
  }), [queuedMessages])

  return (
    <MessageQueueContext.Provider value={queueState}>
      <AssistantRuntimeProvider runtime={runtime}>
        <SessionResetter sessionId={sessionId} />
        {children}
      </AssistantRuntimeProvider>
    </MessageQueueContext.Provider>
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
