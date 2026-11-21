package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// slidingWindowScript implements an atomic sliding-window log rate limiter in Redis.
// KEYS[1] = rate limit ZSET key
// ARGV[1] = now (unix ms)
// ARGV[2] = window (ms)
// ARGV[3] = limit (max requests)
// ARGV[4] = unique member id
// Returns: { allowed (0|1), remaining, retry_after_ms }
var slidingWindowScript = goredis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call("ZREMRANGEBYSCORE", key, 0, now - window)
local count = redis.call("ZCARD", key)

if count < limit then
  redis.call("ZADD", key, now, member)
  redis.call("PEXPIRE", key, window)
  local remaining = limit - count - 1
  return {1, remaining, 0}
end

local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
local retry_ms = window
if oldest and #oldest >= 2 then
  local oldest_ts = tonumber(oldest[2])
  retry_ms = math.max(1, (oldest_ts + window) - now)
end
redis.call("PEXPIRE", key, window)
return {0, 0, retry_ms}
`)

// RateLimitResult reports the outcome of a rate-limit evaluation.
type RateLimitResult struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
	ResetAt    time.Time
}

// SlidingWindowLimiter executes distributed sliding-window rate limits in Redis.
type SlidingWindowLimiter struct {
	rdb *goredis.Client
	kb  KeyBuilder
}

// NewSlidingWindowLimiter creates a Redis-backed sliding-window rate limiter.
func NewSlidingWindowLimiter(rdb *goredis.Client, kb KeyBuilder) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{rdb: rdb, kb: kb}
}

// Allow checks whether subject may execute a request on route within limit/window.
func (l *SlidingWindowLimiter) Allow(ctx context.Context, route, subject string, limit int, window time.Duration) (RateLimitResult, error) {
	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := window.Milliseconds()
	if windowMs < 1 {
		windowMs = 1
	}

	key := l.kb.RateLimit(route, subject)
	member := fmt.Sprintf("%d:%s", nowMs, uuid.NewString())

	raw, err := slidingWindowScript.Run(ctx, l.rdb, []string{key}, nowMs, windowMs, limit, member).Slice()
	if err != nil {
		return RateLimitResult{}, fmt.Errorf("run sliding window script: %w", err)
	}
	if len(raw) != 3 {
		return RateLimitResult{}, fmt.Errorf("unexpected rate limit script result length: %d", len(raw))
	}

	allowedVal, ok1 := raw[0].(int64)
	remainingVal, ok2 := raw[1].(int64)
	retryMsVal, ok3 := raw[2].(int64)
	if !ok1 || !ok2 || !ok3 {
		return RateLimitResult{}, fmt.Errorf("unexpected rate limit script result types")
	}

	retryAfter := time.Duration(retryMsVal) * time.Millisecond
	resetAt := now.Add(window)
	if allowedVal == 0 && retryAfter > 0 {
		resetAt = now.Add(retryAfter)
	}

	return RateLimitResult{
		Allowed:    allowedVal == 1,
		Limit:      limit,
		Remaining:  int(remainingVal),
		RetryAfter: retryAfter,
		ResetAt:    resetAt,
	}, nil
}
