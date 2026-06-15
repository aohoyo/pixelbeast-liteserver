package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ==================== 默认配置测试 ====================

func TestDefaultServerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// 验证基础字段
	if cm.Server.Name != "PixelBeast Server" {
		t.Errorf("Server.Name = %q, 期望 %q", cm.Server.Name, "PixelBeast Server")
	}
	if cm.Server.Admin.Port != 9527 {
		t.Errorf("Admin.Port = %d, 期望 9527", cm.Server.Admin.Port)
	}
	if cm.Server.Admin.Username != "admin" {
		t.Errorf("Admin.Username = %q, 期望 %q", cm.Server.Admin.Username, "admin")
	}
	if cm.Server.Admin.Path != "/admin" {
		t.Errorf("Admin.Path = %q, 期望 %q", cm.Server.Admin.Path, "/admin")
	}
	if cm.Server.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %q, 期望 %q", cm.Server.Timezone, "Asia/Shanghai")
	}

	// 目录默认值
	if cm.Server.Directories.Sites != "./sites" {
		t.Errorf("Directories.Sites = %q, 期望 %q", cm.Server.Directories.Sites, "./sites")
	}
	if cm.Server.Directories.FTP != "./ftp" {
		t.Errorf("Directories.FTP = %q, 期望 %q", cm.Server.Directories.FTP, "./ftp")
	}
	if cm.Server.Directories.Backup != "./backups" {
		t.Errorf("Directories.Backup = %q, 期望 %q", cm.Server.Directories.Backup, "./backups")
	}

	// 备份配置
	if cm.Server.Backup.Schedule != "daily" {
		t.Errorf("Backup.Schedule = %q, 期望 %q", cm.Server.Backup.Schedule, "daily")
	}
	if cm.Server.Backup.Retention != 3 {
		t.Errorf("Backup.Retention = %d, 期望 3", cm.Server.Backup.Retention)
	}

	// 日志配置
	if cm.Server.Log.RetentionDays != 30 {
		t.Errorf("Log.RetentionDays = %d, 期望 30", cm.Server.Log.RetentionDays)
	}
	if cm.Server.Log.MaxSizeMB != 100 {
		t.Errorf("Log.MaxSizeMB = %d, 期望 100", cm.Server.Log.MaxSizeMB)
	}

	// 自启动
	if cm.Server.AutoStart.Enabled != true {
		t.Error("AutoStart.Enabled 应为 true")
	}

	// 密码应加密存储
	if cm.Server.Admin.Password == "" {
		t.Error("Admin.Password 不应为空")
	}
	if len(cm.Server.Admin.Password) < 20 {
		t.Error("Admin.Password 应为加密后的密文")
	}

	// 默认站点
	if len(cm.Sites.Sites) != 1 {
		t.Errorf("默认站点数 = %d, 期望 1", len(cm.Sites.Sites))
	} else {
		s := cm.Sites.Sites[0]
		if s.ID != "default" {
			t.Errorf("默认站点 ID = %q, 期望 %q", s.ID, "default")
		}
		if s.Type != "static" {
			t.Errorf("默认站点 Type = %q, 期望 %q", s.Type, "static")
		}
	}

	// 默认 FTP 配置
	if cm.FTP.Enabled != false {
		t.Error("FTP.Enabled 默认应为 false")
	}
	if cm.FTP.Port != 2121 {
		t.Errorf("FTP.Port = %d, 期望 2121", cm.FTP.Port)
	}
}

// ==================== 保存/加载测试 ====================

func TestSaveServerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// 修改配置并保存
	cm.Server.Name = "Test Server"
	cm.Server.Admin.Port = 8080
	if err := cm.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 验证文件已写入
	data, err := os.ReadFile(filepath.Join(tmpDir, "server.json"))
	if err != nil {
		t.Fatalf("读取 server.json 失败: %v", err)
	}

	var saved ServerConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("解析 server.json 失败: %v", err)
	}
	if saved.Name != "Test Server" {
		t.Errorf("保存后 Name = %q, 期望 %q", saved.Name, "Test Server")
	}
	if saved.Admin.Port != 8080 {
		t.Errorf("保存后 Port = %d, 期望 8080", saved.Admin.Port)
	}
}

func TestLoadServerConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// 第一次创建并修改
	cm1, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("第一次 NewConfigManager 失败: %v", err)
	}
	cm1.Server.Name = "Reload Test"
	cm1.Server.Admin.Port = 9090
	if err := cm1.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 重新加载验证持久化
	cm2, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("第二次 NewConfigManager 失败: %v", err)
	}
	if cm2.Server.Name != "Reload Test" {
		t.Errorf("重载后 Name = %q, 期望 %q", cm2.Server.Name, "Reload Test")
	}
	if cm2.Server.Admin.Port != 9090 {
		t.Errorf("重载后 Port = %d, 期望 9090", cm2.Server.Admin.Port)
	}
}

// ==================== 密码加密测试 ====================

func TestPasswordEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// 密码应使用 bcrypt 哈希存储
	if cm.Server.Admin.Password == "" {
		t.Error("Admin.Password 不应为空")
	}
	if !isBcryptHash(cm.Server.Admin.Password) {
		t.Errorf("Admin.Password 应为 bcrypt 哈希格式，实际: %q", cm.Server.Admin.Password)
	}

	// 验证初始密码文件不再写入磁盘
	passwordFile := filepath.Join(tmpDir, "initial_password.txt")
	if _, err := os.Stat(passwordFile); !os.IsNotExist(err) {
		t.Error("初始密码文件不应写入磁盘")
	}

	// 由于无法从 bcrypt 哈希反推明文，通过 ValidateAdmin 间接验证
	// 新创建的配置没有明文密码，只能通过 SetAdminPassword 设置后验证
	if err := cm.SetAdminPassword("testpass"); err != nil {
		t.Fatalf("SetAdminPassword 失败: %v", err)
	}

	// 验证密码正确
	if !cm.ValidateAdmin("admin", "testpass") {
		t.Error("密码验证应返回 true")
	}
	if cm.ValidateAdmin("admin", "wrong") {
		t.Error("错误密码应返回 false")
	}
	if cm.ValidateAdmin("wrong", "testpass") {
		t.Error("错误用户名应返回 false")
	}

	// 修改密码
	if err := cm.SetAdminPassword("newpass"); err != nil {
		t.Fatalf("SetAdminPassword 失败: %v", err)
	}
	if !cm.ValidateAdmin("admin", "newpass") {
		t.Error("新密码验证应返回 true")
	}
	if cm.ValidateAdmin("admin", "testpass") {
		t.Error("旧密码不应再有效")
	}

	// GetAdminPassword 对 bcrypt 哈希应返回错误
	_, err = cm.GetAdminPassword()
	if err == nil {
		t.Error("GetAdminPassword 对 bcrypt 哈希应返回错误")
	}
}

// ==================== FTP 用户管理测试 ====================

