package main

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig holds the server configuration
type ServerConfig struct {
	Server  ServerSettings  `mapstructure:"server"`
	Library LibrarySettings `mapstructure:"library"`
}

// ServerSettings contains HTTP server settings
type ServerSettings struct {
	Port          int             `mapstructure:"port"`
	Auth          AuthSettings    `mapstructure:"auth"`
	Timeout       TimeoutConfig   `mapstructure:"timeout"`
	CORS          CORSConfig      `mapstructure:"cors"`
	RateLimit     RateLimitConfig `mapstructure:"rate_limit"`
	IPWhitelist   IPWhitelistConfig `mapstructure:"ip_whitelist"` // Enhanced IP whitelist
	MaxConcurrent int             `mapstructure:"max_concurrent"`
}

// AuthSettings contains authentication settings
type AuthSettings struct {
	Enabled   bool     `mapstructure:"enabled"`
	HeaderKey string   `mapstructure:"header_key"`
	SecretKey string   `mapstructure:"secret_key"` // Kept for backward compatibility
	APIKeys   []string `mapstructure:"api_keys"`   // Enhanced: support multiple API keys
}

// TimeoutConfig contains timeout settings
type TimeoutConfig struct {
	Read  time.Duration `mapstructure:"read"`
	Write time.Duration `mapstructure:"write"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Origins []string `mapstructure:"origins"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	RequestsPerMin int      `mapstructure:"requests_per_min"`
	Whitelist      []string `mapstructure:"whitelist"`
}

// IPWhitelistConfig contains IP whitelist settings
type IPWhitelistConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	IPs     []string `mapstructure:"ips"` // Supports individual IPs and CIDR ranges
}

// LibrarySettings contains settings for the archive library
type LibrarySettings struct {
	MaxFileSize int64         `mapstructure:"max_file_size"`
	Timeout     time.Duration `mapstructure:"timeout"`
	Debug       bool          `mapstructure:"debug"`
}

// LoadConfig loads configuration from file or environment variables
func LoadConfig(configPath string) (*ServerConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.auth.enabled", true)
	v.SetDefault("server.auth.header_key", "X-API-Key")
	v.SetDefault("server.auth.secret_key", "")
	v.SetDefault("server.auth.api_keys", []string{})
	v.SetDefault("server.timeout.read", 30*time.Second)
	v.SetDefault("server.timeout.write", 30*time.Second)
	v.SetDefault("server.cors.enabled", true)
	v.SetDefault("server.cors.origins", []string{"*"})
	v.SetDefault("server.rate_limit.enabled", true)
	v.SetDefault("server.rate_limit.requests_per_min", 60)
	v.SetDefault("server.rate_limit.whitelist", []string{})
	v.SetDefault("server.ip_whitelist.enabled", false)
	v.SetDefault("server.ip_whitelist.ips", []string{})
	v.SetDefault("server.max_concurrent", 100)
	v.SetDefault("library.max_file_size", 500*1024*1024) // 500MB
	v.SetDefault("library.timeout", 30*time.Second)
	v.SetDefault("library.debug", false)

	// Read from config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Read from environment variables
	v.SetEnvPrefix("STREAM7Z")
	v.AutomaticEnv()

	// Unmarshal config
	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate validates the configuration
func (c *ServerConfig) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Server.Port)
	}

	if c.Server.Auth.Enabled {
		// Check if we have any valid API keys
		hasValidKey := false
		if c.Server.Auth.SecretKey != "" {
			hasValidKey = true
		}
		if len(c.Server.Auth.APIKeys) > 0 {
			hasValidKey = true
		}
		if !hasValidKey {
			return fmt.Errorf("auth is enabled but no API keys are configured (set secret_key or api_keys)")
		}
	}

	if c.Server.MaxConcurrent < 1 {
		return fmt.Errorf("max_concurrent must be at least 1")
	}

	if c.Library.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size cannot be negative")
	}

	if c.Server.IPWhitelist.Enabled && len(c.Server.IPWhitelist.IPs) == 0 {
		return fmt.Errorf("ip_whitelist is enabled but no IPs are configured")
	}

	return nil
}

// GetAllAPIKeys returns all configured API keys (including legacy secret_key)
func (c *ServerConfig) GetAllAPIKeys() []string {
	keys := make([]string, 0)
	
	// Add legacy secret_key if present
	if c.Server.Auth.SecretKey != "" {
		keys = append(keys, c.Server.Auth.SecretKey)
	}
	
	// Add new api_keys
	keys = append(keys, c.Server.Auth.APIKeys...)
	
	return keys
}

// Example config file content
const ExampleConfig = `# Stream-7z Server Configuration
# 压缩包流式预览服务器配置文件

server:
  # 服务器端口 / Server port
  port: 8080
  
  # 认证配置 / Authentication configuration
  auth:
    # 是否启用认证 / Enable authentication
    enabled: true
    # API Key 请求头名称 / API Key header name
    header_key: "X-API-Key"
    # 单个密钥（向后兼容）/ Single key (backward compatible)
    secret_key: "your-secret-key-here"
    # 多个API密钥（推荐）/ Multiple API keys (recommended)
    api_keys:
      - "api-key-for-app1"
      - "api-key-for-app2"
      - "api-key-for-user1"
  
  # 超时配置 / Timeout configuration
  timeout:
    read: 30s   # 读取超时 / Read timeout
    write: 30s  # 写入超时 / Write timeout
  
  # CORS 跨域配置 / CORS configuration
  cors:
    enabled: true
    origins:
      - "*"
      # - "https://your-domain.com"
  
  # 速率限制配置 / Rate limiting configuration
  rate_limit:
    # 是否启用速率限制 / Enable rate limiting
    enabled: true
    # 每分钟请求数 / Requests per minute
    requests_per_min: 60
    # 白名单IP（不受速率限制）/ Whitelist IPs (bypass rate limit)
    whitelist:
      - "127.0.0.1"
      - "::1"
  
  # IP白名单配置（严格访问控制）/ IP whitelist configuration (strict access control)
  ip_whitelist:
    # 是否启用IP白名单 / Enable IP whitelist
    enabled: false
    # 允许访问的IP列表（支持单个IP和CIDR格式）/ Allowed IPs (supports individual IPs and CIDR)
    ips:
      - "192.168.1.100"           # 单个IP / Single IP
      - "192.168.1.0/24"          # CIDR范围 / CIDR range
      - "10.0.0.0/8"              # 内网段 / Private network
      - "2001:db8::/32"           # IPv6 CIDR
  
  # 最大并发请求数 / Maximum concurrent requests
  max_concurrent: 100

# 压缩包库配置 / Archive library configuration
library:
  # 最大文件大小（字节）/ Maximum file size (bytes)
  max_file_size: 524288000  # 500MB
  # 操作超时 / Operation timeout
  timeout: 30s
  # 调试模式 / Debug mode
  debug: false
`
