<template>
  <div class="node-config">
    <label>模型</label>
    <select v-model="cfg.model" @change="emitUpdate">
      <option v-for="m in models" :key="m.id" :value="m.id">{{ m.name }}</option>
    </select>

    <label>系统提示词</label>
    <textarea v-model="cfg.system" @input="emitUpdate" rows="2" placeholder="系统角色设定..."></textarea>

    <label>提示词模板</label>
    <textarea v-model="cfg.prompt" @input="emitUpdate" rows="5" placeholder="支持 {{节点名.output}} 引用上游..."></textarea>

    <label>Temperature <span class="val">{{ cfg.temperature }}</span></label>
    <input type="range" v-model.number="cfg.temperature" @input="emitUpdate" min="0" max="2" step="0.1" />

    <label>Top P <span class="val">{{ cfg.top_p }}</span></label>
    <input type="range" v-model.number="cfg.top_p" @input="emitUpdate" min="0" max="1" step="0.05" />

    <label>最大 Token</label>
    <input type="number" v-model.number="cfg.max_tokens" @input="emitUpdate" min="1" max="65536" />

    <label>思考等级</label>
    <select v-model="cfg.thinking_level" @change="emitUpdate">
      <option value="">跟随模型设置</option>
      <option value="off">关</option>
      <option value="low">低</option>
      <option value="medium">中</option>
      <option value="high">高</option>
      <option value="max">最高</option>
    </select>

    <label>输出格式</label>
    <select v-model="cfg.output_format_type" @change="onOutputFormatChange">
      <option value="">普通文本</option>
      <option value="json_auto">JSON 自动</option>
      <option value="json_object">JSON 对象</option>
      <option value="json_array">JSON 数组</option>
      <option value="custom">自定义 Schema</option>
    </select>
    <span v-if="cfg.output_format_type === 'json_auto'" class="hint" :class="{ detected: jsonAutoDetected }">
      {{ jsonAutoDetected ? '✅ 已检测到 JSON 输出要求' : '⚠ 请确认提示词中包含 JSON 输出样例说明' }}
    </span>

    <template v-if="cfg.output_format_type === 'json_object'">
      <label>字段定义</label>
      <div class="json-fields">
        <div v-for="(f, i) in jsonFields" :key="i" class="json-field-row">
          <input v-model="f.key" placeholder="字段名" @input="emitUpdate" class="field-key" />
          <select v-model="f.type" @change="emitUpdate" class="field-type">
            <option value="string">string</option>
            <option value="number">number</option>
            <option value="boolean">boolean</option>
            <option value="array">array</option>
            <option value="object">object</option>
          </select>
          <input v-model="f.desc" placeholder="描述" @input="emitUpdate" class="field-desc" />
          <button class="field-del" @click="removeJsonField(i)">×</button>
        </div>
        <button class="field-add" @click="addJsonField">+ 添加字段</button>
      </div>
      <label>示例（可选）</label>
      <textarea v-model="cfg.output_format_example" @input="emitUpdate" rows="2" placeholder='{"k1":"v1","k2":2}'></textarea>
    </template>

    <template v-if="cfg.output_format_type === 'json_array'">
      <label>元素类型</label>
      <select v-model="cfg.array_item_type" @change="emitUpdate">
        <option value="string">string</option>
        <option value="number">number</option>
        <option value="boolean">boolean</option>
        <option value="object">object</option>
      </select>
      <label>示例（可选）</label>
      <textarea v-model="cfg.output_format_example" @input="emitUpdate" rows="2" placeholder='["a","b"]'></textarea>
    </template>

    <template v-if="cfg.output_format_type === 'custom'">
      <label>JSON Schema</label>
      <textarea v-model="cfg.output_format" @input="emitUpdate" rows="5" placeholder='{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}'></textarea>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted, computed } from 'vue'
import { API_BASE } from '@/constants'

interface JsonField { key: string; type: string; desc: string }
interface Cfg {
  model: string; system: string; prompt: string
  temperature: number; top_p: number; max_tokens: number
  thinking_level: string
  output_format_type: string; output_format: string; output_format_example: string
  array_item_type: string
}

