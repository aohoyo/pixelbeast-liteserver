// Package admin 管理面板模块
package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

// ==================== 常量 ====================

const (
	SessionTimeout   = 24 * time.Hour
	MaxLoginAttempts = 5
	LockoutDuration  = 15 * time.Minute
	CSRFTokenTimeout = 2 * time.Hour
)

// ==================== 类型定义 ====================

type (
	LoginAttempt struct {
		Count       int
		LastTime    time.Time
		LockedUntil time.Time
	}
	CSRFToken struct {
		Value     string
		ExpiresAt time.Time
	}
	Session struct {
		Username  string
		CreatedAt time.Time
		ExpiresAt time.Time
	}
)

// PasswordValidator 密码验证器接口
type PasswordValidator interface {
	ValidateAdmin(username, password string) bool
	ValidateFTPUser(username, password string) bool
}

// Handler 管理面板处理器
type Handler struct {
	ConfigManager    *config.ConfigManager // 配置管理器
	ServerManager    *handlers.ServerManager
	passwordValidator PasswordValidator // 密码验证器（支持加密密码）
	adminPath        string              // 安全入口路径
	sessions         map[string]*Session
	loginAttempts    map[string]*LoginAttempt
	csrfTokens       map[string]*CSRFToken
	mu               sync.RWMutex
}

// ==================== 构造函数 ====================

// New 创建管理面板处理器
func New(cm *config.ConfigManager, configPath string) *Handler {
	adminPath := cm.Server.AdminPath
	if adminPath == "" {
		adminPath = "/admin"
	}

	h := &Handler{
		adminPath:         adminPath,
		sessions:          make(map[string]*Session),
		loginAttempts:     make(map[string]*LoginAttempt),
		csrfTokens:        make(map[string]*CSRFToken),
		passwordValidator: cm,
		ConfigManager:     cm,
	}

	// 初始化分享服务
	InitShareService(configPath)

	go h.cleanupSessions()
	return h
}

// SetAdminPath 设置安全入口路径
func (h *Handler) SetAdminPath(path string) {
	if path != "" {
		h.adminPath = path
	}
}

// GetAdminPath 获取安全入口路径
func (h *Handler) GetAdminPath() string {
	return h.adminPath
}

func (h *Handler) SetServerManager(sm *handlers.ServerManager) {
	h.ServerManager = sm
}

func (h *Handler) SetConfigManager(cm *config.ConfigManager) {
	h.ConfigManager = cm
}

