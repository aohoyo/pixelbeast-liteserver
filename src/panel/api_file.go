package panel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	fileop "pixelbeast/src/file"
	"pixelbeast/src/logger"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ==================== HTTP 文件管理 ====================

const (
	MaxUploadSize     = 500 << 20 // 500MB 上传限制
	MaxFileEditSize   = 10 * 1024 * 1024 // 10MB 文件编辑大小限制
	DefaultShareHours = 24        // 默认分享有效时长（小时）
)

// resolvePath 统一路径解析
// 支持绝对路径、相对路径和 ".." 导航
func resolvePath(subPath string) string {
	rootDir, _ := os.Getwd()

	// 空路径返回项目目录
	if subPath == "" {
		return rootDir
	}

	// "." 或 "./" 返回项目目录
	if subPath == "." || subPath == "./" {
		return rootDir
	}

	// 处理正斜杠格式的 Windows 绝对路径 (如 C:/Users/...)
	// filepath.IsAbs 期望反斜杠,所以需要先转换
	normalizedPath := filepath.FromSlash(subPath)

	// 检查是否是绝对路径(Linux/Unix 以 / 开头)
	// 限制最终路径在根目录下，防止路径遍历攻击
	cleaned := func(p string) string {
		c := filepath.Clean(p)
		if !strings.HasPrefix(c, rootDir+string(os.PathSeparator)) && c != rootDir {
			return filepath.Join(rootDir, filepath.Base(p))
		}
		return c
	}

	if filepath.IsAbs(normalizedPath) {
		return cleaned(normalizedPath)
	}

	// 检查是否是 Windows 驱动器格式的绝对路径
	if len(subPath) >= 2 && subPath[1] == ':' {
		return cleaned(normalizedPath)
	}

	// 相对路径:相对于项目目录
	return cleaned(filepath.Join(rootDir, normalizedPath))
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	subPath := r.URL.Query().Get("path")
	dirsOnly := r.URL.Query().Get("dirsOnly") == "true"

	// Windows 特殊处理：虚拟"此电脑"路径
	if subPath == "此电脑" {
		h.listDrives(w, r)
		return
	}

	absPath := resolvePath(subPath)

	fileEntries, err := fileop.ListDirEntries(absPath, dirsOnly)
	if err != nil {
		// 目录不存在时返回空列表
		if os.IsNotExist(err) {
			programDir, _ := os.Getwd()
			Success(w, map[string]interface{}{
				"path":        filepath.Clean(filepath.ToSlash(absPath)),
				"program_dir": filepath.Clean(filepath.ToSlash(programDir)),
				"files":       []fileop.FileEntry{},
			})
			return
		}
		InternalServerError(w, err.Error())
		return
	}

	// 返回当前路径（清理多余斜杠）
	displayPath := filepath.Clean(filepath.ToSlash(absPath))
	programDir, _ := os.Getwd()

	Success(w, map[string]interface{}{
		"path":        displayPath,
		"program_dir": filepath.Clean(filepath.ToSlash(programDir)),
		"files":       fileEntries,
	})
}

// listDrives 列出 Windows 驱动器
func (h *Handler) listDrives(w http.ResponseWriter, r *http.Request) {
	files := make([]map[string]interface{}, 0)

	// 检测操作系统
	if runtime.GOOS == "windows" {
		// Windows:检测所有可用驱动器
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			drivePath := string(drive) + ":/"
			if _, err := os.Stat(drivePath); err == nil {
				files = append(files, map[string]interface{}{
					"name":     string(drive) + ":",
					"is_dir":   true,
					"size":     int64(0),
					"modified": "",
				})
			}
		}
	} else {
		// 非 Windows:返回根目录
		files = append(files, map[string]interface{}{
			"name":     "/",
			"is_dir":   true,
			"size":     int64(0),
			"modified": "",
		})
	}

	Success(w, map[string]interface{}{
		"path":        "此电脑",
		"program_dir": "",
		"files":       files,
	})
}

