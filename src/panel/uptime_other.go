//go:build !windows

package panel

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/host"
)

var (
	systemBootOnce sync.Once
	systemBootTime int64
)

// getSystemUptime 获取系统运行时长（缓存 BootTime）
func getSystemUptime() time.Duration {
	systemBootOnce.Do(func() {
		info, err := host.Info()
		if err == nil && info.BootTime > 0 {
			systemBootTime = int64(info.BootTime)
		}
	})
	if systemBootTime <= 0 {
		return 0
	}
	return time.Since(time.Unix(systemBootTime, 0))
}
