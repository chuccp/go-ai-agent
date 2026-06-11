import type { ChatModelAdapter, ChatModelRunResult } from '@assistant-ui/react'

interface WebSocketAdapterOptions {
  getWs: () => WebSocket | null
  sessionId: () => number | null
  modelId: () => string
  thinkLevel: () => string
  flowId: () => number | null
  onSessionCreated?: (sessionId: number) => void
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

      // Build payload — backend expects "messages" array, not standalone "content"
      const payload: any = {
        type: 'agent',
        session_id: opts.sessionId(),
        messages: [{ role: 'user', content: textContent.trim() }],
        model: opts.modelId(),
        stream: true,
        options: { think_level: opts.thinkLevel() },
      }

      const flowId = opts.flowId()
      if (flowId) {
        payload.options.flow_id = flowId
        payload.type = 'flow_start'
      }

      console.log('[WS] ✅ Sending payload:', JSON.stringify(payload))

      // Streaming via async queue
      const queue: ChatModelRunResult[] = []
      let done = false
      let accumulatedText = ''
      let toolCalls: any[] = []

      const handler = (evt: MessageEvent) => {
        try {
          const msg = JSON.parse(evt.data)

          switch (msg.type) {
            case 'session_created':
              if (msg.session_id && opts.onSessionCreated) {
                opts.onSessionCreated(msg.session_id)
              }
              break

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

          const content: any[] = []
          if (accumulatedText) {
            content.push({ type: 'text', text: accumulatedText })
          }
          content.push(...toolCalls)

          if (content.length > 0) {
            queue.push({ content: [...content] })
          }
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
