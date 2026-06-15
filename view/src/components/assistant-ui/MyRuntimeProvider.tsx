import { type ReactNode, useMemo } from 'react'
import { AssistantRuntimeProvider, useLocalRuntime } from '@assistant-ui/react'
import { createStreamingWebSocketAdapter } from './WebSocketAdapter'
import { createIpcAdapter } from './IpcAdapter'
import { IS_DESKTOP } from '@/constants'

export interface ThreadMessageLike {
  role: 'assistant' | 'user' | 'system'
  content: string | { type: 'text'; text: string }[]
}

export interface PendingFlow {
  executionId: number
  question?: string
}

interface RuntimeProviderProps {
  children: ReactNode
  getWs?: () => WebSocket | null
  sessionId: () => number | null
  modelId: () => string
  thinkLevel: () => string
  flowId: () => number | null
  onSessionCreated?: (sessionId: number) => void
  pendingFlow?: () => PendingFlow | null
  onFlowResponseSent?: () => void
  onFlowWaiting?: (executionId: number, question: string) => void
  onFlowEnded?: () => void
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
  pendingFlow,
  onFlowResponseSent,
  onFlowWaiting,
  onFlowEnded,
  initialMessages,
}: RuntimeProviderProps) {
  const adapter = useMemo(
    () => {
      const base = { sessionId, modelId, thinkLevel, flowId, onSessionCreated, pendingFlow, onFlowResponseSent, onFlowWaiting, onFlowEnded }
      if (IS_DESKTOP) {
        return createIpcAdapter(base)
      }
      return createStreamingWebSocketAdapter({ getWs: getWs!, ...base })
    },
    [getWs, sessionId, modelId, thinkLevel, flowId, onSessionCreated, pendingFlow, onFlowResponseSent, onFlowWaiting, onFlowEnded]
  )

  const runtime = useLocalRuntime(adapter, { initialMessages })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      {children}
    </AssistantRuntimeProvider>
  )
}
