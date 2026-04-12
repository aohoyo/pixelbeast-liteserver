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
	"pixelbeast/src/handlers"
)

// ==================== FTP 状态 ====================

func (h *Handler) getFtpStatus(w http.ResponseWriter, r *http.Request) {
	ftpRunning := h.ConfigManager.FTP.Enabled
	ftpPort := h.ConfigManager.FTP.Port
	if h.ServerManager != nil {
		ftpRunning = h.ServerManager.IsFTPRunning()
	}
	Success(w, map[string]interface{}{
		"running": ftpRunning,
		"port":    ftpPort,
	})
}

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

	// 用户主动切换时保存配置
	h.ConfigManager.FTP.Enabled = h.ServerManager.IsFTPRunning()
	if err := h.ConfigManager.Save(); err != nil {
		Error(w, http.StatusInternalServerError, "保存配置失败")
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 切换FTP服务: %s (用户=%s)", msg, username)
	SuccessMessage(w, msg)
}

// handleFTPServiceAction 通用 FTP 服务操作处理器
// saveConfig: 需要保存配置时传入（如启停操作），否则传 nil
func (h *Handler) handleFTPServiceAction(w http.ResponseWriter, r *http.Request, action func() error, actionName string, saveConfig func() error) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := action(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	if saveConfig != nil {
		if err := saveConfig(); err != nil {
			Error(w, http.StatusInternalServerError, "保存配置失败")
			return
		}
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] %sFTP服务 (用户=%s)", actionName, username)
	SuccessMessage(w, "FTP服务已"+actionName)
}

func (h *Handler) startFTP(w http.ResponseWriter, r *http.Request) {
	h.handleFTPServiceAction(w, r, h.ServerManager.StartFTP, "启动", func() error {
		h.ConfigManager.FTP.Enabled = true
		return h.ConfigManager.Save()
	})
}

func (h *Handler) stopFTP(w http.ResponseWriter, r *http.Request) {
	h.handleFTPServiceAction(w, r, h.ServerManager.StopFTP, "停止", func() error {
		h.ConfigManager.FTP.Enabled = false
		return h.ConfigManager.Save()
	})
}

func (h *Handler) restartFTP(w http.ResponseWriter, r *http.Request) {
	h.handleFTPServiceAction(w, r, h.ServerManager.RestartFTP, "重启", nil)
}

