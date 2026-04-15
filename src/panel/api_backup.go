package panel

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"pixelbeast/src/backup"
	"pixelbeast/src/logger"
)

// ==================== 备份管理 ====================

// listBackups 列出备份文件
func (h *Handler) listBackups(w http.ResponseWriter, r *http.Request) {
	backupDir := resolvePath(h.ConfigManager.GetBackupDir())

	backups, err := backup.ListBackups(backupDir)
	if err != nil {
		InternalServerErrorLog(w, err)
		return
	}
	if backups == nil {
		backups = []backup.BackupInfo{}
	}

	Success(w, map[string]interface{}{
		"backups": backups,
		"dir":     h.ConfigManager.GetBackupDir(),
	})
}

// createBackup 手动创建备份
func (h *Handler) createBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	backupDir := resolvePath(h.ConfigManager.GetBackupDir())
	items := h.ConfigManager.Server.Backup.Items

	dirs := map[string]string{
		"config": resolvePath(h.ConfigManager.ConfigDir()),
		"sites":  resolvePath(h.ConfigManager.GetSitesDir()),
		"ftp":    resolvePath(h.ConfigManager.GetFTPRoot()),
	}

	backupName, err := backup.CreateBackup(backupDir, items, dirs)
	if err != nil {
		InternalServerErrorLog(w, err)
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelConfigChange(username, "创建备份 "+backupName, true)
	Success(w, map[string]interface{}{
		"name":    backupName,
		"message": "备份创建成功",
	})
}

// deleteBackup 删除备份
func (h *Handler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	name, ok := data["name"].(string)
	if !ok {
		BadRequest(w, "备份文件名不能为空")
		return
	}

	backupDir := resolvePath(h.ConfigManager.GetBackupDir())
	if err := backup.DeleteBackup(backupDir, name); err != nil {
		InternalServerErrorLog(w, err)
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelConfigChange(username, "删除备份 "+name, true)
	SuccessMessage(w, "备份已删除")
}

// downloadBackup 下载备份文件
func (h *Handler) downloadBackup(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if err := backup.ValidateBackupName(name); err != nil {
		BadRequest(w, "无效的备份文件名")
		return
	}

	backupDir := resolvePath(h.ConfigManager.GetBackupDir())
	absPath := filepath.Join(backupDir, name)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		BadRequest(w, "备份文件不存在")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, absPath)
}

// restoreBackup 从备份恢复
func (h *Handler) restoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	if err := backup.ValidateBackupName(req.Name); err != nil {
		BadRequest(w, "无效的备份文件名")
		return
	}

	backupDir := resolvePath(h.ConfigManager.GetBackupDir())
	configDir := h.ConfigManager.ConfigDir()

	if err := backup.RestoreBackup(backupDir, req.Name, configDir); err != nil {
		InternalServerErrorLog(w, err)
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelConfigChange(username, "从备份恢复 "+req.Name, true)
	SuccessMessage(w, "备份恢复成功，重新加载配置生效")
}
