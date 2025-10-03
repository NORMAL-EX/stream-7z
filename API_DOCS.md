# Stream-7z API 文档

## 概述

Stream-7z 是一个基于 HTTP Range 请求的压缩包流式预览服务，支持在不下载完整压缩包的情况下，在线预览和提取压缩包内的文件。

**版本:** 2.0 (Enhanced Edition - POST JSON API)  
**基础路径:** `http://your-server:8080`  
**请求方式:** POST (JSON Body)  
**Content-Type:** `application/json`

## 特性

- ✅ 支持多种压缩格式 (ZIP, RAR, 7Z, TAR, TAR.GZ 等)
- ✅ 基于 HTTP Range 请求，按需加载数据
- ✅ 密码保护的压缩包支持
- ✅ API Key 认证 (支持多密钥)
- ✅ IP 白名单访问控制
- ✅ 速率限制防护
- ✅ 并发请求限制
- ✅ CORS 跨域支持

## 认证

所有 API 端点（除了 `/health` 和 `/api/docs`）都需要在请求头中包含有效的 API Key。

### 请求头

```http
X-API-Key: your-api-key-here
```

### 认证错误响应

**401 Unauthorized - 缺少 API Key**
```json
{
  "error": "Unauthorized: API key is required",
  "code": "MISSING_API_KEY"
}
```

**401 Unauthorized - 无效的 API Key**
```json
{
  "error": "Unauthorized: Invalid API key",
  "code": "INVALID_API_KEY"
}
```

## IP 白名单

如果服务器启用了 IP 白名单，只有白名单中的 IP 地址才能访问服务。

### 错误响应

**403 Forbidden - IP 不在白名单**
```json
{
  "error": "Access denied: IP not in whitelist",
  "code": "IP_NOT_WHITELISTED"
}
```

## 速率限制

默认情况下，每个 IP 地址每分钟最多可以发送 60 个请求。

### 错误响应

**429 Too Many Requests**
```json
{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED"
}
```

## API 端点

### 1. 健康检查

检查服务器是否正常运行。

**端点:** `GET /health`  
**认证:** 不需要  
**速率限制:** 不受限制

#### 请求示例

```bash
curl http://localhost:8080/health
```

#### 响应示例

```json
{
  "status": "ok",
  "time": "2025-10-01T12:00:00Z"
}
```

---

### 2. 获取压缩包信息

获取压缩包的元数据信息，包括文件数量、总大小、是否加密等。

**端点:** `POST /api/info`  
**认证:** 需要  
**速率限制:** 受限制  
**Content-Type:** `application/json`

#### 请求体参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| url | string | 是 | 压缩包的完整 URL |
| password | string | 否 | 压缩包密码（如果加密） |

#### 请求示例

```bash
# 无密码压缩包
curl -X POST http://localhost:8080/api/info \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip"
  }'

# 加密压缩包
curl -X POST http://localhost:8080/api/info \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip",
    "password": "mypassword"
  }'
```

#### 响应示例

```json
{
  "isEncrypted": true,
  "requiresPassword": true,
  "totalFiles": 42,
  "totalSize": 104857600,
  "format": "zip",
  "comment": "This is a comment in the archive"
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| isEncrypted | boolean | 压缩包是否加密 |
| requiresPassword | boolean | 是否需要密码才能访问 |
| totalFiles | integer | 压缩包中的文件总数 |
| totalSize | integer | 解压后的总大小（字节） |
| format | string | 压缩包格式 (zip/rar/7z/tar 等) |
| comment | string | 压缩包注释（如果有） |

#### 错误响应

**400 Bad Request - 缺少 URL 参数**
```json
{
  "error": "url is required",
  "code": "MISSING_URL"
}
```

**400 Bad Request - 无效的 JSON**
```json
{
  "error": "Invalid JSON: <error details>",
  "code": "INVALID_JSON"
}
```

**400 Bad Request - Content-Type 错误**
```json
{
  "error": "Content-Type must be application/json",
  "code": "INVALID_CONTENT_TYPE"
}
```

**401 Unauthorized - 密码错误**
```json
{
  "error": "Incorrect password",
  "code": "WRONG_PASSWORD"
}
```

**401 Unauthorized - 需要密码**
```json
{
  "error": "Password required",
  "code": "PASSWORD_REQUIRED"
}
```

**500 Internal Server Error - 无法打开压缩包**
```json
{
  "error": "Failed to get archive info",
  "code": "INTERNAL_ERROR"
}
```

---

### 3. 列出压缩包文件

列出压缩包中的所有文件和目录。

**端点:** `POST /api/list`  
**认证:** 需要  
**速率限制:** 受限制  
**Content-Type:** `application/json`

#### 请求体参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| url | string | 是 | 压缩包的完整 URL |
| password | string | 否 | 压缩包密码（如果加密） |
| innerPath | string | 否 | 内部路径，空字符串列出所有文件，"/"列出根目录第一层 |

#### 请求示例

```bash
# 列出所有文件（递归）
curl -X POST http://localhost:8080/api/list \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip"
  }'

