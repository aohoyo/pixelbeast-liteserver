//go:build !windows

package admin

import "os"

// isAdmin 非 Windows 平台检测 root 权限
func isAdmin() bool {
	return os.Getuid() == 0
}

// canWriteDir 检测是否可写入指定目录
func canWriteDir(dir string) bool {
	return true
}
