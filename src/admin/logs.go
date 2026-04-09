package admin

import (
	"archive/tar"
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

// resolveLogPath 根据 category + type 解析日志文件路径
func resolveLogPath(logDir, category, logType string) (string, error) {
	switch category {
	case "http":
		if logType == "error" {
			return filepath.Join(logDir, "http", "error.log"), nil
		}
		return filepath.Join(logDir, "http", "access.log"), nil
	case "ftp":
		if logType == "error" {
			return filepath.Join(logDir, "ftp", "error.log"), nil
		}
		return filepath.Join(logDir, "ftp", "access.log"), nil
	case "panel":
		if logType == "" {
			logType = "operation"
		}
		return filepath.Join(logDir, "panel", logType+".log"), nil
	default:
		return "", fmt.Errorf("未知的日志分类: %s", category)
	}
}

// resolveDatedPath 查找历史日期文件，若无则回退原路径
// 返回 (文件路径, 是否为日期文件) — 调用方可根据此决定是否按日期过滤
func resolveDatedPath(logPath, date string) (string, bool) {
	if date == "" {
		return logPath, false
	}
	datedPath := strings.TrimSuffix(logPath, ".log") + "." + date + ".log"
	if _, err := os.Stat(datedPath); err == nil {
		return datedPath, true
	}
	gzPath := datedPath + ".gz"
	if _, err := os.Stat(gzPath); err == nil {
		return gzPath, true
	}
	// 没有找到日期文件，回退到当前文件，不做日期过滤
	return logPath, false
}

// ============ 公共：文件读取 & 解析 ============

// readLogFile 读取日志文件，支持 search/level/date 过滤
func readLogFile(path, search, level, date string, limit int) ([]LogEntry, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, 0, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	entries := make([]LogEntry, 0, limit)
	matched := 0
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if !matchLogLine(line, date, search, level) {
			continue
		}

		matched++
		if len(entries) < limit {
			entries = append(entries, parseLogEntry(line))
		}
	}

	return entries, matched, scanner.Err()
}

// matchLogLine 检查日志行是否匹配过滤条件
func matchLogLine(line, date, search, level string) bool {
	if date != "" && !strings.HasPrefix(line, date) {
		return false
	}
	if search != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(search)) {
		return false
	}
	if level != "" {
		lineLevel := extractLogLevel(line)
		if lineLevel != "" && lineLevel != level {
			return false
		}
	}
	return true
}

// parseLogEntry 解析日志条目
func parseLogEntry(line string) LogEntry {
	entry := LogEntry{Raw: line}

	if len(line) >= 19 {
		if _, err := time.Parse("2006-01-02 15:04:05", line[:19]); err == nil {
			entry.Timestamp = line[:19]
			line = strings.TrimSpace(line[19:])
		}
	}

	entry.Level = extractLogLevel(line)
	entry.Message = line
	return entry
}

// extractLogLevel 提取日志级别
func extractLogLevel(line string) string {
	upper := strings.ToUpper(line)
	for _, l := range []string{"DEBUG", "INFO", "WARN", "ERROR", "AUTH"} {
		if strings.Contains(upper, "["+l+"]") {
			return strings.ToLower(l)
		}
	}
	return ""
}

// ============ API Handlers ============

