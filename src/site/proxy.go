package site

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/logger"
	
)

// ProxyHandler 反向代理处理器
type ProxyHandler struct {
	proxy       *httputil.ReverseProxy
	target      *url.URL
	stripPrefix string
	wsEnabled   bool
	timeout     time.Duration
	SiteID      string
}

// NewProxyHandler 创建反向代理处理器
func NewProxyHandler(cfg *config.ProxyConfig, siteID string) (*ProxyHandler, error) {
	// 解析目标 URL
	target, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("parse target URL: %w", err)
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 设置错误处理器
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.LogPanelRuntime(logger.LogLevelError, "[Proxy] 代理错误: %v, 目标: %s, 请求: %s", err, cfg.Target, r.URL.Path)
		http.Error(w, "代理请求失败", http.StatusBadGateway)
	}

	// 自定义 Director 修改请求
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		// 保存原始 Host
		originalHost := req.Host

		director(req)

		// 设置 Host 为目标主机（解决 403 问题）
		req.Host = target.Host
		req.Header.Set("Host", target.Host)

		// 移除前缀
		if cfg.StripPrefix != "" && strings.HasPrefix(req.URL.Path, cfg.StripPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, cfg.StripPrefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			// 更新 RawPath 以支持编码的路径
			req.URL.RawPath = req.URL.EscapedPath()
		}

		// 设置代理请求头
		req.Header.Set("X-Forwarded-Host", originalHost)
		req.Header.Set("X-Forwarded-Proto", scheme(req))
		req.Header.Set("X-Real-IP", getRemoteIP(req))

		// 关闭 WebSocket 连接升级时的请求体
		if isWebSocketUpgrade(req) {
			req.Header.Set("Connection", "Upgrade")
		}
	}

	// 修改响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 设置 CORS 头（如果需要）
		// resp.Header.Set("X-Proxy", "LiteFeather")
		return nil
	}

	timeout := 30 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	return &ProxyHandler{
		proxy:       proxy,
		target:      target,
		stripPrefix: cfg.StripPrefix,
		wsEnabled:   cfg.Websocket,
		timeout:     timeout,
		SiteID:      siteID,
	}, nil
}

// ServeHTTP 实现 http.Handler
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// 使用包装 ResponseWriter 捕获状态码
	rw := &proxyResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// WebSocket 升级检查
	if h.wsEnabled && isWebSocketUpgrade(r) {
		h.proxy.ServeHTTP(rw, r)
		h.logRequest(r, start, rw.statusCode)
		return
	}

	// 普通 HTTP 代理
	h.proxy.ServeHTTP(rw, r)
	h.logRequest(r, start, rw.statusCode)
}

// logRequest 记录代理请求日志
func (h *ProxyHandler) logRequest(r *http.Request, start time.Time, status int) {
	duration := time.Since(start)
	logger.LogHTTPAccess(r.Method, r.URL.Path, r.RemoteAddr, status, duration)
	if h.SiteID != "" {
		logger.LogHTTPAccessBySite(h.SiteID, r.Method, r.URL.Path, r.RemoteAddr, status, duration)
	}
}

// proxyResponseWriter 包装 ResponseWriter 捕获状态码
type proxyResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *proxyResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// isWebSocketUpgrade 检查是否为 WebSocket 升级请求
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// scheme 返回请求的协议（http 或 https）
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// getRemoteIP 获取客户端真实 IP
func getRemoteIP(r *http.Request) string {
	// 检查 X-Forwarded-For 头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个 IP
		if i := strings.Index(xff, ","); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return xff
	}

	// 检查 X-Real-IP 头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	if ip := r.RemoteAddr; ip != "" {
		if i := strings.LastIndex(ip, ":"); i != -1 {
			return ip[:i]
		}
		return ip
	}

	return "unknown"
}
