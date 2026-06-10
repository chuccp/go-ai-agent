<template>
  <div class="message-list" ref="listRef">
    <div v-if="messages.length === 0" class="empty-state">
      <div class="empty-icon">💬</div>
      <div class="empty-title">开始新的对话</div>
      <div class="empty-sub">选择模型后发送消息，AI 将在这里回复</div>
    </div>

    <div
      v-for="(msg, index) in messages"
      :key="index"
      :class="['msg-row', msg.role]"
    >
      <!-- Avatar -->
      <div :class="['avatar', msg.role]">
        {{ avatarText(msg.role) }}
      </div>

      <!-- Content -->
      <div class="bubble-wrap">
        <div v-if="msg.role === 'tool'" class="tool-label">{{ msg.content }}</div>
        <div
          v-else
          :class="['bubble', msg.role]"
          v-html="renderContent(msg.content)"
        ></div>
        <router-link
          v-if="msg.flowId"
          :to="'/designer/' + msg.flowId"
          class="flow-link"
        >查看流程 →</router-link>
      </div>
    </div>

    <!-- Streaming cursor -->
    <div v-if="isStreaming && lastMsgIsAssistant" class="msg-row assistant">
      <div class="avatar assistant">AI</div>
      <div class="streaming-cursor">
        <span class="dot"></span><span class="dot"></span><span class="dot"></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { marked } from 'marked'

interface Message {
  role: string
  content: string
  flowId?: number
}

const props = defineProps<{
  messages: Message[]
  isStreaming: boolean
}>()

const listRef = ref<HTMLElement | null>(null)

const lastMsgIsAssistant = computed(() => {
  const len = props.messages.length
  return len > 0 && props.messages[len - 1].role === 'assistant'
})

watch(
  () => [props.messages, props.messages.length],
  () => {
    requestAnimationFrame(() => {
      if (listRef.value) {
        listRef.value.scrollTop = listRef.value.scrollHeight
      }
    })
  },
  { deep: true }
)

function avatarText(role: string): string {
  if (role === 'user') return 'You'
  if (role === 'assistant') return 'AI'
  return 'Sys'
}

marked.setOptions({ breaks: true, gfm: true })

function renderContent(text: string): string {
  if (!text) return ''
  if (text.includes('```') || text.includes('**') || text.includes('##') || text.includes('* ')) {
    try {
      return marked.parse(text) as string
    } catch { /* fall through */ }
  }
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>')
}
</script>

<style scoped>
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 20px 16px;
  background: #ffffff;
  scroll-behavior: smooth;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #94a3b8;
  gap: 8px;
}
.empty-icon { font-size: 48px; margin-bottom: 8px; }
.empty-title { font-size: 18px; font-weight: 600; color: #64748b; }
.empty-sub { font-size: 13px; }

/* Message row */
.msg-row {
  display: flex;
  gap: 10px;
  margin-bottom: 24px;
  max-width: 820px;
  margin-left: auto;
  margin-right: auto;
}
.msg-row.user {
  flex-direction: row-reverse;
}

/* Avatar */
.avatar {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
  color: #fff;
}
.avatar.user { background: #6366f1; }
.avatar.assistant { background: #10b981; }
.avatar.tool { background: #f59e0b; }

/* Bubble */
.bubble-wrap {
  flex: 1;
  min-width: 0;
}
.msg-row.user .bubble-wrap {
  text-align: right;
}
.bubble {
  display: inline-block;
  max-width: 100%;
  padding: 12px 16px;
  border-radius: 16px;
  font-size: 14px;
  line-height: 1.65;
  word-break: break-word;
  text-align: left;
}
.bubble.user {
  background: #eef2ff;
  color: #1e293b;
  border-bottom-right-radius: 4px;
}
.bubble.assistant {
  background: #f8fafc;
  color: #1e293b;
  border: 1px solid #e2e8f0;
  border-bottom-left-radius: 4px;
}
.bubble.user :deep(p) { margin: 0; }
.bubble.assistant :deep(p) { margin: 0 0 8px 0; }
.bubble.assistant :deep(p:last-child) { margin-bottom: 0; }

/* Code blocks in markdown */
.bubble.assistant :deep(pre) {
  background: #1e293b;
  color: #e2e8f0;
  padding: 14px 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
  font-size: 13px;
  line-height: 1.5;
}
.bubble.assistant :deep(code) {
  font-family: 'SF Mono', 'Menlo', 'Monaco', monospace;
  font-size: 13px;
}
.bubble.assistant :deep(p code) {
  background: #e2e8f0;
  color: #1e293b;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.bubble.assistant :deep(ul), .bubble.assistant :deep(ol) {
  padding-left: 20px;
  margin: 4px 0;
}
.bubble.assistant :deep(blockquote) {
  border-left: 3px solid #6366f1;
  padding-left: 12px;
  color: #64748b;
  margin: 8px 0;
}

/* Tool messages */
.tool-label {
  display: inline-block;
  padding: 8px 12px;
  font-size: 12px;
  color: #64748b;
  background: #f1f5f9;
  border-radius: 8px;
  border: 1px dashed #cbd5e1;
  line-height: 1.5;
}

/* Flow link */
.flow-link {
  display: inline-block;
  margin-top: 6px;
  color: #6366f1;
  text-decoration: none;
  font-size: 12px;
  font-weight: 500;
}
.flow-link:hover { text-decoration: underline; }

/* Streaming cursor */
.streaming-cursor {
  display: flex;
  gap: 4px;
  align-items: center;
  padding: 12px 16px;
}
.streaming-cursor .dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #94a3b8;
  animation: bounce 1.4s infinite both;
}
.streaming-cursor .dot:nth-child(2) { animation-delay: 0.2s; }
.streaming-cursor .dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.4; }
  40% { transform: scale(1); opacity: 1; }
}
</style>
