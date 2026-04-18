package panel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"pixelbeast/src/logger"
	"runtime"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// ==================== Web 终端 ====================

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// terminalSession 终端会话
type terminalSession struct {
	ptmx *os.File
	cmd  *exec.Cmd
	conn *websocket.Conn
	mu   sync.Mutex
}

// handleTerminalWS WebSocket 终端处理
func (h *Handler) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	// 验证登录状态
	session := h.getSession(r)
	if session == nil {
		http.Error(w, "未登录", http.StatusUnauthorized)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, fmt.Sprintf("终端 WebSocket 升级失败: %v", err))
		return
	}
	defer conn.Close()

	// 创建 PTY
	shell := getShell()
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 设置初始工作目录（从查询参数获取）
	if cwd := r.URL.Query().Get("cwd"); cwd != "" {
		cmd.Dir = cwd
	}

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, fmt.Sprintf("创建 PTY 失败: %v", err))
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n创建终端失败: %v\r\n", err)))
		return
	}

	ts := &terminalSession{
		ptmx: ptmx,
		cmd:  cmd,
		conn: conn,
	}

	logger.LogPanelRuntime(logger.LogLevelInfo, fmt.Sprintf("终端会话创建: PID=%d, shell=%s", cmd.Process.Pid, shell))

	// PTY → WebSocket（读取终端输出并发送到浏览器）
	go ts.ptyToWS()

	// WebSocket → PTY（读取浏览器输入并发送到终端）
	ts.wsToPTY()

	// 清理
	ts.close()
}

// ptyToWS 从 PTY 读取输出并发送到 WebSocket
func (ts *terminalSession) ptyToWS() {
	buf := make([]byte, 4096)
	for {
		n, err := ts.ptmx.Read(buf)
		if err != nil {
			// PTY 关闭（进程退出）
			ts.mu.Lock()
			ts.conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[90m--- 进程已退出 ---\x1b[0m\r\n"))
			ts.mu.Unlock()
			return
		}
		ts.mu.Lock()
		err = ts.conn.WriteMessage(websocket.TextMessage, buf[:n])
		ts.mu.Unlock()
		if err != nil {
			return
		}
	}
}

// wsToPTY 从 WebSocket 读取输入并发送到 PTY
func (ts *terminalSession) wsToPTY() {
	for {
		_, message, err := ts.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg struct {
			Type string `json:"type"`
			Data string `json:"data"`
			Rows uint16 `json:"rows"`
			Cols uint16 `json:"cols"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "input":
			// 键盘输入
			ts.ptmx.Write([]byte(msg.Data))
		case "resize":
			// 终端大小调整
			if msg.Rows > 0 && msg.Cols > 0 {
				pty.Setsize(ts.ptmx, &pty.Winsize{
					Rows: msg.Rows,
					Cols: msg.Cols,
				})
			}
		}
	}
}

// close 清理终端会话
func (ts *terminalSession) close() {
	if ts.ptmx != nil {
		ts.ptmx.Close()
	}
	if ts.cmd != nil && ts.cmd.Process != nil {
		ts.cmd.Process.Kill()
		ts.cmd.Wait()
	}
	logger.LogPanelRuntime(logger.LogLevelInfo, "终端会话关闭")
}

// getShell 获取可用的 shell
func getShell() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	// 优先使用 bash，其次 sh
	for _, sh := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	return "sh"
}

// handleTerminalCommand 通过 WebSocket 发送初始命令（供文件管理器调用）
func (h *Handler) handleTerminalCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "无效的请求参数")
		return
	}

	Success(w, map[string]interface{}{
		"command": req.Command,
	})
}