func (h *Handler) reloadFTP(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	// 重载配置文件
	if err := h.ServerManager.ReloadSites(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	// 同步管理面板和 FTP 服务器的配置指针
	h.ConfigManager = h.ServerManager.ConfigManager
	h.ServerManager.SyncFTPConfig()
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 重载FTP配置 (用户=%s)", username)
	SuccessMessage(w, "配置已重载")
}

func (h *Handler) saveFtpPort(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "方法不允许")
		return
	}
	var data struct {
		Port int `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil || data.Port < 1 || data.Port > 65535 {
		BadRequest(w, "端口无效，范围 1-65535")
		return
	}
	h.ConfigManager.FTP.Port = data.Port
	if err := h.ConfigManager.Save(); err != nil {
		Error(w, http.StatusInternalServerError, "保存配置失败")
		return
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 修改端口: %d (用户=%s)", data.Port, username)
	SuccessMessage(w, "FTP 端口已更新，重启服务生效")
}

// ==================== FTP 文件管理 ====================

func (h *Handler) listFtpFiles(w http.ResponseWriter, r *http.Request) {
	subPath := r.URL.Query().Get("path")
	dirsOnly := r.URL.Query().Get("dirsOnly") == "true"

	if strings.Contains(subPath, "..") {
		BadRequest(w, "无效路径")
		return
	}

	absRoot, _ := filepath.Abs(h.ConfigManager.GetFTPRoot())
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
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
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
		BadRequest(w, "无效路径")
		return
	}

	targetDir := filepath.Join(h.ConfigManager.GetFTPRoot(), destPath)
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
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
		BadRequest(w, "无效路径")
		return
	}

	// 构建完整路径
	fullPath := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(req.Path, "/"), req.Name)
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(h.ConfigManager.GetFTPRoot())
	if !strings.HasPrefix(absPath, absRoot) {
		Forbidden(w, "访问被拒绝")
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
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	if strings.Contains(req.Path, "..") || strings.Contains(req.Name, "..") {
		BadRequest(w, "无效路径")
		return
	}

	newDir := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(req.Path, "/"), req.Name)
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
		Error(w, http.StatusBadRequest, "无效路径")
		return
	}

	// 构建完整路径
	fullPath := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(path, "/"), name)
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(h.ConfigManager.GetFTPRoot())
	if !strings.HasPrefix(absPath, absRoot) {
		Forbidden(w, "访问被拒绝")
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
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	var req struct {
		Path    string `json:"path"`
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	if strings.Contains(req.Path, "..") || strings.Contains(req.OldName, "..") || strings.Contains(req.NewName, "..") {
		BadRequest(w, "无效路径")
		return
	}

	// 源文件路径
	oldPath := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(req.Path, "/"), req.OldName)
	absOld, _ := filepath.Abs(oldPath)
	absRoot, _ := filepath.Abs(h.ConfigManager.GetFTPRoot())
	if !strings.HasPrefix(absOld, absRoot) {
		Forbidden(w, "访问被拒绝")
		return
	}

	// 目标文件路径
	newPath := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(req.Path, "/"), req.NewName)
	absNew, _ := filepath.Abs(newPath)
	if !strings.HasPrefix(absNew, absRoot) {
		Forbidden(w, "访问被拒绝")
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
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	var req struct {
		SrcPath string `json:"srcPath"`
		SrcName string `json:"srcName"`
		DstName string `json:"dstName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	if strings.Contains(req.SrcPath, "..") || strings.Contains(req.SrcName, "..") || strings.Contains(req.DstName, "..") {
		BadRequest(w, "无效路径")
		return
	}

	// 源文件路径
	srcPath := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(req.SrcPath, "/"), req.SrcName)
	absSrc, _ := filepath.Abs(srcPath)
	absRoot, _ := filepath.Abs(h.ConfigManager.GetFTPRoot())
	if !strings.HasPrefix(absSrc, absRoot) {
		Forbidden(w, "访问被拒绝")
		return
	}

	// 目标文件路径
	dstPath := filepath.Join(h.ConfigManager.GetFTPRoot(), strings.TrimPrefix(req.SrcPath, "/"), req.DstName)
	absDst, _ := filepath.Abs(dstPath)
	if !strings.HasPrefix(absDst, absRoot) {
		Forbidden(w, "访问被拒绝")
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

// calculateDirSize 计算目录总大小
func calculateDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func (h *Handler) listFtpUsers(w http.ResponseWriter, r *http.Request) {
	users := make([]map[string]interface{}, 0)
	for _, u := range h.ConfigManager.FTP.Users {
		status := u.Status
		if status == "" {
			status = "enabled"
		}

		// 计算已用空间
		var usedSpace int64
		if u.RootPath != "" {
			if info, err := os.Stat(u.RootPath); err == nil && info.IsDir() {
				usedSpace = calculateDirSize(u.RootPath)
			}
		}

		users = append(users, map[string]interface{}{
			"username":       u.Username,
			"password":       "••••••••",
			"rootPath":       u.RootPath,
			"status":         status,
			"quota":          u.Quota,
			"usedSpace":      usedSpace,
			"expiryDays":     u.ExpiryDays,
			"expiryDate":     u.ExpiryDate,
			"remark":         u.Remark,
			"speedLimit":     u.SpeedLimit,
			"maxConnections": u.MaxConnections,
			"bandwidth":      u.Bandwidth,
			"maxFiles":       u.MaxFiles,
			"maxFileSize":    u.MaxFileSize,
		})
	}
	Success(w, map[string]interface{}{
		"users": users,
		"total": len(users),
	})
}

func (h *Handler) addFtpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RootPath   string `json:"rootPath"`
		Quota      int64  `json:"quota"`
		ExpiryDays int    `json:"expiryDays"`
		Status     string `json:"status"`
		Remark     string `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	if req.Username == "" {
		BadRequest(w, "用户名不能为空")
		return
	}

	if req.Password == "" {
		BadRequest(w, "密码不能为空")
		return
	}

	// 检查用户是否已存在
	for _, u := range h.ConfigManager.FTP.Users {
		if u.Username == req.Username {
			BadRequest(w, "用户已存在")
			return
		}
	}

	// 设置默认根目录
	if req.RootPath == "" {
		req.RootPath = filepath.Join(h.ConfigManager.GetFTPRoot(), req.Username)
	}

	// 自动创建用户目录
	if err := os.MkdirAll(req.RootPath, 0755); err != nil {
		InternalServerError(w, "创建用户目录失败: "+err.Error())
		return
	}

	// 加密密码
	encryptedPassword, err := h.ConfigManager.EncryptPassword(req.Password)
	if err != nil {
		InternalServerError(w, "密码加密失败: "+err.Error())
		return
	}

	// 设置默认状态
	if req.Status == "" {
		req.Status = "enabled"
	}

	// 计算过期时间
	var expiryDate string
	if req.ExpiryDays > 0 {
		expiryDate = time.Now().AddDate(0, 0, req.ExpiryDays).Format("2006-01-02")
	}

	// 添加用户到配置
	h.ConfigManager.FTP.Users = append(h.ConfigManager.FTP.Users, config.FTPUser{
		Username:   req.Username,
		Password:   encryptedPassword,
		RootPath:   req.RootPath,
		Status:     req.Status,
		Quota:      req.Quota,
		ExpiryDays: req.ExpiryDays,
		ExpiryDate: expiryDate,
		Remark:     req.Remark,
	})

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败: "+err.Error())
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 添加用户: %s (用户=%s)", req.Username, username)

	Success(w, map[string]interface{}{
		"username": req.Username,
		"rootPath": req.RootPath,
		"status":   req.Status,
	})
}

func (h *Handler) deleteFtpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 body 获取参数
	var req struct {
		Username    string `json:"username"`
		DeleteFiles bool   `json:"deleteFiles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	if req.Username == "" {
		BadRequest(w, "用户名不能为空")
		return
	}

	// 查找用户并获取其根目录
	var userRootPath string
	found := false
	for i, u := range h.ConfigManager.FTP.Users {
		if u.Username == req.Username {
			userRootPath = u.RootPath
			h.ConfigManager.FTP.Users = append(h.ConfigManager.FTP.Users[:i], h.ConfigManager.FTP.Users[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		BadRequest(w, "用户不存在")
		return
	}

	// 如果勾选删除文件，删除用户目录
	if req.DeleteFiles && userRootPath != "" {
		// 安全检查：确保路径在 FTP 根目录下
		absRoot, _ := filepath.Abs(h.ConfigManager.GetFTPRoot())
		absUserPath, _ := filepath.Abs(userRootPath)

		if strings.HasPrefix(absUserPath, absRoot) {
			if err := os.RemoveAll(absUserPath); err != nil {
				// 记录错误但继续删除用户
				fmt.Printf("[FTP] 删除用户目录失败: %v\n", err)
			}
		}
	}

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败: "+err.Error())
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 删除用户: %s (用户=%s)", req.Username, username)
	SuccessMessage(w, "用户已删除")
}

func (h *Handler) toggleFtpUserStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var req struct {
		Username string `json:"username"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	// 查找用户并更新状态
	found := false
	for i := range h.ConfigManager.FTP.Users {
		if h.ConfigManager.FTP.Users[i].Username == req.Username {
			if req.Enabled {
				h.ConfigManager.FTP.Users[i].Status = "enabled"
			} else {
				h.ConfigManager.FTP.Users[i].Status = "disabled"
			}
			found = true
			break
		}
	}

	if !found {
		BadRequest(w, "用户不存在")
		return
	}

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败: "+err.Error())
		return
	}

	// 同步 FTP 服务器，使状态变更立即生效
	if h.ServerManager != nil {
		h.ServerManager.SyncFTPConfig()
	}

	statusText := "已禁用"
	if req.Enabled {
		statusText = "已启用"
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 切换用户状态: %s -> %s (用户=%s)", req.Username, statusText, username)
	SuccessMessage(w, "用户"+statusText)
}

func (h *Handler) batchFtpUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var req struct {
		Action    string   `json:"action"` // enable, disable, delete
		Usernames []string `json:"usernames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
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
		for _, u := range h.ConfigManager.FTP.Users {
			if !deleteSet[u.Username] {
				newUsers = append(newUsers, u)
			}
		}
		h.ConfigManager.FTP.Users = newUsers

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
		for i, u := range h.ConfigManager.FTP.Users {
			if userSet[u.Username] {
				h.ConfigManager.FTP.Users[i].Status = status
			}
		}
	}

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败: "+err.Error())
			return
		}
	}

	// 同步 FTP 服务器，使状态变更立即生效
	if h.ServerManager != nil {
		h.ServerManager.SyncFTPConfig()
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 批量%s: %d个用户 (用户=%s)", req.Action, len(req.Usernames), username)
	SuccessMessage(w, "批量操作完成")
}

