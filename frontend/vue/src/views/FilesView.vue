<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listFiles,
  uploadToPath,
  mkdir,
  renameFile,
  deleteFile,
  compressFiles,
  extractFile,
  shareFile,
  touchFile,
} from '@/api/modules/files'
import type { FileEntry } from '@/types/file'
import FileEditor from '@/components/FileEditor.vue'

const currentPath = ref('.')
const files = ref<FileEntry[]>([])
const loading = ref(false)
const selection = ref<FileEntry[]>([])

// 编辑器
const editorVisible = ref(false)
const editingFile = ref<{ path: string; name: string } | null>(null)

// 上传
const uploadRef = ref<HTMLInputElement>()

// 面包屑
const breadcrumbs = computed(() => {
  const parts = currentPath.value.replace(/\\/g, '/').split('/').filter(Boolean)
  const crumbs: { name: string; path: string }[] = []
  let acc = ''
  for (const p of parts) {
    acc = acc ? `${acc}/${p}` : p
    crumbs.push({ name: p, path: acc })
  }
  return crumbs
})

async function load() {
  loading.value = true
  try {
    const res = await listFiles(currentPath.value)
    currentPath.value = res.path
    // 排序：目录优先，再按名称
    files.value = res.files.sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function navigateTo(path: string) {
  currentPath.value = path
  load()
}

function enterDir(entry: FileEntry) {
  if (entry.is_dir) {
    currentPath.value = `${currentPath.value}/${entry.name}`.replace(/\/+/g, '/')
    load()
  }
}

// 上传
function triggerUpload() {
  uploadRef.value?.click()
}

async function handleUploadChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  const fileList = Array.from(input.files)
  input.value = '' // 重置以便重复选择
  for (const file of fileList) {
    try {
      await uploadToPath(file, currentPath.value)
      ElMessage.success(`${file.name} 上传成功`)
    } catch (err) {
      ElMessage.error(`${file.name} 上传失败：${err instanceof Error ? err.message : ''}`)
    }
  }
  await load()
}

// 新建目录
async function handleMkdir() {
  const result = await ElMessageBox.prompt('目录名称', '新建目录', {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
  }).catch(() => null)
  if (!result?.value) return
  try {
    await mkdir(`${currentPath.value}/${result.value}`.replace(/\/+/g, '/'))
    ElMessage.success('目录已创建')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  }
}

// 新建文件
async function handleTouch() {
  const result = await ElMessageBox.prompt('文件名称', '新建文件', {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
  }).catch(() => null)
  if (!result?.value) return
  try {
    await touchFile(currentPath.value, result.value)
    ElMessage.success('文件已创建')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  }
}

// 重命名
async function handleRename(entry: FileEntry) {
  const result = await ElMessageBox.prompt('新名称', '重命名', {
    inputValue: entry.name,
    confirmButtonText: '确定',
    cancelButtonText: '取消',
  }).catch(() => null)
  if (!result?.value || result.value === entry.name) return
  try {
    await renameFile(currentPath.value, entry.name, result.value)
    ElMessage.success('重命名成功')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '重命名失败')
  }
}

// 删除（移入回收站）
async function handleDelete(entry: FileEntry) {
  await ElMessageBox.confirm(`确定删除「${entry.name}」？（移入回收站）`, '删除', { type: 'warning' })
  try {
    await deleteFile(currentPath.value, entry.name)
    ElMessage.success('已移入回收站')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

// 批量删除
async function handleBatchDelete() {
  if (!selection.value.length) {
    ElMessage.warning('请先选择文件')
    return
  }
  await ElMessageBox.confirm(`确定删除 ${selection.value.length} 项？（移入回收站）`, '批量删除', { type: 'warning' })
  for (const entry of selection.value) {
    try {
      await deleteFile(currentPath.value, entry.name)
    } catch {
      // 继续
    }
  }
  ElMessage.success('批量删除完成')
  await load()
}

// 压缩
async function handleCompress(entry?: FileEntry) {
  const targets = entry ? [entry.name] : selection.value.map((f) => f.name)
  if (!targets.length) {
    ElMessage.warning('请先选择文件')
    return
  }
  try {
    await compressFiles(currentPath.value, targets, 'zip', 'archive')
    ElMessage.success('压缩成功')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '压缩失败')
  }
}

// 解压
async function handleExtract(entry: FileEntry) {
  try {
    await extractFile(currentPath.value, entry.name)
    ElMessage.success('解压成功')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '解压失败')
  }
}

// 分享
async function handleShare(entry: FileEntry) {
  try {
    const res = await shareFile(currentPath.value, entry.name, 24)
    ElMessageBox.alert(`分享链接（24小时有效）：\n${res.url}`, '分享成功', {
      confirmButtonText: '复制',
    }).then(() => {
      navigator.clipboard?.writeText(res.url)
      ElMessage.success('已复制')
    })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '分享失败')
  }
}

