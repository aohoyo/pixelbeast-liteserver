<script setup lang="ts">
import { reactive, watch, computed, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { FtpUser } from '@/types/ftp'

const props = defineProps<{
  modelValue: boolean
  user: FtpUser | null
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  submit: [data: { username: string; password?: string; rootPath?: string; quota?: number; remark?: string; expiryDays?: number }]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const formRef = ref<FormInstance>()
const form = reactive({
  username: '',
  password: '',
  rootPath: '',
  quota: 0,
  remark: '',
  expiryDays: 0,
})

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    if (props.user) {
      Object.assign(form, {
        username: props.user.username,
        password: '',
        rootPath: props.user.rootPath,
        quota: props.user.quota,
        remark: props.user.remark,
        expiryDays: props.user.expiryDays,
      })
    } else {
      Object.assign(form, { username: '', password: '', rootPath: '', quota: 0, remark: '', expiryDays: 0 })
    }
  },
)

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  const data: { username: string; password?: string; rootPath?: string; quota?: number; remark?: string; expiryDays?: number } = {
    username: form.username,
    rootPath: form.rootPath,
    quota: form.quota,
    remark: form.remark,
    expiryDays: form.expiryDays,
  }
  if (form.password) data.password = form.password
  emit('submit', data)
  visible.value = false
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="user ? '编辑 FTP 用户' : '添加 FTP 用户'"
    width="520px"
    destroy-on-close
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" :disabled="!!user" placeholder="ftpuser" />
      </el-form-item>
      <el-form-item :label="user ? '新密码' : '密码'">
        <el-input v-model="form.password" type="password" show-password :placeholder="user ? '留空不修改' : '请输入密码'" />
      </el-form-item>
      <el-form-item label="根目录">
        <el-input v-model="form.rootPath" placeholder="/ftpuser（留空自动派生）" />
      </el-form-item>
      <el-form-item label="配额 (MB)">
        <el-input-number v-model="form.quota" :min="0" controls-position="right" />
        <span class="hint">0 = 无限</span>
      </el-form-item>
      <el-form-item label="过期天数">
        <el-input-number v-model="form.expiryDays" :min="0" controls-position="right" />
        <span class="hint">0 = 永久</span>
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.remark" placeholder="可选" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.hint {
  margin-left: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
