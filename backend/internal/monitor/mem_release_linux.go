//go:build linux

package monitor

import (
	"os"
	"syscall"
)

// FreeSystemMemory 释放系统级内存（Linux）
// 1. sync 刷写脏页
// 2. drop_caches 清除页面缓存/目录项/inode
// 3. 尝试 compact 压缩内存碎片
func FreeSystemMemory() error {
	// 1. 刷写脏页到磁盘
	syscall.Sync()

	// 2. 清除页面缓存（1=页面缓存, 2=目录项和inode, 3=全部）
	if err := os.WriteFile("/proc/sys/vm/drop_caches", []byte("3"), 0200); err != nil {
		// 非 root 可能没权限，降级尝试只清除页面缓存
		_ = os.WriteFile("/proc/sys/vm/drop_caches", []byte("1"), 0200)
	}

	// 3. 压缩内存碎片（需要 root，失败忽略）
	_ = os.WriteFile("/proc/sys/vm/compact_memory", []byte("1"), 0200)

	return nil
}
