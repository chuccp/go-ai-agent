<template>
  <div class="flow-selector">
    <select :value="selectedFlowId ?? ''" @change="onChange">
      <option value="">⚡ 选择流程...</option>
      <option v-for="flow in flowStore.flows" :key="flow.id" :value="flow.id">{{ flow.name }}</option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useFlowStore } from '@/stores/flow'

defineProps<{ selectedFlowId: number | null }>()
const emit = defineEmits<{ select: [id: number | null] }>()
const flowStore = useFlowStore()

function onChange(e: Event) {
  const val = (e.target as HTMLSelectElement).value
  emit('select', val ? Number(val) : null)
}

onMounted(() => flowStore.fetchFlows())
</script>

<style scoped>
.flow-selector select {
  padding: 5px 10px; font-size: 12px; border: 1px solid #ddd; border-radius: 5px;
  background: #fafafa; color: #666; cursor: pointer; outline: none; min-width: 140px;
}
.flow-selector select:focus { border-color: #4a9eff; }
</style>
