package panel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pixelbeast/src/logger"
)

// ==================== 回收站管理 ====================

// trashMeta 回收站条目元数据
type trashMeta struct {
	OriginalPath string `json:"original_path"` // 原始完整路径
	OriginalName string `json:"original_name"` // 原始文件名
	DeletedAt    string `json:"deleted_at"`    // 删除时间
	IsDir        bool   `json:"is_dir"`        // 是否为目录
	Size         int64  `json:"size"`          // 文件/目录大小（字节）
}

// trashItem 回收站条目（API 响应用）
type trashItem struct {
	ID           string `json:"id"`            // 回收站条目 ID（目录名）
	OriginalPath string `json:"original_path"` // 原始路径
	OriginalName string `json:"original_name"` // 原始文件名
	DeletedAt    string `json:"deleted_at"`    // 删除时间
	IsDir        bool   `json:"is_dir"`        // 是否为目录
	Size         int64  `json:"size"`          // 大小
}

// getTrashDir 获取回收站根目录
func getTrashDir() string {
	rootDir, _ := os.Getwd()
	return filepath.Join(rootDir, ".trash")
}

// getTrashEntryPath 获取回收站条目路径
func getTrashEntryPath(id string) string {
	return filepath.Join(getTrashDir(), id)
}

// moveToTrash 将文件/目录移入回收站，返回回收站条目 ID
func moveToTrash(absPath string) (string, error) {
	trashDir := getTrashDir()
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return "", fmt.Errorf("创建回收站目录失败: %w", err)
	}

	// 生成唯一 ID：时间戳 + 随机后缀
	id := fmt.Sprintf("%d_%s", time.Now().UnixMilli(), randomHex(6))
	entryDir := filepath.Join(trashDir, id)
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return "", fmt.Errorf("创建回收站条目失败: %w", err)
	}

	// 目标路径（在回收站条目内保留原文件名）
	fileName := filepath.Base(absPath)
	dstPath := filepath.Join(entryDir, fileName)

	// 先尝试 Rename（同设备最快）
	err := os.Rename(absPath, dstPath)
	if err != nil {
		// 跨设备 Rename 失败，回退到复制+删除
		if isCrossDeviceErr(err) {
			if copyErr := copyRecursive(absPath, dstPath); copyErr != nil {
				os.RemoveAll(entryDir)
				return "", fmt.Errorf("复制到回收站失败: %w", copyErr)
			}
			if removeErr := os.RemoveAll(absPath); removeErr != nil {
				// 回滚：删掉已复制的
				os.RemoveAll(entryDir)
				return "", fmt.Errorf("删除原文件失败: %w", removeErr)
			}
		} else {
			os.RemoveAll(entryDir)
			return "", fmt.Errorf("移入回收站失败: %w", err)
		}
	}

	// 获取文件信息
	isDir := false
	var size int64
	if info, statErr := os.Stat(dstPath); statErr == nil {
		isDir = info.IsDir()
		if !isDir {
			size = info.Size()
		} else {
			size = dirSize(dstPath)
		}
	}

	// 写入元数据
	meta := trashMeta{
		OriginalPath: absPath,
		OriginalName: fileName,
		DeletedAt:    time.Now().Format("2006-01-02 15:04:05"),
		IsDir:        isDir,
		Size:         size,
	}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	if writeErr := os.WriteFile(filepath.Join(entryDir, ".meta.json"), metaData, 0600); writeErr != nil {
		// 元数据写入失败不影响主流程，记录日志
		logger.LogPanelRuntime(logger.LogLevelError, fmt.Sprintf("写入回收站元数据失败: %v", writeErr))
	}

	return id, nil
}

