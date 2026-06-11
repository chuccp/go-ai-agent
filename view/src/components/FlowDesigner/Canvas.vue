<template>
  <div class="canvas-wrapper">
    <div
      class="flow-canvas" ref="canvasRef" tabindex="0"
      @drop="onDrop" @dragover.prevent
      @mousedown="onCanvasMouseDown" @keydown="onKeyDown"
      @wheel.prevent="onWheel"
    >
      <div v-if="nodes.length === 0" class="canvas-hint">
        <div class="hint-icon">🎨</div>
        <div class="hint-text">点击底部 ＋ 添加节点</div>
        <div class="hint-sub">或通过对话让 AI 帮你创建</div>
      </div>

      <div class="canvas-content" :style="{ transform: `scale(${zoom})`, transformOrigin: '0 0' }">
        <!-- Nodes -->
        <div v-for="node in nodes" :key="node.id"
          class="flow-node" :class="[node.type, { selected: selectedNodeId === node.id }]"
          :style="{ left: node.position_x + 'px', top: node.position_y + 'px' }"
          @mousedown="onNodeMouseDown($event, node)" @selectstart.prevent>

          <template v-if="isContainerNode(node)">
            <!-- Container node: Dify-style transparent header + internal body -->
            <div class="container-header">
              <span class="cn-icon" :style="{ background: getNodeColorHex(node.type) }">{{ getIcon(node.type) }}</span>
              <span class="cn-label">{{ node.label }}</span>
            </div>
            <div class="container-body">
              <span class="container-hint">循环体</span>
              <div class="handle input" @mousedown.stop="onHandleMouseDown($event, node, 'input')"></div>
              <div class="handle output loop-start" @mousedown.stop="onHandleMouseDown($event, node, 'loop_start')" title="循环体出口">▸</div>
              <div class="handle output" @mousedown.stop="onHandleMouseDown($event, node, 'output')"></div>
            </div>
          </template>

          <template v-else>
            <!-- Regular node: Dify-style white card with colored icon -->
            <div class="node-header">
              <span class="nc-icon" :style="{ background: getNodeColorHex(node.type) }">{{ getIcon(node.type) }}</span>
              <span class="nc-label">{{ node.label }}</span>
              <span class="nc-type">{{ getTypeLabel(node.type) }}</span>
            </div>
            <div v-if="getNodePreview(node)" class="node-desc">{{ getNodePreview(node) }}</div>
            <div class="handle input" @mousedown.stop="onHandleMouseDown($event, node, 'input')"></div>
            <div class="handle output true-handle" v-if="node.type === 'condition'" @mousedown.stop="onHandleMouseDown($event, node, 'true')" title="True">T</div>
            <div class="handle output false-handle" v-if="node.type === 'condition'" @mousedown.stop="onHandleMouseDown($event, node, 'false')" title="False">F</div>
            <div class="handle output" v-if="node.type !== 'condition'" @mousedown.stop="onHandleMouseDown($event, node, 'output')"></div>
          </template>
        </div>

        <!-- Edges: rendered AFTER nodes so they're visible on top -->
        <svg class="edges-layer" :style="{ width: canvasW + 'px', height: canvasH + 'px' }">
          <path v-for="edge in edges" :key="edge.id"
            :d="edgePath(edge)" fill="none"
            :stroke="selectedEdgeId === edge.id ? '#6366f1' : '#c0c8d0'"
            :stroke-width="selectedEdgeId === edge.id ? 2.5 : 2"
            marker-end="url(#arrowhead)"
            @click.stop="selectEdge(edge)" @dblclick.stop="deleteEdge(edge.id)" />
          <defs>
            <marker id="arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <polygon points="0 0, 8 3, 0 6" fill="#c0c8d0" />
            </marker>
          </defs>
        </svg>

        <!-- Edge midpoint + buttons (Dify-style block insertion) -->
        <div
          v-for="edge in edges" :key="'btn-' + edge.id"
          class="edge-add-btn"
          :style="{ left: edgeMidpoint(edge).x + 'px', top: edgeMidpoint(edge).y + 'px' }"
          @click.stop="onEdgeAddClick(edge)"
          @mousedown.stop
          title="添加节点"
        >+</div>

        <!-- Drawing edge preview -->
        <svg v-if="drawingEdge" class="drawing-edge" :style="{ position:'absolute',top:0,left:0,width:canvasW+'px',height:canvasH+'px',pointerEvents:'none',zIndex:50 }">
          <path :d="`M ${drawStart.x} ${drawStart.y} C ${drawStart.x+50} ${drawStart.y}, ${drawEnd.x-50} ${drawEnd.y}, ${drawEnd.x} ${drawEnd.y}`" fill="none" stroke="#6366f1" stroke-width="2" stroke-dasharray="5,5" />
        </svg>
      </div>
    </div>

    <!-- Quick-add picker at handle -->
    <div v-if="quickPick" class="quick-pick" :style="{ position:'absolute', left:quickPick.x+'px', top:quickPick.y+'px', zIndex:200 }" @mousedown.stop>
      <button v-for="qt in quickTypes" :key="qt.type" class="quick-item" @click="onQuickAdd(qt.type)">
        <span>{{ qt.icon }}</span> {{ qt.label }}
      </button>
      <button class="quick-close" @click="quickPick = null">×</button>
    </div>

    <div v-if="selectedEdgeId" class="edge-tip">选中连线 — Delete 删除</div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, nextTick, onMounted } from 'vue'
