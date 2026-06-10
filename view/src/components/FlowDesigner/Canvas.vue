<template>
  <div class="canvas-wrapper">
    <!-- 缩放控制 -->
    <div class="zoom-controls">
      <button @click="zoomIn" title="放大">+</button>
      <span>{{ Math.round(zoom * 100) }}%</span>
      <button @click="zoomOut" title="缩小">−</button>
      <button @click="zoomReset" title="重置">↺</button>
    </div>

    <div
      class="flow-canvas"
      ref="canvasRef"
      tabindex="0"
      @drop="onDrop"
      @dragover.prevent
      @click.self="onCanvasClick"
      @keydown="onKeyDown"
      @wheel.prevent="onWheel"
    >
      <div class="canvas-content" :style="{ transform: `scale(${zoom})`, transformOrigin: '0 0' }">
        <!-- 连线 -->
        <svg class="edges-layer" :style="{ width: canvasW + 'px', height: canvasH + 'px' }">
          <line
            v-for="edge in edges"
            :key="edge.id"
            :x1="getHandlePos(edge.source_node_id, edge.source_handle || 'output').x"
            :y1="getHandlePos(edge.source_node_id, edge.source_handle || 'output').y"
            :x2="getHandlePos(edge.target_node_id, edge.target_handle || 'input').x"
            :y2="getHandlePos(edge.target_node_id, edge.target_handle || 'input').y"
            :stroke="selectedEdgeId === edge.id ? '#ff4d4f' : (edge.source_handle === 'true' ? '#52c41a' : edge.source_handle === 'false' ? '#ff4d4f' : '#b0b0b0')"
            :stroke-width="selectedEdgeId === edge.id ? 3 : 2"
            marker-end="url(#arrowhead)"
            @click.stop="selectEdge(edge)"
            @dblclick.stop="deleteEdge(edge.id)"
          />
          <!-- 边标签 -->
          <text
            v-for="edge in edges.filter(e => e.source_handle === 'true' || e.source_handle === 'false')"
            :key="'t' + edge.id"
            :x="(getHandlePos(edge.source_node_id, edge.source_handle).x + getHandlePos(edge.target_node_id, 'input').x) / 2"
            :y="(getHandlePos(edge.source_node_id, edge.source_handle).y + getHandlePos(edge.target_node_id, 'input').y) / 2 - 6"
            text-anchor="middle"
            :fill="edge.source_handle === 'true' ? '#52c41a' : '#ff4d4f'"
            font-size="12"
            font-weight="bold"
          >{{ edge.source_handle }}</text>
          <defs>
            <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
              <polygon points="0 0, 10 3.5, 0 7" fill="#b0b0b0" />
            </marker>
          </defs>
        </svg>

        <!-- 空画布提示 -->
        <div v-if="nodes.length === 0" class="canvas-hint">
          <div class="hint-icon">🎨</div>
          <div class="hint-text">从左侧拖入节点开始设计流程</div>
          <div class="hint-sub">或通过对话让 AI 帮你创建</div>
        </div>

        <!-- 节点 -->
        <div
          v-for="node in nodes"
          :key="node.id"
          class="flow-node"
          :class="[node.type, { selected: selectedNodeId === node.id }]"
          :style="{ left: node.position_x + 'px', top: node.position_y + 'px' }"
          @mousedown="onNodeMouseDown($event, node)"
          @selectstart.prevent
        >
          <div class="node-header">
            <span class="node-icon">{{ getIcon(node.type) }}</span>
            <span class="node-label">{{ node.label }}</span>
          </div>
          <div class="node-body">
            <span class="node-type-name">{{ getTypeLabel(node.type) }}</span>
            <span v-if="getNodePreview(node)" class="node-preview">{{ getNodePreview(node) }}</span>
          </div>
          <!-- 连接点 -->
          <div class="handle input" @mousedown.stop="onHandleMouseDown($event, node, 'input')"></div>
          <template v-if="node.type === 'condition'">
            <div class="handle output true-handle" @mousedown.stop="onHandleMouseDown($event, node, 'true')" title="True">T</div>
            <div class="handle output false-handle" @mousedown.stop="onHandleMouseDown($event, node, 'false')" title="False">F</div>
          </template>
          <template v-else>
            <div class="handle output" @mousedown.stop="onHandleMouseDown($event, node, 'output')"></div>
          </template>
        </div>

        <!-- 拖拽中的临时连线 -->
        <svg v-if="drawingEdge" class="drawing-line" :style="{ position: 'absolute', top: 0, left: 0, width: canvasW + 'px', height: canvasH + 'px', pointerEvents: 'none', zIndex: 50 }">
          <line :x1="drawStart.x" :y1="drawStart.y" :x2="drawEnd.x" :y2="drawEnd.y" stroke="#4a9eff" stroke-width="2" stroke-dasharray="5,5" />
        </svg>
      </div>
    </div>

    <div v-if="selectedEdgeId" class="edge-tip">选中连线 — Delete 删除</div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import type { FlowNode, FlowEdge, NodeType } from '@/types/flow'

