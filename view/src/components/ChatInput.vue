<template>
  <div :class="['chat-input', { 'center-mode': centerMode }]">
    <div :class="['input-wrap', { disabled }]">
      <textarea
        ref="inputRef"
        v-model="text"
        placeholder="输入消息… (Enter 发送, Shift+Enter 换行)"
        :disabled="disabled"
        @keydown="handleKeydown"
        @input="autoResize"
        rows="1"
      ></textarea>
      <button
        class="send-btn"
        :disabled="disabled || !text.trim()"
        @click="submit"
        title="发送"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
      </button>
    </div>
    <div class="input-hint" v-if="!disabled && !centerMode">Enter 发送 · Shift+Enter 换行</div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'

defineProps<{ disabled: boolean; centerMode: boolean }>()
const emit = defineEmits<{ send: [content: string] }>()

const text = ref('')
const inputRef = ref<HTMLTextAreaElement | null>(null)

function submit() {
  if (text.value.trim()) {
    emit('send', text.value.trim())
    text.value = ''
    nextTick(() => autoResize())
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    submit()
  }
}

function autoResize() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 160) + 'px'
}
</script>

<style scoped>
.chat-input {
  padding: 14px 20px 10px;
  background: #fff;
  border-top: 1px solid #e2e8f0;
}
.chat-input.center-mode {
  border-top: none;
  padding: 0;
  width: 100%;
  max-width: 720px;
  margin: 0 auto;
}

.input-wrap {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  padding: 6px 6px 6px 18px;
  transition: border-color 0.2s, box-shadow 0.2s;
  max-width: 820px;
  margin: 0 auto;
}
.input-wrap:focus-within {
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,0.08);
  background: #fff;
}
.input-wrap.disabled {
  opacity: 0.6;
}
textarea {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 15px;
  font-family: inherit;
  line-height: 1.5;
  resize: none;
  outline: none;
  padding: 9px 0;
  color: #1e293b;
  max-height: 160px;
  min-height: 24px;
}
textarea::placeholder { color: #94a3b8; }
textarea:disabled { cursor: not-allowed; }
.send-btn {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  border: none;
  background: #6366f1;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.15s, transform 0.15s;
}
.send-btn:hover:not(:disabled) { background: #4f46e5; transform: scale(1.05); }
.send-btn:disabled {
  background: #cbd5e1;
  cursor: not-allowed;
}
.input-hint {
  text-align: center;
  font-size: 11px;
  color: #94a3b8;
  margin-top: 6px;
  max-width: 820px;
  margin-left: auto;
  margin-right: auto;
}
</style>
