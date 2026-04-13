// Package admin 管理面板模块
package panel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"os"
	"sync"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/ftp"
	"pixelbeast/src/logger"
	"pixelbeast/src/site"
	"pixelbeast/src/ssl"
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
	ConfigManager     *config.ConfigManager // 配置管理器
	SiteManager       *site.SiteManager     // 站点管理器
	SSLManager        *ssl.SSLManager       // SSL 管理器
	FTPServer         *ftp.FTPServer        // FTP 服务器
	ftpRunning        bool                  // FTP 运行状态
	passwordValidator PasswordValidator     // 密码验证器（支持加密密码）
	adminPath         string               // 安全入口路径
	Version           string               // 版本号
	sessions          map[string]*Session
	loginAttempts     map[string]*LoginAttempt
	csrfTokens        map[string]*CSRFToken
	mu                sync.RWMutex
}

// ==================== 构造函数 ====================

// New 创建管理面板处理器
func New(cm *config.ConfigManager, configPath string) *Handler {
	adminPath := cm.Server.Admin.Path
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

func (h *Handler) SetSiteManager(sm *site.SiteManager) {
	h.SiteManager = sm
	h.SSLManager = sm.SSLManager
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
	if boundDomain := h.ConfigManager.Server.Admin.Domain; boundDomain != "" {
		host := r.Host
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

	// ACME 文件验证（公开访问，用于 SSL 证书验证，绕过安全入口检查）
	if strings.HasPrefix(path, "/.well-known/acme-challenge/") {
		if h.SSLManager != nil {
			h.SSLManager.GetACMEChallengeHandler().ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}

	// 安全入口检查：只有匹配 adminPath 的请求才会被处理
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

	// 委托到路由器
	h.getRouter().ServeHTTP(w, r)
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
	name := h.ConfigManager.Server.Name
	if name == "" {
		name = "PixelBeast Server"
	}
	html := strings.ReplaceAll(string(data), "{{SERVER_NAME}}", name)
	html = strings.ReplaceAll(html, "{{VERSION}}", h.Version)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *Handler) indexPage(w http.ResponseWriter, r *http.Request) {
	data, _ := fs.ReadFile(staticFS, "index.html")
	name := h.ConfigManager.Server.Name
	if name == "" {
		name = "PixelBeast Server"
	}
	html := strings.ReplaceAll(string(data), "{{SERVER_NAME}}", name)
	html = strings.ReplaceAll(html, "{{VERSION}}", h.Version)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *Handler) serveFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(staticFS, "favicon.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Write(data)
}

// serveStaticFile 通用静态文件服务
func (h *Handler) serveStaticFile(w http.ResponseWriter, r *http.Request, subdir, contentType string) {
	file := strings.TrimPrefix(r.URL.Path, "/"+subdir+"/")
	data, err := fs.ReadFile(staticFS, subdir+"/"+file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(data)
}

func (h *Handler) serveCSS(w http.ResponseWriter, r *http.Request) {
	h.serveStaticFile(w, r, "css", "text/css")
}

func (h *Handler) serveJS(w http.ResponseWriter, r *http.Request) {
	h.serveStaticFile(w, r, "js", "application/javascript")
}

func (h *Handler) serveComponents(w http.ResponseWriter, r *http.Request) {
	h.serveStaticFile(w, r, "components", "text/html; charset=utf-8")
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
		logger.LogPanelAuth("登录", username, clientIP, false, "用户名或密码错误")
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
	logger.LogPanelAuth("登录", username, clientIP, true, "登录成功")
	SuccessMessage(w, "登录成功")
}

func (h *Handler) logoutAPI(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	h.clearSession(w, r)
	logger.LogPanelAuth("登出", username, clientIP, true, "用户主动登出")
	SuccessMessage(w, "已登出")
}

// getHomeDir 获取用户主目录（带缓存）
var (
	homeDirOnce sync.Once
	homeDirValue string
	homeDirErr  error
)

func getHomeDir() string {
	homeDirOnce.Do(func() {
		homeDirValue, homeDirErr = os.UserHomeDir()
		if homeDirErr != nil {
			homeDirValue = ""
		}
	})
	return homeDirValue
}

// ==================== FTP 服务管理 ====================

func (h *Handler) startFTPSvc() error {
	if h.FTPServer == nil {
		return fmt.Errorf("FTP 服务器未初始化")
	}
	if err := h.FTPServer.Start(); err != nil {
		return err
	}
	h.ftpRunning = true
	logger.LogPanelRuntime(logger.LogLevelInfo, "[FTP] 服务已启动")
	return nil
}

func (h *Handler) stopFTPSvc() error {
	if h.FTPServer == nil || !h.ftpRunning {
		return nil
	}
	if err := h.FTPServer.Stop(); err != nil {
		return err
	}
	h.ftpRunning = false
	logger.LogPanelRuntime(logger.LogLevelInfo, "[FTP] 服务已停止")
	return nil
}

func (h *Handler) restartFTPSvc() error {
	if err := h.stopFTPSvc(); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	return h.startFTPSvc()
}

func (h *Handler) syncFTPConfig() {
	if h.FTPServer != nil && h.ConfigManager != nil {
		h.FTPServer.Config = h.ConfigManager.FTP
		h.FTPServer.SetValidator(h.ConfigManager)
	}
}

// SetFTPRunning 设置 FTP 运行状态（由 main.go 调用）
func (h *Handler) SetFTPRunning(running bool) {
	h.ftpRunning = running
}
