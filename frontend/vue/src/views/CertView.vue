<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listCerts,
  requestCert,
  renewCert,
  deleteCert,
  deployCert,
  pasteCert,
  getCertProgress,
} from '@/api/modules/certs'
import type { CertInfo, CertProgress } from '@/types/cert'

const certs = ref<CertInfo[]>([])
const loading = ref(false)

// 申请对话框
const requestVisible = ref(false)
const requestForm = ref({ domain: '', email: '', challenge_method: 'http-auto' })

// 粘贴证书对话框
const pasteVisible = ref(false)
const pasteForm = ref({ domain: '', cert_pem: '', key_pem: '' })

// 进度
const progressVisible = ref(false)
const progressData = ref<CertProgress | null>(null)
const progressDomain = ref('')
let progressTimer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    certs.value = await listCerts()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载证书失败')
  } finally {
    loading.value = false
  }
}

async function handleRequest() {
  if (!requestForm.value.domain) {
    ElMessage.error('请输入域名')
    return
  }
  const { domain, email, challenge_method } = requestForm.value
  try {
    await requestCert(domain, email, challenge_method)
    ElMessage.success('证书申请已提交')
    requestVisible.value = false
    // http-file / dns 走异步流程，可查看实时进度
    if (challenge_method === 'http-file' || challenge_method === 'dns') {
      startProgressPoll(domain)
    } else {
      await load()
    }
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '申请失败')
  }
}

async function handleRenew(cert: CertInfo) {
  try {
    await renewCert(cert.domain)
    ElMessage.success('续期已触发')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '续期失败')
  }
}

