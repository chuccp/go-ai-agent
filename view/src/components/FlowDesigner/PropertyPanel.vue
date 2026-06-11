<template>
  <div :class="['property-panel', { open: node }]">
    <div class="panel-header">
      <h3>节点属性</h3>
      <button class="close-btn" @click="$emit('close')">×</button>
    </div>
    <div v-if="node" class="props">
      <label>名称</label>
      <input v-model="node.label" @input="emitUpdate" />

      <label>类型</label>
      <select v-model="node.type" @change="emitUpdate">
        <option value="start">开始</option>
        <option value="end">结束</option>
        <option value="llm">LLM 调用</option>
        <option value="user_input">用户输入</option>
        <option value="split">文本拆分</option>
        <option value="condition">条件分支</option>
        <option value="transform">数据变换</option>
        <option value="for_each">批量处理</option>
        <option value="script">Script 脚本</option>
        <option value="iterator">按序迭代</option>
        <option value="loop">循环执行</option>
        <option value="image_gen">图片生成</option>
        <option value="audio_gen">音频生成</option>
        <option value="video_gen">视频生成</option>
      </select>

      <!-- Dispatch to per-type config components -->
      <LlmConfig v-if="node.type === 'llm'" :config="nodeConfig" @update="onChildUpdate" />
      <ImageGenConfig v-if="node.type === 'image_gen'" :config="nodeConfig" @update="onChildUpdate" />
      <AudioGenConfig v-if="node.type === 'audio_gen'" :config="nodeConfig" @update="onChildUpdate" />
      <VideoGenConfig v-if="node.type === 'video_gen'" :config="nodeConfig" @update="onChildUpdate" />
      <SplitConfig v-if="node.type === 'split'" :config="nodeConfig" :nodes="nodes" :edges="edges" :node-id="node.id" @update="onChildUpdate" />
      <ForEachConfig v-if="node.type === 'for_each'" :config="nodeConfig" :nodes="nodes" :edges="edges" :node-id="node.id" @update="onChildUpdate" />
      <IteratorConfig v-if="node.type === 'iterator'" :config="nodeConfig" :nodes="nodes" :edges="edges" :node-id="node.id" @update="onChildUpdate" />
      <LoopConfig v-if="node.type === 'loop'" :config="nodeConfig" @update="onChildUpdate" />
      <ConditionConfig v-if="node.type === 'condition'" :config="nodeConfig" @update="onChildUpdate" />
      <TransformConfig v-if="node.type === 'transform'" :config="nodeConfig" @update="onChildUpdate" />
      <ScriptConfig v-if="node.type === 'script'" :config="nodeConfig" @update="onChildUpdate" />
      <UserInputConfig v-if="node.type === 'user_input'" :config="nodeConfig" @update="onChildUpdate" />

      <button class="delete-btn" @click="$emit('delete:node', node!.id)">删除节点</button>
    </div>
    <div v-else class="empty">选择节点编辑属性</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FlowNode, FlowEdge } from '@/types/flow'

import LlmConfig from './props/LlmConfig.vue'
import ImageGenConfig from './props/ImageGenConfig.vue'
import AudioGenConfig from './props/AudioGenConfig.vue'
import VideoGenConfig from './props/VideoGenConfig.vue'
import SplitConfig from './props/SplitConfig.vue'
import ForEachConfig from './props/ForEachConfig.vue'
import IteratorConfig from './props/IteratorConfig.vue'
import LoopConfig from './props/LoopConfig.vue'
import ConditionConfig from './props/ConditionConfig.vue'
import TransformConfig from './props/TransformConfig.vue'
import ScriptConfig from './props/ScriptConfig.vue'
import UserInputConfig from './props/UserInputConfig.vue'

const props = defineProps<{ node: FlowNode | null; nodes?: FlowNode[]; edges?: FlowEdge[] }>()
const emit = defineEmits<{ 'update:node': [node: FlowNode]; 'delete:node': [id: number]; 'close': [] }>()

