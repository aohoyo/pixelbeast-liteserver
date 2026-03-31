package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ==================== 文件分享服务 ====================

// ShareLink 分享链接
type ShareLink struct {
	Token         string    `json:"token"`
	FilePath      string    `json:"filePath"`
	FileName      string    `json:"fileName"`
	FileSize      int64     `json:"fileSize"`
	ExpiresAt     time.Time `json:"expiresAt"`
	CreatedAt     time.Time `json:"createdAt"`
	DownloadCount int       `json:"downloadCount"`
	Password      string    `json:"password,omitempty"`
}

// ShareService 分享服务
type ShareService struct {
	links    map[string]*ShareLink
	mu       sync.RWMutex
	filePath string
}

// 全局分享服务实例
var shareService *ShareService

// InitShareService 初始化分享服务
func InitShareService(configDir string) {
	shareService = &ShareService{
		links:    make(map[string]*ShareLink),
		filePath: filepath.Join(configDir, "shares.json"),
	}
	shareService.load()
}

// load 从文件加载分享链接
func (s *ShareService) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("[Share] 加载分享数据失败: %v\n", err)
		}
		return
	}

	var links []*ShareLink
	if err := json.Unmarshal(data, &links); err != nil {
		fmt.Printf("[Share] 解析分享数据失败: %v\n", err)
		return
	}

	now := time.Now()
	for _, link := range links {
		// 过滤已过期的链接
		if now.After(link.ExpiresAt) {
			continue
		}
		// 补充文件大小（旧数据可能没有）
		if link.FileSize == 0 {
			if fi, err := os.Stat(link.FilePath); err == nil {
				link.FileSize = fi.Size()
			}
		}
		s.links[link.Token] = link
	}

	fmt.Printf("[Share] 已加载 %d 个有效分享链接\n", len(s.links))
}

// save 保存分享链接到文件
func (s *ShareService) save() {
	s.mu.RLock()
	links := make([]*ShareLink, 0, len(s.links))
	for _, link := range s.links {
		links = append(links, link)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		fmt.Printf("[Share] 序列化分享数据失败: %v\n", err)
		return
	}

	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("[Share] 创建目录失败: %v\n", err)
		return
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		fmt.Printf("[Share] 保存分享数据失败: %v\n", err)
	}
}

// cleanExpired 清理过期链接
func (s *ShareService) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	changed := false
	for token, link := range s.links {
		if now.After(link.ExpiresAt) {
			delete(s.links, token)
			changed = true
		}
	}

	if changed {
		s.save()
	}
}

// shareFile 创建分享链接
func (h *Handler) shareFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path     string `json:"path"`
		Name     string `json:"name"`
		Duration int    `json:"duration"` // 有效时长（小时），默认 24
		Password string `json:"password"` // 访问密码（可选）
	}

	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return
	}

	// 安全检查
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	// 默认 24 小时
	if req.Duration <= 0 {
		req.Duration = 24
	}

	// 生成 token
	token := generateShareToken()

	// 构建文件路径
	targetDir := resolvePath(req.Path)
	filePath := filepath.Join(targetDir, req.Name)

	// 检查文件是否存在并获取大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		BadRequest(w, "文件不存在")
		return
	}

	// 创建分享链接
	link := &ShareLink{
		Token:         token,
		FilePath:      filePath,
		FileName:      req.Name,
		FileSize:      fileInfo.Size(),
		ExpiresAt:     time.Now().Add(time.Duration(req.Duration) * time.Hour),
		CreatedAt:     time.Now(),
		DownloadCount: 0,
		Password:      req.Password,
	}

	// 存储并保存
	shareService.mu.Lock()
	shareService.links[token] = link
	shareService.mu.Unlock()
	shareService.save()

	// 构建分享 URL（不包含 adminPath，走公开路由）
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	shareURL := fmt.Sprintf("%s://%s/s/%s", scheme, r.Host, token)

	Success(w, map[string]interface{}{
		"token":     token,
		"url":       shareURL,
		"expiresAt": link.ExpiresAt.Format(time.RFC3339),
		"fileName":  req.Name,
		"fileSize":  fileInfo.Size(),
	})
}

