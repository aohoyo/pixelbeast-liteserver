<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { getSystemStatus, freeMemory } from '@/api/modules/system'
import { listSites } from '@/api/modules/sites'
import { getFtpStatus } from '@/api/modules/ftp'
import type { SystemStatus } from '@/types/system'
import type { Site } from '@/types/site'
import { ElMessage } from 'element-plus'

const router = useRouter()
const status = ref<SystemStatus | null>(null)
const sites = ref<Site[]>([])
const ftpRunning = ref(false)
const loading = ref(true)
const cpuHistory = ref<number[]>([])
let pollTimer: ReturnType<typeof setInterval> | null = null

async function loadAll() {
  try {
    const [sys, sitesRes, ftpRes] = await Promise.all([
      getSystemStatus(),
      listSites().catch(() => [] as Site[]),
      getFtpStatus().catch(() => ({ running: false })),
    ])
    status.value = sys
    sites.value = sitesRes
    ftpRunning.value = ftpRes.running
    cpuHistory.value = [...sys.cpu_history, sys.cpu_percent].slice(-10)
  } catch {
    // 忽略轮询错误
  } finally {
    loading.value = false
  }
}

async function handleFreeMemory() {
  try {
    const res = await freeMemory()
    const freed = (res as { freed_mb?: number }).freed_mb
    ElMessage.success(freed ? `已释放 ${freed.toFixed(1)} MB 内存` : '内存释放完成')
    await loadAll()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '释放失败')
  }
}

function formatUptime(ms: number): string {
  if (!ms) return '—'
  const sec = Math.floor(ms / 1000)
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (d > 0) return `${d}天 ${h}小时`
  if (h > 0) return `${h}小时 ${m}分钟`
  return `${m}分钟`
}

onMounted(() => {
  loadAll()
  pollTimer = setInterval(loadAll, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="home-view" v-loading="loading">
    <h2 class="page-title">系统概览</h2>

    <!-- 服务状态卡片 -->
    <div class="cards-row">
      <div class="card clickable" @click="router.push('/sites')">
        <div class="card-icon">🌐</div>
        <div class="card-body">
          <div class="card-value">{{ sites.filter((s) => s.enabled).length }} / {{ sites.length }}</div>
          <div class="card-label">站点（启用/总数）</div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon">📁</div>
        <div class="card-body">
          <div class="card-value" :class="ftpRunning ? 'text-success' : 'text-muted'">
            {{ ftpRunning ? '运行' : '停止' }}
          </div>
          <div class="card-label">FTP 服务</div>
        </div>
      </div>

      <div class="card" @click="router.push('/terminal')">
        <div class="card-icon">💻</div>
        <div class="card-body">
          <div class="card-value">{{ status?.hostname || '—' }}</div>
          <div class="card-label">{{ status?.os_name_short || '' }}</div>
        </div>
      </div>
    </div>

    <!-- 监控指标 -->
    <div class="metrics-grid" v-if="status">
      <!-- CPU -->
      <div class="metric-card">
        <div class="metric-header">
          <span class="metric-title">CPU</span>
          <span class="metric-value">{{ status.cpu_percent.toFixed(1) }}%</span>
        </div>
        <el-progress :percentage="Math.min(status.cpu_percent, 100)" :color="'#f97316'" :show-text="false" :stroke-width="8" />
        <div class="metric-detail">
          {{ status.cpu_model }} · {{ status.cpu_cores }} 核 / {{ status.cpu_threads }} 线程
        </div>
      </div>

      <!-- 内存 -->
      <div class="metric-card">
        <div class="metric-header">
          <span class="metric-title">内存</span>
          <span class="metric-value">{{ status.memory_percent.toFixed(1) }}%</span>
        </div>
        <el-progress :percentage="Math.min(status.memory_percent, 100)" :color="'#10b981'" :show-text="false" :stroke-width="8" />
        <div class="metric-detail">
          {{ status.memory_used_gb.toFixed(1) }} / {{ status.memory_total_gb.toFixed(1) }} GB
          <el-button link size="small" type="primary" @click="handleFreeMemory">释放</el-button>
        </div>
      </div>

      <!-- 磁盘 -->
      <div class="metric-card">
        <div class="metric-header">
          <span class="metric-title">磁盘</span>
          <span class="metric-value">{{ status.disk_percent.toFixed(1) }}%</span>
        </div>
        <el-progress :percentage="Math.min(status.disk_percent, 100)" :color="'#3b82f6'" :show-text="false" :stroke-width="8" />
        <div class="metric-detail">
          {{ status.disk_used_gb.toFixed(1) }} / {{ status.disk_total_gb.toFixed(1) }} GB · {{ status.disk_mount }}
        </div>
      </div>

      <!-- 网络 -->
      <div class="metric-card">
        <div class="metric-header">
          <span class="metric-title">网络</span>
        </div>
        <div class="metric-row">
          <span>↑ {{ status.net_sent_rate_kb.toFixed(1) }} KB/s</span>
          <span>↓ {{ status.net_recv_rate_kb.toFixed(1) }} KB/s</span>
        </div>
        <div class="metric-detail">
          累计 ↑ {{ status.net_total_sent_gb.toFixed(2) }} GB · ↓ {{ status.net_total_recv_gb.toFixed(2) }} GB
        </div>
      </div>

      <!-- 系统信息 -->
      <div class="metric-card info-card">
        <div class="metric-header">
          <span class="metric-title">系统信息</span>
        </div>
        <div class="info-list">
          <div class="info-item"><span class="info-label">运行时长</span><span>{{ formatUptime(status.system_uptime_ms) }}</span></div>
          <div class="info-item"><span class="info-label">进程数</span><span>{{ status.process_total }}</span></div>
          <div class="info-item"><span class="info-label">负载</span><span>{{ status.load_avg?.map((v) => v.toFixed(2)).join(' / ') || '—' }}</span></div>
          <div class="info-item"><span class="info-label">内核</span><span class="truncate">{{ status.kernel }}</span></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.home-view {
  max-width: 1200px;
}

.page-title {
  margin: 0 0 20px;
  font-size: 20px;
}

.cards-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 10px;

  &.clickable {
    cursor: pointer;
    transition: border-color 0.2s;
    &:hover { border-color: var(--el-color-primary); }
  }

  .card-icon { font-size: 32px; }
  .card-value { font-size: 22px; font-weight: 700; color: var(--el-color-primary); }
  .card-label { font-size: 13px; color: var(--el-text-color-secondary); margin-top: 2px; }
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.metric-card {
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 10px;

  .metric-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .metric-title { font-weight: 600; }
    .metric-value { font-size: 18px; font-weight: 700; color: var(--el-color-primary); }
  }

  .metric-detail {
    margin-top: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .metric-row {
    display: flex;
    justify-content: space-between;
    font-size: 14px;
    font-weight: 500;
    margin: 4px 0;
  }
}

.info-card .info-list {
  display: flex;
  flex-direction: column;
  gap: 6px;

  .info-item {
    display: flex;
    justify-content: space-between;
    font-size: 13px;

    .info-label { color: var(--el-text-color-secondary); }
    .truncate {
      max-width: 180px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}

.text-success { color: var(--el-color-success) !important; }
.text-muted { color: var(--el-text-color-placeholder) !important; }
</style>
