package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements RateLimiter using Redis as storage
type RedisRateLimiter struct {
	redis *redis.Client
	// prefix for redis keys to avoid collisions
	keyPrefix string
}

// NewRedisRateLimiter creates a new RedisRateLimiter
func NewRedisRateLimiter(redis *redis.Client, keyPrefix string) *RedisRateLimiter {
	return &RedisRateLimiter{
		redis:     redis,
		keyPrefix: keyPrefix,
	}
}

// formatKey formats the rate limit key with prefix
func (l *RedisRateLimiter) formatKey(key string) string {
	return fmt.Sprintf("%s:ratelimit:%s", l.keyPrefix, key)
}

// Allow implements RateLimiter.Allow using Redis sorted sets
// It uses a sliding window algorithm
func (l *RedisRateLimiter) Allow(key string, limit Rate) (bool, RateLimitInfo) {
	ctx := context.Background()
	now := time.Now()
	windowKey := l.formatKey(key)

	// Use pipeline for atomic operations
	pipe := l.redis.Pipeline()

	// Remove old entries outside the window
	windowStart := now.Add(-limit.Window).UnixNano()
	pipe.ZRemRangeByScore(ctx, windowKey, "0", strconv.FormatInt(windowStart, 10))

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
		// TODO: Add proper error logging
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
	windowKey := l.formatKey(key)
	return l.redis.Del(ctx, windowKey).Err()
}
