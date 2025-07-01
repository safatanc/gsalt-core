package middlewares

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
)

// RateLimiter defines the interface for rate limiting implementations
type RateLimiter interface {
	Allow(key string, limit Rate) (bool, RateLimitInfo)
	Reset(key string) error
}

// Rate defines the rate limit configuration
type Rate struct {
	Requests int
	Window   time.Duration
}

// RateLimitInfo contains information about the current rate limit status
type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// RateLimitMiddleware handles rate limiting
type RateLimitMiddleware struct {
	limiter RateLimiter
}

// NewRateLimitMiddleware creates a new RateLimitMiddleware
func NewRateLimitMiddleware(limiter RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// RedisRateLimiter implements RateLimiter using Redis
type RedisRateLimiter struct {
	redis     *redis.Client
	keyPrefix string
}

// NewRedisRateLimiter creates a new RedisRateLimiter
func NewRedisRateLimiter(redis *redis.Client, keyPrefix string) *RedisRateLimiter {
	return &RedisRateLimiter{
		redis:     redis,
		keyPrefix: keyPrefix,
	}
}

// Allow implements RateLimiter.Allow using Redis sorted sets
func (l *RedisRateLimiter) Allow(key string, limit Rate) (bool, RateLimitInfo) {
	ctx := context.Background()
	now := time.Now()
	windowKey := fmt.Sprintf("%s:ratelimit:%s", l.keyPrefix, key)

	// Use pipeline for atomic operations
	pipe := l.redis.Pipeline()

	// Remove old entries outside the window
	windowStart := now.Add(-limit.Window).UnixNano()
	pipe.ZRemRangeByScore(ctx, windowKey, "0", fmt.Sprintf("%d", windowStart))

	// Get current count
	pipe.ZCard(ctx, windowKey)

	// Add current request
	pipe.ZAdd(ctx, windowKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})

	// Set expiry to clean up old keys
	pipe.Expire(ctx, windowKey, limit.Window)

	// Execute pipeline
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		// On error, fail open to allow request but log error
		return true, RateLimitInfo{
			Limit:     limit.Requests,
			Remaining: 0,
			Reset:     now.Add(limit.Window),
		}
	}

	// Get count from pipeline result
	count := cmds[1].(*redis.IntCmd).Val()

	// Calculate remaining and allowed
	remaining := limit.Requests - int(count)
	allowed := remaining >= 0

	return allowed, RateLimitInfo{
		Limit:     limit.Requests,
		Remaining: remaining,
		Reset:     now.Add(limit.Window),
	}
}

// Reset implements RateLimiter.Reset
func (l *RedisRateLimiter) Reset(key string) error {
	ctx := context.Background()
	windowKey := fmt.Sprintf("%s:ratelimit:%s", l.keyPrefix, key)
	return l.redis.Del(ctx, windowKey).Err()
}

// Common rate limits
var (
	PublicAPILimit = Rate{
		Requests: 30,
		Window:   time.Minute,
	}

	AuthenticatedAPILimit = Rate{
		Requests: 60,
		Window:   time.Minute,
	}

	MerchantAPILimit = Rate{
		Requests: 100,
		Window:   time.Minute,
	}

	AuthLimit = Rate{
		Requests: 10,
		Window:   time.Minute,
	}
)

// LimitByIP creates a middleware that rate limits by IP address
func (m *RateLimitMiddleware) LimitByIP(limit Rate) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := fmt.Sprintf("ip:%s", getIPAddress(c))
		return m.handleRateLimit(c, key, limit)
	}
}

// LimitByUser creates a middleware that rate limits by user ID
func (m *RateLimitMiddleware) LimitByUser(limit Rate) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if userID := c.Locals("user_id"); userID != nil {
			key := fmt.Sprintf("user:%v", userID)
			return m.handleRateLimit(c, key, limit)
		}
		return m.LimitByIP(limit)(c)
	}
}

// handleRateLimit handles the rate limiting logic
func (m *RateLimitMiddleware) handleRateLimit(c *fiber.Ctx, key string, limit Rate) error {
	// Check rate limit
	allowed, info := m.limiter.Allow(key, limit)

	// Set rate limit headers
	c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
	c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
	c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.Reset.Unix()))

	if !allowed {
		return pkg.ErrorResponse(c, errors.NewTooManyRequestsError("Rate limit exceeded", info.Limit, info.Reset.Unix()))
	}

	return c.Next()
}

// getIPAddress gets the client IP address from request
func getIPAddress(c *fiber.Ctx) string {
	// Try X-Forwarded-For header
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP header
	if xrip := c.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Fall back to RemoteIP
	return c.IP()
}
