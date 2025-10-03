package lib

import (
	"net/http"
	"time"
)

// Config holds configuration for the archive library
type Config struct {
	// HTTP client configuration
	HTTPClient *http.Client

	// Timeout for HTTP requests
	Timeout time.Duration

	// Custom headers to include in requests
	Headers map[string]string

	// User agent string
	UserAgent string

	// Maximum file size to process (in bytes, 0 = unlimited)
	MaxFileSize int64

	// Buffer size for reading
	BufferSize int

	// Enable debug logging
	Debug bool
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		Timeout:     30 * time.Second,
		Headers:     make(map[string]string),
		UserAgent:   "Stream-7z/1.0",
		MaxFileSize: 0, // 0 = 无限制
		BufferSize:  32 * 1024,          // 32KB buffer
		Debug:       false,
	}
}

// Clone creates a copy of the configuration
func (c *Config) Clone() *Config {
	headers := make(map[string]string)
	for k, v := range c.Headers {
		headers[k] = v
	}

	return &Config{
		HTTPClient:  c.HTTPClient,
		Timeout:     c.Timeout,
		Headers:     headers,
		UserAgent:   c.UserAgent,
		MaxFileSize: c.MaxFileSize,
		BufferSize:  c.BufferSize,
		Debug:       c.Debug,
	}
}

// WithHTTPClient sets a custom HTTP client
func (c *Config) WithHTTPClient(client *http.Client) *Config {
	c.HTTPClient = client
	return c
}

// WithTimeout sets the timeout duration
func (c *Config) WithTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	return c
}

// WithHeaders sets custom headers
func (c *Config) WithHeaders(headers map[string]string) *Config {
	c.Headers = headers
	return c
}

// WithHeader adds a single header
func (c *Config) WithHeader(key, value string) *Config {
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
	c.Headers[key] = value
	return c
}

// WithUserAgent sets the user agent
func (c *Config) WithUserAgent(ua string) *Config {
	c.UserAgent = ua
	return c
}

// WithMaxFileSize sets the maximum file size
func (c *Config) WithMaxFileSize(size int64) *Config {
	c.MaxFileSize = size
	return c
}

// WithDebug enables or disables debug logging
func (c *Config) WithDebug(debug bool) *Config {
	c.Debug = debug
	return c
}
