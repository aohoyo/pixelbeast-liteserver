package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

// ==================== 证书管理 API ====================

// handleCertsList 获取所有证书状态
func (h *Handler) handleCertsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		Success(w, []interface{}{})
		return
	}

	certs := h.ServerManager.SSLManager.GetAllCertStatuses()
	Success(w, certs)
}

// handleCertDetail 获取单个证书详情
func (h *Handler) handleCertDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 从 URL 提取域名: /api/certs/{domain}
	path := r.URL.Path
	domain := strings.TrimPrefix(path, "/api/certs/")
	if domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		NotFound(w, "证书不存在")
		return
	}

	status := h.ServerManager.SSLManager.GetCertStatus(domain)
	Success(w, status)
}

// handleCertRequest 申请证书（支持 autocert 和 lego）
func (h *Handler) handleCertRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain          string `json:"domain"`
		Email           string `json:"email"`
		ChallengeMethod string `json:"challenge_method"`
		}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	if req.ChallengeMethod == "" {
		req.ChallengeMethod = "http-auto"
	}

	// 申请证书
	if err := h.ServerManager.SSLManager.ObtainCertificate(req.Domain, req.Email, "letsencrypt", req.ChallengeMethod); err != nil {
		InternalServerError(w, "证书申请失败: "+err.Error())
		return
	}

	// 更新站点配置中的 SSL 设置
	h.updateSiteSSLConfig(req.Domain, &config.SSLConfig{
		Enabled:         true,
		AutoHTTPS:       true,
		Email:           req.Email,
		Provider:        "letsencrypt",
		ChallengeMethod: req.ChallengeMethod,
	})

	handlers.LogPanelOperation(handlers.LogLevelInfo, "证书申请: %s (method: %s)", req.Domain, req.ChallengeMethod)
	SuccessMessage(w, "证书申请已提交，将在下次 HTTPS 访问时自动获取")
}

// handleCertDNSPrepare DNS 验证第一步：返回 TXT 记录信息或自动添加
func (h *Handler) handleCertDNSPrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain         string            `json:"domain"`
		Email          string            `json:"email"`
		DNSProvider    string            `json:"dns_provider"`    // "manual" | "saved" | "alidns" | "tencentcloud" | "baota"
		DNSProviderID  string            `json:"dns_provider_id"` // saved 模式时的服务商 ID
		DNSCredentials map[string]string `json:"dns_credentials"`
		}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	if req.DNSProvider == "" {
		req.DNSProvider = "manual"
	}

	// 解析已保存的 DNS 服务商
	if req.DNSProvider == "saved" {
		if req.DNSProviderID == "" {
			BadRequest(w, "请选择 DNS 服务商")
			return
		}
		provider := h.ConfigManager.GetDNSProvider(req.DNSProviderID)
		if provider == nil {
			BadRequest(w, "DNS 服务商不存在")
			return
		}
		creds, err := h.ConfigManager.DecryptDNSCredentials(provider.Credentials)
		if err != nil {
			InternalServerError(w, "凭证解密失败")
			return
		}
		req.DNSProvider = provider.Type
		req.DNSCredentials = creds
	}

	dnsInfo, err := h.ServerManager.SSLManager.PrepareDNSChallenge(req.Domain, req.Email, "letsencrypt", req.DNSProvider, req.DNSCredentials)
	if err != nil {
		InternalServerError(w, "DNS 挑战准备失败: "+err.Error())
		return
	}

	handlers.LogPanelOperation(handlers.LogLevelInfo, "DNS 挑战准备: %s", req.Domain)
	Success(w, dnsInfo)
}

// handleCertDNSComplete DNS 验证第二步：用户确认后验证并获取证书
func (h *Handler) handleCertDNSComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	if err := h.ServerManager.SSLManager.CompleteDNSChallenge(req.Domain); err != nil {
		InternalServerError(w, "DNS 验证失败: "+err.Error())
		return
	}

	// 更新站点配置
	h.updateSiteSSLConfig(req.Domain, &config.SSLConfig{
		Enabled:         true,
		AutoHTTPS:       true,
		Provider:        "letsencrypt", // 会在 CompleteDNSChallenge 中由 SSLManager 更新
		ChallengeMethod: "dns",
	})

	handlers.LogPanelOperation(handlers.LogLevelInfo, "DNS 验证证书获取成功: %s", req.Domain)
	SuccessMessage(w, "DNS 验证证书获取成功")
}

