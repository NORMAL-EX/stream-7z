package handlers

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// IPWhitelistMiddleware provides strict IP whitelist checking
type IPWhitelistMiddleware struct {
	enabled   bool
	whitelist map[string]bool
	subnets   []*net.IPNet
	logger    *zap.Logger
}

// NewIPWhitelistMiddleware creates a new IP whitelist middleware
// whitelist can contain individual IPs (e.g., "192.168.1.1") or CIDR ranges (e.g., "192.168.1.0/24")
func NewIPWhitelistMiddleware(enabled bool, whitelist []string, logger *zap.Logger) *IPWhitelistMiddleware {
	whitelistMap := make(map[string]bool)
	var subnets []*net.IPNet

	for _, entry := range whitelist {
		// Check if it's a CIDR range
		if strings.Contains(entry, "/") {
			_, subnet, err := net.ParseCIDR(entry)
			if err != nil {
				logger.Warn("invalid CIDR in whitelist", zap.String("cidr", entry), zap.Error(err))
				continue
			}
			subnets = append(subnets, subnet)
		} else {
			// Individual IP
			whitelistMap[entry] = true
		}
	}

	return &IPWhitelistMiddleware{
		enabled:   enabled,
		whitelist: whitelistMap,
		subnets:   subnets,
		logger:    logger,
	}
}

// Handler returns the middleware handler
func (iw *IPWhitelistMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !iw.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Get client IP
			clientIP := getClientIP(r)
			
			// Check if IP is in whitelist
			if !iw.isIPAllowed(clientIP) {
				iw.logger.Warn("IP not in whitelist",
					zap.String("ip", clientIP),
					zap.String("path", r.URL.Path),
				)
				respondJSON(w, http.StatusForbidden, ErrorResponse{
					Error: "Access denied: IP not in whitelist",
					Code:  "IP_NOT_WHITELISTED",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isIPAllowed checks if an IP is in the whitelist
func (iw *IPWhitelistMiddleware) isIPAllowed(ipStr string) bool {
	// Check individual IPs
	if iw.whitelist[ipStr] {
		return true
	}

	// Check CIDR ranges
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, subnet := range iw.subnets {
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}
