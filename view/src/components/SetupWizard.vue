<template>
  <div class="setup-container">
    <div class="setup-card">
      <div class="card-header">
        <div class="logo">⚡</div>
        <h1>go-ai-agent</h1>
        <p class="subtitle">首次运行，请完成以下配置以开始使用</p>
      </div>

      <!-- Success state -->
      <div v-if="done" class="success-state">
        <div class="success-icon">✓</div>
        <h2>设置完成！</h2>
        <p>系统已成功初始化，即将跳转...</p>
      </div>

      <!-- Step indicator -->
      <div v-if="!done" class="steps-indicator">
        <template v-for="(step, i) in steps" :key="i">
          <div :class="['step', { active: currentStep === i, done: currentStep > i }]">
            <span class="step-num">{{ currentStep > i ? '✓' : i + 1 }}</span>
            <span class="step-label">{{ step }}</span>
          </div>
          <div v-if="i < steps.length - 1" :class="['step-line', { filled: currentStep > i }]" />
        </template>
      </div>

      <div class="step-transition-wrapper">
        <!-- Step 1: Database -->
        <transition name="fade" mode="out-in">
        <div v-if="!done && currentStep === 0" key="step1" class="step-content">
          <div class="form-row">
            <label class="form-label">数据库类型 <span class="required">*</span></label>
            <select v-model="dbForm.type" class="form-select">
              <option v-for="dt in DB_TYPES" :key="dt.value" :value="dt.value">{{ dt.label }}</option>
            </select>
          </div>

          <hr class="section-divider" />

          <!-- SQLite fields -->
          <div v-if="dbForm.type === 'sqlite'" class="form-row">
            <label class="form-label">文件路径</label>
            <input v-model="dbForm.path" class="form-input" placeholder="./data/go-ai-agent.db" />
          </div>

          <!-- MySQL / PostgreSQL fields -->
          <div v-if="dbForm.type !== 'sqlite'" class="form-grid">
            <div class="form-row">
              <label class="form-label">主机地址 <span class="required">*</span></label>
              <input v-model="dbForm.host" class="form-input" placeholder="localhost" />
            </div>

            <div class="form-row">
              <label class="form-label">端口</label>
              <input v-model.number="dbForm.port" type="number" class="form-input" :placeholder="dbForm.type === 'mysql' ? '3306' : '5432'" />
            </div>

            <div class="form-row">
              <label class="form-label">数据库名 <span class="required">*</span></label>
              <input v-model="dbForm.database" class="form-input" placeholder="go_ai_agent" />
            </div>

            <div class="form-row">
              <label class="form-label">用户名 <span class="required">*</span></label>
              <input v-model="dbForm.username" class="form-input" placeholder="root" />
            </div>

            <div class="form-row">
              <label class="form-label">密码</label>
              <div class="pwd-wrap">
                <input v-model="dbForm.password" :type="showDbPassword ? 'text' : 'password'" class="form-input" placeholder="密码" />
                <button type="button" class="pwd-toggle" @click="showDbPassword = !showDbPassword">{{ showDbPassword ? '🙈' : '👁' }}</button>
              </div>
            </div>

            <div v-if="dbForm.type === 'mysql'" class="form-row">
              <label class="form-label">字符集</label>
              <input v-model="dbForm.charset" class="form-input" placeholder="utf8mb4" />
            </div>

            <div v-if="dbForm.type === 'postgresql'" class="form-row">
              <label class="form-label">SSL 模式</label>
              <select v-model="dbForm.sslMode" class="form-select">
                <option value="disable">disable</option>
                <option value="require">require</option>
                <option value="verify-ca">verify-ca</option>
                <option value="verify-full">verify-full</option>
              </select>
            </div>
          </div>

          <div v-if="error" class="error-msg">{{ error }}</div>

          <div class="step-footer">
            <button v-if="dbForm.type !== 'sqlite'" class="btn btn-outline" @click="onTestConnection" :disabled="testing">
              {{ testing ? '测试中...' : '测试连接' }}
            </button>
            <button class="btn btn-primary" @click="onNextDb" :disabled="saving">
              {{ saving ? '保存中...' : '下一步 →' }}
            </button>
          </div>
        </div>
      </transition>

      <!-- Step 2: Admin Account -->
      <transition name="fade" mode="out-in">
        <div v-if="!done && currentStep === 1" key="step2" class="step-content">
          <h3>创建管理员账号</h3>
          <p class="step-desc">设置管理员账号以管理系统</p>

          <div v-if="adminExists" class="info-banner">
            已存在管理员账号，输入用户名和密码将重置管理员密码。
          </div>

          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">用户名 <span class="required">*</span></label>
              <input v-model="adminForm.username" class="form-input" placeholder="admin" />
            </div>

            <div class="form-row">
              <label class="form-label">密码 <span class="required">*</span></label>
              <div class="pwd-wrap">
                <input v-model="adminForm.password" :type="showAdminPassword ? 'text' : 'password'" class="form-input" placeholder="至少6位" />
                <button type="button" class="pwd-toggle" @click="showAdminPassword = !showAdminPassword">{{ showAdminPassword ? '🙈' : '👁' }}</button>
              </div>
            </div>

            <div class="form-row">
              <label class="form-label">确认密码 <span class="required">*</span></label>
              <div class="pwd-wrap">
                <input v-model="adminForm.confirmPassword" :type="showAdminConfirm ? 'text' : 'password'" class="form-input" placeholder="再次输入密码" />
                <button type="button" class="pwd-toggle" @click="showAdminConfirm = !showAdminConfirm">{{ showAdminConfirm ? '🙈' : '👁' }}</button>
              </div>
            </div>
          </div>

          <div v-if="error" class="error-msg">{{ error }}</div>

          <div class="step-footer">
            <button class="btn btn-ghost" @click="currentStep = 0">← 上一步</button>
            <button v-if="adminExists" class="btn btn-ghost" @click="currentStep = 2">跳过</button>
            <button class="btn btn-primary" @click="onNextAdmin" :disabled="saving">
              {{ saving ? '保存中...' : '下一步 →' }}
            </button>
          </div>
        </div>
      </transition>

      <!-- Step 3: Base Model -->
      <transition name="fade" mode="out-in">
        <div v-if="!done && currentStep === 2" key="step3" class="step-content step-compact">
          <h3>配置基础大模型</h3>

          <h4>接口类型</h4>
          <div class="api-type-group">
            <label
              v-for="t in API_TYPES"
              :key="t.value"
              :class="['api-type-card', { selected: apiType === t.value }]"
            >
              <div class="api-type-icon">{{ t.value === 'openai' ? '🔷' : t.value === 'claude' ? '🟠' : '🟢' }}</div>
              <div class="api-type-text">
                <span class="api-type-name">{{ t.label }}</span>
              </div>
              <input type="radio" v-model="apiType" :value="t.value" />
            </label>
          </div>

          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">提供商 <span class="required">*</span></label>
              <select v-model="modelForm.provider" class="form-select">
                <option v-for="p in availableProviders" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
            </div>

            <div class="form-row">
              <label class="form-label">显示名称</label>
              <input v-model="modelForm.name" class="form-input" placeholder="留空将自动生成" />
            </div>

            <div class="form-row">
              <label class="form-label">模型标识 <span class="required">*</span></label>
              <input v-model="modelForm.model" class="form-input" placeholder="例: gpt-4o, claude-sonnet-4-20250514" />
            </div>

            <div class="form-row">
              <label class="form-label">API Key <span class="required">*</span></label>
              <div class="pwd-wrap">
                <input v-model="modelForm.api_key" :type="showModelKey ? 'text' : 'password'" class="form-input" placeholder="sk-..." />
                <button type="button" class="pwd-toggle" @click="showModelKey = !showModelKey">{{ showModelKey ? '🙈' : '👁' }}</button>
              </div>
            </div>

            <div class="form-row">
              <label class="form-label">Base URL <span class="required">*</span></label>
              <input v-model="modelForm.base_url" class="form-input" placeholder="例: https://api.openai.com/v1" />
            </div>

          </div>

          <div v-if="error" class="error-msg">{{ error }}</div>

          <div class="step-footer">
            <button class="btn btn-ghost" @click="currentStep = 1">← 上一步</button>
            <button class="btn btn-success" @click="onComplete" :disabled="saving">
              {{ saving ? '初始化中...' : '✓ 完成设置' }}
            </button>
          </div>
        </div>
      </transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSetupStore, DB_TYPES, API_TYPES, PROVIDERS_BY_API } from '../stores/setup'

