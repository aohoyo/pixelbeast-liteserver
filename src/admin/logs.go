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

// handleLogsList 获取日志文件列表
func (h *Handler) handleLogsList(w http.ResponseWriter, r *http.Request) {
	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	files := []LogFileInfo{}
	categories := []string{"system", "http", "ftp", "panel"}

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

			info, err := entry.Info()
			if err != nil {
				continue
			}

			name := entry.Name()
			ext := filepath.Ext(name)
			
			files = append(files, LogFileInfo{
				Name:       name,
				Category:   cat,
				Type:       strings.TrimSuffix(name, ext),
				Size:       info.Size(),
				ModifiedAt: info.ModTime(),
				Compressed: ext == ".gz",
			})
		}
	}

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
	limit := parseIntParam(r.URL.Query().Get("limit"), 200)

	if category == "" {
		category = "system"
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	// 确定日志文件路径
	var logPath string
	
	switch category {
	case "system":
		logPath = filepath.Join(logDir, "system", "server.log")
	case "http":
		// HTTP 日志按站点分：site-{id}.log
		if logType == "" {
			logType = "default"
		}
		logPath = filepath.Join(logDir, "http", fmt.Sprintf("site-%s.log", logType))
	case "ftp":
		// FTP 日志按用户分：user-{name}.log
		if logType == "" {
			logType = "anonymous"
		}
		logPath = filepath.Join(logDir, "ftp", fmt.Sprintf("user-%s.log", logType))
	case "panel":
		// 面板日志：统一记录到 server.log
		logPath = filepath.Join(logDir, "panel", "server.log")
	default:
		Error(w, 400, "未知的日志分类")
		return
	}

	// 检查历史日期
	if date != "" {
		logPath = strings.TrimSuffix(logPath, ".log") + "." + date + ".log"
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			logPath = logPath + ".gz"
		}
	}

	entries, total, err := readLogFile(logPath, search, level, 0, limit)
	if err != nil {
		if os.IsNotExist(err) {
			Success(w, map[string]interface{}{
				"entries": []LogEntry{},
				"total":   0,
				"file":    filepath.Base(logPath),
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
	matched := 0

	for scanner.Scan() {
		line := scanner.Text()

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

		entry := parseLogEntry(line)
		matched++

		if matched > offset && len(entries) < limit {
			entries = append(entries, entry)
		}
	}

	return entries, matched, scanner.Err()
}

// parseLogEntry 解析日志条目
func parseLogEntry(line string) LogEntry {
	entry := LogEntry{Raw: line}

	if len(line) >= 19 {
		ts := line[:19]
		if _, err := time.Parse("2006-01-02 15:04:05", ts); err == nil {
			entry.Timestamp = ts
			line = strings.TrimSpace(line[19:])
		}
	}

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
	logType := r.URL.Query().Get("type")
	date := r.URL.Query().Get("date")

	if category == "" {
		category = "system"
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	stat := LogStats{Category: category, Type: logType}

	// 获取日志路径
	var logPath string
	switch category {
	case "system":
		logPath = filepath.Join(logDir, "system", "server.log")
	case "http":
		if logType == "" {
			logType = "default"
		}
		logPath = filepath.Join(logDir, "http", fmt.Sprintf("site-%s.log", logType))
	case "ftp":
		if logType == "" {
			logType = "anonymous"
		}
		logPath = filepath.Join(logDir, "ftp", fmt.Sprintf("user-%s.log", logType))
	case "panel":
		logPath = filepath.Join(logDir, "panel", "server.log")
	}

	file, err := os.Open(logPath)
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
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

	Success(w, []LogStats{stat})
}

// handleLogsDownload 下载日志文件
func (h *Handler) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	date := r.URL.Query().Get("date")

	if category == "" {
		Error(w, 400, "缺少参数")
		return
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	// 构建日志路径
	var logPath string
	switch category {
	case "system":
		logPath = filepath.Join(logDir, "system", "server.log")
	case "http":
		if logType == "" {
			logType = "default"
		}
		logPath = filepath.Join(logDir, "http", fmt.Sprintf("site-%s.log", logType))
	case "ftp":
		if logType == "" {
			logType = "anonymous"
		}
		logPath = filepath.Join(logDir, "ftp", fmt.Sprintf("user-%s.log", logType))
	case "panel":
		logPath = filepath.Join(logDir, "panel", "server.log")
	default:
		Error(w, 400, "未知的日志分类")
		return
	}

	// 历史日期
	if date != "" {
		logPath = strings.TrimSuffix(logPath, ".log") + "." + date + ".log"
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		Error(w, 404, "文件不存在")
		return
	}

	filename := filepath.Base(logPath)
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

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	var logPath string
	switch category {
	case "system":
		logPath = filepath.Join(logDir, "system", "server.log")
	case "http":
		if logType == "" {
			logType = "default"
		}
		logPath = filepath.Join(logDir, "http", fmt.Sprintf("site-%s.log", logType))
	case "ftp":
		if logType == "" {
			logType = "anonymous"
		}
		logPath = filepath.Join(logDir, "ftp", fmt.Sprintf("user-%s.log", logType))
	case "panel":
		logPath = filepath.Join(logDir, "panel", "server.log")
	default:
		Error(w, 400, "未知的日志分类")
		return
	}

	if err := os.Truncate(logPath, 0); err != nil {
		if os.IsNotExist(err) {
			Success(w, map[string]interface{}{"cleared": 0, "message": "日志文件不存在"})
			return
		}
		Error(w, 500, "清空失败: "+err.Error())
		return
	}

	Success(w, map[string]interface{}{"cleared": 1, "message": "日志已清空"})
}

// handleLogsConfig 获取/更新日志配置
func (h *Handler) handleLogsConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		Success(w, h.ConfigManager.Server.Log)
		return
	}

	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var cfg struct {
		RetentionDays int    `json:"retention_days"`
		MaxSizeMB     int    `json:"max_size_mb"`
		CompressDays  int    `json:"compress_days"`
		Level         string `json:"level"`
	}

	if err := parseJSONBody(r, &cfg); err != nil {
		BadRequest(w, "参数错误: "+err.Error())
		return
	}

	if cfg.RetentionDays > 0 {
		h.ConfigManager.Server.Log.RetentionDays = cfg.RetentionDays
	}
	if cfg.MaxSizeMB > 0 {
		h.ConfigManager.Server.Log.MaxSizeMB = cfg.MaxSizeMB
	}
	if cfg.CompressDays > 0 {
		h.ConfigManager.Server.Log.CompressDays = cfg.CompressDays
	}
	if cfg.Level != "" {
		h.ConfigManager.Server.Log.Level = cfg.Level
	}

	if err := h.ConfigManager.Save(); err != nil {
		Error(w, 500, "保存配置失败: "+err.Error())
		return
	}

	handlers.SetLogConfig(&h.ConfigManager.Server.Log)
	Success(w, h.ConfigManager.Server.Log)
}