<template>
  <div class="bottom-bar">
    <div class="zoom-group">
      <button class="bar-btn" @click="$emit('zoom-out')" title="缩小"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="5" y1="12" x2="19" y2="12"/></svg></button>
      <span class="bar-pct">{{ Math.round((zoom || 1) * 100) }}%</span>
      <button class="bar-btn" @click="$emit('zoom-in')" title="放大"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg></button>
      <button class="bar-btn bar-reset" @click="$emit('zoom-reset')" title="重置">↺</button>
    </div>
    <button class="add-btn" @click="show = !show" title="添加节点">＋</button>

    <!-- Popover: nodes grouped by category -->
    <div v-if="show" class="node-popover" @mousedown.stop>
      <div class="popover-groups">
        <div v-for="grp in groupedTypes" :key="grp.key" class="node-group">
          <div class="group-label">{{ grp.label }}</div>
          <div class="group-grid">
            <button
              v-for="nt in grp.items"
              :key="nt.type"
              class="node-item"
              :style="{ borderColor: nt.color }"
              @click="addNode(nt)"
            >
              <span class="item-icon">{{ nt.icon }}</span>
              <span class="item-label">{{ nt.label }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { NodeType } from '@/types/flow'

import { ALL_NODE_TYPES, NODE_CATEGORIES } from '@/types/flow'

const props = defineProps<{ zoom?: number }>()
const emit = defineEmits<{ 'add-node': [type: NodeType, label: string]; 'zoom-in': []; 'zoom-out': []; 'zoom-reset': [] }>()

const show = ref(false)
const showZoom = ref(false)

const groupedTypes = computed(() =>
  NODE_CATEGORIES.map(g => ({ key: g.key, label: g.label, items: ALL_NODE_TYPES.filter(n => n.category === g.key) }))
)

function addNode(nt: typeof ALL_NODE_TYPES[number]) {
  emit('add-node', nt.type, nt.label)
  show.value = false
}

// Close popover on outside click
function onClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.node-adder')) show.value = false
}
import { onMounted, onUnmounted } from 'vue'
onMounted(() => document.addEventListener('mousedown', onClickOutside))
onUnmounted(() => document.removeEventListener('mousedown', onClickOutside))
</script>

<style scoped>
.bottom-bar {
  position: absolute; bottom: 16px; left: 50%; transform: translateX(-50%);
  z-index: 80; display: flex; align-items: center; gap: 8px;
  background: #fff; border-radius: 12px; padding: 4px 8px;
  box-shadow: 0px 1px 3px rgba(16,24,40,0.1), 0px 1px 2px rgba(16,24,40,0.06);
  border: 1px solid #eaecf0;
}

.zoom-group { display: flex; align-items: center; gap: 2px; }
.bar-btn {
  width: 32px; height: 32px; border: none; border-radius: 50%;
  background: transparent; color: #64748b; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: background 0.15s;
}
.bar-btn:hover { background: #f1f5f9; }
.bar-reset { font-size: 14px; }
.bar-pct { font-size: 12px; color: #64748b; font-weight: 500; min-width: 36px; text-align: center; }

.add-btn {
  width: 36px; height: 36px; border-radius: 50%; border: none;
  background: #6366f1; color: #fff; font-size: 22px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: transform 0.15s, box-shadow 0.15s;
  line-height: 1; flex-shrink: 0;
}
.add-btn:hover { transform: scale(1.08); }

.node-popover {
  position: absolute; bottom: 56px; left: 50%; transform: translateX(-50%);
  background: #fff; border: 1px solid #e2e8f0; border-radius: 14px;
  box-shadow: 0 8px 30px rgba(0,0,0,0.12); padding: 10px 12px;
  width: 400px; max-height: 380px; overflow-y: auto;
}

.popover-groups { display: flex; flex-direction: column; gap: 10px; }
.node-group { }
.group-label { font-size: 11px; color: #94a3b8; font-weight: 600; margin-bottom: 4px; padding-left: 2px; text-transform: uppercase; letter-spacing: 0.5px; }
.group-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 5px; }
.node-item {
  display: flex; align-items: center; gap: 6px;
  padding: 7px 8px; border: 1px solid #e2e8f0; border-left: 3px solid;
  border-radius: 8px; background: #fff; cursor: pointer;
  font-size: 12px; color: #334155; transition: all 0.15s;
}
.node-item:hover { background: #f8fafc; transform: translateY(-1px); box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.item-icon { font-size: 16px; flex-shrink: 0; }
.item-label { font-weight: 500; }
</style>
