package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

// handleSitesList 处理站点列表
func (h *Handler) handleSitesList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.listSites(w, r)
	} else if r.Method == http.MethodPost {
		h.createSite(w, r)
	} else {
		MethodNotAllowed(w, "方法不允许")
	}
}

// handleSitesDetail 处理站点详情
func (h *Handler) handleSitesDetail(w http.ResponseWriter, r *http.Request) {
	// 从路径提取站点 ID
	id := extractIDFromPath(r.URL.Path, "/api/sites/")
	if id == "" {
		BadRequest(w, "缺少站点 ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSite(w, r, id)
	case http.MethodPut:
		h.updateSite(w, r, id)
	case http.MethodDelete:
		h.deleteSite(w, r, id)
	default:
		MethodNotAllowed(w, "方法不允许")
	}
}

// handleSiteToggle 处理站点启用/禁用
func (h *Handler) handleSiteToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "方法不允许")
		return
	}

	var req struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	site := h.ConfigManager.GetSiteByID(req.ID)
	if site == nil {
		NotFound(w, "站点不存在")
		return
	}

	site.Enabled = req.Enabled
	site.UpdatedAt = time.Now().Format(time.RFC3339)

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败")
			return
		}
	}

	// 立即生效（无需全量重载）
	if h.ServerManager != nil {
		if req.Enabled {
			site := h.ConfigManager.GetSiteByID(req.ID)
			if site != nil {
				h.ServerManager.AddSiteRuntime(site)
			}
		} else {
			h.ServerManager.DeleteSiteRuntime(req.ID)
		}
	}

	username := h.getSessionUsername(r)
	siteName := h.ConfigManager.GetSiteByID(req.ID)
	siteNameStr := req.ID
	if siteName != nil { siteNameStr = siteName.Name }
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 切换状态: %s -> %v (用户=%s)", siteNameStr, req.Enabled, username)
	SuccessMessage(w, "站点状态已更新")
}

// handleSiteStart 启动单个站点
func (h *Handler) handleSiteStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "方法不允许")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}

	if err := h.ServerManager.StartSite(req.ID); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 启动: %s (用户=%s)", req.ID, username)
	SuccessMessage(w, "站点已启动")
}

// handleSiteStop 停止单个站点
func (h *Handler) handleSiteStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "方法不允许")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}

	if err := h.ServerManager.StopSite(req.ID); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 停止: %s (用户=%s)", req.ID, username)
	SuccessMessage(w, "站点已停止")
}

// handleSiteRestart 重启单个站点
func (h *Handler) handleSiteRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "方法不允许")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}

	if err := h.ServerManager.RestartSite(req.ID); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 重启: %s (用户=%s)", req.ID, username)
	SuccessMessage(w, "站点已重启")
}

// getSitesStatus 获取站点服务状态
func (h *Handler) getSitesStatus(w http.ResponseWriter, r *http.Request) {
	sitesRunning := false
	sitesPort := 0
	if h.ServerManager != nil {
		sitesRunning = h.ServerManager.IsSitesRunning()
	}
	if h.ConfigManager != nil {
		sitesPort = h.ConfigManager.GetSharedPort()
	}
	Success(w, map[string]interface{}{
		"running": sitesRunning,
		"port":    sitesPort,
	})
}

// toggleSitesService 切换站点服务
func (h *Handler) toggleSitesService(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	var err error
	var msg string
	if h.ServerManager.IsSitesRunning() {
		err, msg = h.ServerManager.StopSitesServer(), "站点服务已停止"
	} else {
		err, msg = h.ServerManager.StartSitesServer(), "站点服务已启动"
	}
	if err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 切换站点服务: %s (用户=%s)", msg, username)
	SuccessMessage(w, msg)
}

