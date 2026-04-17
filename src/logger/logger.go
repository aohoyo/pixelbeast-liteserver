package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pixelbeast/src/config"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
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

var nameToLevel = map[string]LogLevel{
	"debug": LogLevelDebug,
	"info":  LogLevelInfo,
	"warn":  LogLevelWarn,
	"error": LogLevelError,
	"auth":  LogLevelAuth,
}

// Logger 日志记录器
type Logger struct {
	baseDir     string
	files       map[string]*os.File
	mu          sync.RWMutex
	config      *config.LogConfig
	currentDate string
	stopCleanup chan struct{}
}

var globalLogger *Logger

// InitLogger 初始化日志记录器
func InitLogger(logDir string) error {
	return InitLoggerWithConfig(logDir, nil)
}

// InitLoggerWithConfig 带配置初始化日志记录器
func InitLoggerWithConfig(logDir string, cfg *config.LogConfig) error {
	// 创建各分类目录
	categories := []string{string(LogCategoryHTTP), string(LogCategoryFTP), string(LogCategoryPanel)}
	for _, cat := range categories {
		dir := filepath.Join(logDir, cat)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if cfg == nil {
		cfg = &config.LogConfig{
			RetentionDays: 30,
			MaxSizeMB:     100,
			CompressDays:  7,
			CleanupHour:   3,
			Level:         "info",
		}
	}

	globalLogger = &Logger{
		baseDir:     logDir,
		files:       make(map[string]*os.File),
		config:      cfg,
		currentDate: time.Now().Format("2006-01-02"),
		stopCleanup: make(chan struct{}),
	}

	// 启动定时清理
	go globalLogger.cleanupScheduler()

	// 启动日期检查（用于日志轮转）
	go globalLogger.dateChecker()

	return nil
}

// SetConfig 更新日志配置
func SetLogConfig(cfg *config.LogConfig) {
	if globalLogger != nil && cfg != nil {
		globalLogger.mu.Lock()
		globalLogger.config = cfg
		globalLogger.mu.Unlock()
	}
}

// Close 关闭所有日志文件
func Close() error {
	if globalLogger == nil {
		return nil
	}

	close(globalLogger.stopCleanup)

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	for name, file := range globalLogger.files {
		file.Close()
		delete(globalLogger.files, name)
	}

	return nil
}

// getLevel 获取分类的日志级别
func (l *Logger) getLevel(category LogCategory) LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 先检查分类级别
	if l.config != nil && l.config.Levels != nil {
		if levelStr, ok := l.config.Levels[string(category)]; ok {
			if level, ok := nameToLevel[levelStr]; ok {
				return level
			}
		}
	}

	// 使用全局级别
	if l.config != nil {
		if level, ok := nameToLevel[l.config.Level]; ok {
			return level
		}
	}

	return LogLevelInfo
}

// shouldLog 检查是否应该记录该级别日志
func (l *Logger) shouldLog(category LogCategory, level LogLevel) bool {
	configLevel := l.getLevel(category)

	// Auth 级别特殊处理，与 Info 同级
	checkLevel := level
	if level == LogLevelAuth {
		checkLevel = LogLevelInfo
	}

	return checkLevel >= configLevel
}

// getFile 获取或创建日志文件（带日期轮转）
func (l *Logger) getFile(category LogCategory, logType string) (*os.File, error) {
	now := time.Now()
	currentDate := now.Format("2006-01-02")

	// 带日期的 key
	key := fmt.Sprintf("%s/%s/%s", category, logType, currentDate)

	l.mu.RLock()
	if file, exists := l.files[key]; exists {
		l.mu.RUnlock()
		return file, nil
	}
	l.mu.RUnlock()

	// 需要创建新文件
	l.mu.Lock()
	defer l.mu.Unlock()

	// 再次检查
	if file, exists := l.files[key]; exists {
		return file, nil
	}

	// 创建目录
	dir := filepath.Join(l.baseDir, string(category))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// 创建当前日志文件（不带日期后缀，方便查看）
	currentPath := filepath.Join(dir, logType+".log")
	file, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	l.files[key] = file
	return file, nil
}

// rotate 日志轮转
func (l *Logger) rotate(category LogCategory, logType string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Join(l.baseDir, string(category))
	currentPath := filepath.Join(dir, logType+".log")

	// 检查当前文件是否存在
	info, err := os.Stat(currentPath)
	if os.IsNotExist(err) {
		return nil
	}

	// 检查文件大小
	maxSize := int64(100 * 1024 * 1024) // 默认 100MB
	if l.config != nil && l.config.MaxSizeMB > 0 {
		maxSize = int64(l.config.MaxSizeMB) * 1024 * 1024
	}

	if info.Size() < maxSize {
		return nil
	}

	// 关闭旧文件句柄
	key := fmt.Sprintf("%s/%s/%s", category, logType, l.currentDate)
	if file, exists := l.files[key]; exists {
		file.Close()
		delete(l.files, key)
	}

	// 重命名文件
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.%s.log", logType, time.Now().Format("2006-01-02_150405")))
	if err := os.Rename(currentPath, backupPath); err != nil {
		return err
	}

	log.Printf("[Logger] 日志轮转: %s -> %s", currentPath, backupPath)
	return nil
}

