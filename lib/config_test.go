package lib

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if config.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.UserAgent != "Stream-7z/1.0" {
		t.Errorf("Expected UserAgent 'Stream-7z/1.0', got '%s'", config.UserAgent)
	}

	if config.MaxFileSize != 500*1024*1024 {
		t.Errorf("Expected MaxFileSize 500MB, got %d", config.MaxFileSize)
	}

	if config.BufferSize != 32*1024 {
		t.Errorf("Expected BufferSize 32KB, got %d", config.BufferSize)
	}
}

func TestConfigClone(t *testing.T) {
	original := DefaultConfig()
	original.WithHeader("X-Custom", "value")

	clone := original.Clone()

	if clone == nil {
		t.Fatal("Clone returned nil")
	}

	// Verify headers are cloned
	if clone.Headers["X-Custom"] != "value" {
		t.Error("Headers not properly cloned")
	}

	// Modify clone and ensure original is not affected
	clone.WithHeader("X-Custom", "modified")

	if original.Headers["X-Custom"] == "modified" {
		t.Error("Modifying clone affected original")
	}
}

func TestConfigChaining(t *testing.T) {
	config := DefaultConfig().
		WithTimeout(60 * time.Second).
		WithUserAgent("TestAgent").
		WithMaxFileSize(1024).
		WithDebug(true)

	if config.Timeout != 60*time.Second {
		t.Error("Timeout not set correctly")
	}

	if config.UserAgent != "TestAgent" {
		t.Error("UserAgent not set correctly")
	}

	if config.MaxFileSize != 1024 {
		t.Error("MaxFileSize not set correctly")
	}

	if !config.Debug {
		t.Error("Debug not set correctly")
	}
}

func TestConfigHeaders(t *testing.T) {
	config := DefaultConfig()

	// Test WithHeader
	config.WithHeader("Authorization", "Bearer token")

	if config.Headers["Authorization"] != "Bearer token" {
		t.Error("Header not set correctly")
	}

	// Test WithHeaders
	headers := map[string]string{
		"X-Custom-1": "value1",
		"X-Custom-2": "value2",
	}
	config.WithHeaders(headers)

	if config.Headers["X-Custom-1"] != "value1" {
		t.Error("Headers not set correctly")
	}

	if config.Headers["X-Custom-2"] != "value2" {
		t.Error("Headers not set correctly")
	}
}
