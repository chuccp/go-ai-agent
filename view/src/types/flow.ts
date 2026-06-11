// ==================== Flow Types ====================

// Node type constants (matches backend flow/nodes/const.go)
export const NodeType = {
  Start: 'start',
  End: 'end',
  LLM: 'llm',
  UserInput: 'user_input',
  ForEach: 'for_each',
  Split: 'split',
  Transform: 'transform',
  Condition: 'condition',
  Script: 'script',
  Iterator: 'iterator',
  Loop: 'loop',
} as const

// Flow event constants (matches backend flow/engine/stream.go)
export const FlowEventType = {
  NodeStart: 'flow_node_start',
  NodeChunk: 'flow_node_chunk',
  NodeDone: 'flow_node_done',
  WaitingUser: 'flow_waiting_user',
  FlowComplete: 'flow_complete',
  FlowError: 'flow_error',
} as const

export interface FlowDefinition {
  id: number
  name: string
  description: string
  category: string
  config: Record<string, any>
  created_at: string
  updated_at: string
}

export interface FlowDetail extends FlowDefinition {
  nodes: FlowNode[]
  edges: FlowEdge[]
}

export interface FlowNode {
  id: number
  flow_id: number
  type: NodeType
  label: string
  config: Record<string, any>
  position_x: number
  position_y: number
  group_id?: number | null
}

export interface FlowEdge {
  id: number
  flow_id: number
  source_node_id: number
  target_node_id: number
  source_handle: string
  target_handle: string
  label: string
}

export type NodeType = 'start' | 'end' | 'llm' | 'user_input' | 'for_each' | 'split' | 'transform' | 'condition' | 'script' | 'iterator' | 'loop' | 'image_gen' | 'audio_gen' | 'video_gen'

// ── Centralized node definitions (used by NodePanel, Canvas, PropertyPanel) ──
export interface NodeDef {
  type: NodeType
  label: string
  icon: string
  color: string
  category: string
}

export const NODE_CATEGORIES: { key: string; label: string }[] = [
  { key: 'basic', label: '基础' },
  { key: 'ai', label: 'AI' },
  { key: 'logic', label: '逻辑' },
  { key: 'process', label: '处理' },
]

export const ALL_NODE_TYPES: NodeDef[] = [
  { type: 'start', label: '开始', icon: '▶', color: '#52c41a', category: 'basic' },
  { type: 'end', label: '结束', icon: '⏹', color: '#ff4d4f', category: 'basic' },
  { type: 'user_input', label: '用户输入', icon: '👤', color: '#faad14', category: 'basic' },
  { type: 'llm', label: 'LLM 调用', icon: '🤖', color: '#4a9eff', category: 'ai' },
  { type: 'image_gen', label: '图片生成', icon: '🖼', color: '#36cfc9', category: 'ai' },
  { type: 'audio_gen', label: '音频生成', icon: '🔊', color: '#9254de', category: 'ai' },
  { type: 'video_gen', label: '视频生成', icon: '🎬', color: '#f759ab', category: 'ai' },
  { type: 'condition', label: '条件分支', icon: '🔀', color: '#fa8c16', category: 'logic' },
  { type: 'loop', label: '循环执行', icon: '🔄', color: '#1890ff', category: 'logic' },
  { type: 'for_each', label: '并发批量', icon: '⚡', color: '#eb2f96', category: 'logic' },
  { type: 'iterator', label: '按序迭代', icon: '📋', color: '#fa541c', category: 'logic' },
  { type: 'split', label: '文本拆分', icon: '✂', color: '#722ed1', category: 'process' },
  { type: 'transform', label: '数据变换', icon: '⚙', color: '#13c2c2', category: 'process' },
  { type: 'script', label: '脚本', icon: '🐍', color: '#2f54eb', category: 'process' },
]

export function getNodeLabel(type: NodeType): string { return ALL_NODE_TYPES.find(n => n.type === type)?.label || type }
export function getNodeIcon(type: NodeType): string { return ALL_NODE_TYPES.find(n => n.type === type)?.icon || '●' }
export function getNodeColor(type: NodeType): string { return ALL_NODE_TYPES.find(n => n.type === type)?.color || '#999' }

export interface NodeTypeInfo {
  type: NodeType
  label: string
  description: string
}

export interface FlowExecution {
  id: number
  flow_id: number
  session_id: number
  status: 'created' | 'running' | 'waiting_user' | 'completed' | 'error'
  current_node_id: number | null
  context: string
  created_at: string
  updated_at: string
}

export interface FlowEvent {
  type: 'flow_node_start' | 'flow_node_chunk' | 'flow_node_done' | 'flow_waiting_user' | 'flow_complete' | 'flow_error'
  execution_id: number
  node_id?: number
  node_label?: string
  node_type?: string
  content?: string
  message?: string
  status?: string
}

export interface FlowSaveRequest {
  name: string
  description: string
  category: string
  config: string
  nodes: Omit<FlowNode, 'flow_id'>[]
  edges: Omit<FlowEdge, 'id' | 'flow_id'>[]
}
