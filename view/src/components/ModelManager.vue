<template>
  <div class="model-manager">
    <header class="toolbar">
      <div class="toolbar-left">
        <a href="#/" class="back-btn">← 聊天</a>
        <span class="toolbar-divider">|</span>
        <span class="toolbar-title">模型管理</span>
      </div>
      <div class="toolbar-right">
        <select v-model="filterCategory" @change="onCategoryChange" class="filter-select">
          <option v-for="c in CATEGORIES" :key="c.value" :value="c.value">{{ c.label }}</option>
        </select>
        <button class="save-btn" @click="openEditor()">＋ 添加模型</button>
      </div>
    </header>

    <div class="model-list-page">
      <div class="mm-table-wrap">
        <table class="mm-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>提供商</th>
              <th>模型标识</th>
              <th>分类</th>
              <th>API Key</th>
              <th>Base URL</th>
              <th>思考</th>
              <th>默认</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in filteredModels" :key="m.id">
              <td><strong>{{ m.name }}</strong><span v-if="m.is_base" class="base-tag">⭐</span><br><span class="desc">{{ m.description }}</span></td>
              <td>{{ providerLabel(m.provider) }}</td>
              <td><code>{{ m.model }}</code></td>
              <td>{{ categoryLabel(m.category) }}</td>
              <td>{{ maskKey(m.api_key) }}</td>
              <td>{{ m.base_url || '-' }}</td>
              <td><span :class="['think-badge', 'think-' + (m.thinking_level || 'off')]">{{ thinkLabel(m.thinking_level) }}</span></td>
              <td>
                <button :class="['default-badge', { active: m.is_default }]" @click="setDefault(m)">
                  {{ m.is_default ? '默认' : '设为默认' }}
                </button>
              </td>
              <td>
                <button class="edit-btn" @click="openEditor(m)">编辑</button>
                <button class="del-btn" @click="onDelete(m)">删除</button>
                <button v-if="!m.is_base" class="base-btn" @click="setBase(m)">⭐ 基础</button>
              </td>
            </tr>
            <tr v-if="filteredModels.length === 0">
              <td colspan="9" class="empty-row">暂无模型，点击「添加模型」开始配置</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 编辑弹窗 -->
    <Teleport to="body">
      <div v-if="showEditor" class="dialog-overlay">
        <div class="dialog">
          <!-- Step 1: Category selection (new model only) -->
          <template v-if="!editingId && formStep === 'category'">
            <h3>选择模型类别</h3>
            <div class="category-group">
              <label
                v-for="c in CATEGORIES"
                :key="c.value"
                :class="['category-card', { selected: form.category === c.value }]"
                @click="form.category = c.value"
              >
                <span class="category-icon">{{ c.value === 'llm' ? '💬' : c.value === 'image' ? '🖼' : c.value === 'voice' ? '🔊' : '🎬' }}</span>
                <span class="category-name">{{ c.label }}</span>
              </label>
            </div>
            <div class="dialog-footer">
              <button class="save-btn" @click="goToDetails">下一步 →</button>
              <button class="cancel-btn" @click="showEditor = false">取消</button>
            </div>
          </template>

          <!-- Step 2: Model details -->
          <template v-if="editingId || formStep === 'details'">
            <h3>{{ editingId ? '编辑模型' : `添加模型 · ${categoryLabel(form.category)}` }}</h3>
            <h4>接口类型</h4>
            <div class="api-type-group">
              <label
                v-for="t in API_TYPES"
                :key="t.value"
                :class="['api-type-card', { selected: apiType === t.value }]"
              >
                <span class="api-type-name">{{ t.label }}</span>
                <input type="radio" v-model="apiType" :value="t.value" />
              </label>
            </div>

            <div class="form-grid">
              <label>提供商</label>
              <select v-model="form.provider">
                <option v-for="p in availableProviders" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>

              <label>显示名称</label>
              <input v-model="form.name" placeholder="类别-提供商" />

              <label>模型标识</label>
              <input v-model="form.model" placeholder="例: gpt-4o, claude-sonnet-4-20250514" />

              <label>API Key</label>
              <div class="key-wrap">
                <input v-model="form.api_key" :type="showKey ? 'text' : 'password'" placeholder="sk-..." />
                <button type="button" class="key-toggle" @click="showKey = !showKey">{{ showKey ? '🙈' : '👁' }}</button>
              </div>

              <label>Base URL</label>
              <input v-model="form.base_url" placeholder="留空则使用默认地址" />

              <label>描述</label>
              <input v-model="form.description" placeholder="模型描述" />

              <label>设为默认</label>
              <label class="checkbox"><input type="checkbox" v-model="form.is_default" /> 该分类的默认模型</label>

              <label>基础模型</label>
              <label class="checkbox"><input type="checkbox" v-model="form.is_base" /> 标记为基础模型（全局唯一）</label>

              <template v-if="form.category === 'llm'">
                <label>思考等级</label>
                <select v-model="form.thinking_level">
                  <option v-for="t in THINK_LEVELS" :key="t.value" :value="t.value">{{ t.label }}</option>
                </select>

                <label>多模态</label>
                <label class="checkbox"><input type="checkbox" v-model="form.supports_multimodal" /> 支持多模态（图片输入）</label>
              </template>
            </div>
            <div class="dialog-footer">
              <button v-if="!editingId" class="cancel-btn" @click="formStep = 'category'">← 上一步</button>
              <button class="save-btn" @click="onSave">保存</button>
              <button class="cancel-btn" @click="showEditor = false">取消</button>
            </div>
          </template>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useModelStore, API_TYPES, PROVIDERS_BY_API, CATEGORIES } from '@/stores/model'
