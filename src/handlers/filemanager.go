package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"pixelbeast/src/config"
)

// Bookmark 文件管理书签
type Bookmark struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
	Icon     string `json:"icon"`
}

// FileManager 独立文件管理器
type FileManager struct {
	mu        sync.RWMutex
	bookmarks map[string]*Bookmark
}

// NewFileManager 创建文件管理器
func NewFileManager() *FileManager {
	fm := &FileManager{
		bookmarks: make(map[string]*Bookmark),
	}

	// 添加默认书签
	defaultBookmarks := []*Bookmark{
		{
			ID:       "data",
			Name:     "数据目录",
			Path:     "./data",
			Readonly: false,
			Icon:     "folder",
		},
		{
			ID:       "log",
			Name:     "日志目录",
			Path:     "./log",
			Readonly: true,
			Icon:     "file-alt",
		},
	}

	for _, bm := range defaultBookmarks {
		fm.AddBookmark(bm)
	}

	return fm
}

// AddBookmark 添加书签
func (fm *FileManager) AddBookmark(bm *Bookmark) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// 验证并规范化路径
	absPath, err := filepath.Abs(bm.Path)
	if err != nil {
		return err
	}

	// 检查路径是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// 不存在则创建
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return err
		}
	}

	bm.Path = absPath
	fm.bookmarks[bm.ID] = bm
	return nil
}

// RemoveBookmark 移除书签
func (fm *FileManager) RemoveBookmark(id string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	delete(fm.bookmarks, id)
}

// GetBookmark 获取书签
func (fm *FileManager) GetBookmark(id string) (*Bookmark, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	bm, ok := fm.bookmarks[id]
	return bm, ok
}

// ListBookmarks 列出所有书签
func (fm *FileManager) ListBookmarks() []*Bookmark {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make([]*Bookmark, 0, len(fm.bookmarks))
	for _, bm := range fm.bookmarks {
		result = append(result, bm)
	}
	return result
}

// GetFullPath 获取书签的完整路径
func (fm *FileManager) GetFullPath(bookmarkID, relativePath string) (string, error) {
	fm.mu.RLock()
	bm, ok := fm.bookmarks[bookmarkID]
	fm.mu.RUnlock()

	if !ok {
		return "", os.ErrNotExist
	}

	// 规范化相对路径
	relativePath = filepath.Clean("/" + relativePath)
	if relativePath == "/" {
		relativePath = ""
	}

	fullPath := filepath.Join(bm.Path, relativePath)

	// 安全检查：确保结果路径在书签路径内
	if !isPathWithin(fullPath, bm.Path) {
		return "", os.ErrPermission
	}

	return fullPath, nil
}

// isPathWithin 检查路径是否在基础路径内
func isPathWithin(path, basePath string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..")
}

// AddSiteBookmark 添加站点根目录作为书签
func (fm *FileManager) AddSiteBookmark(siteID, sitePath string) error {
	// 如果路径为空，跳过
	if sitePath == "" {
		return nil
	}

	// 检查书签是否已存在
	if _, exists := fm.GetBookmark(siteID); exists {
		return nil
	}

	return fm.AddBookmark(&Bookmark{
		ID:       siteID,
		Name:     "站点: " + siteID,
		Path:     sitePath,
		Readonly: false,
		Icon:     "globe",
	})
}

// RemoveSiteBookmark 移除站点书签
func (fm *FileManager) RemoveSiteBookmark(siteID string) {
	fm.RemoveBookmark(siteID)
}

// UpdateBookmarksFromConfig 根据站点配置更新书签
func (fm *FileManager) UpdateBookmarksFromConfig(sites []config.SiteConfig) {
	for _, site := range sites {
		if site.Type == "static" && site.Root != "" {
			fm.AddSiteBookmark(site.ID, site.Root)
		}
	}
}