const props = defineProps<{ nodes: FlowNode[]; edges: FlowEdge[] }>()
const emit = defineEmits<{
  'update:nodes': [nodes: FlowNode[]]
  'update:edges': [edges: FlowEdge[]]
  'selectNode': [node: FlowNode | null]
}>()

const canvasRef = ref<HTMLElement | null>(null)
const selectedNodeId = ref<number | null>(null)
const selectedEdgeId = ref<number | null>(null)
const zoom = ref(1)
const canvasW = ref(3000)
const canvasH = ref(2000)
let nextEdgeId = 1000

const GRID = 20
function snap(v: number) { return Math.round(v / GRID) * GRID }

const nodeStyles: Record<string, { color: string; bg: string; icon: string }> = {
  start:    { color: '#52c41a', bg: '#f6ffed', icon: '▶' },
  end:      { color: '#ff4d4f', bg: '#fff2f0', icon: '⏹' },
  llm:      { color: '#4a9eff', bg: '#f0f5ff', icon: '🤖' },
  user_input: { color: '#faad14', bg: '#fffbe6', icon: '👤' },
  split:    { color: '#722ed1', bg: '#f9f0ff', icon: '✂' },
  condition:{ color: '#fa8c16', bg: '#fff7e6', icon: '🔀' },
  transform:{ color: '#13c2c2', bg: '#e6fffb', icon: '⚙' },
  for_each: { color: '#eb2f96', bg: '#fff0f6', icon: '🔁' },
  script:   { color: '#2f54eb', bg: '#f0f5ff', icon: '🐍' },
  iterator: { color: '#fa541c', bg: '#fff2e8', icon: '📋' },
  loop:     { color: '#1890ff', bg: '#e6f7ff', icon: '🔄' },
}

const nodeNames: Record<string, string> = {
  start: '开始', end: '结束', llm: 'LLM调用', user_input: '用户输入',
  split: '文本拆分', condition: '条件分支', transform: '数据变换', for_each: '遍历', script: '脚本', iterator: '迭代', loop: '循环',
}

function getIcon(t: NodeType) { return nodeStyles[t]?.icon || '●' }
function getTypeLabel(t: NodeType) { return nodeNames[t] || t }

function safeParseConfig(raw: any): Record<string, any> {
  if (typeof raw !== 'string') return raw || {}
  try { return JSON.parse(raw || '{}') } catch {}
  const sanitized = raw.replace(/[\x00-\x1F\x7F]/g, c =>
    ({ '\n': '\\n', '\r': '\\r', '\t': '\\t', '\b': '\\b', '\f': '\\f' } as Record<string, string>)[c] || '\\u' + ('000' + c.charCodeAt(0).toString(16)).slice(-4)
  )
  try { return JSON.parse(sanitized || '{}') } catch { return {} }
}

