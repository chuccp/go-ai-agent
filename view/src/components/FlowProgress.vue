<template>
  <div v-if="isRunning || events.length > 0" class="flow-progress">
    <div class="header">
      <span class="title">流程执行进度</span>
      <span v-if="isRunning" class="running">执行中...</span>
      <span v-else class="done">完成</span>
    </div>
    <div class="nodes">
      <div
        v-for="(event, idx) in nodeEvents"
        :key="idx"
        class="node-event"
        :class="event.type"
      >
        <span class="dot" :class="event.dotClass"></span>
        <span class="label">{{ event.label }}</span>
        <span class="status">{{ event.statusText }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FlowEvent } from '@/types/flow'

const props = defineProps<{
  events: FlowEvent[]
  isRunning: boolean
}>()

interface NodeEventItem {
  type: string
  label: string
  statusText: string
  dotClass: string
}

const nodeEvents = computed<NodeEventItem[]>(() => {
  const map = new Map<string, NodeEventItem>()
  for (const e of props.events) {
    const key = e.node_label || e.type
    if (!map.has(key)) {
      map.set(key, { type: e.type, label: key, statusText: '', dotClass: '' })
    }
    const item = map.get(key)!
    if (e.type === 'flow_node_start' || e.type === 'flow_node_chunk') {
      item.dotClass = 'running'
      item.statusText = e.type === 'flow_node_chunk' ? '流式输出...' : '执行中...'
    } else if (e.type === 'flow_node_done') {
      item.dotClass = 'done'
      item.statusText = '✅'
    } else if (e.type === 'flow_error') {
      item.dotClass = 'error'
      item.statusText = '❌'
    } else if (e.type === 'flow_waiting_user') {
      item.dotClass = 'waiting'
      item.statusText = '⏳'
    } else if (e.type === 'flow_complete') {
      item.dotClass = 'done'
      item.statusText = '✅'
    }
  }
  return Array.from(map.values())
})
</script>

<style scoped>
.flow-progress {
  background: #fff;
  border-top: 1px solid #e0e0e0;
  padding: 10px 20px;
  max-height: 150px;
  overflow-y: auto;
  flex-shrink: 0;
}

.header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.title {
  font-size: 12px;
  font-weight: 600;
  color: #666;
}

.running { font-size: 11px; color: #4a9eff; }
.done { font-size: 11px; color: #52c41a; }

.nodes {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.node-event {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  padding: 2px 8px;
  background: #f5f5f5;
  border-radius: 10px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ccc;
}

.dot.running {
  background: #4a9eff;
  animation: pulse 1s infinite;
}

.dot.done { background: #52c41a; }
.dot.error { background: #ff4d4f; }
.dot.waiting { background: #faad14; animation: pulse 1s infinite; }

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.label { color: #333; }
.status { color: #999; }
</style>
