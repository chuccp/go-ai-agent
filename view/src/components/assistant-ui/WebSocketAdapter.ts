import type { ChatModelAdapter, ChatModelRunResult } from '@assistant-ui/react'

import type { PendingFlow } from './MyRuntimeProvider'

interface WebSocketAdapterOptions {
  getWs: () => WebSocket | null
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

export function createStreamingWebSocketAdapter(opts: WebSocketAdapterOptions): ChatModelAdapter {
  return {
    async *run({ messages, abortSignal }): AsyncGenerator<ChatModelRunResult> {
      console.log('[WS] ═══ run() called ═══')
      console.log('[WS] messages count:', messages.length)
      console.log('[WS] messages:', JSON.stringify(messages, null, 2))

      const ws = opts.getWs()
      console.log('[WS] ws state:', ws?.readyState, '(1=OPEN)')

      if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.log('[WS] WebSocket not connected!')
        yield { content: [{ type: 'text', text: 'WebSocket not connected' }] }
        return
      }

      // Get the last user message
      const lastMsg = messages[messages.length - 1]
      console.log('[WS] lastMsg:', JSON.stringify(lastMsg, null, 2))

      if (!lastMsg) {
        console.log('[WS] No last message!')
        return
      }

      console.log('[WS] lastMsg.role:', lastMsg.role)

      if (lastMsg.role !== 'user') {
        console.log('[WS] Last message is not user, role:', lastMsg.role)
        return
      }

      // Extract text content
      let textContent = ''
      console.log('[WS] lastMsg.content type:', typeof lastMsg.content)
      console.log('[WS] lastMsg.content:', JSON.stringify(lastMsg.content, null, 2))

      if (Array.isArray(lastMsg.content)) {
        console.log('[WS] Content is array, length:', lastMsg.content.length)
        for (let i = 0; i < lastMsg.content.length; i++) {
          const part = lastMsg.content[i] as any
          console.log(`[WS] content[${i}]:`, JSON.stringify(part))
          if (part.type === 'text' && part.text) {
            textContent += part.text
            console.log('[WS] Found text:', part.text)
          }
        }
      } else if (typeof lastMsg.content === 'string') {
        textContent = lastMsg.content
        console.log('[WS] Content is string:', textContent)
      }

      console.log('[WS] Final textContent:', JSON.stringify(textContent))

      if (!textContent.trim()) {
        console.log('[WS] ⚠️ EMPTY CONTENT - not sending!')
        return
      }

      const pending = opts.pendingFlow ? opts.pendingFlow() : null
      const flowId = opts.flowId()

      // Build payload — backend expects "messages" array, not standalone "content"
      let payload: any
      if (pending?.executionId) {
        payload = {
          type: 'flow_user_response',
          session_id: opts.sessionId(),
          options: { execution_id: pending.executionId, response: textContent.trim() },
        }
        if (opts.onFlowResponseSent) opts.onFlowResponseSent()
      } else {
        payload = {
          type: flowId ? 'flow_start' : 'agent',
          session_id: opts.sessionId(),
          messages: [{ role: 'user', content: textContent.trim() }],
          model: opts.modelId(),
          stream: true,
          options: { think_level: opts.thinkLevel() },
        }
        if (flowId) {
          payload.options.flow_id = flowId
        }
      }

      console.log('[WS] ✅ Sending payload:', JSON.stringify(payload))

      // Streaming via async queue
      const queue: ChatModelRunResult[] = []
      let done = false
      let accumulatedText = ''
      let toolCalls: any[] = []

      const flush = () => {
        const content: any[] = []
        if (accumulatedText) {
          content.push({ type: 'text', text: accumulatedText })
        }
        content.push(...toolCalls)
        if (content.length > 0) {
          queue.push({ content: [...content] })
        }
      }

      const handler = (evt: MessageEvent) => {
        try {
          const msg = JSON.parse(evt.data)

          switch (msg.type) {
            case 'session_created':
              if (msg.session_id && opts.onSessionCreated) {
                opts.onSessionCreated(msg.session_id)
              }
              break

            case 'flow_started':
              accumulatedText += `▶️ ${msg.message || 'Flow started'}\n`
              flush()
              break

            case 'flow_node_start':
            case 'flow_node_done':
              // Optional low-noise progress; skip by default.
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
              return

            case 'flow_complete':
              if (msg.message || msg.content) {
                accumulatedText += `\n✅ ${msg.message || msg.content}`
                flush()
              }
              if (opts.onFlowEnded) opts.onFlowEnded()
              done = true
              return

            case 'flow_error':
              accumulatedText += `\n❌ ${msg.message || msg.content || 'Flow error'}`
              flush()
              if (opts.onFlowEnded) opts.onFlowEnded()
              done = true
              return

            case 'chunk':
              if (msg.done) {
                done = true
                return
              }
              if (msg.content) {
                accumulatedText += msg.content
              }
              break

            case 'tool_call':
              toolCalls.push({
                type: 'tool-call' as const,
                toolCallId: `tc-${Date.now()}-${toolCalls.length}`,
                toolName: msg.message || msg.name || 'tool',
                args: {},
                argsText: '',
              })
              break

            case 'tool_result':
              if (msg.message) {
                accumulatedText += `\n\n📋 ${msg.message}`
              }
              break

            case 'error':
              accumulatedText += `\n\n❌ ${msg.message || msg.content || 'Error'}`
              done = true
              return
          }

          flush()
        } catch {}
      }

      ws.addEventListener('message', handler)

      abortSignal.addEventListener('abort', () => {
        ws.removeEventListener('message', handler)
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'stop' }))
        }
        done = true
      })

      ws.send(JSON.stringify(payload))

      while (!done) {
        if (queue.length > 0) {
          yield queue.shift()!
        } else {
          await new Promise(r => setTimeout(r, 50))
        }
      }

      while (queue.length > 0) {
        yield queue.shift()!
      }

      ws.removeEventListener('message', handler)
    },
  }
}