function getNodePreview(node: FlowNode): string {
  const cfg = safeParseConfig(node.config)
  switch (node.type) {
    case 'llm': {
      const m = cfg.model || ''
      return m ? `模型: ${m}` : ''
    }
    case 'user_input': {
      const p = cfg.prompt || ''
      if (typeof p === 'string' && p.trim()) return p.length > 80 ? p.substring(0, 80) + '…' : p
      return cfg.confirm_only ? '[确认模式]' : ''
    }
    case 'split':
      return cfg.source_key ? `来源: ${cfg.source_key}` : (cfg.delimiter ? `分隔: "${cfg.delimiter}"` : '')
    case 'for_each': {
      const p = cfg.prompt || ''
      if (typeof p === 'string' && p.trim()) return p.length > 80 ? p.substring(0, 80) + '…' : p
      return cfg.items_key ? `遍历: ${cfg.items_key}` : ''
    }
    case 'script':
      return cfg.script ? cfg.script.substring(0, 60) + (cfg.script.length > 60 ? '…' : '') : ''
    case 'transform':
      return cfg.expression ? cfg.expression.substring(0, 60) + (cfg.expression.length > 60 ? '…' : '') : ''
    case 'condition':
      return cfg.expression ? cfg.expression.substring(0, 60) + (cfg.expression.length > 60 ? '…' : '') : ''
    default:
      return ''
  }
}

function getHandlePos(nodeId: number, handle: string): { x: number; y: number } {
  const node = props.nodes.find(n => n.id === nodeId)
  if (!node) return { x: 0, y: 0 }
  const y = node.position_y + 28
  if (handle === 'input') return { x: node.position_x, y }
  if (handle === 'true') return { x: node.position_x + 170, y: y - 8 }
  if (handle === 'false') return { x: node.position_x + 170, y: y + 12 }
  return { x: node.position_x + 170, y } // output
}

function selectNode(node: FlowNode) { selectedNodeId.value = node.id; selectedEdgeId.value = null; emit('selectNode', node) }
function selectEdge(edge: FlowEdge) { selectedEdgeId.value = edge.id; selectedNodeId.value = null; emit('selectNode', null) }
function deleteEdge(id: number) { emit('update:edges', props.edges.filter(e => e.id !== id)); selectedEdgeId.value = null }
function onCanvasClick() { selectedNodeId.value = null; selectedEdgeId.value = null; emit('selectNode', null) }
function onKeyDown(e: KeyboardEvent) { if ((e.key === 'Delete' || e.key === 'Backspace') && selectedEdgeId.value) deleteEdge(selectedEdgeId.value) }

function zoomIn() { zoom.value = Math.min(2, zoom.value + 0.1) }
function zoomOut() { zoom.value = Math.max(0.3, zoom.value - 0.1) }
function zoomReset() { zoom.value = 1 }
function onWheel(e: WheelEvent) { if (e.ctrlKey || e.metaKey) { zoom.value = Math.max(0.3, Math.min(2, zoom.value - e.deltaY * 0.001)); } }

function onDrop(event: DragEvent) {
  const nodeType = event.dataTransfer!.getData('node-type') as NodeType
  const nodeLabel = event.dataTransfer!.getData('node-label')
  if (!nodeType) return
  const rect = canvasRef.value!.getBoundingClientRect()
  const newNode: FlowNode = {
    id: Date.now(), flow_id: 0, type: nodeType, label: nodeLabel || nodeType, config: {},
    position_x: snap(Math.max(0, (event.clientX - rect.left) / zoom.value - 85)),
    position_y: snap(Math.max(0, (event.clientY - rect.top) / zoom.value - 28)),
  }
  emit('update:nodes', [...props.nodes, newNode])
}

// ========== 拖拽移动 ==========
const drawingEdge = ref(false)
const drawStart = reactive({ x: 0, y: 0 })
const drawEnd = reactive({ x: 0, y: 0 })
let drawSourceNodeId: number | null = null
let drawSourceHandle = ''
let draggingNode: FlowNode | null = null
let dragStartPos = { x: 0, y: 0 }
let dragOffset = { x: 0, y: 0 }
let hasDragged = false

