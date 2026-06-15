<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listFtpUsers,
  addFtpUser,
  deleteFtpUser,
  toggleFtpUser,
  batchFtpUsers,
  getFtpStatus,
} from '@/api/modules/ftp'
import type { FtpUser, FtpStatus } from '@/types/ftp'
import FtpUserDialog from '@/components/FtpUserDialog.vue'

const users = ref<FtpUser[]>([])
const loading = ref(false)
const selection = ref<FtpUser[]>([])
const status = ref<FtpStatus>({ running: false, port: 2121 })

const dialogVisible = ref(false)
const editingUser = ref<FtpUser | null>(null)

async function load() {
  loading.value = true
  try {
    const [usersRes, statusRes] = await Promise.all([listFtpUsers(), getFtpStatus()])
    users.value = usersRes.users
    status.value = statusRes
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingUser.value = null
  dialogVisible.value = true
}

function openEdit(user: FtpUser) {
  editingUser.value = user
  dialogVisible.value = true
}

async function handleSubmit(data: { username: string; password?: string; rootPath?: string; quota?: number; remark?: string; expiryDays?: number }) {
  try {
    if (editingUser.value) {
      const update: Record<string, unknown> = {}
      if (data.password) update.password = data.password
      if (data.rootPath) update.rootPath = data.rootPath
      if (data.quota !== undefined) update.quota = data.quota
      if (data.remark !== undefined) update.remark = data.remark
      if (data.expiryDays !== undefined) update.expiryDays = data.expiryDays
      // 用 username 路径更新
      const { updateFtpUser } = await import('@/api/modules/ftp')
      await updateFtpUser(editingUser.value.username, update)
      ElMessage.success('用户已更新')
    } else {
      if (!data.password) {
        ElMessage.error('请输入密码')
        return
      }
      await addFtpUser({ username: data.username, password: data.password, rootPath: data.rootPath, quota: data.quota, remark: data.remark, expiryDays: data.expiryDays })
      ElMessage.success('用户已添加')
    }
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function handleDelete(user: FtpUser) {
  await ElMessageBox.confirm(`确定删除 FTP 用户「${user.username}」？`, '删除用户', { type: 'warning' })
  try {
    await deleteFtpUser(user.username)
    ElMessage.success('用户已删除')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function handleToggle(user: FtpUser, enabled: boolean) {
  try {
    await toggleFtpUser(user.username, enabled)
    user.status = enabled ? 'enabled' : 'disabled'
  } catch (e) {
    user.status = !enabled ? 'enabled' : 'disabled'
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

async function handleBatch(action: 'enable' | 'disable' | 'delete') {
  if (!selection.value.length) {
    ElMessage.warning('请先选择用户')
    return
  }
  const usernames = selection.value.map((u) => u.username)
  if (action === 'delete') {
    await ElMessageBox.confirm(`确定批量删除 ${usernames.length} 个用户？`, '批量操作', { type: 'warning' })
  }
  try {
    await batchFtpUsers(action, usernames)
    ElMessage.success('批量操作完成')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function formatSize(bytes: number): string {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function onSelectionChange(rows: FtpUser[]) {
  selection.value = rows
}

onMounted(load)
</script>

<template>
  <div class="ftp-view">
    <div class="status-bar">
      <el-tag :type="status.running ? 'success' : 'info'">
        FTP 服务：{{ status.running ? '运行中' : '已停止' }}
      </el-tag>
      <el-tag type="info">端口：{{ status.port }}</el-tag>
    </div>

    <div class="toolbar">
      <el-button type="primary" @click="openCreate">+ 添加用户</el-button>
      <el-button :disabled="!selection.length" @click="handleBatch('enable')">启用</el-button>
      <el-button :disabled="!selection.length" @click="handleBatch('disable')">禁用</el-button>
      <el-button type="danger" :disabled="!selection.length" @click="handleBatch('delete')">删除</el-button>
      <el-button :icon="'Refresh'" circle @click="load" />
    </div>

    <el-table v-loading="loading" :data="users" @selection-change="onSelectionChange" row-key="username" stripe>
      <el-table-column type="selection" width="44" />
      <el-table-column label="用户名" prop="username" min-width="120" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.status === 'enabled'" @change="(v: boolean) => handleToggle(row, v)" />
        </template>
      </el-table-column>
      <el-table-column label="根目录" prop="rootPath" min-width="180" show-overflow-tooltip />
      <el-table-column label="配额" width="100">
        <template #default="{ row }">
          {{ row.quota ? `${row.quota} MB` : '无限' }}
        </template>
      </el-table-column>
      <el-table-column label="已用" width="100">
        <template #default="{ row }">{{ formatSize(row.usedSpace) }}</template>
      </el-table-column>
      <el-table-column label="限速" width="90">
        <template #default="{ row }">
          {{ row.speedLimit ? `${row.speedLimit} KB/s` : '无限' }}
        </template>
      </el-table-column>
      <el-table-column label="过期" width="110">
        <template #default="{ row }">
          <span v-if="row.expiryDate">{{ row.expiryDate }}</span>
          <span v-else class="muted">永久</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link size="small" @click="openEdit(row)">编辑</el-button>
          <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="暂无 FTP 用户" />
      </template>
    </el-table>

    <FtpUserDialog v-model="dialogVisible" :user="editingUser" @submit="handleSubmit" />
  </div>
</template>

<style scoped lang="scss">
.ftp-view {
  max-width: 1200px;
}

.status-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.muted {
  color: var(--el-text-color-placeholder);
}
</style>
