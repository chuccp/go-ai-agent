import { defineStore } from 'pinia'
import { ref } from 'vue'

const API_BASE = ''

export interface SetupStatus {
  initialized: boolean
  dbConfigured: boolean
  hasAdmin: boolean
  hasBaseModel: boolean
}

export interface DbConfig {
  type: string
  path?: string
  host?: string
  port?: number
  username?: string
  password?: string
  database?: string
  charset?: string
  sslMode?: string
}

export interface ModelConfig {
  name: string
  provider: string
  model: string
  category?: string
  api_key?: string
  base_url?: string
  description?: string
}

export const DB_TYPES = [
  { value: 'sqlite', label: 'SQLite (本地文件)' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'postgresql', label: 'PostgreSQL' },
]

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

export const PROVIDERS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic Claude' },
  { value: 'gemini', label: 'Google Gemini' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'deepseek-openai', label: 'DeepSeek (OpenAI)' },
  { value: 'volcengine', label: '火山方舟' },
  { value: 'volcengine-claude', label: '火山方舟 (Claude)' },
  { value: 'qwen', label: '阿里云百炼' },
  { value: 'qwen-openai', label: '阿里云百炼 (OpenAI)' },
  { value: 'zhipu', label: '智谱 GLM' },
  { value: 'zhipu-openai', label: '智谱 GLM (OpenAI)' },
  { value: 'baidu', label: '百度千帆' },
  { value: 'baidu-openai', label: '百度千帆 (OpenAI)' },
  { value: 'xiaomi', label: '小米 MiMo' },
  { value: 'xiaomi-openai', label: '小米 MiMo (OpenAI)' },
  { value: 'qiniu', label: '七牛云 AI' },
  { value: 'qiniu-openai', label: '七牛云 AI (OpenAI)' },
  { value: 'bedrock', label: 'AWS Bedrock' },
  { value: 'vertex', label: 'Google Vertex AI' },
  { value: 'minimax', label: 'MiniMax' },
  { value: 'openai_compat', label: 'OpenAI 兼容（自定义）' },
  { value: 'claude_compat', label: 'Claude 兼容（自定义）' },
]

export interface ProviderDefault {
  model: string
  baseUrl: string
}

export const useSetupStore = defineStore('setup', () => {
  const status = ref<SetupStatus>({
    initialized: false,
    dbConfigured: false,
    hasAdmin: false,
    hasBaseModel: false,
  })
  const currentStep = ref(0)
  const saving = ref(false)
  const testing = ref(false)
  const error = ref('')
  const providerDefaults = ref<Record<string, Record<string, ProviderDefault>>>({})

  async function fetchStatus(): Promise<SetupStatus> {
    const res = await fetch(`${API_BASE}/api/setup/status`)
    if (!res.ok) {
      throw new Error('获取状态失败')
    }
    const data = await res.json()
    status.value = data.data as SetupStatus
    return status.value
  }

  async function initDatabase(db: DbConfig): Promise<void> {
    saving.value = true
    error.value = ''
    try {
      const res = await fetch(`${API_BASE}/api/setup/db`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(db),
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.msg || '数据库配置失败')
      }
      await fetchStatus()
    } catch (e: any) {
      error.value = e.message || '数据库配置失败'
      throw e
    } finally {
      saving.value = false
    }
  }

  async function testConnection(db: DbConfig): Promise<boolean> {
    testing.value = true
    error.value = ''
    try {
      const res = await fetch(`${API_BASE}/api/setup/db/test`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(db),
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.msg || '连接测试失败')
      }
      return true
    } catch (e: any) {
      error.value = e.message || '连接测试失败'
      throw e
    } finally {
      testing.value = false
    }
  }

  async function initAdmin(username: string, password: string): Promise<void> {
    saving.value = true
    error.value = ''
    try {
      const res = await fetch(`${API_BASE}/api/setup/admin`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.msg || '管理员配置失败')
      }
      await fetchStatus()
    } catch (e: any) {
      error.value = e.message || '管理员配置失败'
      throw e
    } finally {
      saving.value = false
    }
  }

  async function checkAdminExists(): Promise<{ hasAdmin: boolean; adminName: string }> {
    const res = await fetch(`${API_BASE}/api/setup/admin/exists`)
    const data = await res.json()
    return data.data as { hasAdmin: boolean; adminName: string }
  }

  async function initBaseModel(m: ModelConfig): Promise<void> {
    saving.value = true
    error.value = ''
    try {
      const res = await fetch(`${API_BASE}/api/setup/model`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(m),
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.msg || '模型配置失败')
      }
      await fetchStatus()
    } catch (e: any) {
      error.value = e.message || '模型配置失败'
      throw e
    } finally {
      saving.value = false
    }
  }

  async function completeSetup(): Promise<void> {
    saving.value = true
    error.value = ''
    try {
      const res = await fetch(`${API_BASE}/api/setup/complete`, {
        method: 'POST',
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.msg || '初始化完成失败')
      }
      await fetchStatus()
    } catch (e: any) {
      error.value = e.message || '初始化完成失败'
      throw e
    } finally {
      saving.value = false
    }
  }

  async function fetchProviderDefaults(): Promise<Record<string, Record<string, ProviderDefault>>> {
    const res = await fetch(`${API_BASE}/api/setup/providers`)
    if (!res.ok) return {}
    const data = await res.json()
    providerDefaults.value = (data.data || {}) as Record<string, Record<string, ProviderDefault>>
    return providerDefaults.value
  }

  return {
    status,
    currentStep,
    saving,
    testing,
    error,
    providerDefaults,
    fetchStatus,
    initDatabase,
    testConnection,
    initAdmin,
    checkAdminExists,
    initBaseModel,
    completeSetup,
    fetchProviderDefaults,
  }
})