function onNodeMouseDown(event: MouseEvent, node: FlowNode) {
  draggingNode = node
  hasDragged = false
  dragStartPos = { x: event.clientX, y: event.clientY }
  dragOffset = { x: event.clientX - node.position_x * zoom.value, y: event.clientY - node.position_y * zoom.value }
  const onMove = (e: MouseEvent) => {
    if (!draggingNode) return
    const dx = Math.abs(e.clientX - dragStartPos.x)
    const dy = Math.abs(e.clientY - dragStartPos.y)
    if (!hasDragged && dx < 6 && dy < 6) return
    hasDragged = true
    emit('update:nodes', props.nodes.map(n => n.id === draggingNode!.id
      ? { ...n, position_x: snap(Math.max(0, (e.clientX - dragOffset.x) / zoom.value)), position_y: snap(Math.max(0, (e.clientY - dragOffset.y) / zoom.value)) }
      : n))
  }
  const onUp = () => {
    if (!hasDragged) selectNode(node)
    draggingNode = null; hasDragged = false
    window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp)
  }
  window.addEventListener('mousemove', onMove); window.addEventListener('mouseup', onUp)
}

function onHandleMouseDown(event: MouseEvent, node: FlowNode, handle: string) {
  drawingEdge.value = true; drawSourceNodeId = node.id; drawSourceHandle = handle
  const pos = getHandlePos(node.id, handle)
  drawStart.x = pos.x; drawStart.y = pos.y; drawEnd.x = pos.x; drawEnd.y = pos.y
  const onMove = (e: MouseEvent) => {
    const rect = canvasRef.value!.getBoundingClientRect()
    drawEnd.x = (e.clientX - rect.left) / zoom.value; drawEnd.y = (e.clientY - rect.top) / zoom.value
  }
  const onUp = (e: MouseEvent) => {
    drawingEdge.value = false; window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp)
    const targetEl = document.elementFromPoint(e.clientX, e.clientY)
    const targetNodeEl = targetEl?.closest('.flow-node')
    if (targetNodeEl) {
      const allNodes = document.querySelectorAll('.flow-node')
      const idx = Array.from(allNodes).indexOf(targetNodeEl as Element)
      const targetNode = props.nodes[idx]
      if (targetNode && targetNode.id !== node.id) {
        const srcH = handle === 'true' || handle === 'false' ? handle : 'output'
        const newEdge: FlowEdge = {
          id: nextEdgeId++, flow_id: 0,
          source_node_id: node.id, target_node_id: targetNode.id,
          source_handle: srcH, target_handle: 'input', label: '',
        }
        if (!props.edges.some(e => e.source_node_id === newEdge.source_node_id && e.target_node_id === newEdge.target_node_id && e.source_handle === newEdge.source_handle))
          emit('update:edges', [...props.edges, newEdge])
      }
    }
    drawSourceNodeId = null
  }
  window.addEventListener('mousemove', onMove); window.addEventListener('mouseup', onUp)
}
</script>

