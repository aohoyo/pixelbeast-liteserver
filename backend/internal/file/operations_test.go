package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ==================== 路径安全测试 ====================

func TestCheckPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"正常路径", "normal/path", false},
		{"父目录遍历", "../etc/passwd", true},
		{"嵌套遍历", "path/../../etc", true},
		{"URL编码遍历", "..%2F..%2Fetc", true},
		{"安全路径", "safe/path/file.txt", false},
		{"空路径", "", false},
		{"单点", "./file", false},
		{"路径中间遍历", "a/../b/../../c", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPathTraversal(tt.path)
			if result != tt.expected {
				t.Errorf("CheckPathTraversal(%q) = %v, 期望 %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsPathWithin(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	tests := []struct {
		name     string
		path     string
		base     string
		expected bool
	}{
		{"根内文件", filepath.Join(tmpDir, "file.txt"), tmpDir, true},
		{"子目录文件", filepath.Join(subDir, "file.txt"), tmpDir, true},
		{"子目录自身", subDir, tmpDir, true},
		{"根目录自身", tmpDir, tmpDir, true},
		{"外部路径", "/tmp/other", tmpDir, false},
		{"遍历逃逸", filepath.Join(tmpDir, "..", "other"), tmpDir, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithin(tt.path, tt.base)
			if result != tt.expected {
				t.Errorf("IsPathWithin(%q, %q) = %v, 期望 %v", tt.path, tt.base, result, tt.expected)
			}
		})
	}
}

// ==================== 文件管理器测试 ====================

func TestNewFileManager(t *testing.T) {
	fm := NewFileManager()
	if fm == nil {
		t.Fatal("NewFileManager 返回 nil")
	}

	// 默认应有日志目录书签
	bm, ok := fm.GetBookmark("log")
	if !ok {
		t.Error("应有默认书签 'log'")
	}
	if bm.Name != "日志目录" {
		t.Errorf("书签名称 = %q, 期望 %q", bm.Name, "日志目录")
	}
}

func TestFileManagerBookmarkCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager()

	// 添加自定义书签
	bm := &Bookmark{
		ID:   "test",
		Name: "测试目录",
		Path: tmpDir,
	}
	if err := fm.AddBookmark(bm); err != nil {
		t.Fatalf("AddBookmark 失败: %v", err)
	}

	// 查询
	got, ok := fm.GetBookmark("test")
	if !ok {
		t.Error("GetBookmark 应找到 'test'")
	}
	if got.Name != "测试目录" {
		t.Errorf("Name = %q, 期望 %q", got.Name, "测试目录")
	}

	// 路径应被规范化为绝对路径
	if !filepath.IsAbs(got.Path) {
		t.Errorf("Path 应为绝对路径, 实际: %q", got.Path)
	}

	// 列表
	bookmarks := fm.ListBookmarks()
	if len(bookmarks) < 2 { // 默认 log + 自定义 test
		t.Errorf("ListBookmarks 长度 = %d, 期望 >= 2", len(bookmarks))
	}

	// 删除
	fm.RemoveBookmark("test")
	if _, ok := fm.GetBookmark("test"); ok {
		t.Error("删除后不应找到 'test'")
	}
}

func TestFileManagerGetFullPath(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager()

	bm := &Bookmark{ID: "test", Name: "测试", Path: tmpDir}
	fm.AddBookmark(bm)

	// 正常路径
	full, err := fm.GetFullPath("test", "subdir/file.txt")
	if err != nil {
		t.Fatalf("GetFullPath 失败: %v", err)
	}
	expected := filepath.Join(tmpDir, "subdir/file.txt")
	if full != expected {
		t.Errorf("GetFullPath = %q, 期望 %q", full, expected)
	}

	// 根路径
	full, err = fm.GetFullPath("test", "")
	if err != nil {
		t.Fatalf("GetFullPath 根路径失败: %v", err)
	}
	if full != tmpDir {
		t.Errorf("根路径 = %q, 期望 %q", full, tmpDir)
	}

	// 遍历攻击：filepath.Clean("/") 会消除 .. 使结果在安全路径内
	// 函数通过路径规范化实现安全，不报错但结果不会逃逸
	traversalPath, err := fm.GetFullPath("test", "../../etc/passwd")
	if err != nil {
		// 如果报错也是可接受的
		t.Logf("路径遍历返回错误（可接受）: %v", err)
	} else {
		// 不报错时，结果必须在 tmpDir 内
		if !strings.HasPrefix(traversalPath, tmpDir) {
			t.Errorf("路径遍历逃逸: %q 不在 %q 内", traversalPath, tmpDir)
		}
	}

	// 不存在的书签
	_, err = fm.GetFullPath("nonexistent", "file")
	if err == nil {
		t.Error("不存在的书签应返回错误")
	}
}

