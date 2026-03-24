// Package admin 管理面板模块
package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
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

// Handler 管理面板处理器
type Handler struct {
	Config        *config.Config
	ConfigPath    string
	ServerManager *handlers.ServerManager
	sessions      map[string]*Session
	loginAttempts map[string]*LoginAttempt
	csrfTokens    map[string]*CSRFToken
	mu            sync.RWMutex
}

// ==================== 构造函数 ====================

func New(cfg *config.Config, configPath string) *Handler {
	h := &Handler{
		Config:        cfg,
		ConfigPath:    configPath,
		sessions:      make(map[string]*Session),
		loginAttempts: make(map[string]*LoginAttempt),
		csrfTokens:    make(map[string]*CSRFToken),
	}
	go h.cleanupSessions()
	return h
}

func (h *Handler) SetServerManager(sm *handlers.ServerManager) {
	h.ServerManager = sm
}

// ==================== 路由 ====================

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "" {
		path = "/"
	}

	// 公开路由
	switch path {
	case "/login":
		h.loginPage(w, r)
		return
	case "/api/login":
		h.loginAPI(w, r)
		return
	case "/api/logout":
		h.logoutAPI(w, r)
		return
	}

	// 静态资源
	if strings.HasPrefix(path, "/css/") {
		h.serveCSS(w, r)
		return
	}
	if strings.HasPrefix(path, "/js/") {
		h.serveJS(w, r)
		return
	}
	if strings.HasPrefix(path, "/components/") {
		h.serveComponents(w, r)
		return
	}
	if strings.HasPrefix(path, "/images/") {
		h.serveImages(w, r)
		return
	}
	if strings.HasPrefix(path, "/icons/") {
		h.serveIcons(w, r)
		return
	}

	// 认证检查
	session := h.getSession(r)
	if session == nil {
		if strings.HasPrefix(path, "/api/") {
			Unauthorized(w, "未登录")
		} else {
			http.Redirect(w, r, h.Config.Admin.Path+"/login", http.StatusFound)
		}
		return
	}

	// 已认证路由
	switch path {
	// 页面
	case "/", "/index.html":
		h.indexPage(w, r)
	// 状态
	case "/api/status":
		h.getStatus(w, r)
	// 配置
	case "/api/config":
		h.getConfig(w, r)
	case "/api/config/save":
		h.saveConfig(w, r)
	// HTTP 文件管理
	case "/api/files":
		h.listFiles(w, r)
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
	// 日志
	case "/api/logs":
		h.getLogs(w, r)
	case "/api/logs/clear":
		h.clearLogs(w, r)
	// FTP 服务
	case "/api/service/ftp/toggle":
		h.toggleFTP(w, r)
	case "/api/service/ftp/start":
		h.startFTP(w, r)
	case "/api/service/ftp/stop":
		h.stopFTP(w, r)
	case "/api/service/ftp/restart":
		h.restartFTP(w, r)
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
	case "/api/ftp/users/add":
		h.addFtpUser(w, r)
	case "/api/ftp/users/delete":
		h.deleteFtpUser(w, r)
	// 站点管理
	case "/api/sites":
		h.handleSitesList(w, r)
	case "/api/sites/toggle":
		h.handleSiteToggle(w, r)
	default:
		// 处理带 ID 的站点路由
		if strings.HasPrefix(path, "/api/sites/") {
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

func (h *Handler) serveCSS(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/css/")
	data, err := fs.ReadFile(staticFS, "css/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css")
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
	r.ParseForm()
	username, password := r.FormValue("username"), r.FormValue("password")
	if subtle.ConstantTimeCompare([]byte(username), []byte(h.Config.Admin.Username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(h.Config.Admin.Password)) != 1 {
		h.recordLoginAttempt(clientIP, false)
		handlers.LogPanelAuth("登录", username, clientIP, false, "用户名或密码错误")
		h.mu.RLock()
		remaining := MaxLoginAttempts - h.loginAttempts[clientIP].Count
		h.mu.RUnlock()
		// 统一格式，在 data 中返回 remaining
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