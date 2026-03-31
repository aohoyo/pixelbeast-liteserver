package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"pixelbeast/src/config"
)

// VirtualHost 虚拟主机
type VirtualHost struct {
	Config  *config.SiteConfig
	Handler http.Handler
}

// VirtualHostRouter 虚拟主机路由器
type VirtualHostRouter struct {
	mu          sync.RWMutex
	hosts       map[string]*VirtualHost // domain -> host
	defaultHost *VirtualHost
	portBased   map[int]*VirtualHost // port -> host
	sharedPort  int                  // 共享端口（如 8080）
}

// NewVirtualHostRouter 创建虚拟主机路由器
func NewVirtualHostRouter() *VirtualHostRouter {
	return &VirtualHostRouter{
		hosts:      make(map[string]*VirtualHost),
		portBased:  make(map[int]*VirtualHost),
		sharedPort: 3380,
	}
}

// SetSharedPort 设置共享端口
func (r *VirtualHostRouter) SetSharedPort(port int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sharedPort = port
}

// AddHost 添加虚拟主机
func (r *VirtualHostRouter) AddHost(cfg *config.SiteConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 根据站点类型创建处理器
	var handler http.Handler
	var err error

	switch cfg.Type {
	case "static":
		handler = NewHTTPServer(cfg.Root)
	case "proxy":
		if cfg.Proxy == nil {
			return fmt.Errorf("proxy type requires proxy config")
		}
		handler, err = NewProxyHandler(cfg.Proxy)
		if err != nil {
			return fmt.Errorf("create proxy handler: %w", err)
		}
	default:
		return fmt.Errorf("unknown site type: %s", cfg.Type)
	}

	vh := &VirtualHost{
		Config:  cfg,
		Handler: handler,
	}

	// 独立端口路由
	if cfg.Port > 0 && cfg.Port != r.sharedPort {
		r.portBased[cfg.Port] = vh
		log.Printf("[Vhost] 站点 %s 绑定独立端口 %d", cfg.Name, cfg.Port)
	}

	// 域名路由
	for _, domain := range cfg.Domain {
		if domain != "" {
			r.hosts[domain] = vh
			log.Printf("[Vhost] 站点 %s 绑定域名 %s", cfg.Name, domain)
		}
	}

	// 设置为默认主机（第一个启用的站点）
	if r.defaultHost == nil && cfg.Enabled {
		r.defaultHost = vh
		log.Printf("[Vhost] 站点 %s 设为默认站点", cfg.Name)
	}

	return nil
}

// RemoveHost 移除虚拟主机
func (r *VirtualHostRouter) RemoveHost(siteID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 收集要删除的域名
	domainsToRemove := make([]string, 0)
	portToRemove := 0

	for domain, vh := range r.hosts {
		if vh.Config.ID == siteID {
			domainsToRemove = append(domainsToRemove, domain)
		}
	}

	for port, vh := range r.portBased {
		if vh.Config.ID == siteID {
			portToRemove = port
			break
		}
	}

	// 删除域名映射
	for _, domain := range domainsToRemove {
		delete(r.hosts, domain)
	}

	// 删除端口映射
	if portToRemove > 0 {
		delete(r.portBased, portToRemove)
	}

	// 如果删除的是默认主机，重新选择
	if r.defaultHost != nil && r.defaultHost.Config.ID == siteID {
		r.defaultHost = nil
		for _, vh := range r.hosts {
			if vh.Config.Enabled {
				r.defaultHost = vh
				break
			}
		}
	}
}

// UpdateHost 更新虚拟主机
func (r *VirtualHostRouter) UpdateHost(cfg *config.SiteConfig) error {
	// 先移除旧的
	r.RemoveHost(cfg.ID)
	// 再添加新的
	return r.AddHost(cfg)
}

// ServeHTTP 实现 http.Handler
func (r *VirtualHostRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 提取主机名和端口
	host := req.Host
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}

	// 1. 优先检查独立端口路由
	// 注意：这里需要从实际的监听端口获取，简化处理
	// 实际使用时，可以在 ServeHTTP 外层包装来区分端口

	// 2. 检查域名路由
	if host != "" {
		if vh, ok := r.hosts[host]; ok && vh.Config.Enabled {
			vh.Handler.ServeHTTP(w, req)
			return
		}

		// 检查通配符域名 (*.example.com)
		for domain, vh := range r.hosts {
			if strings.HasPrefix(domain, "*.") && vh.Config.Enabled {
				suffix := domain[1:] // *.example.com -> .example.com
				if strings.HasSuffix(host, suffix) {
					vh.Handler.ServeHTTP(w, req)
					return
				}
			}
		}
	}

	// 3. 返回默认站点
	if r.defaultHost != nil && r.defaultHost.Config.Enabled {
		r.defaultHost.Handler.ServeHTTP(w, req)
		return
	}

	// 4. 没有匹配的站点
	http.NotFound(w, req)
}

// GetHostInfo 获取主机信息（用于状态显示）
func (r *VirtualHostRouter) GetHostInfo() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make([]map[string]interface{}, 0)

	// 去重：每个站点只显示一次
	seen := make(map[string]bool)

	for _, vh := range r.hosts {
		if !seen[vh.Config.ID] {
			seen[vh.Config.ID] = true
			info = append(info, map[string]interface{}{
				"id":      vh.Config.ID,
				"name":    vh.Config.Name,
				"type":    vh.Config.Type,
				"enabled": vh.Config.Enabled,
				"domains": vh.Config.Domain,
				"port":    vh.Config.Port,
			})
		}
	}

	for _, vh := range r.portBased {
		if !seen[vh.Config.ID] {
			seen[vh.Config.ID] = true
			info = append(info, map[string]interface{}{
				"id":      vh.Config.ID,
				"name":    vh.Config.Name,
				"type":    vh.Config.Type,
				"enabled": vh.Config.Enabled,
				"domains": vh.Config.Domain,
				"port":    vh.Config.Port,
			})
		}
	}

	return info
}

// Reload 重新加载所有站点配置
func (r *VirtualHostRouter) Reload(sites []config.SiteConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空现有配置
	r.hosts = make(map[string]*VirtualHost)
	r.portBased = make(map[int]*VirtualHost)
	r.defaultHost = nil

	// 重新添加站点
	for _, site := range sites {
		if site.Enabled {
			if err := r.AddHost(&site); err != nil {
				log.Printf("[Vhost] 重载站点 %s 失败: %v", site.Name, err)
			}
		}
	}

	return nil
}
