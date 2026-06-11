<template>
  <div class="node-config">
    <label>提示文本</label>
    <textarea v-model="cfg.prompt" @input="emitUpdate" rows="3" placeholder="提示用户的文本..."></textarea>
    <label>
      <input type="checkbox" v-model="cfg.confirm_only" @change="emitUpdate" />
      仅确认（无需输入文本）
    </label>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
const props = defineProps<{ config: Record<string, any> }>()
const emit = defineEmits<{ update: [config: Record<string, any>] }>()

const cfg = reactive({ prompt: '', confirm_only: false })

watch(() => props.config, (c) => {
  cfg.prompt = c.prompt || ''
  cfg.confirm_only = c.confirm_only || false
}, { immediate: true })

function emitUpdate() { emit('update', { ...cfg }) }
</script>
