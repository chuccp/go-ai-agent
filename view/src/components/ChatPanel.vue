<template>
  <div :class="['chat-panel', { 'is-empty': messages.length === 0 }]">
    <div v-if="messages.length === 0" class="center-area">
      <div class="welcome">
        <div class="welcome-logo">✨</div>
        <div class="welcome-title">开始新的对话</div>
        <div class="welcome-sub">选择模型后发送消息，AI 将在这里回复</div>
      </div>
      <ChatInput
        :disabled="isStreaming" :center-mode="true" :accept-files="acceptFiles"
        :models="dbModels" :model-id="selectedModelId" :think-level="thinkLevel"
        @send="$emit('send', $event)"
        @upload="(files: File[]) => $emit('upload', files)"
        @model-change="onModelSelect"
        @think-change="(l: string) => { thinkLevel = l; $emit('think-change', l) }"
      />
    </div>
    <template v-else>
      <MessageList :messages="messages" :is-streaming="isStreaming" />
      <ChatInput
        :disabled="isStreaming" :center-mode="false" :accept-files="acceptFiles"
        :models="dbModels" :model-id="selectedModelId" :think-level="thinkLevel"
        @send="$emit('send', $event)"
        @upload="(files: File[]) => $emit('upload', files)"
        @model-change="onModelSelect"
        @think-change="(l: string) => { thinkLevel = l; $emit('think-change', l) }"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { API_BASE } from '@/constants'
import MessageList from './MessageList.vue'
import ChatInput from './ChatInput.vue'

interface Message { role: string; content: string }
interface DBModel {
  id: string; name: string; provider: string; model: string
  category: string; is_default: boolean; supports_multimodal?: boolean
  thinking_level?: string
}

defineProps<{ messages: Message[]; isStreaming: boolean }>()
const emit = defineEmits<{
  send: [content: string]
  upload: [files: File[]]
  'model-change': [model: string]
  'think-change': [level: string]
}>()

const dbModels = ref<DBModel[]>([])
const selectedModelId = ref('')
const thinkLevel = ref('off')
const hasOCRModel = ref(false)

const acceptFiles = computed(() => {
  const model = dbModels.value.find(m => m.id === selectedModelId.value)
  const multimodal = model?.supports_multimodal || false
  const files: string[] = []
  if (multimodal) files.push('image/*')
  if (hasOCRModel.value) files.push('image/*')
  files.push('.txt,.md,.csv')
  files.push('.pdf,.docx,.doc')
  return files.join(',')
})

async function loadModels() {
  try {
    const res = await fetch(`${API_BASE}/api/models?category=llm`)
    const data = await res.json()
    const models = data.data?.models || []
    dbModels.value = models
    const def = models.find((m: DBModel) => m.is_default)
    const sel = def || (models.length > 0 ? models[0] : null)
    if (sel) {
      selectedModelId.value = sel.id
      thinkLevel.value = sel.thinking_level || 'off'
      emit('model-change', sel.id)
      emit('think-change', thinkLevel.value)
    }
    checkOCR()
  } catch { /* ignore */ }
}

async function checkOCR() {
  try {
    const res = await fetch(`${API_BASE}/api/models?category=ocr`)
    const data = await res.json()
    const ocrModels = data.data?.models || []
    hasOCRModel.value = ocrModels.length > 0
  } catch { hasOCRModel.value = false }
}

function onModelSelect(id: string) {
  selectedModelId.value = id
  const model = dbModels.value.find(m => m.id === id)
  thinkLevel.value = model?.thinking_level || 'off'
  emit('model-change', id)
  emit('think-change', thinkLevel.value)
}

onMounted(loadModels)
</script>

<style scoped>
.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
.chat-panel.is-empty { justify-content: center; }

.center-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 20px;
  gap: 24px;
}
.welcome { text-align: center; color: #64748b; }
.welcome-logo { font-size: 40px; margin-bottom: 12px; }
.welcome-title { font-size: 22px; font-weight: 600; }
.welcome-sub { font-size: 14px; color: #94a3b8; margin-top: 6px; }
</style>
