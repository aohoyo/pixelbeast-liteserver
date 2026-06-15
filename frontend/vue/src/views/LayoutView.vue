<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

interface NavItem {
  index: string
  label: string
  icon: string
  route?: string // 路由名，未迁移的模块无 route（仅展示占位）
}

// 导航项：全部模块已迁移
const navItems: NavItem[] = [
  { index: 'home', label: '首页', icon: '🏠', route: 'home' },
  { index: 'sites', label: '站点管理', icon: '🌐', route: 'sites' },
  { index: 'ftp', label: 'FTP', icon: '📁', route: 'ftp' },
  { index: 'files', label: '文件管理', icon: '📂', route: 'files' },
  { index: 'terminal', label: '终端', icon: '💻', route: 'terminal' },
  { index: 'cert', label: 'SSL 证书', icon: '🔒', route: 'cert' },
  { index: 'logs', label: '日志', icon: '📜', route: 'logs' },
  { index: 'settings', label: '设置', icon: '⚙️', route: 'settings' },
]

function handleSelect(index: string) {
  const item = navItems.find((n) => n.index === index)
  if (item?.route) {
    router.push({ name: item.route })
  } else if (item) {
    import('element-plus').then(({ ElMessage }) => {
      ElMessage.info(`「${item.label}」开发中（Vue 迁移进行时）`)
    })
  }
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

// 当前激活项
const activeIndex = computed(() => String(route.name ?? 'home'))
</script>

<template>
  <div class="layout">
    <!-- 侧边栏 -->
    <aside class="layout-sidebar">
      <div class="brand">
        <span class="logo">🦖</span>
        <span class="name">像素兽</span>
      </div>
      <el-menu
        :default-active="activeIndex"
        class="sidebar-menu"
        @select="handleSelect"
      >
        <el-menu-item v-for="item in navItems" :key="item.index" :index="item.index">
          <span class="nav-icon">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
    </aside>

    <!-- 主体 -->
    <div class="layout-main">
      <header class="layout-header">
        <div class="header-title">PixelBeast 管理面板</div>
        <div class="header-actions">
          <span v-if="auth.username" class="user">{{ auth.username }}</span>
          <el-button size="small" @click="handleLogout">登出</el-button>
        </div>
      </header>
      <main class="layout-body">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped lang="scss">
.layout {
  display: flex;
  min-height: 100vh;
}

.layout-sidebar {
  width: 200px;
  flex-shrink: 0;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color);
  display: flex;
  flex-direction: column;

  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 16px 20px;
    height: 56px;
    border-bottom: 1px solid var(--el-border-color);

    .logo {
      font-size: 22px;
    }
    .name {
      font-weight: 700;
      color: var(--el-color-primary);
      font-size: 16px;
    }
  }

  .sidebar-menu {
    border-right: none;
    flex: 1;
  }

  .nav-icon {
    margin-right: 8px;
  }
}

.layout-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.layout-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color);

  .header-title {
    font-weight: 600;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;

    .user {
      color: var(--el-text-color-secondary);
      font-size: 13px;
    }
  }
}

.layout-body {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}
</style>