// startSitesService 启动站点服务
func (h *Handler) startSitesService(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.StartSitesServer(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 启动站点服务 (用户=%s)", username)
	SuccessMessage(w, "站点服务已启动")
}

// stopSitesService 停止站点服务
func (h *Handler) stopSitesService(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.StopSitesServer(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 停止站点服务 (用户=%s)", username)
	SuccessMessage(w, "站点服务已停止")
}

// restartSitesService 重启站点服务
func (h *Handler) restartSitesService(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.RestartSitesServer(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 重启站点服务 (用户=%s)", username)
	SuccessMessage(w, "站点服务已重启")
}

// reloadSitesConfig 重载站点配置
func (h *Handler) reloadSitesConfig(w http.ResponseWriter, r *http.Request) {
	if h.ServerManager == nil {
		Error(w, http.StatusOK, "服务管理器未初始化")
		return
	}
	if err := h.ServerManager.ReloadSites(); err != nil {
		Error(w, http.StatusOK, err.Error())
		return
	}
	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[服务] 重载站点配置 (用户=%s)", username)
	SuccessMessage(w, "站点配置已重载")
}

// handleSitesBatch 处理批量操作
func (h *Handler) handleSitesBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "方法不允许")
		return
	}

	var req struct {
		Action string   `json:"action"` // enable, disable, delete
		IDs    []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	if len(req.IDs) == 0 {
		BadRequest(w, "未选择站点")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var count int
	for _, id := range req.IDs {
		site := h.ConfigManager.GetSiteByID(id)
		if site == nil {
			continue
		}

		switch req.Action {
		case "enable":
			site.Enabled = true
			site.UpdatedAt = time.Now().Format(time.RFC3339)
			count++
		case "disable":
			site.Enabled = false
			site.UpdatedAt = time.Now().Format(time.RFC3339)
			count++
		case "delete":
			if len(h.ConfigManager.Sites.Sites) <= 1 {
				continue // 不允许删除最后一个站点
			}
			if err := h.ConfigManager.DeleteSite(id); err == nil {
				count++
			}
		}
	}

	if count == 0 {
		SuccessMessage(w, "没有站点被修改")
		return
	}

	// 使用 ConfigManager 保存
	if h.ConfigManager != nil {
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败")
			return
		}
	}

	// 立即生效（无需全量重载）
	if h.ServerManager != nil {
		for _, id := range req.IDs {
			switch req.Action {
			case "enable":
				site := h.ConfigManager.GetSiteByID(id)
				if site != nil {
					h.ServerManager.AddSiteRuntime(site)
				}
			case "disable":
				h.ServerManager.DeleteSiteRuntime(id)
			case "delete":
				h.ServerManager.DeleteSiteRuntime(id)
			}
		}
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 批量%s: %d个站点 (用户=%s)", req.Action, count, username)
	SuccessMessage(w, fmt.Sprintf("已处理 %d 个站点", count))
}

// listSites 列出所有站点
func (h *Handler) listSites(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sites := make([]map[string]interface{}, 0)
	for i := range h.ConfigManager.Sites.Sites {
		m := siteToMap(h.ConfigManager.Sites.Sites[i])
		m["root"] = h.ConfigManager.GetSiteRoot(&h.ConfigManager.Sites.Sites[i])
		sites = append(sites, m)
	}

	Success(w, sites)
}

// createSite 创建站点
func (h *Handler) createSite(w http.ResponseWriter, r *http.Request) {
	var site config.SiteConfig
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	// 验证必填字段
	if site.Name == "" || site.Type == "" {
		BadRequest(w, "站点名称和类型不能为空")
		return
	}

	// 生成 ID
	if site.ID == "" {
		site.ID = generateID()
	}

	// 验证类型
	if site.Type != "static" && site.Type != "proxy" {
		BadRequest(w, "站点类型必须是 static 或 proxy")
		return
	}

	// 设置默认值
	if site.IndexFiles == nil {
		site.IndexFiles = []string{"index.html", "index.htm"}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查 ID 是否已存在
	if h.ConfigManager.GetSiteByID(site.ID) != nil {
		BadRequest(w, "站点 ID 已存在")
		return
	}

	// 添加站点
	if err := h.ConfigManager.AddSite(site); err != nil {
		InternalServerError(w, "添加站点失败")
		return
	}

	// 保存配置
	if h.ConfigManager != nil {
		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, "保存配置失败")
			return
		}
	}

	// 立即生效（无需全量重载）
	if h.ServerManager != nil {
		h.ServerManager.AddSiteRuntime(&site)
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 创建站点: %s (用户=%s)", site.Name, username)
	Success(w, siteToMap(site))
}

// getSite 获取站点详情
func (h *Handler) getSite(w http.ResponseWriter, r *http.Request, id string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	site := h.ConfigManager.GetSiteByID(id)
	if site == nil {
		NotFound(w, "站点不存在")
		return
	}

	Success(w, siteToMap(*site))
}

// updateSite 更新站点
func (h *Handler) updateSite(w http.ResponseWriter, r *http.Request, id string) {
	var site config.SiteConfig
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		BadRequest(w, "参数错误")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查站点是否存在
	if h.ConfigManager.GetSiteByID(id) == nil {
		NotFound(w, "站点不存在")
		return
	}

	// 更新站点
	if err := h.ConfigManager.UpdateSite(id, site); err != nil {
		InternalServerError(w, "更新站点失败: "+err.Error())
		return
	}

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	// 立即生效（无需全量重载）
	if h.ServerManager != nil {
		site := h.ConfigManager.GetSiteByID(id)
		if site != nil {
			h.ServerManager.UpdateSiteRuntime(site)
		}
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 更新站点: %s (用户=%s)", id, username)
	SuccessMessage(w, "站点已更新")
}

// deleteSite 删除站点
func (h *Handler) deleteSite(w http.ResponseWriter, r *http.Request, id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查站点是否存在
	site := h.ConfigManager.GetSiteByID(id)
	if site == nil {
		NotFound(w, "站点不存在")
		return
	}

	// 不允许删除最后一个站点
	if len(h.ConfigManager.Sites.Sites) <= 1 {
		BadRequest(w, "不能删除最后一个站点")
		return
	}

	// 删除站点
	if err := h.ConfigManager.DeleteSite(id); err != nil {
		InternalServerError(w, "删除站点失败: "+err.Error())
		return
	}

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	// 立即生效（无需全量重载）
	if h.ServerManager != nil {
		h.ServerManager.DeleteSiteRuntime(id)
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[站点] 删除站点: %s (用户=%s)", site.Name, username)
	SuccessMessage(w, "站点已删除")
}

// siteToMap 将站点配置转换为 map（用于 JSON 响应）
func siteToMap(site config.SiteConfig) map[string]interface{} {
	return map[string]interface{}{
		"id":          site.ID,
		"name":        site.Name,
		"enabled":     site.Enabled,
		"type":        site.Type,
		"port":        site.Port,
		"domain":      site.Domain,
		"index_files": site.IndexFiles,
		"auto_index":  site.AutoIndex,
		"proxy":       site.Proxy,
		"ssl":         site.SSL,
		"created_at":  site.CreatedAt,
		"updated_at":  site.UpdatedAt,
	}
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("site_%d", time.Now().UnixNano())
}

// extractIDFromPath 从路径中提取 ID
func extractIDFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}
