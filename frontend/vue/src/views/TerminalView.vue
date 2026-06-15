<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const terminalRef = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
const status = ref<'connecting' | 'connected' | 'disconnected'>('connecting')

function connect() {
  if (!term) return
  status.value = 'connecting'

  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  // WS 也需带 adminPath 前缀（后端安全入口校验）
  const base = window.location.pathname.match(/^(\/[^/]+)/)?.[1] ?? ''
  const url = `${proto}//${location.host}${base}/api/terminal/ws`

  ws = new WebSocket(url)

  ws.onopen = () => {
    status.value = 'connected'
    fit()
  }

  // 服务器 → 客户端：原始字节（ANSI 文本帧）
  ws.onmessage = (event) => {
    if (typeof event.data === 'string') {
      term?.write(event.data)
    }
  }

  ws.onclose = () => {
    status.value = 'disconnected'
  }

  ws.onerror = () => {
    status.value = 'disconnected'
  }

  // 客户端 → 服务器：JSON（type: input / resize）
  term.onData((data) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    }
  })
}

function fit() {
  if (!fitAddon || !term) return
  fitAddon.fit()
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(
      JSON.stringify({
        type: 'resize',
        rows: term.rows,
        cols: term.cols,
      }),
    )
  }
}

function reconnect() {
  if (ws) {
    ws.close()
    ws = null
  }
  connect()
}

onMounted(async () => {
  await nextTick()
  term = new Terminal({
    theme: {
      background: '#0c0a09',
      foreground: '#fafaf9',
      cursor: '#f97316',
    },
    fontSize: 13,
    fontFamily: "'Consolas', 'Monaco', monospace",
    cursorBlink: true,
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalRef.value!)
  fit()
  connect()

  resizeObserver = new ResizeObserver(() => fit())
  resizeObserver.observe(terminalRef.value!)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  ws?.close()
  term?.dispose()
})
</script>

<template>
  <div class="terminal-view">
    <div class="terminal-header">
      <el-tag :type="status === 'connected' ? 'success' : status === 'connecting' ? 'info' : 'danger'" size="small">
        {{ status === 'connected' ? '已连接' : status === 'connecting' ? '连接中' : '已断开' }}
      </el-tag>
      <el-button size="small" @click="reconnect" :disabled="status === 'connecting'">重连</el-button>
    </div>
    <div ref="terminalRef" class="terminal-container" />
  </div>
</template>

<style scoped lang="scss">
.terminal-view {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 140px);
}

.terminal-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.terminal-container {
  flex: 1;
  background: #0c0a09;
  border-radius: 8px;
  padding: 8px;
  overflow: hidden;

  :deep(.xterm) {
    height: 100%;
  }
}
</style>
