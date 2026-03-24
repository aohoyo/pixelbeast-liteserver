package admin

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pixelbeast/src/handlers"
)

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Type       string    `json:"type"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Compressed bool      `json:"compressed"`
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level,omitempty"`
	Message   string `json:"message"`
	Raw       string `json:"raw"`
}

// LogStats 日志统计
type LogStats struct {
	Category string `json:"category"`
	Type     string `json:"type"`
	Count    int    `json:"count"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
}

// 日志分类和类型
var logCategories = []string{"http", "ftp", "panel"}
var logTypes = map[string][]string{
	"http":  {"access", "error"},
	"ftp":   {"access", "error"},
	"panel": {"access", "api", "auth"},
}

// handleLogsList 获取日志文件列表
func (h *Handler) handleLogsList(w http.ResponseWriter, r *http.Request) {
	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	files := []LogFileInfo{}

	for _, cat := range logCategories {
		catDir := filepath.Join(logDir, cat)
		entries, err := os.ReadDir(catDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			name := entry.Name()
			ext := filepath.Ext(name)
			baseName := strings.TrimSuffix(name, ext)

			// 确定日志类型
			logType := ""
			for _, t := range logTypes[cat] {
				if strings.HasPrefix(baseName, t) {
					logType = t
					break
				}
			}

			if logType == "" {
				logType = "other"
			}

			files = append(files, LogFileInfo{
				Name:       name,
				Category:   cat,
				Type:       logType,
				Size:       info.Size(),
				ModifiedAt: info.ModTime(),
				Compressed: ext == ".gz",
			})
		}
	}

	// 按修改时间降序排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModifiedAt.After(files[j].ModifiedAt)
	})

	Success(w, files)
}

// handleLogsRead 读取日志内容
func (h *Handler) handleLogsRead(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	date := r.URL.Query().Get("date")
	search := r.URL.Query().Get("search")
	level := r.URL.Query().Get("level")
	offset := parseIntParam(r.URL.Query().Get("offset"), 0)
	limit := parseIntParam(r.URL.Query().Get("limit"), 100)

	if category == "" {
		category = "http"
	}
	if logType == "" {
		logType = "access"
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	// 确定文件路径
	var logPath string
	if date != "" {
		// 指定日期的历史日志
		logPath = filepath.Join(logDir, category, fmt.Sprintf("%s.%s.log", logType, date))
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			// 尝试压缩文件
			logPath = logPath + ".gz"
		}
	} else {
		// 当前日志
		logPath = filepath.Join(logDir, category, logType+".log")
	}

	// 读取文件
	entries, total, err := readLogFile(logPath, search, level, offset, limit)
	if err != nil {
		if os.IsNotExist(err) {
			Success(w, map[string]interface{}{
				"entries": []LogEntry{},
				"total":   0,
				"file":    logPath,
			})
			return
		}
		Error(w, 500, "读取日志失败: "+err.Error())
		return
	}

	Success(w, map[string]interface{}{
		"entries": entries,
		"total":   total,
		"file":    filepath.Base(logPath),
	})
}

// readLogFile 读取日志文件
func readLogFile(path string, search, level string, offset, limit int) ([]LogEntry, int, error) {
	var reader io.Reader

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	// 检查是否是压缩文件
	if strings.HasSuffix(path, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, 0, err
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = file
	}

	entries := []LogEntry{}
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	matched := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// 搜索过滤
		if search != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(search)) {
			continue
		}

		// 级别过滤
		if level != "" {
			lineLevel := extractLogLevel(line)
			if lineLevel != "" && lineLevel != level {
				continue
			}
		}

		// 解析日志条目
		entry := parseLogEntry(line)
		matched++

		// 分页
		if matched <= offset {
			continue
		}
		if len(entries) >= limit {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	return entries, matched, nil
}

// parseLogEntry 解析日志条目
func parseLogEntry(line string) LogEntry {
	entry := LogEntry{Raw: line}

	// 尝试解析时间戳 (格式: 2006-01-02 15:04:05)
	if len(line) >= 19 {
		ts := line[:19]
		if _, err := time.Parse("2006-01-02 15:04:05", ts); err == nil {
			entry.Timestamp = ts
			line = strings.TrimSpace(line[19:])
		}
	}

	// 提取级别
	entry.Level = extractLogLevel(line)
	entry.Message = line

	return entry
}

// extractLogLevel 提取日志级别
func extractLogLevel(line string) string {
	line = strings.ToUpper(line)
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "AUTH"}
	for _, l := range levels {
		if strings.Contains(line, "["+l+"]") {
			return strings.ToLower(l)
		}
	}
	return ""
}

// handleLogsStats 获取日志统计
func (h *Handler) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	date := r.URL.Query().Get("date")

	if category == "" {
		category = "http"
	}

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	stats := []LogStats{}
	types := logTypes[category]

	for _, t := range types {
		logPath := filepath.Join(logDir, category, t+".log")
		stat := LogStats{
			Category: category,
			Type:     t,
		}

		// 读取并统计
		file, err := os.Open(logPath)
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// 只统计今天的日志
				if strings.HasPrefix(line, date) {
					stat.Count++
					if strings.Contains(strings.ToUpper(line), "[ERROR]") {
						stat.Errors++
					}
					if strings.Contains(strings.ToUpper(line), "[WARN]") {
						stat.Warnings++
					}
				}
			}
			file.Close()
		}

		stats = append(stats, stat)
	}

	Success(w, stats)
}

// handleLogsDownload 下载日志文件
func (h *Handler) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	filename := r.URL.Query().Get("file")

	if category == "" || filename == "" {
		Error(w, 400, "缺少参数")
		return
	}

	// 安全检查
	if strings.Contains(category, "..") || strings.Contains(filename, "..") {
		Error(w, 400, "非法参数")
		return
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	logPath := filepath.Join(logDir, category, filename)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		Error(w, 404, "文件不存在")
		return
	}

	// 设置下载头
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, logPath)
}

// handleLogsClear 清空日志
func (h *Handler) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	before := r.URL.Query().Get("before") // 清理此日期之前的日志

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	cleared := 0

	if category != "" && logType != "" {
		// 清空指定日志
		logPath := filepath.Join(logDir, category, logType+".log")
		if err := os.Truncate(logPath, 0); err != nil {
			Error(w, 500, "清空失败: "+err.Error())
			return
		}
		cleared = 1
	} else if before != "" {
		// 清理指定日期之前的日志
		cutoff, err := time.Parse("2006-01-02", before)
		if err != nil {
			Error(w, 400, "日期格式错误")
			return
		}

		categories := logCategories
		if category != "" {
			categories = []string{category}
		}

		for _, cat := range categories {
			catDir := filepath.Join(logDir, cat)
			entries, err := os.ReadDir(catDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				// 跳过当前日志文件
				name := entry.Name()
				if !strings.Contains(name, ".log.") && !strings.HasSuffix(name, ".gz") {
					continue
				}

				info, err := entry.Info()
				if err != nil {
					continue
				}

				if info.ModTime().Before(cutoff) {
					os.Remove(filepath.Join(catDir, name))
					cleared++
				}
			}
		}
	} else {
		Error(w, 400, "缺少参数")
		return
	}

	Success(w, map[string]interface{}{
		"cleared": cleared,
		"message": fmt.Sprintf("已清理 %d 个日志文件", cleared),
	})
}

// handleLogsConfig 获取/更新日志配置
func (h *Handler) handleLogsConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		Success(w, h.Config.Log)
		return
	}

	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 解析请求体
	var cfg struct {
		RetentionDays int               `json:"retention_days"`
		MaxSizeMB     int               `json:"max_size_mb"`
		CompressDays  int               `json:"compress_days"`
		CleanupHour   int               `json:"cleanup_hour"`
		Level         string            `json:"level"`
		Levels        map[string]string `json:"levels"`
	}

	if err := parseJSONBody(r, &cfg); err != nil {
		BadRequest(w, "参数错误: "+err.Error())
		return
	}

	// 更新配置
	if cfg.RetentionDays > 0 {
		h.Config.Log.RetentionDays = cfg.RetentionDays
	}
	if cfg.MaxSizeMB > 0 {
		h.Config.Log.MaxSizeMB = cfg.MaxSizeMB
	}
	if cfg.CompressDays > 0 {
		h.Config.Log.CompressDays = cfg.CompressDays
	}
	if cfg.CleanupHour >= 0 && cfg.CleanupHour <= 23 {
		h.Config.Log.CleanupHour = cfg.CleanupHour
	}
	if cfg.Level != "" {
		h.Config.Log.Level = cfg.Level
	}
	if cfg.Levels != nil {
		h.Config.Log.Levels = cfg.Levels
	}

	// 保存配置
	if err := h.Config.Save(h.ConfigPath); err != nil {
		Error(w, 500, "保存配置失败: "+err.Error())
		return
	}

	// 更新日志配置
	handlers.SetLogConfig(&h.Config.Log)

	Success(w, h.Config.Log)
}