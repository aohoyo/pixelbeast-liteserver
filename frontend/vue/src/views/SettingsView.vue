<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getConfig, saveConfig, resetConfig, getLogConfig, saveLogConfig } from '@/api/modules/config'
import { listBackups, createBackup, deleteBackup, restoreBackup, downloadBackupUrl } from '@/api/modules/backup'
import type { ServerConfig } from '@/types/config'
import type { BackupInfo } from '@/types/backup'

const activeTab = ref('general')
const loading = ref(false)
const saving = ref(false)

// 深拷贝的配置（独立编辑，不直接改原对象）
const form = reactive<ServerConfig>({} as ServerConfig)
const original = ref<ServerConfig | null>(null)

async function load() {
  loading.value = true
  try {
    const cfg = await getConfig()
    original.value = JSON.parse(JSON.stringify(cfg))
    Object.assign(form, cfg)
    // 备份项同步
    backupItems.value = [...form.backup.items]
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载配置失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    form.backup.items = backupItems.value.filter((v) => v)
    await saveConfig(form)
    ElMessage.success('配置已保存')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleReset() {
  await ElMessageBox.confirm('重置将恢复默认服务配置（保留站点和 FTP 数据），并生成新的随机管理员密码。确定继续？', '重置配置', { type: 'warning' })
  try {
    const cfg = await resetConfig()
    Object.assign(form, cfg)
    ElMessage.success('配置已重置，请查看终端获取新密码')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '重置失败')
  }
}

// 备份管理
const backups = ref<BackupInfo[]>([])
const backupDir = ref('')
const backupLoading = ref(false)
const backupItems = ref<string[]>(['config', 'sites', 'ftp'])

async function loadBackups() {
  backupLoading.value = true
  try {
    const res = await listBackups()
    backups.value = res.backups
    backupDir.value = res.dir
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载备份失败')
  } finally {
    backupLoading.value = false
  }
}

async function handleCreateBackup() {
  try {
    await createBackup()
    ElMessage.success('备份创建成功')
    await loadBackups()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  }
}

async function handleDeleteBackup(name: string) {
  await ElMessageBox.confirm(`确定删除备份 ${name}？`, '删除', { type: 'warning' })
  try {
    await deleteBackup(name)
    ElMessage.success('已删除')
    await loadBackups()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function handleRestoreBackup(name: string) {
  await ElMessageBox.confirm(`确定从备份 ${name} 恢复？当前配置将被覆盖。`, '恢复备份', { type: 'warning' })
  try {
    await restoreBackup(name)
    ElMessage.success('恢复成功，重新加载配置生效')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '恢复失败')
  }
}

// 日志配置
const logForm = reactive({ retention_days: 30, max_size_mb: 100, compress_days: 7, level: 'info' })

async function loadLogConfig() {
  try {
    const cfg = await getLogConfig()
    Object.assign(logForm, cfg)
  } catch {
    // 忽略
  }
}

async function saveLogSettings() {
  try {
    await saveLogConfig(logForm)
    ElMessage.success('日志配置已保存')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

onMounted(async () => {
  await load()
  await loadLogConfig()
})
</script>

<template>
  <div class="settings-view" v-loading="loading">
    <el-tabs v-model="activeTab" @tab-change="(name: string) => name === 'backup' && loadBackups()">
      <!-- 常规 -->
      <el-tab-pane label="常规" name="general">
        <el-form :model="form" label-width="130px" class="settings-form">
          <el-form-item label="服务器名称">
            <el-input v-model="form.name" />
          </el-form-item>
          <el-form-item label="时区">
            <el-input v-model="form.timezone" placeholder="Asia/Shanghai" />
          </el-form-item>
          <el-divider content-position="left">管理面板</el-divider>
          <el-form-item label="端口">
            <el-input-number v-model="form.admin.port" :min="1" :max="65535" controls-position="right" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="form.admin.username" />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="form.admin.password" type="password" show-password placeholder="留空不修改" />
          </el-form-item>
          <el-form-item label="安全入口路径">
            <el-input v-model="form.admin.path" placeholder="/admin" />
          </el-form-item>
          <el-form-item label="绑定域名">
            <el-input v-model="form.admin.domain" placeholder="留空允许所有域名" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 目录 -->
      <el-tab-pane label="目录" name="directories">
        <el-form :model="form" label-width="130px" class="settings-form">
          <el-form-item label="站点目录">
            <el-input v-model="form.directories.sites" />
          </el-form-item>
          <el-form-item label="FTP 目录">
            <el-input v-model="form.directories.ftp" />
          </el-form-item>
          <el-form-item label="备份目录">
            <el-input v-model="form.directories.backup" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 日志 -->
      <el-tab-pane label="日志" name="log">
        <el-form :model="logForm" label-width="130px" class="settings-form">
          <el-form-item label="保留天数">
            <el-input-number v-model="logForm.retention_days" :min="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="单文件上限 (MB)">
            <el-input-number v-model="logForm.max_size_mb" :min="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="压缩天数">
            <el-input-number v-model="logForm.compress_days" :min="0" controls-position="right" />
          </el-form-item>
          <el-form-item label="日志级别">
            <el-select v-model="logForm.level">
              <el-option value="debug" label="Debug" />
              <el-option value="info" label="Info" />
              <el-option value="warn" label="Warn" />
              <el-option value="error" label="Error" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="saveLogSettings">保存日志配置</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 备份 -->
      <el-tab-pane label="备份" name="backup">
        <div class="backup-section">
          <div class="backup-controls">
            <span class="label">自动备份</span>
            <el-switch v-model="form.backup.auto_enabled" />
            <el-select v-model="form.backup.schedule" style="width: 110px; margin-left: 12px">
              <el-option value="daily" label="每天" />
              <el-option value="weekly" label="每周" />
              <el-option value="monthly" label="每月" />
            </el-select>
            <span class="label" style="margin-left: 16px">保留份数</span>
            <el-input-number v-model="form.backup.retention" :min="1" :max="100" controls-position="right" style="width: 110px" />
          </div>
          <el-button type="primary" @click="handleCreateBackup">立即创建备份</el-button>
        </div>

        <el-table :data="backups" v-loading="backupLoading" stripe style="margin-top: 16px">
          <el-table-column label="文件名" prop="name" min-width="280" show-overflow-tooltip />
          <el-table-column label="大小" width="120">
            <template #default="{ row }">{{ (row.size / 1024 / 1024).toFixed(2) }} MB</template>
          </el-table-column>
          <el-table-column label="时间" prop="modified" width="180" />
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button link size="small" @click="handleRestoreBackup(row.name)">恢复</el-button>
              <el-link :href="downloadBackupUrl(row.name)" :underline="false" target="_blank" style="margin: 0 8px">
                <el-button link size="small">下载</el-button>
              </el-link>
              <el-button link size="small" type="danger" @click="handleDeleteBackup(row.name)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无备份" />
          </template>
        </el-table>
        <p v-if="backupDir" class="muted">备份目录：{{ backupDir }}</p>
      </el-tab-pane>
    </el-tabs>

    <div class="footer-actions">
      <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
      <el-button type="danger" plain @click="handleReset">重置为默认</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.settings-view {
  max-width: 800px;
}

.settings-form {
  max-width: 600px;
}

.backup-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;

  .backup-controls {
    display: flex;
    align-items: center;
    gap: 8px;

    .label {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }
}

.footer-actions {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color);
}

.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
