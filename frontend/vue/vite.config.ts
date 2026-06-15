import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 开发模式：Vite dev server (5173) 通过 proxy 转发 /api 到 Go 后端 (9527)
// 生产模式：npm run build 输出到 dist/，由 frontend/embed.go 嵌入二进制
export default defineConfig({
  plugins: [vue()],
  // 资源基准路径：匹配后端安全入口 /admin（自定义路径需同步修改）
  base: '/admin/',
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    // 代理后端 API：开发时前端请求 /admin/api/* 由 proxy 转发到 Go 后端
    proxy: {
      '/admin/api': {
        target: 'http://localhost:9527',
        changeOrigin: true,
      },
      '/admin/assets': {
        target: 'http://localhost:9527',
        changeOrigin: true,
      },
      '/admin/s': {
        target: 'http://localhost:9527',
        changeOrigin: true,
      },
      '/admin/share': {
        target: 'http://localhost:9527',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
