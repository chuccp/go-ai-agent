<template>
  <div class="property-panel">
    <h3>属性</h3>
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
        <option value="for_each">ForEach 遍历</option>
        <option value="script">Script 脚本</option>
        <option value="iterator">按序迭代</option>
        <option value="loop">循环执行</option>
        <option value="image_gen">图片生成</option>
        <option value="audio_gen">音频生成</option>
        <option value="video_gen">视频生成</option>
      </select>

      <!-- LLM 节点 -->
      <template v-if="node.type === 'llm'">
        <label>模型</label>
        <select v-model="llmConfig.model" @change="onConfigChange">
          <option v-for="m in models" :key="m.id" :value="m.id">{{ m.name }}</option>
        </select>
        <label>系统提示词</label>
        <textarea v-model="llmConfig.system" @input="onConfigChange" rows="2" placeholder="系统角色设定..."></textarea>
        <label>提示词模板</label>
        <textarea v-model="llmConfig.prompt" @input="onConfigChange" rows="5" placeholder="支持 {{节点名.output}} 引用上游..."></textarea>
        <label>最大 Token</label>
        <input type="number" v-model.number="llmConfig.max_tokens" @input="onConfigChange" />
        <label>输出格式</label>
        <select v-model="llmConfig.output_format_type" @change="onConfigChange">
          <option value="">普通文本</option>
          <option value="json_array">JSON 数组</option>
          <option value="json_object">JSON 对象</option>
          <option value="custom">自定义 Schema</option>
        </select>
        <template v-if="llmConfig.output_format_type === 'custom'">
          <label>Schema JSON</label>
          <textarea v-model="llmConfig.output_format" @input="onConfigChange" rows="4" placeholder='{"type":"object","properties":{...}}'></textarea>
        </template>
        <template v-if="llmConfig.output_format_type === 'json_array' || llmConfig.output_format_type === 'json_object'">
          <label>格式示例（可选）</label>
          <textarea v-model="llmConfig.output_format_example" @input="onConfigChange" rows="2" :placeholder="formatPlaceholder"></textarea>
        </template>
      </template>

      <!-- ImageGen 节点 -->
      <template v-if="node.type === 'image_gen'">
        <label>模型</label>
        <select v-model="imgConfig.model" @change="onConfigChange">
          <option v-for="m in imageModels" :key="m.id" :value="m.id">{{ m.name }}</option>
        </select>
        <label>提示词</label>
        <textarea v-model="imgConfig.prompt" @input="onConfigChange" rows="3" placeholder="图片描述，支持 {{节点名.output}}..."></textarea>
        <label>生成数量</label>
        <input type="number" v-model.number="imgConfig.max_number" @input="onConfigChange" min="1" max="10" />
        <label>比例</label>
        <select v-model="imgConfig.scale" @change="onConfigChange">
          <option value="">默认</option>
          <option value="1:1">1:1</option>
          <option value="16:9">16:9</option>
          <option value="9:16">9:16</option>
          <option value="4:3">4:3</option>
          <option value="3:4">3:4</option>
        </select>
      </template>

      <!-- AudioGen 节点 -->
      <template v-if="node.type === 'audio_gen'">
        <label>模型</label>
        <select v-model="audConfig.model" @change="onConfigChange">
          <option v-for="m in audioModels" :key="m.id" :value="m.id">{{ m.name }}</option>
        </select>
        <label>文本</label>
        <textarea v-model="audConfig.text" @input="onConfigChange" rows="3" placeholder="要合成的文本，支持 {{节点名.output}}..."></textarea>
        <label>音色</label>
        <input v-model="audConfig.voice" @input="onConfigChange" placeholder="音色名称" />
      </template>

      <!-- VideoGen 节点 -->
      <template v-if="node.type === 'video_gen'">
        <label>模型</label>
        <select v-model="vidConfig.model" @change="onConfigChange">
          <option v-for="m in videoModels" :key="m.id" :value="m.id">{{ m.name }}</option>
        </select>
        <label>提示词</label>
        <textarea v-model="vidConfig.prompt" @input="onConfigChange" rows="3" placeholder="视频描述，支持 {{节点名.output}}..."></textarea>
        <label>时长（秒）</label>
        <input type="number" v-model.number="vidConfig.duration" @input="onConfigChange" min="1" max="60" />
      </template>

      <!-- Split 节点 -->
      <template v-if="node.type === 'split'">
        <label>数据来源</label>
        <select v-model="spConfig.source_key" @change="onConfigChange">
          <option value="">-- 选择节点 --</option>
          <optgroup v-for="(group, gIdx) in upstreamOptions" :key="gIdx" :label="group.node.label">
            <option v-for="opt in group.fields.filter(f => f.key.includes('output'))" :key="opt.key" :value="group.node.label">
              {{ group.node.label }}.output
            </option>
          </optgroup>
        </select>
        <label>拆分方式</label>
        <select v-model="spConfig.delimiter" @change="onConfigChange">
          <option value="paragraph">按段落（空行）</option>
          <option value="line">按行</option>
          <option value="，">按逗号</option>
          <option value="。">按句号</option>
        </select>
      </template>

      <!-- ForEach 节点 -->
      <template v-if="node.type === 'for_each'">
        <label>遍历数据来源</label>
        <select v-model="feConfig.items_key" @change="onConfigChange">
          <option value="">-- 选择节点 --</option>
          <optgroup v-for="(group, gIdx) in upstreamOptions" :key="gIdx" :label="group.node.label + ' (' + group.node.type + ')'">
            <option v-for="opt in group.fields" :key="opt.key" :value="opt.key">
              {{ opt.label }}
            </option>
          </optgroup>
        </select>
        <span class="hint">将上游输出的 JSON 数组逐项传递给下游节点处理</span>
      </template>

      <!-- Iterator 节点 -->
      <template v-if="node.type === 'iterator'">
        <label>数据来源</label>
        <select v-model="itConfig.items_key" @change="onConfigChange">
          <option value="">-- 选择节点 --</option>
          <optgroup v-for="(group, gIdx) in upstreamOptions" :key="gIdx" :label="group.node.label">
            <option v-for="opt in group.fields" :key="opt.key" :value="group.node.label">
              {{ group.node.label }}.output
            </option>
          </optgroup>
        </select>
        <label>每项提示词</label>
        <textarea v-model="itConfig.prompt" @input="onConfigChange" rows="4" placeholder="扩写以下内容：&#10;{{item.output}}"></textarea>
      </template>

      <!-- Loop 节点 -->
      <template v-if="node.type === 'loop'">
        <label>最大循环次数</label>
        <input type="number" v-model.number="lpConfig.max_iterations" @input="onConfigChange" min="1" max="100" />
        <label>中断条件字段</label>
        <input v-model="lpConfig.break_field" @input="onConfigChange" placeholder="如：检测.output" />
        <label>中断判断</label>
        <select v-model="lpConfig.break_operator" @change="onConfigChange">
          <option value="contains">包含</option>
          <option value="equals">等于</option>
          <option value="not_empty">不为空</option>
        </select>
        <template v-if="lpConfig.break_operator !== 'not_empty'">
          <label>中断比较值</label>
          <input v-model="lpConfig.break_value" @input="onConfigChange" placeholder="比较的目标值" />
        </template>
      </template>

      <!-- UserInput 节点 -->
      <template v-if="node.type === 'user_input'">
        <label>提示文本</label>
        <textarea v-model="uiConfig.prompt" @input="onConfigChange" rows="3" placeholder="提示用户的文本..."></textarea>
        <label>
          <input type="checkbox" v-model="uiConfig.confirm_only" @change="onConfigChange" />
          仅确认（无需输入文本）
        </label>
      </template>

      <button class="delete-btn" @click="emit('delete:node', node!.id)">删除节点</button>
    </div>
    <div v-else class="empty">选择节点编辑属性</div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import type { FlowNode, FlowEdge } from '@/types/flow'

