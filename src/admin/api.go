package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

var startTime = time.Now()

// ==================== 状态 ====================

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := h.getSessionUsername(r)
	defer func() {
		handlers.LogPanelAPI(username, r.Method, r.URL.Path, 200, time.Since(start))
	}()

	processMem := handlers.GetProcessMemory()

	// 获取服务器状态
	adminRunning := false
	sitesRunning := false
	ftpRunning := h.Config.FTP.Enabled
	adminPort := h.Config.Global.AdminPort

	if h.ServerManager != nil {
		adminRunning = h.ServerManager.IsAdminRunning()
		sitesRunning = h.ServerManager.IsSitesRunning()
		ftpRunning = h.ServerManager.IsFTPRunning()
	}

	// 构建站点列表
	sites := make([]map[string]interface{}, 0)
	for _, site := range h.Config.Sites {
		sites = append(sites, map[string]interface{}{
			"id":      site.ID,
			"name":    site.Name,
			"enabled": site.Enabled,
			"type":    site.Type,
			"port":    site.Port,
			"domains": site.Domain,
		})
	}

	session := h.getSession(r)
	var csrfToken string
	if session != nil {
		if cookie, _ := r.Cookie("session_id"); cookie != nil {
			csrfToken = h.generateCSRFToken(cookie.Value)
		}
	}

	// 构建状态数据
	statusData := map[string]interface{}{
		"memory_mb":     processMem / 1024 / 1024,
		"goroutines":    runtime.NumGoroutine(),
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"uptime":        time.Since(startTime).Milliseconds(),
		"admin_running": adminRunning,
		"admin_port":    adminPort,
		"sites_running": sitesRunning,
		"sites_count":   len(h.Config.Sites),
		"sites":         sites,
		"ftp_running":   ftpRunning,
		"ftp_port":      h.Config.FTP.Port,
		"ftp_root":      h.Config.FTP.Root,
		"data_dir":      h.Config.Global.DataDir,
	}

	// 如果有 CSRF token，添加到响应中
	if csrfToken != "" {
		statusData["csrf_token"] = csrfToken
	}

	Success(w, statusData)
}

// ==================== 配置 ====================

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	Success(w, h.Config)
}

func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	if r.Method != http.MethodPost {
		BadRequest(w, "Method not allowed")
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		BadRequest(w, "Invalid JSON")
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}
	if err := newConfig.Save(h.ConfigPath); err != nil {
		InternalServerError(w, err.Error())
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}
	*h.Config = newConfig
	handlers.LogPanelConfigChange(username, "保存配置", true)
	SuccessMessage(w, "配置已保存")
}

// ==================== 日志 ====================

func (h *Handler) getLogs(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if lines == 0 {
		lines = 100
	}

	// 兼容旧的单层参数
	if category == "" {
		if logType == "access" {
			category = "http"
		} else if logType == "error" {
			category = "http"
		} else if logType == "server" {
			category = "http"
		} else {
			category = "http"
			logType = "access"
		}
	}
	if logType == "" {
		logType = "access"
	}

	// 安全检查：防止路径遍历
	if strings.Contains(category, "..") || strings.Contains(logType, "..") {
		BadRequest(w, "Invalid log type")
		return
	}

	// 构建日志文件路径
	logPath := handlers.GetLogFilePath(category, logType)
	if logPath == "" {
		BadRequest(w, "Invalid log type")
		return
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		Success(w, map[string]interface{}{"logs": []string{}})
		return
	}
	allLines := strings.Split(string(data), "\n")
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}
	Success(w, map[string]interface{}{"logs": allLines[start:]})
}

func (h *Handler) clearLogs(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")

	// 兼容旧的单层参数
	if category == "" {
		if logType == "access" {
			category = "http"
		} else if logType == "error" {
			category = "http"
		} else if logType == "server" {
			category = "http"
		} else {
			category = "http"
			logType = "access"
		}
	}
	if logType == "" {
		logType = "access"
	}

	// 安全检查：防止路径遍历
	if strings.Contains(category, "..") || strings.Contains(logType, "..") {
		BadRequest(w, "Invalid log type")
		return
	}

	// 构建日志文件路径
	logPath := handlers.GetLogFilePath(category, logType)
	if logPath == "" {
		BadRequest(w, "Invalid log type")
		return
	}

	os.WriteFile(logPath, []byte{}, 0644)
	SuccessMessage(w, "日志已清空")
}
