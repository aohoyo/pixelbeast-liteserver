<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { listLogs, readLogs, getLogsStats, clearLog } from '@/api/modules/logs'
import type { LogFileInfo, LogEntry, LogStats } from '@/types/log'

const files = ref<LogFileInfo[]>([])
const entries = ref<LogEntry[]>([])
const stats = ref<LogStats | null>(null)
const loading = ref(false)

const currentCategory = ref('panel')
const currentType = ref('')
const searchKeyword = ref('')
const levelFilter = ref('')
const total = ref(0)

const categories = [
  { label: '面板日志', value: 'panel' },
  { label: 'HTTP 日志', value: 'http' },
  { label: 'FTP 日志', value: 'ftp' },
]

const levelColors: Record<string, string> = {
  error: 'danger',
  warn: 'warning',
  info: 'info',
  auth: 'success',
  debug: '',
}

async function loadFiles() {
  try {
    files.value = await listLogs()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载日志列表失败')
  }
}

async function loadContent() {
  loading.value = true
  try {
    const params = {
      category: currentCategory.value,
      type: currentType.value || undefined,
      search: searchKeyword.value || undefined,
      level: levelFilter.value || undefined,
      limit: 500,
    }
    const [readRes, statsRes] = await Promise.all([readLogs(params), getLogsStats(params)])
    entries.value = readRes.entries
    total.value = readRes.total
    stats.value = statsRes[0] ?? null
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载日志内容失败')
  } finally {
    loading.value = false
  }
}

async function handleClear() {
  try {
    await clearLog(currentCategory.value, currentType.value)
    ElMessage.success('日志已清空')
    await loadContent()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '清空失败')
  }
}

function selectType(type: string) {
  currentType.value = currentType.value === type ? '' : type
}

watch([currentCategory, currentType, searchKeyword, levelFilter], () => {
  loadContent()
})

onMounted(() => {
  loadFiles()
  loadContent()
})
</script>

<template>
  <div class="logs-view">
    <!-- 类别 + 过滤 -->
    <div class="filter-bar">
      <el-radio-group v-model="currentCategory" size="small">
        <el-radio-button v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</el-radio-button>
      </el-radio-group>

      <el-select v-model="levelFilter" placeholder="日志级别" clearable size="small" style="width: 120px">
        <el-option value="debug" label="Debug" />
        <el-option value="info" label="Info" />
        <el-option value="warn" label="Warn" />
        <el-option value="error" label="Error" />
        <el-option value="auth" label="Auth" />
      </el-select>

      <el-input v-model="searchKeyword" placeholder="搜索..." clearable size="small" style="width: 200px" />

      <el-button size="small" :icon="'Refresh'" circle @click="loadContent" />
      <el-button size="small" type="danger" plain @click="handleClear">清空当前</el-button>
    </div>

    <!-- 统计 -->
    <div v-if="stats" class="stats-bar">
      <el-tag size="small">总计 {{ stats.count }}</el-tag>
      <el-tag v-if="stats.errors" size="small" type="danger">错误 {{ stats.errors }}</el-tag>
      <el-tag v-if="stats.warnings" size="small" type="warning">警告 {{ stats.warnings }}</el-tag>
      <span class="muted">显示 {{ entries.length }} / {{ total }} 条</span>
    </div>

    <!-- 类型切换（access / error） -->
    <div class="type-tabs" v-if="currentCategory !== 'panel'">
      <el-button size="small" :type="currentType === '' ? 'primary' : ''" @click="selectType('')">访问</el-button>
      <el-button size="small" :type="currentType === 'error' ? 'primary' : ''" @click="selectType('error')">错误</el-button>
    </div>

    <!-- 日志列表 -->
    <div class="log-list" v-loading="loading">
      <div v-for="(entry, i) in entries" :key="i" class="log-line">
        <span class="log-time">{{ entry.timestamp }}</span>
        <el-tag v-if="entry.level" :type="(levelColors[entry.level] as any)" size="small" class="log-level">
          {{ entry.level.toUpperCase() }}
        </el-tag>
        <span class="log-msg">{{ entry.message }}</span>
      </div>
      <el-empty v-if="!loading && !entries.length" description="无日志" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.logs-view {
  max-width: 1200px;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.stats-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .muted {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.type-tabs {
  margin-bottom: 12px;
}

.log-list {
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 12px;
  max-height: 60vh;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
}

.log-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 2px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  .log-time {
    color: var(--el-text-color-secondary);
    flex-shrink: 0;
    white-space: nowrap;
  }

  .log-level {
    flex-shrink: 0;
  }

  .log-msg {
    word-break: break-all;
  }
}
</style>
