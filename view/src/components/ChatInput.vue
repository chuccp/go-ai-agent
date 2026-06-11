<template>
  <div :class="['chat-input', { 'center-mode': centerMode }]">
    <!-- File previews -->
    <div v-if="files.length > 0" class="file-preview">
      <div v-for="(f, i) in files" :key="i" class="file-tag">
        <span class="file-icon">{{ fileIcon(f.type) }}</span>
        <span class="file-name">{{ f.name }}</span>
        <span class="file-size">{{ formatSize(f.size) }}</span>
        <button class="file-remove" @click="removeFile(i)" title="移除">×</button>
      </div>
    </div>

    <div :class="['input-wrap', { disabled }]">
      <!-- File picker -->
      <button v-if="acceptFiles" class="attach-btn" :disabled="disabled" @click="triggerFileInput" title="上传文件">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
      </button>
      <input ref="fileInputRef" type="file" :accept="acceptFiles" multiple hidden @change="onFileChange" />

      <textarea
        ref="inputRef" v-model="text"
        placeholder="输入消息… (Enter 发送, Shift+Enter 换行)"
        :disabled="disabled"
        @keydown="handleKeydown" @input="autoResize" rows="1"
      ></textarea>

      <!-- Gemini-style model picker: button → popover -->
      <div v-if="models && models.length" class="model-picker-wrap">
        <button class="model-pill" :disabled="disabled" @click="showPopover = !showPopover" @blur="onPopoverBlur">
          <span class="pill-model">{{ currentModelName }}</span>
          <span v-if="thinkLevel !== undefined && thinkLevel !== 'off'" class="pill-think">· {{ thinkLabel(thinkLevel) }}</span>
          <span class="pill-arrow">▾</span>
        </button>
        <div v-if="showPopover" class="model-popover" @mousedown.prevent>
          <div class="popover-label">模型</div>
          <button
            v-for="m in models" :key="m.id"
            :class="['popover-item', { active: m.id === modelId }]"
            @click="selectModel(m)"
          >{{ m.name }}</button>
          <div class="popover-divider"></div>
          <div class="popover-label">思考</div>
          <div class="think-options">
            <button
              v-for="t in THINK_LEVELS" :key="t.value"
              :class="['think-opt', { active: thinkLevel === t.value }]"
              @click="selectThink(t.value)"
            >{{ t.label }}</button>
          </div>
        </div>
      </div>

      <button class="send-btn" :disabled="disabled || (!text.trim() && files.length === 0)" @click="submit" title="发送">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
      </button>
    </div>
    <div class="input-hint" v-if="!disabled && !centerMode">Enter 发送 · Shift+Enter 换行</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'

interface DBModel {
  id: string; name: string; provider?: string; model?: string
  category?: string; is_default?: boolean; thinking_level?: string
}

import { THINK_LEVELS, thinkLabel, fileIcon, formatSize } from '@/constants'

const props = defineProps<{
  disabled: boolean
  centerMode: boolean
  acceptFiles?: string
  thinkLevel?: string
  models?: DBModel[]
  modelId?: string
}>()
const emit = defineEmits<{
  send: [content: string]
  upload: [files: File[]]
  'think-change': [level: string]
  'model-change': [id: string]
}>()

const text = ref('')
const files = ref<File[]>([])
const inputRef = ref<HTMLTextAreaElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const showPopover = ref(false)

const currentModelName = computed(() =>
  props.models?.find(m => m.id === props.modelId)?.name || '选择模型'
)

function selectModel(m: DBModel) {
  emit('model-change', m.id)
  showPopover.value = false
}
function selectThink(level: string) {
  emit('think-change', level)
}
function onPopoverBlur(e: FocusEvent) {
  const t = e.relatedTarget as HTMLElement
  if (!t || !t.closest('.model-popover')) showPopover.value = false
}
function onClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.model-picker-wrap')) showPopover.value = false
}
onMounted(() => document.addEventListener('mousedown', onClickOutside))
onUnmounted(() => document.removeEventListener('mousedown', onClickOutside))

function submit() {
  const hasText = text.value.trim().length > 0
  const hasFiles = files.value.length > 0
  if (!hasText && !hasFiles) return
  if (hasFiles) { emit('upload', [...files.value]); files.value = [] }
  if (hasText) { emit('send', text.value.trim()); text.value = ''; nextTick(() => autoResize()) }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit() }
}

function autoResize() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 160) + 'px'
}

function triggerFileInput() { fileInputRef.value?.click() }

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) { for (let i = 0; i < input.files.length; i++) files.value.push(input.files[i]) }
  input.value = ''
}

