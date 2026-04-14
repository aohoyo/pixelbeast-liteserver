# 第二轮代码自审报告

**审查日期**: 2026-04-14
**审查范围**: `src/panel/*.go`, `src/static/admin/js/core/*.js`, `src/static/admin/css/**/*.css`, `src/config/config.go`

---

## 新发现的问题

### P1 — 需要修复

#### 1. CSRF 中间件逻辑缺陷
- **文件**: `middleware.go:90-97`
- **问题**: 当 CSRF token 不存在时，先检查 token 是否过期再返回"验证失败"。但如果 `!exists`，`csrfToken.ExpiresAt` 是零值，`time.Now().After(csrfToken.ExpiresAt)` 恒为 false，导致"CSRF 验证失败"而非"Token 已过期"。当 token 为空字符串时直接跳到"验证失败"也不够明确。
- **建议**: 重构判断顺序：先检查 `!exists` → "请刷新页面获取 CSRF Token"；再检查 `token == ""` → "缺少 CSRF Token"；最后比较值 → "CSRF 验证失败"。

#### 2. `rand.Read()` 返回值未检查
- **文件**: `handler.go:179`, `handler.go:185`, `api_ssl.go:691`
- **问题**: `crypto/rand.Read(bytes)` 的 error 被忽略。虽然实际中极少失败，但密码学安全场景应检查。
- **建议**: 添加 `if _, err := rand.Read(bytes); err != nil { return "" }` 或使用 `io.ReadFull(rand.Reader, bytes)`。

#### 3. `parseJSONBody` 中 defer Close() 会导致请求体不可重读
- **文件**: `response.go:118-121`
- **问题**: `parseJSONBody` 内部 `defer r.Body.Close()`，但 Go 的 `http.Request.Body` 在 handler 返回后由 `http.Server` 自动关闭。显式 Close 不会造成问题，但如果其他代码再次尝试读取 Body，会失败。更重要的是，这个函数名暗示只是解析 JSON，但副作用是关闭 Body。
- **建议**: 移除 `defer r.Body.Close()`，让 Go 标准库管理 Body 生命周期。

#### 4. init() 中启动的 goroutine 永不退出
- **文件**: `api_system.go:76-165`
- **问题**: `init()` 中启动了两个无限循环 goroutine（CPU 采样和网络/磁盘 IO 采样），使用 `for { ... time.Sleep(...) }` 模式。程序退出时无法优雅停止，但更重要的是：如果在测试中导入此包，goroutine 会持续运行。
- **建议**: 使用 `context.WithCancel` + `select` 模式，或移到 `Handler` 初始化中而非 `init()`。

#### 5. 证书 DNS/文件验证异步 goroutine 无退出机制
- **文件**: `api_ssl.go:181-198`, `api_ssl.go:269-285`
- **问题**: `handleCertDNSComplete` 和 `handleCertFileComplete` 启动 `go func()` 执行验证，但没有超时控制。如果 ACME 验证卡住，goroutine 会永久阻塞。
- **建议**: 使用 `context.WithTimeout`（建议 5 分钟超时），并将进度存储到 map 中供轮询查询。

#### 6. `restartServer` 使用 `os.Exit(0)` 不够安全
- **文件**: `api_system.go:1871-1876`
- **问题**: `os.Exit(0)` 会立即终止进程，不执行 defer，不刷新日志缓冲区。
- **建议**: 通过 channel 通知 main goroutine 优雅关闭，或在 `os.Exit` 前显式 `logger.Flush()` / `http.Server.Shutdown()`。

### P2 — 建议优化

#### 7. `loginAPI` 中 `remaining` 计算可能 panic
- **文件**: `handler.go:458-460`
- **问题**: 当 `checkLoginAttempt` 返回 `false`（被锁定），`h.loginAttempts[clientIP]` 仍存在，不会 panic。但如果 `recordLoginAttempt` 之后再取 `remaining`（先 record 再 RLock），计数已经被重置为 0。实际执行顺序是：先 record(false) → count=5 → 再 RLock 读取 count=5，所以 remaining=0。逻辑正确但依赖隐式顺序。
- **建议**: 在 `recordLoginAttempt` 内部返回 remaining 值，避免多次加锁。

#### 8. 多处 `if h.ConfigManager != nil` 冗余检查
- **文件**: `api_site.go:76-81`, `api_site.go:290-295`, `api_site.go:379-384`
- **问题**: `Handler.ConfigManager` 在 `New()` 中必须初始化，不可能为 nil。多处 `if h.ConfigManager != nil` 检查是冗余的。
- **建议**: 移除这些 nil 检查，保持一致风格。如果确实需要防御，在 `New()` 中 panic。

#### 9. `listFtpUsers` 每次请求都计算所有用户目录大小
- **文件**: `api_ftp.go:405-441`
- **问题**: `calculateDirSize` 使用 `filepath.Walk` 遍历目录统计大小，如果 FTP 用户目录很大（GB 级），每次请求列表都会阻塞数秒。
- **建议**: 缓存目录大小（每 60 秒刷新一次），或改为前端按需查询。