import type { FlowNode, FlowEdge, NodeType } from '@/types/flow'
import { getNodeLabel, getNodeIcon } from '@/types/flow'

const props = defineProps<{ nodes: FlowNode[]; edges: FlowEdge[] }>()
const emit = defineEmits<{
  'update:nodes': [nodes: FlowNode[]]
  'update:edges': [edges: FlowEdge[]]
  'selectNode': [node: FlowNode | null]
}>()
defineExpose({ zoomIn, zoomOut, zoomReset })

const canvasRef = ref<HTMLElement | null>(null)
const selectedNodeId = ref<number | null>(null)
const selectedEdgeId = ref<number | null>(null)
const zoom = ref(1)
const canvasW = ref(4000)
const canvasH = ref(3000)
let nextEdgeId = 1000

const NODE_W = 240
const GRID = 20
const snap = (v: number) => Math.round(v / GRID) * GRID

// ── Quick-add picker at handle ──
interface QuickPick { nodeId: number; handle: string; x: number; y: number; insertBetween?: { sourceId: number; sourceHandle: string; targetId: number; targetHandle: string; edgeId: number } }
const quickPick = ref<QuickPick | null>(null)
const quickTypes = [
  { type: 'llm' as NodeType, label: 'LLM', icon: '🤖' },
  { type: 'user_input' as NodeType, label: '输入', icon: '👤' },
  { type: 'condition' as NodeType, label: '条件', icon: '🔀' },
  { type: 'transform' as NodeType, label: '变换', icon: '⚙' },
  { type: 'end' as NodeType, label: '结束', icon: '⏹' },
]

function onQuickAdd(type: NodeType) {
  if (!quickPick.value) return
  const p = quickPick.value
  const newNode: FlowNode = {
    id: Date.now(), flow_id: 0, type, label: type, config: {},
    position_x: snap(p.x - NODE_W / 2),
    position_y: snap(p.y - 20),
  }

  if (p.insertBetween) {
    // Insert between two nodes: remove old edge, create two new edges
    const { sourceId, sourceHandle, targetId, targetHandle, edgeId } = p.insertBetween
    const newEdges = props.edges.filter(e => e.id !== edgeId)
    newEdges.push(
      { id: nextEdgeId++, flow_id: 0, source_node_id: sourceId, target_node_id: newNode.id, source_handle: sourceHandle, target_handle: 'input', label: '' },
      { id: nextEdgeId++, flow_id: 0, source_node_id: newNode.id, target_node_id: targetId, source_handle: 'output', target_handle: targetHandle, label: '' },
    )
    emit('update:nodes', [...props.nodes, newNode])
    emit('update:edges', newEdges)
  } else {
    // Append after source node
    const sourceNode = props.nodes.find(n => n.id === p.nodeId)
    const srcH = p.handle === 'true' || p.handle === 'false' ? p.handle : 'output'
    emit('update:nodes', [...props.nodes, newNode])
    emit('update:edges', [...props.edges, {
      id: nextEdgeId++, flow_id: 0,
      source_node_id: p.nodeId, target_node_id: newNode.id,
      source_handle: srcH, target_handle: 'input', label: '',
    }])
  }
  quickPick.value = null
}

