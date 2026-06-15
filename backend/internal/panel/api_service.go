package panel

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"pixelbeast/backend/internal/logger"
)

const (
	serviceName    = "PixelBeast"
	servicePlistID = "com.pixelbeast.liteserver"
)

// getExeCommand 获取当前可执行文件的完整启动命令（exe路径 + -config 参数）
func getExeCommand() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取可执行文件路径失败: %v", err)
	}
	// 解析为绝对路径
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		absPath = exePath
	}
	return absPath, nil
}

// ==================== AutoStart Status ====================

func (h *Handler) getAutoStartStatus(w http.ResponseWriter, r *http.Request) {
	installed := h.isAutoStartInstalled()
	platform := runtime.GOOS

	Success(w, map[string]interface{}{
		"platform":  platform,
		"installed": installed,
		"enabled":   h.ConfigManager.Server.AutoStart.Enabled,
	})
}

// ==================== Enable AutoStart ====================

func (h *Handler) enableAutoStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	username := h.getSessionUsername(r)

	var err error
	switch runtime.GOOS {
	case "windows":
		err = h.enableAutoStartWindows()
	case "linux":
		err = h.enableAutoStartLinux()
	case "darwin":
		err = h.enableAutoStartDarwin()
	default:
		BadRequest(w, "不支持的平台: "+runtime.GOOS)
		return
	}

	if err != nil {
		logger.LogPanelConfigChange(username, "开启开机自启", false)
		InternalServerErrorLog(w, err, "开启开机自启失败")
		return
	}

	// 更新配置
	h.ConfigManager.Server.AutoStart.Enabled = true
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	logger.LogPanelConfigChange(username, "开启开机自启", true)
	SuccessMessage(w, "开机自启已开启")
}

// ==================== Disable AutoStart ====================

func (h *Handler) disableAutoStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	username := h.getSessionUsername(r)

	var err error
	switch runtime.GOOS {
	case "windows":
		err = h.disableAutoStartWindows()
	case "linux":
		err = h.disableAutoStartLinux()
	case "darwin":
		err = h.disableAutoStartDarwin()
	default:
		BadRequest(w, "不支持的平台: "+runtime.GOOS)
		return
	}

	if err != nil {
		logger.LogPanelConfigChange(username, "关闭开机自启", false)
		InternalServerErrorLog(w, err, "关闭开机自启失败")
		return
	}

	// 更新配置
	h.ConfigManager.Server.AutoStart.Enabled = false
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	logger.LogPanelConfigChange(username, "关闭开机自启", true)
	SuccessMessage(w, "开机自启已关闭")
}

// ==================== Check if installed ====================

func (h *Handler) isAutoStartInstalled() bool {
	switch runtime.GOOS {
	case "windows":
		return h.isAutoStartInstalledWindows()
	case "linux":
		return h.isAutoStartInstalledLinux()
	case "darwin":
		return h.isAutoStartInstalledDarwin()
	}
	return false
}

// ==================== Windows 实现 ====================

// Windows 注册表路径: HKCU\Software\Microsoft\Windows\CurrentVersion\Run
func (h *Handler) enableAutoStartWindows() error {
	exePath, err := getExeCommand()
	if err != nil {
		return err
	}
	configDir := h.ConfigManager.ConfigDir()
	value := fmt.Sprintf(`"%s" -config "%s"`, exePath, configDir)

	// 使用 reg add 命令添加注册表项
	cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", serviceName, "/t", "REG_SZ", "/d", value, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("注册表写入失败: %s (%v)", string(output), err)
	}
	return nil
}

func (h *Handler) disableAutoStartWindows() error {
	cmd := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", serviceName, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("注册表删除失败: %s (%v)", string(output), err)
	}
	return nil
}

func (h *Handler) isAutoStartInstalledWindows() bool {
	cmd := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", serviceName)
	return cmd.Run() == nil
}

// ==================== Linux 实现 ====================

// systemd 用户服务: ~/.config/systemd/user/pixelbeast.service
func (h *Handler) linuxServiceFilePath() string {
	configDir, _ := os.UserConfigDir()
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "systemd", "user", "pixelbeast.service")
}

