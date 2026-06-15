import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

// 管理面板基础路径：与 vite.config.ts 的 base 和后端 adminPath 保持一致
const router = createRouter({
  history: createWebHistory('/admin/'),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/',
      component: () => import('@/views/LayoutView.vue'),
      children: [
        {
          path: '',
          name: 'home',
          component: () => import('@/views/HomeView.vue'),
        },
        {
          path: 'sites',
          name: 'sites',
          component: () => import('@/views/SitesView.vue'),
        },
        {
          path: 'ftp',
          name: 'ftp',
          component: () => import('@/views/FtpView.vue'),
        },
        {
          path: 'files',
          name: 'files',
          component: () => import('@/views/FilesView.vue'),
        },
        {
          path: 'terminal',
          name: 'terminal',
          component: () => import('@/views/TerminalView.vue'),
        },
        {
          path: 'cert',
          name: 'cert',
          component: () => import('@/views/CertView.vue'),
        },
        {
          path: 'logs',
          name: 'logs',
          component: () => import('@/views/LogsView.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
        },
      ],
    },
  ],
})

// 路由守卫：未登录跳转 /login
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.name !== 'login' && !auth.isLoggedIn) {
    return { name: 'login' }
  }
})

export default router
