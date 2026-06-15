package frontend

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// Vue 构建产物（npm run build 生成）。
// 开发时若 dist/ 不存在，fallback 到磁盘读取或 stub。
//
//go:embed dist
var embeddedVueFS embed.FS

// GetStaticFS 获取前端静态文件系统。
//
// 优先级：
//  1. 磁盘上的 frontend/vue/dist（开发模式，支持热更新）
//  2. 嵌入的 dist（生产二进制）
//
// 设计：前端自己负责打包自己的资源，后端通过此函数获取 FS，
// 不需要知道前端文件在磁盘上的具体位置——前后端解耦。
func GetStaticFS() fs.FS {
	cwd, _ := os.Getwd()

	// 优先从磁盘读取（开发模式，支持改完 npm run build 后即时生效）
	vueDist := filepath.Join(cwd, "frontend", "vue", "dist")
	if _, err := os.Stat(vueDist); err == nil {
		return os.DirFS(vueDist)
	}

	// 生产模式：使用嵌入的 dist
	sub, _ := fs.Sub(embeddedVueFS, "dist")
	return sub
}
