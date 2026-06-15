<script setup lang="ts">
import { reactive, watch, computed, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { Site, SiteType, ProxyConfig } from '@/types/site'

const props = defineProps<{
  modelValue: boolean
  site: Site | null // null = 新建
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  submit: [data: Partial<Site>]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

// 表单数据（独立副本，避免直接改原对象）
const form = reactive({
  name: '',
  type: 'static' as SiteType,
  port: 0,
  domain: [] as string[],
  root: '',
  index_files: ['index.html', 'index.htm'],
  auto_index: true,
  proxy_target: '',
  ssl_enabled: false,
  force_https: true,
})

const formRef = ref<FormInstance>()

const rules: FormRules = {
  name: [{ required: true, message: '请输入站点名称', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
}

// 域名输入（逗号分隔字符串 ↔ 数组）
const domainText = computed({
  get: () => form.domain.join(', '),
  set: (v: string) => {
    form.domain = v
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  },
})

// 打开时同步 props.site → form
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    if (props.site) {
      // 编辑：回填
      Object.assign(form, {
        name: props.site.name,
        type: props.site.type,
        port: props.site.port,
        domain: [...props.site.domain],
        root: props.site.root ?? '',
        index_files: props.site.index_files?.length ? [...props.site.index_files] : ['index.html', 'index.htm'],
        auto_index: props.site.auto_index,
        proxy_target: props.site.proxy?.target ?? '',
        ssl_enabled: props.site.ssl?.enabled ?? false,
        force_https: props.site.ssl?.force_https ?? true,
      })
    } else {
      // 新建：默认值
      Object.assign(form, {
        name: '',
        type: 'static',
        port: 0,
        domain: [],
        root: '',
        index_files: ['index.html', 'index.htm'],
        auto_index: true,
        proxy_target: '',
        ssl_enabled: false,
        force_https: true,
      })
    }
  },
)

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  const data: Partial<Site> = {
    name: form.name,
    type: form.type,
    port: form.port,
    domain: form.domain,
    auto_index: form.auto_index,
  }
  if (form.type === 'static') {
    data.index_files = form.index_files
    if (form.root) data.root = form.root
  }
  if (form.type === 'proxy' && form.proxy_target) {
    const proxy: ProxyConfig = { target: form.proxy_target }
    data.proxy = proxy
  }
  data.ssl = {
    enabled: form.ssl_enabled,
    force_https: form.force_https,
  }

  emit('submit', data)
  visible.value = false
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="site ? '编辑站点' : '添加站点'"
    width="560px"
    destroy-on-close
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-form-item label="站点名称" prop="name">
        <el-input v-model="form.name" placeholder="我的网站" />
      </el-form-item>

      <el-form-item label="类型" prop="type">
        <el-radio-group v-model="form.type">
          <el-radio-button value="static">静态站点</el-radio-button>
          <el-radio-button value="proxy">反向代理</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="端口" prop="port">
        <el-input-number v-model="form.port" :min="0" :max="65535" controls-position="right" />
        <span class="form-hint">0 = 使用共享端口</span>
      </el-form-item>

      <el-form-item label="域名">
        <el-input v-model="domainText" placeholder="example.com, www.example.com（逗号分隔）" />
      </el-form-item>

      <!-- 静态站点专属 -->
      <template v-if="form.type === 'static'">
        <el-form-item label="根目录">
          <el-input v-model="form.root" placeholder="./www/my-site（留空使用默认派生目录）" />
        </el-form-item>
        <el-form-item label="首页文件">
          <el-input
            :model-value="form.index_files.join(', ')"
            @update:model-value="
              (v: string) =>
                (form.index_files = v.split(',').map((s) => s.trim()).filter(Boolean))
            "
            placeholder="index.html, index.htm"
          />
        </el-form-item>
        <el-form-item label="目录列表">
          <el-switch v-model="form.auto_index" />
          <span class="form-hint">无首页文件时显示目录</span>
        </el-form-item>
      </template>

      <!-- 反向代理专属 -->
      <template v-else>
        <el-form-item label="代理目标">
          <el-input v-model="form.proxy_target" placeholder="http://127.0.0.1:3000" />
        </el-form-item>
      </template>

      <el-divider content-position="left">SSL</el-divider>
      <el-form-item label="启用 SSL">
        <el-switch v-model="form.ssl_enabled" />
      </el-form-item>
      <el-form-item v-if="form.ssl_enabled" label="强制 HTTPS">
        <el-switch v-model="form.force_https" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.form-hint {
  margin-left: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
