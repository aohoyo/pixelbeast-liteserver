//go:build darwin

package monitor

import (
	"fmt"
	"os/exec"
)

// FreeSystemMemory 释放系统级内存（macOS）
// 1. purge 命令清除不活跃内存
// 2. memory_pressure 通知系统释放内存
func FreeSystemMemory() error {
	var lastErr error

	// 1. purge 命令
	if path, err := exec.LookPath("purge"); err == nil {
		if err := exec.Command(path).Run(); err != nil {
			lastErr = err
		}
	} else {
		lastErr = fmt.Errorf("purge 命令不可用（需安装 Xcode Command Line Tools）")
	}

	// 2. memory_pressure（macOS 10.12+）
	if path, err := exec.LookPath("memory_pressure"); err == nil {
		_ = exec.Command(path).Run()
	}

	return lastErr
}
