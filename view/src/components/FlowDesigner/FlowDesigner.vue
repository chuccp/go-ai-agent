<template>
  <div class="flow-designer">
    <!-- Flow List View (no flow selected) -->
    <div v-if="!isEdit" class="flow-list-page">
      <header class="toolbar">
        <div class="toolbar-left"><a href="#/" class="back-btn">← 聊天</a></div>
        <div class="toolbar-right">
          <button class="icon-btn" @click="importFlow" title="导入JSON">📤</button>
          <button class="icon-btn" @click="toggleDark">{{ darkMode ? '☀' : '🌙' }}</button>
          <button class="save-btn" @click="createNew">＋ 新建流程</button>
        </div>
      </header>
      <div class="flow-grid">
        <div v-for="f in flowStore.flows" :key="f.id" class="flow-card">
          <div class="card-name">{{ f.name }}</div>
          <div class="card-meta">{{ f.category || '未分类' }} · {{ f.updated_at?.substring(0,10) }}</div>
          <div class="card-actions">
            <button class="card-btn edit" @click="router.push('/designer/' + f.id)">✏️ 编辑</button>
            <button class="card-btn dup" @click="duplicateFlow(f.id)">📋 复制</button>
            <button class="card-btn del" @click="deleteFlow(f.id)">🗑 删除</button>
          </div>
        </div>
        <div v-if="flowStore.flows.length === 0" class="empty-list">暂无流程，点击"新建流程"开始</div>
      </div>
    </div>

    <!-- Editor View (editing a flow) -->
    <template v-else>
      <header class="toolbar">
        <div class="toolbar-left">
          <a href="#/designer" class="back-btn">← 流程列表</a>
          <span class="toolbar-divider">|</span>
          <input v-model="flowName" placeholder="流程名称" class="name-input" />
          <input v-model="flowCategory" placeholder="分类" class="cat-input" />
          <span v-if="saveMsg" :class="['save-msg', saveOk ? 'ok' : 'err']">{{ saveMsg }}</span>
        </div>
        <div class="toolbar-right">
          <button class="icon-btn" @click="duplicateFlow(flowId!)" title="复制">📋</button>
          <button class="icon-btn" @click="exportFlow" title="导出">📥</button>
          <button class="icon-btn" @click="importFlow" title="导入">📤</button>
          <button class="icon-btn" @click="toggleDark">{{ darkMode ? '☀' : '🌙' }}</button>
          <button v-if="flowId" class="icon-btn del-btn" @click="deleteFlow(flowId)">🗑</button>
          <button class="save-btn" @click="saveFlow">💾 保存</button>
        </div>
      </header>
      <div class="designer-body">
        <NodePanel />
        <Canvas :nodes="nodes" :edges="edges" @update:nodes="nodes = $event" @update:edges="edges = $event" @select-node="selectedNode = $event" />
        <PropertyPanel :node="selectedNode" :nodes="nodes" :edges="edges" @update:node="updateNode" @delete:node="deleteNode" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFlowStore } from '@/stores/flow'
import type { FlowNode, FlowEdge } from '@/types/flow'
import NodePanel from './NodePanel.vue'
import Canvas from './Canvas.vue'
import PropertyPanel from './PropertyPanel.vue'

const route = useRoute()
const router = useRouter()
const flowStore = useFlowStore()

const routeId = computed(() => route.params.id as string | undefined)
const flowId = computed(() => {
  const id = routeId.value
  if (!id) return null
  const n = Number(id)
  return isNaN(n) ? null : n
})
const isNew = computed(() => routeId.value === 'new')
const isEdit = computed(() => flowId.value !== null || isNew.value)

const flowName = ref('')
const flowCategory = ref('picture_book')
const nodes = ref<FlowNode[]>([])
const edges = ref<FlowEdge[]>([])
const selectedNode = ref<FlowNode | null>(null)

onMounted(async () => {
  if (flowId.value) {
    const flow = await flowStore.fetchFlow(flowId.value)
    if (flow) {
      flowName.value = flow.name
      flowCategory.value = flow.category
      nodes.value = flow.nodes || []
      edges.value = flow.edges || []
    }
  }
  // new flow: keep empty defaults
})

function updateNode(updated: FlowNode) {
  const idx = nodes.value.findIndex(n => n.id === updated.id)
  if (idx >= 0) {
    nodes.value[idx] = updated
  }
}

function deleteNode(nodeId: number) {
  nodes.value = nodes.value.filter(n => n.id !== nodeId)
  edges.value = edges.value.filter(e => e.source_node_id !== nodeId && e.target_node_id !== nodeId)
  if (selectedNode.value?.id === nodeId) {
    selectedNode.value = null
  }
}

const saveMsg = ref('')
const darkMode = ref(false)
const saveOk = ref(false)

