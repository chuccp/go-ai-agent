import type { ChatModelAdapter, ChatModelRunResult } from '@assistant-ui/react'
import type { PendingFlow, PendingQuestion } from './MyRuntimeProvider'

// Wails runtime globals — injected into the webview by Wails, not available in web mode.
const wailsRuntime = (window as any).runtime
const wailsEventsOn = (name: string, cb: (...args: any[]) => void) => wailsRuntime?.EventsOn(name, cb) ?? (() => {})
const wailsEventsOff = (...names: string[]) => { if (wailsRuntime) wailsRuntime.EventsOff(...names) }
const wailsApp = () => (window as any).go?.main?.App

interface IpcAdapterOptions {
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
}

export function createIpcAdapter(opts: IpcAdapterOptions): ChatModelAdapter {
  return {
    async *run({ messages, abortSignal }): AsyncGenerator<ChatModelRunResult> {
      const lastMsg = messages[messages.length - 1]
      if (!lastMsg || lastMsg.role !== 'user') return

      // Extract text content
      let textContent = ''
      if (Array.isArray(lastMsg.content)) {
        for (const part of lastMsg.content as any[]) {
          if (part.type === 'text' && part.text) {
            textContent += part.text
          }
        }
      } else if (typeof lastMsg.content === 'string') {
        textContent = lastMsg.content
      }

      if (!textContent.trim()) return

      const app = wailsApp()
      if (!app) {
        yield { content: [{ type: 'text', text: 'IPC not available (not running in Wails desktop)' }] }
        return
      }

      const pending = opts.pendingFlow ? opts.pendingFlow() : null
      const pendingQ = opts.pendingQuestion ? opts.pendingQuestion() : null
      let activeExecutionId = pending?.executionId ?? 0
      let activeSessionId = opts.sessionId() ?? 0

      if (pendingQ) {
        // User answered a previous ask_user question — send the reply via IPC.
        const answersJSON = JSON.stringify(extractAnswersIPC(messages, pendingQ))
        const raw: string = await app.QuestionReply(pendingQ.id, answersJSON)
        let resp: any = {}
        try { resp = JSON.parse(raw) } catch {}
        if (resp.error) {
          yield { content: [{ type: 'text', text: `Question error: ${resp.error}` }] }
          return
        }
        if (opts.onQuestionAnswered) opts.onQuestionAnswered()
        // The agent is still blocked on the tool; it will resume and emit
        // chunk events. We don't yield anything here — the event listeners
        // below will pick up the agent's subsequent output.
      } else if (pending?.executionId) {
        // Continue a paused flow via IPC bridge
        const raw: string = await app.FlowRespond(pending.executionId, textContent.trim())
        let resp: any = {}
        try { resp = JSON.parse(raw) } catch {}
        if (resp.error) {
          yield { content: [{ type: 'text', text: `Flow error: ${resp.error}` }] }
          if (opts.onFlowEnded) opts.onFlowEnded()
          return
        }
        if (opts.onFlowResponseSent) opts.onFlowResponseSent()
      } else {
        // Normal agent chat via Wails IPC
        const sessionId = opts.sessionId() ?? 0
        const modelId = opts.modelId()
        const thinkLevel = opts.thinkLevel()
        const flowId = opts.flowId() ?? 0

        const rawResult: string = await app.AgentChat(sessionId, modelId, textContent.trim(), thinkLevel, flowId)
        let result: any = {}
        try {
          result = JSON.parse(rawResult)
        } catch {
          yield { content: [{ type: 'text', text: String(rawResult) }] }
          return
        }

        if (result.error) {
          yield { content: [{ type: 'text', text: `Error: ${result.error}` }] }
          return
        }

        const newSessionId: number = result.session_id
        if (newSessionId && opts.onSessionCreated && !sessionId) {
          opts.onSessionCreated(newSessionId)
        }

        // Use the actual session ID from the response for event listening
        // Don't wait for React state update which is async
        activeSessionId = newSessionId || sessionId
      }

      // Streaming via async queue
      const queue: ChatModelRunResult[] = []
      let done = false
      let accumulatedText = ''
      const toolCalls: any[] = []

      // Use the actual session ID (may have been updated above for new sessions)
      const eventPrefix = `chat:${activeSessionId}:`

      const flush = () => {
        const parts: any[] = []
        if (accumulatedText) {
          parts.push({ type: 'text', text: accumulatedText })
        }
        parts.push(...toolCalls)
        if (parts.length > 0) {
          queue.push({ content: parts })
        }
      }

      const unsubChunk = wailsEventsOn(`${eventPrefix}chunk`, (event: any) => {
        if (event.content) {
          accumulatedText += event.content
        }
        if (event.reasoning) {
          accumulatedText += event.reasoning
        }
        flush()
        if (event.done) {
          done = true
        }
      })

      const unsubToolCall = wailsEventsOn(`${eventPrefix}tool_call`, (event: any) => {
        toolCalls.push({
          type: 'tool-call' as const,
          toolCallId: `tc-${Date.now()}-${toolCalls.length}`,
          toolName: event.tool_name || 'tool',
          args: {},
          argsText: event.tool_input || '',
        })
        flush()
      })

      const unsubToolResult = wailsEventsOn(`${eventPrefix}tool_result`, (event: any) => {
        if (event.message || event.tool_output) {
          accumulatedText += `\n\n📋 ${event.message || event.tool_output}`
        }
        flush()
      })

      const unsubError = wailsEventsOn(`${eventPrefix}error`, (event: any) => {
        accumulatedText += `\n\n❌ ${event.message || event.content || 'Error'}`
        done = true
      })

      const unsubSessionCreated = wailsEventsOn(`${eventPrefix}session_created`, (event: any) => {
        if (event.session_id && opts.onSessionCreated && !opts.sessionId()) {
          opts.onSessionCreated(event.session_id)
        }
      })

      // In desktop mode flow events are delivered via Wails runtime events.
      const flowEventName = `chat:${activeSessionId}:flow_event`
      console.log('[IPC] subscribing to flow events on:', flowEventName)
      const unsubFlow = wailsEventsOn(flowEventName, (msg: any) => {
        console.log('[IPC] flow event received:', msg?.type, msg)
        try {
          if (msg.type === 'flow_started' && activeExecutionId === 0 && msg.execution_id) {
            activeExecutionId = Number(msg.execution_id)
          }
          if (activeExecutionId && msg.execution_id && Number(msg.execution_id) !== activeExecutionId) {
            return
          }
          switch (msg.type) {
            case 'flow_started':
              accumulatedText += `▶️ ${msg.message || 'Flow started'}\n`
              flush()
              break
            case 'flow_node_start':
              if (msg.node_label && msg.node_type) {
                accumulatedText += `\n⚙️ [${msg.node_label}] (${msg.node_type})`
                flush()
              }
              break
            case 'flow_node_chunk':
              // Flow's internal LLM streaming output
              if (msg.content) {
                accumulatedText += msg.content
                flush()
              }
              break
            case 'flow_node_done':
              // Node finished
              break
            case 'flow_waiting_user':
              if (msg.message) {
                accumulatedText += `\n${msg.message}`
              }
              flush()
              if (msg.execution_id && opts.onFlowWaiting) {
                opts.onFlowWaiting(Number(msg.execution_id), msg.message || '')
              }
              done = true
              break
            case 'flow_complete':
              if (msg.message || msg.content) {
                accumulatedText += `\n✅ ${msg.message || msg.content}`
                flush()
              }
              if (opts.onFlowEnded) opts.onFlowEnded()
              // Do NOT set done=true: the agent is still processing the tool
              // result and will emit its own chunk{done:true} to end the turn.
              break
            case 'flow_error':
              accumulatedText += `\n❌ ${msg.message || msg.content || 'Flow error'}`
              flush()
              if (opts.onFlowEnded) opts.onFlowEnded()
              // Do NOT set done=true: let the agent handle the error.
              break
          }
        } catch (e) { console.error('IPC flow event handler error:', e) }
      })

      // ask_user questions are delivered via a session-scoped Wails event.
      const questionEventName = `chat:${activeSessionId}:question_asked`
      const unsubQuestion = wailsEventsOn(questionEventName, (req: any) => {
        try {
          if (req && opts.onQuestionAsked) {
            opts.onQuestionAsked({
              id: Number(req.id),
              sessionId: Number(req.session_id ?? req.sessionId ?? activeSessionId),
              questions: req.questions || [],
            })
          }
          done = true
        } catch (e) { console.error('IPC question event handler error:', e) }
      })

      abortSignal.addEventListener('abort', () => {
        // Notify the backend to cancel the active agent chat
        if (activeSessionId > 0) {
          const app = wailsApp()
          if (app?.AgentStop) {
            try { app.AgentStop(activeSessionId) } catch {}
          }
        }
        done = true
      })

      // Yield loop
      while (!done || queue.length > 0) {
        if (queue.length > 0) {
          yield queue.shift()!
        } else if (!done) {
          await new Promise(r => setTimeout(r, 50))
        } else {
          // Done with empty queue: yield empty result so the library knows we finished
          if (!accumulatedText) {
            yield { content: [{ type: 'text', text: '' }] }
          }
          break
        }
      }

      // Cleanup
      unsubChunk()
      unsubToolCall()
      unsubToolResult()
      unsubError()
      unsubSessionCreated()
      unsubFlow()
      unsubQuestion()
      wailsEventsOff(
        `${eventPrefix}chunk`,
        `${eventPrefix}tool_call`,
        `${eventPrefix}tool_result`,
        `${eventPrefix}error`,
        `${eventPrefix}session_created`,
        flowEventName,
        questionEventName,
      )
    },
  }
}

// extractAnswersIPC builds the Answer payload for a QuestionReply IPC call
// from the user's latest text input (treated as a custom answer to the first
// question). A richer UI would render option buttons and build the array from
// selected labels.
function extractAnswersIPC(messages: readonly any[], pendingQ: PendingQuestion): string[][] {
  const lastMsg = messages[messages.length - 1]
  let text = ''
  if (Array.isArray(lastMsg?.content)) {
    for (const part of lastMsg.content) {
      if (part?.type === 'text' && part.text) text += part.text
    }
  } else if (typeof lastMsg?.content === 'string') {
    text = lastMsg.content
  }
  const answers: string[][] = []
  for (let i = 0; i < (pendingQ.questions?.length || 0); i++) {
    answers.push(i === 0 ? [text.trim()] : [])
  }
  return answers
}
