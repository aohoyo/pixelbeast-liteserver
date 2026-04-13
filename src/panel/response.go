package panel

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ==================== 统一响应结构 ====================

// Response 统一 API 响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 常用响应码
const (
	CodeSuccess          = 200 // 成功
	CodeBadRequest       = 400 // 请求参数错误
	CodeUnauthorized     = 401 // 未认证
	CodeForbidden        = 403 // 禁止访问
	CodeNotFound         = 404 // 资源不存在
	CodeMethodNotAllowed = 405 // 方法不允许
	CodeTooManyRequests  = 429 // 请求过多
	CodeInternalError    = 500 // 内部错误
)

// ==================== 响应函数 ====================

// respondJSON 发送 JSON 响应（底层函数）
// 先序列化再写入，确保编码失败时能正确返回 500
func respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	body, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":500,"message":"JSON 编码失败"}`))
		return
	}
	w.WriteHeader(code)
	w.Write(body)
}

// Success 成功响应（带数据）
func Success(w http.ResponseWriter, data interface{}) {
	respondJSON(w, http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessMessage 成功响应（仅消息）
func SuccessMessage(w http.ResponseWriter, message string) {
	respondJSON(w, http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
	})
}

// SuccessWithData 成功响应（数据+消息）
func SuccessWithData(w http.ResponseWriter, data interface{}, message string) {
	respondJSON(w, http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(w http.ResponseWriter, httpCode int, message string) {
	respondJSON(w, httpCode, Response{
		Code:    httpCode,
		Message: message,
	})
}

// BadRequest 400 错误请求
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Unauthorized 401 未授权
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden 403 禁止访问
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

// NotFound 404 未找到
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// MethodNotAllowed 405 方法不允许
func MethodNotAllowed(w http.ResponseWriter, message string) {
	Error(w, http.StatusMethodNotAllowed, message)
}

// TooManyRequests 429 请求过多
func TooManyRequests(w http.ResponseWriter, message string) {
	Error(w, http.StatusTooManyRequests, message)
}

// InternalServerError 500 内部错误
func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}

// ==================== 辅助函数 ====================

// parseJSONBody 解析 JSON 请求体
func parseJSONBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// parseIntParam 解析整数参数
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := json.Number(s).Int64(); err == nil {
		fmt.Sscanf(s, "%d", &val)
		return val
	}
	return defaultVal
}