import type { AIModel } from '@/stores/model'
import { THINK_LEVELS, thinkLabel } from '@/constants'

const store = useModelStore()
const filterCategory = ref('llm')
const showEditor = ref(false)
const editingId = ref<number | null>(null)
const apiType = ref('openai')
const formStep = ref<'category' | 'details'>('category')
const showKey = ref(false)

const form = ref({
  name: '', provider: 'openai', model: '', category: 'llm',
  api_key: '', base_url: '', description: '', is_default: false, is_base: false,
  supports_multimodal: false,
  thinking_level: 'off',
})

const availableProviders = computed(() => PROVIDERS_BY_API[apiType.value] || [])

const filteredModels = computed(() => store.getByCategory(filterCategory.value))

// Fetch provider defaults early
store.fetchProviderDefaults().catch(() => {})

// When API type changes, reset provider and name
watch(apiType, () => {
  form.value.provider = availableProviders.value[0]?.value || ''
  form.value.name = `${form.value.category}-${form.value.provider}`
})

// Auto-fill model/base_url/name when provider changes
watch(() => form.value.provider, (p) => {
  const def = store.providerDefaults[apiType.value]?.[p]
  if (def) {
    form.value.model = def.model
    form.value.base_url = def.baseUrl || ''
  }
  form.value.name = `${form.value.category}-${p}`
})

// Auto-fill name when category changes
watch(() => form.value.category, (c) => {
  form.value.name = `${c}-${form.value.provider}`
})

function onCategoryChange() {
  store.fetchModels(filterCategory.value)
}

function providerLabel(p: string) {
  // Search across all API types
  for (const providers of Object.values(PROVIDERS_BY_API)) {
    const found = providers.find(x => x.value === p)
    if (found) return found.label
  }
  return p
}

function categoryLabel(c: string) {
  return CATEGORIES.find(x => x.value === c)?.label || c
}

function maskKey(key: string) {
  if (!key) return '-'
  if (key.length <= 8) return '****'
  return key.slice(0, 4) + '****' + key.slice(-4)
}

