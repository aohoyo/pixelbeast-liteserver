import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'

import App from './App.vue'
import router from './router'
import { ensureAuthHandler } from './stores/auth'
import './styles/main.scss'

const app = createApp(App)

app.use(createPinia())
// 注册认证失效回调（在 router 之前，确保守卫能读到 store 状态）
ensureAuthHandler()
app.use(router)
app.use(ElementPlus)

app.mount('#app')
