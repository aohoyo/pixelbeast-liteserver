//go:build !windows

package admin

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/host"
)

// getSystemUptime 获取系统运行时长（缓存 BootTime）
func getSystemUptime() time.Duration {
	var bootTime int64
	once := func() {
		info, err := host.Info()
		if err == nil && info.BootTime > 0 {
			bootTime = int64(info.BootTime)
		}
	}

	systemBootOnce.Do(once)
	if bootTime <= 0 {
		return 0
	}
	return time.Since(time.Unix(bootTime, 0))
}

var systemBootOnce sync.Once