func (h *Handler) generateSystemdUnit() (string, error) {
	exePath, err := getExeCommand()
	if err != nil {
		return "", err
	}
	configDir := h.ConfigManager.ConfigDir()
	workDir, _ := os.Getwd()

	unit := fmt.Sprintf(`[Unit]
Description=PixelBeast LiteServer
After=network.target

[Service]
Type=simple
ExecStart=%s -config %s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, exePath, configDir, workDir)

	return unit, nil
}

func (h *Handler) enableAutoStartLinux() error {
	servicePath := h.linuxServiceFilePath()

	// 确保目录存在
	os.MkdirAll(filepath.Dir(servicePath), 0755)

	// 生成并写入 service 文件
	content, err := h.generateSystemdUnit()
	if err != nil {
		return err
	}
	if err := os.WriteFile(servicePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入服务文件失败: %v", err)
	}

	// daemon-reload + enable
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	if output, err := exec.Command("systemctl", "--user", "enable", "pixelbeast.service").CombinedOutput(); err != nil {
		return fmt.Errorf("启用服务失败: %s (%v)", strings.TrimSpace(string(output)), err)
	}

	// 确保 user lingering 开启（用户未登录时服务也能运行）
	exec.Command("loginctl", "enable-linger", "").Run()

	return nil
}

func (h *Handler) disableAutoStartLinux() error {
	exec.Command("systemctl", "--user", "disable", "pixelbeast.service").Run()
	exec.Command("systemctl", "--user", "stop", "pixelbeast.service").Run()

	servicePath := h.linuxServiceFilePath()
	os.Remove(servicePath)
	exec.Command("systemctl", "--user", "daemon-reload").Run()

	return nil
}

func (h *Handler) isAutoStartInstalledLinux() bool {
	// 检查 service 文件是否存在且已 enabled
	if _, err := os.Stat(h.linuxServiceFilePath()); err != nil {
		return false
	}
	cmd := exec.Command("systemctl", "--user", "is-enabled", "pixelbeast.service")
	output, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)) == "enabled"
}

// ==================== macOS 实现 ====================

// LaunchAgent plist: ~/Library/LaunchAgents/com.pixelbeast.liteserver.plist
func (h *Handler) darwinPlistPath() string {
	home := getHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", servicePlistID+".plist")
}

func (h *Handler) generateLaunchdPlist() (string, error) {
	exePath, err := getExeCommand()
	if err != nil {
		return "", err
	}
	configDir := h.ConfigManager.ConfigDir()
	workDir, _ := os.Getwd()

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>-config</string>
        <string>%s</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>/tmp/pixelbeast.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/pixelbeast.err</string>
</dict>
</plist>
`, servicePlistID, exePath, configDir, workDir)

	return plist, nil
}

func (h *Handler) enableAutoStartDarwin() error {
	plistPath := h.darwinPlistPath()

	// 先 unload 已有的（如果存在）
	exec.Command("launchctl", "unload", plistPath).Run()

	// 确保目录存在
	os.MkdirAll(filepath.Dir(plistPath), 0755)

	// 生成并写入 plist
	content, err := h.generateLaunchdPlist()
	if err != nil {
		return err
	}
	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 plist 文件失败: %v", err)
	}

	// load
	if output, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("加载 plist 失败: %s (%v)", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func (h *Handler) disableAutoStartDarwin() error {
	plistPath := h.darwinPlistPath()

	// unload
	exec.Command("launchctl", "unload", plistPath).Run()

	// 删除 plist 文件
	os.Remove(plistPath)

	return nil
}

func (h *Handler) isAutoStartInstalledDarwin() bool {
	plistPath := h.darwinPlistPath()
	if _, err := os.Stat(plistPath); err != nil {
		return false
	}
	// 检查 launchctl 是否已加载
	output, _ := exec.Command("launchctl", "list", servicePlistID).CombinedOutput()
	// 如果返回包含 PID 或 ExitStatus 说明已加载
	return len(strings.TrimSpace(string(output))) > 0
}
