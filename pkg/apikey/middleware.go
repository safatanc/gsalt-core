package apikey

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/safatanc/gsalt-core/pkg/ratelimit"
)

// contextKey is a custom type for context keys
type contextKey string

// Context keys
const (
	APIKeyContextKey   contextKey = "api_key"
	MerchantContextKey contextKey = "merchant_id"
)

// Service interface for API key operations
type Service interface {
	GetAPIKey(ctx context.Context, key string) (*APIKey, error)
	LogAPIKeyUsage(ctx context.Context, usage *APIKeyUsage) error
}

// Middleware creates a new API key middleware
func Middleware(svc Service, limiter ratelimit.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from header
			key := r.Header.Get("X-API-Key")
			if key == "" {
				// No API key, continue without merchant context
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from service
			apiKey, err := svc.GetAPIKey(r.Context(), key)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid API key",
				})
				return
			}

			// Check if API key is active
			if !apiKey.IsActive() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "API key is inactive or expired",
				})
				return
			}

			// Check rate limit
			allowed, info := limiter.Allow(
				"apikey:"+apiKey.ID.String(),
				ratelimit.Rate{
					Requests: apiKey.RateLimit,
					Window:   time.Minute,
				},
			)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", string(info.Limit))
			w.Header().Set("X-RateLimit-Remaining", string(info.Remaining))
			w.Header().Set("X-RateLimit-Reset", string(info.Reset.Unix()))

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

			// Check required scopes
			if !hasRequiredScopes(r, apiKey) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Insufficient permissions",
				})
				return
			}

			// Log API key usage asynchronously
			go func() {
				usage := &APIKeyUsage{
					APIKeyID:   apiKey.ID,
					Endpoint:   r.URL.Path,
					Method:     r.Method,
					IPAddress:  getIPAddress(r),
					UserAgent:  r.UserAgent(),
					StatusCode: http.StatusOK, // Will be updated by responseWriter
					CreatedAt:  time.Now(),
				}
				if err := svc.LogAPIKeyUsage(context.Background(), usage); err != nil {
					// TODO: Add proper error logging
				}
			}()

			// Add API key and merchant ID to context
			ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
			ctx = context.WithValue(ctx, MerchantContextKey, apiKey.MerchantID)

			// Serve with enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// getIPAddress gets the client IP address from request
func getIPAddress(r *http.Request) string {
	// Try X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// hasRequiredScopes checks if the API key has all required scopes for the request
func hasRequiredScopes(r *http.Request, key *APIKey) bool {
	// Define required scopes based on method and path
	var requiredScopes []Scope

	// Add READ scope for GET requests
	if r.Method == http.MethodGet {
		requiredScopes = append(requiredScopes, ScopeRead)
	}

	// Add WRITE scope for POST/PUT/DELETE requests
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
		requiredScopes = append(requiredScopes, ScopeWrite)
	}

	// Add PAYMENT scope for payment endpoints
	if strings.HasPrefix(r.URL.Path, "/api/v1/payments") {
		requiredScopes = append(requiredScopes, ScopePayment)
	}

	// Add WITHDRAWAL scope for withdrawal endpoints
	if strings.HasPrefix(r.URL.Path, "/api/v1/withdrawals") {
		requiredScopes = append(requiredScopes, ScopeWithdrawal)
	}

	// Check if API key has all required scopes
	for _, scope := range requiredScopes {
		if !key.HasScope(scope) {
			return false
		}
	}

	return true
}