// ── Node helpers ──
const getIcon = (t: NodeType) => getNodeIcon(t)
const getTypeLabel = (t: NodeType) => getNodeLabel(t)
const nodeColors: Record<string, string> = {
  start: '#17b26a', end: '#f04438', llm: '#528bff', user_input: '#f79009',
  split: '#875bf7', condition: '#f38744', transform: '#15b79e',
  for_each: '#dd2590', iterator: '#dd2590', loop: '#2e90fa', script: '#6172f3',
  image_gen: '#36cfc9', audio_gen: '#9254de', video_gen: '#f759ab',
}
const getNodeColorHex = (t: NodeType) => nodeColors[t] || '#98a2b3'
const isContainerNode = (node: FlowNode): boolean => {
  if (node.type !== 'for_each' && node.type !== 'iterator' && node.type !== 'loop') return false
  const cfg = typeof node.config === 'string' ? safeParse(node.config) : (node.config || {})
  return !cfg.items_key
}

function safeParse(raw: any): Record<string, any> {
  if (typeof raw !== 'string') return raw || {}
  try { return JSON.parse(raw || '{}') } catch { return {} }
}

function getNodePreview(node: FlowNode): string {
  const cfg = safeParse(node.config)
  switch (node.type) {
    case 'llm': return cfg.model ? `模型: ${cfg.model}` : ''
    case 'user_input': return cfg.confirm_only ? '确认模式' : (typeof cfg.prompt === 'string' ? cfg.prompt.slice(0, 40) : '')
    case 'split': return cfg.source_key ? `来源: ${cfg.source_key}` : ''
    case 'for_each': case 'iterator': case 'loop': return cfg.items_key ? `来源: ${cfg.items_key}` : ''
    case 'script': return cfg.script ? cfg.script.slice(0, 40) + '…' : ''
    case 'transform': return cfg.template ? cfg.template.slice(0, 40) + '…' : ''
    case 'condition': return cfg.field || ''
    default: return ''
  }
}

// ── Edges ──
function getHandlePos(nodeId: number, handle: string): { x: number; y: number } {
  const node = props.nodes.find(n => n.id === nodeId)
  if (!node) return { x: 0, y: 0 }
  const x = node.position_x, y = node.position_y
  if (handle === 'input') return { x, y: y + 24 }
  if (handle === 'loop_start') return { x: x + NODE_W / 2, y: y + 74 }
  if (handle === 'true') return { x: x + NODE_W, y: y + 18 }
  if (handle === 'false') return { x: x + NODE_W, y: y + 32 }
  return { x: x + NODE_W, y: y + 24 }
}

function edgePath(edge: FlowEdge): string {
  const s = getHandlePos(edge.source_node_id, edge.source_handle || 'output')
  const t = getHandlePos(edge.target_node_id, edge.target_handle || 'input')
  const dx = t.x - s.x, dy = t.y - s.y
  const cp = Math.max(40, Math.abs(dx) * 0.16)
  // loop_start: curve exits bottom, goes downward away from container
  if (edge.source_handle === 'loop_start') {
    const down = Math.max(40, Math.abs(dy) * 0.3)
    return `M ${s.x} ${s.y} C ${s.x} ${s.y + down}, ${t.x} ${t.y - down}, ${t.x} ${t.y}`
  }
  return `M ${s.x} ${s.y} C ${s.x + cp} ${s.y}, ${t.x - cp} ${t.y}, ${t.x} ${t.y}`
}

function selectNode(node: FlowNode) { selectedNodeId.value = node.id; selectedEdgeId.value = null; emit('selectNode', node) }
function selectEdge(edge: FlowEdge) { selectedEdgeId.value = edge.id; selectedNodeId.value = null; emit('selectNode', null) }
function deleteEdge(id: number) { emit('update:edges', props.edges.filter(e => e.id !== id)); selectedEdgeId.value = null }
function onCanvasMouseDown(e: MouseEvent) {
  quickPick.value = null
  if (!(e.target as HTMLElement).closest('.flow-node')) {
    selectedNodeId.value = null; selectedEdgeId.value = null; emit('selectNode', null)
  }
}
function edgeMidpoint(edge: FlowEdge): { x: number; y: number } {
  const s = getHandlePos(edge.source_node_id, edge.source_handle || 'output')
  const t = getHandlePos(edge.target_node_id, edge.target_handle || 'input')
  // Midpoint of the Bézier curve at t=0.5
  const cp = Math.max(40, Math.abs(t.x - s.x) * 0.16)
  const mx = 0.125 * s.x + 0.375 * (s.x + cp) + 0.375 * (t.x - cp) + 0.125 * t.x
  const my = 0.125 * s.y + 0.375 * s.y + 0.375 * t.y + 0.125 * t.y
  return { x: mx, y: my }
}

