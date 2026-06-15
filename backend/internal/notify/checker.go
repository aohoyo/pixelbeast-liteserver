package notify

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// HealthResult 健康检查结果
type HealthResult struct {
	URL        string    `json:"url"`
	Status     string    `json:"status"`     // "ok" / "down"
	StatusCode int       `json:"status_code"`
	Latency    int64     `json:"latency"`    // 毫秒
	CheckedAt  time.Time `json:"checked_at"`
	Error      string    `json:"error,omitempty"`
}

// CheckHTTP 检测单个 URL 的健康状态
func CheckHTTP(url string, timeout int, expectedStatus int) HealthResult {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives:   true,
			MaxIdleConns:        1,
			DisableCompression:  true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 跟随重定向
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start).Milliseconds()

	result := HealthResult{
		URL:        url,
		CheckedAt:  time.Now(),
		Latency:    latency,
	}

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if expectedStatus > 0 {
		if resp.StatusCode == expectedStatus {
			result.Status = "ok"
		} else {
			result.Status = "down"
			result.Error = fmt.Sprintf("预期状态码 %d，实际 %d", expectedStatus, resp.StatusCode)
		}
	} else {
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			result.Status = "ok"
		} else {
			result.Status = "down"
			result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	return result
}

// CheckCertExpiry 检查证书文件到期时间
func CheckCertExpiry(certFile string) (*CertInfo, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("读取证书文件失败: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("无法解析 PEM 格式")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)

	return &CertInfo{
		Domain:    cert.Subject.CommonName,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		DaysLeft:  daysLeft,
		IsExpired: now.After(cert.NotAfter),
	}, nil
}

// CertInfo 证书信息
type CertInfo struct {
	Domain    string    `json:"domain"`
	Issuer    string    `json:"issuer"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	DaysLeft  int       `json:"days_left"`
	IsExpired bool      `json:"is_expired"`
}
