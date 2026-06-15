package panel

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// cachedFile 预计算的缓存文件
type cachedFile struct {
	data        []byte
	gzipData    []byte // 预压缩 gzip 数据
	etag        string // ETag 值（含引号）
	lastMod     time.Time
	contentType string
	size        int
}

// FileCache 静态文件缓存
type FileCache struct {
	mu    sync.RWMutex
	files map[string]*cachedFile // key: "subdir/filename"
	fsys  fs.FS
}

var fileCache = &FileCache{}

// initFileCache 初始化文件缓存（遍历文件系统并预计算 ETag/gzip）
func initFileCache(fsys fs.FS) {
	fc := &FileCache{
		fsys:  fsys,
		files: make(map[string]*cachedFile),
	}

	dirs := []string{"css", "js", "components", "images", "icons"}
	for _, dir := range dirs {
		entries, err := fs.ReadDir(fsys, dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := dir + "/" + entry.Name()
			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				continue
			}
			cf := buildCachedFile(data, path, entry)
			fc.files[path] = cf
		}
	}

	// favicon.ico
	if data, err := fs.ReadFile(fsys, "favicon.ico"); err == nil {
		fc.files["favicon.ico"] = buildCachedFile(data, "favicon.ico", nil)
	}

	fileCache = fc
}

// buildCachedFile 构建缓存条目
func buildCachedFile(data []byte, path string, entry fs.DirEntry) *cachedFile {
	// ETag: SHA-256 前 16 位 hex + 文件大小
	hash := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(hash[:8]) + "-" + strconv.Itoa(len(data)) + `"`

	ct := contentTypeForPath(path)

	cf := &cachedFile{
		data:        data,
		etag:        etag,
		contentType: ct,
		size:        len(data),
	}

	// 获取修改时间
	if entry != nil {
		if info, err := entry.Info(); err == nil {
			cf.lastMod = info.ModTime()
		}
	}
	if cf.lastMod.IsZero() {
		cf.lastMod = time.Now()
	}

	// 预压缩文本文件（仅当压缩后更小时保留）
	if isTextContent(ct) && len(data) > 150 {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		gw.Write(data)
		gw.Close()
		if buf.Len() < len(data) {
			cf.gzipData = buf.Bytes()
		}
	}

	return cf
}

// HandleCached 处理带缓存的静态文件请求
func (fc *FileCache) HandleCached(w http.ResponseWriter, r *http.Request, path, fallbackContentType string) {
	cf := fc.get(path)

	// 缓存未命中：动态读取（开发模式文件可能变化）
	if cf == nil {
		data, err := fs.ReadFile(fc.fsys, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ct := fallbackContentType
		if ct == "" {
			ct = contentTypeForPath(path)
		}
		hash := sha256.Sum256(data)
		etag := `"` + hex.EncodeToString(hash[:8]) + "-" + strconv.Itoa(len(data)) + `"`

		w.Header().Set("Content-Type", ct)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		if match := r.Header.Get("If-None-Match"); match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Write(data)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", cf.contentType)
	w.Header().Set("ETag", cf.etag)
	w.Header().Set("Last-Modified", cf.lastMod.UTC().Format(http.TimeFormat))
	w.Header().Set("Cache-Control", cacheControlForType(cf.contentType))

	// 条件请求：If-None-Match
	if match := r.Header.Get("If-None-Match"); match == cf.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// 条件请求：If-Modified-Since
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil && !cf.lastMod.After(t) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// gzip 协商
	acceptGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
	if acceptGzip && cf.gzipData != nil {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", strconv.Itoa(len(cf.gzipData)))
		w.Write(cf.gzipData)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(cf.size))
	w.Write(cf.data)
}

func (fc *FileCache) get(path string) *cachedFile {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.files[path]
}

// cacheControlForType 根据内容类型返回 Cache-Control
func cacheControlForType(contentType string) string {
	switch {
	case strings.Contains(contentType, "javascript") || strings.Contains(contentType, "css"):
		return "public, max-age=31536000, immutable"
	case strings.HasPrefix(contentType, "image/"):
		return "public, max-age=604800"
	case strings.Contains(contentType, "html"):
		return "public, max-age=300"
	default:
		return "public, max-age=3600"
	}
}

// contentTypeForPath 根据文件扩展名返回 Content-Type
func contentTypeForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".webp":
		return "image/webp"
	case ".json":
		return "application/json"
	case ".woff", ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}

// isTextContent 判断是否为可压缩的文本内容
func isTextContent(ct string) bool {
	return strings.Contains(ct, "text/") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "svg") ||
		strings.Contains(ct, "xml")
}