// getQuickDirs 返回系统快捷目录
func (h *Handler) getQuickDirs(w http.ResponseWriter, r *http.Request) {
	type quickDir struct {
		Path      string `json:"path,omitempty"`
		Name      string `json:"name,omitempty"`
		Icon      string `json:"icon,omitempty"`
		Section   string `json:"section,omitempty"`
		IsDefault bool   `json:"isDefault,omitempty"`
	}

	dirs := make([]quickDir, 0)

	// 项目目录
	programDir, _ := os.Getwd()
	dirs = append(dirs, quickDir{
		Path: ".", Name: "项目目录", Icon: "folder", IsDefault: true,
	})

	if runtime.GOOS == "windows" {
		// 此电脑
		dirs = append(dirs, quickDir{Section: "系统目录"})
		dirs = append(dirs, quickDir{Path: "此电脑", Name: "此电脑", Icon: "computer"})

		// 用户目录
		homeDir := getHomeDir()
		if homeDir != "" {
			dirs = append(dirs, quickDir{Section: "用户目录"})
			userDirs := []struct{ sub, name, icon string }{
				{"Desktop", "桌面", "desktop"},
				{"Documents", "文档", "file-text"},
				{"Downloads", "下载", "download"},
				{"Pictures", "图片", "image"},
				{"Music", "音乐", "music"},
				{"Videos", "视频", "video"},
			}
			for _, ud := range userDirs {
				fullPath := filepath.Join(homeDir, ud.sub)
				if _, err := os.Stat(fullPath); err == nil {
					dirs = append(dirs, quickDir{Path: filepath.ToSlash(fullPath), Name: ud.name, Icon: ud.icon})
				}
			}
		}

		// 动态检测盘符
		drives := []string{}
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			drivePath := string(drive) + ":/"
			if _, err := os.Stat(drivePath); err == nil {
				drives = append(drives, string(drive)+":/")
			}
		}
		if len(drives) > 0 {
			dirs = append(dirs, quickDir{Section: "磁盘"})
			for _, d := range drives {
				name := strings.ToUpper(string(d[0])) + " 盘"
				dirs = append(dirs, quickDir{Path: d, Name: name, Icon: "hard-drive"})
			}
		}
	} else if runtime.GOOS == "darwin" {
		dirs = append(dirs, quickDir{Section: "系统目录"})
		dirs = append(dirs, quickDir{Path: "/", Name: "根目录", Icon: "server"})
		dirs = append(dirs, quickDir{Path: "/Users", Name: "Users", Icon: "users"})
		dirs = append(dirs, quickDir{Path: "/Applications", Name: "Applications", Icon: "package"})
		dirs = append(dirs, quickDir{Path: "/Library", Name: "Library", Icon: "folder"})
		dirs = append(dirs, quickDir{Path: "/tmp", Name: "tmp", Icon: "trash"})
	} else {
		// Linux
		dirs = append(dirs, quickDir{Section: "系统目录"})
		dirs = append(dirs, quickDir{Path: "/", Name: "根目录", Icon: "server"})
		dirs = append(dirs, quickDir{Path: "/home", Name: "home", Icon: "home"})
		dirs = append(dirs, quickDir{Path: "/var", Name: "var", Icon: "database"})
		dirs = append(dirs, quickDir{Path: "/etc", Name: "etc", Icon: "settings"})
		dirs = append(dirs, quickDir{Path: "/tmp", Name: "tmp", Icon: "trash"})
		dirs = append(dirs, quickDir{Path: "/usr", Name: "usr", Icon: "package"})
	}

	Success(w, map[string]interface{}{
		"dirs":        dirs,
		"program_dir": filepath.Clean(filepath.ToSlash(programDir)),
	})
}

func (h *Handler) uploadChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	uploadID := r.FormValue("uploadID")
	chunkIndex := r.FormValue("chunkIndex")
	file, _, err := r.FormFile("chunk")
	if err != nil {
		BadRequest(w, err.Error())
		return
	}
	defer file.Close()

	chunkDir := filepath.Join(os.TempDir(), "litefeather-uploads", uploadID)
	os.MkdirAll(chunkDir, 0755)

	dst := filepath.Join(chunkDir, fmt.Sprintf("chunk_%s", chunkIndex))
	f, err := os.Create(dst)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer f.Close()

	if _, err = f.ReadFrom(file); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	Success(w, map[string]interface{}{"chunkIndex": chunkIndex})
}

