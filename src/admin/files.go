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
	
	if subPath == "" || subPath == "/" {
		return rootDir
	}
	
	// 处理正斜杠格式的 Windows 绝对路径 (如 C:/Users/...)
	// filepath.IsAbs 期望反斜杠，所以需要先转换
	normalizedPath := filepath.FromSlash(subPath)
	
	// 检查是否是绝对路径
	if filepath.IsAbs(normalizedPath) {
		// 绝对路径：直接使用，并清理 ".."
		return filepath.Clean(normalizedPath)
	}
	
	// 检查是否是原始的正斜杠格式绝对路径
	if len(subPath) >= 2 && subPath[1] == ':' {
		// Windows 驱动器格式 (C:/...)
		return filepath.Clean(normalizedPath)
	}
	
	// 相对路径：相对于程序目录
	return filepath.Join(rootDir, normalizedPath)
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	subPath := r.URL.Query().Get("path")

	// Windows 特殊处理：虚拟"此电脑"路径
	if subPath == "此电脑" {
		h.listDrives(w, r)
		return
	}

	absPath := resolvePath(subPath)

	entries, err := os.ReadDir(absPath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
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

	// 返回当前路径（转为正斜杠格式，便于前端显示）
	displayPath := filepath.ToSlash(absPath)
	
	// 获取程序目录
	programDir, _ := os.Getwd()
	programDirDisplay := filepath.ToSlash(programDir)
	
	Success(w, map[string]interface{}{
		"path":        displayPath,
		"program_dir": programDirDisplay,
		"files":       files,
	})
}

// listDrives 列出 Windows 驱动器
func (h *Handler) listDrives(w http.ResponseWriter, r *http.Request) {
	files := make([]map[string]interface{}, 0)
	
	// 检测操作系统
	if runtime.GOOS == "windows" {
		// Windows：检测所有可用驱动器
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
		// 非 Windows：返回根目录
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

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	// 增加上传限制到 500MB
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
	if strings.Contains(destPath, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	targetDir := resolvePath(destPath)
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
	os.MkdirAll(targetDir, 0755)

	destFile := filepath.Join(targetDir, req.Filename)
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
		SrcPath string `json:"srcPath"`
		SrcName string `json:"srcName"`
		DstName string `json:"dstName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if strings.Contains(req.SrcPath, "..") || strings.Contains(req.SrcName, "..") || strings.Contains(req.DstName, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	// 获取目标目录
	targetDir := resolvePath(req.SrcPath)

	// 源文件路径
	srcPath := filepath.Join(targetDir, req.SrcName)

	// 目标文件路径
	dstPath := filepath.Join(targetDir, req.DstName)

	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dstPath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer dstFile.Close()

	// 复制内容
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	SuccessMessage(w, "复制成功")
}