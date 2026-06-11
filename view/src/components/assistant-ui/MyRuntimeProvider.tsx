import { type ReactNode, useMemo } from 'react'
import { AssistantRuntimeProvider, useLocalRuntime } from '@assistant-ui/react'
import { createStreamingWebSocketAdapter } from './WebSocketAdapter'

interface RuntimeProviderProps {
  children: ReactNode
  getWs: () => WebSocket | null
  sessionId: () => number | null
  modelId: () => string
  thinkLevel: () => string
  flowId: () => number | null
  onSessionCreated?: (sessionId: number) => void
}

export function MyRuntimeProvider({
  children,
  getWs,
  sessionId,
  modelId,
  thinkLevel,
  flowId,
  onSessionCreated,
}: RuntimeProviderProps) {
  // Memoize the adapter to prevent runtime resets
  const adapter = useMemo(
    () => createStreamingWebSocketAdapter({
      getWs,
      sessionId,
      modelId,
      thinkLevel,
      flowId,
      onSessionCreated,
    }),
    [getWs, sessionId, modelId, thinkLevel, flowId, onSessionCreated]
  )

  const runtime = useLocalRuntime(adapter)

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      {children}
    </AssistantRuntimeProvider>
  )
}
