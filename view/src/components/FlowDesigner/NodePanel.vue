<template>
  <div class="node-panel">
    <h3>节点面板</h3>
    <div class="node-types">
      <div
        v-for="nt in nodeTypes"
        :key="nt.type"
        class="node-card"
        :style="{ borderLeftColor: nt.color }"
        draggable="true"
        @dragstart="onDragStart($event, nt)"
      >
        <span class="card-icon">{{ nt.icon }}</span>
        <div class="card-info">
          <span class="card-label">{{ nt.label }}</span>
          <span class="card-desc">{{ nt.desc }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { NodeTypeInfo, NodeType } from '@/types/flow'

const nodeTypes: (NodeTypeInfo & { icon: string; color: string })[] = [
  { type: 'start', label: '开始', description: '流程入口', icon: '▶', color: '#52c41a' },
  { type: 'end', label: '结束', description: '流程出口', icon: '⏹', color: '#ff4d4f' },
  { type: 'llm', label: 'LLM 调用', description: '大语言模型生成内容', icon: '🤖', color: '#4a9eff' },
  { type: 'user_input', label: '用户输入', description: '等待确认或输入', icon: '👤', color: '#faad14' },
  { type: 'split', label: '文本拆分', description: '按分隔符拆分为数组', icon: '✂', color: '#722ed1' },
  { type: 'condition', label: '条件分支', description: 'if/else 条件判断', icon: '🔀', color: '#fa8c16' },
  { type: 'transform', label: '数据变换', description: 'Go模板变换数据', icon: '⚙', color: '#13c2c2' },
  { type: 'for_each', label: 'ForEach 遍历', description: '逐项处理数组', icon: '🔁', color: '#eb2f96' },
  { type: 'script', label: '脚本节点', description: 'Python 自定义代码', icon: '🐍', color: '#2f54eb' },
  { type: 'iterator', label: '按序迭代', description: '逐项顺序处理，失败跳过', icon: '📋', color: '#eb2f96' },
  { type: 'loop', label: '循环执行', description: '重复执行子节点直到条件满足', icon: '🔄', color: '#1890ff' },
  { type: 'image_gen', label: '图片生成', description: '调用图片生成模型', icon: '🖼', color: '#36cfc9' },
  { type: 'audio_gen', label: '音频生成', description: '调用语音合成模型', icon: '🔊', color: '#9254de' },
  { type: 'video_gen', label: '视频生成', description: '调用视频生成模型', icon: '🎬', color: '#f759ab' },
]

function onDragStart(event: DragEvent, nt: typeof nodeTypes[0]) {
  event.dataTransfer!.setData('node-type', nt.type)
  event.dataTransfer!.setData('node-label', nt.label)
  event.dataTransfer!.effectAllowed = 'copy'
}
</script>

<style scoped>
.node-panel { width: 200px; background: #fff; border-right: 1px solid #e8e8e8; padding: 12px; overflow-y: auto; flex-shrink: 0; }
.node-panel h3 { font-size: 13px; color: #999; margin-bottom: 10px; font-weight: 500; text-transform: uppercase; letter-spacing: 1px; }
.node-types { display: flex; flex-direction: column; gap: 6px; }
.node-card { display: flex; align-items: center; gap: 8px; padding: 8px; border: 1px solid #e8e8e8; border-left: 3px solid; border-radius: 6px; cursor: grab; background: #fff; transition: all 0.15s; }
.node-card:hover { background: #fafafa; box-shadow: 0 2px 6px rgba(0,0,0,0.08); transform: translateX(2px); }
.node-card:active { cursor: grabbing; }
.card-icon { font-size: 18px; width: 24px; text-align: center; flex-shrink: 0; }
.card-info { display: flex; flex-direction: column; overflow: hidden; }
.card-label { font-size: 12px; font-weight: 600; color: #333; }
.card-desc { font-size: 10px; color: #999; margin-top: 1px; }
</style>