// write 基础日志写入
func (l *Logger) write(category LogCategory, logType string, level LogLevel, format string, args ...interface{}) {
	if l == nil {
		return
	}

	// 检查日志级别
	if !l.shouldLog(category, level) {
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
	case LogLevelDebug:
		levelStr = "[DEBUG] "
	case LogLevelWarn:
		levelStr = "\033[33m[WARN]\033[0m "
	case LogLevelError:
		levelStr = "\033[31m[ERROR]\033[0m "
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

// cleanupScheduler 定时清理调度器
func (l *Logger) cleanupScheduler() {
	// 计算下次清理时间
	nextCleanup := func() time.Duration {
		now := time.Now()
		cleanupHour := 3
		if l.config != nil && l.config.CleanupHour >= 0 && l.config.CleanupHour <= 23 {
			cleanupHour = l.config.CleanupHour
		}

		next := time.Date(now.Year(), now.Month(), now.Day(), cleanupHour, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		return time.Until(next)
	}

	for {
		select {
		case <-l.stopCleanup:
			return
		case <-time.After(nextCleanup()):
			l.doCleanup()
		}
	}
}

// dateChecker 日期检查器（用于日志轮转）
func (l *Logger) dateChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCleanup:
			return
		case <-ticker.C:
			now := time.Now().Format("2006-01-02")
			if now != l.currentDate {
				l.mu.Lock()
				l.currentDate = now
				// 关闭所有旧的文件句柄
				for key, file := range l.files {
					file.Close()
					delete(l.files, key)
				}
				l.mu.Unlock()
			}
		}
	}
}

// doCleanup 执行日志清理
func (l *Logger) doCleanup() {
	l.mu.RLock()
	retentionDays := l.config.RetentionDays
	compressDays := l.config.CompressDays
	l.mu.RUnlock()

	if retentionDays <= 0 {
		retentionDays = 30
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	compressCutoff := time.Now().AddDate(0, 0, -compressDays)

	categories := []string{string(LogCategoryHTTP), string(LogCategoryFTP), string(LogCategoryPanel)}

	for _, cat := range categories {
		dir := filepath.Join(l.baseDir, cat)
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			name := f.Name()
			// 跳过当前日志文件（不含日期的视为当前日志）
			isCurrentLog := strings.HasSuffix(name, ".log") && !strings.Contains(strings.TrimSuffix(name, ".log"), ".")
			isSiteUser := (strings.HasPrefix(name, "site-") || strings.HasPrefix(name, "user-")) && strings.HasSuffix(name, ".log") && !strings.Contains(strings.TrimSuffix(name, ".log"), ".")
			if isCurrentLog || isSiteUser {
				continue
			}

			info, err := f.Info()
			if err != nil {
				continue
			}

			filePath := filepath.Join(dir, name)

			// 删除过期日志
			if info.ModTime().Before(cutoff) {
				os.Remove(filePath)
				log.Printf("[Logger] 清理过期日志: %s", filePath)
				continue
			}

			// 压缩旧日志
			if info.ModTime().Before(compressCutoff) && !strings.HasSuffix(name, ".gz") {
				if err := l.compressFile(filePath); err == nil {
					log.Printf("[Logger] 压缩日志: %s", filePath)
				}
			}
		}
	}
}

// compressFile 压缩文件
func (l *Logger) compressFile(path string) error {
	// 读取原文件
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	// 创建压缩文件
	out, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	defer out.Close()

	// 写入压缩数据
	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return err
	}
	gz.Close()

	// 删除原文件
	return os.Remove(path)
}

// ============ 通用日志方法 ============

// Log 通用日志
func Log(category LogCategory, logType string, level LogLevel, format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.write(category, logType, level, format, args...)
	}
}

// ============ HTTP 日志（统一写 access.log / error.log）============

// LogHTTPAccess 记录 HTTP 访问
func LogHTTPAccess(method, path, remoteAddr string, statusCode int, duration time.Duration) {
	Log(LogCategoryHTTP, "access", LogLevelInfo,
		"[HTTP] %s %s from %s -> %d (%v)", method, path, remoteAddr, statusCode, duration)
}

// LogHTTPAccessBySite 按站点记录 HTTP 访问（写入统一 access.log，带站点标记）
func LogHTTPAccessBySite(siteID, method, path, remoteAddr string, statusCode int, duration time.Duration) {
	if siteID == "" {
		siteID = "default"
	}
	Log(LogCategoryHTTP, "access", LogLevelInfo,
		"[HTTP] [%s] %s %s from %s -> %d (%v)", siteID, method, path, remoteAddr, statusCode, duration)
}

