// Package admin 管理面板模块
package panel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"strings"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pixelbeast/backend/internal/config"
	"pixelbeast/backend/internal/ftp"
	"pixelbeast/backend/internal/logger"
	"pixelbeast/backend/internal/site"
	"pixelbeast/backend/internal/ssl"
)

// ==================== 常量 ====================

const (
	SessionTimeout   = 24 * time.Hour
	MaxLoginAttempts = 5
	LockoutDuration  = 15 * time.Minute
	CSRFTokenTimeout = 2 * time.Hour

	ServiceRestartDelay = 500 * time.Millisecond // 服务重启间隔
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
		SessionID string
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
	cancel            context.CancelFunc
	mu                sync.RWMutex
}

// ==================== 构造函数 ====================

// New 创建管理面板处理器
func New(cm *config.ConfigManager, configPath string) *Handler {
	adminPath := cm.Server.Admin.Path
	if adminPath == "" {
		adminPath = "/admin"
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := &Handler{
		adminPath:         adminPath,
		sessions:          make(map[string]*Session),
		loginAttempts:     make(map[string]*LoginAttempt),
		csrfTokens:        make(map[string]*CSRFToken),
		cancel:            cancel,
		passwordValidator: cm,
		ConfigManager:     cm,
	}

	// 初始化分享服务
	InitShareService(configPath)

	go h.cleanupSessions(ctx)
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

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成会话 ID 失败: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (h *Handler) generateCSRFToken(sessionID string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	token := hex.EncodeToString(bytes)
	h.mu.Lock()
	h.csrfTokens[token] = &CSRFToken{Value: token, SessionID: sessionID, ExpiresAt: time.Now().Add(CSRFTokenTimeout)}
	h.mu.Unlock()
	return token
}

func (h *Handler) cleanupSessions(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
}

// Close 停止后台 goroutine，释放资源
func (h *Handler) Close() {
	if h.cancel != nil {
		h.cancel()
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

func (h *Handler) setSession(w http.ResponseWriter, r *http.Request, username string) error {
	sessionID, err := generateSessionID()
	if err != nil {
		return err
	}
	now := time.Now()
	h.mu.Lock()
	h.sessions[sessionID] = &Session{Username: username, CreatedAt: now, ExpiresAt: now.Add(SessionTimeout)}
	h.mu.Unlock()
	isSecure := r.URL.Scheme == "https" || r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name: "session_id", Value: sessionID, Path: "/", HttpOnly: true,
		Secure: isSecure, SameSite: http.SameSiteStrictMode, MaxAge: int(SessionTimeout.Seconds()),
	})
	return nil
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
	// Vue SPA：直接提供 index.html，由前端路由处理 /login
	data, _ := fs.ReadFile(staticFS, "index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	w.Write(data)
}

func (h *Handler) indexPage(w http.ResponseWriter, r *http.Request) {
	// 读取 index.html（Vue SPA 或原生版，取决于 embed.go 返回的 FS）
	data, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	htmlContent := string(data)

	// 兼容原生版的占位符替换（Vue 版无占位符时为空操作）
	name := h.ConfigManager.Server.Name
	if name == "" {
		name = "PixelBeast Server"
	}
	htmlContent = strings.ReplaceAll(htmlContent, "{{SERVER_NAME}}", html.EscapeString(name))
	htmlContent = strings.ReplaceAll(htmlContent, "{{VERSION}}", h.Version)

	// 原生版 CSRF token 注入（Vue 版通过 /api/system/status 获取，无此占位符）
	csrfPlaceholder := "{{CSRF_TOKEN}}"
	if strings.Contains(htmlContent, csrfPlaceholder) {
		session := h.getSession(r)
		csrfToken := ""
		if session != nil {
			if cookie, err := r.Cookie("session_id"); err == nil {
				csrfToken = h.generateCSRFToken(cookie.Value)
			}
		}
		htmlContent = strings.ReplaceAll(htmlContent, csrfPlaceholder, csrfToken)
	}

	// 原生版的版本号缓存破坏（Vue 版资源已含 hash，此替换无副作用）
	v := h.Version
	if v == "" {
		v = "dev"
	}
	if strings.Contains(htmlContent, "{{VERSION}}") == false {
		// 仅当原生版时执行（避免重复替换）
		htmlContent = strings.ReplaceAll(htmlContent, ".css", ".css?v="+v)
		htmlContent = strings.ReplaceAll(htmlContent, ".js", ".js?v="+v)
		htmlContent = strings.ReplaceAll(htmlContent, "</head>", `<meta name="app-version" content="`+v+`">`+"\n</head>")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	w.Write([]byte(htmlContent))
}

func (h *Handler) serveFavicon(w http.ResponseWriter, r *http.Request) {
	fileCache.HandleCached(w, r, "favicon.ico", "image/x-icon")
}

// serveStaticFile 通用静态文件服务（带缓存）
func (h *Handler) serveStaticFile(w http.ResponseWriter, r *http.Request, subdir, contentType string) {
	file := strings.TrimPrefix(r.URL.Path, "/"+subdir+"/")
	// 去除版本查询参数（?v=xxx），不影响文件路径
	file = strings.SplitN(file, "?", 2)[0]
	if file == "" {
		http.NotFound(w, r)
		return
	}
	fileCache.HandleCached(w, r, subdir+"/"+file, contentType)
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
	file = strings.SplitN(file, "?", 2)[0]
	if file == "" {
		http.NotFound(w, r)
		return
	}
	fileCache.HandleCached(w, r, "images/"+file, "")
}

func (h *Handler) serveIcons(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/icons/")
	file = strings.SplitN(file, "?", 2)[0]
	if file == "" {
		http.NotFound(w, r)
		return
	}
	fileCache.HandleCached(w, r, "icons/"+file, "image/svg+xml")
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
		respondJSON(w, http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "用户名或密码错误",
		})
		return
	}
	h.recordLoginAttempt(clientIP, true)

	// 会话固定防护：先清除旧 session，再生成新 session
	h.clearSession(w, r)
	if err := h.setSession(w, r, username); err != nil {
		InternalServerError(w, "创建会话失败")
		return
	}

	// 首次登录删除初始密码文件
	if h.ConfigManager.Server.Admin.RequirePasswordChange {
		passwordFile := filepath.Join(h.ConfigManager.ConfigDir(), "initial_password.txt")
		os.Remove(passwordFile)
		h.ConfigManager.Server.Admin.RequirePasswordChange = false
		h.ConfigManager.Save()
	}

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
	time.Sleep(ServiceRestartDelay)
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