const router = useRouter()
const store = useSetupStore()

const steps = ['数据库配置', '管理员账号', '基础模型']
const currentStep = ref(0)
const done = ref(false)
const saving = ref(false)
const testing = ref(false)
const error = ref('')
const adminExists = ref(false)
const showDbPassword = ref(false)
const showAdminPassword = ref(false)
const showAdminConfirm = ref(false)
const showModelKey = ref(false)

const dbForm = reactive({
  type: 'sqlite',
  path: './data/go-ai-agent.db',
  host: 'localhost',
  port: 3306,
  database: 'go_ai_agent',
  username: 'root',
  password: '',
  charset: 'utf8mb4',
  sslMode: 'disable',
})

const adminForm = reactive({
  username: '',
  password: '',
  confirmPassword: '',
})

const modelForm = reactive({
  name: '',
  provider: 'openai',
  model: '',
  category: 'llm',
  api_key: '',
  base_url: '',
  description: '',
})

const apiType = ref('openai')
const availableProviders = computed(() => PROVIDERS_BY_API[apiType.value] || [])

watch(apiType, () => {
  modelForm.provider = availableProviders.value[0]?.value || ''
  modelForm.name = `${modelForm.category}-${modelForm.provider}`
})

// Fetch defaults early and independently — don't let fetchStatus failure block it
store.fetchProviderDefaults().catch(() => {})

