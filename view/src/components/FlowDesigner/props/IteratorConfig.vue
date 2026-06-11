<template>
  <div class="node-config">
    <label>数据来源</label>
    <select v-model="cfg.items_key" @change="emitUpdate">
      <option value="">-- 选择节点 --</option>
      <optgroup v-for="(group, gIdx) in upstreamOpts" :key="gIdx" :label="group.node.label">
        <option v-for="opt in group.fields" :key="opt.key" :value="group.node.label">{{ group.node.label }}.output</option>
      </optgroup>
    </select>
    <span class="hint">按顺序逐项传给下游节点，失败跳过继续下一项</span>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch, computed } from 'vue'
import type { FlowNode, FlowEdge } from '@/types/flow'

const props = defineProps<{ config: Record<string, any>; nodes?: FlowNode[]; edges?: FlowEdge[]; nodeId?: number }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ items_key: '' })

watch(() => props.config, (c) => { cfg.items_key = c.items_key || '' }, { immediate: true })

const upstreamOpts = computed(() => {
  if (!props.nodes || !props.edges || !props.nodeId) return []
  const incoming = props.edges.filter(e => e.target_node_id === props.nodeId)
  const sources = incoming.length > 0
    ? incoming.map(e => props.nodes!.find(n => n.id === e.source_node_id)).filter(Boolean) as FlowNode[]
    : props.nodes.filter(n => n.id !== props.nodeId && n.type !== 'start' && n.type !== 'end')
  return sources.map(n => ({ node: n, fields: [{ key: n.label, label: `${n.label}.output` }] }))
})

function emitUpdate() { emit('update', { ...cfg }) }
</script>