<style scoped>
.canvas-wrapper { flex: 1; position: relative; overflow: hidden; display: flex; flex-direction: column; }
.zoom-controls { position: absolute; bottom: 16px; right: 16px; z-index: 100; display: flex; gap: 4px; background: #fff; border-radius: 6px; padding: 4px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
.zoom-controls button { width: 28px; height: 28px; border: 1px solid #ddd; border-radius: 4px; background: #fff; cursor: pointer; font-size: 14px; display: flex; align-items: center; justify-content: center; }
.zoom-controls button:hover { background: #f0f0f0; }
.zoom-controls span { font-size: 12px; color: #666; display: flex; align-items: center; padding: 0 4px; min-width: 36px; justify-content: center; }
.flow-canvas { flex: 1; overflow: auto; background: #f5f6f8; background-image: radial-gradient(circle, #d9d9d9 1px, transparent 1px); background-size: 20px 20px; outline: none; position: relative; }
.canvas-content { position: relative; min-width: 3000px; min-height: 2000px; }
.canvas-hint { position: absolute; top: 40%; left: 50%; transform: translate(-50%, -50%); text-align: center; z-index: 10; user-select: none; pointer-events: none; }
.hint-icon { font-size: 48px; margin-bottom: 12px; opacity: 0.6; }
.hint-text { font-size: 16px; color: #999; margin-bottom: 6px; }
.hint-sub { font-size: 13px; color: #bbb; }
.edges-layer { position: absolute; top: 0; left: 0; pointer-events: none; z-index: 1; }
.edges-layer line { pointer-events: stroke; cursor: pointer; }

/* ====== 节点样式 ====== */
.flow-node { position: absolute; width: 170px; border-radius: 8px; cursor: grab; z-index: 5; user-select: none; -webkit-user-select: none; box-shadow: 0 1px 3px rgba(0,0,0,0.08); transition: box-shadow 0.15s; overflow: hidden; }
.flow-node * { user-select: none; -webkit-user-select: none; cursor: grab; }
.flow-node:active { cursor: grabbing; }
.flow-node:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
.flow-node.selected { box-shadow: 0 0 0 2px #4a9eff, 0 4px 12px rgba(0,0,0,0.15); }

.node-header { display: flex; align-items: center; gap: 6px; padding: 6px 10px; color: #fff; font-size: 13px; font-weight: 500; }
.node-icon { font-size: 14px; width: 20px; text-align: center; flex-shrink: 0; }
.node-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.node-body { padding: 4px 10px 6px; background: #fff; font-size: 10px; color: #999; }
.node-type-name { text-transform: uppercase; letter-spacing: 0.5px; }
.node-preview { display: block; margin-top: 2px; font-size: 11px; color: #555; line-height: 1.3; white-space: pre-line; overflow: hidden; max-height: 44px; }

/* 各类型颜色 */
.flow-node.start .node-header { background: #52c41a; }
.flow-node.end .node-header { background: #ff4d4f; }
.flow-node.llm .node-header { background: #4a9eff; }
.flow-node.user_input .node-header { background: #faad14; }
.flow-node.split .node-header { background: #722ed1; }
.flow-node.condition .node-header { background: #fa8c16; }
.flow-node.transform .node-header { background: #13c2c2; }
.flow-node.for_each .node-header { background: #eb2f96; }
.flow-node.script .node-header { background: #2f54eb; }

/* 连接点 */
.handle { position: absolute; width: 10px; height: 10px; background: #fff; border: 2px solid #999; border-radius: 50%; cursor: crosshair !important; z-index: 6; transition: all 0.15s; }
.handle:hover { background: #4a9eff; border-color: #4a9eff; transform: scale(1.3); }
.handle.input { left: -5px; top: 50%; transform: translateY(-50%); }
.handle.input:hover { transform: translateY(-50%) scale(1.3); }
.handle.output { right: -5px; top: 50%; transform: translateY(-50%); }
.handle.output:hover { transform: translateY(-50%) scale(1.3); }
.true-handle { top: 35% !important; background: #e6ffe6; border-color: #52c41a; color: #52c41a; font-size: 8px; display: flex; align-items: center; justify-content: center; transform: translateY(-50%) !important; }
.true-handle:hover { transform: translateY(-50%) scale(1.3) !important; }
.false-handle { top: 65% !important; background: #ffe6e6; border-color: #ff4d4f; color: #ff4d4f; font-size: 8px; display: flex; align-items: center; justify-content: center; transform: translateY(-50%) !important; }
.false-handle:hover { transform: translateY(-50%) scale(1.3) !important; }

.edge-tip { position: absolute; bottom: 60px; left: 50%; transform: translateX(-50%); background: #ff4d4f; color: #fff; padding: 6px 16px; border-radius: 4px; font-size: 12px; z-index: 100; }
</style>
