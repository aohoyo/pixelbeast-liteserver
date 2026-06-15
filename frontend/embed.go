package frontend

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// 嵌入原生版前端（始终存在，作为兜底）
//
//go:embed admin
var embeddedAdminFS embed.FS

// GetStaticFS 获取前端静态文件系统。
//
// 优先级：
//  1. Vue 构建产物 frontend/vue/dist（开发/生产磁盘模式，支持热更新）
//  2. 原生版 frontend/admin（磁盘）
//  3. 嵌入的原生版（生产二进制，兜底）
//
// 设计：前端自己负责打包自己的资源，后端通过此函数获取 FS，
// 不需要知道前端文件在磁盘上的具体位置——前后端解耦。
func GetStaticFS() fs.FS {
	cwd, _ := os.Getwd()

	// 优先使用 Vue 构建产物
	vueDist := filepath.Join(cwd, "frontend", "vue", "dist")
	if _, err := os.Stat(vueDist); err == nil {
		return os.DirFS(vueDist)
	}
	// 退回原生版（磁盘）
	adminDir := filepath.Join(cwd, "frontend", "admin")
	if _, err := os.Stat(adminDir); err == nil {
		return os.DirFS(adminDir)
	}

	// 生产模式：嵌入的原生版（兜底）
	sub, _ := fs.Sub(embeddedAdminFS, "admin")
	return sub
}
