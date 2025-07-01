package ratelimit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// KeyFunc is a function that generates a rate limit key from a request
type KeyFunc func(*http.Request) string

// GetIPKey returns a rate limit key based on IP address
func GetIPKey(r *http.Request) string {
	// Get IP from X-Forwarded-For header or RemoteAddr
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	// If multiple IPs, take the first one
	if idx := strings.Index(ip, ","); idx != -1 {
		ip = ip[:idx]
	}
	// Remove port if present
	if idx := strings.Index(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return fmt.Sprintf("ip:%s", ip)
}

// GetUserKey returns a rate limit key based on user ID from context
func GetUserKey(r *http.Request) string {
	// TODO: Get user ID from your auth context
	userID := r.Context().Value("user_id")
	if userID == nil {
		return ""
	}
	return fmt.Sprintf("user:%s", userID)
}

// GetMerchantKey returns a rate limit key based on merchant ID from context
func GetMerchantKey(r *http.Request) string {
	// TODO: Get merchant ID from your auth context
	merchantID := r.Context().Value("merchant_id")
	if merchantID == nil {
		return ""
	}
	return fmt.Sprintf("merchant:%s", merchantID)
}

// RateLimitMiddleware creates a new rate limiting middleware
func RateLimitMiddleware(limiter RateLimiter, limit Rate, keyFn KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get rate limit key
			key := keyFn(r)
			if key == "" {
				// If no key (e.g., no user ID for user-based limiting),
				// skip rate limiting
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			allowed, info := limiter.Allow(key, limit)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.Reset.Unix()))

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Rate limit exceeded",
					"limit": info.Limit,
					"reset": info.Reset.Unix(),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
