package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Middleware function type
type Middleware func(http.Handler) http.Handler

// Chain combines multiple middleware
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// AuthMiddleware provides API key authentication
type AuthMiddleware struct {
	enabled   bool
	headerKey string
	secretKey string
	logger    *zap.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(enabled bool, headerKey, secretKey string, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		enabled:   enabled,
		headerKey: headerKey,
		secretKey: secretKey,
		logger:    logger,
	}
}

// Handler returns the middleware handler
func (am *AuthMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !am.enabled {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get(am.headerKey)
			if apiKey == "" {
				am.logger.Warn("missing API key",
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("path", r.URL.Path),
				)
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: "Unauthorized",
					Code:  "MISSING_API_KEY",
				})
				return
			}

			if apiKey != am.secretKey {
				am.logger.Warn("invalid API key",
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("path", r.URL.Path),
				)
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: "Unauthorized",
					Code:  "INVALID_API_KEY",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
type CORSMiddleware struct {
	enabled bool
	origins []string
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(enabled bool, origins []string) *CORSMiddleware {
	return &CORSMiddleware{
		enabled: enabled,
		origins: origins,
	}
}

// Handler returns the middleware handler
func (cm *CORSMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cm.enabled {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range cm.origins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	enabled        bool
	requestsPerMin int
	whitelist      map[string]bool
	buckets        map[string]*tokenBucket
	mu             sync.RWMutex
	logger         *zap.Logger
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

type tokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(enabled bool, requestsPerMin int, whitelist []string, logger *zap.Logger) *RateLimiter {
	whitelistMap := make(map[string]bool)
	for _, ip := range whitelist {
		whitelistMap[ip] = true
	}

	rl := &RateLimiter{
		enabled:        enabled,
		requestsPerMin: requestsPerMin,
		whitelist:      whitelistMap,
		buckets:        make(map[string]*tokenBucket),
		logger:         logger,
		stopCleanup:    make(chan struct{}),
	}

	// Start cleanup goroutine
	if enabled {
		rl.cleanupTicker = time.NewTicker(5 * time.Minute)
		go rl.cleanupOldBuckets()
	}

	return rl
}

// Handler returns the middleware handler
func (rl *RateLimiter) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Get client IP
			clientIP := getClientIP(r)

			// Check whitelist
			if rl.whitelist[clientIP] {
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			if !rl.allow(clientIP) {
				rl.logger.Warn("rate limit exceeded",
					zap.String("ip", clientIP),
					zap.String("path", r.URL.Path),
				)
				respondJSON(w, http.StatusTooManyRequests, ErrorResponse{
					Error: "Rate limit exceeded",
					Code:  "RATE_LIMIT_EXCEEDED",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if a request should be allowed
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(rl.requestsPerMin),
			lastUpdate: now,
		}
		rl.buckets[ip] = bucket
	}

	// Refill tokens based on time passed
	elapsed := now.Sub(bucket.lastUpdate)
	tokensToAdd := elapsed.Seconds() * float64(rl.requestsPerMin) / 60.0
	bucket.tokens = min(bucket.tokens+tokensToAdd, float64(rl.requestsPerMin))
	bucket.lastUpdate = now

	// Check if we have tokens
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// cleanupOldBuckets removes inactive buckets
func (rl *RateLimiter) cleanupOldBuckets() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, bucket := range rl.buckets {
				if now.Sub(bucket.lastUpdate) > 10*time.Minute {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
		close(rl.stopCleanup)
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", duration),
			)
		})
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
					)

					respondJSON(w, http.StatusInternalServerError, ErrorResponse{
						Error: "Internal server error",
						Code:  "INTERNAL_ERROR",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// ConcurrencyLimitMiddleware limits concurrent requests
func ConcurrencyLimitMiddleware(maxConcurrent int, logger *zap.Logger) Middleware {
	semaphore := make(chan struct{}, maxConcurrent)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				next.ServeHTTP(w, r)
			default:
				logger.Warn("max concurrent requests reached",
					zap.String("remote_addr", r.RemoteAddr),
				)
				respondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
					Error: "Server is busy, please try again later",
					Code:  "TOO_MANY_REQUESTS",
				})
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts the real client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ContextKey is used for context values
type ContextKey string

const (
	// RequestIDKey is the key for request ID in context
	RequestIDKey ContextKey = "request_id"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := fmt.Sprintf("%d", time.Now().UnixNano())
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
