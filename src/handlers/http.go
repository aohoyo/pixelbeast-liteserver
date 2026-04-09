package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	Root   string
	SiteID string
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(root string, siteID string) *HTTPServer {
	return &HTTPServer{Root: root, SiteID: siteID}
}

// ServeHTTP 处理HTTP请求
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// 清理路径
	path := filepath.Clean(r.URL.Path)

	// 防止目录遍历攻击
	if strings.Contains(path, "..") {
		s.logHTTPError(r.Method, r.URL.Path, r.RemoteAddr, "目录遍历攻击", http.StatusForbidden)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 构建完整路径
	fullPath := filepath.Join(s.Root, path)

	// 安全检查：确保路径在根目录内
	absRoot, err := filepath.Abs(s.Root)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(absPath, absRoot) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 检查文件或目录是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logHTTPError(r.Method, r.URL.Path, r.RemoteAddr, "文件不存在", http.StatusNotFound)
			http.NotFound(w, r)
			return
		}
		s.logHTTPError(r.Method, r.URL.Path, r.RemoteAddr, err.Error(), http.StatusInternalServerError)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 如果是目录，尝试提供index.html或目录列表
	if info.IsDir() {
		indexPath := filepath.Join(absPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			s.logRequest(r, start, http.StatusOK)
			return
		}

		// 生成目录列表
		s.serveDirectory(w, r, absPath, path)
		s.logRequest(r, start, http.StatusOK)
		return
	}

	// 提供文件
	http.ServeFile(w, r, absPath)
	s.logRequest(r, start, http.StatusOK)
}

// serveDirectory 生成目录列表
func (s *HTTPServer) serveDirectory(w http.ResponseWriter, r *http.Request, dirPath, urlPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 生成HTML
	io.WriteString(w, `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>目录列表 - `+urlPath+`</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 40px auto; padding: 20px; background: #f5f5f5; }
        h1 { color: #333; border-bottom: 2px solid #667eea; padding-bottom: 10px; }
        ul { list-style: none; padding: 0; }
        li { padding: 10px; margin: 5px 0; background: white; border-radius: 5px; }
        li:hover { background: #e8e8e8; }
        a { color: #667eea; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .icon { margin-right: 10px; }
        .size { color: #888; float: right; font-size: 0.9em; }
        .parent { margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>📁 `+urlPath+`</h1>
`)

	// 父目录链接
	if urlPath != "/" {
		parentPath := filepath.Dir(urlPath)
		if parentPath == "." {
			parentPath = "/"
		}
		io.WriteString(w, `<div class="parent"><a href="`+parentPath+`">📁 ..</a></div>`)
	}

	io.WriteString(w, "<ul>\n")

	for _, entry := range entries {
		name := entry.Name()
		href := filepath.Join(urlPath, name)
		if strings.HasPrefix(href, "./") {
			href = href[2:]
		}
		if !strings.HasPrefix(href, "/") {
			href = "/" + href
		}

		var icon string
		var sizeStr string

		if entry.IsDir() {
			icon = "📁"
		} else {
			icon = "📄"
			if info, err := entry.Info(); err == nil {
				sizeStr = formatSize(info.Size())
			}
		}

		io.WriteString(w, `<li><span class="icon">`+icon+`</span><a href="`+href+`">`+name+`</a><span class="size">`+sizeStr+`</span></li>`+"\n")
	}

	io.WriteString(w, `</ul>
</body>
</html>`)
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// logHTTPError 记录HTTP错误（同时写入通用和按站点日志）
func (s *HTTPServer) logHTTPError(method, path, remoteAddr, errMsg string, statusCode int) {
	LogHTTPError(method, path, remoteAddr, errMsg, statusCode)
	if s.SiteID != "" {
		LogHTTPErrorBySite(s.SiteID, method, path, remoteAddr, errMsg, statusCode)
	}
}

// logRequest 记录请求日志（同时写入通用和按站点日志）
func (s *HTTPServer) logRequest(r *http.Request, start time.Time, status int) {
	duration := time.Since(start)
	LogHTTPAccess(r.Method, r.URL.Path, r.RemoteAddr, status, duration)
	if s.SiteID != "" {
		LogHTTPAccessBySite(s.SiteID, r.Method, r.URL.Path, r.RemoteAddr, status, duration)
	}
}
