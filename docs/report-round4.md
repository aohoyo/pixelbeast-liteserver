# Round 4 扣分点排查报告

**日期**: 2026-04-14
**基线**: `report-final.md` 第四轮后评分（综合 9.0）

---

## 一、当前评分（< 9.0 分项）

| 维度 | 当前分数 | 短板 |
|------|----------|------|
| **安全性** | 8.5 | 分享路径无白名单、canWriteDir 失效 |
| **错误处理** | 8.5 | 28 处非标准日志、exec 错误忽略 |
| **资源管理** | 8.5 | 部分清理操作错误忽略 |
| **前端质量** | 8.5 | escapeHtml 重复、off/once 泄漏、键盘劫持 |
| 代码质量 | 9.0 | — |
| 综合 | 9.0 | — |

---

## 二、最大提分点：非标准日志（影响「错误处理」+0.5 ~ +1.0）

**共 28 处** `fmt.Printf/Println` 或 `log.Printf` 应改为 `logger` 包的结构化日志。

### 2.1 panel/ 包 — 17 处

| 文件 | 行号 | 当前代码 | 应改为 |
|------|------|----------|--------|
| api_file.go | 1019 | `fmt.Printf("[Share] 加载分享数据失败: %v\n", err)` | `logger.LogPanelError(...)` |
| api_file.go | 1026 | `fmt.Printf("[Share] 解析分享数据失败: %v\n", err)` | `logger.LogPanelError(...)` |
| api_file.go | 1045 | `fmt.Printf("[Share] 已加载 %d 个有效分享链接\n", ...)` | `logger.LogPanelRuntime(...)` |
| api_file.go | 1059 | `fmt.Printf("[Share] 序列化分享数据失败: %v\n", err)` | `logger.LogPanelError(...)` |
| api_file.go | 1066 | `fmt.Printf("[Share] 创建目录失败: %v\n", err)` | `logger.LogPanelError(...)` |
| api_file.go | 1071 | `fmt.Printf("[Share] 保存分享数据失败: %v\n", err)` | `logger.LogPanelError(...)` |
| api_ftp.go | 588 | `fmt.Printf("[FTP] 删除用户目录失败: %v\n", err)` | `logger.LogFTPError(...)` |
| api_system.go | 1676 | `fmt.Printf("[NTP] %s 查询失败: %v\n", ...)` | `logger.LogPanelError(...)` |
| api_system.go | 1682 | `fmt.Printf("[NTP] %s 时间异常: %v\n", ...)` | `logger.LogPanelError(...)` |
| api_system.go | 1687 | `fmt.Printf("[NTP] %s 成功, 偏移: %v\n", ...)` | `logger.LogPanelRuntime(...)` |
| api_system.go | 1700 | `fmt.Println("[NTP] 所有服务器均查询失败")` | `logger.LogPanelError(...)` |
| api_ssl.go | 201 | `log.Printf("[SSL] DNS 验证失败 %s: %v", ...)` | `logger.LogPanelError(...)` |
| api_ssl.go | 211 | `log.Printf("[SSL] DNS 验证成功但保存配置失败 %s: %v", ...)` | `logger.LogPanelError(...)` |
| api_ssl.go | 215 | `log.Printf("[SSL] DNS 验证超时 %s", ...)` | `logger.LogPanelError(...)` |
| api_ssl.go | 300 | `log.Printf("[SSL] 文件验证失败 %s: %v", ...)` | `logger.LogPanelError(...)` |
| api_ssl.go | 309 | `log.Printf("[SSL] 文件验证成功但保存配置失败 %s: %v", ...)` | `logger.LogPanelError(...)` |
| api_ssl.go | 313 | `log.Printf("[SSL] 文件验证超时 %s", ...)` | `logger.LogPanelError(...)` |

### 2.2 site/ 包 — 14 处

