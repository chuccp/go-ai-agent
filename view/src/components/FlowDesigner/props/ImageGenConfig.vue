<template>
  <div class="node-config">
    <label>模型</label>
    <select v-model="cfg.model" @change="emitUpdate">
      <option v-for="m in imageModels" :key="m.id" :value="m.id">{{ m.name }}</option>
    </select>
    <label>提示词</label>
    <textarea v-model="cfg.prompt" @input="emitUpdate" rows="3" placeholder="图片描述，支持 {{节点名.output}}..."></textarea>
    <label>生成数量</label>
    <input type="number" v-model.number="cfg.max_number" @input="emitUpdate" min="1" max="10" />
    <label>比例</label>
    <select v-model="cfg.scale" @change="emitUpdate">
      <option value="">默认</option>
      <option value="1:1">1:1</option>
      <option value="16:9">16:9</option>
      <option value="9:16">9:16</option>
      <option value="4:3">4:3</option>
      <option value="3:4">3:4</option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { API_BASE } from '@/constants'
const imageModels = ref<{ id: string; name: string }[]>([])
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ model: '', prompt: '', max_number: 1, scale: '' })

watch(() => props.config, (c) => {
  cfg.model = c.model || ''
  cfg.prompt = c.prompt || ''
  cfg.max_number = c.max_number || 1
  cfg.scale = c.scale || ''
}, { immediate: true })

function emitUpdate() { emit('update', { ...cfg }) }

onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE}/api/models?category=image`)
    const data = await res.json()
    imageModels.value = data.data?.models || []
  } catch {}
})
</script>