// listTrash 列出回收站内容
func (h *Handler) listTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	trashDir := getTrashDir()
	entries, err := os.ReadDir(trashDir)
	if err != nil {
		if os.IsNotExist(err) {
			Success(w, map[string]interface{}{
				"items": []trashItem{},
				"total": 0,
			})
			return
		}
		InternalServerErrorLog(w, err)
		return
	}

	items := make([]trashItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// 读取元数据
		metaPath := filepath.Join(trashDir, entry.Name(), ".meta.json")
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			// 没有元数据的条目跳过
			continue
		}
		var meta trashMeta
		if err := json.Unmarshal(metaData, &meta); err != nil {
			continue
		}

		items = append(items, trashItem{
			ID:           entry.Name(),
			OriginalPath: meta.OriginalPath,
			OriginalName: meta.OriginalName,
			DeletedAt:    meta.DeletedAt,
			IsDir:        meta.IsDir,
			Size:         meta.Size,
		})
	}

	// 按删除时间倒序
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID > items[j].ID
	})

	Success(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// restoreTrash 恢复回收站条目
func (h *Handler) restoreTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if req.ID == "" || strings.Contains(req.ID, "/") || strings.Contains(req.ID, "\\") {
		BadRequest(w, "无效的回收站条目 ID")
		return
	}

	entryPath := getTrashEntryPath(req.ID)
	metaPath := filepath.Join(entryPath, ".meta.json")

	// 读取元数据获取原始路径
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		BadRequest(w, "回收站条目不存在")
		return
	}
	var meta trashMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		InternalServerErrorLog(w, err)
		return
	}

	// 查找回收站条目中的实际文件（排除 .meta.json）
	entries, _ := os.ReadDir(entryPath)
	var fileEntry fs.DirEntry
	for _, e := range entries {
		if e.Name() != ".meta.json" {
			fileEntry = e
			break
		}
	}
	if fileEntry == nil {
		BadRequest(w, "回收站条目中无文件")
		return
	}

	srcPath := filepath.Join(entryPath, fileEntry.Name())
	dstPath := meta.OriginalPath

	// 检查目标路径是否已存在同名文件
	if _, statErr := os.Stat(dstPath); statErr == nil {
		// 已存在，添加后缀
		ext := filepath.Ext(meta.OriginalName)
		nameNoExt := strings.TrimSuffix(meta.OriginalName, ext)
		dstPath = filepath.Join(filepath.Dir(dstPath), nameNoExt+"_已恢复"+ext)
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		InternalServerErrorLog(w, fmt.Errorf("创建目标目录失败: %w", err))
		return
	}

	// 移回原位
	if err := os.Rename(srcPath, dstPath); err != nil {
		if isCrossDeviceErr(err) {
			if copyErr := copyRecursive(srcPath, dstPath); copyErr != nil {
				InternalServerErrorLog(w, fmt.Errorf("恢复文件失败: %w", copyErr))
				return
			}
			os.RemoveAll(entryPath)
		} else {
			InternalServerErrorLog(w, fmt.Errorf("恢复文件失败: %w", err))
			return
		}
	} else {
		// 移动成功，清理回收站条目
		os.RemoveAll(entryPath)
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "恢复", dstPath, true)
	SuccessMessage(w, "恢复成功")
}

// permanentDeleteTrash 永久删除回收站条目
func (h *Handler) permanentDeleteTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if req.ID == "" || strings.Contains(req.ID, "/") || strings.Contains(req.ID, "\\") {
		BadRequest(w, "无效的回收站条目 ID")
		return
	}

	entryPath := getTrashEntryPath(req.ID)
	if _, err := os.Stat(entryPath); err != nil {
		BadRequest(w, "回收站条目不存在")
		return
	}

	if err := os.RemoveAll(entryPath); err != nil {
		InternalServerErrorLog(w, err)
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "永久删除", entryPath, true)
	SuccessMessage(w, "已永久删除")
}

// clearTrash 清空回收站
func (h *Handler) clearTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	trashDir := getTrashDir()
	if err := os.RemoveAll(trashDir); err != nil {
		InternalServerErrorLog(w, err)
		return
	}

	username := h.getSessionUsername(r)
	logger.LogPanelFileOp(username, "清空回收站", trashDir, true)
	SuccessMessage(w, "回收站已清空")
}

// ==================== 回收站辅助函数 ====================

// randomHex 生成随机十六进制字符串
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// isCrossDeviceErr 判断是否为跨设备错误
func isCrossDeviceErr(err error) bool {
	return strings.Contains(err.Error(), "invalid cross-device link") ||
		strings.Contains(err.Error(), "cross-device")
}

// copyRecursive 递归复制文件或目录
func copyRecursive(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info)
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copyFile(srcPath, dstPath, info); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile 复制单个文件
func copyFile(src, dst string, info fs.FileInfo) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 使用 ReadFrom 进行高效复制
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}
	return nil
}

// dirSize 计算目录总大小
func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

