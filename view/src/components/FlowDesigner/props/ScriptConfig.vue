<template>
  <div class="node-config">
    <label>脚本代码</label>
    <textarea v-model="cfg.script" @input="emitUpdate" rows="6" placeholder="Python/Starlark 脚本..."></textarea>
    <span class="hint">可通过 ctx["节点名.output"] 访问上游数据</span>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ script: '' })
watch(() => props.config, (c) => { cfg.script = c.script || '' }, { immediate: true })
function emitUpdate() { emit('update', { ...cfg }) }
</script>