function removeFile(i: number) { files.value.splice(i, 1) }

</script>

<style scoped>
.chat-input {
  padding: 14px 20px 10px;
  background: #fff;
  border-top: 1px solid #e2e8f0;
}
.chat-input.center-mode { border-top: none; padding: 0; width: 100%; max-width: 720px; margin: 0 auto; }

.file-preview { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; max-width: 820px; margin-left: auto; margin-right: auto; }
.file-tag { display: flex; align-items: center; gap: 4px; background: #f0f4ff; border: 1px solid #c7d2fe; border-radius: 8px; padding: 4px 8px; font-size: 12px; color: #4f46e5; }
.file-icon { font-size: 14px; }
.file-name { max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-size { color: #94a3b8; font-size: 11px; }
.file-remove { background: none; border: none; cursor: pointer; color: #94a3b8; font-size: 16px; padding: 0 2px; line-height: 1; }
.file-remove:hover { color: #ef4444; }

.input-wrap {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  padding: 6px 6px 6px 14px;
  transition: border-color 0.2s, box-shadow 0.2s;
  max-width: 820px;
  margin: 0 auto;
}
.input-wrap:focus-within { border-color: #6366f1; box-shadow: 0 0 0 3px rgba(99,102,241,0.08); background: #fff; }
.input-wrap.disabled { opacity: 0.6; }

.attach-btn {
  width: 36px; height: 36px; border-radius: 50%; border: none; background: transparent;
  color: #94a3b8; cursor: pointer; display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; transition: color 0.15s, background 0.15s;
}
.attach-btn:hover:not(:disabled) { color: #6366f1; background: #eef2ff; }
.attach-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* Gemini-style model picker */
.model-picker-wrap { position: relative; flex-shrink: 0; }
.model-pill {
  display: flex; align-items: center; gap: 3px;
  padding: 4px 8px; border: 1px solid #e2e8f0; border-radius: 14px;
  background: #fff; cursor: pointer; font-size: 12px; color: #334155;
  white-space: nowrap;
}
.model-pill:hover { border-color: #6366f1; }
.model-pill:disabled { opacity: 0.5; cursor: not-allowed; }
.pill-model { font-weight: 500; }
.pill-think { color: #6366f1; font-weight: 600; font-size: 11px; }
.pill-arrow { font-size: 10px; color: #94a3b8; }

.model-popover {
  position: absolute; bottom: 100%; right: 0;
  margin-bottom: 8px; background: #fff; border: 1px solid #e2e8f0;
  border-radius: 12px; box-shadow: 0 8px 30px rgba(0,0,0,0.12);
  padding: 8px; min-width: 180px; z-index: 100;
}
.popover-label { font-size: 11px; color: #94a3b8; padding: 4px 8px 2px; font-weight: 500; }
.popover-divider { height: 1px; background: #f1f5f9; margin: 6px 0; }
.popover-item {
  display: block; width: 100%; padding: 6px 8px; border: none; border-radius: 8px;
  background: transparent; cursor: pointer; text-align: left; font-size: 13px; color: #334155;
}
.popover-item:hover { background: #f8fafc; }
.popover-item.active { background: #eef2ff; color: #6366f1; font-weight: 600; }
.think-options { display: flex; gap: 2px; padding: 2px 4px; }
.think-opt {
  flex: 1; padding: 4px 0; border: 1px solid #e2e8f0; border-radius: 6px;
  background: transparent; cursor: pointer; text-align: center; font-size: 11px; color: #64748b;
}
.think-opt:hover { background: #f8fafc; border-color: #c7d2fe; }
.think-opt.active { background: #eef2ff; border-color: #6366f1; color: #6366f1; font-weight: 600; }

textarea {
  flex: 1; border: none; background: transparent; font-size: 15px;
  font-family: inherit; line-height: 1.5; resize: none; outline: none;
  padding: 9px 0; color: #1e293b; max-height: 160px; min-height: 24px;
}
textarea::placeholder { color: #94a3b8; }
textarea:disabled { cursor: not-allowed; }

.send-btn {
  width: 38px; height: 38px; border-radius: 50%; border: none; background: #6366f1;
  color: #fff; cursor: pointer; display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; transition: background 0.15s, transform 0.15s;
}
.send-btn:hover:not(:disabled) { background: #4f46e5; transform: scale(1.05); }
.send-btn:disabled { background: #cbd5e1; cursor: not-allowed; }

.input-hint {
  text-align: center; font-size: 11px; color: #94a3b8; margin-top: 6px;
  max-width: 820px; margin-left: auto; margin-right: auto;
}
</style>
