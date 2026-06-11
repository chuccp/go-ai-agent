<template>
  <div class="node-config">
    <label>最大循环次数</label>
    <input type="number" v-model.number="cfg.max_iterations" @input="emitUpdate" min="1" max="100" />
    <label>中断条件字段</label>
    <input v-model="cfg.break_field" @input="emitUpdate" placeholder="如：检测.output" />
    <label>中断判断</label>
    <select v-model="cfg.break_operator" @change="emitUpdate">
      <option value="contains">包含</option>
      <option value="equals">等于</option>
      <option value="not_empty">不为空</option>
    </select>
    <template v-if="cfg.break_operator !== 'not_empty'">
      <label>中断比较值</label>
      <input v-model="cfg.break_value" @input="emitUpdate" placeholder="比较的目标值" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ max_iterations: 10, break_field: '', break_operator: 'contains', break_value: '' })

watch(() => props.config, (c) => {
  cfg.max_iterations = c.max_iterations || 10
  cfg.break_field = c.break_field || ''
  cfg.break_operator = c.break_operator || 'contains'
  cfg.break_value = c.break_value || ''
}, { immediate: true })

function emitUpdate() { emit('update', { ...cfg }) }
</script>