function onEdgeAddClick(edge: FlowEdge) {
  // Insert a node between source and target
  quickPick.value = {
    nodeId: edge.source_node_id,
    handle: 'output',
    x: edgeMidpoint(edge).x,
    y: edgeMidpoint(edge).y,
    insertBetween: { sourceId: edge.source_node_id, sourceHandle: edge.source_handle || 'output', targetId: edge.target_node_id, targetHandle: edge.target_handle || 'input', edgeId: edge.id },
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Delete' || e.key === 'Backspace') {
    if (selectedEdgeId.value) { deleteEdge(selectedEdgeId.value); return }
    if (selectedNodeId.value) {
      emit('update:nodes', props.nodes.filter(n => n.id !== selectedNodeId.value))
      emit('update:edges', props.edges.filter(e => e.source_node_id !== selectedNodeId.value && e.target_node_id !== selectedNodeId.value))
      selectedNodeId.value = null; emit('selectNode', null)
    }
  }
}

// ── Zoom ──
function zoomIn() { zoom.value = Math.min(2, zoom.value + 0.1) }
function zoomOut() { zoom.value = Math.max(0.3, zoom.value - 0.1) }
function zoomReset() { zoom.value = 1 }
function onWheel(e: WheelEvent) { if (e.ctrlKey || e.metaKey) zoom.value = Math.max(0.3, Math.min(2, zoom.value - e.deltaY * 0.001)) }

function scrollToCenter() {
  if (!props.nodes?.length || !canvasRef.value) return
  nextTick(() => {
    const el = canvasRef.value!
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
    props.nodes.forEach(n => {
      if (n.position_x < minX) minX = n.position_x
      if (n.position_y < minY) minY = n.position_y
      if (n.position_x + NODE_W > maxX) maxX = n.position_x + NODE_W
      if (n.position_y + 80 > maxY) maxY = n.position_y + 80
    })
    if (minX === Infinity) return
    el.scrollLeft = Math.max(0, (minX + maxX) / 2 * zoom.value - el.clientWidth / 2)
    el.scrollTop = Math.max(0, (minY + maxY) / 2 * zoom.value - el.clientHeight / 2)
  })
}
watch(() => props.nodes, () => scrollToCenter(), { deep: true })
onMounted(() => nextTick(scrollToCenter))

// ── Drop ──
function onDrop(event: DragEvent) {
  const nodeType = event.dataTransfer!.getData('node-type') as NodeType
  const nodeLabel = event.dataTransfer!.getData('node-label')
  if (!nodeType) return
  const rect = canvasRef.value!.getBoundingClientRect()
  emit('update:nodes', [...props.nodes, {
    id: Date.now(), flow_id: 0, type: nodeType, label: nodeLabel || nodeType, config: {},
    position_x: snap(Math.max(0, (event.clientX - rect.left) / zoom.value - NODE_W / 2)),
    position_y: snap(Math.max(0, (event.clientY - rect.top) / zoom.value - 14)),
  }])
}

// ── Drag move ──
const drawingEdge = ref(false)
const drawStart = reactive({ x: 0, y: 0 })
const drawEnd = reactive({ x: 0, y: 0 })
let draggingNode: FlowNode | null = null, dragOffset = { x: 0, y: 0 }, hasDragged = false, dragStartPos = { x: 0, y: 0 }

