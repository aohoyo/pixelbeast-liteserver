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

	"pixelbeast/src/config"
)

// ==================== FTP 服务控制 ====================

func (h *Handler) toggleFTP(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	var err error
	var msg string
	if h.ServerManager.IsFTPRunning() {
		err, msg = h.ServerManager.StopFTP(), "FTP服务已停止"
	} else {
		err, msg = h.ServerManager.StartFTP(), "FTP服务已启动"
	}
	if err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	SuccessMessage(w, msg)
}

func (h *Handler) startFTP(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.StartFTP(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	SuccessMessage(w, "FTP服务已启动")
}

func (h *Handler) stopFTP(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.StopFTP(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	SuccessMessage(w, "FTP服务已停止")
}

func (h *Handler) restartFTP(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.RestartFTP(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	SuccessMessage(w, "FTP服务重启成功")
}

func (h *Handler) reloadFTP(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.ReloadSites(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	SuccessMessage(w, "配置已重载")
}

// ==================== FTP 文件管理 ====================

func (h *Handler) listFtpFiles(w http.ResponseWriter, r *http.Request) {
	subPath := r.URL.Query().Get("path")
	dirsOnly := r.URL.Query().Get("dirsOnly") == "true"

	if strings.Contains(subPath, "..") {
		BadRequest(w, "Invalid path")
		return
	}

	absRoot, _ := filepath.Abs(h.Config.FTP.Root)
	absPath := absRoot
	if subPath != "" && subPath != "/" {
		absPath = filepath.Join(absRoot, subPath)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		// 如果只需要目录，跳过文件
		if dirsOnly && !entry.IsDir() {
			continue
		}
		info, _ := entry.Info()
		files = append(files, map[string]interface{}{
			"name": entry.Name(), "is_dir": entry.IsDir(),
			"size": info.Size(), "modified": info.ModTime().Format(time.RFC3339),
		})
	}
	Success(w, map[string]interface{}{"path": subPath, "files": files})
}

func (h *Handler) uploadFtpFile(w http.ResponseWriter, r *http.Request) {
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

	targetDir := filepath.Join(h.Config.FTP.Root, destPath)
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

func (h *Handler) deleteFtpFile(w http.ResponseWriter, r *http.Request) {
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
	fullPath := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(req.Path, "/"), req.Name)
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(h.Config.FTP.Root)
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

func (h *Handler) mkdirFtp(w http.ResponseWriter, r *http.Request) {
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

	newDir := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(req.Path, "/"), req.Name)
	if err := os.MkdirAll(newDir, 0755); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	SuccessMessage(w, "目录创建成功")
}

// downloadFtpFile 下载 FTP 文件
func (h *Handler) downloadFtpFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	name := r.URL.Query().Get("name")

	if strings.Contains(path, "..") || strings.Contains(name, "..") {
		Error(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// 构建完整路径
	fullPath := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(path, "/"), name)
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(h.Config.FTP.Root)
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

// renameFtpFile 重命名 FTP 文件
func (h *Handler) renameFtpFile(w http.ResponseWriter, r *http.Request) {
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
	oldPath := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(req.Path, "/"), req.OldName)
	absOld, _ := filepath.Abs(oldPath)
	absRoot, _ := filepath.Abs(h.Config.FTP.Root)
	if !strings.HasPrefix(absOld, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 目标文件路径
	newPath := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(req.Path, "/"), req.NewName)
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

// copyFtpFile 复制 FTP 文件
func (h *Handler) copyFtpFile(w http.ResponseWriter, r *http.Request) {
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
	srcPath := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(req.SrcPath, "/"), req.SrcName)
	absSrc, _ := filepath.Abs(srcPath)
	absRoot, _ := filepath.Abs(h.Config.FTP.Root)
	if !strings.HasPrefix(absSrc, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 目标文件路径
	dstPath := filepath.Join(h.Config.FTP.Root, strings.TrimPrefix(req.SrcPath, "/"), req.DstName)
	absDst, _ := filepath.Abs(dstPath)
	if !strings.HasPrefix(absDst, absRoot) {
		Forbidden(w, "Access denied")
		return
	}

	// 打开源文件
	srcFile, err := os.Open(absSrc)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(absDst)
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

// ==================== FTP 用户管理 ====================

func (h *Handler) listFtpUsers(w http.ResponseWriter, r *http.Request) {
	users := make([]map[string]interface{}, 0)
	for _, u := range h.Config.FTP.Users {
		status := u.Status
		if status == "" {
			status = "enabled"
		}
		users = append(users, map[string]interface{}{
			"username":   u.Username,
			"password":   u.Password,
			"rootPath":   u.RootPath,
			"status":     status,
			"quota":      u.Quota,
			"expiryDays": u.ExpiryDays,
			"remark":     u.Remark,
		})
	}
	Success(w, map[string]interface{}{
		"users": users,
		"total": len(users),
	})
}

func (h *Handler) addFtpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		RootPath string `json:"rootPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	if req.Username == "" {
		BadRequest(w, "用户名不能为空")
		return
	}

	// 检查用户是否已存在
	for _, u := range h.Config.FTP.Users {
		if u.Username == req.Username {
			BadRequest(w, "用户已存在")
			return
		}
	}

	// 设置默认根目录
	if req.RootPath == "" {
		req.RootPath = filepath.Join(h.Config.FTP.Root, req.Username)
	}

	// 自动创建用户目录
	if err := os.MkdirAll(req.RootPath, 0755); err != nil {
		InternalServerError(w, "创建用户目录失败: "+err.Error())
		return
	}

	// 添加用户到配置
	h.Config.FTP.Users = append(h.Config.FTP.Users, config.FTPUser{
		Username: req.Username,
		Password: req.Password,
		RootPath: req.RootPath,
	})

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		// 同步到 ConfigManager
		h.ConfigManager.FTP.Users = h.Config.FTP.Users
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败: "+err.Error())
			return
		}
	}

	Success(w, map[string]interface{}{
		"username": req.Username,
		"rootPath": req.RootPath,
	})
}

func (h *Handler) deleteFtpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从 URL 或 body 获取用户名
	username := r.URL.Query().Get("username")
	if username == "" {
		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			BadRequest(w, "Invalid JSON")
			return
		}
		username = req.Username
	}

	if username == "" {
		BadRequest(w, "用户名不能为空")
		return
	}

	// 查找并删除用户
	found := false
	for i, u := range h.Config.FTP.Users {
		if u.Username == username {
			h.Config.FTP.Users = append(h.Config.FTP.Users[:i], h.Config.FTP.Users[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		BadRequest(w, "用户不存在")
		return
	}

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		h.ConfigManager.FTP.Users = h.Config.FTP.Users
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败: "+err.Error())
			return
		}
	}

	SuccessMessage(w, "用户已删除")
}

func (h *Handler) toggleFtpUserStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	// 查找用户
	for i, u := range h.Config.FTP.Users {
		if u.Username == req.Username {
			// 更新状态（存储在用户对象的扩展字段中）
			// 注意：当前 FTPUser 结构可能没有 status 字段，这里用备注字段暂存
			h.Config.FTP.Users[i].Status = map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]
			break
		}
	}

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		h.ConfigManager.FTP.Users = h.Config.FTP.Users
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败: "+err.Error())
			return
		}
	}

	SuccessMessage(w, "状态已更新")
}

func (h *Handler) batchFtpUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Action   string   `json:"action"`   // enable, disable, delete
		Usernames []string `json:"usernames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	switch req.Action {
	case "delete":
		// 批量删除
		deleteSet := make(map[string]bool)
		for _, u := range req.Usernames {
			deleteSet[u] = true
		}
		newUsers := make([]config.FTPUser, 0)
		for _, u := range h.Config.FTP.Users {
			if !deleteSet[u.Username] {
				newUsers = append(newUsers, u)
			}
		}
		h.Config.FTP.Users = newUsers

	case "enable", "disable":
		// 批量启用/禁用
		status := "enabled"
		if req.Action == "disable" {
			status = "disabled"
		}
		userSet := make(map[string]bool)
		for _, u := range req.Usernames {
			userSet[u] = true
		}
		for i, u := range h.Config.FTP.Users {
			if userSet[u.Username] {
				h.Config.FTP.Users[i].Status = status
			}
		}
	}

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		h.ConfigManager.FTP.Users = h.Config.FTP.Users
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败: "+err.Error())
			return
		}
	}

	SuccessMessage(w, "批量操作完成")
}