async function handleDelete(cert: CertInfo) {
  await ElMessageBox.confirm(`确定删除证书 ${cert.domain}？`, '删除证书', { type: 'warning' })
  try {
    await deleteCert(cert.domain)
    ElMessage.success('证书已删除')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function handleDeploy(cert: CertInfo) {
  try {
    const res = await deployCert(cert.domain, [])
    ElMessage.success(`已部署到 ${res.deployed} 个站点`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '部署失败')
  }
}

async function handlePaste() {
  if (!pasteForm.value.domain || !pasteForm.value.cert_pem || !pasteForm.value.key_pem) {
    ElMessage.error('请填写完整')
    return
  }
  try {
    await pasteCert(pasteForm.value.domain, pasteForm.value.cert_pem, pasteForm.value.key_pem)
    ElMessage.success('证书已保存')
    pasteVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

// 进度轮询
function startProgressPoll(domain: string) {
  progressDomain.value = domain
  progressVisible.value = true
  const poll = async () => {
    try {
      const p = await getCertProgress(domain)
      progressData.value = p
      if (p && (p.status === 'success' || p.status === 'error')) {
        stopProgressPoll()
        await load()
      }
    } catch {
      stopProgressPoll()
    }
  }
  poll()
  progressTimer = setInterval(poll, 1500)
}

function stopProgressPoll() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

function daysLeftClass(days: number): string {
  if (days <= 0) return 'expired'
  if (days <= 7) return 'critical'
  if (days <= 30) return 'warning'
  return 'ok'
}

onMounted(load)
onUnmounted(stopProgressPoll)
</script>

<template>
  <div class="cert-view">
    <div class="toolbar">
      <el-button type="primary" @click="requestVisible = true">+ 申请证书</el-button>
      <el-button @click="pasteVisible = true">粘贴证书</el-button>
      <el-button :icon="'Refresh'" circle @click="load" />
    </div>

    <el-table v-loading="loading" :data="certs" stripe>
      <el-table-column label="域名" prop="domain" min-width="180" />
      <el-table-column label="颁发者" width="140">
        <template #default="{ row }">{{ row.issuer || '—' }}</template>
      </el-table-column>
      <el-table-column label="到期" width="120">
        <template #default="{ row }">
          <span v-if="row.days_left >= 0" :class="['days-left', daysLeftClass(row.days_left)]">
            {{ row.days_left }} 天
          </span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="row.type === 'auto' ? 'success' : row.type === 'custom' ? 'warning' : 'info'">
            {{ row.type === 'auto' ? '自动' : row.type === 'custom' ? '自定义' : '自签' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="有效期" min-width="200">
        <template #default="{ row }">
          <span v-if="row.not_after" class="muted">
            {{ row.not_before?.slice(0, 10) }} ~ {{ row.not_after?.slice(0, 10) }}
          </span>
          <span v-else class="muted">未签发</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link size="small" @click="handleDeploy(row)">部署</el-button>
          <el-button link size="small" type="success" @click="handleRenew(row)">续期</el-button>
          <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="暂无证书" />
      </template>
    </el-table>

    <!-- 申请对话框 -->
    <el-dialog v-model="requestVisible" title="申请证书（Let's Encrypt）" width="480px">
      <el-form :model="requestForm" label-width="90px">
        <el-form-item label="域名" required>
          <el-input v-model="requestForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="requestForm.email" placeholder="you@example.com" />
        </el-form-item>
        <el-form-item label="验证方式">
          <el-select v-model="requestForm.challenge_method">
            <el-option value="http-auto" label="HTTP 自动（推荐）" />
            <el-option value="http-file" label="HTTP 文件验证" />
            <el-option value="dns" label="DNS 验证" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="requestVisible = false">取消</el-button>
        <el-button type="primary" @click="handleRequest">申请</el-button>
      </template>
    </el-dialog>

    <!-- 粘贴证书 -->
    <el-dialog v-model="pasteVisible" title="粘贴证书" width="600px">
      <el-form :model="pasteForm" label-width="90px">
        <el-form-item label="域名" required>
          <el-input v-model="pasteForm.domain" />
        </el-form-item>
        <el-form-item label="证书 PEM" required>
          <el-input v-model="pasteForm.cert_pem" type="textarea" :rows="6" placeholder="-----BEGIN CERTIFICATE-----" />
        </el-form-item>
        <el-form-item label="私钥 PEM" required>
          <el-input v-model="pasteForm.key_pem" type="textarea" :rows="6" placeholder="-----BEGIN PRIVATE KEY-----" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pasteVisible = false">取消</el-button>
        <el-button type="primary" @click="handlePaste">保存</el-button>
      </template>
    </el-dialog>

    <!-- 进度对话框 -->
    <el-dialog v-model="progressVisible" title="证书申请进度" width="640px" @close="stopProgressPoll">
      <div v-if="progressData" class="progress-content">
        <el-alert :type="progressData.status === 'error' ? 'error' : progressData.status === 'success' ? 'success' : 'info'" :closable="false" show-icon>
          {{ progressData.step_text }}（{{ progressData.status }}）
        </el-alert>
        <div class="progress-logs">
          <div v-for="(log, i) in progressData.logs" :key="i" :class="['log-line', log.level]">
            <span class="log-time">{{ log.time }}</span>
            <span>{{ log.message }}</span>
          </div>
        </div>
      </div>
      <el-empty v-else description="等待进度数据..." />
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.cert-view {
  max-width: 1200px;
}

.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.days-left {
  font-weight: 600;
  &.ok { color: var(--el-color-success); }
  &.warning { color: var(--el-color-warning); }
  &.critical { color: var(--el-color-danger); }
  &.expired { color: var(--el-text-color-placeholder); }
}

.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.progress-content {
  .progress-logs {
    margin-top: 12px;
    max-height: 300px;
    overflow-y: auto;
    background: var(--el-bg-color-page);
    padding: 12px;
    border-radius: 8px;
    font-family: monospace;
    font-size: 12px;

    .log-line {
      padding: 2px 0;
      &.error { color: var(--el-color-danger); }
      &.success { color: var(--el-color-success); }
      &.warn { color: var(--el-color-warning); }

      .log-time {
        color: var(--el-text-color-secondary);
        margin-right: 8px;
      }
    }
  }
}
</style>
