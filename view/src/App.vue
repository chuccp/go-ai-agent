<template>
  <div v-if="checking" class="app-loading">
    <div class="spinner"></div>
  </div>
  <router-view v-else />
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const checking = ref(true)

onMounted(async () => {
  try {
    const res = await fetch('/api/setup/status')
    if (res.ok) {
      const data = await res.json()
      const status = data.data
      if (!status.initialized && route.path !== '/setup') {
        router.push('/setup')
      }
    }
  } catch {
    // Server may not be ready yet — stay on current page
  } finally {
    checking.value = false
  }
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.app-loading {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #e2e8f0;
  border-top: 3px solid #6366f1;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