async function openEditor(m?: AIModel) {
	await store.fetchProviderDefaults()
  if (m) {
    editingId.value = m.id
    form.value = {
      name: m.name, provider: m.provider, model: m.model, category: m.category,
      api_key: m.api_key || '', base_url: m.base_url || '', description: m.description || '',
      is_default: m.is_default, is_base: m.is_base,
      supports_multimodal: m.supports_multimodal || false,
	      thinking_level: m.thinking_level || 'off',
    }
    // Try to infer API type from provider
    for (const [k, providers] of Object.entries(PROVIDERS_BY_API)) {
      if (providers.some(p => p.value === m.provider)) {
        apiType.value = k
        break
      }
    }
  } else {
    editingId.value = null
    formStep.value = 'category'
    apiType.value = 'openai'
    form.value = {
      name: '', provider: 'openai', model: '', category: filterCategory.value,
      api_key: '', base_url: '', description: '', is_default: false, is_base: false,
      supports_multimodal: false,
	      thinking_level: 'off',
    }
  }
  showEditor.value = true
}

function goToDetails() {
  form.value.name = `${form.value.category}-${form.value.provider}`
  // Auto-fill model/base_url for current provider (watch won't fire if unchanged)
  const def = store.providerDefaults[apiType.value]?.[form.value.provider]
  if (def) {
    form.value.model = def.model
    form.value.base_url = def.baseUrl || ''
  }
  formStep.value = 'details'
}

async function onSave() {
  if (!form.value.name || !form.value.model) {
    alert('请填写显示名称和模型标识')
    return
  }
  if (editingId.value) {
    await store.updateModel(editingId.value, form.value)
  } else {
    await store.createModel(form.value)
  }
  showEditor.value = false
}

async function setDefault(m: AIModel) {
  if (!m.is_default) {
    await store.setDefault(m.id)
  }
}

async function setBase(m: AIModel) {
  await store.setBase(m.id)
}

async function onDelete(m: AIModel) {
  if (confirm(`确定删除模型「${m.name}」？`)) {
    await store.deleteModel(m.id)
  }
}

onMounted(() => store.fetchModels())
</script>

