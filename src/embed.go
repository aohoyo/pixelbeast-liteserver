package embedfs

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed static/admin
var embeddedFS embed.FS

// GetStaticFS 获取静态文件系统
// 开发模式：从 src/static/admin/ 读取（支持热更新）
// 生产模式：从嵌入的 FS 读取（单二进制）
func GetStaticFS() fs.FS {
	// 优先从文件系统读取（开发模式）
	if _, err := os.Stat("src/static/admin"); err == nil {
		return os.DirFS("src/static/admin")
	}

	// 生产模式：使用嵌入的文件系统
	sub, _ := fs.Sub(embeddedFS, "static/admin")
	return sub
}