// 编辑
function handleEdit(entry: FileEntry) {
  editingFile.value = { path: currentPath.value, name: entry.name }
  editorVisible.value = true
}

// 下载
function downloadUrl(entry: FileEntry): string {
  const base = window.location.pathname.match(/^(\/[^/]+)/)?.[1] ?? ''
  return `${base}/api/files/download?path=${encodeURIComponent(currentPath.value)}&name=${encodeURIComponent(entry.name)}`
}

function formatSize(size: number): string {
  if (!size) return '—'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function formatDate(s: string): string {
  if (!s) return '—'
  return s.replace('T', ' ').slice(0, 19)
}

function isArchive(name: string): boolean {
  return /\.(zip|tar\.gz|tgz|tar|gz|7z)$/i.test(name)
}

function onSelectionChange(rows: FileEntry[]) {
  selection.value = rows
}

onMounted(load)
</script>

<template>
  <div class="files-view">
    <!-- 面包屑 -->
    <div class="breadcrumb-bar">
      <el-button :icon="'ArrowBack'" circle size="small" @click="navigateTo('.')" />
      <el-breadcrumb separator="/">
        <el-breadcrumb-item @click="navigateTo('.')">根</el-breadcrumb-item>
        <el-breadcrumb-item v-for="crumb in breadcrumbs" :key="crumb.path" @click="navigateTo(crumb.path)">
          {{ crumb.name }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="left">
        <el-button size="small" @click="triggerUpload">上传</el-button>
        <el-button size="small" @click="handleMkdir">新建目录</el-button>
        <el-button size="small" @click="handleTouch">新建文件</el-button>
        <el-button size="small" :disabled="!selection.length" @click="handleCompress()">压缩</el-button>
        <el-button size="small" type="danger" :disabled="!selection.length" @click="handleBatchDelete">删除</el-button>
      </div>
      <el-button size="small" :icon="'Refresh'" circle @click="load" />
    </div>
    <input ref="uploadRef" type="file" multiple hidden @change="handleUploadChange" />

    <!-- 文件列表 -->
    <el-table
      v-loading="loading"
      :data="files"
      @selection-change="onSelectionChange"
      @row-dblclick="enterDir"
      stripe
    >
      <el-table-column type="selection" width="44" />
      <el-table-column label="名称" min-width="280">
        <template #default="{ row }">
          <div class="name-cell" @click="enterDir(row)">
            <span class="file-icon">{{ row.is_dir ? '📁' : getFileIcon(row.name) }}</span>
            <span :class="{ clickable: row.is_dir }">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="大小" width="100">
        <template #default="{ row }">{{ row.is_dir ? '—' : formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="修改时间" width="170">
        <template #default="{ row }">{{ formatDate(row.modified) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button v-if="!row.is_dir" link size="small" @click="handleEdit(row)">编辑</el-button>
          <el-link v-if="!row.is_dir" :href="downloadUrl(row)" :underline="false" target="_blank">
            <el-button link size="small">下载</el-button>
          </el-link>
          <el-button link size="small" @click="handleRename(row)">重命名</el-button>
          <el-button v-if="!row.is_dir && isArchive(row.name)" link size="small" type="success" @click="handleExtract(row)">解压</el-button>
          <el-button link size="small" @click="handleCompress(row)">压缩</el-button>
          <el-button v-if="!row.is_dir" link size="small" @click="handleShare(row)">分享</el-button>
          <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="空目录" />
      </template>
    </el-table>

    <FileEditor v-model="editorVisible" :file="editingFile" />
  </div>
</template>

<script lang="ts">
// 文件图标（按扩展名）
function getFileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() ?? ''
  const map: Record<string, string> = {
    js: '📜', ts: '📜', go: '📜', py: '📜', sh: '📜', json: '⚙️',
    html: '🌐', css: '🎨', md: '📝', txt: '📄',
    zip: '📦', gz: '📦', tar: '📦', '7z': '📦',
    png: '🖼️', jpg: '🖼️', jpeg: '🖼️', gif: '🖼️', svg: '🖼️',
    mp4: '🎬', mp3: '🎵',
  }
  return map[ext] ?? '📄'
}
export default { name: 'FilesView' }
</script>

<style scoped lang="scss">
.files-view {
  max-width: 1400px;
}

.breadcrumb-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;

  .left {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: default;

  .file-icon {
    font-size: 16px;
  }

  .clickable {
    cursor: pointer;
    color: var(--el-color-primary);

    &:hover {
      text-decoration: underline;
    }
  }
}
</style>
