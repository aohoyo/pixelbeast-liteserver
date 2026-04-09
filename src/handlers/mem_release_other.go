//go:build !linux && !darwin && !windows

package handlers

// FreeSystemMemory 释放系统级内存（不支持的平台）
func FreeSystemMemory() error {
	return nil
}