function onNodeMouseDown(event: MouseEvent, node: FlowNode) {
  draggingNode = node; hasDragged = false
  dragStartPos = { x: event.clientX, y: event.clientY }
  dragOffset = { x: event.clientX - node.position_x * zoom.value, y: event.clientY - node.position_y * zoom.value }
  const onMove = (e: MouseEvent) => {
    if (!draggingNode || Math.abs(e.clientX - dragStartPos.x) + Math.abs(e.clientY - dragStartPos.y) < 6) return
    hasDragged = true
    emit('update:nodes', props.nodes.map(n => n.id === draggingNode!.id
      ? { ...n, position_x: snap(Math.max(0, (e.clientX - dragOffset.x) / zoom.value)), position_y: snap(Math.max(0, (e.clientY - dragOffset.y) / zoom.value)) }
      : n))
  }
  const onUp = () => { if (!hasDragged) selectNode(node); draggingNode = null; window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp) }
  window.addEventListener('mousemove', onMove); window.addEventListener('mouseup', onUp)
}

function onHandleMouseDown(event: MouseEvent, node: FlowNode, handle: string) {
  event.stopPropagation()
  const startX = event.clientX, startY = event.clientY
  drawingEdge.value = true
  const pos = getHandlePos(node.id, handle)
  drawStart.x = pos.x; drawStart.y = pos.y; drawEnd.x = pos.x; drawEnd.y = pos.y
  const srcH = handle === 'true' || handle === 'false' ? handle : (handle === 'loop_start' ? 'loop_start' : 'output')
  const onMove = (e: MouseEvent) => {
    const rect = canvasRef.value!.getBoundingClientRect()
    drawEnd.x = (e.clientX - rect.left) / zoom.value; drawEnd.y = (e.clientY - rect.top) / zoom.value
  }
  const onUp = (e: MouseEvent) => {
    drawingEdge.value = false; window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp)
    const dx = e.clientX - startX, dy = e.clientY - startY
    // Short click → show quick-add picker (only for output-type handles)
    if (Math.abs(dx) + Math.abs(dy) < 8 && handle !== 'input') {
      const rect = canvasRef.value!.getBoundingClientRect()
      quickPick.value = { nodeId: node.id, handle: srcH, x: (e.clientX - rect.left) / zoom.value, y: (e.clientY - rect.top) / zoom.value }
      return
    }
    // Drag → try to connect to target node
    const targetEl = document.elementFromPoint(e.clientX, e.clientY)?.closest('.flow-node')
    if (!targetEl) { quickPick.value = null; return }
    const allNodes = Array.from(document.querySelectorAll('.flow-node'))
    const idx = allNodes.indexOf(targetEl as Element)
    const targetNode = props.nodes[idx]
    if (targetNode && targetNode.id !== node.id && !props.edges.some(ee => ee.source_node_id === node.id && ee.target_node_id === targetNode.id && ee.source_handle === srcH)) {
      emit('update:edges', [...props.edges, { id: nextEdgeId++, flow_id: 0, source_node_id: node.id, target_node_id: targetNode.id, source_handle: srcH, target_handle: 'input', label: '' }])
    }
  }
  window.addEventListener('mousemove', onMove); window.addEventListener('mouseup', onUp)
}
</script>

