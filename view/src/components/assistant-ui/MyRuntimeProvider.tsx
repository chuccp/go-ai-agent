import { type ReactNode, useMemo } from 'react'
import { AssistantRuntimeProvider, useLocalRuntime } from '@assistant-ui/react'
import { createStreamingWebSocketAdapter } from './WebSocketAdapter'
import { createIpcAdapter } from './IpcAdapter'
import { IS_DESKTOP } from '@/constants'

interface ThreadMessageLike {
  role: 'assistant' | 'user' | 'system'
  content: string
}

interface RuntimeProviderProps {
  children: ReactNode
  getWs?: () => WebSocket | null
  sessionId: () => number | null
  modelId: () => string
  thinkLevel: () => string
  flowId: () => number | null
  onSessionCreated?: (sessionId: number) => void
  initialMessages?: readonly ThreadMessageLike[]
}

export function MyRuntimeProvider({
  children,
  getWs,
  sessionId,
  modelId,
  thinkLevel,
  flowId,
  onSessionCreated,
  initialMessages,
}: RuntimeProviderProps) {
  const adapter = useMemo(
    () => {
      if (IS_DESKTOP) {
        return createIpcAdapter({
          sessionId,
          modelId,
          thinkLevel,
          flowId,
          onSessionCreated,
        })
      }
      return createStreamingWebSocketAdapter({
        getWs: getWs!,
        sessionId,
        modelId,
        thinkLevel,
        flowId,
        onSessionCreated,
      })
    },
    [getWs, sessionId, modelId, thinkLevel, flowId, onSessionCreated]
  )

  const runtime = useLocalRuntime(adapter, { initialMessages })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      {children}
    </AssistantRuntimeProvider>
  )
}
