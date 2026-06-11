<template>
  <div class="node-config">
    <label>模板</label>
    <textarea v-model="cfg.template" @input="emitUpdate" rows="4" placeholder="Go 模板语法，输出 JSON..."></textarea>
    <span class="hint">通过 .节点名.output 引用上游数据</span>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ template: '' })
watch(() => props.config, (c) => { cfg.template = c.template || '' }, { immediate: true })
function emitUpdate() { emit('update', { ...cfg }) }
</script>
