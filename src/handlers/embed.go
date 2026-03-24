package handlers

import (
	"io/fs"
	"os"
)

// GetStaticFS 获取静态文件系统
// 注意：此函数已废弃，请使用 main.go 中的 getStaticFS()
// 保留此函数是为了向后兼容
func GetStaticFS() fs.FS {
	// 从文件系统读取
	if _, err := os.Stat("src/static/admin"); err == nil {
		return os.DirFS("src/static/admin")
	}
	return os.DirFS("src/static/admin")
}
