package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ==================== HTTP 文件管理 ====================

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
	if filepath.IsAbs(normalizedPath) {
		// 绝对路径:直接使用,并清理 ".."
		return filepath.Clean(normalizedPath)
	}

	// 检查是否是 Windows 驱动器格式的绝对路径
	if len(subPath) >= 2 && subPath[1] == ':' {
		// Windows 驱动器格式 (C:/...)
		return filepath.Clean(normalizedPath)
	}

	// 相对路径:相对于项目目录
	return filepath.Join(rootDir, normalizedPath)
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

	entries, err := os.ReadDir(absPath)
	if err != nil {
		// 目录不存在时返回空列表
		if os.IsNotExist(err) {
			programDir, _ := os.Getwd()
			Success(w, map[string]interface{}{
				"path":        filepath.Clean(filepath.ToSlash(absPath)),
				"program_dir": filepath.Clean(filepath.ToSlash(programDir)),
				"files":       []map[string]interface{}{},
			})
			return
		}
		InternalServerError(w, err.Error())
		return
	}

	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		// 如果只需要目录，跳过文件
		if dirsOnly && !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"modified": info.ModTime().Format(time.RFC3339),
		})
	}

	// 返回当前路径（清理多余斜杠）
	displayPath := filepath.Clean(filepath.ToSlash(absPath))
	programDir, _ := os.Getwd()

	Success(w, map[string]interface{}{
		"path":        displayPath,
		"program_dir": filepath.Clean(filepath.ToSlash(programDir)),
		"files":       files,
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
		homeDir, _ := os.UserHomeDir()
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
	if uploadID == "" || strings.Contains(uploadID, "..") || strings.Contains(uploadID, "/") || strings.Contains(uploadID, "\\") {
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
	maxSize := int64(500 << 20)
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
	if strings.Contains(destPath, "..") || strings.Contains(relativePath, "..") {
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
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
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
	if strings.Contains(req.Path, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	newDir := resolvePath(req.Path)

	if err := os.MkdirAll(newDir, 0755); err != nil {
		InternalServerError(w, err.Error())
		return
	}
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
	if strings.Contains(req.Path, "..") || strings.Contains(req.OldName, "..") || strings.Contains(req.NewName, "..") {
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
	SuccessMessage(w, "重命名成功")
}

// downloadFile 下载文件
func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	name := r.URL.Query().Get("name")

	if strings.Contains(path, "..") || strings.Contains(name, "..") {
		Error(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// 构建完整路径
	targetDir := resolvePath(path)
	absPath := filepath.Join(targetDir, name)

	// 检查是否为目录
	info, err := os.Stat(absPath)
	if err != nil {
		Error(w, http.StatusNotFound, "文件不存在")
		return
	}
	if info.IsDir() {
		Error(w, http.StatusBadRequest, "不能下载目录")
		return
	}

	// 设置下载头
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	// 打开文件并发送
	file, err := os.Open(absPath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer file.Close()

	io.Copy(w, file)
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
	if strings.Contains(req.SrcPath, "..") || strings.Contains(req.SrcName, "..") || strings.Contains(req.DstPath, "..") || strings.Contains(req.DstName, "..") {
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
		if err := h.copyDir(srcPath, dstPath); err != nil {
			InternalServerError(w, err.Error())
			return
		}
	} else {
		// 文件:复制
		srcFile, err := os.Open(srcPath)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
		defer srcFile.Close()

		// 确保目标目录存在
		os.MkdirAll(dstDir, 0755)

		dstFile, err := os.Create(dstPath)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
		defer dstFile.Close()

		if _, err = io.Copy(dstFile, srcFile); err != nil {
			InternalServerError(w, err.Error())
			return
		}
	}

	SuccessMessage(w, "复制成功")
}

// copyDir 递归复制目录
func (h *Handler) copyDir(srcPath, dstPath string) error {
	// 创建目标目录
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return err
	}

	// 遍历源目录
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		src := filepath.Join(srcPath, entry.Name())
		dst := filepath.Join(dstPath, entry.Name())

		if entry.IsDir() {
			if err := h.copyDir(src, dst); err != nil {
				return err
			}
		} else {
			// 复制文件
			srcFile, err := os.Open(src)
			if err != nil {
				return err
			}
			dstFile, err := os.Create(dst)
			if err != nil {
				srcFile.Close()
				return err
			}
			_, err = io.Copy(dstFile, srcFile)
			srcFile.Close()
			dstFile.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
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
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
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
	if strings.Contains(req.SrcPath, "..") || strings.Contains(req.SrcName, "..") || strings.Contains(req.DstPath, "..") {
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
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
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

	if strings.Contains(path, "..") || strings.Contains(name, "..") {
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

	if strings.Contains(path, "..") || strings.Contains(name, "..") {
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
	if info.Size() > 10*1024*1024 {
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

	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
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

	SuccessMessage(w, "保存成功")
}
