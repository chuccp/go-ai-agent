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

export type NodeType = 'start' | 'end' | 'llm' | 'user_input' | 'for_each' | 'split' | 'transform' | 'condition' | 'script' | 'iterator' | 'loop'

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
  nodes: Omit<FlowNode, 'id' | 'flow_id'>[]
  edges: Omit<FlowEdge, 'id' | 'flow_id'>[]
}