// ==================== 路由 ====================

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "" {
		path = "/"
	}

	// 域名绑定检查：如果配置了绑定域名，只允许该域名访问
	if boundDomain := h.ConfigManager.Server.AdminDomain; boundDomain != "" {
		host := r.Host
		// 去掉端口
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		if host != boundDomain {
			http.NotFound(w, r)
			return
		}
	}

	// 分享链接下载（公开访问，优先处理，绕过安全入口检查）
	if strings.HasPrefix(path, "/s/") || strings.HasPrefix(path, "/share/") {
		h.downloadSharedFile(w, r)
		return
	}

	// 安全入口检查：只有匹配 adminPath 的请求才会被处理
	// 其他请求返回 404，隐藏后台存在
	adminPath := h.adminPath
	if !strings.HasPrefix(path, adminPath) {
		http.NotFound(w, r)
		return
	}

	// 去掉安全入口前缀，得到实际路径
	actualPath := strings.TrimPrefix(path, adminPath)
	if actualPath == "" {
		actualPath = "/"
	}

	// 修改 r.URL.Path，让后续处理函数可以直接使用
	r.URL.Path = actualPath

	// 静态资源（不需要认证，优先处理）
	if strings.HasPrefix(actualPath, "/css/") {
		h.serveCSS(w, r)
		return
	}
	if strings.HasPrefix(actualPath, "/js/") {
		h.serveJS(w, r)
		return
	}
	if strings.HasPrefix(actualPath, "/components/") {
		h.serveComponents(w, r)
		return
	}
	if strings.HasPrefix(actualPath, "/images/") {
		h.serveImages(w, r)
		return
	}
	if strings.HasPrefix(actualPath, "/icons/") {
		h.serveIcons(w, r)
		return
	}

	// 记录面板访问日志（公开路由之后，认证路由之前）
	startTime := time.Now()

	// 公开路由（不需要认证）
	switch actualPath {
	case "/login":
		h.loginPage(w, r)
		return
	case "/api/login":
		h.loginAPI(w, r)
		return
	case "/api/logout":
		h.logoutAPI(w, r)
		return
	case "/favicon.svg", "/favicon.ico":
		h.serveFavicon(w, r)
		return
	}

	// 认证检查
	session := h.getSession(r)
	if session == nil {
		if strings.HasPrefix(actualPath, "/api/") {
			Unauthorized(w, "未登录")
		} else {
			http.Redirect(w, r, adminPath+"/login", http.StatusFound)
		}
		return
	}

	// 记录已认证用户的访问日志
	remoteAddr := r.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		remoteAddr = remoteAddr[:idx]
	}
	
	// 使用包装 ResponseWriter 来捕获状态码
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	
	// 处理请求（在 defer 中记录日志）
	defer func() {
		duration := time.Since(startTime)
		// 所有已认证请求都记录日志
		if strings.HasPrefix(actualPath, "/api/") {
			handlers.LogPanelAPI(session.Username, r.Method, actualPath, remoteAddr, rw.statusCode, duration)
		} else if actualPath == "/" || actualPath == "/index.html" {
			// 首页访问
			handlers.LogPanelAccess(session.Username, actualPath, remoteAddr)
		}
	}()

	// 替换 w 为 rw
	w = rw

	// 已认证路由
	switch actualPath {
	// 页面
	case "/", "/index.html":
		h.indexPage(w, r)
	// 状态
	case "/api/status":
		h.getStatus(w, r)
	// 系统监控
	case "/api/system/status":
		h.getSystemStatus(w, r)
	case "/api/system/free-memory":
		h.freeMemory(w, r)
	case "/api/system/cleanup-scan":
		h.scanCleanup(w, r)
	case "/api/system/cleanup":
		h.executeCleanup(w, r)
	case "/api/system/time/sync":
		h.syncSystemTime(w, r)
	// 配置
	case "/api/config":
		h.getConfig(w, r)
	case "/api/config/save":
		h.saveConfig(w, r)
	case "/api/config/reset":
		h.resetConfig(w, r)
	// HTTP 文件管理
	case "/api/files":
		h.listFiles(w, r)
	case "/api/files/quick-dirs":
		h.getQuickDirs(w, r)
	case "/api/files/upload":
		h.uploadFile(w, r)
	case "/api/files/upload/chunk":
		h.uploadChunk(w, r)
	case "/api/files/upload/merge":
		h.mergeChunks(w, r)
	case "/api/files/delete":
		h.deleteFile(w, r)
	case "/api/files/mkdir":
		h.mkdir(w, r)
	case "/api/files/rename":
		h.renameFile(w, r)
	case "/api/files/download":
		h.downloadFile(w, r)
	case "/api/files/copy":
		h.copyFile(w, r)
	case "/api/files/touch":
		h.touchFile(w, r)
	case "/api/files/move":
		h.moveFile(w, r)
	case "/api/files/chmod":
		h.chmodFile(w, r)
	case "/api/files/permissions":
		h.getFilePermissions(w, r)
	case "/api/files/read":
		h.readFileContent(w, r)
	case "/api/files/save":
		h.saveFileContent(w, r)
	case "/api/files/compress":
		h.compressFiles(w, r)
	case "/api/files/extract":
		h.extractFile(w, r)
	case "/api/files/share":
		h.shareFile(w, r)
	case "/api/files/share/list":
		h.listShareLinks(w, r)
	case "/api/files/share/delete":
		h.deleteShareLink(w, r)
	// 日志管理
	case "/api/logs":
		h.handleLogsList(w, r)
	case "/api/logs/read":
		h.handleLogsRead(w, r)
	case "/api/logs/stats":
		h.handleLogsStats(w, r)
	case "/api/logs/download":
		h.handleLogsDownload(w, r)
	case "/api/logs/clear":
		h.handleLogsClear(w, r)
	case "/api/logs/config":
		h.handleLogsConfig(w, r)
	// FTP 服务
	case "/api/service/ftp/toggle":
		h.toggleFTP(w, r)
	case "/api/service/ftp/start":
		h.startFTP(w, r)
	case "/api/service/ftp/stop":
		h.stopFTP(w, r)
	case "/api/service/ftp/restart":
		h.restartFTP(w, r)
	case "/api/service/ftp/reload":
		h.reloadFTP(w, r)
	// FTP 文件管理
	case "/api/ftp/files":
		h.listFtpFiles(w, r)
	case "/api/ftp/files/upload":
		h.uploadFtpFile(w, r)
	case "/api/ftp/files/delete":
		h.deleteFtpFile(w, r)
	case "/api/ftp/files/mkdir":
		h.mkdirFtp(w, r)
	case "/api/ftp/files/download":
		h.downloadFtpFile(w, r)
	case "/api/ftp/files/rename":
		h.renameFtpFile(w, r)
	case "/api/ftp/files/copy":
		h.copyFtpFile(w, r)
	// FTP 用户
	case "/api/ftp/users":
		h.listFtpUsers(w, r)
	case "/api/ftp/users/add":
		h.addFtpUser(w, r)
	case "/api/ftp/users/delete":
		h.deleteFtpUser(w, r)
	case "/api/ftp/users/toggle":
		h.toggleFtpUserStatus(w, r)
	case "/api/ftp/users/batch":
		h.batchFtpUsers(w, r)
	// 站点管理
	case "/api/sites":
		h.handleSitesList(w, r)
	case "/api/sites/toggle":
		h.handleSiteToggle(w, r)
	case "/api/sites/batch":
		h.handleSitesBatch(w, r)
	default:
		// 处理带 ID 的站点路由
		if strings.HasPrefix(actualPath, "/api/sites/") {
			h.handleSitesDetail(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// ==================== 会话管理 ====================

func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (h *Handler) generateCSRFToken(sessionID string) string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)
	h.mu.Lock()
	h.csrfTokens[sessionID] = &CSRFToken{Value: token, ExpiresAt: time.Now().Add(CSRFTokenTimeout)}
	h.mu.Unlock()
	return token
}

func (h *Handler) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for id, s := range h.sessions {
			if now.After(s.ExpiresAt) {
				delete(h.sessions, id)
			}
		}
		for ip, a := range h.loginAttempts {
			if now.After(a.LastTime.Add(time.Hour)) {
				delete(h.loginAttempts, ip)
			}
		}
		for id, t := range h.csrfTokens {
			if now.After(t.ExpiresAt) {
				delete(h.csrfTokens, id)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Handler) getSession(r *http.Request) *Session {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	h.mu.RLock()
	session, exists := h.sessions[cookie.Value]
	h.mu.RUnlock()
	if !exists || time.Now().After(session.ExpiresAt) {
		return nil
	}
	return session
}

func (h *Handler) getSessionUsername(r *http.Request) string {
	if session := h.getSession(r); session != nil {
		return session.Username
	}
	return ""
}

func (h *Handler) setSession(w http.ResponseWriter, r *http.Request, username string) {
	sessionID := generateSessionID()
	now := time.Now()
	h.mu.Lock()
	h.sessions[sessionID] = &Session{Username: username, CreatedAt: now, ExpiresAt: now.Add(SessionTimeout)}
	h.mu.Unlock()
	isSecure := r.URL.Scheme == "https" || r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name: "session_id", Value: sessionID, Path: "/", HttpOnly: true,
		Secure: isSecure, SameSite: http.SameSiteStrictMode, MaxAge: int(SessionTimeout.Seconds()),
	})
}

