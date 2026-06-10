<template>
  <div class="session-list">
    <div class="sidebar-header">
      <span class="logo">🤖 Go AI Agent</span>
    </div>
    <div class="sidebar-new">
      <button class="new-chat-btn" @click="showDialog = true">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        新对话
      </button>
    </div>
    <div class="sessions">
      <div v-for="session in sessions" :key="session.id"
        :class="['session-item', { active: session.id === activeSessionId }]"
        @click="$emit('select', session.id)">
        <span class="title">{{ session.title || '新对话' }}</span>
        <button class="delete-btn" @click.stop="$emit('delete', session.id)" title="删除">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
        </button>
      </div>
    </div>
    <div class="sidebar-footer">
      <a href="#/models" class="nav-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
        模型管理
      </a>
      <a href="#/designer" class="nav-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
        流程设计器
      </a>
    </div>

    <!-- New session dialog -->
    <Teleport to="body">
      <div v-if="showDialog" class="dialog-overlay">
        <div class="dialog">
          <h3>新建会话</h3>
          <p class="dialog-sub">选择模式开始对话</p>
          <button class="dialog-option free" @click="startChat(null)">
            <span class="dopt-icon">💬</span>
            <span class="dopt-text">
              <strong>自由对话</strong>
              <small>不选择流程，直接与 AI 聊天</small>
            </span>
          </button>
          <div v-if="flows.length > 0" class="flow-options">
            <div v-for="f in flows" :key="f.id" class="flow-row">
              <button class="dialog-option flow" @click="startChat(f.id)">
                <span class="dopt-icon">⚡</span>
                <span class="dopt-text">
                  <strong>{{ f.name }}</strong>
                  <small>{{ f.description || f.category }}</small>
                </span>
              </button>
              <button class="flow-del" @click.stop="deleteFlow(f.id)" title="删除流程">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/></svg>
              </button>
            </div>
          </div>
          <div class="dialog-footer">
            <button class="dialog-close" @click="showDialog = false">取消</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useFlowStore } from '@/stores/flow'

interface Session { id: number; title: string; created_at: string; updated_at: string }

defineProps<{ sessions: Session[]; activeSessionId: number | null }>()
const emit = defineEmits<{
  select: [id: number]
  new: [flowId: number | null]
  delete: [id: number]
}>()

const showDialog = ref(false)
const flowStore = useFlowStore()
const flows = flowStore.flows

function startChat(flowId: number | null) {
  showDialog.value = false
  emit('new', flowId)
}

async function deleteFlow(id: number) {
  if (confirm('确定删除此流程？')) {
    await flowStore.deleteFlow(id)
  }
}

onMounted(() => flowStore.fetchFlows())
</script>

<style scoped>
.session-list {
  width: 260px;
  background: #f8fafc;
  border-right: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  height: 100vh;
}
.sidebar-header {
  padding: 16px;
  display: flex;
  align-items: center;
}
.logo {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}
.sidebar-new {
  padding: 0 12px 12px;
}
.new-chat-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 9px 0;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: #334155;
  transition: all 0.15s;
}
.new-chat-btn:hover {
  border-color: #6366f1;
  color: #6366f1;
  background: #eef2ff;
}
.sessions {
  flex: 1;
  overflow-y: auto;
  padding: 0 10px;
}
.session-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 10px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
  margin-bottom: 2px;
}
.session-item:hover { background: #e2e8f0; }
.session-item.active { background: #eef2ff; }
.session-item.active .title { color: #4f46e5; font-weight: 600; }
.title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: #475569;
}
.delete-btn {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s;
}
.session-item:hover .delete-btn { opacity: 1; }
.delete-btn:hover { color: #ef4444; background: #fee2e2; }
.sidebar-footer {
  padding: 10px 12px;
  border-top: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.nav-link {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  font-size: 13px;
  color: #64748b;
  text-decoration: none;
  transition: all 0.15s;
}
.nav-link:hover {
  background: #e2e8f0;
  color: #334155;
}

/* Dialog */
.dialog-overlay {
  position: fixed; inset: 0; background: rgba(15,23,42,0.5);
  z-index: 1000; display: flex; align-items: center; justify-content: center;
  backdrop-filter: blur(2px);
}
.dialog {
  background: #fff; border-radius: 16px; padding: 24px;
  width: 400px; max-height: 75vh; overflow-y: auto;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
}
.dialog h3 { font-size: 18px; margin-bottom: 4px; color: #1e293b; }
.dialog-sub { font-size: 13px; color: #94a3b8; margin-bottom: 20px; }
.dialog-option {
  display: flex; align-items: center; gap: 12px;
  width: 100%; padding: 14px; border: 1px solid #e2e8f0;
  border-radius: 12px; background: #fff; cursor: pointer;
  text-align: left; margin-bottom: 8px; transition: all 0.15s;
}
.dialog-option:hover { border-color: #6366f1; background: #f8fafc; }
.dialog-option.free { border-color: #10b981; }
.dialog-option.free:hover { background: #f0fdf4; }
.dopt-icon { font-size: 20px; flex-shrink: 0; }
.dopt-text { display: flex; flex-direction: column; }
.dopt-text strong { font-size: 14px; color: #1e293b; }
.dopt-text small { font-size: 11px; color: #94a3b8; margin-top: 2px; }
.flow-options { margin-top: 4px; }
.flow-row { display: flex; gap: 6px; align-items: stretch; margin-bottom: 8px; }
.flow-row .dialog-option { flex: 1; margin-bottom: 0; }
.flow-del {
  background: none; border: 1px solid #e2e8f0; border-radius: 10px;
  cursor: pointer; padding: 0 10px; color: #94a3b8;
  display: flex; align-items: center;
  transition: all 0.15s;
}
.flow-del:hover { color: #ef4444; border-color: #ef4444; background: #fef2f2; }
.dialog-footer { display: flex; justify-content: flex-end; margin-top: 12px; }
.dialog-close {
  padding: 8px 18px; border: 1px solid #e2e8f0; border-radius: 8px;
  background: #fff; color: #64748b; cursor: pointer; font-size: 13px;
}
.dialog-close:hover { background: #f1f5f9; color: #334155; }
</style>
