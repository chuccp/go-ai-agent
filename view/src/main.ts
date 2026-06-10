import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import ChatHome from './components/ChatHome.vue'
import FlowDesigner from './components/FlowDesigner/FlowDesigner.vue'
import ModelManager from './components/ModelManager.vue'
import SetupWizard from './components/SetupWizard.vue'

const routes = [
  { path: '/', component: ChatHome },
  { path: '/designer', component: FlowDesigner },
  { path: '/designer/:id', component: FlowDesigner },
  { path: '/models', component: ModelManager },
  { path: '/setup', component: SetupWizard },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
