<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { readFileInfo, saveFileContent } from '@/api/modules/files'

const props = defineProps<{
  modelValue: boolean
  file: { path: string; name: string } | null
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
}>()

const content = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)
const fileType = ref('text')
const dirty = computed(() => content.value !== originalContent.value)

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

watch(
  () => props.modelValue,
  async (open) => {
    if (!open || !props.file) return
    loading.value = true
    try {
      const res = await readFileInfo(props.file.path, props.file.name)
      content.value = res.content
      originalContent.value = res.content
      fileType.value = res.type
    } catch (e) {
      ElMessage.error(e instanceof Error ? e.message : '读取文件失败')
    } finally {
      loading.value = false
    }
  },
)

async function handleSave() {
  if (!props.file) return
  saving.value = true
  try {
    await saveFileContent(props.file.path, props.file.name, content.value)
    originalContent.value = content.value
    ElMessage.success('保存成功')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function handleClose() {
  if (dirty.value) {
    ElMessageBox.confirm('有未保存的修改，确定关闭？', '提示', { type: 'warning' })
      .then(() => (visible.value = false))
      .catch(() => {})
    return
  }
  visible.value = false
}
</script>

<script lang="ts">
import { computed } from 'vue'
import { ElMessageBox } from 'element-plus'
export default { name: 'FileEditor' }
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="file ? `编辑：${file.name}` : '编辑文件'"
    width="80%"
    top="5vh"
    :before-close="handleClose"
    destroy-on-close
  >
    <div v-loading="loading" class="editor-wrap">
      <div class="editor-meta">
        <el-tag size="small">{{ fileType }}</el-tag>
        <el-tag v-if="dirty" size="small" type="warning">未保存</el-tag>
      </div>
      <el-input
        v-model="content"
        type="textarea"
        :autosize="{ minRows: 20, maxRows: 30 }"
        class="editor-textarea"
        :input-style="{ fontFamily: 'Consolas, Monaco, monospace', fontSize: '13px' }"
      />
    </div>
    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
      <el-button type="primary" :loading="saving" :disabled="!dirty" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.editor-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.editor-meta {
  display: flex;
  gap: 8px;
}

.editor-textarea {
  :deep(textarea) {
    font-family: 'Consolas', 'Monaco', monospace !important;
    font-size: 13px !important;
    line-height: 1.5 !important;
  }
}
</style>
