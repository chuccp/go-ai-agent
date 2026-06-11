import { API_BASE } from '@/constants'
import { defineStore } from 'pinia'
import { ref } from 'vue'


export interface AIModel {
  id: number
  name: string
  provider: string
  model: string
  category: string
  api_key: string
  base_url: string
  is_default: boolean
  is_base: boolean
  supports_multimodal: boolean
  thinking_level: string
  description: string
  created_at: string
  updated_at: string
}

export interface ProviderDefault {
  model: string
  baseUrl: string
}

export const API_TYPES = [
  { value: 'openai', label: 'OpenAI 兼容接口' },
  { value: 'claude', label: 'Claude 兼容接口' },
  { value: 'native', label: '厂家接口' },
]

export const PROVIDERS_BY_API: Record<string, { value: string; label: string }[]> = {
  openai: [
    { value: 'openai', label: 'OpenAI' },
    { value: 'deepseek-openai', label: 'DeepSeek' },
    { value: 'qwen-openai', label: '阿里云百炼' },
    { value: 'zhipu-openai', label: '智谱 GLM' },
    { value: 'baidu-openai', label: '百度千帆' },
    { value: 'xiaomi-openai', label: '小米 MiMo' },
    { value: 'qiniu-openai', label: '七牛云 AI' },
    { value: 'openai_compat', label: 'OpenAI 兼容（自定义）' },
  ],
  claude: [
    { value: 'anthropic', label: 'Anthropic' },
    { value: 'deepseek', label: 'DeepSeek' },
    { value: 'qwen', label: '阿里云百炼' },
    { value: 'zhipu', label: '智谱 GLM' },
    { value: 'volcengine-claude', label: '火山方舟' },
    { value: 'baidu', label: '百度千帆' },
    { value: 'xiaomi', label: '小米 MiMo' },
    { value: 'qiniu', label: '七牛云 AI' },
    { value: 'bedrock', label: 'AWS Bedrock' },
    { value: 'vertex', label: 'Google Vertex AI' },
    { value: 'claude_compat', label: 'Claude 兼容（自定义）' },
  ],
  native: [
    { value: 'gemini', label: 'Google Gemini' },
    { value: 'volcengine', label: '火山方舟' },
    { value: 'minimax', label: 'MiniMax' },
  ],
}

export const CATEGORIES = [
  { value: 'llm', label: 'LLM 大模型' },
  { value: 'image', label: '图片生成' },
  { value: 'voice', label: '语音合成' },
  { value: 'video', label: '视频生成' },
]

export const useModelStore = defineStore('model', () => {
  const models = ref<AIModel[]>([])
  const loading = ref(false)
  const providerDefaults = ref<Record<string, Record<string, ProviderDefault>>>({})

  async function fetchModels(category?: string) {
    loading.value = true
    try {
      const params = category ? `?category=${category}` : ''
      const res = await fetch(`${API_BASE}/api/ai-models${params}`)
      const data = await res.json()
      models.value = data.data || []
    } catch (e) { console.error('fetchModels failed', e) }
    finally { loading.value = false }
  }

  async function fetchProviderDefaults(): Promise<Record<string, Record<string, ProviderDefault>>> {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/providers`)
      if (!res.ok) return {}
      const data = await res.json()
      providerDefaults.value = (data.data || {}) as Record<string, Record<string, ProviderDefault>>
      return providerDefaults.value
    } catch { return {} }
  }

  async function createModel(m: Partial<AIModel>): Promise<AIModel | null> {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(m),
      })
      const data = await res.json()
      await fetchModels()
      return data.data || null
    } catch (e) { console.error('createModel failed', e); return null }
  }

  async function updateModel(id: number, m: Partial<AIModel>): Promise<AIModel | null> {
    try {
      const res = await fetch(`${API_BASE}/api/ai-models/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(m),
      })
      const data = await res.json()
      await fetchModels()
      return data.data || null
    } catch (e) { console.error('updateModel failed', e); return null }
  }

  async function deleteModel(id: number) {
    try {
      await fetch(`${API_BASE}/api/ai-models/${id}`, { method: 'DELETE' })
      models.value = models.value.filter(m => m.id !== id)
    } catch (e) { console.error('deleteModel failed', e) }
  }

  async function setDefault(id: number) {
    try {
      await fetch(`${API_BASE}/api/ai-models/${id}/default`, { method: 'PUT' })
      await fetchModels()
    } catch (e) { console.error('setDefault failed', e) }
  }

  async function setBase(id: number) {
    try {
      await fetch(`${API_BASE}/api/ai-models/${id}/base`, { method: 'PUT' })
      await fetchModels()
    } catch (e) { console.error('setBase failed', e) }
  }

  function getByCategory(category: string): AIModel[] {
    return models.value.filter(m => m.category === category)
  }

  function getDefault(category: string): AIModel | undefined {
    return models.value.find(m => m.category === category && m.is_default)
  }

  return { models, loading, providerDefaults, fetchModels, fetchProviderDefaults, createModel, updateModel, deleteModel, setDefault, setBase, getByCategory, getDefault }
})
