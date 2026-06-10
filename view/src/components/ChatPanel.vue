<template>
  <div :class="['chat-panel', { 'is-empty': messages.length === 0 }]">
    <div class="top-bar">
      <div class="model-picker">
        <span class="model-label">模型</span>
        <select :value="selectedModelId" @change="onModelSelect">
          <option v-for="m in dbModels" :key="m.id" :value="m.id">
            {{ m.name }}
          </option>
        </select>
      </div>
    </div>
    <div v-if="messages.length === 0" class="center-area">
      <div class="welcome">
        <div class="welcome-logo">✨</div>
        <div class="welcome-title">开始新的对话</div>
        <div class="welcome-sub">选择模型后发送消息，AI 将在这里回复</div>
      </div>
      <ChatInput :disabled="isStreaming" :center-mode="true" :accept-files="acceptFiles" @send="$emit('send', $event)" @upload="(files: File[]) => $emit('upload', files)" />
    </div>
    <template v-else>
      <MessageList :messages="messages" :is-streaming="isStreaming" :thinking="thinking" />
      <ChatInput :disabled="isStreaming" :center-mode="false" :accept-files="acceptFiles" @send="$emit('send', $event)" @upload="(files: File[]) => $emit('upload', files)" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import MessageList from './MessageList.vue'
import ChatInput from './ChatInput.vue'

interface Message { role: string; content: string }
interface DBModel {
  id: string; name: string; provider: string; model: string
  category: string; is_default: boolean; supports_multimodal?: boolean
}

defineProps<{ messages: Message[]; isStreaming: boolean; thinking: string }>()
const emit = defineEmits<{
  send: [content: string]
  upload: [files: File[]]
  'model-change': [model: string]
}>()

const API_BASE = ''
const dbModels = ref<DBModel[]>([])
const selectedModelId = ref('')
const hasOCRModel = ref(false)

// Accept files based on current model capabilities
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
    if (def) {
      selectedModelId.value = def.id
      emit('model-change', def.id)
    } else if (models.length > 0) {
      selectedModelId.value = models[0].id
      emit('model-change', models[0].id)
    }
    // Check OCR model availability
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

function onModelSelect(e: Event) {
  const val = (e.target as HTMLSelectElement).value
  selectedModelId.value = val
  emit('model-change', val)
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
.chat-panel.is-empty {
  justify-content: center;
}

.top-bar {
  display: flex;
  align-items: center;
  padding: 10px 20px;
  border-bottom: 1px solid #f1f5f9;
  background: #fff;
  flex-shrink: 0;
}
.is-empty .top-bar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  border-bottom: none;
}

.model-picker {
  display: flex;
  align-items: center;
  gap: 8px;
}
.model-label {
  font-size: 12px;
  color: #94a3b8;
  font-weight: 500;
}
.model-picker select {
  padding: 5px 10px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 13px;
  background: #f8fafc;
  color: #334155;
  cursor: pointer;
  outline: none;
  min-width: 200px;
}
.model-picker select:focus {
  border-color: #6366f1;
}

/* Center area for empty state */
.center-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 20px;
  gap: 24px;
}
.welcome {
  text-align: center;
  color: #64748b;
}
.welcome-logo { font-size: 40px; margin-bottom: 12px; }
.welcome-title { font-size: 22px; font-weight: 600; }
.welcome-sub { font-size: 14px; color: #94a3b8; margin-top: 6px; }
</style>