<style scoped>
.model-manager { height: 100vh; display: flex; flex-direction: column; background: #f5f5f5; }

/* Toolbar — matches FlowDesigner */
.toolbar { display: flex; align-items: center; justify-content: space-between; padding: 0 16px; height: 44px; background: #fff; border-bottom: 1px solid #e0e0e0; flex-shrink: 0; }
.toolbar-left { display: flex; align-items: center; gap: 8px; }
.toolbar-right { display: flex; align-items: center; gap: 6px; }
.toolbar-divider { color: #ddd; font-size: 16px; user-select: none; }
.toolbar-title { font-size: 14px; font-weight: 600; color: #333; }
.back-btn { color: #4a9eff; text-decoration: none; font-size: 13px; padding: 4px 8px; border-radius: 4px; }
.back-btn:hover { background: #f0f5ff; }
.filter-select { padding: 5px 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 13px; outline: none; background: #fff; }
.filter-select:focus { border-color: #4a9eff; }
.toolbar .save-btn { background: #4a9eff; color: #fff; border: none; padding: 6px 14px; border-radius: 4px; cursor: pointer; font-size: 13px; font-weight: 500; }
.toolbar .save-btn:hover { background: #3a8eef; }

/* Content area */
.model-list-page { flex: 1; overflow-y: auto; padding: 20px; }
.mm-table-wrap { background: #fff; border: 1px solid #e0e0e0; border-radius: 8px; overflow-x: auto; }
.mm-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.mm-table th { background: #fafafa; padding: 10px 12px; text-align: left; color: #888; font-weight: 500; border-bottom: 1px solid #eee; }
.mm-table td { padding: 10px 12px; border-bottom: 1px solid #f0f0f0; vertical-align: top; }
.mm-table tr:hover td { background: #fafbfc; }
.desc { font-size: 11px; color: #aaa; }
.mm-table code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-size: 12px; }
.empty-row { text-align: center; color: #bbb; padding: 40px; }

.default-badge { border: 1px solid #ddd; background: #fff; padding: 3px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; color: #999; }
.default-badge.active { background: #e6f7ff; border-color: #4a9eff; color: #4a9eff; cursor: default; }
.edit-btn { background: none; border: 1px solid #ddd; padding: 3px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; color: #666; margin-right: 4px; }
.edit-btn:hover { border-color: #4a9eff; color: #4a9eff; }
.del-btn { background: none; border: 1px solid #ddd; padding: 3px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; color: #999; }
.del-btn:hover { border-color: #ff4d4f; color: #ff4d4f; }
.base-btn { background: none; border: 1px solid #ddd; padding: 3px 8px; border-radius: 4px; cursor: pointer; font-size: 11px; color: #999; margin-left: 4px; }
.base-btn:hover { border-color: #f0a030; color: #f0a030; }
.base-tag { font-size: 12px; margin-left: 4px; }

/* Thinking badge */
.think-badge { font-size: 11px; padding: 2px 8px; border-radius: 10px; font-weight: 600; }
.think-off { background: #f1f5f9; color: #94a3b8; }
.think-low { background: #fef3c7; color: #d97706; }
.think-medium { background: #e0e7ff; color: #4f46e5; }
.think-high { background: #dcfce7; color: #16a34a; }
.think-max { background: #fce7f3; color: #db2777; }

.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.dialog { background: #fff; border-radius: 12px; padding: 24px; width: 540px; max-height: 85vh; overflow-y: auto; box-shadow: 0 8px 30px rgba(0,0,0,0.15); }
.dialog h3 { font-size: 18px; margin-bottom: 16px; }
.form-grid { display: grid; grid-template-columns: 100px 1fr; gap: 10px; align-items: center; }
.form-grid label { font-size: 13px; color: #666; font-weight: 500; }
.form-grid input, .form-grid select { padding: 7px 10px; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; }
.key-wrap { position: relative; display: flex; align-items: center; }
.key-wrap input { flex: 1; padding-right: 36px; }
.key-toggle { position: absolute; right: 2px; top: 50%; transform: translateY(-50%); background: none; border: none; cursor: pointer; font-size: 16px; padding: 4px 8px; line-height: 1; }
.form-grid .checkbox { display: flex; align-items: center; gap: 6px; font-weight: 400; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
.dialog-footer .save-btn { background: #4a9eff; color: #fff; border: none; padding: 8px 20px; border-radius: 6px; cursor: pointer; font-size: 14px; }
.dialog-footer .save-btn:hover { background: #3a8eef; }
.cancel-btn { background: #fff; border: 1px solid #ddd; padding: 8px 20px; border-radius: 6px; cursor: pointer; font-size: 14px; color: #999; }
.cancel-btn:hover { color: #333; border-color: #bbb; }

/* API type selector */
.api-type-group {
  display: flex;
  gap: 10px;
  margin-bottom: 18px;
}
.api-type-card {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px 8px;
  border: 2px solid #eef0f2;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.25s ease;
  background: #fafbfc;
}
.api-type-card:hover { border-color: #c8d6e5; background: #f0f4f8; }
.api-type-card.selected { border-color: #4a9eff; background: #f0f6ff; }
.api-type-name { font-size: 13px; font-weight: 600; color: #444; }
.api-type-card.selected .api-type-name { color: #4a9eff; }
.api-type-card input { display: none; }

.dialog h4 { font-size: 13px; font-weight: 600; color: #444; margin: 0 0 10px 0; }

/* Category selector */
.category-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 20px;
}
.category-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px 12px;
  border: 2px solid #eef0f2;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.25s ease;
  background: #fafbfc;
}
.category-card:hover { border-color: #c8d6e5; background: #f0f4f8; transform: translateY(-2px); }
.category-card.selected { border-color: #4a9eff; background: #f0f6ff; box-shadow: 0 4px 16px rgba(74,158,255,0.12); }
.category-icon { font-size: 28px; line-height: 1; }
.category-name { font-size: 14px; font-weight: 600; color: #444; }
.category-card.selected .category-name { color: #4a9eff; }
</style>
