<template>
  <div class="node-config">
    <label>检查字段</label>
    <input v-model="cfg.field" @input="emitUpdate" placeholder="如: 节点名.output" />
    <label>判断方式</label>
    <select v-model="cfg.operator" @change="emitUpdate">
      <option value="contains">包含</option>
      <option value="equals">等于</option>
      <option value="not_empty">不为空</option>
    </select>
    <template v-if="cfg.operator !== 'not_empty'">
      <label>比较值</label>
      <input v-model="cfg.value" @input="emitUpdate" placeholder="比较的目标值" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ field: '', operator: 'contains', value: '' })

watch(() => props.config, (c) => {
  cfg.field = c.field || ''
  cfg.operator = c.operator || 'contains'
  cfg.value = c.value || ''
}, { immediate: true })

function emitUpdate() { emit('update', { ...cfg }) }
</script>