const API_BASE = ''

interface LLMConfig { model: string; system: string; prompt: string; max_tokens: number; output_format_type: string; output_format: string; output_format_example: string }
interface FEConfig { items_key: string }
interface SPConfig { source_key: string; delimiter: string }
interface ITConfig { items_key: string; prompt: string }
interface LPConfig { max_iterations: number; break_field: string; break_operator: string; break_value: string }
interface UIConfig { prompt: string; confirm_only: boolean }
interface ImgConfig { model: string; prompt: string; max_number: number; scale: string }
interface AudConfig { model: string; text: string; voice: string }
interface VidConfig { model: string; prompt: string; duration: number }

const props = defineProps<{ node: FlowNode | null; nodes?: FlowNode[]; edges?: FlowEdge[] }>()
const emit = defineEmits<{ 'update:node': [node: FlowNode]; 'delete:node': [id: number] }>()

const models = ref<{ id: string; name: string }[]>([])
const imageModels = ref<{ id: string; name: string }[]>([])
const audioModels = ref<{ id: string; name: string }[]>([])
const videoModels = ref<{ id: string; name: string }[]>([])

const llmConfig = ref<LLMConfig>({ model: '', system: '', prompt: '', max_tokens: 4096, output_format_type: '', output_format: '', output_format_example: '' })
const spConfig = ref<SPConfig>({ source_key: '', delimiter: 'paragraph' })
const feConfig = ref<FEConfig>({ items_key: '' })
const itConfig = ref<ITConfig>({ items_key: '', prompt: '' })
const lpConfig = ref<LPConfig>({ max_iterations: 10, break_field: '', break_operator: 'contains', break_value: '' })
const uiConfig = ref<UIConfig>({ prompt: '', confirm_only: false })
const imgConfig = ref<ImgConfig>({ model: '', prompt: '', max_number: 1, scale: '' })
const audConfig = ref<AudConfig>({ model: '', text: '', voice: '' })
const vidConfig = ref<VidConfig>({ model: '', prompt: '', duration: 5 })

