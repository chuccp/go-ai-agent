import type { ChatModelAdapter, ChatModelRunResult } from '@assistant-ui/react'

/**
 * Simple WebSocket adapter for the go-agent-sdk chat protocol.
 *
 * Protocol:
 *   Send:    { type: "chat",  message: "user text", session_id: 123 }
 *   Send:    { type: "stop" }
 *   Receive: { type: "chunk", content: "text", done: false, conversation_id: "..." }
 *   Receive: { type: "done",  done: true }
 *   Receive: { type: "error", message: "error text" }
 */

export function createSimpleWebSocketAdapter(
  getWs: () => WebSocket | null,
  getSessionId: () => number,
): ChatModelAdapter {
  return {
    async *run({ messages, abortSignal }): AsyncGenerator<ChatModelRunResult> {
      const ws = getWs()
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        yield { content: [{ type: 'text', text: '❌ WebSocket not connected. Is the server running on port 19009?' }] }
        return
      }

      // Extract text from the last user message
      const lastMsg = messages[messages.length - 1]
      if (!lastMsg || lastMsg.role !== 'user') return

      let textContent = ''
      if (Array.isArray(lastMsg.content)) {
        for (const part of lastMsg.content) {
          if (part.type === 'text' && part.text) textContent += part.text
        }
      } else if (typeof lastMsg.content === 'string') {
        textContent = lastMsg.content
      }

      if (!textContent.trim()) return

      // --- Streaming via event queue ---
      // Yield the FULL accumulated text each time (not deltas), so
      // assistant-ui sees progressive replacements of the same content block.
      const queue: ChatModelRunResult[] = []
      let done = false
      let fullText = ''

      const push = () => {
        if (fullText) {
          queue.push({ content: [{ type: 'text', text: fullText }] })
        }
      }

      const handler = (evt: MessageEvent) => {
        try {
          const msg = JSON.parse(evt.data)
          switch (msg.type) {
            case 'chunk':
              if (msg.content) {
                fullText += msg.content
                push()
              }
              break
            case 'done':
              done = true
              return
            case 'error':
              fullText += `\n\n❌ ${msg.message || 'Unknown error'}`
              push()
              done = true
              return
          }
        } catch { /* ignore parse errors */ }
      }

      ws.addEventListener('message', handler)

      // Handle abort (user clicks stop)
      abortSignal.addEventListener('abort', () => {
        ws.removeEventListener('message', handler)
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'stop' }))
        }
        done = true
      })

      // Send the message with session_id
      const sessionId = getSessionId()
      ws.send(JSON.stringify({ type: 'chat', message: textContent.trim(), session_id: sessionId }))

      // Yield results as they arrive
      while (!done) {
        if (queue.length > 0) {
          yield queue.shift()!
        } else {
          await new Promise(r => setTimeout(r, 50))
        }
      }

      // Drain remaining
      while (queue.length > 0) {
        yield queue.shift()!
      }

      ws.removeEventListener('message', handler)
    },
  }
}