// getShareInfo 获取分享信息（预览页面用）
func (h *Handler) getShareInfo(w http.ResponseWriter, r *http.Request) {
	// 从 URL 获取 token
	token := strings.TrimPrefix(r.URL.Path, "/s/info/")
	if token == "" {
		Error(w, http.StatusBadRequest, "无效的分享链接")
		return
	}

	// 查找分享链接
	shareService.mu.RLock()
	link, exists := shareService.links[token]
	shareService.mu.RUnlock()

	if !exists {
		Error(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}

	// 检查是否过期
	if time.Now().After(link.ExpiresAt) {
		// 删除过期链接
		shareService.mu.Lock()
		delete(shareService.links, token)
		shareService.mu.Unlock()
		shareService.save()
		Error(w, http.StatusGone, "分享链接已过期")
		return
	}

	// 返回分享信息（不返回文件路径和密码）
	Success(w, map[string]interface{}{
		"token":         link.Token,
		"fileName":      link.FileName,
		"fileSize":      link.FileSize,
		"expiresAt":     link.ExpiresAt.Format(time.RFC3339),
		"createdAt":     link.CreatedAt.Format(time.RFC3339),
		"downloadCount": link.DownloadCount,
		"hasPassword":   link.Password != "",
	})
}

// downloadSharedFile 下载分享文件
func (h *Handler) downloadSharedFile(w http.ResponseWriter, r *http.Request) {
	// 从 URL 获取 token（支持 /s/{token}/download 和 /s/{token}）
	path := r.URL.Path
	token := ""

	// 匹配 /s/{token}/download
	if strings.HasSuffix(path, "/download") {
		token = strings.TrimSuffix(strings.TrimPrefix(path, "/s/"), "/download")
	} else {
		// 兼容旧格式 /s/{token} 直接下载（显示预览页）
		token = strings.TrimPrefix(path, "/s/")
	}

	// 如果没有 /download 后缀，显示预览页
	if !strings.HasSuffix(path, "/download") {
		h.serveSharePage(w, r, token)
		return
	}

	if token == "" {
		Error(w, http.StatusBadRequest, "无效的分享链接")
		return
	}

	// 查找分享链接
	shareService.mu.RLock()
	link, exists := shareService.links[token]
	shareService.mu.RUnlock()

	if !exists {
		Error(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}

	// 检查是否过期
	if time.Now().After(link.ExpiresAt) {
		shareService.mu.Lock()
		delete(shareService.links, token)
		shareService.mu.Unlock()
		shareService.save()
		Error(w, http.StatusGone, "分享链接已过期")
		return
	}

	// 检查密码
	if link.Password != "" {
		password := r.URL.Query().Get("password")
		if password != link.Password {
			Error(w, http.StatusForbidden, "密码错误")
			return
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(link.FilePath); err != nil {
		Error(w, http.StatusNotFound, "文件不存在")
		return
	}

	// 增加下载计数
	shareService.mu.Lock()
	link.DownloadCount++
	shareService.mu.Unlock()

	// 发送文件
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", link.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, link.FilePath)
}

// sharePageData 预览页面数据
type sharePageData struct {
	FileName      string
	FileSize      string
	ExpiresAt     string
	ExpiresAtISO  string
	DownloadCount int
	Token         string
	HasPassword   bool
}

// serveSharePage 显示分享预览页面
func (h *Handler) serveSharePage(w http.ResponseWriter, r *http.Request, token string) {
	// 查找分享链接
	shareService.mu.RLock()
	link, exists := shareService.links[token]
	shareService.mu.RUnlock()

	if !exists || time.Now().After(link.ExpiresAt) {
		// 显示过期页面
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(shareExpiredPageHTML))
		return
	}

	// 准备数据
	data := sharePageData{
		FileName:      link.FileName,
		FileSize:      formatFileSize(link.FileSize),
		ExpiresAt:     link.ExpiresAt.Format("2006-01-02 15:04"),
		ExpiresAtISO:  link.ExpiresAt.Format(time.RFC3339),
		DownloadCount: link.DownloadCount,
		Token:         link.Token,
		HasPassword:   link.Password != "",
	}

	// 解析并执行模板
	tmpl, err := template.New("share").Parse(sharePageHTML)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// formatFileSize 格式化文件大小
func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// listShareLinks 列出所有分享链接
func (h *Handler) listShareLinks(w http.ResponseWriter, r *http.Request) {
	// 先清理过期链接
	shareService.cleanExpired()

	shareService.mu.RLock()
	defer shareService.mu.RUnlock()

	// 构建列表
	links := make([]map[string]interface{}, 0)
	for _, link := range shareService.links {
		links = append(links, map[string]interface{}{
			"token":         link.Token,
			"fileName":      link.FileName,
			"fileSize":      link.FileSize,
			"expiresAt":     link.ExpiresAt.Format(time.RFC3339),
			"createdAt":     link.CreatedAt.Format(time.RFC3339),
			"downloadCount": link.DownloadCount,
		})
	}

	Success(w, map[string]interface{}{
		"links": links,
	})
}

// deleteShareLink 删除分享链接
func (h *Handler) deleteShareLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Token string `json:"token"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	shareService.mu.Lock()
	delete(shareService.links, req.Token)
	shareService.mu.Unlock()
	shareService.save()

	SuccessMessage(w, "已删除")
}

// generateShareToken 生成分享 token
func generateShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// 分享过期页面 HTML
const shareExpiredPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>链接已失效 - 像素兽</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: linear-gradient(135deg, #0c0a09 0%, #1c1917 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; color: #fafaf9; }
        .container { text-align: center; padding: 40px; }
        .icon { font-size: 64px; margin-bottom: 24px; }
        h1 { font-size: 24px; margin-bottom: 12px; }
        p { color: #78716c; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">⚠️</div>
        <h1>分享链接已失效</h1>
        <p>此链接不存在或已过期</p>
    </div>
</body>
</html>`

// 分享预览页面 HTML 模板
const sharePageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.FileName}} - 像素兽文件分享</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: linear-gradient(135deg, #0c0a09 0%, #1c1917 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; color: #fafaf9; padding: 20px; }
        .card { background: #1c1917; border: 1px solid #44403c; border-radius: 20px; padding: 32px 40px; max-width: 520px; width: 100%; }
        .header { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
        .logo { font-size: 32px; }
        .brand { display: flex; flex-direction: column; }
        .brand-name { font-size: 18px; font-weight: 600; color: #f97316; }
        .brand-desc { font-size: 12px; color: #78716c; }
        .file-section { display: flex; align-items: center; gap: 20px; padding: 20px; background: #0c0a09; border-radius: 12px; margin-bottom: 20px; }
        .file-icon { font-size: 48px; }
        .file-info { flex: 1; min-width: 0; }
        .file-name { font-size: 18px; font-weight: 600; color: #fafaf9; margin-bottom: 4px; word-break: break-all; }
        .file-size { font-size: 13px; color: #78716c; }
        .info-row { display: flex; gap: 24px; margin-bottom: 20px; }
        .info-item { flex: 1; }
        .info-label { font-size: 12px; color: #78716c; margin-bottom: 4px; }
        .info-value { font-size: 14px; color: #fafaf9; font-family: -apple-system, BlinkMacSystemFont, sans-serif; }
        .info-value.highlight { color: #f97316; font-weight: 600; }
        .password-section { margin-bottom: 16px; }
        .password-input { width: 100%; padding: 12px 16px; border: 1px solid #44403c; border-radius: 10px; background: #0c0a09; color: #fafaf9; font-size: 14px; }
        .password-input:focus { outline: none; border-color: #f97316; }
        .password-input::placeholder { color: #57534e; }
        .download-btn { display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; padding: 16px; background: linear-gradient(135deg, #f97316, #fb923c); color: white; border: none; border-radius: 12px; font-size: 16px; font-weight: 600; cursor: pointer; transition: all 0.2s; }
        .download-btn:hover { transform: translateY(-2px); box-shadow: 0 8px 24px rgba(249, 115, 22, 0.35); }
        .download-btn:active { transform: translateY(0); }
        .error-msg { color: #ef4444; font-size: 13px; text-align: center; margin-top: 12px; display: none; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #57534e; }
        @media (max-width: 480px) {
            .card { padding: 24px; }
            .info-row { flex-direction: column; gap: 12px; }
            .file-section { padding: 16px; }
            .file-icon { font-size: 36px; }
            .file-name { font-size: 16px; }
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="header">
            <div class="logo">🪶</div>
            <div class="brand">
                <span class="brand-name">像素兽文件分享</span>
                <span class="brand-desc">安全便捷的文件传输服务</span>
            </div>
        </div>
        
        <div class="file-section">
            <div class="file-icon">📄</div>
            <div class="file-info">
                <div class="file-name">{{.FileName}}</div>
                <div class="file-size">{{.FileSize}}</div>
            </div>
        </div>
        
        <div class="info-row">
            <div class="info-item">
                <div class="info-label">过期时间</div>
                <div class="info-value">{{.ExpiresAt}}</div>
            </div>
            <div class="info-item">
                <div class="info-label">剩余时间</div>
                <div class="info-value highlight" id="remaining">计算中...</div>
            </div>
            <div class="info-item">
                <div class="info-label">下载次数</div>
                <div class="info-value">{{.DownloadCount}} 次</div>
            </div>
        </div>
        
        {{if .HasPassword}}
        <div class="password-section">
            <input type="password" class="password-input" id="password" placeholder="🔒 请输入访问密码">
        </div>
        {{end}}
        
        <button class="download-btn" id="downloadBtn">
            <span>📥</span> 下载文件
        </button>
        <div class="error-msg" id="errorMsg">密码错误</div>
        
        <div class="footer">由 像素兽 PixelBeast 提供技术支持</div>
    </div>
    
    <script>
        const token = '{{.Token}}';
        const hasPassword = {{.HasPassword}};
        const expiresAt = new Date('{{.ExpiresAtISO}}').getTime();
        
        // 从URL参数自动填充提取码
        const urlParams = new URLSearchParams(window.location.search);
        const pwdParam = urlParams.get('pwd');
        if (pwdParam && hasPassword) {
            const pwdInput = document.getElementById('password');
            if (pwdInput) {
                pwdInput.value = pwdParam;
            }
        }
        
        function updateRemaining() {
            const now = Date.now();
            const diff = expiresAt - now;
            const el = document.getElementById('remaining');
            if (diff <= 0) { el.textContent = '已过期'; return; }
            const days = Math.floor(diff / 86400000);
            const hours = Math.floor((diff % 86400000) / 3600000);
            const mins = Math.floor((diff % 3600000) / 60000);
            if (days > 0) el.textContent = days + '天' + hours + '小时';
            else if (hours > 0) el.textContent = hours + '小时' + mins + '分';
            else el.textContent = mins + '分钟';
        }
        updateRemaining();
        setInterval(updateRemaining, 60000);
        
        document.getElementById('downloadBtn').addEventListener('click', function() {
            let url = '/s/' + token + '/download';
            const errorEl = document.getElementById('errorMsg');
            errorEl.style.display = 'none';
            
            if (hasPassword) {
                const pwd = document.getElementById('password').value;
                if (!pwd) {
                    errorEl.textContent = '请输入访问密码';
                    errorEl.style.display = 'block';
                    return;
                }
                url += '?password=' + encodeURIComponent(pwd);
            }
            
            fetch(url, { method: 'HEAD' }).then(resp => {
                if (resp.ok) window.location.href = url;
                else { errorEl.textContent = '密码错误'; errorEl.style.display = 'block'; }
            }).catch(() => { window.location.href = url; });
        });
    </script>
</body>
</html>`