const nodeConfig = computed(() => {
  if (!props.node) return {}
  const cfg = props.node.config
  if (typeof cfg === 'string') {
    try { return JSON.parse(cfg || '{}') } catch {
      const s = cfg.replace(/[\x00-\x1F\x7F]/g, c =>
        ({ '\n': '\\n', '\r': '\\r', '\t': '\\t', '\b': '\\b', '\f': '\\f' } as Record<string, string>)[c] || '\\u' + ('000' + c.charCodeAt(0).toString(16)).slice(-4)
      )
      try { return JSON.parse(s || '{}') } catch { return {} }
    }
  }
  return cfg || {}
})

function onChildUpdate(childCfg: Record<string, any>) {
  if (!props.node) return
  emit('update:node', { ...props.node, config: JSON.stringify(childCfg) })
}

function emitUpdate() {
  if (props.node) emit('update:node', { ...props.node })
}
</script>

<style scoped>
/* Dify-style slide-out panel */
.property-panel {
  position: absolute; top: 0; right: 0; bottom: 0; width: 360px;
  background: #fff; border-left: 1px solid #eaecf0;
  box-shadow: -4px 0px 16px rgba(16,24,40,0.06); z-index: 50;
  transform: translateX(100%); transition: transform 0.2s cubic-bezier(0.4,0,0.2,1);
  display: flex; flex-direction: column;
}
.property-panel.open { transform: translateX(0); }

.panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; border-bottom: 1px solid #eaecf0; flex-shrink: 0;
}
.panel-header h3 { font-size: 14px; font-weight: 600; color: #101828; margin: 0; }
.close-btn {
  width: 28px; height: 28px; border-radius: 6px; border: none; background: transparent;
  font-size: 16px; color: #667085; cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.close-btn:hover { background: #f2f4f7; color: #101828; }

.props { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; padding: 16px; }
.props > label { font-size: 12px; color: #344054; font-weight: 500; }
.props > input, .props > select {
  padding: 8px 10px; border: 1px solid #d0d5dd; border-radius: 8px;
  font-size: 13px; font-family: inherit; color: #101828; background: #fff;
}
.props > input:focus, .props > select:focus { border-color: #528bff; box-shadow: 0 0 0 3px rgba(82,139,255,0.12); outline: none; }

.empty { font-size: 13px; color: #98a2b3; text-align: center; margin-top: 24px; }
.delete-btn { margin-top: 8px; background: #fef3f2; color: #f04438; border: 1px solid #fecaca; padding: 8px 16px; border-radius: 8px; cursor: pointer; font-size: 12px; font-weight: 500; }
.delete-btn:hover { background: #f04438; color: #fff; border-color: #f04438; }

/* Shared across child components */
:deep(.node-config) { display: flex; flex-direction: column; gap: 10px; }
:deep(.node-config label) { font-size: 12px; color: #344054; font-weight: 500; }
:deep(.node-config input), :deep(.node-config select), :deep(.node-config textarea) {
  padding: 8px 10px; border: 1px solid #d0d5dd; border-radius: 8px;
  font-size: 13px; font-family: inherit; color: #101828; background: #fff;
}
:deep(.node-config input:focus), :deep(.node-config select:focus), :deep(.node-config textarea:focus) {
  border-color: #528bff; box-shadow: 0 0 0 3px rgba(82,139,255,0.12); outline: none;
}
:deep(.node-config textarea) { resize: vertical; min-height: 50px; }
:deep(.node-config input[type="range"]) { padding: 0; border: none; box-shadow: none; }
:deep(.node-config .val) { font-weight: 500; color: #528bff; margin-left: 4px; }
:deep(.node-config .hint) { font-size: 11px; color: #98a2b3; margin-top: -4px; }
:deep(.json-fields) { display: flex; flex-direction: column; gap: 4px; }
:deep(.json-field-row) { display: flex; gap: 4px; align-items: center; }
:deep(.field-key) { flex: 2; min-width: 0; }
:deep(.field-type) { flex: 1; min-width: 0; }
:deep(.field-desc) { flex: 1.5; min-width: 0; }
:deep(.field-del) { background: none; border: none; color: #98a2b3; cursor: pointer; font-size: 16px; padding: 0 2px; line-height: 1; }
:deep(.field-del:hover) { color: #f04438; }
:deep(.field-add) { background: none; border: 1px dashed #d0d5dd; border-radius: 6px; padding: 6px; cursor: pointer; font-size: 11px; color: #667085; }
:deep(.field-add:hover) { border-color: #528bff; color: #528bff; background: #f5f8ff; }
</style>
