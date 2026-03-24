package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pixelbeast/src/config"
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
	id := extractIDFromPath(r.URL.Path, "/admin/api/sites/")
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

	site := h.Config.GetSiteByID(req.ID)
	if site == nil {
		NotFound(w, "站点不存在")
		return
	}

	site.Enabled = req.Enabled
	site.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.Config.Save(h.ConfigPath); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	// 重新加载站点
	if h.ServerManager != nil {
		h.ServerManager.ReloadSites()
	}

	SuccessMessage(w, "站点状态已更新")
}

// listSites 列出所有站点
func (h *Handler) listSites(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sites := make([]map[string]interface{}, 0)
	for _, site := range h.Config.Sites {
		sites = append(sites, siteToMap(site))
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
	if h.Config.GetSiteByID(site.ID) != nil {
		BadRequest(w, "站点 ID 已存在")
		return
	}

	// 添加站点
	if err := h.Config.AddSite(site); err != nil {
		InternalServerError(w, "添加站点失败")
		return
	}

	// 保存配置
	if err := h.Config.Save(h.ConfigPath); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	// 创建站点根目录
	if site.Type == "static" && site.Root != "" {
		// 目录会在 CreateDefaultDirectories 中创建
	}

	// 重新加载站点
	if h.ServerManager != nil {
		h.ServerManager.ReloadSites()
	}

	Success(w, siteToMap(site))
}

// getSite 获取站点详情
func (h *Handler) getSite(w http.ResponseWriter, r *http.Request, id string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	site := h.Config.GetSiteByID(id)
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
	if h.Config.GetSiteByID(id) == nil {
		NotFound(w, "站点不存在")
		return
	}

	// 更新站点
	if !h.Config.UpdateSite(id, site) {
		InternalServerError(w, "更新站点失败")
		return
	}

	// 保存配置
	if err := h.Config.Save(h.ConfigPath); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	// 重新加载站点
	if h.ServerManager != nil {
		h.ServerManager.ReloadSites()
	}

	SuccessMessage(w, "站点已更新")
}

// deleteSite 删除站点
func (h *Handler) deleteSite(w http.ResponseWriter, r *http.Request, id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查站点是否存在
	site := h.Config.GetSiteByID(id)
	if site == nil {
		NotFound(w, "站点不存在")
		return
	}

	// 不允许删除最后一个站点
	if len(h.Config.Sites) <= 1 {
		BadRequest(w, "不能删除最后一个站点")
		return
	}

	// 删除站点
	if !h.Config.DeleteSite(id) {
		InternalServerError(w, "删除站点失败")
		return
	}

	// 保存配置
	if err := h.Config.Save(h.ConfigPath); err != nil {
		InternalServerError(w, "保存配置失败")
		return
	}

	// 重新加载站点
	if h.ServerManager != nil {
		h.ServerManager.ReloadSites()
	}

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
		"root":        site.Root,
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
