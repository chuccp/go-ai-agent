import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useNavigate } from 'react-router-dom'
import ReactFlow, {
  Background,
  Handle,
  Position,
  MiniMap,
  Panel,
  EdgeLabelRenderer,
  BaseEdge,
  getBezierPath,
  useNodesState,
  useEdgesState,
  addEdge,
  useReactFlow,
  ReactFlowProvider,
  SelectionMode,
  type Node,
  type Edge,
  type Connection,
  type EdgeProps,
} from 'reactflow'
import 'reactflow/dist/style.css'
import { useFlowStore } from '@/stores/flowStore'
import { API_BASE } from '@/constants'
import { ALL_NODE_TYPES, NODE_CATEGORIES, getNodeDef, type NodeType, type FlowNode, type FlowEdge, type FlowDefinition } from '@/types/flow'

/* ============================================================
   Helper: convert backend FlowNode/FlowEdge <-> reactflow Node/Edge
   ============================================================ */

function flowNodeToRF(n: FlowNode): Node {
  const def = getNodeDef(n.type)
  return {
    id: String(n.id),
    type: 'flowNode',
    position: { x: n.position_x, y: n.position_y },
    data: {
      label: n.label || def?.labelKey || n.type,
      nodeType: n.type,
      config: (() => { try { return typeof n.config === 'string' ? JSON.parse(n.config) : (n.config || {}) } catch { return {} } })(),
      dbId: n.id,
      flowId: n.flow_id,
      groupId: n.group_id,
    },
  }
}

function flowEdgeToRF(e: FlowEdge, nodes?: FlowNode[]): Edge {
  // Determine label for condition branches
  let label = e.label || undefined
  if (!label && nodes) {
    const sourceNode = nodes.find(n => n.id === e.source_node_id)
    if (sourceNode?.type === 'condition') {
      label = e.source_handle === 'yes' ? 'TRUE' : e.source_handle === 'no' ? 'FALSE' : undefined
    }
  }
  return {
    id: String(e.id),
    source: String(e.source_node_id),
    target: String(e.target_node_id),
    sourceHandle: e.source_handle || undefined,
    targetHandle: e.target_handle || undefined,
    label,
    type: 'default',
    animated: true,
    style: { stroke: '#8b8fa3', strokeWidth: 2 },
    labelStyle: { fontSize: 10, fontWeight: 600, fill: '#676f83' },
    labelBgStyle: { fill: '#f9fafb', stroke: '#e2e8f0', strokeWidth: 0.5, rx: 4, ry: 4 },
    labelBgPadding: [4, 2] as [number, number],
  }
}

function rfNodeToFlowNode(rf: Node, flowId: number): FlowNode {
  return {
    id: rf.data.dbId || 0,
    flow_id: flowId,
    type: rf.data.nodeType as NodeType,
    label: rf.data.label,
    config: rf.data.config || {},
    position_x: rf.position.x,
    position_y: rf.position.y,
    group_id: rf.data.groupId || null,
  }
}

function rfEdgeToFlowEdge(rf: Edge, flowId: number): FlowEdge {
  const idNum = parseInt(rf.id, 10)
  return {
    id: isNaN(idNum) ? 0 : idNum,
    flow_id: flowId,
    source_node_id: parseInt(rf.source, 10),
    target_node_id: parseInt(rf.target, 10),
    source_handle: rf.sourceHandle || '',
    target_handle: rf.targetHandle || '',
    label: typeof rf.label === 'string' ? rf.label : '',
  }
}

/* ============================================================
   Inline SVG icon helper (no external icon library)
   ============================================================ */

function Icon({ d, size = 16, color = 'currentColor', style }: { d: string; size?: number; color?: string; style?: React.CSSProperties }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" style={style}>
      <path d={d} />
    </svg>
  )
}

const ICONS = {
  back: 'M19 12H5M12 19l-7-7 7-7',
  plus: 'M12 5v14M5 12h14',
  save: 'M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2zM17 21v-8H7v8M7 3v5h8',
  trash: 'M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2',
  export: 'M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3',
  import: 'M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M17 8l-5-5-5 5M12 3v12',
  copy: 'M20 9h-9a2 2 0 00-2 2v9m2-11h9a2 2 0 012 2v9M4 4h9a2 2 0 012 2v9H4a2 2 0 01-2-2V6a2 2 0 012-2z',
  edit: 'M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z',
  pointer: 'M3 3l7.07 16.97 2.51-7.39 7.39-2.51L3 3z',
  hand: 'M14 11V6a2 2 0 00-4 0v1M10 10V4a2 2 0 00-4 0v6m4 0v-1a2 2 0 00-4 0v4a8 8 0 0016 0v-5a2 2 0 00-4 0',
  layout: 'M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z',
  maximize: 'M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3',
  close: 'M18 6L6 18M6 6l12 12',
  search: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z',
  list: 'M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01',
}

/* ============================================================
   Custom Edge Component with midpoint "+" button
   ============================================================ */

function CustomEdgeWithAddButton(props: EdgeProps) {
  const { sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition, id, markerEnd, style, data } = props
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX: sourceX - 8,
    sourceY,
    targetX: targetX + 8,
    targetY,
    sourcePosition,
    targetPosition,
    curvature: 0.16,
  })
  const [hovered, setHovered] = useState(false)

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (data?.onEdgeAddClick) {
      data.onEdgeAddClick(id, { x: labelX, y: labelY })
    }
  }

  return (
    <>
      <path
        d={edgePath}
        markerEnd={markerEnd}
        style={{ ...style, strokeWidth: 2 }}
        fill="none"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      />
      <EdgeLabelRenderer>
        <div
          className="nopan nodrag"
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            pointerEvents: 'all',
            opacity: hovered ? 1 : 0,
            transition: 'opacity 0.15s',
          }}
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
        >
          <button
            style={{
              width: 16,
              height: 16,
              borderRadius: '50%',
              background: '#155aef',
              border: 'none',
              color: '#fff',
              fontSize: 12,
              lineHeight: 1,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'transform 0.15s',
            }}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform = 'scale(1.3)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'scale(1)' }}
            onClick={handleClick}
            title="Insert node"
          >
            +
          </button>
        </div>
      </EdgeLabelRenderer>
    </>
  )
}

/* ============================================================
   Custom Connection Line (while dragging)
   ============================================================ */

function CustomConnectionLine({ fromX, fromY, toX, toY }: { fromX: number; fromY: number; toX: number; toY: number }) {
  const [path] = getBezierPath({
    sourceX: fromX,
    sourceY: fromY,
    targetX: toX,
    targetY: toY,
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
    curvature: 0.16,
  })
  return (
    <g>
      <path d={path} fill="none" stroke="#d0d5dd" strokeWidth={2} />
      <rect x={toX - 1} y={toY - 4} width={2} height={8} fill="#2970ff" rx={1} />
    </g>
  )
}

const edgeTypes = { default: CustomEdgeWithAddButton }

/* ============================================================
   Custom Node Component (registered as 'flowNode')
   ============================================================ */

