package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ==================== HTTP 文件管理 ====================

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	subPath := r.URL.Query().Get("path")

	if strings.Contains(subPath, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	absRoot, _ := filepath.Abs(h.Config.HTTP.Root)
	absPath := absRoot
	if subPath != "" && subPath != "/" {
		absPath = filepath.Join(absRoot, subPath)
	}
	if !strings.HasPrefix(absPath, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		info, _ := entry.Info()
		files = append(files, map[string]interface{}{
			"name": entry.Name(), "is_dir": entry.IsDir(),
			"size": info.Size(), "modified": info.ModTime().Format(time.RFC3339),
		})
	}
	Success(w, map[string]interface{}{"path": subPath, "files": files})
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

	targetDir := filepath.Join(h.Config.HTTP.Root, destPath)
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

	targetDir := filepath.Join(h.Config.HTTP.Root, req.DestPath)
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
	fullPath := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(req.Path, "/"), req.Name)
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(h.Config.HTTP.Root)
	if !strings.HasPrefix(absPath, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

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

	newDir := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(req.Path, "/"), req.Name)
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

	// 源文件路径
	oldPath := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(req.Path, "/"), req.OldName)
	absOld, _ := filepath.Abs(oldPath)
	absRoot, _ := filepath.Abs(h.Config.HTTP.Root)
	if !strings.HasPrefix(absOld, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 目标文件路径
	newPath := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(req.Path, "/"), req.NewName)
	absNew, _ := filepath.Abs(newPath)
	if !strings.HasPrefix(absNew, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 重命名
	if err := os.Rename(absOld, absNew); err != nil {
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
	fullPath := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(path, "/"), name)
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(h.Config.HTTP.Root)
	if !strings.HasPrefix(absPath, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

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

	// 源文件路径
	srcPath := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(req.SrcPath, "/"), req.SrcName)
	srcAbs, _ := filepath.Abs(srcPath)
	absRoot, _ := filepath.Abs(h.Config.HTTP.Root)
	if !strings.HasPrefix(srcAbs, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 目标文件路径
	dstPath := filepath.Join(h.Config.HTTP.Root, strings.TrimPrefix(req.SrcPath, "/"), req.DstName)
	dstAbs, _ := filepath.Abs(dstPath)
	if !strings.HasPrefix(dstAbs, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 打开源文件
	srcFile, err := os.Open(srcAbs)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dstAbs)
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

