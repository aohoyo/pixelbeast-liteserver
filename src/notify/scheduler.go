package notify

import (
	"fmt"
	"strings"
	"time"
)

// Scheduler 定时检测调度器
type Scheduler struct {
	manager    *Manager
	siteLoader SiteLoader // 加载站点配置的回调
	certLoader CertLoader // 加载证书路径的回调
	stopCh     chan struct{}
}

// SiteLoader 站点配置加载接口
type SiteLoader interface {
	GetSiteCheckTargets() []SiteCheckTarget
}

// SiteCheckTarget 站点检测目标
type SiteCheckTarget struct {
	SiteID   string
	Name     string
	URL      string
	CertFile string // SSL 证书文件路径
}

// CertLoader 证书加载接口
type CertLoader interface {
	GetAllCertFiles() []string
}

// NewScheduler 创建调度器
func NewScheduler(manager *Manager, siteLoader SiteLoader, certLoader CertLoader) *Scheduler {
	return &Scheduler{
		manager:    manager,
		siteLoader: siteLoader,
		certLoader: certLoader,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动定时检测
func (s *Scheduler) Start() {
	cfg := s.manager.GetConfig()

	// 启动健康检查
	for _, rule := range cfg.Rules {
		if rule.Type == AlertTypeHealthCheck && rule.Enabled && rule.CheckInterval > 0 {
			go s.runHealthCheck(rule)
		}
	}

	// 启动 SSL 证书检查（每小时一次）
	go s.runCertCheck()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// runHealthCheck 定时健康检查
func (s *Scheduler) runHealthCheck(rule AlertRule) {
	interval := time.Duration(rule.CheckInterval) * time.Second
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	timeout := rule.CheckTimeout
	if timeout <= 0 {
		timeout = 10
	}
	expectedStatus := rule.ExpectedStatus

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次立即执行
	s.doHealthCheck(rule, timeout, expectedStatus)

	for {
		select {
		case <-ticker.C:
			s.doHealthCheck(rule, timeout, expectedStatus)
		case <-s.stopCh:
			return
		}
	}
}

// doHealthCheck 执行健康检查
func (s *Scheduler) doHealthCheck(rule AlertRule, timeout, expectedStatus int) {
	cfg := s.manager.GetConfig()

	// 获取检测 URL
	urls := rule.CheckURLs
	if len(urls) == 0 && s.siteLoader != nil {
		targets := s.siteLoader.GetSiteCheckTargets()
		for _, t := range targets {
			if t.URL != "" {
				urls = append(urls, t.URL)
			}
		}
	}

	for _, url := range urls {
		result := CheckHTTP(url, timeout, expectedStatus)

		if result.Status == "down" {
			msg := fmt.Sprintf("**URL**: %s\n**状态**: %s\n**错误**: %s\n**延迟**: %dms",
				result.URL, result.Status, result.Error, result.Latency)
			level := AlertLevelCritical
			if result.StatusCode >= 500 {
				level = AlertLevelWarning
			}
			s.manager.FireAlert(AlertTypeHealthCheck, level, url,
				"站点不可达", msg)
		} else {
			s.manager.ResolveAlert(url,
				"站点已恢复",
				fmt.Sprintf("**URL**: %s\n**延迟**: %dms", url, result.Latency))
		}
	}
	_ = cfg // 避免未使用警告
}

// runCertCheck 定时证书检查
func (s *Scheduler) runCertCheck() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// 首次延迟 1 分钟执行（等系统启动完成）
	select {
	case <-time.After(1 * time.Minute):
		s.doCertCheck()
	case <-s.stopCh:
		return
	}

	for {
		select {
		case <-ticker.C:
			s.doCertCheck()
		case <-s.stopCh:
			return
		}
	}
}

// doCertCheck 执行证书检查
func (s *Scheduler) doCertCheck() {
	cfg := s.manager.GetConfig()

	// 获取告警天数
	daysBefore := []int{30, 14, 7, 3, 1}
	for _, rule := range cfg.Rules {
		if rule.Type == AlertTypeSSLCert && rule.Enabled && len(rule.SSLDaysBefore) > 0 {
			daysBefore = rule.SSLDaysBefore
			break
		}
	}

	// 收集证书文件
	var certFiles []string
	if s.certLoader != nil {
		certFiles = s.certLoader.GetAllCertFiles()
	}
	if s.siteLoader != nil {
		targets := s.siteLoader.GetSiteCheckTargets()
		for _, t := range targets {
			if t.CertFile != "" {
				certFiles = append(certFiles, t.CertFile)
			}
		}
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueFiles []string
	for _, f := range certFiles {
		if !seen[f] && f != "" {
			seen[f] = true
			uniqueFiles = append(uniqueFiles, f)
		}
	}

	for _, certFile := range uniqueFiles {
		info, err := CheckCertExpiry(certFile)
		if err != nil {
			continue
		}

		// 检查是否需要告警
		for _, days := range daysBefore {
			if info.DaysLeft == days || (info.DaysLeft < days && info.DaysLeft > 0) {
				msg := fmt.Sprintf("**域名**: %s\n**颁发者**: %s\n**到期时间**: %s\n**剩余天数**: %d 天",
					info.Domain, info.Issuer,
					info.NotAfter.Format("2006-01-02 15:04:05"),
					info.DaysLeft)

				level := AlertLevelWarning
				if info.DaysLeft <= 3 {
					level = AlertLevelCritical
				}

				s.manager.FireAlert(AlertTypeSSLCert, level, certFile,
					fmt.Sprintf("SSL 证书即将到期 (%d 天)", info.DaysLeft), msg)
				break
			}
		}

		// 已过期
		if info.IsExpired {
			daysExpired := -info.DaysLeft
			msg := fmt.Sprintf("**域名**: %s\n**颁发者**: %s\n**到期时间**: %s\n**已过期**: %d 天",
				info.Domain, info.Issuer,
				info.NotAfter.Format("2006-01-02 15:04:05"),
				daysExpired)

			s.manager.FireAlert(AlertTypeSSLCert, AlertLevelCritical, certFile,
				fmt.Sprintf("SSL 证书已过期 (%d 天)", daysExpired), msg)
		}
	}
	_ = strings.TrimSpace("") // 避免 import 未使用
}
