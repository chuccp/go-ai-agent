<template>
  <div class="message-list" ref="listRef">
    <div
      v-for="(msg, index) in messages"
      :key="index"
      :class="['msg-row', msg.role]"
    >
      <!-- Avatar -->
      <div v-if="msg.role === 'user'" class="avatar user">You</div>
      <div v-else-if="msg.role === 'assistant'" class="avatar assistant">AI</div>

      <!-- Content -->
      <div :class="['msg-content', msg.role]">
        <div v-if="msg.role === 'tool'" class="tool-msg">{{ msg.content }}</div>
        <div
          v-else
          :class="['bubble', msg.role]"
          v-html="renderContent(msg.content)"
        ></div>
        <!-- Attachments in user messages -->
        <div v-if="msg.role === 'user' && msg.attachments?.length" class="attachments">
          <div v-for="(att, i) in msg.attachments" :key="i" class="att-item">
            <span class="att-icon">{{ fileIcon(att.type) }}</span>
            <span class="att-name">{{ att.name }}</span>
          </div>
        </div>
        <router-link
          v-if="msg.flowId"
          :to="'/designer/' + msg.flowId"
          class="flow-link"
        >查看流程 →</router-link>
      </div>
    </div>

    <!-- Thinking indicator -->
    <div v-if="thinking" class="msg-row assistant">
      <div class="avatar assistant">AI</div>
      <div class="thinking-label">{{ thinking }}</div>
    </div>

    <!-- Streaming cursor -->
    <div v-if="isStreaming && lastMsgIsAssistant && !thinking" class="msg-row assistant">
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

interface Attachment {
  id?: string; name: string; type: string; size?: number; path?: string
}

interface Message {
  role: string
  content: string
  flowId?: number
  attachments?: Attachment[]
}

const props = defineProps<{
  messages: Message[]
  isStreaming: boolean
  thinking: string
}>()

const listRef = ref<HTMLElement | null>(null)

const lastMsgIsAssistant = computed(() => {
  const len = props.messages.length
  return len > 0 && props.messages[len - 1].role === 'assistant'
})

watch(
  () => [props.messages, props.messages.length, props.thinking],
  () => {
    requestAnimationFrame(() => {
      if (listRef.value) {
        listRef.value.scrollTop = listRef.value.scrollHeight
      }
    })
  },
  { deep: true }
)

marked.setOptions({ breaks: true, gfm: true })

function fileIcon(mime: string): string {
  if (mime.startsWith('image/')) return '🖼'
  if (mime.startsWith('text/')) return '📄'
  if (mime.includes('pdf')) return '📕'
  if (mime.includes('doc')) return '📝'
  return '📎'
}

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

/* Message row */
.msg-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  max-width: 820px;
  margin-left: auto;
  margin-right: auto;
}
.msg-row.user {
  flex-direction: row-reverse;
}

/* Avatar */
.avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
  color: #fff;
  margin-top: 2px;
}
.avatar.user { background: #6366f1; }
.avatar.assistant { background: #10b981; }

/* Message content wrapper */
.msg-content {
  flex: 1;
  min-width: 0;
}
.msg-content.user {
  display: flex;
  justify-content: flex-end;
}

/* Bubble */
.bubble {
  font-size: 14px;
  line-height: 1.7;
  word-break: break-word;
}
.bubble.user {
  display: inline-block;
  max-width: 100%;
  padding: 10px 16px;
  border-radius: 18px;
  background: #eef2ff;
  color: #1e293b;
  border-bottom-right-radius: 4px;
}
.bubble.assistant {
  /* Gemini-style: no box, just clean text */
  color: #1e293b;
  padding: 0;
}

.bubble.user :deep(p) { margin: 0; }
.bubble.assistant :deep(p) { margin: 0 0 10px 0; }
.bubble.assistant :deep(p:last-child) { margin-bottom: 0; }

/* Code blocks in markdown */
.bubble.assistant :deep(pre) {
  background: #1e293b;
  color: #e2e8f0;
  padding: 14px 16px;
  border-radius: 10px;
  overflow-x: auto;
  margin: 10px 0;
  font-size: 13px;
  line-height: 1.5;
}
.bubble.assistant :deep(code) {
  font-family: 'SF Mono', 'Menlo', 'Monaco', monospace;
  font-size: 13px;
}
.bubble.assistant :deep(p code) {
  background: #f1f5f9;
  color: #1e293b;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.bubble.assistant :deep(ul), .bubble.assistant :deep(ol) {
  padding-left: 20px;
  margin: 6px 0;
}
.bubble.assistant :deep(li) {
  margin-bottom: 4px;
}
.bubble.assistant :deep(blockquote) {
  border-left: 3px solid #6366f1;
  padding-left: 12px;
  color: #64748b;
  margin: 10px 0;
}
.bubble.assistant :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
  font-size: 13px;
}
.bubble.assistant :deep(th),
.bubble.assistant :deep(td) {
  border: 1px solid #e2e8f0;
  padding: 8px 12px;
  text-align: left;
}
.bubble.assistant :deep(th) {
  background: #f8fafc;
  font-weight: 600;
  color: #475569;
}

/* Attachment items in user messages */
.attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.msg-content.user .attachments {
  justify-content: flex-end;
}
.att-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: rgba(99,102,241,0.08);
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  color: #6366f1;
}
.att-icon { font-size: 13px; }
.att-name { max-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* Tool messages — subtle, no icon prefix */
.tool-msg {
  font-size: 12px;
  color: #94a3b8;
  padding: 4px 0;
  line-height: 1.5;
}

/* Thinking indicator */
.thinking-label {
  font-size: 13px;
  color: #94a3b8;
  padding: 6px 0;
  font-style: italic;
  animation: fadeInUp 0.2s ease;
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
  padding: 4px 0;
}
.streaming-cursor .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #cbd5e1;
  animation: bounce 1.4s infinite both;
}
.streaming-cursor .dot:nth-child(2) { animation-delay: 0.2s; }
.streaming-cursor .dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.4; }
  40% { transform: scale(1); opacity: 1; }
}

@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