function FlowNodeComponent({ id, data, selected }: { id: string; data: any; selected: boolean }) {
  const { t } = useTranslation()
  const def = getNodeDef(data.nodeType as NodeType)
  const icon = def?.icon || '?'
  const color = def?.color || '#8b8fa3'
  const label = data.label || t(def?.labelKey || data.nodeType)
  const [hovered, setHovered] = useState(false)
  const canDelete = data.nodeType !== 'start' && data.nodeType !== 'end'

  const handleSourceClick = (handleId: string) => (e: React.MouseEvent) => {
    e.stopPropagation()
    if (data.onHandleClick) {
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
      data.onHandleClick(id, handleId, { x: rect.right + 8, y: rect.top })
    }
  }

  // build a small preview from config
  const preview = useMemo(() => {
    const c = data.config || {}
    switch (data.nodeType) {
      case 'llm': return c.model || ''
      case 'condition': return c.field ? `${c.field} ${c.operator || ''} ${c.value || ''}` : ''
      case 'user_input': return c.prompt_text || ''
      case 'transform': return c.template ? (c.template.substring(0, 40) + (c.template.length > 40 ? '...' : '')) : ''
      case 'script': return c.code ? (c.code.substring(0, 40) + (c.code.length > 40 ? '...' : '')) : ''
      case 'split': return c.delimiter || ''
      case 'loop': return c.max_iterations ? `max: ${c.max_iterations}` : ''
      case 'for_each':
      case 'iterator': return c.items_key || ''
      default: return ''
    }
  }, [data.config, data.nodeType])

  const isCondition = data.nodeType === 'condition'
  const isLoop = data.nodeType === 'loop'
  const hasInput = data.nodeType !== 'start'
  const hasOutput = data.nodeType !== 'end'

  return (
    <div
      style={{
        width: 240,
        background: '#fff',
        borderRadius: 12,
        border: selected ? `2px solid ${color}` : '1.5px solid #e2e8f0',
        boxShadow: selected ? `0 0 0 3px ${color}22` : '0 2px 8px rgba(0,0,0,0.06)',
        overflow: 'visible',
        transition: 'border-color 0.15s, box-shadow 0.15s',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Hover action bar */}
      {(hovered || selected) && canDelete && (
        <div
          style={{
            position: 'absolute',
            top: -28,
            right: 0,
            display: 'flex',
            gap: 2,
            background: 'rgba(255,255,255,0.95)',
            borderRadius: 8,
            padding: '2px 4px',
            border: '0.5px solid rgba(16,24,40,0.08)',
            boxShadow: '0 2px 6px rgba(0,0,0,0.08)',
            zIndex: 10,
          }}
          onMouseDown={e => e.stopPropagation()}
          onClick={e => e.stopPropagation()}
        >
          <button
            style={{ width: 22, height: 22, borderRadius: 4, border: 'none', background: 'transparent', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#676f83' }}
            title="Duplicate"
            onClick={() => data.onDuplicate?.(id)}
          >
            <Icon d={ICONS.copy} size={12} />
          </button>
          <button
            style={{ width: 22, height: 22, borderRadius: 4, border: 'none', background: 'transparent', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#f04438' }}
            title="Delete"
            onClick={() => data.onDelete?.(id)}
          >
            <Icon d={ICONS.trash} size={12} color="#f04438" />
          </button>
        </div>
      )}
      {hasInput && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ width: 10, height: 10, background: '#fff', border: `2px solid ${color}`, top: '50%' }}
        />
      )}
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '10px 12px',
        borderBottom: '1px solid #f2f4f7',
        borderRadius: '12px 12px 0 0',
        background: `${color}0d`,
      }}>
        <div style={{
          width: 28, height: 28, borderRadius: 8, background: `${color}1a`,
          display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14,
        }}>
          {icon}
        </div>
        <div style={{ flex: 1, fontSize: 13, fontWeight: 600, color: '#101828', lineHeight: '18px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {label}
        </div>
        <div style={{ fontSize: 10, color: '#676f83', background: '#f2f4f7', padding: '1px 6px', borderRadius: 4 }}>
          {data.nodeType}
        </div>
      </div>
      {/* Body preview */}
      {preview && (
        <div style={{ padding: '6px 12px 8px', fontSize: 11, color: '#676f83', lineHeight: '16px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {preview}
        </div>
      )}
      {/* Condition: two output handles with labels */}
      {isCondition && hasOutput && (
        <>
          <div style={{ position: 'absolute', right: -8, top: '35%', transform: 'translateY(-50%) translateX(100%)', fontSize: 9, fontWeight: 700, color: '#17b26a', paddingLeft: 4 }}>TRUE</div>
          <Handle
            type="source"
            position={Position.Right}
            id="yes"
            style={{ width: 10, height: 10, background: '#17b26a', border: '2px solid #fff', top: '35%', cursor: 'crosshair' }}
            onClick={handleSourceClick('yes')}
          />
          <div style={{ position: 'absolute', right: -8, top: '65%', transform: 'translateY(-50%) translateX(100%)', fontSize: 9, fontWeight: 700, color: '#f04438', paddingLeft: 4 }}>FALSE</div>
          <Handle
            type="source"
            position={Position.Right}
            id="no"
            style={{ width: 10, height: 10, background: '#f04438', border: '2px solid #fff', top: '65%', cursor: 'crosshair' }}
            onClick={handleSourceClick('no')}
          />
        </>
      )}
      {/* Loop: bottom output handle */}
      {isLoop && hasOutput && (
        <Handle
          type="source"
          position={Position.Bottom}
          id="loop_body"
          style={{ width: 10, height: 10, background: color, border: '2px solid #fff', cursor: 'crosshair' }}
          onClick={handleSourceClick('loop_body')}
        />
      )}
      {/* Default single output handle */}
      {hasOutput && !isCondition && !isLoop && (
        <Handle
          type="source"
          position={Position.Right}
          style={{ width: 10, height: 10, background: '#fff', border: `2px solid ${color}`, top: '50%', cursor: 'crosshair' }}
          onClick={handleSourceClick('source')}
        />
      )}
    </div>
  )
}

const nodeTypes = { flowNode: FlowNodeComponent }

/* ============================================================
   BlockSelector popup
   ============================================================ */

function BlockSelector({
  anchor,
  onPick,
  onClose,
}: {
  anchor: { top: number; left: number }
  onPick: (type: NodeType) => void
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    const q = search.toLowerCase()
    if (!q) return ALL_NODE_TYPES
    return ALL_NODE_TYPES.filter(
      n => t(n.labelKey).toLowerCase().includes(q) || n.type.toLowerCase().includes(q)
    )
  }, [search, t])

  const grouped = useMemo(() => {
    const map: Record<string, typeof ALL_NODE_TYPES> = {}
    for (const cat of NODE_CATEGORIES) map[cat.key] = []
    for (const n of filtered) {
      if (!map[n.category]) map[n.category] = []
      map[n.category].push(n)
    }
    return map
  }, [filtered])

  return (
    <div
      style={{
        position: 'fixed', top: anchor.top, left: anchor.left, zIndex: 1000,
        width: 260, maxHeight: 420, background: '#fff', borderRadius: 12,
        border: '0.5px solid rgba(16,24,40,0.08)', boxShadow: '0 8px 24px rgba(16,24,40,0.12)',
        display: 'flex', flexDirection: 'column', overflow: 'hidden',
      }}
    >
      {/* search */}
      <div style={{ padding: '10px 12px 8px', borderBottom: '1px solid #f2f4f7' }}>
        <input
          autoFocus
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder={t('flow.searchNodes')}
          style={{
            width: '100%', padding: '6px 10px', border: '1px solid #d0d5dd', borderRadius: 8,
            fontSize: 13, outline: 'none',
          }}
        />
      </div>
      {/* list */}
      <div style={{ overflowY: 'auto', flex: 1, padding: '6px 0' }}>
        {NODE_CATEGORIES.map(cat => {
          const items = grouped[cat.key] || []
          if (items.length === 0) return null
          return (
            <div key={cat.key}>
              <div style={{ padding: '6px 14px 2px', fontSize: 11, fontWeight: 600, color: '#98a2b3', textTransform: 'uppercase', letterSpacing: 0.5 }}>
                {t(cat.labelKey)}
              </div>
              {items.map(n => (
                <div
                  key={n.type}
                  onClick={() => { onPick(n.type); onClose() }}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 10, padding: '7px 14px',
                    cursor: 'pointer', transition: 'background 0.1s',
                  }}
                  onMouseEnter={e => (e.currentTarget.style.background = '#f9fafb')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <div style={{
                    width: 28, height: 28, borderRadius: 8, background: `${n.color}1a`,
                    display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, flexShrink: 0,
                  }}>
                    {n.icon}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 13, fontWeight: 500, color: '#101828' }}>{t(n.labelKey)}</div>
                    <div style={{ fontSize: 11, color: '#98a2b3' }}>{n.type}</div>
                  </div>
                </div>
              ))}
            </div>
          )
        })}
        {filtered.length === 0 && (
          <div style={{ padding: '20px 14px', textAlign: 'center', fontSize: 13, color: '#98a2b3' }}>
            {t('flow.noMatch')}
          </div>
        )}
      </div>
    </div>
  )
}

/* ============================================================
   PropertyPanel (slide-in right panel)
   ============================================================ */

