//go:build darwin

package monitor

import (
	"fmt"
	"os/exec"
)

// FreeSystemMemory 释放系统级内存（macOS）
// 使用 purge 命令清除不活跃内存
func FreeSystemMemory() error {
	path, err := exec.LookPath("purge")
	if err != nil {
		return fmt.Errorf("purge 命令不可用（需安装 Xcode Command Line Tools）")
	}
	return exec.Command(path).Run()
}
