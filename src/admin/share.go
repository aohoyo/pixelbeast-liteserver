package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	ExpiresAt     time.Time `json:"expiresAt"`
	CreatedAt     time.Time `json:"createdAt"`
	DownloadCount int       `json:"downloadCount"`
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

	// 检查文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		BadRequest(w, "文件不存在")
		return
	}

	// 创建分享链接
	link := &ShareLink{
		Token:         token,
		FilePath:      filePath,
		FileName:      req.Name,
		ExpiresAt:     time.Now().Add(time.Duration(req.Duration) * time.Hour),
		CreatedAt:     time.Now(),
		DownloadCount: 0,
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
	})
}

// downloadSharedFile 下载分享文件
func (h *Handler) downloadSharedFile(w http.ResponseWriter, r *http.Request) {
	// 从 URL 获取 token（支持 /s/ 和 /share/ 两种路径）
	token := strings.TrimPrefix(r.URL.Path, "/s/")
	if token == r.URL.Path {
		token = strings.TrimPrefix(r.URL.Path, "/share/")
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
		// 删除过期链接并保存
		shareService.mu.Lock()
		delete(shareService.links, token)
		shareService.mu.Unlock()
		shareService.save()
		Error(w, http.StatusGone, "分享链接已过期")
		return
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