func TestFileManagerSiteBookmark(t *testing.T) {
	tmpDir := t.TempDir()
	siteDir := filepath.Join(tmpDir, "mysite")
	os.MkdirAll(siteDir, 0755)

	fm := NewFileManager()

	// 添加站点书签
	if err := fm.AddSiteBookmark("site1", siteDir); err != nil {
		t.Fatalf("AddSiteBookmark 失败: %v", err)
	}

	bm, ok := fm.GetBookmark("site1")
	if !ok {
		t.Error("应找到站点书签")
	}
	if bm.Icon != "globe" {
		t.Errorf("Icon = %q, 期望 %q", bm.Icon, "globe")
	}

	// 重复添加应跳过
	if err := fm.AddSiteBookmark("site1", "/other"); err != nil {
		t.Fatalf("重复添加应不报错: %v", err)
	}
	// 路径不应改变
	bm2, _ := fm.GetBookmark("site1")
	if bm2.Path != bm.Path {
		t.Error("重复添加后路径不应改变")
	}

	// 空路径应跳过
	if err := fm.AddSiteBookmark("empty", ""); err != nil {
		t.Errorf("空路径不应报错: %v", err)
	}
	if _, ok := fm.GetBookmark("empty"); ok {
		t.Error("空路径不应创建书签")
	}

	// 移除
	fm.RemoveSiteBookmark("site1")
	if _, ok := fm.GetBookmark("site1"); ok {
		t.Error("移除后不应找到")
	}
}

// ==================== 目录操作测试 ====================

func TestListDirEntries(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件（关闭句柄，避免 Windows 上 TempDir 清理失败）
	for _, name := range []string{"file1.txt", "file2.go"} {
		f, err := os.Create(filepath.Join(tmpDir, name))
		if err != nil {
			t.Fatalf("创建 %s 失败: %v", name, err)
		}
		f.Close()
	}
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	// 所有条目
	entries, err := ListDirEntries(tmpDir, false)
	if err != nil {
		t.Fatalf("ListDirEntries 失败: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("条目数 = %d, 期望 3", len(entries))
	}

	// 仅目录
	dirEntries, err := ListDirEntries(tmpDir, true)
	if err != nil {
		t.Fatalf("ListDirEntries (dirsOnly) 失败: %v", err)
	}
	if len(dirEntries) != 1 {
		t.Errorf("目录数 = %d, 期望 1", len(dirEntries))
	}
	if !dirEntries[0].IsDir {
		t.Error("应为目录")
	}
	if dirEntries[0].Name != "subdir" {
		t.Errorf("目录名 = %q, 期望 %q", dirEntries[0].Name, "subdir")
	}

	// 不存在的目录
	_, err = ListDirEntries("/nonexistent/path", false)
	if err == nil {
		t.Error("不存在的目录应返回错误")
	}
}

func TestCopySingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建源文件
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("Hello, World!")
	os.WriteFile(srcPath, content, 0644)

	// 复制
	dstPath := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := CopySingleFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopySingleFile 失败: %v", err)
	}

	// 验证内容
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("内容 = %q, 期望 %q", string(data), string(content))
	}
}

func TestCopyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建源目录结构
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("file2"), 0644)

	// 复制
	dstDir := filepath.Join(tmpDir, "dst")
	if err := CopyDirectory(srcDir, dstDir); err != nil {
		t.Fatalf("CopyDirectory 失败: %v", err)
	}

	// 验证
	data1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		t.Fatalf("读取 file1 失败: %v", err)
	}
	if string(data1) != "file1" {
		t.Errorf("file1 内容 = %q, 期望 %q", string(data1), "file1")
	}

	data2, err := os.ReadFile(filepath.Join(dstDir, "sub", "file2.txt"))
	if err != nil {
		t.Fatalf("读取 file2 失败: %v", err)
	}
	if string(data2) != "file2" {
		t.Errorf("file2 内容 = %q, 期望 %q", string(data2), "file2")
	}
}
