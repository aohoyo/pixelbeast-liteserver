package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager 通知管理器
type Manager struct {
	mu       sync.RWMutex
	config   *NotifyConfig
	configPath string
	// 告警状态跟踪（避免重复告警）
	lastAlert map[string]int64 // ruleID -> 上次告警时间戳
	// 停止通道
	stopCh chan struct{}
}

// NewManager 创建通知管理器
func NewManager(configPath string) (*Manager, error) {
	m := &Manager{
		configPath: configPath,
		lastAlert:  make(map[string]int64),
		stopCh:     make(chan struct{}),
	}

	// 加载配置
	if err := m.load(); err != nil {
		if os.IsNotExist(err) {
			// 首次运行，创建默认配置
			m.config = &NotifyConfig{
				Enabled: false,
				Channels: map[string]Channel{},
				Rules:    []AlertRule{},
				History:  []Alert{},
			}
			_ = m.save()
		} else {
			return nil, err
		}
	}

	return m, nil
}

// load 加载配置
func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.config)
}

// save 保存配置
func (m *Manager) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath, data, 0600)
}

// GetConfig 获取配置（只读）
func (m *Manager) GetConfig() NotifyConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config
}

// UpdateConfig 更新配置
func (m *Manager) UpdateConfig(cfg NotifyConfig) error {
	m.mu.Lock()
	m.config = &cfg
	m.mu.Unlock()
	return m.save()
}

// FireAlert 触发告警
func (m *Manager) FireAlert(alertType AlertType, level AlertLevel, source, title, message string) {
	m.mu.Lock()

	if !m.config.Enabled {
		m.mu.Unlock()
		return
	}

	// 创建告警记录
	alert := Alert{
		ID:        uuid.New().String(),
		Type:      alertType,
		Level:     level,
		Status:    AlertStatusFiring,
		Title:     title,
		Message:   message,
		Source:    source,
		CreatedAt: time.Now().Unix(),
	}

	// 添加到历史（最多保留 500 条）
	m.config.History = append(m.config.History, alert)
	if len(m.config.History) > 500 {
		m.config.History = m.config.History[len(m.config.History)-500:]
	}

	// 查找匹配规则，检查静默期
	silence := int64(600) // 默认 10 分钟
	for _, rule := range m.config.Rules {
		if rule.Type == alertType && rule.Enabled {
			if rule.SilenceSeconds > 0 {
				silence = int64(rule.SilenceSeconds)
			}
		}
	}

	alertKey := fmt.Sprintf("%s:%s", alertType, source)
	now := time.Now().Unix()
	if lastFire, ok := m.lastAlert[alertKey]; ok && (now-lastFire) < silence {
		// 静默期内，只记录不通知
		_ = m.save()
		m.mu.Unlock()
		return
	}
	m.lastAlert[alertKey] = now

	// 获取通知渠道
	channels := m.getChannelsForType(alertType)

	m.mu.Unlock()

	// 保存配置（含历史记录）
	_ = m.save()

	// 异步发送通知
	for _, ch := range channels {
		go func(c Channel) {
			if err := Send(c, title, message); err != nil {
				fmt.Printf("[通知] 发送失败 [%s]: %v\n", c.Type, err)
			}
		}(ch)
	}
}

// ResolveAlert 标记告警恢复
func (m *Manager) ResolveAlert(source string, title, message string) {
	m.mu.Lock()

	if !m.config.Enabled {
		m.mu.Unlock()
		return
	}

	alert := Alert{
		ID:         uuid.New().String(),
		Type:       AlertTypeHealthCheck,
		Level:      AlertLevelInfo,
		Status:     AlertStatusResolved,
		Title:      title,
		Message:    message,
		Source:     source,
		CreatedAt:  time.Now().Unix(),
		ResolvedAt: time.Now().Unix(),
	}

	m.config.History = append(m.config.History, alert)
	if len(m.config.History) > 500 {
		m.config.History = m.config.History[len(m.config.History)-500:]
	}

	// 清除静默期，允许恢复通知
	delete(m.lastAlert, fmt.Sprintf("health_check:%s", source))

	channels := m.getChannelsForType(AlertTypeHealthCheck)
	m.mu.Unlock()

	_ = m.save()

	for _, ch := range channels {
		go func(c Channel) {
			if err := Send(c, title, message); err != nil {
				fmt.Printf("[通知] 恢复通知发送失败 [%s]: %v\n", c.Type, err)
			}
		}(ch)
	}
}

// getChannelsForType 获取指定告警类型的通知渠道
func (m *Manager) getChannelsForType(alertType AlertType) []Channel {
	var channels []Channel
	for _, rule := range m.config.Rules {
		if rule.Type == alertType && rule.Enabled {
			for _, chName := range rule.Channels {
				if ch, ok := m.config.Channels[chName]; ok && ch.Enabled {
					channels = append(channels, ch)
				}
			}
		}
	}
	// 如果规则没指定渠道，发送到所有启用的渠道
	if len(channels) == 0 {
		for _, ch := range m.config.Channels {
			if ch.Enabled {
				channels = append(channels, ch)
			}
		}
	}
	return channels
}

// GetHistory 获取告警历史
func (m *Manager) GetHistory(limit int) []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history := m.config.History
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}
	// 倒序返回
	result := make([]Alert, len(history))
	for i, h := range history {
		result[len(history)-1-i] = h
	}
	return result
}

// ClearHistory 清空告警历史
func (m *Manager) ClearHistory() error {
	m.mu.Lock()
	m.config.History = []Alert{}
	m.mu.Unlock()
	return m.save()
}

// Stop 停止通知管理器
func (m *Manager) Stop() {
	close(m.stopCh)
}
