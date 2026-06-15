package frontend

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed admin
var embeddedFS embed.FS

// GetStaticFS 获取前端静态文件系统。
// 开发模式：从磁盘读 frontend/admin（支持热更新）
// 生产模式：从嵌入的 FS 读取（单二进制）
//
// 设计：前端自己负责打包自己的资源，后端通过此函数获取 FS，
// 不需要知道前端文件在磁盘上的具体位置——前后端解耦。
func GetStaticFS() fs.FS {
	// 优先从文件系统读取（开发模式）
	if _, err := os.Stat("frontend/admin"); err == nil {
		return os.DirFS("frontend/admin")
	}

	// 生产模式：使用嵌入的文件系统
	sub, _ := fs.Sub(embeddedFS, "admin")
	return sub
}
