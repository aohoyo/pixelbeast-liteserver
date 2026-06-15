package notify

// 告警级别
type AlertLevel string

const (
	AlertLevelWarning AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelInfo AlertLevel = "info"
)

// 告警状态
type AlertStatus string

const (
	AlertStatusFiring AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)

// 告警类型
type AlertType string

const (
	AlertTypeHealthCheck AlertType = "health_check"
	AlertTypeSSLCert AlertType = "ssl_cert"
)

// Alert 单条告警记录
type Alert struct {
	ID        string      `json:"id"`
	Type      AlertType   `json:"type"`
	Level     AlertLevel  `json:"level"`
	Status    AlertStatus `json:"status"`
	Title     string      `json:"title"`
	Message   string      `json:"message"`
	Source    string      `json:"source"`    // 关联的站点/证书 ID
	CreatedAt int64       `json:"created_at"` // Unix 时间戳
	ResolvedAt int64      `json:"resolved_at,omitempty"`
}

// NotifyConfig 通知配置
type NotifyConfig struct {
	Enabled  bool                `json:"enabled"`
	Channels map[string]Channel  `json:"channels"`  // 通知渠道
	Rules    []AlertRule         `json:"rules"`     // 告警规则
	History  []Alert             `json:"history"`   // 告警历史
}

// Channel 通知渠道配置
type Channel struct {
	Type    string `json:"type"`    // feishu, email, telegram, browser
	Enabled bool   `json:"enabled"`
	// 飞书
	// 邮件 SMTP
	SMTPHost string `json:"smtp_host,omitempty"`
	SMTPPort int    `json:"smtp_port,omitempty"`
	SMTPUser string `json:"smtp_user,omitempty"`
	SMTPPass string `json:"smtp_pass,omitempty"`
	To       string `json:"to,omitempty"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        AlertType  `json:"type"`
	Enabled     bool       `json:"enabled"`
	// 健康检查规则
	CheckURLs      []string `json:"check_urls,omitempty"`
	CheckInterval  int      `json:"check_interval,omitempty"`  // 秒
	CheckTimeout   int      `json:"check_timeout,omitempty"`   // 秒
	ExpectedStatus int      `json:"expected_status,omitempty"`  // HTTP 状态码
	// SSL 规则
	SSLDaysBefore []int `json:"ssl_days_before,omitempty"` // 提前几天告警，如 [30,14,7,3,1]
	// 通用
	SilenceSeconds int      `json:"silence_seconds,omitempty"` // 静默期（秒）
	Channels       []string `json:"channels,omitempty"`        // 通知渠道名称
}
