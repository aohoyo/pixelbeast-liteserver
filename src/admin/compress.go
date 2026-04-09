package admin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"pixelbeast/src/handlers"
	"strings"
)

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
		err = h.createZip(srcDir, req.Files, outputPath)
	case "tar.gz", "targz":
		outputPath = filepath.Join(srcDir, targetName+".tar.gz")
		err = h.createTarGz(srcDir, req.Files, outputPath)
	default:
		BadRequest(w, "Unsupported format: "+req.Format)
		return
	}

	if err != nil {
		InternalServerError(w, "压缩失败: "+err.Error())
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelFileOp(username, "压缩", req.Target, true)

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
		err = h.extractZip(srcFile, srcDir)
	case ".gz":
		// 可能是 .tar.gz
		if strings.HasSuffix(strings.ToLower(req.Name), ".tar.gz") {
			err = h.extractTarGz(srcFile, srcDir)
		} else {
			err = h.extractGz(srcFile, srcDir)
		}
	default:
		// 检查是否是 .tar.gz
		if strings.HasSuffix(strings.ToLower(req.Name), ".tar.gz") || strings.HasSuffix(strings.ToLower(req.Name), ".tgz") {
			err = h.extractTarGz(srcFile, srcDir)
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
	handlers.LogPanelFileOp(username, "解压", req.Path, true)
	SuccessMessage(w, "解压成功")
}

// ==================== ZIP 操作 ====================

// createZip 创建 zip 压缩包
func (h *Handler) createZip(srcDir string, files []string, outputPath string) error {
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		filePath := filepath.Join(srcDir, file)
		if err := h.addToZip(zipWriter, filePath, srcDir, file); err != nil {
			return err
		}
	}

	return nil
}

// addToZip 添加文件/文件夹到 zip
func (h *Handler) addToZip(zipWriter *zip.Writer, filePath, baseDir, relPath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// 创建 zip 条目
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = relPath
	if info.IsDir() {
		header.Name += "/"
	} else {
		header.Method = zip.Deflate
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// 目录：递归添加内容
		entries, err := os.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			childPath := filepath.Join(filePath, entry.Name())
			childRel := filepath.Join(relPath, entry.Name())
			if err := h.addToZip(zipWriter, childPath, baseDir, childRel); err != nil {
				return err
			}
		}
		return nil
	}

	// 文件：写入内容
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}

// extractZip 解压 zip 文件
func (h *Handler) extractZip(srcFile, destDir string) error {
	reader, err := zip.OpenReader(srcFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		destPath := filepath.Join(destDir, file.Name)

		// 安全检查：防止 zip slip
		if !filepath.IsAbs(destPath) {
			absDest, _ := filepath.Abs(destPath)
			absDestDir, _ := filepath.Abs(destDir)
			if !strings.HasPrefix(absDest, absDestDir) {
				return fmt.Errorf("invalid file path in zip: %s", file.Name)
			}
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(destPath, file.Mode())
			continue
		}

		// 创建父目录
		os.MkdirAll(filepath.Dir(destPath), 0755)

		// 解压文件
		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			destFile.Close()
			return err
		}

		_, err = io.Copy(destFile, src)
		src.Close()
		destFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// ==================== TAR.GZ 操作 ====================

// createTarGz 创建 tar.gz 压缩包
func (h *Handler) createTarGz(srcDir string, files []string, outputPath string) error {
	tarFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for _, file := range files {
		filePath := filepath.Join(srcDir, file)
		if err := h.addToTar(tarWriter, filePath, srcDir, file); err != nil {
			return err
		}
	}

	return nil
}

// addToTar 添加文件/文件夹到 tar
func (h *Handler) addToTar(tarWriter *tar.Writer, filePath, baseDir, relPath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// 创建 tar 条目
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relPath

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		// 目录：递归添加内容
		entries, err := os.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			childPath := filepath.Join(filePath, entry.Name())
			childRel := filepath.Join(relPath, entry.Name())
			if err := h.addToTar(tarWriter, childPath, baseDir, childRel); err != nil {
				return err
			}
		}
		return nil
	}

	// 文件：写入内容
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tarWriter, file)
	return err
}

// extractTarGz 解压 tar.gz 文件
func (h *Handler) extractTarGz(srcFile, destDir string) error {
	file, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, header.Name)

		// 安全检查：防止 tar slip
		absDest, _ := filepath.Abs(destPath)
		absDestDir, _ := filepath.Abs(destDir)
		if !strings.HasPrefix(absDest, absDestDir) {
			return fmt.Errorf("invalid file path in tar: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(destPath, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(destPath), 0755)
			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(destFile, tarReader); err != nil {
				destFile.Close()
				return err
			}
			destFile.Close()
		}
	}

	return nil
}

// extractGz 解压单个 .gz 文件
func (h *Handler) extractGz(srcFile, destDir string) error {
	file, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	// 目标文件名：去掉 .gz 后缀
	destName := strings.TrimSuffix(filepath.Base(srcFile), ".gz")
	destPath := filepath.Join(destDir, destName)

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, gzReader)
	return err
}
