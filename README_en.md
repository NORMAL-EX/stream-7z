<div align="center">
  <h1>Stream-7z</h1>
  <p>HTTP Range-based Archive Streaming Library</p>
  
  English | [简体中文](./README.md)
  
  <img src="https://img.shields.io/github/license/NORMAL-EX/stream-7z" alt="License">
  <img src="https://img.shields.io/github/go-mod/go-version/NORMAL-EX/stream-7z" alt="Go Version">
  <img src="https://img.shields.io/github/v/release/NORMAL-EX/stream-7z" alt="Release">
</div>

## ✨ Features

- 🚀 **HTTP Range Requests**: No need to download entire archive, intelligently fetches data on-demand
- 📦 **Multi-format Support**: Supports ZIP, RAR, 7Z, TAR and more
- 🔐 **Password Protection**: Auto-detects encryption and verifies passwords
- 🌐 **HTTP API Service**: Can be deployed as a web service
- 🎯 **Streaming Processing**: Low memory footprint, suitable for large files
- ⚡ **High Performance**: Supports concurrent requests with optimizations
- 🔧 **Flexible Configuration**: Customizable HTTP headers, timeouts, etc.
- 🛡️ **Production Ready**: Complete error handling and logging

## 📖 Background

This project extracts the archive preview functionality from [Alist](https://github.com/alistgo/alist) and turns it into an independent, general-purpose Go library. Using HTTP Range request technology, it enables previewing and extracting archive contents without downloading the entire file.

## 🚀 Quick Start

### Installation

```bash
go get github.com/NORMAL-EX/stream-7z
```

### Using as a Library

```go
package main

import (
    "fmt"
    "log"
    "github.com/NORMAL-EX/stream-7z/lib"
)

func main() {
    // Get archive info
    archiveURL := "https://example.com/archive.zip"
    info, err := lib.QuickInfo(archiveURL, "", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Total Files: %d\n", info.TotalFiles)
    fmt.Printf("Total Size: %d bytes\n", info.TotalSize)
    
    // List files
    files, err := lib.QuickList(archiveURL, "", "", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, file := range files {
        fmt.Printf("%s (%d bytes)\n", file.Path, file.Size)
    }
    
    // Extract a file
    reader, size, err := lib.QuickExtract(archiveURL, "readme.txt", "", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()
    
    // Process file content...
}
```

### Using as HTTP Service

#### 1. Using Pre-compiled Binary

```bash
# Download release
# wget https://github.com/NORMAL-EX/stream-7z/releases/latest/download/stream-7z-server-linux-amd64

# Create config file
cat > config.yaml <<EOF
server:
  port: 8080
  auth:
    enabled: true
    header_key: "X-API-Key"
    secret_key: "your-secret-key"
EOF

# Start server
./stream-7z-server -config config.yaml
```

#### 2. Building from Source

```bash
# Clone repository
git clone https://github.com/NORMAL-EX/stream-7z.git
cd stream-7z

# Build server
go build -o stream-7z-server ./cmd/server

# Run
./stream-7z-server -config config.yaml
```

#### 3. API Usage Examples

```bash
# Get archive info
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/info?url=https://example.com/archive.zip"

# List files
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/list?url=https://example.com/archive.zip"

# Extract file
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/extract?url=https://example.com/archive.zip&filePath=readme.txt" \
  -o readme.txt

# Handle encrypted archives
curl -H "X-API-Key: your-secret-key" \
  "http://localhost:8080/api/info?url=https://example.com/encrypted.zip&password=mypassword"
```

## 📚 API Documentation

### Library API

#### Creating Archive Instance

```go
// Using default config
archive, err := lib.NewArchive(url, nil)

// Using custom config
config := lib.DefaultConfig()
config.WithTimeout(60 * time.Second)
config.WithHeader("Authorization", "Bearer token")
archive, err := lib.NewArchive(url, config)
```

#### Main Methods

```go
// Get archive metadata
info, err := archive.GetInfo(password)

// List files (with optional inner path)
files, err := archive.ListFiles(innerPath, password)

// Extract single file
reader, size, err := archive.ExtractFile(filePath, password)

// Close archive
archive.Close()
```

#### Convenience Functions

```go
// Quick info (auto create and close)
info, err := lib.QuickInfo(url, password, config)

// Quick list
files, err := lib.QuickList(url, innerPath, password, config)

// Quick extract
reader, size, err := lib.QuickExtract(url, filePath, password, config)
```

### HTTP API

All API endpoints require `X-API-Key` header (if authentication is enabled).

#### GET /api/info

Get archive metadata.

**Query Parameters:**
- `url` (required): Archive URL
- `password` (optional): Password

**Response Example:**
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

List files in archive.

**Query Parameters:**
- `url` (required): Archive URL
- `password` (optional): Password
- `innerPath` (optional): Inner path, leave empty for root

**Response Example:**
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

Extract a single file from archive.

**Query Parameters:**
- `url` (required): Archive URL
- `filePath` (required): File path
- `password` (optional): Password

**Response:** File content (binary stream)

**Response Headers:**
- `Content-Type: application/octet-stream`
- `Content-Disposition: attachment; filename="filename"`
- `Content-Length: <size>`

#### GET /health

Health check endpoint (no authentication required).

**Response Example:**
```json
{
  "status": "ok",
  "time": "2025-01-01T00:00:00Z"
}
```

## 🔧 Configuration

### Server Configuration File

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

### Environment Variables

Can also be configured via environment variables:

```bash
export STREAM7Z_SERVER_PORT=8080
export STREAM7Z_SERVER_AUTH_ENABLED=true
export STREAM7Z_SERVER_AUTH_SECRET_KEY=your-secret-key
```

### Library Configuration Options

```go
config := lib.DefaultConfig()

// Set timeout
config.WithTimeout(60 * time.Second)

// Custom HTTP headers
config.WithHeader("User-Agent", "MyApp/1.0")
config.WithHeader("Authorization", "Bearer token")

// Set max file size (bytes)
config.WithMaxFileSize(500 * 1024 * 1024)

// Enable debug logging
config.WithDebug(true)
```

## 📋 Supported Formats

| Format | Extension | Password Support | Notes |
|--------|-----------|------------------|-------|
| ZIP | .zip | ✅ | Standard and encrypted ZIP |
| RAR | .rar | ✅ | RAR4 and RAR5 |
| 7Z | .7z | ✅ | Standard 7z format |
| TAR | .tar | ❌ | Uncompressed TAR |
| TAR+GZIP | .tar.gz, .tgz | ❌ | GZIP compressed TAR |
| TAR+BZIP2 | .tar.bz2, .tbz2 | ❌ | BZIP2 compressed TAR |
| TAR+XZ | .tar.xz, .txz | ❌ | XZ compressed TAR |

## 🎮 Console Demo Program

The project includes an interactive console program to demonstrate features:

```bash
# Build
go build -o demo ./cmd/demo

# Run
./demo

# Follow prompts to enter archive URL and password (if needed)
```

Features:
- 🌳 Tree-style file structure display
- 📊 Real-time progress bar
- 🎨 Colored terminal output
- ⌨️ Interactive file extraction
- 🛑 Graceful Ctrl+C handling

## 🔐 Security Features

- ✅ Path traversal protection
- ✅ API key authentication
- ✅ Rate limiting
- ✅ Concurrency limiting
- ✅ File size limits
- ✅ IP whitelist support
- ✅ Passwords not logged

## 🚀 Performance Optimizations

- HTTP Range requests for on-demand data fetching
- Intelligent caching mechanism
- Connection pool reuse
- Streaming to avoid memory overflow
- Concurrent request support
- Context timeout control

## 📦 Deployment

### Docker Deployment (Recommended)

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

Build and run:

```bash
docker build -t stream-7z .
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml stream-7z
```

### Systemd Service

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

## 🧪 Testing

```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./lib/...

# Run tests with coverage
go test -v -race -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please ensure:
- Code passes `go fmt` and `golint` checks
- Add necessary tests
- Update relevant documentation

## 📄 License

This project is open source under the [MIT License](./LICENSE).

## 🙏 Acknowledgments

- [Alist](https://github.com/alistgo/alist) - Core implementation reference
- [yeka/zip](https://github.com/yeka/zip) - Encrypted ZIP support
- [nwaples/rardecode](https://github.com/nwaples/rardecode) - RAR decompression
- [bodgit/sevenzip](https://github.com/bodgit/sevenzip) - 7z support
- [ulikunitz/xz](https://github.com/ulikunitz/xz) - XZ compression support

## 📮 Contact

- Issue reporting: [GitHub Issues](https://github.com/NORMAL-EX/stream-7z/issues)
- Pull Requests: [GitHub PRs](https://github.com/NORMAL-EX/stream-7z/pulls)

## 🗺️ Roadmap

- [ ] Support more formats (e.g., ISO)
- [ ] Web UI interface
- [ ] Performance monitoring and statistics
- [ ] Chunked concurrent download optimization
- [ ] Cache layer support (Redis/Memcached)
- [ ] Cloud storage support (S3, OSS, etc.)

---

<div align="center">
Made with ❤️ by NORMAL-EX
</div>
