<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listSites,
  createSite,
  updateSite,
  deleteSite,
  toggleSite,
  startSite,
  stopSite,
  restartSite,
  batchSites,
} from '@/api/modules/sites'
import type { Site } from '@/types/site'
import SiteEditDialog from '@/components/SiteEditDialog.vue'

const sites = ref<Site[]>([])
const loading = ref(false)
const selection = ref<Site[]>([])

// 编辑对话框
const dialogVisible = ref(false)
const editingSite = ref<Site | null>(null)

async function load() {
  loading.value = true
  try {
    sites.value = await listSites()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载站点失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingSite.value = null
  dialogVisible.value = true
}

function openEdit(site: Site) {
  editingSite.value = site
  dialogVisible.value = true
}

async function handleSubmit(data: Partial<Site>) {
  try {
    if (editingSite.value) {
      await updateSite(editingSite.value.id, data)
      ElMessage.success('站点已更新')
    } else {
      await createSite(data)
      ElMessage.success('站点已创建')
    }
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function handleDelete(site: Site) {
  await ElMessageBox.confirm(`确定删除站点「${site.name}」？此操作不可恢复。`, '删除站点', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消',
  })
  try {
    await deleteSite(site.id)
    ElMessage.success('站点已删除')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function handleToggle(site: Site, enabled: boolean) {
  try {
    await toggleSite(site.id, enabled)
    site.enabled = enabled
  } catch (e) {
    // 失败时回滚开关状态
    site.enabled = !enabled
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

// 单站点启停重启（运行时控制，不影响 enabled 配置）
async function handleRuntimeAction(site: Site, action: 'start' | 'stop' | 'restart') {
  const fn = action === 'start' ? startSite : action === 'stop' ? stopSite : restartSite
  const label = action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'
  try {
    await fn(site.id)
    ElMessage.success(`站点已${label}`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : `${label}失败`)
  }
}

async function handleBatch(action: 'enable' | 'disable' | 'delete') {
  if (selection.value.length === 0) {
    ElMessage.warning('请先选择站点')
    return
  }
  const ids = selection.value.map((s) => s.id)
  const label = action === 'enable' ? '启用' : action === 'disable' ? '禁用' : '删除'
  if (action === 'delete') {
    await ElMessageBox.confirm(`确定批量${label} ${ids.length} 个站点？`, '批量操作', {
      type: 'warning',
    })
  }
  try {
    await batchSites(action, ids)
    ElMessage.success(`已${label} ${ids.length} 个站点`)
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '批量操作失败')
  }
}

function handleSelectionChange(rows: Site[]) {
  selection.value = rows
}

onMounted(load)
</script>

<template>
  <div class="sites-view">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="left">
        <el-button type="primary" @click="openCreate">+ 添加网站</el-button>
        <el-button :disabled="!selection.length" @click="handleBatch('enable')">启用</el-button>
        <el-button :disabled="!selection.length" @click="handleBatch('disable')">禁用</el-button>
        <el-button type="danger" :disabled="!selection.length" @click="handleBatch('delete')">
          删除
        </el-button>
      </div>
      <div class="right">
        <el-button :icon="'Refresh'" circle @click="load" />
      </div>
    </div>

    <!-- 表格 -->
    <el-table
      v-loading="loading"
      :data="sites"
      @selection-change="handleSelectionChange"
      row-key="id"
      stripe
    >
      <el-table-column type="selection" width="44" />
      <el-table-column label="站点名称" min-width="180">
        <template #default="{ row }">
          <div class="name-cell">
            <span class="name">{{ row.name }}</span>
            <el-tag size="small" :type="row.type === 'static' ? 'info' : 'warning'">
              {{ row.type === 'static' ? '静态' : '代理' }}
            </el-tag>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-switch
            :model-value="row.enabled"
            @change="(v: boolean) => handleToggle(row, v)"
          />
        </template>
      </el-table-column>

      <el-table-column label="域名" min-width="160">
        <template #default="{ row }">
          <span v-if="row.domain?.length">{{ row.domain.join(', ') }}</span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>

      <el-table-column label="端口" width="90">
        <template #default="{ row }">{{ row.port || '共享' }}</template>
      </el-table-column>

      <el-table-column label="SSL" width="70">
        <template #default="{ row }">
          <el-tag v-if="row.ssl?.enabled" type="success" size="small">开</el-tag>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button link size="small" @click="openEdit(row)">编辑</el-button>
          <el-button link size="small" type="success" @click="handleRuntimeAction(row, 'start')">
            启动
          </el-button>
          <el-button link size="small" type="warning" @click="handleRuntimeAction(row, 'stop')">
            停止
          </el-button>
          <el-button link size="small" @click="handleRuntimeAction(row, 'restart')">重启</el-button>
          <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>

      <template #empty>
        <el-empty description="暂无站点">
          <el-button type="primary" @click="openCreate">添加网站</el-button>
        </el-empty>
      </template>
    </el-table>

    <SiteEditDialog
      v-model="dialogVisible"
      :site="editingSite"
      @submit="handleSubmit"
    />
  </div>
</template>

<style scoped lang="scss">
.sites-view {
  max-width: 1200px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  .left {
    display: flex;
    gap: 8px;
  }
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;

  .name {
    font-weight: 500;
  }
}

.muted {
  color: var(--el-text-color-placeholder);
}
</style>