const formatPlaceholder = computed(() => {
  switch (llmConfig.value.output_format_type) {
    case 'json_array': return '["段落1", "段落2", ...]'
    case 'json_object': return '{"title": "...", "content": "..."}'
    default: return ''
  }
})

const upstreamOptions = computed(() => {
  if (!props.node || !props.nodes || !props.edges || (props.node.type !== 'for_each' && props.node.type !== 'split' && props.node.type !== 'iterator' && props.node.type !== 'loop')) return []
  const incoming = props.edges.filter(e => e.target_node_id === props.node!.id)
  const sources = incoming.length > 0
    ? incoming.map(e => props.nodes!.find(n => n.id === e.source_node_id)).filter(Boolean) as FlowNode[]
    : props.nodes.filter(n => n.id !== props.node!.id && n.type !== 'start' && n.type !== 'end')
  return sources.map(n => ({
    node: n,
    fields: [
      { key: n.label, label: `${n.label}.output` },
      ...(n.type === 'llm' ? [{ key: n.label + '.prompt', label: `${n.label}.prompt` }] : []),
    ],
  }))
})

async function loadModels() {
  try {
    const [llmRes, imgRes, audRes, vidRes] = await Promise.all([
      fetch(`${API_BASE}/api/models?category=llm`),
      fetch(`${API_BASE}/api/models?category=image`),
      fetch(`${API_BASE}/api/models?category=voice`),
      fetch(`${API_BASE}/api/models?category=video`),
    ])
    const llmData = await llmRes.json()
    models.value = llmData.data?.models || []
    const imgData = await imgRes.json()
    imageModels.value = imgData.data?.models || []
    const audData = await audRes.json()
    audioModels.value = audData.data?.models || []
    const vidData = await vidRes.json()
    videoModels.value = vidData.data?.models || []
  } catch {}
}

onMounted(loadModels)

function buildOutputFormat(cfg: LLMConfig): string {
  switch (cfg.output_format_type) {
    case 'json_array':
      return JSON.stringify({ type: 'json', json_schema: { type: 'array' }, example: cfg.output_format_example || undefined })
    case 'json_object':
      return JSON.stringify({ type: 'json', json_schema: { type: 'object' }, example: cfg.output_format_example || undefined })
    case 'custom':
      return cfg.output_format
    default:
      return ''
  }
}

function parseOutputFormat(fmt: string): { output_format_type: string; output_format: string; output_format_example: string } {
  if (!fmt) return { output_format_type: '', output_format: '', output_format_example: '' }
  try {
    const obj = JSON.parse(fmt)
    if (obj.type === 'json' && obj.json_schema?.type === 'array') {
      return { output_format_type: 'json_array', output_format: fmt, output_format_example: obj.example || '' }
    }
    if (obj.type === 'json' && obj.json_schema?.type === 'object') {
      return { output_format_type: 'json_object', output_format: fmt, output_format_example: obj.example || '' }
    }
    return { output_format_type: 'custom', output_format: fmt, output_format_example: '' }
  } catch {
    return { output_format_type: 'custom', output_format: fmt, output_format_example: '' }
  }
}

