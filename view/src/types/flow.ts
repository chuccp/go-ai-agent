export type NodeType = 'start' | 'end' | 'llm' | 'user_input' | 'for_each' | 'split' | 'transform' | 'condition' | 'switch' | 'execute' | 'script' | 'iterator' | 'loop' | 'image_gen' | 'audio_gen' | 'video_gen'

export interface NodeDef {
  type: NodeType
  labelKey: string
  icon: string
  color: string
  category: string
}

export const NODE_CATEGORIES = [
  { key: 'basic', labelKey: 'nodeCategories.basic' },
  { key: 'ai', labelKey: 'nodeCategories.ai' },
  { key: 'logic', labelKey: 'nodeCategories.logic' },
  { key: 'process', labelKey: 'nodeCategories.process' },
] as const

export const ALL_NODE_TYPES: NodeDef[] = [
  { type: 'start', labelKey: 'nodes.start', icon: '▶', color: '#17b26a', category: 'basic' },
  { type: 'end', labelKey: 'nodes.end', icon: '⏹', color: '#f04438', category: 'basic' },
  { type: 'user_input', labelKey: 'nodes.userInput', icon: '👤', color: '#f79009', category: 'basic' },
  { type: 'llm', labelKey: 'nodes.llm', icon: '🤖', color: '#6366f1', category: 'ai' },
  { type: 'image_gen', labelKey: 'nodes.imageGen', icon: '🖼', color: '#36cfc9', category: 'ai' },
  { type: 'audio_gen', labelKey: 'nodes.audioGen', icon: '🔊', color: '#9254de', category: 'ai' },
  { type: 'video_gen', labelKey: 'nodes.videoGen', icon: '🎬', color: '#f759ab', category: 'ai' },
  { type: 'condition', labelKey: 'nodes.condition', icon: '🔀', color: '#f38744', category: 'logic' },
  { type: 'switch', labelKey: 'nodes.switch', icon: '🔘', color: '#d444f1', category: 'logic' },
  { type: 'loop', labelKey: 'nodes.loop', icon: '🔄', color: '#2e90fa', category: 'logic' },
  { type: 'for_each', labelKey: 'nodes.forEach', icon: '⚡', color: '#dd2590', category: 'logic' },
  { type: 'iterator', labelKey: 'nodes.iterator', icon: '📋', color: '#fa541c', category: 'logic' },
  { type: 'split', labelKey: 'nodes.split', icon: '✂', color: '#875bf7', category: 'process' },
  { type: 'transform', labelKey: 'nodes.transform', icon: '⚙', color: '#15b79e', category: 'process' },
  { type: 'execute', labelKey: 'nodes.execute', icon: '💻', color: '#0ea5e9', category: 'process' },
  { type: 'script', labelKey: 'nodes.script', icon: '🐍', color: '#6172f3', category: 'process' },
]

export interface FlowDefinition {
  id: number
  name: string
  description: string
  category: string
  path?: string
  config: Record<string, any>
  form_schema?: FormSchema | null
  settings?: FlowSettings | null
  icon?: string
  created_at: string
  updated_at: string
}

export interface FormSchema {
  fields: FormField[]
}

export interface FormField {
  name: string
  label: string
  type: 'text' | 'number' | 'textarea' | 'select' | 'radio' | 'checkbox' | 'file'
  required?: boolean
  default?: any
  options?: string[]
}

export interface FlowSettings {
  icon?: string
  default_model?: string
  timeout?: number
  allow_chat?: boolean
  [key: string]: any
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
  type: string
  execution_id: number
  node_id?: number
  node_label?: string
  node_type?: string
  content?: string
  message?: string
  status?: string
}

export function getNodeDef(type: NodeType): NodeDef | undefined {
  return ALL_NODE_TYPES.find(n => n.type === type)
}

/** Returns true if the icon string is a filename (e.g. "icon.png") rather than an emoji. */
export function isIconFilename(icon?: string): boolean {
  if (!icon) return false
  return icon.includes('.') && /\.(png|jpg|jpeg|gif|svg|webp)$/i.test(icon)
}