func (h *Handler) clearSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		h.mu.Lock()
		delete(h.sessions, cookie.Value)
		h.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
}

// ==================== 认证 ====================

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	return ip
}

func (h *Handler) checkLoginAttempt(ip string) (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	attempt, exists := h.loginAttempts[ip]
	if !exists {
		return true, ""
	}
	if time.Now().Before(attempt.LockedUntil) {
		return false, fmt.Sprintf("登录尝试过多，请 %.0f 分钟后重试", time.Until(attempt.LockedUntil).Minutes())
	}
	if time.Now().After(attempt.LockedUntil) {
		attempt.Count = 0
		attempt.LockedUntil = time.Time{}
	}
	return true, ""
}

func (h *Handler) recordLoginAttempt(ip string, success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	attempt, exists := h.loginAttempts[ip]
	if !exists {
		attempt = &LoginAttempt{}
		h.loginAttempts[ip] = attempt
	}
	now := time.Now()
	attempt.LastTime = now
	if success {
		attempt.Count = 0
		attempt.LockedUntil = time.Time{}
	} else {
		attempt.Count++
		if attempt.Count >= MaxLoginAttempts {
			attempt.LockedUntil = now.Add(LockoutDuration)
		}
	}
}

// ==================== 页面 ====================

func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	if h.getSession(r) != nil {
		http.Redirect(w, r, "./", http.StatusFound)
		return
	}
	data, _ := fs.ReadFile(staticFS, "views/login.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (h *Handler) indexPage(w http.ResponseWriter, r *http.Request) {
	data, _ := fs.ReadFile(staticFS, "index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (h *Handler) serveFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(staticFS, "favicon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(data)
}

func (h *Handler) serveCSS(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/css/")
	data, err := fs.ReadFile(staticFS, "css/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(data)
}

func (h *Handler) serveJS(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/js/")
	data, err := fs.ReadFile(staticFS, "js/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(data)
}

func (h *Handler) serveComponents(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/components/")
	data, err := fs.ReadFile(staticFS, "components/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (h *Handler) serveImages(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/images/")
	data, err := fs.ReadFile(staticFS, "images/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// 根据文件扩展名设置 Content-Type
	ext := strings.ToLower(file[strings.LastIndex(file, ".")+1:])
	switch ext {
	case "svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case "png":
		w.Header().Set("Content-Type", "image/png")
	case "jpg", "jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case "gif":
		w.Header().Set("Content-Type", "image/gif")
	case "ico":
		w.Header().Set("Content-Type", "image/x-icon")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(data)
}

func (h *Handler) serveIcons(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/icons/")
	data, err := fs.ReadFile(staticFS, "icons/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(data)
}

func (h *Handler) loginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}
	clientIP := getClientIP(r)
	if allowed, msg := h.checkLoginAttempt(clientIP); !allowed {
		TooManyRequests(w, msg)
		return
	}
	
	// 解析请求体（支持 JSON 和表单）
	var username, password string
	contentType := r.Header.Get("Content-Type")
	
	if contentType == "application/json" {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			username = req.Username
			password = req.Password
		}
	} else {
		r.ParseForm()
		username = r.FormValue("username")
		password = r.FormValue("password")
	}
	
	// 验证密码
	valid := false
	if h.passwordValidator != nil {
		valid = h.passwordValidator.ValidateAdmin(username, password)
	} else {
		valid = h.ConfigManager.ValidateAdmin(username, password)
	}
	
	if !valid {
		h.recordLoginAttempt(clientIP, false)
		handlers.LogPanelAuth("登录", username, clientIP, false, "用户名或密码错误")
		h.mu.RLock()
		remaining := MaxLoginAttempts - h.loginAttempts[clientIP].Count
		h.mu.RUnlock()
		respondJSON(w, http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "用户名或密码错误",
			Data:    map[string]interface{}{"remaining": remaining},
		})
		return
	}
	h.recordLoginAttempt(clientIP, true)
	h.setSession(w, r, username)
	handlers.LogPanelAuth("登录", username, clientIP, true, "登录成功")
	SuccessMessage(w, "登录成功")
}

func (h *Handler) logoutAPI(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	h.clearSession(w, r)
	handlers.LogPanelAuth("登出", username, clientIP, true, "用户主动登出")
	SuccessMessage(w, "已登出")
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}