func TestFTPUserManagement(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// 添加用户
	user := FTPUser{Username: "testuser", RootPath: "/tmp/test", Status: "enabled"}
	if err := cm.AddFTPUser(user, "testpass"); err != nil {
		t.Fatalf("AddFTPUser 失败: %v", err)
	}

	// 重复添加应失败
	if err := cm.AddFTPUser(FTPUser{Username: "testuser"}, "x"); err == nil {
		t.Error("重复添加应返回错误")
	}

	// 验证密码
	if !cm.ValidateFTPUser("testuser", "testpass") {
		t.Error("密码验证应返回 true")
	}
	if cm.ValidateFTPUser("testuser", "wrong") {
		t.Error("错误密码应返回 false")
	}
	if cm.ValidateFTPUser("nouser", "testpass") {
		t.Error("不存在用户应返回 false")
	}

	// 获取用户
	u := cm.GetFTPUser("testuser")
	if u == nil {
		t.Fatal("GetFTPUser 返回 nil")
	}
	if u.Username != "testuser" {
		t.Errorf("Username = %q, 期望 %q", u.Username, "testuser")
	}

	// 密码字段应为 bcrypt 哈希（无法解密）
	_, err = cm.GetFTPUserPassword("testuser")
	if err == nil {
		t.Error("GetFTPUserPassword 对 bcrypt 哈希应返回错误")
	}

	// 验证密码通过 ValidateFTPUser
	if !cm.ValidateFTPUser("testuser", "testpass") {
		t.Error("密码验证应返回 true")
	}

	// 更新用户
	updated := *u
	updated.Remark = "测试备注"
	if err := cm.UpdateFTPUser("testuser", updated); err != nil {
		t.Fatalf("UpdateFTPUser 失败: %v", err)
	}
	u2 := cm.GetFTPUser("testuser")
	if u2.Remark != "测试备注" {
		t.Errorf("更新后 Remark = %q, 期望 %q", u2.Remark, "测试备注")
	}

	// 删除用户
	if err := cm.DeleteFTPUser("testuser"); err != nil {
		t.Fatalf("DeleteFTPUser 失败: %v", err)
	}
	if cm.GetFTPUser("testuser") != nil {
		t.Error("删除后 GetFTPUser 应返回 nil")
	}
}

// ==================== 站点管理测试 ====================

func TestSiteManagement(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// 默认有一个站点
	if len(cm.Sites.Sites) != 1 {
		t.Fatalf("默认站点数 = %d, 期望 1", len(cm.Sites.Sites))
	}

	// 添加新站点
	site := SiteConfig{
		ID:      "test-site",
		Name:    "测试站点",
		Type:    "static",
		Port:    8080,
		Enabled: true,
	}
	if err := cm.AddSite(site); err != nil {
		t.Fatalf("AddSite 失败: %v", err)
	}

	// 重复 ID 应失败
	if err := cm.AddSite(SiteConfig{ID: "test-site"}); err == nil {
		t.Error("重复 ID 应返回错误")
	}

	// 查询
	s := cm.GetSiteByID("test-site")
	if s == nil {
		t.Fatal("GetSiteByID 返回 nil")
	}
	if s.Name != "测试站点" {
		t.Errorf("Name = %q, 期望 %q", s.Name, "测试站点")
	}

	// 更新
	updated := *s
	updated.Name = "更新站点"
	if err := cm.UpdateSite("test-site", updated); err != nil {
		t.Fatalf("UpdateSite 失败: %v", err)
	}
	if cm.GetSiteByID("test-site").Name != "更新站点" {
		t.Error("更新后名称不匹配")
	}

	// 删除
	if err := cm.DeleteSite("test-site"); err != nil {
		t.Fatalf("DeleteSite 失败: %v", err)
	}
	if cm.GetSiteByID("test-site") != nil {
		t.Error("删除后应返回 nil")
	}
}

// ==================== DNS 服务商管理测试 ====================

func TestDNSProviderManagement(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// 添加 DNS 服务商
	p := DNSProviderConfig{
		ID:          "test-dns",
		Name:        "测试DNS",
		Type:        "alidns",
		Credentials: "encrypted-data",
	}
	if err := cm.AddDNSProvider(p); err != nil {
		t.Fatalf("AddDNSProvider 失败: %v", err)
	}

	// 重复 ID 应失败
	if err := cm.AddDNSProvider(p); err == nil {
		t.Error("重复 ID 应返回错误")
	}

	// 查询
	dp := cm.GetDNSProvider("test-dns")
	if dp == nil {
		t.Fatal("GetDNSProvider 返回 nil")
	}
	if dp.Name != "测试DNS" {
		t.Errorf("Name = %q, 期望 %q", dp.Name, "测试DNS")
	}

	// 列表
	providers := cm.GetDNSProviders()
	if len(providers) != 1 {
		t.Errorf("GetDNSProviders 长度 = %d, 期望 1", len(providers))
	}

	// 更新
	updated := *dp
	updated.Name = "更新DNS"
	if err := cm.UpdateDNSProvider("test-dns", updated); err != nil {
		t.Fatalf("UpdateDNSProvider 失败: %v", err)
	}

	// 删除
	if err := cm.DeleteDNSProvider("test-dns"); err != nil {
		t.Fatalf("DeleteDNSProvider 失败: %v", err)
	}
	if cm.GetDNSProvider("test-dns") != nil {
		t.Error("删除后应返回 nil")
	}
}