function safeParseConfig(raw: any): Record<string, any> {
  if (typeof raw !== 'string') return raw || {}
  try { return JSON.parse(raw || '{}') } catch {}
  // Config may contain raw control characters (e.g. newlines in prompt)
  const sanitized = raw.replace(/[\x00-\x1F\x7F]/g, c =>
    ({ '\n': '\\n', '\r': '\\r', '\t': '\\t', '\b': '\\b', '\f': '\\f' } as Record<string, string>)[c] || '\\u' + ('000' + c.charCodeAt(0).toString(16)).slice(-4)
  )
  try { return JSON.parse(sanitized || '{}') } catch { return {} }
}

watch(() => props.node, (n) => {
  if (!n) return
  const cfg = safeParseConfig(n.config)
  if (n.type === 'llm') {
    const fmt = parseOutputFormat(cfg.output_format || '')
    llmConfig.value = { model: cfg.model || '', system: cfg.system || '', prompt: cfg.prompt || '', max_tokens: cfg.max_tokens || 4096, ...fmt }
  }
  if (n.type === 'split') {
    spConfig.value = { source_key: cfg.source_key || '', delimiter: cfg.delimiter || 'paragraph' }
  }
  if (n.type === 'for_each') {
    feConfig.value = { items_key: cfg.items_key || '' }
  }
  if (n.type === 'iterator') {
    itConfig.value = { items_key: cfg.items_key || '', prompt: cfg.prompt || '' }
  }
  if (n.type === 'loop') {
    lpConfig.value = { max_iterations: cfg.max_iterations || 10, break_field: cfg.break_field || '', break_operator: cfg.break_operator || 'contains', break_value: cfg.break_value || '' }
  }
  if (n.type === 'user_input') {
    uiConfig.value = { prompt: cfg.prompt || '', confirm_only: cfg.confirm_only || false }
  }
  if (n.type === 'image_gen') {
    imgConfig.value = { model: cfg.model || '', prompt: cfg.prompt || '', max_number: cfg.max_number || 1, scale: cfg.scale || '' }
  }
  if (n.type === 'audio_gen') {
    audConfig.value = { model: cfg.model || '', text: cfg.text || '', voice: cfg.voice || '' }
  }
  if (n.type === 'video_gen') {
    vidConfig.value = { model: cfg.model || '', prompt: cfg.prompt || '', duration: cfg.duration || 5 }
  }
}, { immediate: true })

function onConfigChange() {
  if (!props.node) return
  const updated = { ...props.node }
  switch (props.node.type) {
    case 'llm': {
      const { output_format_type, output_format_example, ...rest } = llmConfig.value
      updated.config = JSON.stringify({ ...rest, output_format: buildOutputFormat(llmConfig.value) })
      break
    }
    case 'split': updated.config = JSON.stringify(spConfig.value); break
    case 'for_each': updated.config = JSON.stringify(feConfig.value); break
    case 'iterator': updated.config = JSON.stringify(itConfig.value); break
    case 'loop': updated.config = JSON.stringify(lpConfig.value); break
    case 'user_input': updated.config = JSON.stringify(uiConfig.value); break
    case 'image_gen': updated.config = JSON.stringify(imgConfig.value); break
    case 'audio_gen': updated.config = JSON.stringify(audConfig.value); break
    case 'video_gen': updated.config = JSON.stringify(vidConfig.value); break
  }
  emit('update:node', updated)
}

function emitUpdate() { if (props.node) emit('update:node', { ...props.node }) }
</script>

<style scoped>
.property-panel { width: 280px; background: #fff; border-left: 1px solid #e0e0e0; padding: 12px; overflow-y: auto; flex-shrink: 0; }
.property-panel h3 { font-size: 14px; margin-bottom: 12px; color: #666; }
.props { display: flex; flex-direction: column; gap: 8px; }
.props label { font-size: 12px; color: #888; font-weight: 500; }
.props input, .props select, .props textarea { padding: 6px 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 12px; font-family: inherit; }
.props textarea { resize: vertical; min-height: 50px; }
.hint { font-size: 11px; color: #aaa; margin-top: -4px; }
.delete-btn { margin-top: 12px; background: #ff4d4f; color: #fff; border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; }
.delete-btn:hover { background: #e04040; }
.empty { font-size: 13px; color: #bbb; text-align: center; margin-top: 20px; }
</style>
