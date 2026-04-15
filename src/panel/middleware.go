package panel

import (
	"net/http"
	"strings"
	"time"

	"pixelbeast/src/logger"
)

// Middleware HTTP 中间件类型
type Middleware func(http.Handler) http.Handler

// Chain 将多个中间件串联（执行顺序：从左到右包裹）
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// RecoveryMiddleware panic 恢复中间件
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.LogPanelRuntime(logger.LogLevelError, "[Panic] %s %s: %v", r.Method, r.URL.Path, err)
				InternalServerError(w, "内部错误")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware 请求日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		logger.LogPanelRuntime(logger.LogLevelDebug, "[HTTP] %s %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, duration)
	})
}

// SecurityHeadersMiddleware 安全响应头中间件
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// RequireAuth 认证中间件
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := h.getSession(r)
		if session == nil {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				Unauthorized(w, "未登录")
			} else {
				http.Redirect(w, r, h.adminPath+"/login", http.StatusFound)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CSRPMiddleware CSRF 防护中间件（对状态修改请求检查 CSRF Token）
// 支持同一会话持有多个有效 CSRF Token（解决多标签页/刷新页面导致 token 失效的问题）
func (h *Handler) CSRPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 只检查状态修改方法
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// 登录接口跳过 CSRF（使用登录限速保护）
		if r.URL.Path == "/api/login" || r.URL.Path == "/api/logout" {
			next.ServeHTTP(w, r)
			return
		}

		// 检查 CSRF Token
		session := h.getSession(r)
		if session == nil {
			Unauthorized(w, "未登录")
			return
		}

		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			Forbidden(w, "缺少 CSRF Token")
			return
		}

		// 按 token 值查找（支持同一会话多个 token）
		h.mu.RLock()
		csrfToken, exists := h.csrfTokens[token]
		h.mu.RUnlock()

		if !exists {
			Forbidden(w, "请刷新页面获取 CSRF Token")
			return
		}
		if time.Now().After(csrfToken.ExpiresAt) {
			// 删除过期 token
			h.mu.Lock()
			delete(h.csrfTokens, token)
			h.mu.Unlock()
			Forbidden(w, "CSRF Token 已过期，请刷新页面")
			return
		}
		if csrfToken.SessionID != getSessionID(r) {
			Forbidden(w, "CSRF 验证失败")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getSessionID 从 cookie 获取 session ID（内部使用）
func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}
