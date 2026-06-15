package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupInfo 备份文件元信息
type BackupInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// ListBackups 列出备份目录中的备份文件，按修改时间倒序
func ListBackups(backupDir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取备份目录失败: %w", err)
	}

	backups := make([]BackupInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		if !(strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")) {
			continue
		}
		backups = append(backups, BackupInfo{
			Name:     name,
			Size:     info.Size(),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// 按修改时间倒序（ReadDir 已按名称排序，取反即可）
	for i, j := 0, len(backups)-1; i < j; i, j = i+1, j-1 {
		backups[i], backups[j] = backups[j], backups[i]
	}

	return backups, nil
}

// CreateBackup 创建 tar.gz 备份，items 决定备份内容（config/sites/ftp）
// 返回备份文件名
func CreateBackup(backupDir string, items []string, dirs map[string]string) (string, error) {
	if len(items) == 0 {
		items = []string{"config"}
	}

	os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("2006-01-02_150405")
	backupName := fmt.Sprintf("backup_%s.tar.gz", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	f, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("创建备份失败: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, item := range items {
		srcDir, ok := dirs[item]
		if !ok {
			continue
		}

		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			relPath, err := filepath.Rel(srcDir, path)
			if err != nil || relPath == "." {
				return nil
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return nil
			}
			header.Name = filepath.Join(item, relPath)
			if info.IsDir() {
				header.Name += "/"
			}
			if err := tw.WriteHeader(header); err != nil {
				return nil
			}
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return nil
				}
				defer file.Close()

				if _, copyErr := io.Copy(tw, file); copyErr != nil {
					return nil
				}
			}
			return nil
		})
	}

	return backupName, nil
}

// DeleteBackup 删除指定备份文件，name 需经过安全校验
func DeleteBackup(backupDir, name string) error {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("无效的备份文件名")
	}

	absPath := filepath.Join(backupDir, name)
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("删除备份失败: %w", err)
	}
	return nil
}

// ValidateBackupName 校验备份文件名安全性
func ValidateBackupName(name string) error {
	if name == "" {
		return fmt.Errorf("备份文件名不能为空")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("无效的备份文件名")
	}
	return nil
}

// RestoreBackup 从 tar.gz 备份恢复到目标目录
func RestoreBackup(backupDir, name, configDir string) error {
	absPath := filepath.Join(backupDir, name)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在")
	}

	tmpDir, err := os.MkdirTemp("", "pixelbeast-restore-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取备份失败: %w", err)
		}
		if strings.HasPrefix(header.Name, "/") || strings.Contains(header.Name, "..") {
			continue
		}
		target := filepath.Join(tmpDir, header.Name)
		if header.Typeflag == tar.TypeDir {
			os.MkdirAll(target, os.FileMode(header.Mode))
			continue
		}
		if header.Typeflag == tar.TypeReg {
			os.MkdirAll(filepath.Dir(target), 0755)
			out, err := os.Create(target)
			if err != nil {
				continue
			}
			io.Copy(out, tr)
			out.Close()
		}
	}

	return filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(tmpDir, path)
		dst := filepath.Join(configDir, rel)
		os.MkdirAll(filepath.Dir(dst), 0755)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		os.WriteFile(dst, data, info.Mode())
		return nil
	})
}

// CreateTarGz 创建 tar.gz 压缩包（通用工具函数）
func CreateTarGz(outputPath, srcDir, prefix string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = filepath.Join(prefix, relPath)
		if info.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		}

		return nil
	})
}
