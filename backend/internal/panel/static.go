package panel

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

var staticFS fs.FS

func SetStaticFS(fsys fs.FS) {
	staticFS = fsys
	initFileCache(fsys)
}

// mimeByExt 按扩展名推断 Content-Type（Vue 构建产物）
func mimeByExt(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

// serveVueAssets 提供 Vue 构建产物（/assets/ 目录）
// 兼容原生版与 Vue 版：若 assets/ 不存在则回退到原生 admin 的对应路径
func (h *Handler) serveVueAssets(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/assets/")
	file = strings.SplitN(file, "?", 2)[0]
	if file == "" {
		http.NotFound(w, r)
		return
	}

	// 优先从 assets/ 读（Vue 版）
	if data, err := fs.ReadFile(staticFS, "assets/"+file); err == nil {
		w.Header().Set("Content-Type", mimeByExt(file))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(data)
		return
	}

	http.NotFound(w, r)
}
