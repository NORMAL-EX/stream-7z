<div align="center">
  <h1>Stream-7z</h1>
  <p>基于 HTTP Range 请求的压缩包流式预览库</p>
  
  [English](./README_en.md) | 简体中文
  
  <img src="https://img.shields.io/github/license/NORMAL-EX/stream-7z" alt="License">
  <img src="https://img.shields.io/github/go-mod/go-version/NORMAL-EX/stream-7z" alt="Go Version">
  <img src="https://img.shields.io/github/v/release/NORMAL-EX/stream-7z" alt="Release">
</div>

## ✨ 特性

- 🚀 **基于 HTTP Range 请求**：无需下载完整压缩包，智能按需获取数据
- 📦 **多格式支持**：支持 ZIP、RAR、7Z、TAR 等主流压缩格式
- 🔐 **完整的密码保护支持**：自动检测加密并验证密码
- 🌐 **提供 HTTP API 服务**：可直接部署为 Web 服务
- 🎯 **流式处理**：低内存占用，适合大文件处理
- ⚡ **高性能**：支持并发请求，性能优化
- 🔧 **灵活配置**：支持自定义 HTTP 头、超时等配置
- 🛡️ **生产就绪**：完善的错误处理和日志记录

## 📖 背景

本项目从 [Alist](https://github.com/alistgo/alist) 提取压缩包预览功能，打造为独立、通用的 Go 库。通过 HTTP Range 请求技术，实现了在不下载完整文件的情况下预览和提取压缩包内容。

## 🚀 快速开始

### 安装

```bash
go get github.com/NORMAL-EX/stream-7z
```

### 作为库使用

```go
package main

import (
    "fmt"
    "log"
    "github.com/NORMAL-EX/stream-7z/lib"
)

func main() {
    // 获取压缩包信息
    archiveURL := "https://example.com/archive.zip"
    info, err := lib.QuickInfo(archiveURL, "", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("文件总数: %d\n", info.TotalFiles)
    fmt.Printf("总大小: %d bytes\n", info.TotalSize)
    
    // 列出文件
    files, err := lib.QuickList(archiveURL, "", "", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, file := range files {
        fmt.Printf("%s (%d bytes)\n", file.Path, file.Size)
    }
    
    // 提取单个文件
    reader, size, err := lib.QuickExtract(archiveURL, "readme.txt", "", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()
    
    // 处理文件内容...
}
```

### 作为 HTTP 服务使用

#### 1. 使用预编译二进制

```bash
# 下载发布版本
# wget https://github.com/NORMAL-EX/stream-7z/releases/latest/download/stream-7z-server-linux-amd64

# 创建配置文件
cat > config.yaml <<EOF
server:
  port: 8080
  auth:
    enabled: true
    header_key: "X-API-Key"
    secret_key: "your-secret-key"
EOF

# 启动服务器
./stream-7z-server -config config.yaml
```

#### 2. 从源码编译

```bash
# 克隆仓库
git clone https://github.com/NORMAL-EX/stream-7z.git
cd stream-7z

# 编译服务器
go build -o stream-7z-server ./cmd/server

# 运行
./stream-7z-server -config config.yaml
```

#### 3. API 调用示例

```bash
# 获取压缩包信息
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/info?url=https://example.com/archive.zip"

# 列出文件
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/list?url=https://example.com/archive.zip"

# 提取文件
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/extract?url=https://example.com/archive.zip&filePath=readme.txt" \
  -o readme.txt

# 处理加密压缩包
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/info?url=https://example.com/encrypted.zip&password=mypassword"
```

## 📚 API 文档

### 库 API

#### 创建 Archive 实例

```go
// 使用默认配置
archive, err := lib.NewArchive(url, nil)

// 使用自定义配置
config := lib.DefaultConfig()
config.WithTimeout(60 * time.Second)
config.WithHeader("Authorization", "Bearer token")
archive, err := lib.NewArchive(url, config)
```

#### 主要方法

```go
// 获取压缩包元数据
info, err := archive.GetInfo(password)

// 列出文件（可指定内部路径）
files, err := archive.ListFiles(innerPath, password)

// 提取单个文件
reader, size, err := archive.ExtractFile(filePath, password)

// 关闭archive
archive.Close()
```

#### 便捷函数

```go
// 快速获取信息（自动创建和关闭）
info, err := lib.QuickInfo(url, password, config)

// 快速列出文件
files, err := lib.QuickList(url, innerPath, password, config)

// 快速提取文件
reader, size, err := lib.QuickExtract(url, filePath, password, config)
```

### HTTP API

所有 API 端点都需要在请求头中包含 `X-API-Key`（如果启用了认证）。

#### GET /api/info

获取压缩包元数据。

**查询参数：**
- `url` (必需): 压缩包 URL
- `password` (可选): 密码

**响应示例：**
```json
{
  "isEncrypted": false,
  "requiresPassword": false,
  "totalFiles": 42,
  "totalSize": 1048576,
  "format": "zip",
  "comment": ""
}
```

#### GET /api/list

列出压缩包中的文件。

**查询参数：**
- `url` (必需): 压缩包 URL
- `password` (可选): 密码
- `innerPath` (可选): 内部路径，留空则列出根目录

**响应示例：**
```json
{
  "files": [
    {
      "path": "folder/file.txt",
      "size": 1024,
      "compressedSize": 512,
      "modTime": "2025-01-01T00:00:00Z",
      "isDir": false
    }
  ]
}
```

#### GET /api/extract

提取压缩包中的单个文件。

**查询参数：**
- `url` (必需): 压缩包 URL
- `filePath` (必需): 文件路径
- `password` (可选): 密码

**响应：** 文件内容（二进制流）

**响应头：**
- `Content-Type: application/octet-stream`
- `Content-Disposition: attachment; filename="filename"`
- `Content-Length: <size>`

#### GET /health

健康检查端点（无需认证）。

**响应示例：**
```json
{
  "status": "ok",
  "time": "2025-01-01T00:00:00Z"
}
```

## 🔧 配置说明

### 服务器配置文件

```yaml
server:
  port: 8080
  auth:
    enabled: true
    header_key: "X-API-Key"
    secret_key: "your-secret-key"
  timeout:
    read: 30s
    write: 30s
  cors:
    enabled: true
    origins:
      - "*"
  rate_limit:
    enabled: true
    requests_per_min: 60
    whitelist:
      - "127.0.0.1"
  max_concurrent: 100

library:
  max_file_size: 524288000  # 500MB
  timeout: 30s
  debug: false
```

### 环境变量

也可以通过环境变量配置：

```bash
export STREAM7Z_SERVER_PORT=8080
export STREAM7Z_SERVER_AUTH_ENABLED=true
export STREAM7Z_SERVER_AUTH_SECRET_KEY=your-secret-key
```

### 库配置选项

```go
config := lib.DefaultConfig()

// 设置超时
config.WithTimeout(60 * time.Second)

// 自定义 HTTP 头
config.WithHeader("User-Agent", "MyApp/1.0")
config.WithHeader("Authorization", "Bearer token")

// 设置最大文件大小（字节）
config.WithMaxFileSize(500 * 1024 * 1024)

// 启用调试日志
config.WithDebug(true)
```

## 📋 支持的格式

| 格式 | 扩展名 | 密码支持 | 说明 |
|------|--------|----------|------|
| ZIP | .zip | ✅ | 支持标准 ZIP 和加密 ZIP |
| RAR | .rar | ✅ | 支持 RAR4 和 RAR5 |
| 7Z | .7z | ✅ | 支持标准 7z 格式 |
| TAR | .tar | ❌ | 未压缩的 TAR |
| TAR+GZIP | .tar.gz, .tgz | ❌ | GZIP 压缩的 TAR |
| TAR+BZIP2 | .tar.bz2, .tbz2 | ❌ | BZIP2 压缩的 TAR |
| TAR+XZ | .tar.xz, .txz | ❌ | XZ 压缩的 TAR |

## 🎮 控制台演示程序

项目包含一个交互式控制台程序用于演示功能：

```bash
# 编译
go build -o demo ./cmd/demo

# 运行
./demo

# 按提示输入压缩包 URL 和密码（如需要）
```

功能：
- 🌳 树形显示文件结构
- 📊 实时进度条
- 🎨 彩色终端输出
- ⌨️ 交互式文件提取
- 🛑 优雅的 Ctrl+C 处理

## 🔐 安全特性

- ✅ 路径遍历防护
- ✅ API 密钥认证
- ✅ 速率限制
- ✅ 并发限制
- ✅ 文件大小限制
- ✅ IP 白名单支持
- ✅ 密码不记录日志

## 🚀 性能优化

- 使用 HTTP Range 请求按需获取数据
- 智能缓存机制
- 连接池复用
- 流式处理避免内存溢出
- 支持并发请求
- Context 超时控制

## 📦 部署

### Docker 部署（推荐）

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o stream-7z-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/stream-7z-server .
COPY cmd/server/config.yaml.example ./config.yaml
EXPOSE 8080
CMD ["./stream-7z-server", "-config", "config.yaml"]
```

构建和运行：

```bash
docker build -t stream-7z .
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml stream-7z
```

### Systemd 服务

```ini
[Unit]
Description=Stream-7z HTTP API Server
After=network.target

[Service]
Type=simple
User=stream7z
WorkingDirectory=/opt/stream-7z
ExecStart=/opt/stream-7z/stream-7z-server -config /opt/stream-7z/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## 🧪 测试

```bash
# 运行所有测试
go test -v ./...

# 运行特定包的测试
go test -v ./lib/...

# 运行测试并显示覆盖率
go test -v -race -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt
```

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

请确保：
- 代码通过 `go fmt` 和 `golint` 检查
- 添加必要的测试
- 更新相关文档

## 📄 开源协议

本项目基于 [MIT License](./LICENSE) 开源。

## 🙏 致谢

- [Alist](https://github.com/alistgo/alist) - 核心实现参考
- [yeka/zip](https://github.com/yeka/zip) - 支持加密 ZIP
- [nwaples/rardecode](https://github.com/nwaples/rardecode) - RAR 解压
- [bodgit/sevenzip](https://github.com/bodgit/sevenzip) - 7z 支持
- [ulikunitz/xz](https://github.com/ulikunitz/xz) - XZ 压缩支持

## 📮 联系方式

- 问题反馈: [GitHub Issues](https://github.com/NORMAL-EX/stream-7z/issues)
- Pull Requests: [GitHub PRs](https://github.com/NORMAL-EX/stream-7z/pulls)

## 🗺️ 路线图

- [ ] 支持更多压缩格式（如 ISO）
- [ ] Web UI 界面
- [ ] 性能监控和统计
- [ ] 分块并发下载优化
- [ ] 缓存层支持（Redis/Memcached）
- [ ] 支持云存储（S3, OSS 等）

---

<div align="center">
Made with ❤️ by NORMAL-EX
</div>