func (h *Handler) mergeChunks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		UploadID    string `json:"uploadID"`
		Filename    string `json:"filename"`
		TotalChunks int    `json:"totalChunks"`
		DestPath    string `json:"destPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	targetDir := resolvePath(req.DestPath)

	destFile := filepath.Join(targetDir, req.Filename)
	os.MkdirAll(filepath.Dir(destFile), 0755)
	f, err := os.Create(destFile)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer f.Close()

	chunkDir := filepath.Join(os.TempDir(), "litefeather-uploads", req.UploadID)
	for i := 0; i < req.TotalChunks; i++ {
		chunkData, err := os.ReadFile(filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", i)))
		if err != nil {
			BadRequest(w, fmt.Sprintf("Missing chunk %d", i))
			return
		}
		f.Write(chunkData)
	}
	os.RemoveAll(chunkDir)
	SuccessMessage(w, "合并成功")
}

func (h *Handler) uploadChunkStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	uploadID := r.URL.Query().Get("uploadID")
	if uploadID == "" || fileop.CheckPathTraversal(uploadID) || strings.Contains(uploadID, "/") || strings.Contains(uploadID, "\\") {
		BadRequest(w, "Invalid uploadID")
		return
	}

	chunkDir := filepath.Join(os.TempDir(), "litefeather-uploads", uploadID)
	entries, err := os.ReadDir(chunkDir)
	if err != nil {
		Success(w, map[string]interface{}{"chunks": []int{}})
		return
	}

	chunks := make([]int, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "chunk_") {
			var idx int
			if _, err := fmt.Sscanf(entry.Name(), "chunk_%d", &idx); err == nil {
				chunks = append(chunks, idx)
			}
		}
	}
	Success(w, map[string]interface{}{"chunks": chunks})
}

func (h *Handler) uploadFileWithPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	maxSize := int64(MaxUploadSize)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	if err := r.ParseMultipartForm(maxSize); err != nil {
		BadRequest(w, err.Error())
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		BadRequest(w, err.Error())
		return
	}
	defer file.Close()

	destPath := r.FormValue("path")
	relativePath := r.FormValue("relativePath")
	if fileop.CheckPathTraversal(destPath) || fileop.CheckPathTraversal(relativePath) {
		BadRequest(w, "Invalid path")
		return
	}

	targetDir := resolvePath(destPath)
	if relativePath != "" {
		targetDir = filepath.Join(targetDir, filepath.Dir(relativePath))
	}
	os.MkdirAll(targetDir, 0755)

	dst := filepath.Join(targetDir, handler.Filename)
	f, err := os.Create(dst)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer f.Close()

	if _, err = f.ReadFrom(file); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "上传", dst, true)
	SuccessWithData(w, map[string]interface{}{"filename": handler.Filename}, "上传成功")
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if fileop.CheckPathTraversal(req.Path) || fileop.CheckPathTraversal(req.Name) {
		BadRequest(w, "Invalid path")
		return
	}

	// 构建完整路径
	targetDir := resolvePath(req.Path)
	absPath := filepath.Join(targetDir, req.Name)

	if err := os.RemoveAll(absPath); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "删除", absPath, true)
	SuccessMessage(w, "删除成功")
}

func (h *Handler) mkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	// 安全检查
	if fileop.CheckPathTraversal(req.Path) {
		BadRequest(w, "Invalid path")
		return
	}

	newDir := resolvePath(req.Path)

	if err := os.MkdirAll(newDir, 0755); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "创建目录", req.Path, true)
	SuccessMessage(w, "目录创建成功")
}

// renameFile 重命名文件
func (h *Handler) renameFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Path    string `json:"path"`
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if fileop.CheckPathTraversal(req.Path) || fileop.CheckPathTraversal(req.OldName) || fileop.CheckPathTraversal(req.NewName) {
		BadRequest(w, "Invalid path")
		return
	}

	// 获取目标目录
	targetDir := resolvePath(req.Path)

	// 源文件路径
	oldPath := filepath.Join(targetDir, req.OldName)

	// 目标文件路径
	newPath := filepath.Join(targetDir, req.NewName)

	// 重命名
	if err := os.Rename(oldPath, newPath); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "重命名", req.OldName+" -> "+req.NewName, true)
	SuccessMessage(w, "重命名成功")
}

// downloadFile 下载文件
func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	name := r.URL.Query().Get("name")

	if fileop.CheckPathTraversal(path) || fileop.CheckPathTraversal(name) {
		Error(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// 构建完整路径
	targetDir := resolvePath(path)
	absPath := filepath.Join(targetDir, name)

	if err := fileop.StreamFileForDownload(w, absPath, name); err != nil {
		if os.IsNotExist(err) {
			Error(w, http.StatusNotFound, "文件不存在")
			return
		}
		InternalServerError(w, err.Error())
		return
	}
}

// copyFile 复制文件
func (h *Handler) copyFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		SrcPath string `json:"srcPath"` // 源目录
		SrcName string `json:"srcName"` // 源文件名
		DstPath string `json:"dstPath"` // 目标目录(可选,默认同源目录)
		DstName string `json:"dstName"` // 目标文件名(可选,默认同源文件名)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if fileop.CheckPathTraversal(req.SrcPath) || fileop.CheckPathTraversal(req.SrcName) || fileop.CheckPathTraversal(req.DstPath) || fileop.CheckPathTraversal(req.DstName) {
		BadRequest(w, "Invalid path")
		return
	}

	// 源文件路径
	srcDir := resolvePath(req.SrcPath)
	srcPath := filepath.Join(srcDir, req.SrcName)

	// 目标路径
	dstDir := srcDir // 默认同源目录
	if req.DstPath != "" {
		dstDir = resolvePath(req.DstPath)
	}
	dstName := req.SrcName // 默认同源文件名
	if req.DstName != "" {
		dstName = req.DstName
	}
	dstPath := filepath.Join(dstDir, dstName)

	// 检查源文件是否存在
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		InternalServerError(w, "源文件不存在")
		return
	}

	if srcInfo.IsDir() {
		// 目录:递归复制
		if err := fileop.CopyDirectory(srcPath, dstPath); err != nil {
			InternalServerError(w, err.Error())
			return
		}
	} else {
		// 文件:复制
		if err := fileop.CopySingleFile(srcPath, dstPath); err != nil {
			InternalServerError(w, err.Error())
			return
		}
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "复制", req.SrcName, true)
	SuccessMessage(w, "复制成功")
}

// touchFile 创建空文件
func (h *Handler) touchFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if fileop.CheckPathTraversal(req.Path) || fileop.CheckPathTraversal(req.Name) {
		BadRequest(w, "Invalid path")
		return
	}

	// 获取目标目录
	targetDir := resolvePath(req.Path)
	filePath := filepath.Join(targetDir, req.Name)

	// 创建空文件
	file, err := os.Create(filePath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer file.Close()

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "创建文件", req.Path, true)
	SuccessMessage(w, "文件创建成功")
}

// moveFile 移动文件(剪切+粘贴)
func (h *Handler) moveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		SrcPath string `json:"srcPath"` // 源目录
		SrcName string `json:"srcName"` // 源文件名
		DstPath string `json:"dstPath"` // 目标目录
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if fileop.CheckPathTraversal(req.SrcPath) || fileop.CheckPathTraversal(req.SrcName) || fileop.CheckPathTraversal(req.DstPath) {
		BadRequest(w, "Invalid path")
		return
	}

	// 源文件路径
	srcDir := resolvePath(req.SrcPath)
	srcPath := filepath.Join(srcDir, req.SrcName)

	// 目标路径
	dstDir := resolvePath(req.DstPath)
	dstPath := filepath.Join(dstDir, req.SrcName)

	// 移动(重命名)
	if err := os.Rename(srcPath, dstPath); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "移动", req.SrcPath, true)
	SuccessMessage(w, "移动成功")
}

// chmodFile 修改文件权限(仅 Linux)
func (h *Handler) chmodFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Windows 不支持
	if runtime.GOOS == "windows" {
		Error(w, http.StatusBadRequest, "Windows 系统不支持此操作")
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
		Mode string `json:"mode"` // 如 "755", "644"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if fileop.CheckPathTraversal(req.Path) || fileop.CheckPathTraversal(req.Name) {
		BadRequest(w, "Invalid path")
		return
	}

	// 解析权限模式
	var mode uint32
	if _, err := fmt.Sscanf(req.Mode, "%o", &mode); err != nil {
		BadRequest(w, "Invalid mode format")
		return
	}

	// 获取文件路径
	targetDir := resolvePath(req.Path)
	filePath := filepath.Join(targetDir, req.Name)

	// 修改权限
	if err := os.Chmod(filePath, os.FileMode(mode)); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "修改权限", req.Path, true)
	SuccessMessage(w, "权限修改成功")
}

// getFilePermissions 获取文件权限信息
func (h *Handler) getFilePermissions(w http.ResponseWriter, r *http.Request) {
	// Windows 不支持
	if runtime.GOOS == "windows" {
		Success(w, map[string]interface{}{
			"supported": false,
			"message":   "Windows 系统不支持权限管理",
		})
		return
	}

	path := r.URL.Query().Get("path")
	name := r.URL.Query().Get("name")

	if fileop.CheckPathTraversal(path) || fileop.CheckPathTraversal(name) {
		BadRequest(w, "Invalid path")
		return
	}

	// 获取文件路径
	targetDir := resolvePath(path)
	filePath := filepath.Join(targetDir, name)

	// 获取文件信息
	info, err := os.Stat(filePath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	// 获取权限
	mode := info.Mode()
	perm := fmt.Sprintf("%04o", mode.Perm())

	Success(w, map[string]interface{}{
		"supported": true,
		"mode":      perm,
		"is_dir":    info.IsDir(),
		"size":      info.Size(),
		"modified":  info.ModTime().Format(time.RFC3339),
	})
}

// readFileContent 读取文件内容
func (h *Handler) readFileContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	name := r.URL.Query().Get("name")

	if fileop.CheckPathTraversal(path) || fileop.CheckPathTraversal(name) {
		BadRequest(w, "Invalid path")
		return
	}

	// 获取文件路径
	targetDir := resolvePath(path)
	filePath := filepath.Join(targetDir, name)

	// 检查文件信息
	info, err := os.Stat(filePath)
	if err != nil {
		Error(w, http.StatusNotFound, "文件不存在")
		return
	}
	if info.IsDir() {
		Error(w, http.StatusBadRequest, "不能读取目录")
		return
	}

	// 限制文件大小 (最大 10MB)
	if info.Size() > MaxFileEditSize {
		Error(w, http.StatusBadRequest, "文件太大,超过 10MB 限制")
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	// 检测文件类型
	ext := strings.ToLower(filepath.Ext(name))
	fileType := "text"
	switch ext {
	case ".json":
		fileType = "json"
	case ".md", ".markdown":
		fileType = "markdown"
	case ".js", ".javascript":
		fileType = "javascript"
	case ".ts":
		fileType = "typescript"
	case ".go":
		fileType = "go"
	case ".py":
		fileType = "python"
	case ".css":
		fileType = "css"
	case ".html":
		fileType = "html"
	case ".xml":
		fileType = "xml"
	case ".yaml", ".yml":
		fileType = "yaml"
	case ".sh":
		fileType = "shell"
	case ".sql":
		fileType = "sql"
	}

	Success(w, map[string]interface{}{
		"content":  string(content),
		"size":     info.Size(),
		"modified": info.ModTime().Format(time.RFC3339),
		"type":     fileType,
		"path":     path,
		"name":     name,
	})
}

// saveFileContent 保存文件内容
func (h *Handler) saveFileContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path    string `json:"path"`
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	if fileop.CheckPathTraversal(req.Path) || fileop.CheckPathTraversal(req.Name) {
		BadRequest(w, "Invalid path")
		return
	}

	// 获取文件路径
	targetDir := resolvePath(req.Path)
	filePath := filepath.Join(targetDir, req.Name)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "编辑", req.Path, true)
	SuccessMessage(w, "保存成功")
}

// ==================== 压缩/解压 API ====================

// compressFiles 压缩文件/文件夹
func (h *Handler) compressFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Path   string   `json:"path"`   // 源目录
		Files  []string `json:"files"`  // 要压缩的文件列表
		Target string   `json:"target"` // 目标文件名（不含扩展名）
		Format string   `json:"format"` // zip 或 tar.gz
	}
	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return
	}

	// 安全检查
	if strings.Contains(req.Path, "..") {
		BadRequest(w, "Invalid path")
		return
	}
	for _, f := range req.Files {
		if strings.Contains(f, "..") {
			BadRequest(w, "Invalid file name")
			return
		}
	}

	// 默认格式
	if req.Format == "" {
		req.Format = "zip"
	}

	// 源目录
	srcDir := resolvePath(req.Path)

	// 目标文件名
	targetName := req.Target
	if targetName == "" {
		if len(req.Files) == 1 {
			targetName = req.Files[0]
		} else {
			targetName = "archive"
		}
	}

	// 根据格式压缩
	var outputPath string
	var err error

	switch req.Format {
	case "zip":
		outputPath = filepath.Join(srcDir, targetName+".zip")
		err = fileop.CreateZip(srcDir, req.Files, outputPath)
	case "tar.gz", "targz":
		outputPath = filepath.Join(srcDir, targetName+".tar.gz")
		err = fileop.CreateTarGz(srcDir, req.Files, outputPath)
	default:
		BadRequest(w, "Unsupported format: "+req.Format)
		return
	}

	if err != nil {
		InternalServerError(w, "压缩失败: "+err.Error())
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "压缩", req.Target, true)

	Success(w, map[string]interface{}{
		"file": filepath.Base(outputPath),
		"path": req.Path,
	})
}

// extractFile 解压文件
func (h *Handler) extractFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Path string `json:"path"` // 文件所在目录
		Name string `json:"name"` // 压缩包文件名
	}
	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return
	}

	// 安全检查
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	// 源文件路径
	srcDir := resolvePath(req.Path)
	srcFile := filepath.Join(srcDir, req.Name)

	// 判断格式
	ext := strings.ToLower(filepath.Ext(req.Name))
	var err error

	switch ext {
	case ".zip":
		err = fileop.ExtractZip(srcFile, srcDir)
	case ".gz":
		if strings.HasSuffix(strings.ToLower(req.Name), ".tar.gz") {
			err = fileop.ExtractTarGz(srcFile, srcDir)
		} else {
			err = fileop.ExtractGz(srcFile, srcDir)
		}
	default:
		if strings.HasSuffix(strings.ToLower(req.Name), ".tar.gz") || strings.HasSuffix(strings.ToLower(req.Name), ".tgz") {
			err = fileop.ExtractTarGz(srcFile, srcDir)
		} else {
			BadRequest(w, "Unsupported format: "+ext)
			return
		}
	}

	if err != nil {
		InternalServerError(w, "解压失败: "+err.Error())
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "解压", req.Path, true)
	SuccessMessage(w, "解压成功")
}

// ==================== 文件分享服务 ====================

// ShareLink 分享链接
type ShareLink struct {
	Token         string    `json:"token"`
	FilePath      string    `json:"filePath"`
	FileName      string    `json:"fileName"`
	FileSize      int64     `json:"fileSize"`
	ExpiresAt     time.Time `json:"expiresAt"`
	CreatedAt     time.Time `json:"createdAt"`
	DownloadCount int       `json:"downloadCount"`
	Password      string    `json:"password,omitempty"`
}

// ShareService 分享服务
type ShareService struct {
	links    map[string]*ShareLink
	mu       sync.RWMutex
	filePath string
}

// 全局分享服务实例
var shareService *ShareService

// InitShareService 初始化分享服务
func InitShareService(configDir string) {
	shareService = &ShareService{
		links:    make(map[string]*ShareLink),
		filePath: filepath.Join(configDir, "shares.json"),
	}
	shareService.load()
}

// load 从文件加载分享链接
func (s *ShareService) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.LogPanelRuntime(logger.LogLevelError, "[Share] 加载分享数据失败: %v", err)
		}
		return
	}

	var links []*ShareLink
	if err := json.Unmarshal(data, &links); err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Share] 解析分享数据失败: %v", err)
		return
	}

	now := time.Now()
	for _, link := range links {
		// 过滤已过期的链接
		if now.After(link.ExpiresAt) {
			continue
		}
		// 补充文件大小（旧数据可能没有）
		if link.FileSize == 0 {
			if fi, err := os.Stat(link.FilePath); err == nil {
				link.FileSize = fi.Size()
			}
		}
		s.links[link.Token] = link
	}

	logger.LogPanelRuntime(logger.LogLevelInfo, "[Share] 已加载 %d 个有效分享链接", len(s.links))
}

// save 保存分享链接到文件
func (s *ShareService) save() {
	s.mu.RLock()
	links := make([]*ShareLink, 0, len(s.links))
	for _, link := range s.links {
		links = append(links, link)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Share] 序列化分享数据失败: %v", err)
		return
	}

	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Share] 创建目录失败: %v", err)
		return
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Share] 保存分享数据失败: %v", err)
	}
}

// cleanExpired 清理过期链接
func (s *ShareService) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	changed := false
	for token, link := range s.links {
		if now.After(link.ExpiresAt) {
			delete(s.links, token)
			changed = true
		}
	}

	if changed {
		s.save()
	}
}

// shareFile 创建分享链接
func (h *Handler) shareFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path     string `json:"path"`
		Name     string `json:"name"`
		Duration int    `json:"duration"` // 有效时长（小时），默认 24
		Password string `json:"password"` // 访问密码（可选）
	}

	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return
	}

	// 安全检查
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	// 默认 24 小时
	if req.Duration <= 0 {
		req.Duration = DefaultShareHours
	}

	// 生成 token
	token := generateShareToken()

	// 构建文件路径
	targetDir := resolvePath(req.Path)
	filePath := filepath.Join(targetDir, req.Name)

	// 二次校验：确保最终路径仍在项目目录下
	rootDir, _ := os.Getwd()
	absPath, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absPath, rootDir+string(os.PathSeparator)) && absPath != rootDir {
		BadRequest(w, "Invalid path")
		return
	}

	// 检查文件是否存在并获取大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		BadRequest(w, "文件不存在")
		return
	}

	// 创建分享链接
	link := &ShareLink{
		Token:         token,
		FilePath:      filePath,
		FileName:      req.Name,
		FileSize:      fileInfo.Size(),
		ExpiresAt:     time.Now().Add(time.Duration(req.Duration) * time.Hour),
		CreatedAt:     time.Now(),
		DownloadCount: 0,
		Password:      req.Password,
	}

	// 存储并保存
	shareService.mu.Lock()
	shareService.links[token] = link
	shareService.mu.Unlock()
	shareService.save()

	// 构建分享 URL（不包含 adminPath，走公开路由）
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	shareURL := fmt.Sprintf("%s://%s/s/%s", scheme, r.Host, token)

	Success(w, map[string]interface{}{
		"token":     token,
		"url":       shareURL,
		"expiresAt": link.ExpiresAt.Format(time.RFC3339),
		"fileName":  req.Name,
		"fileSize":  fileInfo.Size(),
	})
}

// getShareInfo 获取分享信息（预览页面用）
func (h *Handler) getShareInfo(w http.ResponseWriter, r *http.Request) {
	// 从 URL 获取 token
	token := strings.TrimPrefix(r.URL.Path, "/s/info/")
	if token == "" {
		Error(w, http.StatusBadRequest, "无效的分享链接")
		return
	}

	// 查找分享链接
	shareService.mu.RLock()
	link, exists := shareService.links[token]
	shareService.mu.RUnlock()

	if !exists {
		Error(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}

	// 检查是否过期
	if time.Now().After(link.ExpiresAt) {
		// 删除过期链接
		shareService.mu.Lock()
		delete(shareService.links, token)
		shareService.mu.Unlock()
		shareService.save()
		Error(w, http.StatusGone, "分享链接已过期")
		return
	}

	// 返回分享信息（不返回文件路径和密码）
	Success(w, map[string]interface{}{
		"token":         link.Token,
		"fileName":      link.FileName,
		"fileSize":      link.FileSize,
		"expiresAt":     link.ExpiresAt.Format(time.RFC3339),
		"createdAt":     link.CreatedAt.Format(time.RFC3339),
		"downloadCount": link.DownloadCount,
		"hasPassword":   link.Password != "",
	})
}

// downloadSharedFile 下载分享文件
func (h *Handler) downloadSharedFile(w http.ResponseWriter, r *http.Request) {
	// 从 URL 获取 token（支持 /s/{token}/download 和 /s/{token}）
	path := r.URL.Path
	token := ""

	// 匹配 /s/{token}/download
	if strings.HasSuffix(path, "/download") {
		token = strings.TrimSuffix(strings.TrimPrefix(path, "/s/"), "/download")
	} else {
		// 兼容旧格式 /s/{token} 直接下载（显示预览页）
		token = strings.TrimPrefix(path, "/s/")
	}

	// 如果没有 /download 后缀，显示预览页
	if !strings.HasSuffix(path, "/download") {
		h.serveSharePage(w, r, token)
		return
	}

	if token == "" {
		Error(w, http.StatusBadRequest, "无效的分享链接")
		return
	}

	// 查找分享链接
	shareService.mu.RLock()
	link, exists := shareService.links[token]
	shareService.mu.RUnlock()

	if !exists {
		Error(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}

	// 检查是否过期
	if time.Now().After(link.ExpiresAt) {
		shareService.mu.Lock()
		delete(shareService.links, token)
		shareService.mu.Unlock()
		shareService.save()
		Error(w, http.StatusGone, "分享链接已过期")
		return
	}

	// 检查密码
	if link.Password != "" {
		password := r.URL.Query().Get("password")
		if password != link.Password {
			Error(w, http.StatusForbidden, "密码错误")
			return
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(link.FilePath); err != nil {
		Error(w, http.StatusNotFound, "文件不存在")
		return
	}

	// 安全校验：确保文件路径在项目目录下（防止存储被篡改）
	rootDir, _ := os.Getwd()
	absFilePath, _ := filepath.Abs(link.FilePath)
	if !strings.HasPrefix(absFilePath, rootDir+string(os.PathSeparator)) && absFilePath != rootDir {
		Error(w, http.StatusForbidden, "访问被拒绝")
		return
	}

	// 增加下载计数
	shareService.mu.Lock()
	link.DownloadCount++
	shareService.mu.Unlock()

	// 发送文件
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", link.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, link.FilePath)
}

// sharePageData 预览页面数据
type sharePageData struct {
	FileName      string
	FileSize      string
	ExpiresAt     string
	ExpiresAtISO  string
	DownloadCount int
	Token         string
	HasPassword   bool
}

// serveSharePage 显示分享预览页面
func (h *Handler) serveSharePage(w http.ResponseWriter, r *http.Request, token string) {
	// 查找分享链接
	shareService.mu.RLock()
	link, exists := shareService.links[token]
	shareService.mu.RUnlock()

	if !exists || time.Now().After(link.ExpiresAt) {
		// 显示过期页面
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(shareExpiredPageHTML))
		return
	}

	// 准备数据
	data := sharePageData{
		FileName:      link.FileName,
		FileSize:      formatFileSize(link.FileSize),
		ExpiresAt:     link.ExpiresAt.Format("2006-01-02 15:04"),
		ExpiresAtISO:  link.ExpiresAt.Format(time.RFC3339),
		DownloadCount: link.DownloadCount,
		Token:         link.Token,
		HasPassword:   link.Password != "",
	}

	// 解析并执行模板
	tmpl, err := template.New("share").Parse(sharePageHTML)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// formatFileSize 格式化文件大小
func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// listShareLinks 列出所有分享链接
func (h *Handler) listShareLinks(w http.ResponseWriter, r *http.Request) {
	// 先清理过期链接
	shareService.cleanExpired()

	shareService.mu.RLock()
	defer shareService.mu.RUnlock()

	// 构建列表
	links := make([]map[string]interface{}, 0)
	for _, link := range shareService.links {
		links = append(links, map[string]interface{}{
			"token":         link.Token,
			"fileName":      link.FileName,
			"fileSize":      link.FileSize,
			"expiresAt":     link.ExpiresAt.Format(time.RFC3339),
			"createdAt":     link.CreatedAt.Format(time.RFC3339),
			"downloadCount": link.DownloadCount,
		})
	}

	Success(w, map[string]interface{}{
		"links": links,
	})
}

// deleteShareLink 删除分享链接
func (h *Handler) deleteShareLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Token string `json:"token"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	shareService.mu.Lock()
	delete(shareService.links, req.Token)
	shareService.mu.Unlock()
	shareService.save()

	SuccessMessage(w, "已删除")
}