function toggleDark() { darkMode.value = !darkMode.value; document.documentElement.classList.toggle("dark", darkMode.value) }
function exportFlow() { const blob = new Blob([JSON.stringify({name:flowName.value,category:flowCategory.value,nodes:nodes.value,edges:edges.value},null,2)],{type:"application/json"}); const a=document.createElement("a"); a.href=URL.createObjectURL(blob); a.download=(flowName.value||"flow")+".json"; a.click() }
function importFlow() { const input=document.createElement("input"); input.type="file"; input.accept=".json"; input.onchange=async(e)=>{ const f=e.target.files[0]; if(!f)return; const text=await f.text(); try{ const data=JSON.parse(text); flowName.value=data.name||""; flowCategory.value=data.category||""; nodes.value=data.nodes||[]; edges.value=data.edges||[] }catch(ex){ alert("JSON 格式错误") } }; input.click() }
async function saveFlow() {
  saveMsg.value = ''
  if (!flowName.value.trim()) {
    saveMsg.value = '请输入流程名称'; saveOk.value = false; return
  }
  const hasStart = nodes.value.some(n => n.type === 'start')
  const hasEnd = nodes.value.some(n => n.type === 'end')
  if (!hasStart || !hasEnd) {
    saveMsg.value = '流程必须包含开始和结束节点'; saveOk.value = false; return
  }

  const result = await flowStore.saveFlow({
    name: flowName.value, description: '', category: flowCategory.value,
    nodes: nodes.value.map(n => ({
      type: n.type, label: n.label,
      config: typeof n.config === 'string' ? n.config : JSON.stringify(n.config),
      position_x: n.position_x, position_y: n.position_y,
    })),
    edges: edges.value.map(e => ({
      source_node_id: e.source_node_id, target_node_id: e.target_node_id,
      source_handle: e.source_handle || 'output',
      target_handle: e.target_handle || 'input',
      label: e.label || '',
    })),
  }, flowId.value || undefined)

  if (result) {
    saveMsg.value = '保存成功'; saveOk.value = true
    if (!flowId.value) router.push(`/designer/${result.id}`)
  } else {
    saveMsg.value = '保存失败'; saveOk.value = false
  }
  setTimeout(() => { saveMsg.value = '' }, 2000)
}

function createNew() {
  flowName.value = ''; flowCategory.value = 'picture_book'; nodes.value = []; edges.value = []
  router.push('/designer/new')
}

async function duplicateFlow(id: number) {
  const res = await fetch(`${API_BASE}/api/flows/${id}/duplicate`, { method: 'POST' })
  if (res.ok) { await flowStore.fetchFlows() }
}

async function deleteFlow(id: number) {
  if (!confirm('确定删除？')) return
  await flowStore.deleteFlow(id)
  if (flowId.value === id) router.push('/designer')
}
</script>

<style scoped>
.flow-designer { height: 100vh; display: flex; flex-direction: column; background: #f5f5f5; }
.toolbar { display: flex; align-items: center; justify-content: space-between; padding: 0 16px; height: 44px; background: #fff; border-bottom: 1px solid #e0e0e0; flex-shrink: 0; }
.toolbar-left { display: flex; align-items: center; gap: 8px; }
.toolbar-right { display: flex; align-items: center; gap: 6px; }
.toolbar-divider { color: #ddd; font-size: 16px; user-select: none; }
.back-btn { color: #4a9eff; text-decoration: none; font-size: 13px; padding: 4px 8px; border-radius: 4px; }
.back-btn:hover { background: #f0f5ff; }
.name-input, .cat-input { padding: 5px 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 13px; outline: none; }
.name-input:focus, .cat-input:focus { border-color: #4a9eff; }
.name-input { width: 160px; }
.cat-input { width: 100px; font-size: 12px; }
.icon-btn { background: none; border: 1px solid transparent; padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 14px; }
.icon-btn:hover { background: #f0f0f0; border-color: #ddd; }
.save-btn { background: #4a9eff; color: #fff; border: none; padding: 6px 14px; border-radius: 4px; cursor: pointer; font-size: 13px; font-weight: 500; }
.save-btn:hover { background: #3a8eef; }
.save-msg { font-size: 12px; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.save-msg.ok { color: #52c41a; background: #f6ffed; }
.save-msg.err { color: #ff4d4f; background: #fff2f0; }
.designer-body { flex: 1; display: flex; overflow: hidden; }
.flow-list-page { flex: 1; overflow-y: auto; }
.flow-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px; padding: 20px; }
.flow-card { background: #fff; border: 1px solid #e0e0e0; border-radius: 8px; padding: 16px; }
.flow-card:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.card-name { font-size: 15px; font-weight: 600; color: #333; margin-bottom: 4px; }
.card-meta { font-size: 12px; color: #999; margin-bottom: 12px; }
.card-actions { display: flex; gap: 6px; }
.card-btn { padding: 4px 10px; border: 1px solid #e0e0e0; border-radius: 4px; font-size: 12px; cursor: pointer; background: #fff; }
.card-btn.edit:hover { border-color: #4a9eff; color: #4a9eff; }
.card-btn.del:hover { border-color: #ff4d4f; color: #ff4d4f; }
.card-btn.dup:hover { border-color: #52c41a; color: #52c41a; }
.empty-list { grid-column: 1/-1; text-align: center; padding: 60px; color: #bbb; font-size: 16px; }
.del-btn:hover { background: #fff2f0 !important; border-color: #ff4d4f !important; color: #ff4d4f !important; }
</style>