<style scoped>
/* ── Dify-style canvas ── */
.canvas-wrapper { flex: 1; position: relative; overflow: hidden; display: flex; flex-direction: column; }
.flow-canvas {
  flex: 1; overflow: auto; outline: none; position: relative;
  background-color: #f2f4f7;
  background-image: radial-gradient(circle, #d0d5dd 1px, transparent 1px);
  background-size: 16px 16px;
}
.canvas-content { position: relative; min-width: 4000px; min-height: 3000px; }
.canvas-hint { position: absolute; top: 50%; left: 50%; transform: translate(-50%,-50%); text-align: center; z-index: 10; user-select: none; pointer-events: none; }
.hint-icon { font-size: 48px; margin-bottom: 12px; opacity: 0.3; }
.hint-text { font-size: 15px; color: #98a2b3; margin-bottom: 4px; }
.hint-sub { font-size: 13px; color: #d0d5dd; }
.edges-layer { position: absolute; top: 0; left: 0; pointer-events: none; z-index: 10; }
.edges-layer path { pointer-events: stroke; cursor: pointer; }

/* ── Node (Dify-style) ── */
.flow-node {
  position: absolute; width: 240px; border-radius: 16px; cursor: grab;
  z-index: 5; user-select: none; background: #fff;
  box-shadow: 0px 1px 2px rgba(16,24,40,0.06), 0px 1px 3px rgba(16,24,40,0.1);
  transition: box-shadow 0.2s; overflow: visible;
}
.flow-node:hover { box-shadow: 0px 4px 8px -2px rgba(16,24,40,0.1), 0px 2px 4px -2px rgba(16,24,40,0.06); }
.flow-node.selected { box-shadow: 0 0 0 2px #528bff, 0px 8px 16px -4px rgba(16,24,40,0.15); }
.flow-node:active { cursor: grabbing; }

/* ── Regular node header (Dify-style: white bg, colored icon) ── */
.node-header {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 12px 8px; background: #fff;
  border-radius: 16px 16px 0 0; line-height: 18px;
}
.nc-icon {
  width: 24px; height: 24px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 14px; flex-shrink: 0;
}
.nc-label { font-size: 12px; font-weight: 600; color: #101828; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.nc-type { font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; color: #98a2b3; flex-shrink: 0; }

/* ── Node description (subtle, below body) ── */
.node-desc {
  padding: 0 12px 12px; background: #fff;
  font-size: 11px; color: #667085; line-height: 1.4;
  border-radius: 0 0 16px 16px;
}

/* ── Container header (transparent bg) ── */
.container-header {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 12px 8px; background: transparent;
  border-radius: 16px 16px 0 0;
}
.cn-icon {
  width: 24px; height: 24px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 14px; flex-shrink: 0;
}
.cn-label { font-size: 12px; font-weight: 600; color: #101828; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* ── Container body ── */
.container-body {
  position: relative; padding: 0;
  background: #f9fafb;
  border-top: 1px solid #eaecf0; border-bottom: 1px solid #eaecf0;
  min-height: 64px; display: flex; align-items: center; justify-content: center;
}
.container-hint { font-size: 12px; color: #d0d5dd; font-weight: 500; }

/* ── Handles ── */
.handle {
  position: absolute; width: 8px; height: 8px; background: #fff;
  border: 2px solid #98a2b3; border-radius: 50%; cursor: crosshair !important; z-index: 6;
  transition: all 0.15s;
}
.handle:hover { background: #155eef; border-color: #155eef; transform: scale(1.4); }
.handle.input { left: -4px; top: 12px; }
.handle.output { right: -4px; top: 12px; }
.true-handle { top: 4px !important; }
.false-handle { top: 20px !important; }
.loop-start {
  right: auto !important; left: 50% !important; bottom: -8px !important;
  transform: translateX(-50%) !important;
  width: 16px !important; height: 16px !important;
  background: #155eef !important; border: none !important; color: #fff; font-size: 9px;
  display: flex; align-items: center; justify-content: center; border-radius: 50% !important;
}
.loop-start:hover { transform: translateX(-50%) scale(1.2) !important; }

/* ── Node type colors (icon backgrounds) ── */
.node-color { width: 24px; height: 24px; border-radius: 6px; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 14px; flex-shrink: 0; }

.edge-tip { position: absolute; bottom: 70px; left: 50%; transform: translateX(-50%); background: #f04438; color: #fff; padding: 4px 12px; border-radius: 6px; font-size: 12px; z-index: 100; }

/* Quick-add picker */
.quick-pick { display: flex; gap: 4px; background: #fff; border-radius: 10px; padding: 6px; box-shadow: 0px 4px 16px rgba(16,24,40,0.12); border: 1px solid #eaecf0; }
.quick-item { display: flex; align-items: center; gap: 4px; padding: 5px 10px; border: 1px solid #eaecf0; border-radius: 8px; background: #fff; cursor: pointer; font-size: 12px; color: #344054; white-space: nowrap; }
.quick-item:hover { border-color: #528bff; background: #f5f8ff; }
.quick-close { border: none; background: none; cursor: pointer; color: #98a2b3; font-size: 14px; padding: 0 2px; }
.quick-close:hover { color: #f04438; }

/* Edge add button */
.edge-add-btn {
  position: absolute; z-index: 20;
  width: 18px; height: 18px; border-radius: 50%;
  background: #fff; border: 1.5px solid #d0d5dd; color: #98a2b3;
  font-size: 13px; line-height: 1; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transform: translate(-50%, -50%);
  opacity: 0; transition: opacity 0.15s, border-color 0.15s;
}
.flow-canvas:hover .edge-add-btn { opacity: 1; }
.edge-add-btn:hover { border-color: #155eef; color: #155eef; }
</style>
