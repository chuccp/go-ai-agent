import { type Ref } from 'vue'

interface Message {
  id?: number; role: string; content: string; reasoning?: string
  flowId?: number; attachments?: any[]; status?: 'thinking' | 'streaming'
}

const MSG_THINKING = '思考中...'

export function useChatStream(messages: Ref<Message[]>, isStreaming: Ref<boolean>, isFlowRunning: Ref<boolean>) {
  let streamId = 0
  let reasoningBuf = ''

  function nextId() { return ++streamId }

  function findStream(): Message | undefined {
    return messages.value.find(m => m.id === streamId && m.status != null)
  }

  function popStale() {
    const m = findStream()
    if (!m) return
    const idx = messages.value.indexOf(m)
    if (m.status === 'thinking') {
      messages.value.splice(idx, 1)
    } else {
      m.status = undefined
    }
    streamId = 0
    reasoningBuf = ''
  }

  function beginStream() {
    streamId = nextId()
    reasoningBuf = ''
    messages.value.push({ id: streamId, role: 'assistant', content: MSG_THINKING, status: 'thinking' })
  }

  function addReasoning(text: string) {
    reasoningBuf += reasoningBuf ? '\n\n' + text : text
  }

  function acceptContent(text: string) {
    const m = findStream()
    if (m) {
      if (reasoningBuf) m.reasoning = reasoningBuf
      m.content = text
      m.status = 'streaming'
      reasoningBuf = ''
    } else {
      streamId = nextId()
      messages.value.push({ id: streamId, role: 'assistant', content: text })
    }
  }

  function appendDelta(delta: string) {
    const last = messages.value[messages.value.length - 1]
    if (last && last.role === 'assistant' && last.status !== 'thinking') {
      last.content += delta
    } else {
      streamId = nextId()
      messages.value.push({ id: streamId, role: 'assistant', content: delta })
    }
  }

  function setThinkingText(text: string) {
    const m = findStream()
    if (m) { m.content = text }
    else { beginStream(); messages.value[messages.value.length - 1].content = text }
  }

  function endStream() {
    isStreaming.value = false
    isFlowRunning.value = false
    popStale()
  }

  return { beginStream, addReasoning, acceptContent, appendDelta, setThinkingText, endStream }
}
