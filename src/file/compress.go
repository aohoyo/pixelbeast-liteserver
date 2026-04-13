package file

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateZip 创建 zip 压缩包
func CreateZip(srcDir string, files []string, outputPath string) error {
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, f := range files {
		filePath := filepath.Join(srcDir, f)
		if err := addToZip(zipWriter, filePath, srcDir, f); err != nil {
			return err
		}
	}

	return nil
}

// addToZip 添加文件/文件夹到 zip
func addToZip(zipWriter *zip.Writer, filePath, baseDir, relPath string) error {
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
		entries, err := os.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			childPath := filepath.Join(filePath, entry.Name())
			childRel := filepath.Join(relPath, entry.Name())
			if err := addToZip(zipWriter, childPath, baseDir, childRel); err != nil {
				return err
			}
		}
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}

// ExtractZip 解压 zip 文件
func ExtractZip(srcFile, destDir string) error {
	reader, err := zip.OpenReader(srcFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.File {
		destPath := filepath.Join(destDir, f.Name)

		// 安全检查：防止 zip slip
		if !filepath.IsAbs(destPath) {
			absDest, _ := filepath.Abs(destPath)
			absDestDir, _ := filepath.Abs(destDir)
			if !strings.HasPrefix(absDest, absDestDir) {
				return fmt.Errorf("invalid file path in zip: %s", f.Name)
			}
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, f.Mode())
			continue
		}

		os.MkdirAll(filepath.Dir(destPath), 0755)

		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		src, err := f.Open()
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

// CreateTarGz 创建 tar.gz 压缩包
func CreateTarGz(srcDir string, files []string, outputPath string) error {
	tarFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for _, f := range files {
		filePath := filepath.Join(srcDir, f)
		if err := addToTar(tarWriter, filePath, srcDir, f); err != nil {
			return err
		}
	}

	return nil
}

// addToTar 添加文件/文件夹到 tar
func addToTar(tarWriter *tar.Writer, filePath, baseDir, relPath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relPath

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			childPath := filepath.Join(filePath, entry.Name())
			childRel := filepath.Join(relPath, entry.Name())
			if err := addToTar(tarWriter, childPath, baseDir, childRel); err != nil {
				return err
			}
		}
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tarWriter, file)
	return err
}

// ExtractTarGz 解压 tar.gz 文件
func ExtractTarGz(srcFile, destDir string) error {
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

// ExtractGz 解压单个 .gz 文件
func ExtractGz(srcFile, destDir string) error {
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
