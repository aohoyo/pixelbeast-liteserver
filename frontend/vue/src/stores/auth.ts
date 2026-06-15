import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login as loginApi, logout as logoutApi, initCSRF } from '@/api/modules/auth'
import { setUnauthorizedHandler } from '@/api/client'

export const useAuthStore = defineStore('auth', () => {
  const isLoggedIn = ref(false)
  const username = ref('')
  const loading = ref(false)

  /** 登录：成功后初始化 CSRF token */
  async function login(user: string, password: string) {
    loading.value = true
    try {
      await loginApi(user, password)
      // 登录成功后拉取首个 CSRF token（后端在 /api/system/status 下发）
      await initCSRF()
      isLoggedIn.value = true
      username.value = user
    } finally {
      loading.value = false
    }
  }

  /** 登出 */
  async function logout() {
    try {
      await logoutApi()
    } finally {
      isLoggedIn.value = false
      username.value = ''
    }
  }

  /** 认证失效（401）时由 API 客户端回调 */
  function handleUnauthorized() {
    isLoggedIn.value = false
    username.value = ''
  }

  return { isLoggedIn, username, loading, login, logout, handleUnauthorized }
})

// 注册认证失效回调（模块加载时一次性注册）
let handlerRegistered = false
export function ensureAuthHandler() {
  if (handlerRegistered) return
  handlerRegistered = true
  setUnauthorizedHandler(() => {
    const store = useAuthStore()
    store.handleUnauthorized()
  })
}
