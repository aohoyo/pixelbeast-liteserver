package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ==================== 成功响应测试 ====================

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	Success(w, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, 期望 %q", contentType, "application/json")
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != CodeSuccess {
		t.Errorf("Code = %d, 期望 %d", resp.Code, CodeSuccess)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %q, 期望 %q", resp.Message, "success")
	}
	if resp.Data == nil {
		t.Error("Data 不应为 nil")
	}
}

func TestSuccessMessage(t *testing.T) {
	w := httptest.NewRecorder()
	SuccessMessage(w, "操作成功")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != CodeSuccess {
		t.Errorf("Code = %d, 期望 %d", resp.Code, CodeSuccess)
	}
	if resp.Message != "操作成功" {
		t.Errorf("Message = %q, 期望 %q", resp.Message, "操作成功")
	}
	if resp.Data != nil {
		t.Error("Data 应为 nil")
	}
}

func TestSuccessWithData(t *testing.T) {
	w := httptest.NewRecorder()
	SuccessWithData(w, map[string]int{"count": 42}, "查询成功")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Message != "查询成功" {
		t.Errorf("Message = %q, 期望 %q", resp.Message, "查询成功")
	}
	if resp.Data == nil {
		t.Error("Data 不应为 nil")
	}
}

// ==================== 错误响应测试 ====================

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, 期望 %d", resp.Code, http.StatusBadRequest)
	}
	if resp.Message != "参数错误" {
		t.Errorf("Message = %q, 期望 %q", resp.Message, "参数错误")
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "请求格式错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusBadRequest)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "未登录")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusUnauthorized)
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "禁止访问")

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusForbidden)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "资源不存在")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusNotFound)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	MethodNotAllowed(w, "方法不允许")

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestTooManyRequests(t *testing.T) {
	w := httptest.NewRecorder()
	TooManyRequests(w, "请求过多")

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalServerError(w, "内部错误")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== 辅助函数测试 ====================

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, map[string]string{"test": "data"})

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, 期望 %d", w.Code, http.StatusOK)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}
	if result["test"] != "data" {
		t.Errorf("test = %q, 期望 %q", result["test"], "data")
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"42", 42},
		{"0", 0},
		{"", 99},
		{"abc", 99},
		{"3.14", 99},
	}
	for _, tt := range tests {
		result := parseIntParam(tt.input, 99)
		if result != tt.expected {
			t.Errorf("parseIntParam(%q, 99) = %d, 期望 %d", tt.input, result, tt.expected)
		}
	}
}

// ==================== 响应码常量测试 ====================

func TestResponseCodes(t *testing.T) {
	codes := map[int]int{
		CodeSuccess:          200,
		CodeBadRequest:       400,
		CodeUnauthorized:     401,
		CodeForbidden:        403,
		CodeNotFound:         404,
		CodeMethodNotAllowed: 405,
		CodeTooManyRequests:  429,
		CodeInternalError:    500,
	}
	for constant, expected := range codes {
		if constant != expected {
			t.Errorf("响应码 = %d, 期望 %d", constant, expected)
		}
	}
}
