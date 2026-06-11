<template>
  <div class="node-config">
    <label>模型</label>
    <select v-model="cfg.model" @change="emitUpdate">
      <option v-for="m in videoModels" :key="m.id" :value="m.id">{{ m.name }}</option>
    </select>
    <label>提示词</label>
    <textarea v-model="cfg.prompt" @input="emitUpdate" rows="3" placeholder="视频描述，支持 {{节点名.output}}..."></textarea>
    <label>时长（秒）</label>
    <input type="number" v-model.number="cfg.duration" @input="emitUpdate" min="1" max="60" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { API_BASE } from '@/constants'
const videoModels = ref<{ id: string; name: string }[]>([])
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ model: '', prompt: '', duration: 5 })

watch(() => props.config, (c) => {
  cfg.model = c.model || ''
  cfg.prompt = c.prompt || ''
  cfg.duration = c.duration || 5
}, { immediate: true })

function emitUpdate() { emit('update', { ...cfg }) }

onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE}/api/models?category=video`)
    const data = await res.json()
    videoModels.value = data.data?.models || []
  } catch {}
})
</script>
