<template>
  <div class="chat-panel">
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
    <MessageList :messages="messages" :is-streaming="isStreaming" />
    <ChatInput :disabled="isStreaming" @send="$emit('send', $event)" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import MessageList from './MessageList.vue'
import ChatInput from './ChatInput.vue'

interface Message { role: string; content: string }
interface DBModel {
  id: string; name: string; provider: string; model: string
  category: string; is_default: boolean
}

defineProps<{ messages: Message[]; isStreaming: boolean }>()
const emit = defineEmits<{
  send: [content: string]
  'model-change': [model: string]
}>()

const API_BASE = ''
const dbModels = ref<DBModel[]>([])
const selectedModelId = ref('')

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
  } catch { /* ignore */ }
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
.top-bar {
  display: flex;
  align-items: center;
  padding: 10px 20px;
  border-bottom: 1px solid #f1f5f9;
  background: #fff;
  flex-shrink: 0;
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
</style>