// handleLogsList 获取日志文件列表
func (h *Handler) handleLogsList(w http.ResponseWriter, r *http.Request) {
	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	files := make([]LogFileInfo, 0)

	for _, cat := range []string{"http", "ftp", "panel"} {
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
		category = "panel"
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	logPath, err := resolveLogPath(logDir, category, logType)
	if err != nil {
		Error(w, 400, err.Error())
		return
	}

	var isDated bool
	logPath, isDated = resolveDatedPath(logPath, date)

	// åªå¨ä½¿ç¨æ¥ææä»¶æ¶æææ¥æè¿æ»¤åå®¹ï¼å½åæ¥å¿æä»¶åå«å¤å¤©æ°æ®ä¸è¿æ»¤
	dateFilter := ""
	if !isDated {
		dateFilter = date  // current file: filter by date prefix
	}

	entries, total, err := readLogFile(logPath, search, level, dateFilter, limit)
	if err != nil {
		if os.IsNotExist(err) {
			Success(w, map[string]interface{}{"entries": []LogEntry{}, "total": 0, "file": filepath.Base(logPath)})
			return
		}
		Error(w, 500, "读取日志失败: "+err.Error())
		return
	}

	Success(w, map[string]interface{}{"entries": entries, "total": total, "file": filepath.Base(logPath)})
}

// handleLogsStats 获取日志统计
func (h *Handler) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	date := r.URL.Query().Get("date")
	search := r.URL.Query().Get("search")
	level := r.URL.Query().Get("level")

	if category == "" {
		category = "panel"
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	logPath, err := resolveLogPath(logDir, category, logType)
	if err != nil {
		Error(w, 400, err.Error())
		return
	}

	var isDated bool
	logPath, isDated = resolveDatedPath(logPath, date)
	dateFilter := ""
	if !isDated {
		dateFilter = date  // current file: filter by date prefix
	}

	stat := LogStats{Category: category, Type: logType}

	file, err := os.Open(logPath)
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if !matchLogLine(line, dateFilter, search, level) {
				continue
			}
			stat.Count++
			upper := strings.ToUpper(line)
			if strings.Contains(upper, "[ERROR]") {
				stat.Errors++
			}
			if strings.Contains(upper, "[WARN]") {
				stat.Warnings++
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

	logPath, err := resolveLogPath(logDir, category, logType)
	if err != nil {
		Error(w, 400, err.Error())
		return
	}

	if date != "" {
		datedPath := strings.TrimSuffix(logPath, ".log") + "." + date + ".log"
		if _, err := os.Stat(datedPath); err == nil {
			logPath = datedPath
		} else {
			gzPath := datedPath + ".gz"
			if _, err := os.Stat(gzPath); err == nil {
				logPath = gzPath
			}
		}
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		Error(w, 404, "文件不存在")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(logPath)))
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

	logPath, err := resolveLogPath(logDir, category, logType)
	if err != nil {
		Error(w, 400, err.Error())
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

// handleLogsBulkClear 批量清理日志
func (h *Handler) handleLogsBulkClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Category    string `json:"category"`
		BeforeDate  string `json:"before_date"`
		Compressed  bool   `json:"compressed"`  // 是否同时清理压缩文件
	}
	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "参数错误: "+err.Error())
		return
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	cleared := 0

	cats := []string{"http", "ftp", "panel"}
	if req.Category != "" {
		cats = []string{req.Category}
	}

	for _, cat := range cats {
		catDir := filepath.Join(logDir, cat)
		entries, err := os.ReadDir(catDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// 跳过当前日志文件（无日期后缀）
			base := strings.TrimSuffix(name, ".gz")
			if strings.HasSuffix(base, ".log") && !strings.Contains(base[:max(0, len(base)-4)], ".") {
				continue
			}

			// 按日期过滤
			if req.BeforeDate != "" {
				// 检查文件修改时间是否早于指定日期
				date, err := time.Parse("2006-01-02", req.BeforeDate)
				if err == nil && info.ModTime().After(date.Add(24*time.Hour)) {
					continue
				}
			}

			// 如果不清理压缩文件，跳过 .gz
			if !req.Compressed && strings.HasSuffix(name, ".gz") {
				continue
			}

			filePath := filepath.Join(catDir, name)
			if err := os.Remove(filePath); err == nil {
				cleared++
			}
		}
	}

	Success(w, map[string]interface{}{
		"cleared": cleared,
		"message": fmt.Sprintf("已清理 %d 个日志文件", cleared),
	})
}

// handleLogsBulkExport 批量导出日志
func (h *Handler) handleLogsBulkExport(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "panel"
	}

	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		Error(w, 500, "日志目录未初始化")
		return
	}

	catDir := filepath.Join(logDir, category)
	entries, err := os.ReadDir(catDir)
	if err != nil {
		Error(w, 404, "无日志文件")
		return
	}

	// 收集所有日志文件路径
	filePaths := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			filePaths = append(filePaths, filepath.Join(catDir, entry.Name()))
		}
	}

	if len(filePaths) == 0 {
		Error(w, 404, "无日志文件可导出")
		return
	}

	// 如果只有一个文件，直接下载
	if len(filePaths) == 1 {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePaths[0])))
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, filePaths[0])
		return
	}

	// 多个文件打包为 tar.gz
	filename := fmt.Sprintf("%s-logs-%s.tar.gz", category, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/gzip")

	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, fp := range filePaths {
		info, err := os.Stat(fp)
		if err != nil {
			continue
		}

		hdr := &tar.Header{
			Name: filepath.Base(fp),
			Size: info.Size(),
			Mode: 0644,
		}
		tw.WriteHeader(hdr)

		f, err := os.Open(fp)
		if err != nil {
			continue
		}
		io.Copy(tw, f)
		f.Close()
	}
}
