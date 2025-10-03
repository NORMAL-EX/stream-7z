package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NORMAL-EX/stream-7z/cmd/server/handlers"
	"github.com/NORMAL-EX/stream-7z/lib"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config file")
	port := flag.Int("port", 0, "Server port (overrides config)")
	flag.Parse()

	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("Starting Stream-7z HTTP API Server (Enhanced Edition)")

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Override port if specified
	if *port > 0 {
		config.Server.Port = *port
	}

	logger.Info("Configuration loaded",
		zap.Int("port", config.Server.Port),
		zap.Bool("auth_enabled", config.Server.Auth.Enabled),
		zap.Int("api_keys_count", len(config.GetAllAPIKeys())),
		zap.Bool("ip_whitelist_enabled", config.Server.IPWhitelist.Enabled),
		zap.Bool("cors_enabled", config.Server.CORS.Enabled),
		zap.Bool("rate_limit_enabled", config.Server.RateLimit.Enabled),
		zap.Int("max_concurrent", config.Server.MaxConcurrent),
	)

	// Create library config
	libConfig := lib.DefaultConfig().
		WithMaxFileSize(config.Library.MaxFileSize).
		WithTimeout(config.Library.Timeout).
		WithDebug(config.Library.Debug)

	// Create handler
	h := handlers.NewHandler(libConfig, logger)

	// Create rate limiter
	rateLimiter := handlers.NewRateLimiter(
		config.Server.RateLimit.Enabled,
		config.Server.RateLimit.RequestsPerMin,
		config.Server.RateLimit.Whitelist,
		logger,
	)
	defer rateLimiter.Stop()

	// Create IP whitelist middleware
	ipWhitelist := handlers.NewIPWhitelistMiddleware(
		config.Server.IPWhitelist.Enabled,
		config.Server.IPWhitelist.IPs,
		logger,
	)

	// Create enhanced auth middleware with multiple API keys
	enhancedAuth := handlers.NewEnhancedAuthMiddleware(
		config.Server.Auth.Enabled,
		config.Server.Auth.HeaderKey,
		config.GetAllAPIKeys(),
		logger,
	)

	// Setup middleware chain
	middleware := handlers.Chain(
		handlers.RecoveryMiddleware(logger),
		handlers.LoggingMiddleware(logger),
		handlers.RequestIDMiddleware(),
		ipWhitelist.Handler(), // Enhanced: IP whitelist comes first for security
		handlers.NewCORSMiddleware(config.Server.CORS.Enabled, config.Server.CORS.Origins).Handler(),
		rateLimiter.Handler(),
		handlers.ConcurrencyLimitMiddleware(config.Server.MaxConcurrent, logger),
		enhancedAuth.Handler(), // Enhanced: support multiple API keys
	)

	// Setup routes
	mux := http.NewServeMux()

	// Health check (no auth required, but subject to IP whitelist if enabled)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Apply only IP whitelist and logging for health check
		ipWhitelistOnly := handlers.Chain(
			handlers.LoggingMiddleware(logger),
			ipWhitelist.Handler(),
		)
		ipWhitelistOnly(h.Health()).ServeHTTP(w, r)
	})

	// API documentation endpoint (no auth required, but subject to IP whitelist)
	mux.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		ipWhitelistOnly := handlers.Chain(
			handlers.LoggingMiddleware(logger),
			ipWhitelist.Handler(),
		)
		ipWhitelistOnly(serveAPIDocs()).ServeHTTP(w, r)
	})

	// API routes (with full middleware chain)
	mux.Handle("/api/info", middleware(h.Info()))
	mux.Handle("/api/list", middleware(h.List()))
	mux.Handle("/api/extract", middleware(h.Extract()))

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Server.Port),
		Handler:      mux,
		ReadTimeout:  config.Server.Timeout.Read,
		WriteTimeout: config.Server.Timeout.Write,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting",
			zap.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	printStartupBanner(config)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// initLogger initializes the logger
func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.TimeKey = "time"

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	return logger
}

// printStartupBanner prints startup information
func printStartupBanner(config *ServerConfig) {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║       Stream-7z HTTP API Server (Enhanced Edition)       ║
║     HTTP Range-based Archive Preview Service            ║
╚═══════════════════════════════════════════════════════════╝

Server Configuration:
  • Port: %d
  • Authentication: %s
  • API Keys: %d configured
  • IP Whitelist: %s
  • CORS: %s
  • Rate Limiting: %s
  • Max Concurrent: %d

Library Configuration:
  • Max File Size: %s
  • Timeout: %s

API Endpoints:
  • GET  /health             - Health check
  • GET  /api/docs           - API documentation
  • POST /api/info           - Get archive metadata
  • POST /api/list           - List files in archive
  • POST /api/extract        - Extract file from archive

Server is ready to accept requests!
Press Ctrl+C to stop the server.
`

	authStatus := "Disabled"
	if config.Server.Auth.Enabled {
		authStatus = fmt.Sprintf("Enabled (Header: %s)", config.Server.Auth.HeaderKey)
	}

	ipWhitelistStatus := "Disabled"
	if config.Server.IPWhitelist.Enabled {
		ipWhitelistStatus = fmt.Sprintf("Enabled (%d IPs)", len(config.Server.IPWhitelist.IPs))
	}

	corsStatus := "Disabled"
	if config.Server.CORS.Enabled {
		corsStatus = "Enabled"
	}

	rateLimitStatus := "Disabled"
	if config.Server.RateLimit.Enabled {
		rateLimitStatus = fmt.Sprintf("Enabled (%d req/min)", config.Server.RateLimit.RequestsPerMin)
	}

	maxFileSize := fmt.Sprintf("%d MB", config.Library.MaxFileSize/(1024*1024))
	if config.Library.MaxFileSize == 0 {
		maxFileSize = "Unlimited"
	}

	fmt.Printf(banner,
		config.Server.Port,
		authStatus,
		len(config.GetAllAPIKeys()),
		ipWhitelistStatus,
		corsStatus,
		rateLimitStatus,
		config.Server.MaxConcurrent,
		maxFileSize,
		config.Library.Timeout,
	)
}

// serveAPIDocs serves the API documentation
func serveAPIDocs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		
		// Simple redirect to API docs
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Stream-7z API Documentation</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; border-bottom: 3px solid #4CAF50; padding-bottom: 10px; }
        .note { background: #f0f0f0; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Stream-7z API Documentation</h1>
        <div class="note">
            <p><strong>Note:</strong> For complete API documentation, please refer to the <code>API_DOCS.md</code> file included with this server.</p>
            <p>You can also access the OpenAPI specification at <code>/api/openapi.json</code> endpoint.</p>
        </div>
        <p>Quick Links:</p>
        <ul>
            <li><a href="/health">Health Check</a></li>
            <li><a href="/api/openapi.json">OpenAPI Specification (JSON)</a></li>
        </ul>
    </div>
</body>
</html>`)
	}
}
