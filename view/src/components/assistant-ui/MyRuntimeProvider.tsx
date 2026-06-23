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

// PendingQuestion mirrors the opencode Question.Request: the agent called
// ask_user and is blocked waiting for the user's answer. The frontend shows
// the question UI; when the user answers, a "question_reply" WS message is
// sent and the tool unblocks.
export interface PendingQuestion {
  id: number
  sessionId: number
  questions: Array<{
    question: string
    header: string
    options?: Array<{ label: string; description?: string }>
    multiple?: boolean
    custom?: boolean
  }>
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
  pendingQuestion?: () => PendingQuestion | null
  onQuestionAsked?: (q: PendingQuestion) => void
  onQuestionAnswered?: () => void
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
  pendingQuestion,
  onQuestionAsked,
  onQuestionAnswered,
  initialMessages,
}: RuntimeProviderProps) {
  const adapter = useMemo(
    () => {
      const base = { sessionId, modelId, thinkLevel, flowId, onSessionCreated, pendingFlow, onFlowResponseSent, onFlowWaiting, onFlowEnded, pendingQuestion, onQuestionAsked, onQuestionAnswered }
      if (IS_DESKTOP) {
        return createIpcAdapter(base)
      }
      return createStreamingWebSocketAdapter({ getWs: getWs!, ...base })
    },
    [getWs, sessionId, modelId, thinkLevel, flowId, onSessionCreated, pendingFlow, onFlowResponseSent, onFlowWaiting, onFlowEnded, pendingQuestion, onQuestionAsked, onQuestionAnswered]
  )

  const runtime = useLocalRuntime(adapter, { initialMessages })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      {children}
    </AssistantRuntimeProvider>
  )
}