// ==================== FTP 用户详情路由 ====================

// handleFtpUserDetail 处理带用户名的 FTP 用户路由
func (h *Handler) handleFtpUserDetail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// 匹配 /api/ftp/users/{username}/config
	if strings.HasSuffix(path, "/config") {
		if r.Method != http.MethodPost {
			MethodNotAllowed(w, "方法不允许")
			return
		}
		username := strings.TrimSuffix(path, "/config")
		username = strings.TrimPrefix(username, "/api/ftp/users/")
		if username == "" {
			BadRequest(w, "用户名不能为空")
			return
		}
		h.updateFtpUserConfig(w, r, username)
		return
	}

	// 匹配 /api/ftp/users/{username}
	username := strings.TrimPrefix(path, "/api/ftp/users/")
	if username == "" || strings.Contains(username, "/") {
		BadRequest(w, "无效的用户名")
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateFtpUser(w, r, username)
	default:
		MethodNotAllowed(w, "方法不允许")
	}
}

// updateFtpUser 更新 FTP 用户信息
func (h *Handler) updateFtpUser(w http.ResponseWriter, r *http.Request, username string) {
	var req struct {
		RootPath   string `json:"rootPath"`
		Password   string `json:"password"`
		Quota      int64  `json:"quota"`
		ExpiryDays int    `json:"expiryDays"`
		Status     string `json:"status"`
		Remark     string `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	user := h.ConfigManager.GetFTPUser(username)
	if user == nil {
		BadRequest(w, "用户不存在")
		return
	}

	// 更新密码（单独处理，因为 UpdateFTPUser 会保留原密码）
	if req.Password != "" {
		if err := h.ConfigManager.SetFTPUserPassword(username, req.Password); err != nil {
			InternalServerError(w, "密码更新失败: "+err.Error())
			return
		}
	}

	// 更新其他字段
	updated := *user
	updated.RootPath = req.RootPath
	updated.Quota = req.Quota
	updated.ExpiryDays = req.ExpiryDays
	updated.Status = req.Status
	updated.Remark = req.Remark

	// 过期天数变更时重新计算过期日期
	if req.ExpiryDays > 0 && req.ExpiryDays != user.ExpiryDays {
		updated.ExpiryDate = time.Now().AddDate(0, 0, req.ExpiryDays).Format("2006-01-02")
	}

	if err := h.ConfigManager.UpdateFTPUser(username, updated); err != nil {
		InternalServerError(w, "更新用户失败: "+err.Error())
		return
	}

	sessionUser := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 更新用户: %s (用户=%s)", username, sessionUser)
	SuccessMessage(w, "用户已更新")
}

// updateFtpUserConfig 更新 FTP 用户配置（速度限制、连接数等）
func (h *Handler) updateFtpUserConfig(w http.ResponseWriter, r *http.Request, username string) {
	var req struct {
		SpeedLimit     int64 `json:"speedLimit"`
		MaxConnections int   `json:"maxConnections"`
		Bandwidth      int64 `json:"bandwidth"`
		MaxFiles       int   `json:"maxFiles"`
		MaxFileSize    int64 `json:"maxFileSize"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	user := h.ConfigManager.GetFTPUser(username)
	if user == nil {
		BadRequest(w, "用户不存在")
		return
	}

	updated := *user
	updated.SpeedLimit = req.SpeedLimit
	updated.MaxConnections = req.MaxConnections
	updated.Bandwidth = req.Bandwidth
	updated.MaxFiles = req.MaxFiles
	updated.MaxFileSize = req.MaxFileSize

	if err := h.ConfigManager.UpdateFTPUser(username, updated); err != nil {
		InternalServerError(w, "保存配置失败: "+err.Error())
		return
	}

	sessionUser := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[FTP] 更新用户配置: %s (用户=%s)", username, sessionUser)
	SuccessMessage(w, "配置已保存")
}