// LogHTTPError 记录 HTTP 错误
func LogHTTPError(method, path, remoteAddr string, errMsg string, statusCode int) {
	Log(LogCategoryHTTP, "error", LogLevelError,
		"[HTTP] %s %s from %s -> %d: %s", method, path, remoteAddr, statusCode, errMsg)
}

// LogHTTPErrorBySite 按站点记录 HTTP 错误（写入统一 error.log，带站点标记）
func LogHTTPErrorBySite(siteID, method, path, remoteAddr string, errMsg string, statusCode int) {
	if siteID == "" {
		siteID = "default"
	}
	Log(LogCategoryHTTP, "error", LogLevelError,
		"[HTTP] [%s] %s %s from %s -> %d: %s", siteID, method, path, remoteAddr, statusCode, errMsg)
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
		"[FTP] [%s] 登录 %s from %s (%s)", username, status, remoteAddr, reason)
}

// LogFTPCommand 记录 FTP 命令
func LogFTPCommand(username, remoteAddr, command, args string) {
	Log(LogCategoryFTP, "access", LogLevelInfo,
		"[FTP] [%s] 命令: %s %s from %s", username, command, args, remoteAddr)
}

// LogFTPTransfer 记录 FTP 传输
func LogFTPTransfer(username, remoteAddr, filename, direction string, size int64, duration time.Duration, success bool) {
	status := "完成"
	if !success {
		status = "失败"
	}
	Log(LogCategoryFTP, "access", LogLevelInfo,
		"[FTP] [%s] 传输%s: %s (大小=%d, 耗时=%v, %s)", username, direction, filename, size, duration, status)
}

// LogFTPError 记录 FTP 错误
func LogFTPError(username, remoteAddr, operation string, errMsg string) {
	Log(LogCategoryFTP, "error", LogLevelError,
		"[FTP] [%s] 错误: %s from %s: %s", username, operation, remoteAddr, errMsg)
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

// ============ Panel 日志（操作日志 operation.log / 登录日志 login.log / 运行日志 runtime.log）============

// LogPanelOperation 记录操作日志 → operation.log
func LogPanelOperation(level LogLevel, format string, args ...interface{}) {
	Log(LogCategoryPanel, "operation", level, format, args...)
}

// LogPanelLogin 记录登录日志 → login.log
func LogPanelLogin(level LogLevel, format string, args ...interface{}) {
	Log(LogCategoryPanel, "login", level, format, args...)
}

// LogPanelRuntime 记录运行日志 → runtime.log
func LogPanelRuntime(level LogLevel, format string, args ...interface{}) {
	Log(LogCategoryPanel, "runtime", level, format, args...)
}

// ---- 以下为兼容旧调用，内部转发到新函数 ----

// LogPanelAuth 记录认证事件 → 登录日志
func LogPanelAuth(event, username, remoteAddr string, success bool, reason string) {
	level := LogLevelAuth
	if !success {
		level = LogLevelWarn
	}
	status := "成功"
	if !success {
		status = "失败"
	}
	LogPanelLogin(level, "[认证] %s %s: 用户=%s from %s (%s)", event, status, username, remoteAddr, reason)
}

// LogPanelFileOp 记录文件操作 → 操作日志
func LogPanelFileOp(username, operation, targetPath string, success bool) {
	status := "成功"
	if !success {
		status = "失败"
	}
	level := LogLevelInfo
	if !success {
		level = LogLevelWarn
	}
	LogPanelOperation(level, "[文件] %s %s (用户=%s, %s)", operation, targetPath, username, status)
}

// LogPanelConfigChange 记录配置变更 → 操作日志
func LogPanelConfigChange(username, configPath string, success bool) {
	status := "成功"
	if !success {
		status = "失败"
	}
	LogPanelOperation(LogLevelInfo, "[配置] 变更: %s (用户=%s, %s)", configPath, username, status)
}

// LogPanelService 记录服务控制 → 操作日志
func LogPanelService(username, service, action string, success bool) {
	status := "成功"
	if !success {
		status = "失败"
	}
	LogPanelOperation(LogLevelInfo, "[服务] %s: %s (用户=%s, %s)", action, service, username, status)
}

// LogPanelSystem 系统级日志 → 运行日志
func LogPanelSystem(level LogLevel, format string, args ...interface{}) {
	LogPanelRuntime(level, format, args...)
}

// GetLogFilePath 获取日志文件路径
func GetLogFilePath(category, logType string) string {
	if globalLogger == nil {
		return ""
	}

	if strings.Contains(category, "..") || strings.Contains(logType, "..") {
		return ""
	}

	return filepath.Join(globalLogger.baseDir, category, logType+".log")
}

// GetLogBaseDir 获取日志目录
func GetLogBaseDir() string {
	if globalLogger == nil {
		return ""
	}
	return globalLogger.baseDir
}

// CleanupNow 立即执行清理
func CleanupNow() {
	if globalLogger != nil {
		globalLogger.doCleanup()
	}
}
