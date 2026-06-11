import { useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

export interface ChatMessage {
  id: number
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  reasoning?: string
  status: 'done' | 'streaming' | 'thinking' | 'error'
  tool_name?: string
  tool_result?: string
  created_at: string
}

export function useChatStream() {
  const msgRef = useRef<ChatMessage[]>([])
  const { t } = useTranslation()

  const beginStream = useCallback((id: number) => {
    const msg: ChatMessage = {
      id, role: 'assistant', content: '', status: 'thinking',
      reasoning: '', created_at: new Date().toISOString(),
    }
    msgRef.current = [...msgRef.current, msg]
    return msgRef.current
  }, [])

  const setThinkingText = useCallback((text: string) => {
    const msgs = msgRef.current
    const last = msgs[msgs.length - 1]
    if (last && last.status === 'thinking') {
      last.content = text || t('chat.running')
      msgRef.current = [...msgs]
    }
    return msgRef.current
  }, [t])

  const addReasoning = useCallback((text: string) => {
    const msgs = msgRef.current
    const last = msgs[msgs.length - 1]
    if (last) {
      last.reasoning = (last.reasoning || '') + text
      last.status = 'streaming'
      msgRef.current = [...msgs]
    }
    return msgRef.current
  }, [])

  const acceptContent = useCallback(() => {
    const msgs = msgRef.current
    const last = msgs[msgs.length - 1]
    if (last) {
      last.status = 'streaming'
      msgRef.current = [...msgs]
    }
    return msgRef.current
  }, [])

  const appendDelta = useCallback((text: string) => {
    const msgs = msgRef.current
    const last = msgs[msgs.length - 1]
    if (last) {
      last.content += text
      last.status = 'streaming'
      msgRef.current = [...msgs]
    }
    return msgRef.current
  }, [])

  const endStream = useCallback(() => {
    const msgs = msgRef.current
    const last = msgs[msgs.length - 1]
    if (last) {
      last.status = 'done'
      msgRef.current = [...msgs]
    }
    return msgRef.current
  }, [])

  return { msgRef, beginStream, setThinkingText, addReasoning, acceptContent, appendDelta, endStream }
}
