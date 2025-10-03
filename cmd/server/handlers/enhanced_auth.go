package handlers

import (
	"net/http"

	"go.uber.org/zap"
)

// EnhancedAuthMiddleware provides API key authentication with multiple keys support
type EnhancedAuthMiddleware struct {
	enabled   bool
	headerKey string
	apiKeys   map[string]bool // Support multiple API keys
	logger    *zap.Logger
}

// NewEnhancedAuthMiddleware creates a new enhanced authentication middleware
// apiKeys: list of valid API keys
func NewEnhancedAuthMiddleware(enabled bool, headerKey string, apiKeys []string, logger *zap.Logger) *EnhancedAuthMiddleware {
	apiKeysMap := make(map[string]bool)
	for _, key := range apiKeys {
		if key != "" {
			apiKeysMap[key] = true
		}
	}

	return &EnhancedAuthMiddleware{
		enabled:   enabled,
		headerKey: headerKey,
		apiKeys:   apiKeysMap,
		logger:    logger,
	}
}

// Handler returns the middleware handler
func (eam *EnhancedAuthMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !eam.enabled {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get(eam.headerKey)
			if apiKey == "" {
				eam.logger.Warn("missing API key",
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("path", r.URL.Path),
				)
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: "Unauthorized: API key is required",
					Code:  "MISSING_API_KEY",
				})
				return
			}

			if !eam.apiKeys[apiKey] {
				eam.logger.Warn("invalid API key",
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("path", r.URL.Path),
				)
				respondJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: "Unauthorized: Invalid API key",
					Code:  "INVALID_API_KEY",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