| 文件 | 行号 | 当前代码 | 应改为 |
|------|------|----------|--------|
| vhost.go | 64 | `log.Printf("[Vhost] 站点 %s 绑定独立端口 %d", ...)` | `logger.LogSiteInfo(...)` |
| vhost.go | 71 | `log.Printf("[Vhost] 站点 %s 绑定域名 %s", ...)` | `logger.LogSiteInfo(...)` |
| vhost.go | 78 | `log.Printf("[Vhost] 站点 %s 设为默认站点", ...)` | `logger.LogSiteInfo(...)` |
| vhost.go | 258 | `log.Printf("[Vhost] 重载站点 %s 失败: %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 79 | `log.Printf("[Sites] 添加站点失败: %s, %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 106 | `log.Printf("[Sites] 添加站点失败: %s, %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 249 | `log.Printf("[Sites] 添加站点路由失败: %s, %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 271 | `log.Printf("[Sites] 更新站点路由失败: %s, %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 307 | `log.Printf("[Sites] 端口 %d 创建处理器失败: %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 336 | `log.Printf("[Sites] 端口 %d TLS服务错误: %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 341 | `log.Printf("[Sites] 端口 %d 服务错误: %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 391 | `log.Printf("[SSL] 端口 80 服务错误: %v", ...)` | `logger.LogSiteError(...)` |
| server.go | 418 | `log.Printf("[SSL] 端口 80 服务错误: %v", ...)` | `logger.LogSiteError(...)` |
| proxy.go | 40 | `log.Printf("[Proxy] 代理错误: %v, 目标: %s, ...)` | `logger.LogSiteError(...)` |

### 2.3 cmd/ 包 — 2 处

| 文件 | 行号 | 当前代码 | 说明 |
|------|------|----------|------|
| main.go | 42 | `log.Fatalf("加载配置失败: %v", err)` | logger 尚未初始化，可保留或改用 `fmt.Fprintf(os.Stderr, ...)` |
| main.go | 59 | `log.Printf("警告: 初始化日志失败: %v", err)` | logger 初始化失败，可保留 |
| main.go | 122 | `log.Printf("[Admin] 服务错误: %v", err)` | logger 已初始化，应改为 `logger.LogPanelError(...)` |

> **注**: main.go:42/59 在 logger 初始化之前/失败时使用 `log` 包是合理的，可保留。main.go:122 应改为 logger。

### 2.4 不需要修改的（4 处）

- `logger/logger.go` 内部 3 处 `log.Printf` — 日志系统自身的输出，必须用标准 log
- `cmd/main.go:35` `fmt.Printf("v%s...")` — CLI 版本输出，不是日志

### 提分预估

替换 28 处非标准日志后，「错误处理」维度可从 **8.5 → 9.0~9.5**。

---

## 三、错误处理遗漏（影响「错误处理」+「资源管理」）

### 3.1 忽略函数返回 error

| 文件 | 行号 | 代码 | 风险 |
|------|------|------|------|
| monitor/system.go | 38 | `kb, _ := strconv.ParseUint(fields[1], 10, 64)` | 解析失败返回 0，内存统计静默错误 |
| panel/api_system.go | 1374 | `sizeMB, _ = calcDirSize(path)` | 清理计算跳过错误目录 |

### 3.2 exec.Command 错误未检查（17 处）

**api_system.go** — 系统清理/时间设置：

| 行号 | 命令 | 风险 |
|------|------|------|
| 1180 | `exec.Command("sudo", "apt-get", "clean", "-y").Run()` | 低（清理操作） |
| 1182 | `exec.Command("sudo", "yum", "clean", "all", "-y").Run()` | 低 |
| 1184 | `exec.Command("sudo", "dnf", "clean", "all", "-y").Run()` | 低 |
| 1425 | `exec.Command("sudo", "rm", "-rf", item.path).Run()` | 中（路径来源需审计） |
| 1429 | `exec.Command("sudo", "rm", "-f", item.path).Run()` | 中 |
| 1848 | `exec.Command("timedatectl", "set-ntp", "true").Run()` | 低（恢复 NTP） |

**api_service.go** — 服务管理：

| 行号 | 命令 | 风险 |
|------|------|------|
| 231 | `exec.Command("systemctl", "--user", "daemon-reload").Run()` | 中（重载不生效无反馈） |
| 237 | `exec.Command("loginctl", "enable-linger", "").Run()` | 中 |
| 243 | `exec.Command("systemctl", "--user", "disable", ...).Run()` | 中 |
| 244 | `exec.Command("systemctl", "--user", "stop", ...).Run()` | 中 |
| 248 | `exec.Command("systemctl", "--user", "daemon-reload").Run()` | 中 |
| 312 | `exec.Command("launchctl", "unload", ...).Run()` | 中 |
| 338 | `exec.Command("launchctl", "unload", ...).Run()` | 中 |

### 3.3 清理操作错误忽略

| 文件 | 行号 | 代码 |
|------|------|------|
| panel/priv_windows.go | 52-53 | `f.Close()` + `os.Remove(f.Name())` 错误忽略 |
| logger/logger.go | 388 | `os.Remove(filePath)` 错误忽略 |
| panel/api_service.go | 247 | `os.Remove(servicePath)` 错误忽略 |
| panel/api_service.go | 341 | `os.Remove(plistPath)` 错误忽略 |

---

## 四、安全性问题（影响「安全性」+0.5）

### 4.1 已知问题（report-final.md 已列出）

1. **分享链接路径无白名单** — `api_file.go` 中 ShareService 可暴露任意可读文件
2. **canWriteDir Linux 永远返回 true** — `priv_other.go:14`，权限检查失效

### 4.2 新发现

3. **exec.Command 路径注入风险** — `api_system.go:1425/1429` 使用 `sudo rm -rf item.path`，`item.path` 来源需确认是否经过严格校验（虽然清理逻辑中路径是硬编码的，但 `sudo` + `rm -rf` 组合应格外谨慎）

---

## 五、前端问题（影响「前端质量」+0.5）

均在 report-final.md 已列出，无新发现：

1. `escapeHtml` 重复定义（utils.js + template.js）
2. `events.js` `off()` 无法移除 `once()` 监听器
3. `keyboard.js` 无条件 `preventDefault`
4. `api.js` 缺少 `AbortController` 超时
5. `@keyframes` 重复定义（button.css, layout.css, settings.css）

---

## 六、已通过检查项（无问题）

| 检查项 | 结果 |
|--------|------|
| `TODO/FIXME/HACK/XXX` 标记 | 0 处 ✓ |
| 空注释行 `//\s*$` | 0 处 ✓ |
| 未使用 import | 0 处 ✓ |
| 明显死代码 | 0 处 ✓ |
| `panic()` 残留 | 0 处 ✓ |
| main.go 优雅关闭 | 已实现 signal.Notify + SIGINT/SIGTERM ✓ |
| 导出符号 godoc 覆盖 | 88 个中仅 1 个缺失（99%） ✓ |

---

## 七、提分优先级排序

| 优先级 | 操作 | 影响维度 | 预估提分 | 工作量 |
|--------|------|----------|----------|--------|
| **P0** | 替换 28 处非标准日志 → logger | 错误处理 | +0.5~1.0 | 小（机械替换） |
| **P1** | 分享链接路径白名单 | 安全性 | +0.2 | 小 |
| **P2** | site/ 包 logger 日志方法补齐（LogSiteInfo/Error） | 错误处理 | +0.3 | 中（需扩展 logger 包） |
| **P2** | exec.Command 错误检查 + 日志 | 错误处理 | +0.2 | 中 |
| **P3** | monitor ParseUint 错误处理 | 错误处理 | +0.1 | 小 |
| **P3** | 前端 escapeHtml 去重 + off/once 修复 | 前端质量 | +0.3 | 中 |
| **P3** | canWriteDir Linux 实际检查 | 安全性 | +0.1 | 小 |
| **P4** | 测试覆盖扩展至 ssl/site/ftp | 代码质量 | +0.2 | 大 |

---

## 八、总结

**最大提分点是替换 28 处非标准日志**。这是纯机械替换，风险极低，可将「错误处理」从 8.5 提至 9.0+，综合分有望从 9.0 提至 **9.2~9.5**。

排第二的是「分享路径白名单」，这是安全性维度最后一块短板。

前端三个小问题（escapeHtml 重复、off/once 泄漏、键盘劫持）合计影响 -0.3，修复后「前端质量」可从 8.5 → 9.0。
