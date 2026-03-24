# API 文档

## 基础信息

- **Base URL**: `/admin/api`
- **认证方式**: Session Cookie + CSRF Token
- **响应格式**: JSON

## 响应格式规范

### 统一响应结构

```go
type Response struct {
    Code    int         `json:"code"`     // 响应码
    Message string      `json:"message"`  // 提示信息
    Data    interface{} `json:"data"`     // 数据内容（可选）
}
```

### 响应码定义

| Code | HTTP Status | 说明 |
|------|-------------|------|
| 200  | 200        | 成功 |
| 400  | 400        | 请求参数错误 |
| 401  | 401        | 未认证 |
| 403  | 403        | 禁止访问 |
| 404  | 404        | 资源不存在 |
| 405  | 405        | 方法不允许 |
| 429  | 429        | 请求过多 |
| 500  | 500        | 内部错误 |

### 响应示例

**成功响应（带数据）**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "memory_mb": 12.5,
    "goroutines": 45
  }
}
```

**成功响应（仅消息）**
```json
{
  "code": 200,
  "message": "操作成功"
}
```

**错误响应**
```json
{
  "code": 400,
  "message": "参数错误"
}
```

## API 端点

### 认证相关

无需认证，无需 CSRF Token。

#### 登录
```http
POST /admin/api/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

#### 登出
```http
POST /admin/api/logout
```

### 状态监控

#### 获取服务器状态
```http
GET /admin/api/status
```

**响应数据**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "memory_mb": 12.5,
    "goroutines": 45,
    "uptime": 3600000,
    "os": "linux",
    "arch": "amd64",
    "http_running": true,
    "http_port": 8080,
    "http_root": "./web",
    "ftp_running": true,
    "ftp_port": 2121,
    "ftp_root": "./ftp",
    "sites": []
  }
}
```

### 配置管理

#### 获取配置
```http
GET /admin/api/config
```

#### 保存配置
```http
POST /admin/api/config/save
Content-Type: application/json

{
  "http": {
    "enabled": true,
    "port": 8080,
    "root": "./web",
    "domain": ""
  },
  "ftp": {
    "enabled": true,
    "port": 2121,
    "root": "./ftp",
    "users": [...]
  },
  "admin": {
    "enabled": true,
    "path": "/admin",
    "username": "admin",
    "password": "admin123"
  }
}
```

### 文件管理

#### 列出文件
```http
GET /admin/api/files?path=/path/to/dir
```

#### 上传文件
```http
POST /admin/api/files/upload
Content-Type: multipart/form-data

file: <binary>
path: /target/path
```

#### 删除文件
```http
POST /admin/api/files/delete
Content-Type: application/json

{
  "path": "/path/to/file"
}
```

#### 创建目录
```http
POST /admin/api/files/mkdir
Content-Type: application/json

{
  "path": "/new/dir"
}
```

### FTP 服务管理

#### 启动 FTP
```http
POST /admin/api/service/ftp/start
```

#### 停止 FTP
```http
POST /admin/api/service/ftp/stop
```

#### 重启 FTP
```http
POST /admin/api/service/ftp/restart
```

### FTP 文件管理

#### 列出文件
```http
GET /admin/api/ftp/files?path=/path/to/dir
```

#### 上传文件
```http
POST /admin/api/ftp/files/upload
Content-Type: multipart/form-data

file: <binary>
path: /target/path
```

#### 删除文件
```http
POST /admin/api/ftp/files/delete
Content-Type: application/json

{
  "path": "/path/to/file"
}
```

#### 创建目录
```http
POST /admin/api/ftp/files/mkdir
Content-Type: application/json

{
  "path": "/new/dir"
}
```

### 日志管理

#### 读取日志
```http
GET /admin/api/logs?category=http|ftp|panel&type=access|error|api|auth&lines=100
```

**参数说明**：
- `category`: 日志分类
  - `http` - HTTP 服务日志
  - `ftp` - FTP 服务日志
  - `panel` - 管理面板日志
- `type`: 日志类型
  - `http/ftp`: `access` (访问日志), `error` (错误日志)
  - `panel`: `access` (访问日志), `api` (API调用), `auth` (认证日志)
- `lines`: 返回行数

#### 清空日志
```http
POST /admin/api/logs/clear?category=http|ftp|panel&type=access|error|api|auth
```

## 前端调用示例

### 使用 api.js

```javascript
// 推荐：使用 api.parseJSON() 自动处理统一格式
const response = await api.get('/api/status');
const data = await api.parseJSON(response);
// code=200 时自动返回 data 字段内容
// code!=200 时自动抛出异常

// 一步完成
const data = await api.getJSON('/api/status');

// POST 请求
const response = await api.post('/api/service/ftp/start');
const data = await api.parseJSON(response);
if (data.message) {
    toast.success(data.message);
}

// 错误处理
try {
    const data = await api.getJSON('/api/status');
} catch (error) {
    toast.error(error.message); // 自动从响应中提取 message
}
```

### 使用 XMLHttpRequest（文件上传）

```javascript
const xhr = new XMLHttpRequest();
xhr.upload.addEventListener('progress', (e) => {
    if (e.lengthComputable) {
        const percent = Math.round((e.loaded / e.total) * 100);
        console.log(`上传进度: ${percent}%`);
    }
});
xhr.addEventListener('load', () => {
    if (xhr.status === 200) {
        console.log('上传成功');
    }
});
xhr.open('POST', '/api/files/upload');
xhr.setRequestHeader('X-CSRF-Token', csrfToken);
const formData = new FormData();
formData.append('file', fileInput.files[0]);
formData.append('path', currentPath);
xhr.send(formData);
```

## 后端实现示例

### 成功响应

```go
// 导入 admin 包
import "pixelbeast/src/admin"

// 带数据
admin.Success(w, data)

// 仅消息
admin.SuccessMessage(w, "操作成功")

// 数据 + 消息
admin.SuccessWithData(w, data, "操作成功")
```

### 错误响应

```go
// 通用错误
admin.Error(w, http.StatusBadRequest, "参数错误")

// 快捷方法
admin.BadRequest(w, "参数错误")           // 400
admin.Unauthorized(w, "未登录")            // 401
admin.Forbidden(w, "禁止访问")             // 403
admin.NotFound(w, "资源不存在")            // 404
admin.MethodNotAllowed(w, "方法不允许")    // 405
admin.TooManyRequests(w, "请求过多")       // 429
admin.InternalServerError(w, "内部错误")   // 500
```