// Auto-fill model/base_url whenever provider changes (e.g. user picks from dropdown)
watch(() => modelForm.provider, (p) => {
  const def = store.providerDefaults[apiType.value]?.[p]
  if (def) {
    modelForm.model = def.model
    modelForm.base_url = def.baseUrl || ''
  }
  modelForm.name = `${modelForm.category}-${p}`
})

// Fetch (or re-fetch) defaults and auto-fill whenever entering step 3
watch(currentStep, async (step) => {
  if (step === 2) {
    await store.fetchProviderDefaults()
    const def = store.providerDefaults[apiType.value]?.[modelForm.provider]
    if (def) {
      modelForm.model = def.model
      modelForm.base_url = def.baseUrl || ''
    }
    modelForm.name = `${modelForm.category}-${modelForm.provider}`
  }
})

onMounted(async () => {
  try {
    await store.fetchStatus()
    if (store.status.initialized) {
      router.push('/')
      return
    }
    if (store.status.dbConfigured) {
      if (store.status.hasAdmin) {
        currentStep.value = 2
      } else {
        currentStep.value = 1
      }
    }
    if (store.status.dbConfigured) {
      const info = await store.checkAdminExists()
      adminExists.value = info.hasAdmin
    }
  } catch {
    // Server may still be starting, stay on step 0
  }
})

async function onTestConnection() {
  testing.value = true
  error.value = ''
  try {
    await store.testConnection({ ...dbForm })
  } catch (e: any) {
    error.value = e.message
  } finally {
    testing.value = false
  }
}