function PropertyPanel({
  node,
  onChange,
  onDelete,
  onClose,
  llmModels,
}: {
  node: Node | null
  onChange: (updated: Node) => void
  onDelete: () => void
  onClose: () => void
  llmModels: { id: number; name: string; model_id: string }[]
}) {
  const { t } = useTranslation()
  if (!node) return null

  const def = getNodeDef(node.data.nodeType as NodeType)
  const color = def?.color || '#8b8fa3'
  const cfg = { ...node.data.config }

  const updateCfg = (key: string, value: any) => {
    onChange({
      ...node,
      data: { ...node.data, config: { ...cfg, [key]: value } },
    })
  }

  const updateLabel = (label: string) => {
    onChange({ ...node, data: { ...node.data, label } })
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '6px 10px', border: '1px solid #d0d5dd', borderRadius: 8,
    fontSize: 13, outline: 'none', background: '#fff',
  }
  const labelStyle: React.CSSProperties = {
    fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block',
  }
  const sectionStyle: React.CSSProperties = {
    padding: '12px 16px', borderBottom: '1px solid #f2f4f7',
  }

  /* ---- type-specific config forms ---- */

  const renderTypeConfig = () => {
    switch (node.data.nodeType) {
      case 'llm':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('flow.config')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.model')}</label>
              <select
                value={cfg.model || ''}
                onChange={e => updateCfg('model', e.target.value)}
                style={inputStyle}
              >
                <option value="">--</option>
                {llmModels.map(m => (
                  <option key={m.id} value={m.model_id}>{m.name} ({m.model_id})</option>
                ))}
              </select>
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.systemPrompt')}</label>
              <textarea
                value={cfg.system_prompt || ''}
                onChange={e => updateCfg('system_prompt', e.target.value)}
                rows={3}
                style={{ ...inputStyle, resize: 'vertical' }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.promptTemplate')}</label>
              <textarea
                value={cfg.prompt_template || ''}
                onChange={e => updateCfg('prompt_template', e.target.value)}
                rows={4}
                style={{ ...inputStyle, resize: 'vertical' }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.temperature')}: {cfg.temperature ?? 0.7}</label>
              <input
                type="range" min={0} max={2} step={0.05}
                value={cfg.temperature ?? 0.7}
                onChange={e => updateCfg('temperature', parseFloat(e.target.value))}
                style={{ width: '100%' }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.maxTokens')}</label>
              <input
                type="number"
                value={cfg.max_tokens ?? ''}
                onChange={e => updateCfg('max_tokens', e.target.value ? parseInt(e.target.value) : null)}
                style={inputStyle}
                placeholder="4096"
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.thinkingLevel')}</label>
              <select
                value={cfg.thinking_level || 'follow_model'}
                onChange={e => updateCfg('thinking_level', e.target.value)}
                style={inputStyle}
              >
                <option value="follow_model">{t('nodeConfig.followModel')}</option>
                <option value="off">{t('think.off')}</option>
                <option value="low">{t('think.low')}</option>
                <option value="medium">{t('think.medium')}</option>
                <option value="high">{t('think.high')}</option>
                <option value="max">{t('think.max')}</option>
              </select>
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.outputFormat')}</label>
              <select
                value={cfg.output_format || 'plain'}
                onChange={e => updateCfg('output_format', e.target.value)}
                style={inputStyle}
              >
                <option value="plain">{t('nodeConfig.plainText')}</option>
                <option value="json_auto">{t('nodeConfig.jsonAuto')}</option>
                <option value="json_object">{t('nodeConfig.jsonObject')}</option>
                <option value="json_array">{t('nodeConfig.jsonArray')}</option>
              </select>
            </div>
          </div>
        )

      case 'condition':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodeConfig.conditionJudge')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.checkField')}</label>
              <input
                value={cfg.field || ''}
                onChange={e => updateCfg('field', e.target.value)}
                style={inputStyle}
                placeholder="node_name.output"
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.operator')}</label>
              <select
                value={cfg.operator || 'contains'}
                onChange={e => updateCfg('operator', e.target.value)}
                style={inputStyle}
              >
                <option value="contains">{t('nodeConfig.contains')}</option>
                <option value="equals">{t('nodeConfig.equals')}</option>
                <option value="not_empty">{t('nodeConfig.notEmpty')}</option>
              </select>
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.compareValue')}</label>
              <input
                value={cfg.value || ''}
                onChange={e => updateCfg('value', e.target.value)}
                style={inputStyle}
              />
            </div>
          </div>
        )

      case 'user_input':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodeConfig.waitingInput')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.promptText')}</label>
              <textarea
                value={cfg.prompt_text || ''}
                onChange={e => updateCfg('prompt_text', e.target.value)}
                rows={3}
                style={{ ...inputStyle, resize: 'vertical' }}
              />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <input
                type="checkbox"
                checked={!!cfg.confirm_only}
                onChange={e => updateCfg('confirm_only', e.target.checked)}
                id="confirmOnly"
              />
              <label htmlFor="confirmOnly" style={{ fontSize: 13, color: '#354052', cursor: 'pointer' }}>
                {t('nodeConfig.confirmOnly')}
              </label>
            </div>
          </div>
        )

      case 'transform':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodeConfig.template')}</div>
            <div style={{ fontSize: 11, color: '#98a2b3', marginBottom: 8 }}>{t('nodeConfig.templateHint')}</div>
            <textarea
              value={cfg.template || ''}
              onChange={e => updateCfg('template', e.target.value)}
              rows={8}
              style={{ ...inputStyle, resize: 'vertical', fontFamily: 'monospace', fontSize: 12 }}
            />
          </div>
        )

      case 'script':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodeConfig.scriptCode')}</div>
            <div style={{ fontSize: 11, color: '#98a2b3', marginBottom: 8 }}>{t('nodeConfig.scriptHint')}</div>
            <textarea
              value={cfg.code || ''}
              onChange={e => updateCfg('code', e.target.value)}
              rows={10}
              style={{ ...inputStyle, resize: 'vertical', fontFamily: 'monospace', fontSize: 12 }}
            />
          </div>
        )

      case 'split':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodeConfig.splitMethod')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.dataSource')}</label>
              <input
                value={cfg.source_key || ''}
                onChange={e => updateCfg('source_key', e.target.value)}
                style={inputStyle}
                placeholder="node_name.output"
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.splitMethod')}</label>
              <select
                value={cfg.delimiter || 'paragraph'}
                onChange={e => updateCfg('delimiter', e.target.value)}
                style={inputStyle}
              >
                <option value="paragraph">{t('nodeConfig.byParagraph')}</option>
                <option value="line">{t('nodeConfig.byLine')}</option>
                <option value="comma">{t('nodeConfig.byComma')}</option>
                <option value="period">{t('nodeConfig.byPeriod')}</option>
                <option value="custom">Custom</option>
              </select>
            </div>
            {cfg.delimiter === 'custom' && (
              <div style={{ marginBottom: 10 }}>
                <label style={labelStyle}>Custom Delimiter</label>
                <input
                  value={cfg.custom_delimiter || ''}
                  onChange={e => updateCfg('custom_delimiter', e.target.value)}
                  style={inputStyle}
                />
              </div>
            )}
          </div>
        )

      case 'for_each':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 6 }}>{t('nodes.forEach')}</div>
            <div style={{ fontSize: 11, color: '#98a2b3', marginBottom: 10 }}>{t('nodeConfig.parallelHint')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.dataSource')}</label>
              <input
                value={cfg.items_key || ''}
                onChange={e => updateCfg('items_key', e.target.value)}
                style={inputStyle}
                placeholder="node_name.output"
              />
            </div>
          </div>
        )

      case 'iterator':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 6 }}>{t('nodes.iterator')}</div>
            <div style={{ fontSize: 11, color: '#98a2b3', marginBottom: 10 }}>{t('nodeConfig.sequentialHint')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.dataSource')}</label>
              <input
                value={cfg.items_key || ''}
                onChange={e => updateCfg('items_key', e.target.value)}
                style={inputStyle}
                placeholder="node_name.output"
              />
            </div>
          </div>
        )

      case 'loop':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodeConfig.loopBody')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.maxIterations')}</label>
              <input
                type="number"
                value={cfg.max_iterations ?? 10}
                onChange={e => updateCfg('max_iterations', parseInt(e.target.value) || 10)}
                style={inputStyle}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.breakField')}</label>
              <input
                value={cfg.break_field || ''}
                onChange={e => updateCfg('break_field', e.target.value)}
                style={inputStyle}
                placeholder="node_name.output"
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.breakOperator')}</label>
              <select
                value={cfg.break_operator || 'equals'}
                onChange={e => updateCfg('break_operator', e.target.value)}
                style={inputStyle}
              >
                <option value="contains">{t('nodeConfig.contains')}</option>
                <option value="equals">{t('nodeConfig.equals')}</option>
                <option value="not_empty">{t('nodeConfig.notEmpty')}</option>
              </select>
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.breakValue')}</label>
              <input
                value={cfg.break_value || ''}
                onChange={e => updateCfg('break_value', e.target.value)}
                style={inputStyle}
              />
            </div>
          </div>
        )

      case 'image_gen':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodes.imageGen')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.model')}</label>
              <input
                value={cfg.model || ''}
                onChange={e => updateCfg('model', e.target.value)}
                style={inputStyle}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.prompt')}</label>
              <textarea
                value={cfg.prompt || ''}
                onChange={e => updateCfg('prompt', e.target.value)}
                rows={3}
                style={{ ...inputStyle, resize: 'vertical' }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.count')}</label>
              <input
                type="number"
                value={cfg.count ?? 1}
                onChange={e => updateCfg('count', parseInt(e.target.value) || 1)}
                style={inputStyle}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.aspectRatio')}</label>
              <select
                value={cfg.aspect_ratio || '1:1'}
                onChange={e => updateCfg('aspect_ratio', e.target.value)}
                style={inputStyle}
              >
                <option value="1:1">1:1</option>
                <option value="16:9">16:9</option>
                <option value="9:16">9:16</option>
                <option value="4:3">4:3</option>
                <option value="3:4">3:4</option>
              </select>
            </div>
          </div>
        )

      case 'audio_gen':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodes.audioGen')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.model')}</label>
              <input
                value={cfg.model || ''}
                onChange={e => updateCfg('model', e.target.value)}
                style={inputStyle}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.text')}</label>
              <textarea
                value={cfg.text || ''}
                onChange={e => updateCfg('text', e.target.value)}
                rows={3}
                style={{ ...inputStyle, resize: 'vertical' }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.voice')}</label>
              <input
                value={cfg.voice || ''}
                onChange={e => updateCfg('voice', e.target.value)}
                style={inputStyle}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.duration')}</label>
              <input
                type="number"
                value={cfg.duration ?? ''}
                onChange={e => updateCfg('duration', e.target.value ? parseInt(e.target.value) : null)}
                style={inputStyle}
              />
            </div>
          </div>
        )

      case 'video_gen':
        return (
          <div style={sectionStyle}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('nodes.videoGen')}</div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.model')}</label>
              <input
                value={cfg.model || ''}
                onChange={e => updateCfg('model', e.target.value)}
                style={inputStyle}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.prompt')}</label>
              <textarea
                value={cfg.prompt || ''}
                onChange={e => updateCfg('prompt', e.target.value)}
                rows={3}
                style={{ ...inputStyle, resize: 'vertical' }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label style={labelStyle}>{t('nodeConfig.duration')}</label>
              <input
                type="number"
                value={cfg.duration ?? ''}
                onChange={e => updateCfg('duration', e.target.value ? parseInt(e.target.value) : null)}
                style={inputStyle}
              />
            </div>
          </div>
        )

      default:
        // start, end: no extra config
        return null
    }
  }

  return (
    <div style={{
      position: 'absolute', top: 0, right: 0, bottom: 0, width: 340,
      background: '#fff', borderLeft: '1px solid #e2e8f0',
      boxShadow: '-4px 0 16px rgba(0,0,0,0.06)', zIndex: 20,
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '12px 16px',
        borderBottom: '1px solid #f2f4f7', flexShrink: 0,
      }}>
        <div style={{ width: 28, height: 28, borderRadius: 8, background: `${color}1a`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 }}>
          {def?.icon || '?'}
        </div>
        <div style={{ flex: 1, fontSize: 14, fontWeight: 600, color: '#101828' }}>
          {t('flow.nodeProperties')}
        </div>
        <button
          onClick={onClose}
          style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4, borderRadius: 6, color: '#676f83', display: 'flex', alignItems: 'center' }}
        >
          <Icon d={ICONS.close} size={18} />
        </button>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {/* Basic info */}
        <div style={{ padding: '12px 16px', borderBottom: '1px solid #f2f4f7' }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: '#101828', marginBottom: 10 }}>{t('flow.basicInfo')}</div>
          <div style={{ marginBottom: 10 }}>
            <label style={labelStyle}>{t('common.name')}</label>
            <input
              value={node.data.label || ''}
              onChange={e => updateLabel(e.target.value)}
              style={inputStyle}
            />
          </div>
          <div>
            <label style={labelStyle}>{t('common.type')}</label>
            <div style={{ ...inputStyle, background: '#f9fafb', cursor: 'default' }}>
              {t(def?.labelKey || node.data.nodeType)} ({node.data.nodeType})
            </div>
          </div>
        </div>

        {/* Type-specific config */}
        {renderTypeConfig()}
      </div>

      {/* Footer: delete button */}
      {node.data.nodeType !== 'start' && node.data.nodeType !== 'end' && (
        <div style={{ padding: '12px 16px', borderTop: '1px solid #f2f4f7', flexShrink: 0 }}>
          <button
            onClick={onDelete}
            style={{
              width: '100%', padding: '8px 0', borderRadius: 8,
              border: '1px solid #f04438', background: '#fef3f2', color: '#f04438',
              fontSize: 13, fontWeight: 500, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
            }}
          >
            <Icon d={ICONS.trash} size={14} color="#f04438" />
            {t('flow.deleteNode')}
          </button>
        </div>
      )}
    </div>
  )
}

