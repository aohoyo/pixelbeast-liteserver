package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
	LogLevelAuth
)

// LogCategory 日志分类
type LogCategory string

const (
	LogCategoryHTTP  LogCategory = "http"
	LogCategoryFTP   LogCategory = "ftp"
	LogCategoryPanel LogCategory = "panel"
)

// Logger 日志记录器
type Logger struct {
	baseDir string
	files   map[string]*os.File // 文件句柄缓存
	mu      sync.RWMutex
}

var globalLogger *Logger

// InitLogger 初始化日志记录器，创建分类目录结构
func InitLogger(logDir string) error {
	// 创建各分类目录
	categories := []string{string(LogCategoryHTTP), string(LogCategoryFTP), string(LogCategoryPanel)}
	for _, cat := range categories {
		dir := filepath.Join(logDir, cat)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// 创建旧的 access.log 和 error.log 作为兼容
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	globalLogger = &Logger{
		baseDir: logDir,
		files:   make(map[string]*os.File),
	}

	return nil
}

// Close 关闭所有日志文件
func Close() error {
	if globalLogger == nil {
		return nil
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	for name, file := range globalLogger.files {
		file.Close()
		delete(globalLogger.files, name)
	}

	return nil
}

// 获取或创建日志文件
func (l *Logger) getFile(category LogCategory, logType string) (*os.File, error) {
	// 特殊处理旧的 access.log 和 error.log（兼容性）
	if category == "" {
		if logType == "access" {
			return l.getFile(LogCategoryHTTP, "access")
		}
		if logType == "error" {
			return l.getFile(LogCategoryHTTP, "error")
		}
	}

	key := string(category) + "/" + logType

	l.mu.RLock()
	if file, exists := l.files[key]; exists {
		l.mu.RUnlock()
		return file, nil
	}
	l.mu.RUnlock()

	// 需要创建新文件
	l.mu.Lock()
	defer l.mu.Unlock()

	// 再次检查，可能其他 goroutine 已经创建
	if file, exists := l.files[key]; exists {
		return file, nil
	}

	// 创建目录和文件
	dir := filepath.Join(l.baseDir, string(category))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, logType+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	l.files[key] = file
	return file, nil
}

// 基础日志写入
func (l *Logger) write(category LogCategory, logType string, level LogLevel, format string, args ...interface{}) {
	if l == nil {
		return
	}

	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	// 控制台输出
	levelStr := ""
	switch level {
	case LogLevelWarn:
		levelStr = "[WARN] "
	case LogLevelError:
		levelStr = "[ERROR] "
	case LogLevelAuth:
		levelStr = "[AUTH] "
	}
	log.Print(levelStr + msg)

	// 文件写入
	file, err := l.getFile(category, logType)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	file.WriteString(timestamp + " " + msg + "\n")
}

// ============ 通用日志方法 ============

// Log 通用日志
func Log(category LogCategory, logType string, level LogLevel, format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.write(category, logType, level, format, args...)
	}
}

// ============ HTTP 日志 ============

// LogHTTPAccess 记录 HTTP 访问
func LogHTTPAccess(method, path, remoteAddr string, statusCode int, duration time.Duration) {
	Log(LogCategoryHTTP, "access", LogLevelInfo,
		"[HTTP] %s %s from %s -> %d (%v)", method, path, remoteAddr, statusCode, duration)
}

// LogHTTPError 记录 HTTP 错误
func LogHTTPError(method, path, remoteAddr string, errMsg string, statusCode int) {
	Log(LogCategoryHTTP, "error", LogLevelError,
		"[HTTP] %s %s from %s -> %d: %s", method, path, remoteAddr, statusCode, errMsg)
}

// ============ FTP 日志 ============

// LogFTPLogin 记录 FTP 登录
func LogFTPLogin(username, remoteAddr string, success bool, reason string) {
	level := LogLevelAuth
	if !success {
		level = LogLevelWarn
	}
	status := "成功"
	if !success {
		status = "失败"
	}
	Log(LogCategoryFTP, "access", level,
		"[FTP] 登录 %s: 用户=%s from %s (%s)", status, username, remoteAddr, reason)
}

// LogFTPCommand 记录 FTP 命令
func LogFTPCommand(username, remoteAddr, command, args string) {
	Log(LogCategoryFTP, "access", LogLevelInfo,
		"[FTP] 命令: %s %s (用户=%s from %s)", command, args, username, remoteAddr)
}

// LogFTPTransfer 记录 FTP 传输
func LogFTPTransfer(username, remoteAddr, filename, direction string, size int64, duration time.Duration, success bool) {
	status := "完成"
	if !success {
		status = "失败"
	}
	Log(LogCategoryFTP, "access", LogLevelInfo,
		"[FTP] 传输%s: %s (用户=%s, 大小=%d, 耗时=%v, %s)", direction, filename, username, size, duration, status)
}

// LogFTPError 记录 FTP 错误
func LogFTPError(username, remoteAddr, operation string, errMsg string) {
	Log(LogCategoryFTP, "error", LogLevelError,
		"[FTP] 错误: %s (用户=%s from %s): %s", operation, username, remoteAddr, errMsg)
}

// LogFTPConnection 记录 FTP 连接
func LogFTPConnection(remoteAddr string, connected bool) {
	action := "连接"
	if !connected {
		action = "断开"
	}
	Log(LogCategoryFTP, "access", LogLevelInfo,
		"[FTP] %s: %s", action, remoteAddr)
}

// ============ Panel 日志 ============

// LogPanelAccess 记录面板访问
func LogPanelAccess(username, path, remoteAddr string) {
	if username == "" {
		username = "未认证"
	}
	Log(LogCategoryPanel, "access", LogLevelInfo,
		"[Panel] 访问 %s (用户=%s from %s)", path, username, remoteAddr)
}

// LogPanelAPI 记录 API 调用
func LogPanelAPI(username, method, path string, statusCode int, duration time.Duration) {
	Log(LogCategoryPanel, "api", LogLevelInfo,
		"[Panel API] %s %s (用户=%s) -> %d (%v)", method, path, username, statusCode, duration)
}

// LogPanelAuth 记录认证事件
func LogPanelAuth(event, username, remoteAddr string, success bool, reason string) {
	level := LogLevelAuth
	if !success {
		level = LogLevelWarn
	}
	status := "成功"
	if !success {
		status = "失败"
	}
	Log(LogCategoryPanel, "auth", level,
		"[Panel] %s %s: 用户=%s from %s (%s)", event, status, username, remoteAddr, reason)
}

// LogPanelFileOp 记录文件操作
func LogPanelFileOp(username, operation, targetPath string, success bool) {
	status := "成功"
	if !success {
		status = "失败"
	}
	level := LogLevelInfo
	if !success {
		level = LogLevelWarn
	}
	Log(LogCategoryPanel, "api", level,
		"[Panel] 文件操作: %s %s (用户=%s, %s)", operation, targetPath, username, status)
}

// LogPanelConfig 记录配置变更
func LogPanelConfigChange(username, configPath string, success bool) {
	status := "成功"
	if !success {
		status = "失败"
	}
	Log(LogCategoryPanel, "api", LogLevelInfo,
		"[Panel] 配置变更: %s (用户=%s, %s)", configPath, username, status)
}

// LogPanelService 记录服务控制
func LogPanelService(username, service, action string, success bool) {
	status := "成功"
	if !success {
		status = "失败"
	}
	Log(LogCategoryPanel, "api", LogLevelInfo,
		"[Panel] 服务%s: %s (用户=%s, %s)", action, service, username, status)
}

// ============ 向后兼容 ============

// LogAccess 记录访问日志（兼容旧代码，写入 http/access.log）
func LogAccess(format string, args ...interface{}) {
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	log.Print(msg)

	if globalLogger != nil {
		globalLogger.write(LogCategoryHTTP, "access", LogLevelInfo, msg)
	}
}

// LogError 记录错误日志（兼容旧代码，写入 http/error.log）
func LogError(format string, args ...interface{}) {
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	log.Print("[ERROR] " + msg)

	if globalLogger != nil {
		globalLogger.write(LogCategoryHTTP, "error", LogLevelError, msg)
	}
}

// GetLogFilePath 获取日志文件路径（用于 API 读取）
func GetLogFilePath(category, logType string) string {
	if globalLogger == nil {
		return ""
	}

	// 安全检查：防止路径遍历
	if strings.Contains(category, "..") || strings.Contains(logType, "..") {
		return ""
	}

	// 兼容旧的单层日志
	if category == "" {
		if logType == "access" {
			logType = "http/access"
		} else if logType == "error" {
			logType = "http/error"
		} else {
			return ""
		}
		parts := strings.Split(logType, "/")
		if len(parts) == 2 {
			category = parts[0]
			logType = parts[1]
		}
	}

	return filepath.Join(globalLogger.baseDir, category, logType+".log")
}