# 列出根目录第一层
curl -X POST http://localhost:8080/api/list \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip",
    "innerPath": "/"
  }'

# 列出特定目录
curl -X POST http://localhost:8080/api/list \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip",
    "innerPath": "docs"
  }'

# 加密压缩包
curl -X POST http://localhost:8080/api/list \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip",
    "password": "mypassword"
  }'
```

#### 响应示例

```json
{
  "files": [
    {
      "path": "README.md",
      "size": 2048,
      "compressedSize": 1024,
      "modTime": "2025-09-15T10:30:00Z",
      "isDir": false
    },
    {
      "path": "docs/",
      "size": 0,
      "compressedSize": 0,
      "modTime": "2025-09-15T10:30:00Z",
      "isDir": true
    },
    {
      "path": "docs/guide.pdf",
      "size": 1048576,
      "compressedSize": 524288,
      "modTime": "2025-09-15T10:30:00Z",
      "isDir": false
    }
  ]
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| files | array | 文件列表 |
| files[].path | string | 文件或目录的路径 |
| files[].size | integer | 文件大小（字节，解压后） |
| files[].compressedSize | integer | 压缩后的大小（字节） |
| files[].modTime | string | 修改时间 (ISO 8601 格式) |
| files[].isDir | boolean | 是否是目录 |

---

### 4. 提取文件

从压缩包中提取单个文件。

**端点:** `POST /api/extract`  
**认证:** 需要  
**速率限制:** 受限制  
**Content-Type:** `application/json`

#### 请求体参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| url | string | 是 | 压缩包的完整 URL |
| file | string | 是 | 要提取的文件路径 |
| password | string | 否 | 压缩包密码（如果加密） |

#### 请求示例

```bash
# 提取文件
curl -X POST http://localhost:8080/api/extract \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip",
    "file": "README.md"
  }' \
  -o README.md

# 提取加密压缩包中的文件
curl -X POST http://localhost:8080/api/extract \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/archive.zip",
    "file": "docs/guide.pdf",
    "password": "mypassword"
  }' \
  -o guide.pdf
```

#### 响应

成功时返回文件的二进制内容。

**响应头:**
```http
Content-Type: application/octet-stream
Content-Disposition: attachment; filename="filename"
Content-Length: <file-size>
```

#### 错误响应

**400 Bad Request - 缺少文件路径**
```json
{
  "error": "file is required",
  "code": "MISSING_FILE"
}
```

**404 Not Found - 文件不存在**
```json
{
  "error": "File not found in archive",
  "code": "FILE_NOT_FOUND"
}
```

---

## 完整使用示例

### Python 示例

```python
import requests

# 配置
API_KEY = "your-api-key-here"
BASE_URL = "http://localhost:8080"
ARCHIVE_URL = "https://example.com/archive.zip"

headers = {
    "X-API-Key": API_KEY,
    "Content-Type": "application/json"
}

# 1. 获取压缩包信息
response = requests.post(
    f"{BASE_URL}/api/info",
    headers=headers,
    json={"url": ARCHIVE_URL}
)
info = response.json()
print(f"压缩包包含 {info['totalFiles']} 个文件")

# 2. 列出文件
response = requests.post(
    f"{BASE_URL}/api/list",
    headers=headers,
    json={"url": ARCHIVE_URL}
)
files = response.json()["files"]
for file in files:
    if not file["isDir"]:
        print(f"{file['path']} - {file['size']} bytes")

# 3. 提取文件
response = requests.post(
    f"{BASE_URL}/api/extract",
    headers=headers,
    json={
        "url": ARCHIVE_URL,
        "file": "README.md"
    }
)
with open("README.md", "wb") as f:
    f.write(response.content)
```

### JavaScript (Node.js) 示例

```javascript
const axios = require('axios');
const fs = require('fs');

const API_KEY = 'your-api-key-here';
const BASE_URL = 'http://localhost:8080';
const ARCHIVE_URL = 'https://example.com/archive.zip';

const headers = {
  'X-API-Key': API_KEY,
  'Content-Type': 'application/json'
};

// 1. 获取压缩包信息
async function getInfo() {
  const response = await axios.post(`${BASE_URL}/api/info`, {
    url: ARCHIVE_URL
  }, { headers });
  console.log(`压缩包包含 ${response.data.totalFiles} 个文件`);
}

// 2. 列出文件
async function listFiles() {
  const response = await axios.post(`${BASE_URL}/api/list`, {
    url: ARCHIVE_URL
  }, { headers });
  response.data.files.forEach(file => {
    if (!file.isDir) {
      console.log(`${file.path} - ${file.size} bytes`);
    }
  });
}

// 3. 提取文件
async function extractFile() {
  const response = await axios.post(`${BASE_URL}/api/extract`, {
    url: ARCHIVE_URL,
    file: 'README.md'
  }, {
    headers,
    responseType: 'arraybuffer'
  });
  fs.writeFileSync('README.md', response.data);
}

// 执行
(async () => {
  await getInfo();
  await listFiles();
  await extractFile();
})();
```

### cURL 完整示例

```bash
#!/bin/bash

API_KEY="your-api-key-here"
BASE_URL="http://localhost:8080"
ARCHIVE_URL="https://example.com/archive.zip"

# 1. 健康检查
echo "=== 健康检查 ==="
curl "$BASE_URL/health"
echo -e "\n"

# 2. 获取压缩包信息
echo "=== 获取压缩包信息 ==="
curl -X POST "$BASE_URL/api/info" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$ARCHIVE_URL\"}"
echo -e "\n"

# 3. 列出文件
echo "=== 列出文件 ==="
curl -X POST "$BASE_URL/api/list" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$ARCHIVE_URL\"}"
echo -e "\n"

# 4. 提取文件
echo "=== 提取文件 ==="
curl -X POST "$BASE_URL/api/extract" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$ARCHIVE_URL\", \"file\": \"README.md\"}" \
  -o README.md
echo "文件已保存为 README.md"
```

## 云盘集成指南

### 场景：云盘压缩包在线预览

当用户在云盘中点击压缩包文件时，可以集成此服务实现在线预览功能。

#### 集成步骤

1. **配置 Stream-7z 服务器**

```yaml
server:
  port: 8080
  auth:
    enabled: true
    api_keys:
      - "cloud-drive-service-key-xyz123"
  ip_whitelist:
    enabled: true
    ips:
      - "192.168.1.100"  # 云盘服务器IP
```

2. **云盘后端调用示例**

```python
# 云盘后端 API
@app.route('/api/preview/archive')
def preview_archive():
    file_id = request.args.get('file_id')
    
    # 获取文件的临时下载URL
    file_url = get_file_download_url(file_id)
    
    # 调用 Stream-7z 服务获取文件列表
    response = requests.post(
        'http://stream-7z-server:8080/api/list',
        headers={
            'X-API-Key': 'cloud-drive-service-key-xyz123',
            'Content-Type': 'application/json'
        },
        json={'url': file_url}
    )
    
    return jsonify(response.json())

@app.route('/api/extract/archive')
def extract_from_archive():
    file_id = request.args.get('file_id')
    file_path = request.args.get('file_path')
    
    file_url = get_file_download_url(file_id)
    
    # 调用 Stream-7z 服务提取文件
    response = requests.post(
        'http://stream-7z-server:8080/api/extract',
        headers={
            'X-API-Key': 'cloud-drive-service-key-xyz123',
            'Content-Type': 'application/json'
        },
        json={
            'url': file_url,
            'file': file_path
        },
        stream=True
    )
    
    return Response(
        response.iter_content(chunk_size=8192),
        content_type=response.headers['Content-Type']
    )
```

3. **前端界面示例**

```javascript
// 云盘前端 - 压缩包预览组件
async function loadArchivePreview(fileId) {
  // 获取压缩包文件列表
  const response = await fetch(`/api/preview/archive?file_id=${fileId}`);
  const data = await response.json();
  
  // 显示文件列表
  const fileList = data.files.map(file => `
    <div class="file-item" onclick="extractFile('${fileId}', '${file.path}')">
      <span class="icon">${file.isDir ? '📁' : '📄'}</span>
      <span class="name">${file.path}</span>
      <span class="size">${formatSize(file.size)}</span>
    </div>
  `).join('');
  
  document.getElementById('archive-content').innerHTML = fileList;
}

async function extractFile(fileId, filePath) {
  // 提取并下载文件
  const url = `/api/extract/archive?file_id=${fileId}&file_path=${encodeURIComponent(filePath)}`;
  window.open(url, '_blank');
}
```

## 错误代码参考

| 错误代码 | HTTP 状态码 | 说明 |
|---------|------------|------|
| MISSING_API_KEY | 401 | 缺少 API Key |
| INVALID_API_KEY | 401 | 无效的 API Key |
| IP_NOT_WHITELISTED | 403 | IP 不在白名单中 |
| RATE_LIMIT_EXCEEDED | 429 | 超过速率限制 |
| TOO_MANY_REQUESTS | 503 | 达到最大并发限制 |
| METHOD_NOT_ALLOWED | 405 | 请求方法不正确（必须使用 POST） |
| INVALID_CONTENT_TYPE | 400 | Content-Type 必须是 application/json |
| INVALID_JSON | 400 | JSON 格式错误 |
| MISSING_URL | 400 | 缺少 URL 参数 |
| MISSING_FILE | 400 | 缺少 file 参数 |
| WRONG_PASSWORD | 401 | 密码错误 |
| PASSWORD_REQUIRED | 401 | 需要密码 |
| FILE_NOT_FOUND | 404 | 文件不存在 |
| PATH_NOT_FOUND | 404 | 路径不存在 |
| UNSUPPORTED_FORMAT | 400 | 不支持的压缩格式 |
| URL_ERROR | 400 | 无法访问 URL |
| INVALID_PATH | 400 | 无效的文件路径 |
| INTERNAL_ERROR | 500 | 内部服务器错误 |

## 性能建议

1. **缓存策略**: 对于频繁访问的压缩包，建议在云盘侧实现缓存
2. **并发控制**: 根据服务器资源调整 `max_concurrent` 参数
3. **文件大小限制**: 设置合理的 `max_file_size` 避免内存溢出
4. **超时设置**: 根据网络状况调整 `timeout` 参数

## 安全建议

1. **使用 HTTPS**: 生产环境建议使用反向代理 (Nginx) 配置 HTTPS
2. **强密钥**: API Key 应使用 32 位以上的随机字符串
3. **密钥轮换**: 定期更换 API Key
4. **IP 白名单**: 生产环境强烈建议启用 IP 白名单
5. **监控日志**: 定期检查访问日志，发现异常及时处理

## 支持的压缩格式

| 格式 | 扩展名 | 密码支持 |
|------|--------|----------|
| ZIP | .zip | ✅ |
| RAR | .rar | ✅ |
| 7-Zip | .7z | ✅ |
| TAR | .tar | ❌ |
| TAR+GZIP | .tar.gz, .tgz | ❌ |
| TAR+BZIP2 | .tar.bz2, .tbz2 | ❌ |
| TAR+XZ | .tar.xz, .txz | ❌ |

## 常见问题

### Q: 如何生成安全的 API Key？

A: 使用以下命令生成随机密钥：

```bash
# Linux/macOS
openssl rand -base64 32

# 或使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

### Q: 如何在 Docker 中运行？

A: 参考项目根目录的 Dockerfile 和 docker-compose.yml

### Q: 支持哪些 HTTP Range 请求？

A: 服务会自动处理 Range 请求，无需客户端特殊处理

### Q: 如何调试问题？

A: 在配置文件中启用 `library.debug: true`，查看详细日志

## 版本历史

### v2.0 (POST JSON API Edition)
- 🔥 **重大变更**：所有 API 端点从 GET 改为 POST，使用 JSON 请求体
- ✨ 新增：更严格的请求验证（Content-Type 检查）
- ✨ 新增：更详细的错误代码和错误信息
- 🐛 修复：innerPath 为空时返回文件列表为空的 bug
- 📚 更新：完整重写 API 文档和示例代码

### v1.0 (Enhanced Edition)
- ✨ 新增：支持多个 API Key
- ✨ 新增：独立的 IP 白名单中间件
- ✨ 新增：CIDR 格式 IP 范围支持
- ✨ 新增：详细的 API 文档
- 🔧 改进：增强的配置文件
- 🔧 改进：更好的错误提示

## 技术支持

- GitHub: https://github.com/NORMAL-EX/stream-7z
- Issues: https://github.com/NORMAL-EX/stream-7z/issues

---

**文档更新时间:** 2025-10-01  
**文档版本:** 2.0 (POST JSON API)  
**API 版本:** 2.0
