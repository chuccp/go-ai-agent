<template>
  <div class="node-config">
    <label>模型</label>
    <select v-model="cfg.model" @change="emitUpdate">
      <option v-for="m in audioModels" :key="m.id" :value="m.id">{{ m.name }}</option>
    </select>
    <label>文本</label>
    <textarea v-model="cfg.text" @input="emitUpdate" rows="3" placeholder="要合成的文本，支持 {{节点名.output}}..."></textarea>
    <label>音色</label>
    <input v-model="cfg.voice" @input="emitUpdate" placeholder="音色名称" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { API_BASE } from '@/constants'
const audioModels = ref<{ id: string; name: string }[]>([])
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ model: '', text: '', voice: '' })

watch(() => props.config, (c) => {
  cfg.model = c.model || ''
  cfg.text = c.text || ''
  cfg.voice = c.voice || ''
}, { immediate: true })

function emitUpdate() { emit('update', { ...cfg }) }

onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE}/api/models?category=voice`)
    const data = await res.json()
    audioModels.value = data.data?.models || []
  } catch {}
})
</script>