const models = ref<{ id: string; name: string }[]>([])
const jsonFields = ref<JsonField[]>([])

// Detect JSON keywords in prompt for json_auto mode
const jsonAutoDetected = computed(() => {
  const text = (cfg.prompt + ' ' + cfg.system).toLowerCase()
  return /json|返回json|输出json|json格式|json schema|json array|json object/i.test(text)
})

const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive<Cfg>({
  model: '', system: '', prompt: '',
  temperature: 0.7, top_p: 0.9, max_tokens: 4096,
  thinking_level: '',
  output_format_type: '', output_format: '', output_format_example: '',
  array_item_type: 'string',
})

watch(() => props.config, (c) => {
  cfg.model = c.model || ''
  cfg.system = c.system || ''
  cfg.prompt = c.prompt || ''
  cfg.temperature = c.temperature ?? 0.7
  cfg.top_p = c.top_p ?? 0.9
  cfg.max_tokens = c.max_tokens || 4096
  cfg.thinking_level = c.thinking_level || ''
  const fmt = parseOutputFormat(c.output_format || '')
  cfg.output_format_type = fmt.type
  cfg.output_format = fmt.raw
  cfg.output_format_example = fmt.example
  cfg.array_item_type = fmt.array_type
  if (fmt.type === 'json_object') {
    try { jsonFields.value = schemaToFields(JSON.parse(fmt.raw).json_schema || {}) } catch { jsonFields.value = [] }
  } else {
    jsonFields.value = []
  }
}, { immediate: true })

function parseOutputFormat(fmt: string) {
  const def = { type: '', raw: '', example: '', array_type: 'string' }
  if (!fmt) return def
  try {
    const obj = JSON.parse(fmt)
    if (obj.type === 'json' && obj.json_schema?.type === 'array')
      return { ...def, type: 'json_array', raw: fmt, example: obj.example || '', array_type: obj.json_schema?.items?.type || 'string' }
    if (obj.type === 'json' && obj.json_schema?.type === 'object')
      return { ...def, type: 'json_object', raw: fmt, example: obj.example || '' }
    return { ...def, type: 'custom', raw: fmt }
  } catch { return { ...def, type: 'custom', raw: fmt } }
}

function schemaToFields(schema: Record<string, any>): JsonField[] {
  const props = schema?.properties || {}
  return Object.entries(props).map(([key, val]: [string, any]) => ({
    key, type: val?.type || 'string', desc: val?.description || '',
  }))
}

function buildOutputFormat(): string {
  switch (cfg.output_format_type) {
    case 'json_auto':
      if (jsonAutoDetected.value) {
        return JSON.stringify({ type: 'json', json_schema: { type: 'object' } })
      }
      return ''
    case 'json_array':
      return JSON.stringify({ type: 'json', json_schema: { type: 'array', items: { type: cfg.array_item_type || 'string' } }, example: cfg.output_format_example || undefined })
    case 'json_object': {
      const props: Record<string, any> = {}
      const required: string[] = []
      for (const f of jsonFields.value) { if (!f.key) continue; props[f.key] = { type: f.type, ...(f.desc ? { description: f.desc } : {}) }; required.push(f.key) }
      return JSON.stringify({ type: 'json', json_schema: { type: 'object', properties: props, required }, example: cfg.output_format_example || undefined })
    }
    case 'custom': return cfg.output_format
    default: return ''
  }
}

function emitUpdate() {
  const { output_format_type, output_format_example, array_item_type, output_format, ...rest } = cfg
  const out: Record<string, any> = { ...rest }
  if (!out.thinking_level) delete out.thinking_level
  out.output_format = buildOutputFormat()
  emit('update', out)
}

function addJsonField() { jsonFields.value.push({ key: '', type: 'string', desc: '' }) }
function removeJsonField(i: number) { jsonFields.value.splice(i, 1); emitUpdate() }
function onOutputFormatChange() {
  if (cfg.output_format_type === 'json_object' && jsonFields.value.length === 0)
    jsonFields.value = [{ key: '', type: 'string', desc: '' }]
  emitUpdate()
}

onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE}/api/models?category=llm`)
    const data = await res.json()
    models.value = data.data?.models || []
  } catch {}
})
</script>
