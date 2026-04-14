package panel

import (
	"encoding/json"
	"net/http"

	"pixelbeast/src/config"
	"pixelbeast/src/logger"
)

// ==================== 配置 ====================

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	// 序列化 ServerConfig，直接返回与 server.json 一致的结构
	cfg := *h.ConfigManager.Server
	cfg.Admin.Password = "" // 不返回密码
	Success(w, cfg)
}

func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	if r.Method != http.MethodPost {
		BadRequest(w, "Method not allowed")
		logger.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	// 使用 map 接收前端数据
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		logger.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	// 记住密码，反序列化后恢复
	oldPassword := h.ConfigManager.Server.Admin.Password

	// JSON 反序列化：直接将前端数据映射到 ServerConfig
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		BadRequest(w, "JSON 解析失败: "+err.Error())
		logger.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	var newCfg config.ServerConfig
	if err := json.Unmarshal(jsonBytes, &newCfg); err != nil {
		BadRequest(w, "配置格式错误: "+err.Error())
		logger.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	// 密码特殊处理：前端传空则保留原密码
	if newCfg.Admin.Password != "" {
		h.ConfigManager.SetAdminPassword(newCfg.Admin.Password)
	} else {
		newCfg.Admin.Password = oldPassword
	}

	*h.ConfigManager.Server = newCfg

	// 解析 sites 配置（单独的配置文件）
	if sites, ok := data["sites"].([]interface{}); ok {
		siteConfigs := make([]config.SiteConfig, 0, len(sites))
		for _, s := range sites {
			if siteMap, ok := s.(map[string]interface{}); ok {
				siteConfig := config.SiteConfig{}
				if v, ok := siteMap["id"].(string); ok {
					siteConfig.ID = v
				}
				if v, ok := siteMap["name"].(string); ok {
					siteConfig.Name = v
				}
				if v, ok := siteMap["enabled"].(bool); ok {
					siteConfig.Enabled = v
				}
				if v, ok := siteMap["port"].(float64); ok {
					siteConfig.Port = int(v)
				}
				siteConfigs = append(siteConfigs, siteConfig)
			}
		}
		h.ConfigManager.Sites.Sites = siteConfigs
	}

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[配置] 保存失败: %v", err)
		InternalServerError(w, err.Error())
		logger.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	logger.LogPanelConfigChange(username, "保存配置", true)
	SuccessMessage(w, "配置已保存")
}

func (h *Handler) resetConfig(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	if r.Method != http.MethodPost {
		BadRequest(w, "Method not allowed")
		return
	}

	if h.ConfigManager == nil {
		InternalServerError(w, "配置管理器未初始化")
		return
	}

	// 使用 ConfigManager 统一的默认配置方法重置
	if err := h.ConfigManager.ResetToDefaults(); err != nil {
		InternalServerError(w, "重置配置失败: "+err.Error())
		return
	}

	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, err.Error())
		logger.LogPanelConfigChange(username, "重置配置", false)
		return
	}

	logger.LogPanelConfigChange(username, "重置配置", true)

	// 返回重置后的完整配置
	cfg := *h.ConfigManager.Server
	cfg.Admin.Password = ""
	Success(w, cfg)
}
