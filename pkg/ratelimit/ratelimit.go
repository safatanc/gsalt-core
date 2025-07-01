package ratelimit

import (
	"time"
)

// Rate defines the rate limit configuration
type Rate struct {
	// Requests is the number of requests allowed in the window
	Requests int
	// Window is the time window for the rate limit
	Window time.Duration
}

// RateLimitInfo contains information about the current rate limit status
type RateLimitInfo struct {
	// Limit is the total number of requests allowed
	Limit int
	// Remaining is the number of requests remaining
	Remaining int
	// Reset is when the rate limit will reset
	Reset time.Time
}

// RateLimiter defines the interface for rate limiting implementations
type RateLimiter interface {
	// Allow checks if a request is allowed and returns rate limit info
	Allow(key string, limit Rate) (bool, RateLimitInfo)
	// Reset resets the rate limit for a key
	Reset(key string) error
}

// Common rate limits
var (
	// PublicAPILimit is for public API endpoints (30 req/min)
	PublicAPILimit = Rate{
		Requests: 30,
		Window:   time.Minute,
	}

	// AuthenticatedAPILimit is for authenticated user endpoints (60 req/min)
	AuthenticatedAPILimit = Rate{
		Requests: 60,
		Window:   time.Minute,
	}

	// MerchantAPILimit is for merchant endpoints (100 req/min)
	MerchantAPILimit = Rate{
		Requests: 100,
		Window:   time.Minute,
	}

	// AuthLimit is for authentication endpoints (10 req/min)
	AuthLimit = Rate{
		Requests: 10,
		Window:   time.Minute,
	}
)