/* ============================================================
   Flow List View
   ============================================================ */

function FlowListView() {
  const { t } = useTranslation()
  const nav = useNavigate()
  const { flows, fetchFlows, deleteFlow, saveFlow } = useFlowStore()
  const [creating, setCreating] = useState(false)
  const [showNewModal, setShowNewModal] = useState(false)
  const [newName, setNewName] = useState('')
  const [newCategory, setNewCategory] = useState('')
  const [filterCategory, setFilterCategory] = useState('')
  const [importing, setImporting] = useState(false)

  useEffect(() => { fetchFlows() }, [])

  const categories = useMemo(() => {
    const set = new Set<string>()
    for (const f of flows) { if (f.category) set.add(f.category) }
    return Array.from(set)
  }, [flows])

  const displayed = useMemo(() => {
    if (!filterCategory) return flows
    return flows.filter(f => f.category === filterCategory)
  }, [flows, filterCategory])

  const handleCreate = async () => {
    if (!newName.trim()) return
    setCreating(true)
    const saved = await saveFlow({ name: newName.trim(), category: newCategory || 'uncategorized', nodes: [], edges: [] })
    setCreating(false)
    setShowNewModal(false)
    setNewName('')
    setNewCategory('')
    if (saved?.id) nav(`/designer/${saved.id}`)
  }

  const handleDuplicate = async (id: number) => {
    try {
      const res = await fetch(`${API_BASE}/api/flows/${id}/duplicate`, { method: 'POST' })
      if (res.ok) fetchFlows()
    } catch (e) { console.error(e) }
  }

  const handleExportCard = async (id: number, name: string) => {
    try {
      const res = await fetch(`${API_BASE}/api/flows/${id}`)
      const body = await res.json()
      const detail = body.data
      if (!detail) return
      const data = {
        type: 'go-ai-agent-flow',
        version: 1,
        name: detail.name,
        description: detail.description,
        category: detail.category,
        nodes: detail.nodes || [],
        edges: detail.edges || [],
      }
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${detail.name || 'flow'}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) { console.error(e) }
  }

  const handleDelete = async (id: number) => {
    if (!window.confirm(t('flow.confirmDelete'))) return
    await deleteFlow(id)
  }

  const handleImport = async () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.json'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      setImporting(true)
      try {
        const text = await file.text()
        const data = JSON.parse(text)
        // Support both versioned format and raw flow JSON
        if (data.type === 'go-ai-agent-flow') {
          // version 1: extract name/description/category/nodes/edges
          if (!data.name) { alert(t('flow.jsonError')); return }
          const saved = await saveFlow({
            name: data.name,
            description: data.description || '',
            category: data.category || 'uncategorized',
            nodes: data.nodes || [],
            edges: data.edges || [],
          })
          if (saved?.id) fetchFlows()
        } else if (data.nodes || data.name) {
          // Raw flow JSON (no type marker) — still supported
          const saved = await saveFlow(data)
          if (saved?.id) fetchFlows()
        } else {
          alert(t('flow.jsonError'))
        }
      } catch { alert(t('flow.jsonError')) }
      finally { setImporting(false) }
    }
    input.click()
  }

  const btnStyle = (primary = false): React.CSSProperties => ({
    padding: '7px 16px', borderRadius: 8, border: primary ? 'none' : '1px solid #d0d5dd',
    background: primary ? '#155aef' : '#fff', color: primary ? '#fff' : '#354052',
    fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 5,
  })

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: '#f2f4f7' }}>
      {/* Toolbar */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px',
        background: '#fff', borderBottom: '0.5px solid rgba(16,24,40,0.08)', flexShrink: 0,
      }}>
        <a href="#/" style={{ display: 'flex', alignItems: 'center', color: '#676f83', textDecoration: 'none', fontSize: 13, gap: 4 }}>
          <Icon d={ICONS.back} size={16} /> {t('common.back')}
        </a>
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <select
            value={filterCategory}
            onChange={e => setFilterCategory(e.target.value)}
            style={{ padding: '5px 8px', border: '1px solid #d0d5dd', borderRadius: 6, fontSize: 12, outline: 'none' }}
          >
            <option value="">{t('common.category')}: {t('common.default')}</option>
            {categories.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
        <button onClick={handleImport} style={btnStyle()} disabled={importing}>
          <Icon d={ICONS.import} size={14} /> {t('common.import')}
        </button>
        <button onClick={() => setShowNewModal(true)} style={btnStyle(true)}>
          <Icon d={ICONS.plus} size={14} color="#fff" /> {t('flow.newFlow')}
        </button>
      </div>

      {/* Grid */}
      <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
        {displayed.length === 0 ? (
          <div style={{ textAlign: 'center', color: '#98a2b3', marginTop: 80, fontSize: 14 }}>
            {t('flow.noFlows')}
          </div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 14 }}>
            {displayed.map(f => (
              <div key={f.id} style={{
                background: '#fff', borderRadius: 12, border: '0.5px solid rgba(16,24,40,0.08)',
                padding: 16, display: 'flex', flexDirection: 'column', gap: 8,
                boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
                transition: 'box-shadow 0.15s',
              }}
                onMouseEnter={e => (e.currentTarget.style.boxShadow = '0 4px 12px rgba(0,0,0,0.08)')}
                onMouseLeave={e => (e.currentTarget.style.boxShadow = '0 1px 4px rgba(0,0,0,0.04)')}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <div style={{ fontSize: 15, fontWeight: 600, color: '#101828', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {f.name}
                  </div>
                  {f.category && (
                    <span style={{ fontSize: 10, color: '#155aef', background: '#eff4ff', padding: '2px 8px', borderRadius: 10, flexShrink: 0 }}>
                      {f.category}
                    </span>
                  )}
                </div>
                {f.description && (
                  <div style={{ fontSize: 12, color: '#676f83', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {f.description}
                  </div>
                )}
                <div style={{ fontSize: 11, color: '#98a2b3' }}>
                  {f.updated_at ? new Date(f.updated_at).toLocaleString() : ''}
                </div>
                <div style={{ display: 'flex', gap: 6, marginTop: 'auto' }}>
                  <button
                    onClick={() => nav(`/designer/${f.id}`)}
                    style={{ ...btnStyle(), flex: 1, justifyContent: 'center' }}
                  >
                    <Icon d={ICONS.edit} size={13} /> {t('common.edit')}
                  </button>
                  <button
                    onClick={() => handleDuplicate(f.id)}
                    style={{ ...btnStyle(), justifyContent: 'center' }}
                  >
                    <Icon d={ICONS.copy} size={13} />
                  </button>
                  <button
                    onClick={() => handleExportCard(f.id, f.name)}
                    style={{ ...btnStyle(), justifyContent: 'center' }}
                    title={t('flow.exportHint')}
                  >
                    <Icon d={ICONS.export} size={13} />
                  </button>
                  <button
                    onClick={() => handleDelete(f.id)}
                    style={{ ...btnStyle(), padding: '7px 10px', color: '#f04438', borderColor: '#f0443833' }}
                  >
                    <Icon d={ICONS.trash} size={13} color="#f04438" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* New flow modal */}
      {showNewModal && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.3)', zIndex: 200,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={() => setShowNewModal(false)}>
          <div
            style={{ background: '#fff', borderRadius: 16, padding: 24, width: 380, boxShadow: '0 12px 32px rgba(0,0,0,0.15)' }}
            onClick={e => e.stopPropagation()}
          >
            <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 16 }}>{t('flow.newFlow')}</div>
            <div style={{ marginBottom: 12 }}>
              <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>{t('flow.flowName')}</label>
              <input
                autoFocus
                value={newName}
                onChange={e => setNewName(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleCreate()}
                style={{ width: '100%', padding: '8px 10px', border: '1px solid #d0d5dd', borderRadius: 8, fontSize: 13, outline: 'none' }}
                placeholder={t('flow.enterName')}
              />
            </div>
            <div style={{ marginBottom: 16 }}>
              <label style={{ fontSize: 12, color: '#354052', fontWeight: 500, marginBottom: 4, display: 'block' }}>{t('common.category')}</label>
              <input
                value={newCategory}
                onChange={e => setNewCategory(e.target.value)}
                style={{ width: '100%', padding: '8px 10px', border: '1px solid #d0d5dd', borderRadius: 8, fontSize: 13, outline: 'none' }}
                placeholder={t('flow.uncategorized')}
              />
            </div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button onClick={() => setShowNewModal(false)} style={btnStyle()}>{t('common.cancel')}</button>
              <button onClick={handleCreate} disabled={creating || !newName.trim()} style={btnStyle(true)}>
                {creating ? t('common.loading') : t('common.create')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

/* ============================================================
   Zoom Controls (rendered inside ReactFlow Panel)
   ============================================================ */

function ZoomControls() {
  const { zoomIn, zoomOut, fitView } = useReactFlow()
  const btnStyle: React.CSSProperties = {
    width: 28, height: 28, borderRadius: 6, border: 'none',
    background: 'transparent', color: '#354052', cursor: 'pointer',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    fontSize: 16, fontWeight: 500, padding: 0,
  }
  return (
    <>
      <button onClick={() => zoomOut()} style={btnStyle} title="Zoom Out">−</button>
      <button onClick={() => zoomIn()} style={btnStyle} title="Zoom In">+</button>
      <button onClick={() => fitView({ padding: 0.2 })} style={btnStyle} title="Fit View">⊡</button>
    </>
  )
}

/* ============================================================
   Flow Editor View
   ============================================================ */

function FlowEditor({ flowId }: { flowId: string }) {
  const { t } = useTranslation()
  const nav = useNavigate()
  const { fetchFlow, saveFlow, deleteFlow } = useFlowStore()

  // editor state
  const [flowName, setFlowName] = useState('')
  const [flowCategory, setFlowCategory] = useState('')
  const [dbFlowId, setDbFlowId] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [error, setError] = useState('')

  // reactflow state
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  // UI state
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [showBlockSelector, setShowBlockSelector] = useState(false)
  const [blockSelectorPos, setBlockSelectorPos] = useState({ top: 0, left: 0 })
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [interactionMode, setInteractionMode] = useState<'pointer' | 'hand'>('pointer')
  const [llmModels, setLlmModels] = useState<{ id: number; name: string; model_id: string }[]>([])
  const [edgeInsertInfo, setEdgeInsertInfo] = useState<{ edgeId: string; sourceId: string; sourceHandle?: string; targetId: string; targetHandle?: string; pos: { x: number; y: number } } | null>(null)

  // node ID counter (for new nodes)
  const [nextNodeId, setNextNodeId] = useState(10000)

  // Undo/Redo history refs
  const historyRef = useRef<{ nodes: Node[]; edges: Edge[] }[]>([])
  const historyIndexRef = useRef(-1)

  // auto-save debounce timer
  const autoSaveTimerRef = (function () {
    // use a mutable ref pattern without useRef
    let timer: ReturnType<typeof setTimeout> | null = null
    return {
      get: () => timer,
      set: (t: ReturnType<typeof setTimeout> | null) => { timer = t },
    }
  })()

  /* ---- Load flow ---- */
  useEffect(() => {
    if (flowId === 'new') {
      // blank canvas with start + end
      const startId = '1'
      const endId = '2'
      setNodes([
        { id: startId, type: 'flowNode', position: { x: 100, y: 200 }, data: { label: t('nodes.start'), nodeType: 'start', config: {}, dbId: 1 } },
        { id: endId, type: 'flowNode', position: { x: 600, y: 200 }, data: { label: t('nodes.end'), nodeType: 'end', config: {}, dbId: 2 } },
      ])
      setEdges([])
      setFlowName('')
      setFlowCategory('')
      setDbFlowId(null)
      setNextNodeId(3)
      setLoading(false)
    } else {
      setLoading(true)
      fetchFlow(parseInt(flowId, 10)).then(detail => {
        if (!detail) { setError('Flow not found'); setLoading(false); return }
        setFlowName(detail.name)
        setFlowCategory(detail.category || '')
        setDbFlowId(detail.id)
        const rfn = (detail.nodes || []).map(flowNodeToRF)
        const rfe = (detail.edges || []).map(e => flowEdgeToRF(e, detail.nodes))
        setNodes(rfn)
        setEdges(rfe)
        // compute next ID
        const maxId = Math.max(0, ...rfn.map(n => parseInt(n.id, 10)).filter(n => !isNaN(n)))
        setNextNodeId(maxId + 1)
        setLoading(false)
      })
    }
  }, [flowId])

  /* ---- Load LLM models ---- */
  useEffect(() => {
    fetch(`${API_BASE}/api/ai-models?category=llm`)
      .then(r => r.json())
      .then(d => setLlmModels((d.data || []).map((m: any) => ({ id: m.id, name: m.name, model_id: m.model_id }))))
      .catch(() => {})
  }, [])

  /* ---- Auto-save with debounce (only when editing existing flow) ---- */
  useEffect(() => {
    if (!dbFlowId) return
    if (!dirty) return
    autoSaveTimerRef.set(setTimeout(() => { handleSave(true) }, 800))
    return () => { const t = autoSaveTimerRef.get(); if (t) clearTimeout(t) }
  }, [dirty, nodes, edges, flowName, flowCategory])

  const markDirty = useCallback(() => setDirty(true), [])

  /* ---- reactflow callbacks ---- */
  const onConnect = useCallback((connection: Connection) => {
    // Prevent self-loops
    if (connection.source === connection.target) return
    // Prevent duplicate edges
    const exists = edges.some(e =>
      e.source === connection.source && e.target === connection.target &&
      e.sourceHandle === connection.sourceHandle && e.targetHandle === connection.targetHandle
    )
    if (exists) return
    // Determine label for condition branches
    const sourceNode = nodes.find(n => n.id === connection.source)
    let label: string | undefined
    if (sourceNode?.data.nodeType === 'condition') {
      label = connection.sourceHandle === 'yes' ? 'TRUE' : connection.sourceHandle === 'no' ? 'FALSE' : undefined
    }
    setEdges(eds => addEdge({
      ...connection,
      type: 'default',
      animated: true,
      style: { stroke: '#8b8fa3', strokeWidth: 2 },
      label,
      labelStyle: { fontSize: 10, fontWeight: 600, fill: '#676f83' },
      labelBgStyle: { fill: '#f9fafb', stroke: '#e2e8f0', strokeWidth: 0.5, rx: 4, ry: 4 },
      labelBgPadding: [4, 2] as [number, number],
    }, eds))
    markDirty()
  }, [setEdges, markDirty, edges, nodes])

  const onNodeClick = useCallback((_: any, node: Node) => {
    setSelectedNodeId(node.id)
  }, [])

  const onPaneClick = useCallback(() => {
    setSelectedNodeId(null)
  }, [])

  // mark dirty on any node/edge change
  const handleNodesChange: typeof onNodesChange = useCallback((changes) => {
    onNodesChange(changes)
    markDirty()
  }, [onNodesChange, markDirty])

  const handleEdgesChange: typeof onEdgesChange = useCallback((changes) => {
    onEdgesChange(changes)
    markDirty()
  }, [onEdgesChange, markDirty])

  /* ---- Selected node ---- */
  const selectedNode = useMemo(() => nodes.find(n => n.id === selectedNodeId) || null, [nodes, selectedNodeId])

  const handleSelectedNodeChange = useCallback((updated: Node) => {
    setNodes(nds => nds.map(n => n.id === updated.id ? updated : n))
    markDirty()
  }, [setNodes, markDirty])

  const handleDeleteSelectedNode = useCallback(() => {
    if (!selectedNodeId) return
    const node = nodes.find(n => n.id === selectedNodeId)
    if (node?.data.nodeType === 'start' || node?.data.nodeType === 'end') return
    setNodes(nds => nds.filter(n => n.id !== selectedNodeId))
    setEdges(eds => eds.filter(e => e.source !== selectedNodeId && e.target !== selectedNodeId))
    setSelectedNodeId(null)
    markDirty()
  }, [selectedNodeId, nodes, setNodes, setEdges, markDirty])

  /* ---- Edge add click handler ---- */
  const { flowToScreenPosition, fitView, zoomIn, zoomOut } = useReactFlow()

  const handleEdgeAddClick = useCallback((edgeId: string, pos: { x: number; y: number }) => {
    const edge = edges.find(e => e.id === edgeId)
    if (!edge) return
    setEdgeInsertInfo({
      edgeId,
      sourceId: edge.source,
      sourceHandle: edge.sourceHandle || undefined,
      targetId: edge.target,
      targetHandle: edge.targetHandle || undefined,
      pos,
    })
    // Convert flow coordinates to screen coordinates for the block selector
    const screenPos = flowToScreenPosition({ x: pos.x, y: pos.y })
    setBlockSelectorPos({ top: screenPos.y, left: screenPos.x + 8 })
    setShowBlockSelector(true)
  }, [edges, flowToScreenPosition])

  // Add onEdgeAddClick callback to edge data
  const edgesWithCallback = useMemo(() => {
    return edges.map(e => ({
      ...e,
      data: { ...e.data, onEdgeAddClick: handleEdgeAddClick },
    }))
  }, [edges, handleEdgeAddClick])

  // Handle source handle click - open block selector to add connected node
  const handleSourceHandleClick = useCallback((nodeId: string, handleId: string, pos: { x: number; y: number }) => {
    setEdgeInsertInfo({
      edgeId: '',  // No edge to remove, just creating new connection
      sourceId: nodeId,
      sourceHandle: handleId,
      targetId: '',
      targetHandle: 'target',
      pos,
    })
    setBlockSelectorPos({ top: pos.y, left: pos.x })
    setShowBlockSelector(true)
  }, [])

  // Add onHandleClick callback to node data
  const nodesWithCallback = useMemo(() => {
    return nodes.map(n => ({
      ...n,
      data: {
        ...n.data,
        onHandleClick: handleSourceHandleClick,
        onDelete: (nodeId: string) => {
          const node = nodes.find(nd => nd.id === nodeId)
          if (node?.data.nodeType === 'start' || node?.data.nodeType === 'end') return
          setNodes(nds => nds.filter(nd => nd.id !== nodeId))
          setEdges(eds => eds.filter(e => e.source !== nodeId && e.target !== nodeId))
          if (selectedNodeId === nodeId) setSelectedNodeId(null)
          markDirty()
        },
        onDuplicate: (nodeId: string) => {
          const node = nodes.find(nd => nd.id === nodeId)
          if (!node) return
          const newId = String(nextNodeId)
          const newNode: Node = {
            ...node,
            id: newId,
            position: { x: node.position.x + 50, y: node.position.y + 50 },
            data: { ...node.data, dbId: 0 },
          }
          setNodes(nds => [...nds, newNode])
          setNextNodeId(nid => nid + 1)
          markDirty()
        },
      },
    }))
  }, [nodes, handleSourceHandleClick, selectedNodeId, nextNodeId, setNodes, setEdges, markDirty])

  /* ---- Add node from block selector ---- */
  const handleAddNode = useCallback((type: NodeType) => {
    const id = String(nextNodeId)
    const def = getNodeDef(type)

    let position: { x: number; y: number }

    if (edgeInsertInfo) {
      if (edgeInsertInfo.edgeId) {
        // Insert node between two connected nodes
        const sourceNode = nodes.find(n => n.id === edgeInsertInfo.sourceId)
        const targetNode = nodes.find(n => n.id === edgeInsertInfo.targetId)
        const midX = sourceNode && targetNode ? (sourceNode.position.x + targetNode.position.x) / 2 : edgeInsertInfo.pos.x
        const midY = sourceNode && targetNode ? (sourceNode.position.y + targetNode.position.y) / 2 : edgeInsertInfo.pos.y
        position = { x: midX, y: midY }
      } else {
        // Add new node connected to source handle
        const sourceNode = nodes.find(n => n.id === edgeInsertInfo.sourceId)
        position = sourceNode
          ? { x: sourceNode.position.x + 300, y: sourceNode.position.y }
          : { x: edgeInsertInfo.pos.x, y: edgeInsertInfo.pos.y }
      }
    } else {
      // Place near center of viewport
      position = { x: 200 + Math.random() * 200, y: 150 + Math.random() * 200 }
    }

    // Create the node first
    const newNode: Node = {
      id,
      type: 'flowNode',
      position,
      data: {
        label: t(def?.labelKey || type),
        nodeType: type,
        config: {},
        dbId: 0,
      },
    }
    setNodes(nds => [...nds, newNode])

    // Then create edges (after node exists)
    if (edgeInsertInfo) {
      if (edgeInsertInfo.edgeId) {
        // Insert node between two connected nodes
        setEdges(eds => {
          const filtered = eds.filter(e => e.id !== edgeInsertInfo.edgeId)
          return [...filtered,
            {
              id: `e-${edgeInsertInfo.sourceId}-${id}`,
              source: edgeInsertInfo.sourceId,
              target: id,
              sourceHandle: edgeInsertInfo.sourceHandle,
              targetHandle: 'target',
              type: 'default',
              animated: true,
              style: { stroke: '#8b8fa3', strokeWidth: 2 },
            },
            {
              id: `e-${id}-${edgeInsertInfo.targetId}`,
              source: id,
              target: edgeInsertInfo.targetId,
              sourceHandle: 'source',
              targetHandle: edgeInsertInfo.targetHandle,
              type: 'default',
              animated: true,
              style: { stroke: '#8b8fa3', strokeWidth: 2 },
            },
          ]
        })
      } else {
        // Create edge from source handle to new node
        setEdges(eds => [...eds, {
          id: `e-${edgeInsertInfo.sourceId}-${id}`,
          source: edgeInsertInfo.sourceId,
          target: id,
          sourceHandle: edgeInsertInfo.sourceHandle,
          targetHandle: 'target',
          type: 'default',
          animated: true,
          style: { stroke: '#8b8fa3', strokeWidth: 2 },
        }])
      }
      setEdgeInsertInfo(null)
    }

    setNextNodeId(nid => nid + 1)
    markDirty()
  }, [nextNodeId, edgeInsertInfo, nodes, setNodes, setEdges, markDirty, t])

  /* ---- Save ---- */
  const handleSave = useCallback(async (silent = false) => {
    if (!silent) {
      // validate
      const hasStart = nodes.some(n => n.data.nodeType === 'start')
      const hasEnd = nodes.some(n => n.data.nodeType === 'end')
      if (!hasStart || !hasEnd) {
        setError(t('flow.needStartEnd'))
        return
      }
      if (!flowName.trim()) {
        setError(t('flow.enterName'))
        return
      }
    }
    setError('')
    setSaving(true)
    const payload = {
      name: flowName.trim() || 'Untitled',
      category: flowCategory || 'uncategorized',
      nodes: nodes.map(n => rfNodeToFlowNode(n, dbFlowId || 0)),
      edges: edges.map(e => rfEdgeToFlowEdge(e, dbFlowId || 0)),
    }
    const saved = await saveFlow(payload, dbFlowId || undefined)
    setSaving(false)
    if (saved) {
      setDirty(false)
      if (saved.id && !dbFlowId) {
        setDbFlowId(saved.id)
        // navigate to the proper URL without reloading
        window.location.hash = `#/designer/${saved.id}`
      }
      if (!silent) {
        // brief success indication handled by dirty clearing
      }
    } else if (!silent) {
      setError(t('flow.saveFailed'))
    }
  }, [nodes, edges, flowName, flowCategory, dbFlowId, saveFlow, t])

  /* ---- Export ---- */
  const handleExport = useCallback(() => {
    const data = {
      type: 'go-ai-agent-flow',
      version: 1,
      name: flowName,
      category: flowCategory,
      nodes: nodes.map(n => rfNodeToFlowNode(n, 0)),
      edges: edges.map(e => rfEdgeToFlowEdge(e, 0)),
    }
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${flowName || 'flow'}.json`
    a.click()
    URL.revokeObjectURL(url)
  }, [flowName, flowCategory, nodes, edges])

  /* ---- Import ---- */
  const handleImport = useCallback(() => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.json'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      try {
        const text = await file.text()
        const data = JSON.parse(text)
        // Support both versioned ("go-ai-agent-flow") and raw flow JSON
        if (data.nodes && Array.isArray(data.nodes)) {
          const rfn = data.nodes.map((n: FlowNode) => flowNodeToRF(n))
          const rfe = (data.edges || []).map((e: FlowEdge) => flowEdgeToRF(e, data.nodes))
          setNodes(rfn)
          setEdges(rfe)
          if (data.name) setFlowName(data.name)
          if (data.category) setFlowCategory(data.category)
          const maxId = Math.max(0, ...rfn.map((n: Node) => parseInt(n.id, 10)).filter((n: number) => !isNaN(n)))
          setNextNodeId(maxId + 1)
          markDirty()
        } else {
          alert(t('flow.jsonError'))
        }
      } catch { alert(t('flow.jsonError')) }
    }
    input.click()
  }, [setNodes, setEdges, markDirty, t])

  /* ---- Delete flow ---- */
  const handleDeleteFlow = useCallback(async () => {
    if (!dbFlowId) return
    if (!window.confirm(t('flow.confirmDelete'))) return
    await deleteFlow(dbFlowId)
    nav('/designer')
  }, [dbFlowId, deleteFlow, nav, t])

  /* ---- Auto-layout (simple: arrange nodes in a horizontal line) ---- */
  const handleAutoLayout = useCallback(() => {
    // topological sort
    const adj: Record<string, string[]> = {}
    const inDeg: Record<string, number> = {}
    for (const n of nodes) { adj[n.id] = []; inDeg[n.id] = 0 }
    for (const e of edges) {
      if (adj[e.source]) adj[e.source].push(e.target)
      inDeg[e.target] = (inDeg[e.target] || 0) + 1
    }
    const queue: string[] = []
    for (const id in inDeg) { if (inDeg[id] === 0) queue.push(id) }
    const sorted: string[] = []
    while (queue.length) {
      const id = queue.shift()!
      sorted.push(id)
      for (const next of adj[id] || []) {
        inDeg[next]--
        if (inDeg[next] === 0) queue.push(next)
      }
    }
    // include any nodes not in sorted (cycles)
    for (const n of nodes) { if (!sorted.includes(n.id)) sorted.push(n.id) }

    // assign positions: x by layer, y stacked
    const xGap = 300
    const yGap = 120
    const posMap: Record<string, { x: number; y: number }> = {}
    const layerMap: Record<string, number> = {}
    // BFS to assign layers
    const visited = new Set<string>()
    const startNode = nodes.find(n => n.data.nodeType === 'start')
    if (startNode) {
      const bfsQueue: { id: string; layer: number }[] = [{ id: startNode.id, layer: 0 }]
      visited.add(startNode.id)
      layerMap[startNode.id] = 0
      while (bfsQueue.length) {
        const { id, layer } = bfsQueue.shift()!
        for (const next of adj[id] || []) {
          if (!visited.has(next)) {
            visited.add(next)
            layerMap[next] = layer + 1
            bfsQueue.push({ id: next, layer: layer + 1 })
          }
        }
      }
    }
    // assign remaining nodes
    for (const n of nodes) {
      if (!(n.id in layerMap)) layerMap[n.id] = 0
    }
    // group by layer
    const layers: Record<number, string[]> = {}
    for (const [id, layer] of Object.entries(layerMap)) {
      if (!layers[layer]) layers[layer] = []
      layers[layer].push(id)
    }
    // position
    const updatedNodes = nodes.map(n => {
      const layer = layerMap[n.id] || 0
      const idx = (layers[layer] || []).indexOf(n.id)
      return {
        ...n,
        position: { x: 100 + layer * xGap, y: 100 + idx * yGap },
      }
    })
    setNodes(updatedNodes)
    markDirty()
  }, [nodes, edges, setNodes, markDirty])

  /* ---- Fullscreen toggle ---- */
  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen()
      setIsFullscreen(true)
    } else {
      document.exitFullscreen()
      setIsFullscreen(false)
    }
  }, [])

  /* ---- Undo/Redo ---- */
  const pushHistory = useCallback(() => {
    const snapshot = { nodes: JSON.parse(JSON.stringify(nodes)), edges: JSON.parse(JSON.stringify(edges)) }
    const idx = historyIndexRef.current
    historyRef.current = historyRef.current.slice(0, idx + 1)
    historyRef.current.push(snapshot)
    if (historyRef.current.length > 50) historyRef.current.shift()
    historyIndexRef.current = historyRef.current.length - 1
  }, [nodes, edges])

  const undo = useCallback(() => {
    const idx = historyIndexRef.current
    if (idx <= 0) return
    historyIndexRef.current = idx - 1
    const snapshot = historyRef.current[idx - 1]
    setNodes(snapshot.nodes)
    setEdges(snapshot.edges)
    markDirty()
  }, [setNodes, setEdges, markDirty])

  const redo = useCallback(() => {
    const idx = historyIndexRef.current
    if (idx >= historyRef.current.length - 1) return
    historyIndexRef.current = idx + 1
    const snapshot = historyRef.current[idx + 1]
    setNodes(snapshot.nodes)
    setEdges(snapshot.edges)
    markDirty()
  }, [setNodes, setEdges, markDirty])

  /* ---- Clipboard for copy/paste ---- */
  const clipboardRef = useRef<{ nodes: Node[]; edges: Edge[] } | null>(null)

  const handleCopy = useCallback(() => {
    const selectedNodes = nodes.filter(n => n.selected)
    if (selectedNodes.length === 0) return
    const selectedNodeIds = new Set(selectedNodes.map(n => n.id))
    const selectedEdges = edges.filter(e => selectedNodeIds.has(e.source) && selectedNodeIds.has(e.target))
    clipboardRef.current = { nodes: selectedNodes, edges: selectedEdges }
  }, [nodes, edges])

  const handlePaste = useCallback(() => {
    if (!clipboardRef.current) return
    pushHistory()
    const idMap: Record<string, string> = {}
    const newNodes = clipboardRef.current.nodes.map(n => {
      const newId = String(nextNodeId + Object.keys(idMap).length)
      idMap[n.id] = newId
      return { ...n, id: newId, position: { x: n.position.x + 50, y: n.position.y + 50 }, selected: false, data: { ...n.data, dbId: 0 } }
    })
    const newEdges = clipboardRef.current.edges.map(e => ({
      ...e,
      id: `e-${idMap[e.source] || e.source}-${idMap[e.target] || e.target}`,
      source: idMap[e.source] || e.source,
      target: idMap[e.target] || e.target,
      selected: false,
    }))
    setNodes(nds => [...nds, ...newNodes])
    setEdges(eds => [...eds, ...newEdges])
    setNextNodeId(nid => nid + newNodes.length)
    markDirty()
  }, [clipboardRef, nextNodeId, setNodes, setEdges, pushHistory, markDirty])

  const handleDuplicateSelected = useCallback(() => {
    const selectedNodes = nodes.filter(n => n.selected)
    if (selectedNodes.length === 0) return
    pushHistory()
    const idMap: Record<string, string> = {}
    const newNodes = selectedNodes.map(n => {
      const newId = String(nextNodeId + Object.keys(idMap).length)
      idMap[n.id] = newId
      return { ...n, id: newId, position: { x: n.position.x + 50, y: n.position.y + 50 }, selected: false, data: { ...n.data, dbId: 0 } }
    })
    const selectedNodeIds = new Set(selectedNodes.map(n => n.id))
    const selectedEdges = edges.filter(e => selectedNodeIds.has(e.source) && selectedNodeIds.has(e.target))
    const newEdges = selectedEdges.map(e => ({
      ...e,
      id: `e-${idMap[e.source] || e.source}-${idMap[e.target] || e.target}`,
      source: idMap[e.source] || e.source,
      target: idMap[e.target] || e.target,
      selected: false,
    }))
    setNodes(nds => [...nds, ...newNodes])
    setEdges(eds => [...eds, ...newEdges])
    setNextNodeId(nid => nid + newNodes.length)
    markDirty()
  }, [nodes, edges, nextNodeId, setNodes, setEdges, pushHistory, markDirty])

  /* ---- Context menu ---- */
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; type: 'node' | 'edge' | 'panel'; targetId?: string } | null>(null)

  const handleNodeContextMenu = useCallback((e: React.MouseEvent, node: Node) => {
    e.preventDefault()
    e.stopPropagation()
    setNodes(nds => nds.map(n => ({ ...n, selected: n.id === node.id })))
    setEdges(eds => eds.map(e => ({ ...e, selected: false })))
    setContextMenu({ x: e.clientX, y: e.clientY, type: 'node', targetId: node.id })
  }, [setNodes, setEdges])

  const handleEdgeContextMenu = useCallback((e: React.MouseEvent, edge: Edge) => {
    e.preventDefault()
    e.stopPropagation()
    setEdges(eds => eds.map(e => ({ ...e, selected: e.id === edge.id })))
    setContextMenu({ x: e.clientX, y: e.clientY, type: 'edge', targetId: edge.id })
  }, [setEdges])

  const handlePaneContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, type: 'panel' })
  }, [])

  const closeContextMenu = useCallback(() => setContextMenu(null), [])

  const contextMenuActions = useMemo(() => {
    if (!contextMenu) return []
    const actions: { label: string; shortcut?: string; action: () => void; danger?: boolean }[] = []
    if (contextMenu.type === 'node') {
      actions.push(
        { label: 'Copy', shortcut: 'Ctrl+C', action: () => { handleCopy(); closeContextMenu() } },
        { label: 'Duplicate', shortcut: 'Ctrl+D', action: () => { handleDuplicateSelected(); closeContextMenu() } },
        { label: 'Delete', shortcut: 'Del', action: () => { pushHistory(); handleDeleteSelectedNode(); closeContextMenu() }, danger: true },
      )
    } else if (contextMenu.type === 'edge') {
      actions.push(
        { label: 'Delete Edge', shortcut: 'Del', action: () => { pushHistory(); if (contextMenu.targetId) setEdges(eds => eds.filter(e => e.id !== contextMenu.targetId)); closeContextMenu() }, danger: true },
      )
    } else if (contextMenu.type === 'panel') {
      actions.push(
        { label: 'Add Block', action: () => { setBlockSelectorPos({ top: contextMenu.y, left: contextMenu.x }); setShowBlockSelector(true); closeContextMenu() } },
        { label: 'Paste', shortcut: 'Ctrl+V', action: () => { handlePaste(); closeContextMenu() } },
        { label: 'Auto Layout', shortcut: 'Ctrl+O', action: () => { pushHistory(); handleAutoLayout(); closeContextMenu() } },
        { label: 'Fit View', shortcut: 'Ctrl+1', action: () => { fitView({ padding: 0.2 }); closeContextMenu() } },
      )
    }
    return actions
  }, [contextMenu, handleCopy, handleDuplicateSelected, handleDeleteSelectedNode, handlePaste, handleAutoLayout, pushHistory, closeContextMenu, setEdges, fitView])

  /* ---- Keyboard shortcuts ---- */
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') return

      const mod = e.metaKey || e.ctrlKey

      if (e.key === 'v' || e.key === 'V') {
        e.preventDefault()
        setInteractionMode('pointer')
      } else if (e.key === 'h' || e.key === 'H') {
        e.preventDefault()
        setInteractionMode('hand')
      } else if (e.key === 'Delete' || e.key === 'Backspace') {
        e.preventDefault()
        pushHistory()
        handleDeleteSelectedNode()
      } else if (e.key === 'f' || e.key === 'F') {
        e.preventDefault()
        toggleFullscreen()
      } else if (mod && e.key === 'o') {
        e.preventDefault()
        pushHistory()
        handleAutoLayout()
      } else if (mod && e.key === 'z' && !e.shiftKey) {
        e.preventDefault()
        undo()
      } else if (mod && (e.key === 'y' || (e.key === 'z' && e.shiftKey))) {
        e.preventDefault()
        redo()
      } else if (mod && e.key === 'c') {
        e.preventDefault()
        handleCopy()
      } else if (mod && e.key === 'v') {
        e.preventDefault()
        handlePaste()
      } else if (mod && e.key === 'd') {
        e.preventDefault()
        handleDuplicateSelected()
      } else if (mod && e.key === '1') {
        e.preventDefault()
        fitView({ padding: 0.2 })
      } else if (mod && e.key === '=') {
        e.preventDefault()
        zoomIn()
      } else if (mod && e.key === '-') {
        e.preventDefault()
        zoomOut()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleDeleteSelectedNode, handleAutoLayout, toggleFullscreen, undo, redo, handleCopy, handlePaste, handleDuplicateSelected, pushHistory, fitView, zoomIn, zoomOut])

  /* ---- Block selector position ---- */
  const openBlockSelector = useCallback((e: React.MouseEvent) => {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    setBlockSelectorPos({ top: rect.top, left: rect.right + 8 })
    setShowBlockSelector(true)
  }, [])

  /* ---- Toolbar button style ---- */
  const toolBtn = (active = false): React.CSSProperties => ({
    width: 36, height: 36, borderRadius: 8, border: active ? '1px solid #155aef' : '1px solid #e2e8f0',
    background: active ? '#eff4ff' : '#fff', color: active ? '#155aef' : '#354052',
    cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 0,
  })

  const mainBtn = (primary = false): React.CSSProperties => ({
    padding: '7px 16px', borderRadius: 8, border: primary ? 'none' : '1px solid #d0d5dd',
    background: primary ? '#155aef' : '#fff', color: primary ? '#fff' : '#354052',
    fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 5,
  })

  if (loading) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
        <div style={{ color: '#676f83', fontSize: 14 }}>{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: '#f2f4f7' }}>
      {/* Top toolbar */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 10, padding: '8px 16px',
        background: '#fff', borderBottom: '0.5px solid rgba(16,24,40,0.08)', flexShrink: 0, zIndex: 30,
      }}>
        <a href="#/designer" style={{ display: 'flex', alignItems: 'center', color: '#676f83', textDecoration: 'none', fontSize: 13, gap: 4, flexShrink: 0 }}>
          <Icon d={ICONS.back} size={16} /> {t('flow.flowList')}
        </a>
        <div style={{ width: 1, height: 20, background: '#e2e8f0', flexShrink: 0 }} />
        <input
          value={flowName}
          onChange={e => { setFlowName(e.target.value); markDirty() }}
          style={{
            padding: '5px 10px', border: '1px solid #d0d5dd', borderRadius: 8,
            fontSize: 14, fontWeight: 600, outline: 'none', width: 200, flexShrink: 0,
          }}
          placeholder={t('flow.flowName')}
        />
        <input
          value={flowCategory}
          onChange={e => { setFlowCategory(e.target.value); markDirty() }}
          style={{
            padding: '5px 10px', border: '1px solid #d0d5dd', borderRadius: 8,
            fontSize: 12, outline: 'none', width: 120, flexShrink: 0,
          }}
          placeholder={t('common.category')}
        />
        {dirty && (
          <span style={{ fontSize: 11, color: '#f79009', flexShrink: 0 }}>*</span>
        )}
        {error && (
          <span style={{ fontSize: 12, color: '#f04438', flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {error}
          </span>
        )}
        <div style={{ flex: 1 }} />
        <button onClick={() => handleSave()} disabled={saving} style={mainBtn(true)}>
          <Icon d={ICONS.save} size={14} color="#fff" />
          {saving ? t('common.loading') : t('common.save')}
        </button>
        <button onClick={handleExport} style={mainBtn()}>
          <Icon d={ICONS.export} size={14} /> {t('common.export')}
        </button>
        <button onClick={handleImport} style={mainBtn()}>
          <Icon d={ICONS.import} size={14} /> {t('common.import')}
        </button>
        {dbFlowId && (
          <button onClick={handleDeleteFlow} style={{ ...mainBtn(), color: '#f04438', borderColor: '#f0443833' }}>
            <Icon d={ICONS.trash} size={14} color="#f04438" />
          </button>
        )}
      </div>

      {/* Canvas area (relative for absolute-positioned panels) */}
      <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        {/* Left operator bar */}
        <div style={{
          position: 'absolute', top: 12, left: 12, zIndex: 20,
          display: 'flex', flexDirection: 'column', gap: 6,
          background: '#fff', borderRadius: 12, padding: 6,
          border: '0.5px solid rgba(16,24,40,0.08)',
          boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
        }}>
          <button onClick={openBlockSelector} style={toolBtn()} title={t('flow.addNode')}>
            <Icon d={ICONS.plus} size={18} />
          </button>
          <button
            onClick={() => setInteractionMode('pointer')}
            style={toolBtn(interactionMode === 'pointer')}
            title={t('flow.pointerMode')}
          >
            <Icon d={ICONS.pointer} size={16} />
          </button>
          <button
            onClick={() => setInteractionMode('hand')}
            style={toolBtn(interactionMode === 'hand')}
            title={t('flow.handMode')}
          >
            <Icon d={ICONS.hand} size={22} />
          </button>
          <div style={{ height: 1, background: '#e2e8f0', margin: '2px 4px' }} />
          <button onClick={handleAutoLayout} style={toolBtn()} title={t('flow.autoLayout')}>
            <Icon d={ICONS.layout} size={16} />
          </button>
          <button onClick={toggleFullscreen} style={toolBtn(isFullscreen)} title={isFullscreen ? t('flow.exitFullscreen') : t('flow.fullscreen')}>
            <Icon d={ICONS.maximize} size={16} />
          </button>
        </div>

        {/* ReactFlow canvas */}
        <ReactFlow
          nodes={nodesWithCallback}
          edges={edgesWithCallback}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          connectionLineComponent={CustomConnectionLine}
          onEdgeMouseEnter={(_, edge) => {
            setEdges(eds => eds.map(e => e.id === edge.id ? { ...e, style: { ...e.style, stroke: '#155aef' } } : e))
          }}
          onEdgeMouseLeave={(_, edge) => {
            setEdges(eds => eds.map(e => e.id === edge.id ? { ...e, style: { ...e.style, stroke: '#8b8fa3' } } : e))
          }}
          nodesDraggable={interactionMode === 'pointer'}
          nodesConnectable={interactionMode === 'pointer'}
          elementsSelectable={true}
          selectionOnDrag={interactionMode === 'pointer'}
          selectionMode={SelectionMode.Partial}
          panOnDrag={interactionMode === 'hand'}
          onEdgeClick={(_, edge) => {
            setEdges(eds => eds.map(e => ({ ...e, selected: e.id === edge.id })))
            setNodes(nds => nds.map(n => ({ ...n, selected: false })))
          }}
          onNodeContextMenu={handleNodeContextMenu}
          onEdgeContextMenu={handleEdgeContextMenu}
          onPaneContextMenu={handlePaneContextMenu}
          fitView
          snapToGrid
          snapGrid={[16, 16]}
          defaultEdgeOptions={{
            type: 'default',
            animated: true,
            style: { stroke: '#8b8fa3', strokeWidth: 2 },
          }}
          style={{ width: '100%', height: '100%' }}
        >
          <Background gap={16} size={1} color="#e2e8f0" />
          <MiniMap
            style={{
              width: 103, height: 73,
              borderRadius: 8, border: '0.5px solid rgba(16,24,40,0.08)',
              background: '#f9fafb', boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
            }}
            maskColor="rgba(233,235,240,0.5)"
            nodeColor="#c8cdd5"
            nodeBorderRadius={4}
          />
          <Panel position="bottom-left" style={{ display: 'flex', gap: 4, background: '#fff', borderRadius: 10, padding: 4, border: '0.5px solid rgba(16,24,40,0.08)', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
            <ZoomControls />
          </Panel>
        </ReactFlow>

        {/* Block selector popup */}
        {showBlockSelector && (
          <>
            <div style={{ position: 'fixed', inset: 0, zIndex: 999 }} onClick={() => { setShowBlockSelector(false); setEdgeInsertInfo(null) }} />
            <BlockSelector
              anchor={blockSelectorPos}
              onPick={handleAddNode}
              onClose={() => { setShowBlockSelector(false); setEdgeInsertInfo(null) }}
            />
          </>
        )}

        {/* Context menu */}
        {contextMenu && (
          <>
            <div style={{ position: 'fixed', inset: 0, zIndex: 1000 }} onClick={closeContextMenu} onContextMenu={e => { e.preventDefault(); closeContextMenu() }} />
            <div style={{
              position: 'fixed',
              left: contextMenu.x,
              top: contextMenu.y,
              zIndex: 1001,
              background: '#fff',
              borderRadius: 10,
              padding: '4px 0',
              minWidth: 180,
              border: '0.5px solid rgba(16,24,40,0.08)',
              boxShadow: '0 4px 12px rgba(16,24,40,0.12)',
            }}>
              {contextMenuActions.map((action, i) => (
                <div
                  key={i}
                  onClick={action.action}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 14px',
                    cursor: 'pointer',
                    fontSize: 13,
                    color: action.danger ? '#f04438' : '#354052',
                    transition: 'background 0.1s',
                  }}
                  onMouseEnter={e => (e.currentTarget.style.background = action.danger ? '#fef3f2' : '#f5f8ff')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <span>{action.label}</span>
                  {action.shortcut && <span style={{ fontSize: 11, color: '#98a2b3', marginLeft: 20 }}>{action.shortcut}</span>}
                </div>
              ))}
            </div>
          </>
        )}

        {/* Property panel */}
        <PropertyPanel
          node={selectedNode}
          onChange={handleSelectedNodeChange}
          onDelete={handleDeleteSelectedNode}
          onClose={() => setSelectedNodeId(null)}
          llmModels={llmModels}
        />

      </div>
    </div>
  )
}

/* ============================================================
   Main Component: dispatches between list and editor
   ============================================================ */

export default function FlowDesigner() {
  const { id } = useParams<{ id: string }>()

  if (id) {
    return (
      <ReactFlowProvider>
        <FlowEditor flowId={id} />
      </ReactFlowProvider>
    )
  }

  return <FlowListView />
}