async function onNextDb() {
  saving.value = true
  error.value = ''
  try {
    await store.initDatabase({ ...dbForm })
    currentStep.value = 1
    const info = await store.checkAdminExists()
    adminExists.value = info.hasAdmin
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function onNextAdmin() {
  if (!adminForm.username || !adminForm.password) {
    error.value = '用户名和密码不能为空'
    return
  }
  if (adminForm.password.length < 6) {
    error.value = '密码长度不能少于6位'
    return
  }
  if (adminForm.password !== adminForm.confirmPassword) {
    error.value = '两次输入的密码不一致'
    return
  }
  saving.value = true
  error.value = ''
  try {
    await store.initAdmin(adminForm.username, adminForm.password)
    currentStep.value = 2
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function onComplete() {
  if (!modelForm.model) {
    error.value = '请填写模型标识'
    return
  }
  if (!modelForm.api_key) {
    error.value = '请填写 API Key'
    return
  }
  if (!modelForm.base_url) {
    error.value = '请填写 Base URL'
    return
  }
  saving.value = true
  error.value = ''
  try {
    await store.initBaseModel({ ...modelForm })
    await store.completeSetup()
    done.value = true
    setTimeout(() => {
      router.push('/')
    }, 2000)
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.setup-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #0f0c29 0%, #1a1a3e 30%, #24243e 60%, #1a1a3e 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

/* ---- Card ---- */
.setup-card {
  background: rgba(255, 255, 255, 0.97);
  border-radius: 20px;
  padding: 44px 48px;
  width: 600px;
  max-width: 100%;
  box-shadow:
    0 4px 6px rgba(0, 0, 0, 0.07),
    0 20px 60px rgba(0, 0, 0, 0.2),
    0 0 120px rgba(74, 158, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.5);
  backdrop-filter: blur(10px);
}

/* ---- Card Header ---- */
.card-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, #4a9eff 0%, #6c5ce7 100%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  margin-bottom: 12px;
  box-shadow: 0 8px 24px rgba(74, 158, 255, 0.3);
}

.setup-card h1 {
  font-size: 22px;
  font-weight: 700;
  color: #1a1a2e;
  margin-bottom: 4px;
  letter-spacing: -0.3px;
}

.subtitle {
  color: #999;
  font-size: 13px;
}

/* ---- Steps indicator ---- */
.steps-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  margin-bottom: 32px;
}

.step {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 24px;
  font-size: 13px;
  font-weight: 500;
  color: #bbb;
  background: #f7f7f8;
  transition: all 0.3s ease;
  white-space: nowrap;
}

.step.active {
  background: linear-gradient(135deg, #4a9eff 0%, #3d8af0 100%);
  color: #fff;
  box-shadow: 0 4px 12px rgba(74, 158, 255, 0.35);
}

.step.done {
  background: #eafaf1;
  color: #52c41a;
}

.step-num {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
  background: rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.step.active .step-num {
  background: rgba(255, 255, 255, 0.25);
}

.step.done .step-num {
  background: rgba(82, 196, 26, 0.12);
}

.step-line {
  width: 24px;
  height: 2px;
  background: #e8e8e8;
  border-radius: 1px;
  margin: 0 4px;
  transition: background 0.4s ease;
}

.step-line.filled {
  background: #52c41a;
}

/* ---- Step content ---- */
.step-content h3 {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a2e;
  margin-bottom: 4px;
}

.step-content h4 {
  font-size: 14px;
  font-weight: 600;
  color: #444;
  margin: 0 0 12px 0;
}

.step-desc {
  color: #999;
  font-size: 13px;
  margin-bottom: 20px;
}

/* ---- Compact variant (step 3) ---- */
.step-compact h3 {
  margin-bottom: 2px;
}

.step-compact .step-desc {
  margin-bottom: 12px;
}

.step-compact h4 {
  margin-bottom: 8px;
  font-size: 13px;
}

.step-compact .api-type-group {
  gap: 8px;
  margin-bottom: 14px;
}

.step-compact .api-type-card {
  padding: 10px 10px;
  gap: 4px;
}

.step-compact .api-type-icon {
  font-size: 18px;
}

.step-compact .api-type-name {
  font-size: 12px;
}

.step-compact .form-grid {
  gap: 10px;
}

.step-compact .form-row {
  gap: 3px;
}

.step-compact .form-label {
  font-size: 12px;
}

.step-compact .form-input,
.step-compact .form-select {
  padding: 7px 10px;
  font-size: 13px;
}

.step-compact .step-footer {
  margin-top: 18px;
  padding-top: 14px;
}

/* ---- Section divider ---- */
.section-divider {
  border: none;
  border-top: 1px solid #eef0f2;
  margin: 18px 0;
}

/* ---- API type cards ---- */
.api-type-group {
  display: flex;
  gap: 12px;
  margin-bottom: 22px;
}

.api-type-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 12px;
  border: 2px solid #eef0f2;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.25s ease;
  background: #fafbfc;
}

.api-type-card:hover {
  border-color: #c8d6e5;
  background: #f0f4f8;
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
}

.api-type-card.selected {
  border-color: #4a9eff;
  background: #f0f6ff;
  box-shadow: 0 4px 16px rgba(74, 158, 255, 0.15);
  transform: translateY(-2px);
}

.api-type-icon {
  font-size: 22px;
  line-height: 1;
}

.api-type-text {
  text-align: center;
}

.api-type-name {
  font-size: 13px;
  font-weight: 600;
  color: #444;
  transition: color 0.2s;
}

.api-type-card.selected .api-type-name {
  color: #4a9eff;
}

.api-type-card input {
  display: none;
}

/* ---- Form ---- */
.form-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.form-label {
  font-size: 13px;
  font-weight: 500;
  color: #555;
}

.form-input,
.form-select {
  padding: 10px 14px;
  border: 1.5px solid #e4e6ea;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  transition: all 0.2s ease;
  background: #f9fafb;
  color: #333;
  width: 100%;
  box-sizing: border-box;
  font-family: inherit;
}

.form-input::placeholder {
  color: #c0c4cc;
}

.form-input:hover,
.form-select:hover {
  border-color: #c8d6e5;
}

.form-input:focus,
.form-select:focus {
  border-color: #4a9eff;
  background: #fff;
  box-shadow: 0 0 0 3px rgba(74, 158, 255, 0.12);
}

.required {
  color: #ff4d4f;
}

/* ---- Password toggle ---- */
.pwd-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.pwd-wrap input {
  flex: 1;
  padding-right: 36px;
}

.pwd-toggle {
  position: absolute;
  right: 2px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  font-size: 16px;
  padding: 4px 8px;
  line-height: 1;
}

/* ---- Step footer ---- */
.step-footer {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 10px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #f3f4f6;
}

/* ---- Buttons ---- */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 10px 22px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  font-family: inherit;
}

.btn:disabled {
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #4a9eff 0%, #3d8af0 100%);
  color: #fff;
  box-shadow: 0 2px 8px rgba(74, 158, 255, 0.3);
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #3d8af0 0%, #2d7ae0 100%);
  box-shadow: 0 4px 14px rgba(74, 158, 255, 0.4);
  transform: translateY(-1px);
}

.btn-primary:disabled {
  background: linear-gradient(135deg, #b0d4ff 0%, #a0c8f8 100%);
  box-shadow: none;
}

.btn-outline {
  background: #fff;
  border: 1.5px solid #4a9eff;
  color: #4a9eff;
  margin-right: auto;
}

.btn-outline:hover:not(:disabled) {
  background: #f0f6ff;
}

.btn-outline:disabled {
  border-color: #b0d4ff;
  color: #b0d4ff;
}

.btn-ghost {
  background: transparent;
  border: 1.5px solid transparent;
  color: #888;
}

.btn-ghost:hover {
  color: #555;
  background: #f5f6f8;
}

.btn-success {
  background: linear-gradient(135deg, #52c41a 0%, #45b012 100%);
  color: #fff;
  box-shadow: 0 2px 8px rgba(82, 196, 26, 0.3);
}

.btn-success:hover:not(:disabled) {
  background: linear-gradient(135deg, #45b012 0%, #389e0d 100%);
  box-shadow: 0 4px 14px rgba(82, 196, 26, 0.4);
  transform: translateY(-1px);
}

.btn-success:disabled {
  background: linear-gradient(135deg, #b7eb8f 0%, #a0d880 100%);
  box-shadow: none;
}

/* ---- Error & info ---- */
.error-msg {
  background: #fff2f0;
  border: 1px solid #ffccc7;
  border-radius: 8px;
  padding: 10px 14px;
  color: #ff4d4f;
  font-size: 13px;
  margin-top: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.info-banner {
  background: #f0f6ff;
  border: 1px solid #b3d8ff;
  border-radius: 8px;
  padding: 10px 14px;
  color: #4a9eff;
  font-size: 13px;
  margin-bottom: 18px;
  display: flex;
  align-items: center;
  gap: 6px;
}

/* ---- Success state ---- */
.success-state {
  text-align: center;
  padding: 48px 0 32px;
}

.success-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: linear-gradient(135deg, #52c41a 0%, #45b012 100%);
  color: #fff;
  font-size: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
  box-shadow: 0 8px 28px rgba(82, 196, 26, 0.35);
  animation: pop-in 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.success-state h2 {
  font-size: 22px;
  font-weight: 600;
  color: #1a1a2e;
  margin-bottom: 8px;
}

.success-state p {
  color: #999;
  font-size: 14px;
}

/* ---- Step transition ---- */
.step-transition-wrapper {
  position: relative;
  overflow: hidden;
}

.fade-enter-active,
.fade-leave-active {
  transition: all 0.25s ease;
}

.fade-leave-active {
  position: absolute;
  left: 0;
  right: 0;
}

.fade-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* ---- Responsive ---- */
@media (max-width: 640px) {
  .setup-card {
    padding: 28px 24px;
    border-radius: 16px;
  }

  .api-type-group {
    flex-direction: column;
  }

  .api-type-card {
    flex-direction: row;
    padding: 12px 16px;
  }

  .step {
    padding: 6px 10px;
    font-size: 12px;
  }

  .step-label {
    display: none;
  }

  .step-line {
    width: 12px;
  }

  .btn {
    padding: 8px 16px;
    font-size: 13px;
  }
}
</style>