#### 10. `Error(w, http.StatusOK, ...)` 返回 200 但语义是错误
- **文件**: `api_site.go:120-125`, `api_ftp.go:34`, `api_ftp.go:66` 等
- **问题**: 多处使用 `Error(w, http.StatusOK, err.Error())` 返回 HTTP 200 但 body 中包含错误信息。这破坏了 HTTP 语义，前端无法通过 status code 判断成功/失败。
- **建议**: 改为 `Error(w, http.StatusBadRequest, ...)` 或适当的 4xx/5xx 状态码。

#### 11. JS 中 `escapeHtml` 重复定义
- **文件**: `utils.js:14-25`, `template.js:16-25`
- **问题**: `escapeHtml` 在 `utils.js` 和 `template.js` 中各定义了一份，逻辑完全相同。
- **建议**: 统一从 `template.js` 导出，`utils.js` 中 re-export 或删除一份。

#### 12. CSS 中大量硬编码颜色值
- **涉及文件**: 几乎所有 `components/*.css`
- **问题**: 虽然 `base.css` 定义了完整的 CSS 变量体系，但各组件 CSS 中仍有大量硬编码的 hex/rgba 值。最严重的文件：
  - `file-manager.css`: 30+ 处硬编码颜色（包括文件编辑器主题 `#1e1e1e`, `#252526`, `#d4d4d4` 等）
  - `main.css`: 20+ 处图表/状态颜色
  - `button.css`: 按钮 variant 色未使用变量
  - `tooltip.css`: `#303133` 重复 12 次
- **建议**: 将组件常用色补充到 `:root` 变量（如 `--tooltip-bg`, `--editor-bg`, `--success-bg` 等），组件引用变量。

#### 13. `login.css` 重复定义 CSS 变量
- **文件**: `login.css:10-27`
- **问题**: 登录页单独定义了 `:root` 变量，与 `base.css` 中的定义重复。
- **建议**: 登录页也应引用 `base.css`，移除重复变量定义。

#### 14. config.go 中 `defaultServerConfig` 使用 panic 处理错误
- **文件**: `config.go:514`
- **问题**: `crypto.EncryptString` 失败时直接 `panic`，在生产环境中会导致进程崩溃。
- **建议**: 返回 error，让调用方决定如何处理。

#### 15. `resolvePath` 每次调用都 `os.Getwd()`
- **文件**: `api_file.go:25-28`
- **问题**: `resolvePath` 每次调用都执行 `os.Getwd()` 获取当前目录。虽然系统调用开销不大，但文件管理 API 调用频繁。
- **建议**: 缓存 `rootDir`（程序启动后工作目录不会变），使用 `sync.Once` 模式。

---

## 优化建议

### 安全相关

1. **CSRF 中间件接入路由**: `CSRPMiddleware` 已实现但尚未接入路由表（`router.go` 中只用了 `RecoveryMiddleware`）。所有状态修改 API 都缺少 CSRF 保护。
   - **方案**: 在 `setupRouter` 中为 `authMux` 包裹 `CSRPMiddleware`。

2. **Content-Security-Policy 头**: 静态文件和页面响应未设置 CSP 头，虽然 CSP 中间件优先级不高，但建议添加基本的 `default-src 'self'` 策略。

### 架构相关

3. **路由分发使用 Go 1.22+ 的 `{id}` 语法**: `router.go` 中使用了 `mux.HandleFunc("/api/sites/{id}", ...)` 等新路由语法，但 `extractIDFromPath` 仍是手动字符串解析。可以统一用 `r.PathValue("id")` 获取参数。

4. **HTTP 文件管理 vs FTP 文件管理 API 去重**: `api_file.go` 和 `api_ftp.go` 中有大量重复的文件操作逻辑（list、delete、mkdir、rename、copy、download）。可以考虑提取为 `fileHandler` 结构体，通过不同的根目录参数复用。

### 性能相关

5. **系统状态 API 缓存**: `/api/system/status` 每次调用都执行大量系统调用（CPU、内存、磁盘、进程列表）。建议添加短期缓存（2-3 秒），避免前端轮询时对系统造成压力。

6. **`canWriteDir` 硬编码返回 true**: `priv_other.go:14` 中 `canWriteDir` 永远返回 true，没有实际检查写权限。建议使用 `os.OpenFile(dir, os.O_RDWR, 0)` 检测。

---

## 当前评分预估

| 维度 | 得分 | 说明 |
|------|------|------|
| **安全性** | 8/10 | P0 已全修，CSRF 未接入路由是主要扣分项 |
| **错误处理** | 8/10 | 大部分 error 已处理，`rand.Read` 未检查、`panic` 用法是主要问题 |
| **资源管理** | 8/10 | 文件操作有 defer Close，goroutine 缺少退出机制是主要问题 |
| **代码质量** | 8.5/10 | 结构清晰、命名规范，冗余 nil 检查和重复代码是小问题 |
| **前端质量** | 7.5/10 | JS 模块化良好，CSS 变量体系完善但执行不彻底 |
| **综合评分** | **8/10** | P0/P1 全修后基础扎实，CSRF 接入 + goroutine 管理 + CSS 清理可达 9/10 |

### 达到 9/10 的最小行动清单

1. **接入 CSRF 中间件**（P1 安全项）
2. **goroutine 添加 context 超时**（P1 稳定性）
3. **`rand.Read` 检查 error**（P1 正确性）
4. **清理 CSS 硬编码颜色**（P2 一致性）
5. **移除 `Error(w, 200, ...)` 模式**（P2 HTTP 语义）