// ==================== 辅助方法测试 ====================

func TestHelperMethods(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}

	// ConfigDir
	if cm.ConfigDir() != tmpDir {
		t.Errorf("ConfigDir = %q, 期望 %q", cm.ConfigDir(), tmpDir)
	}

	// GetSitesDir
	if cm.GetSitesDir() != "./sites" {
		t.Errorf("GetSitesDir = %q, 期望 %q", cm.GetSitesDir(), "./sites")
	}

	// GetFTPRoot
	if cm.GetFTPRoot() != "./ftp" {
		t.Errorf("GetFTPRoot = %q, 期望 %q", cm.GetFTPRoot(), "./ftp")
	}

	// GetBackupDir
	if cm.GetBackupDir() != "./backups" {
		t.Errorf("GetBackupDir = %q, 期望 %q", cm.GetBackupDir(), "./backups")
	}

	// GetSharedPort（默认无站点使用同一端口，返回 0）
	port := cm.GetSharedPort()
	if port != 0 {
		t.Errorf("GetSharedPort = %d, 期望 0", port)
	}

	// GetSiteRoot
	site := cm.GetSiteByID("default")
	if site == nil {
		t.Fatal("默认站点不存在")
	}
	root := cm.GetSiteRoot(site)
	expected := filepath.Join("./sites", "default")
	if root != expected {
		t.Errorf("GetSiteRoot = %q, 期望 %q", root, expected)
	}

	// 自定义根目录
	site.Root = "/custom/path"
	if cm.GetSiteRoot(site) != "/custom/path" {
		t.Errorf("自定义根目录不匹配")
	}

	// ResetToDefaults
	if err := cm.ResetToDefaults(); err != nil {
		t.Fatalf("ResetToDefaults 失败: %v", err)
	}
	if cm.Server.Name != "PixelBeast Server" {
		t.Errorf("重置后 Name = %q, 期望 %q", cm.Server.Name, "PixelBeast Server")
	}
}

func TestSSLConfigHelpers(t *testing.T) {
	// IsAutoCert
	s := &SSLConfig{Enabled: true, AutoHTTPS: true, ChallengeMethod: "http-auto"}
	if !s.IsAutoCert() {
		t.Error("IsAutoCert 应返回 true")
	}

	s = &SSLConfig{Enabled: true, AutoHTTPS: true, ChallengeMethod: "dns"}
	if s.IsAutoCert() {
		t.Error("DNS 验证不应为 AutoCert")
	}

	s = nil
	if s.IsAutoCert() {
		t.Error("nil 不应为 AutoCert")
	}

	// IsCustomCert
	s = &SSLConfig{Enabled: true, AutoHTTPS: false, CertFile: "cert.pem", KeyFile: "key.pem"}
	if !s.IsCustomCert() {
		t.Error("IsCustomCert 应返回 true")
	}

	// GetProvider
	s = &SSLConfig{Provider: ""}
	if s.GetProvider() != "letsencrypt" {
		t.Errorf("默认 Provider = %q, 期望 letsencrypt", s.GetProvider())
	}

	// GetChallengeMethod
	s = &SSLConfig{ChallengeMethod: ""}
	if s.GetChallengeMethod() != "http-auto" {
		t.Errorf("默认 ChallengeMethod = %q, 期望 http-auto", s.GetChallengeMethod())
	}
}