// generateShareToken 生成分享 token
func generateShareToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// 分享过期页面 HTML
const shareExpiredPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>链接已失效 - 像素兽</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: linear-gradient(135deg, #0c0a09 0%, #1c1917 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; color: #fafaf9; }
        .container { text-align: center; padding: 40px; }
        .icon { font-size: 64px; margin-bottom: 24px; }
        h1 { font-size: 24px; margin-bottom: 12px; }
        p { color: #78716c; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">⚠️</div>
        <h1>分享链接已失效</h1>
        <p>此链接不存在或已过期</p>
    </div>
</body>
</html>`

// 分享预览页面 HTML 模板
const sharePageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.FileName}} - 像素兽文件分享</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: linear-gradient(135deg, #0c0a09 0%, #1c1917 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; color: #fafaf9; padding: 20px; }
        .card { background: #1c1917; border: 1px solid #44403c; border-radius: 20px; padding: 32px 40px; max-width: 520px; width: 100%; }
        .header { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
        .logo { font-size: 32px; }
        .brand { display: flex; flex-direction: column; }
        .brand-name { font-size: 18px; font-weight: 600; color: #f97316; }
        .brand-desc { font-size: 12px; color: #78716c; }
        .file-section { display: flex; align-items: center; gap: 20px; padding: 20px; background: #0c0a09; border-radius: 12px; margin-bottom: 20px; }
        .file-icon { font-size: 48px; }
        .file-info { flex: 1; min-width: 0; }
        .file-name { font-size: 18px; font-weight: 600; color: #fafaf9; margin-bottom: 4px; word-break: break-all; }
        .file-size { font-size: 13px; color: #78716c; }
        .info-row { display: flex; gap: 24px; margin-bottom: 20px; }
        .info-item { flex: 1; }
        .info-label { font-size: 12px; color: #78716c; margin-bottom: 4px; }
        .info-value { font-size: 14px; color: #fafaf9; font-family: -apple-system, BlinkMacSystemFont, sans-serif; }
        .info-value.highlight { color: #f97316; font-weight: 600; }
        .password-section { margin-bottom: 16px; }
        .password-input { width: 100%; padding: 12px 16px; border: 1px solid #44403c; border-radius: 10px; background: #0c0a09; color: #fafaf9; font-size: 14px; }
        .password-input:focus { outline: none; border-color: #f97316; }
        .password-input::placeholder { color: #57534e; }
        .download-btn { display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; padding: 16px; background: linear-gradient(135deg, #f97316, #fb923c); color: white; border: none; border-radius: 12px; font-size: 16px; font-weight: 600; cursor: pointer; transition: all 0.2s; }
        .download-btn:hover { transform: translateY(-2px); box-shadow: 0 8px 24px rgba(249, 115, 22, 0.35); }
        .download-btn:active { transform: translateY(0); }
        .error-msg { color: #ef4444; font-size: 13px; text-align: center; margin-top: 12px; display: none; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #57534e; }
        @media (max-width: 480px) {
            .card { padding: 24px; }
            .info-row { flex-direction: column; gap: 12px; }
            .file-section { padding: 16px; }
            .file-icon { font-size: 36px; }
            .file-name { font-size: 16px; }
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="header">
            <div class="logo">🪶</div>
            <div class="brand">
                <span class="brand-name">像素兽文件分享</span>
                <span class="brand-desc">安全便捷的文件传输服务</span>
            </div>
        </div>
        
        <div class="file-section">
            <div class="file-icon">📄</div>
            <div class="file-info">
                <div class="file-name">{{.FileName}}</div>
                <div class="file-size">{{.FileSize}}</div>
            </div>
        </div>
        
        <div class="info-row">
            <div class="info-item">
                <div class="info-label">过期时间</div>
                <div class="info-value">{{.ExpiresAt}}</div>
            </div>
            <div class="info-item">
                <div class="info-label">剩余时间</div>
                <div class="info-value highlight" id="remaining">计算中...</div>
            </div>
            <div class="info-item">
                <div class="info-label">下载次数</div>
                <div class="info-value">{{.DownloadCount}} 次</div>
            </div>
        </div>
        
        {{if .HasPassword}}
        <div class="password-section">
            <input type="password" class="password-input" id="password" placeholder="🔒 请输入访问密码">
        </div>
        {{end}}
        
        <button class="download-btn" id="downloadBtn">
            <span>📥</span> 下载文件
        </button>
        <div class="error-msg" id="errorMsg">密码错误</div>
        
        <div class="footer">由 像素兽 PixelBeast 提供技术支持</div>
    </div>
    
    <script>
        const token = '{{.Token}}';
        const hasPassword = {{.HasPassword}};
        const expiresAt = new Date('{{.ExpiresAtISO}}').getTime();
        
        // 从URL参数自动填充提取码
        const urlParams = new URLSearchParams(window.location.search);
        const pwdParam = urlParams.get('pwd');
        if (pwdParam && hasPassword) {
            const pwdInput = document.getElementById('password');
            if (pwdInput) {
                pwdInput.value = pwdParam;
            }
        }
        
        function updateRemaining() {
            const now = Date.now();
            const diff = expiresAt - now;
            const el = document.getElementById('remaining');
            if (diff <= 0) { el.textContent = '已过期'; return; }
            const days = Math.floor(diff / 86400000);
            const hours = Math.floor((diff % 86400000) / 3600000);
            const mins = Math.floor((diff % 3600000) / 60000);
            if (days > 0) el.textContent = days + '天' + hours + '小时';
            else if (hours > 0) el.textContent = hours + '小时' + mins + '分';
            else el.textContent = mins + '分钟';
        }
        updateRemaining();
        setInterval(updateRemaining, 60000);
        
        document.getElementById('downloadBtn').addEventListener('click', function() {
            let url = '/s/' + token + '/download';
            const errorEl = document.getElementById('errorMsg');
            errorEl.style.display = 'none';
            
            if (hasPassword) {
                const pwd = document.getElementById('password').value;
                if (!pwd) {
                    errorEl.textContent = '请输入访问密码';
                    errorEl.style.display = 'block';
                    return;
                }
                url += '?password=' + encodeURIComponent(pwd);
            }
            
            fetch(url, { method: 'HEAD' }).then(resp => {
                if (resp.ok) window.location.href = url;
                else { errorEl.textContent = '密码错误'; errorEl.style.display = 'block'; }
            }).catch(() => { window.location.href = url; });
        });
    </script>
</body>
</html>`
