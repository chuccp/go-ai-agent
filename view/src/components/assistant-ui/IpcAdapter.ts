import type { ChatModelAdapter, ChatModelRunResult } from '@assistant-ui/react'
import type { PendingFlow } from './MyRuntimeProvider'

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
}

const WS_URL = 'ws://localhost:19009/ws/chat'

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
      let activeExecutionId = pending?.executionId ?? 0

      if (pending?.executionId) {
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
        if (newSessionId && opts.onSessionCreated && !opts.sessionId()) {
          opts.onSessionCreated(newSessionId)
        }
      }

      // Open a side WebSocket to receive flow events (desktop always runs local server on 19009)
      const ws = new WebSocket(WS_URL)
      let wsReady = false
      ws.onopen = () => { wsReady = true }
      ws.onerror = () => {}

      // Streaming via async queue
      const queue: ChatModelRunResult[] = []
      let done = false
      let accumulatedText = ''
      const toolCalls: any[] = []

      const newSessionId = opts.sessionId() ?? 0
      const eventPrefix = `chat:${newSessionId}:`

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
      })

      const unsubToolResult = wailsEventsOn(`${eventPrefix}tool_result`, (event: any) => {
        if (event.message || event.tool_output) {
          accumulatedText += `\n\n📋 ${event.message || event.tool_output}`
        }
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

      const wsHandler = (evt: MessageEvent) => {
        try {
          const msg = JSON.parse(evt.data)
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
              done = true
              break
            case 'flow_error':
              accumulatedText += `\n❌ ${msg.message || msg.content || 'Flow error'}`
              flush()
              if (opts.onFlowEnded) opts.onFlowEnded()
              done = true
              break
          }
        } catch {}
      }
      ws.addEventListener('message', wsHandler)

      abortSignal.addEventListener('abort', () => {
        done = true
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'stop' }))
        }
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
      wailsEventsOff(
        `${eventPrefix}chunk`,
        `${eventPrefix}tool_call`,
        `${eventPrefix}tool_result`,
        `${eventPrefix}error`,
        `${eventPrefix}session_created`,
      )
      ws.removeEventListener('message', wsHandler)
      ws.close()
    },
  }
}