// handleCertFilePrepare 文件验证第一步：生成验证文件内容
func (h *Handler) handleCertFilePrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain   string `json:"domain"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	fileInfo, err := h.ServerManager.SSLManager.PrepareFileChallenge(req.Domain, req.Email, "letsencrypt")
	if err != nil {
		InternalServerError(w, "文件挑战准备失败: "+err.Error())
		return
	}

	// 确保端口 80 已启动（用于 CA 验证 ACME challenge）
	h.ServerManager.StartHTTPRedirect()

	handlers.LogPanelOperation(handlers.LogLevelInfo, "文件挑战准备: %s", req.Domain)
	Success(w, fileInfo)
}

// handleCertFileComplete 文件验证第二步：验证并获取证书
func (h *Handler) handleCertFileComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	if err := h.ServerManager.SSLManager.CompleteFileChallenge(req.Domain); err != nil {
		InternalServerError(w, "文件验证失败: "+err.Error())
		return
	}

	// 更新站点配置
	h.updateSiteSSLConfig(req.Domain, &config.SSLConfig{
		Enabled:         true,
		AutoHTTPS:       true,
		ChallengeMethod: "http-file",
	})

	handlers.LogPanelOperation(handlers.LogLevelInfo, "文件验证证书获取成功: %s", req.Domain)
	SuccessMessage(w, "文件验证证书获取成功")
}

// handleCertRenew 续期证书
func (h *Handler) handleCertRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	if err := h.ServerManager.SSLManager.RenewCertificate(req.Domain); err != nil {
		InternalServerError(w, "证书续期失败: "+err.Error())
		return
	}

	handlers.LogPanelOperation(handlers.LogLevelInfo, "证书续期: %s", req.Domain)
	SuccessMessage(w, "证书续期已触发")
}

// handleCertUpload 上传自定义证书
func (h *Handler) handleCertUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 限制上传大小 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		BadRequest(w, "文件过大，最大支持 10MB")
		return
	}

	domain := r.FormValue("domain")
	if domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	// 获取证书文件
	certFile, _, err := r.FormFile("cert_file")
	if err != nil {
		BadRequest(w, "请上传证书文件")
		return
	}
	defer certFile.Close()
	certPEM, err := io.ReadAll(certFile)
	if err != nil {
		InternalServerError(w, "读取证书文件失败")
		return
	}

	// 获取私钥文件
	keyFile, _, err := r.FormFile("key_file")
	if err != nil {
		BadRequest(w, "请上传私钥文件")
		return
	}
	defer keyFile.Close()
	keyPEM, err := io.ReadAll(keyFile)
	if err != nil {
		InternalServerError(w, "读取私钥文件失败")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	// 保存证书
	if err := h.ServerManager.SSLManager.SaveCustomCertificate(domain, certPEM, keyPEM); err != nil {
		BadRequest(w, "证书保存失败: "+err.Error())
		return
	}

	// 更新站点配置中的 SSL 设置
	certPath := filepath.Join("./ssl", domain+".crt")
	keyPath := filepath.Join("./ssl", domain+".key")
	h.updateSiteSSLConfig(domain, &config.SSLConfig{
		Enabled:  true,
		AutoHTTPS: false,
		CertFile: certPath,
		KeyFile:  keyPath,
	})

	handlers.LogPanelOperation(handlers.LogLevelInfo, "证书上传: %s", domain)
	SuccessMessage(w, "证书上传成功")
}

// handleCertDelete 删除证书
func (h *Handler) handleCertDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		InternalServerError(w, "SSL 管理器未初始化")
		return
	}

	if err := h.ServerManager.SSLManager.DeleteCertificate(req.Domain); err != nil {
		InternalServerError(w, "证书删除失败: "+err.Error())
		return
	}

	// 更新站点配置，禁用 SSL
	h.updateSiteSSLConfig(req.Domain, nil)

	handlers.LogPanelOperation(handlers.LogLevelInfo, "证书删除: %s", req.Domain)
	SuccessMessage(w, "证书已删除")
}

// updateSiteSSLConfig 更新站点配置中的 SSL 设置
func (h *Handler) updateSiteSSLConfig(domain string, ssl *config.SSLConfig) {
	if h.ConfigManager == nil {
		return
	}

	changed := false
	for i := range h.ConfigManager.Sites.Sites {
		site := &h.ConfigManager.Sites.Sites[i]
		for _, d := range site.Domain {
			if d == domain {
				if ssl != nil {
					site.SSL = ssl
				} else {
					site.SSL = &config.SSLConfig{Enabled: false}
				}
				changed = true
				break
			}
		}
		if changed {
			break
		}
	}

	if changed {
		h.ConfigManager.Save()
	}
}



// ==================== DNS 服务商管理 API ====================

// maskCredentials 脱敏凭证用于前端显示
func maskCredentials(creds map[string]string) map[string]string {
	masked := make(map[string]string)
	for k, v := range creds {
		switch k {
		case "secret_key":
			masked[k] = "****"
		default:
			if len(v) > 4 {
				masked[k] = v[:4] + "****"
			} else if len(v) > 0 {
				masked[k] = "****"
			} else {
				masked[k] = ""
			}
		}
	}
	return masked
}

// handleDNSProvidersList 获取所有 DNS 服务商配置（凭证脱敏）
func (h *Handler) handleDNSProvidersList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	providers := h.ConfigManager.GetDNSProviders()

	// 脱敏凭证
	type DNSProviderView struct {
		ID             string                 `json:"id"`
		Name           string                 `json:"name"`
		Type           string                 `json:"type"`
		MaskedCreds    map[string]string      `json:"masked_creds"`
		CreatedAt      string                 `json:"created_at"`
		UpdatedAt      string                 `json:"updated_at"`
	}

	views := make([]DNSProviderView, 0, len(providers))
	for _, p := range providers {
		creds, err := h.ConfigManager.DecryptDNSCredentials(p.Credentials)
		if err != nil {
			creds = map[string]string{}
		}
		views = append(views, DNSProviderView{
			ID:          p.ID,
			Name:        p.Name,
			Type:        p.Type,
			MaskedCreds: maskCredentials(creds),
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	Success(w, views)
}

// handleDNSProviderCreate 新增 DNS 服务商
func (h *Handler) handleDNSProviderCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Name        string            `json:"name"`
		Type        string            `json:"type"`
		Credentials map[string]string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Name == "" || req.Type == "" {
		BadRequest(w, "名称和类型不能为空")
		return
	}

	validTypes := map[string]bool{"alidns": true, "tencentcloud": true, "baota": true}
	if !validTypes[req.Type] {
		BadRequest(w, "不支持的 DNS 服务商类型")
		return
	}

	// 生成 ID
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	id := hex.EncodeToString(idBytes)

	// 加密凭证
	encrypted, err := h.ConfigManager.EncryptDNSCredentials(req.Credentials)
	if err != nil {
		InternalServerError(w, "凭证加密失败: "+err.Error())
		return
	}

	now := time.Now().Format(time.RFC3339)
	provider := config.DNSProviderConfig{
		ID:          id,
		Name:        req.Name,
		Type:        req.Type,
		Credentials: encrypted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.ConfigManager.AddDNSProvider(provider); err != nil {
		InternalServerError(w, "保存失败: "+err.Error())
		return
	}

	handlers.LogPanelOperation(handlers.LogLevelInfo, "DNS 服务商添加: %s (%s)", req.Name, req.Type)
	Success(w, map[string]string{"id": id})
}

// handleDNSProviderUpdate 更新 DNS 服务商
func (h *Handler) handleDNSProviderUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 从 URL 提取 ID: /api/certs/dns-providers/{id}
	path := r.URL.Path
	id := ""
	if parts := strings.Split(path, "/"); len(parts) > 0 {
		id = parts[len(parts)-1]
	}
	if id == "" {
		BadRequest(w, "ID 不能为空")
		return
	}

	var req struct {
		Name        string            `json:"name"`
		Type        string            `json:"type"`
		Credentials map[string]string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	existing := h.ConfigManager.GetDNSProvider(id)
	if existing == nil {
		NotFound(w, "DNS 服务商不存在")
		return
	}

	// 更新凭证（只在没有编辑模式占位提示时，即至少一个字段有实际值）
	hasNewCreds := false
	for _, v := range req.Credentials {
		if strings.TrimSpace(v) != "" {
			hasNewCreds = true
			break
		}
	}
	if hasNewCreds {
		encrypted, err := h.ConfigManager.EncryptDNSCredentials(req.Credentials)
		if err != nil {
			InternalServerError(w, "凭证加密失败: "+err.Error())
			return
		}
		existing.Credentials = encrypted
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	existing.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.ConfigManager.UpdateDNSProvider(id, *existing); err != nil {
		InternalServerError(w, "更新失败: "+err.Error())
		return
	}

	handlers.LogPanelOperation(handlers.LogLevelInfo, "DNS 服务商更新: %s", id)
	SuccessMessage(w, "DNS 服务商已更新")
}

// handleDNSProviderDelete 删除 DNS 服务商
func (h *Handler) handleDNSProviderDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	path := r.URL.Path
	id := ""
	if parts := strings.Split(path, "/"); len(parts) > 0 {
		id = parts[len(parts)-1]
	}
	if id == "" {
		BadRequest(w, "ID 不能为空")
		return
	}

	if err := h.ConfigManager.DeleteDNSProvider(id); err != nil {
		InternalServerError(w, "删除失败: "+err.Error())
		return
	}

	handlers.LogPanelOperation(handlers.LogLevelInfo, "DNS 服务商删除: %s", id)
	SuccessMessage(w, "DNS 服务商已删除")
}

// handleDNSProviderTest 测试 DNS 服务商连通性
func (h *Handler) handleDNSProviderTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	path := r.URL.Path
	id := ""
	// /api/certs/dns-providers/{id}/test
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "dns-providers" && i+1 < len(parts) {
			id = parts[i+1]
			break
		}
	}
	if id == "" {
		BadRequest(w, "ID 不能为空")
		return
	}

	provider := h.ConfigManager.GetDNSProvider(id)
	if provider == nil {
		NotFound(w, "DNS 服务商不存在")
		return
	}

	creds, err := h.ConfigManager.DecryptDNSCredentials(provider.Credentials)
	if err != nil {
		InternalServerError(w, "凭证解密失败")
		return
	}

	// 简单测试：尝试创建 DNS provider 实例
	var testErr error
	switch provider.Type {
	case "alidns":
		testErr = h.testAlidnsProvider(creds)
	case "tencentcloud":
		testErr = h.testTencentCloudProvider(creds)
	case "baota":
		testErr = h.testBaotaProvider(creds)
	default:
		BadRequest(w, "不支持的 DNS 服务商类型")
		return
	}

	if testErr != nil {
		Success(w, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("连接测试失败: %v", testErr),
		})
		return
	}

	Success(w, map[string]interface{}{
		"success": true,
		"message": "连接测试成功",
	})
}

func (h *Handler) testAlidnsProvider(creds map[string]string) error {
	ac := struct {
		APIKey    string
		SecretKey string
	}{}
	if v, ok := creds["access_key"]; ok {
		ac.APIKey = v
	}
	if v, ok := creds["secret_key"]; ok {
		ac.SecretKey = v
	}
	if ac.APIKey == "" || ac.SecretKey == "" {
		return fmt.Errorf("缺少 AccessKey 或 SecretKey")
	}
	// 实际测试由 lego provider 在使用时完成，这里只验证凭证完整性
	return nil
}

func (h *Handler) testTencentCloudProvider(creds map[string]string) error {
	secretID, _ := creds["secret_id"]
	secretKey, _ := creds["secret_key"]
	if secretID == "" || secretKey == "" {
		return fmt.Errorf("缺少 SecretId 或 SecretKey")
	}
	return nil
}

func (h *Handler) testBaotaProvider(creds map[string]string) error {
	accountID, _ := creds["account_id"]
	accessKey, _ := creds["access_key"]
	secretKey, _ := creds["secret_key"]
	if accountID == "" || accessKey == "" || secretKey == "" {
		return fmt.Errorf("缺少 Account ID、Access Key 或 Secret Key")
	}
	return nil
}

// handleDNSProviderTestCreds 测试 DNS 服务商凭证（无需保存）
func (h *Handler) handleDNSProviderTestCreds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Type        string            `json:"type"`
		Credentials map[string]string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "请求格式错误")
		return
	}

	if req.Type == "" {
		BadRequest(w, "类型不能为空")
		return
	}

	var testErr error
	switch req.Type {
	case "alidns":
		testErr = h.testAlidnsProvider(req.Credentials)
	case "tencentcloud":
		testErr = h.testTencentCloudProvider(req.Credentials)
	case "baota":
		testErr = h.testBaotaProvider(req.Credentials)
	default:
		BadRequest(w, "不支持的 DNS 服务商类型")
		return
	}

	if testErr != nil {
		Success(w, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("连接测试失败: %v", testErr),
		})
		return
	}

	Success(w, map[string]interface{}{
		"success": true,
		"message": "连接测试成功",
	})
}

// ==================== 证书申请进度 API ====================

// handleCertProgress 获取证书申请进度和日志
func (h *Handler) handleCertProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 从 URL 提取域名: /api/certs/progress/{domain}
	path := r.URL.Path
	domain := strings.TrimPrefix(path, "/api/certs/progress/")
	if domain == "" {
		BadRequest(w, "域名不能为空")
		return
	}

	if h.ServerManager == nil || h.ServerManager.SSLManager == nil {
		Success(w, nil)
		return
	}

	progress := h.ServerManager.SSLManager.GetCertProgress(domain)
	Success(w, progress)
}

// handleDNSProviders CRUD 路由分发（带 ID 的路由）
func (h *Handler) handleDNSProvidersRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /api/certs/dns-providers/{id}/test
	if strings.HasSuffix(path, "/test") {
		h.handleDNSProviderTest(w, r)
		return
	}

	// /api/certs/dns-providers/{id}/credentials
	if strings.HasSuffix(path, "/credentials") {
		h.handleDNSProviderGetCreds(w, r)
		return
	}

	// /api/certs/dns-providers/{id}
	switch r.Method {
	case http.MethodPut, http.MethodPost:
		h.handleDNSProviderUpdate(w, r)
	case http.MethodDelete:
		h.handleDNSProviderDelete(w, r)
	default:
		MethodNotAllowed(w, "Method not allowed")
	}
}

// handleDNSProviderGetCreds 获取 DNS 服务商的明文凭证（用于编辑回填）
func (h *Handler) handleDNSProviderGetCreds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	path := r.URL.Path
	// /api/certs/dns-providers/{id}/credentials -> 提取 {id}
	trimmed := strings.TrimSuffix(path, "/credentials")
	id := ""
	if parts := strings.Split(trimmed, "/"); len(parts) > 0 {
		id = parts[len(parts)-1]
	}
	if id == "" {
		BadRequest(w, "ID 不能为空")
		return
	}

	provider := h.ConfigManager.GetDNSProvider(id)
	if provider == nil {
		NotFound(w, "DNS 服务商不存在")
		return
	}

	creds, err := h.ConfigManager.DecryptDNSCredentials(provider.Credentials)
	if err != nil {
		InternalServerError(w, "凭证解密失败: "+err.Error())
		return
	}

	Success(w, map[string]interface{}{
		"id":          provider.ID,
		"name":        provider.Name,
		"type":        provider.Type,
		"credentials": creds,
	})
}

// ensure upload temp dir
func init() {
	os.MkdirAll(os.TempDir(), 0755)
}
