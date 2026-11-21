package http

import (
	"context"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
)

// RateLimiter evaluates distributed request quotas.
type RateLimiter interface {
	Allow(ctx context.Context, route, subject string, limit int, window time.Duration) (infraredis.RateLimitResult, error)
}

// RateLimitPolicy configures a route-level rate limit.
type RateLimitPolicy struct {
	Route  string
	Limit  int
	Window time.Duration
}

// KeyFunc extracts the rate-limit subject (e.g. client IP or user ID) from an HTTP request.
type KeyFunc func(r *http.Request) string

// ClientIPKey extracts the best-effort client IP from X-Forwarded-For or RemoteAddr.
func ClientIPKey(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

// RateLimitMiddleware enforces policy via limiter and writes standard rate-limit headers.
func RateLimitMiddleware(limiter RateLimiter, policy RateLimitPolicy, keyFn KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			subject := keyFn(r)
			res, err := limiter.Allow(r.Context(), policy.Route, subject, policy.Limit, policy.Window)
			if err != nil {
				WriteError(w, http.StatusServiceUnavailable, "RATE_LIMITER_UNAVAILABLE", "unable to verify request quota", nil)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(res.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(res.ResetAt.Unix(), 10))

			if !res.Allowed {
				retrySec := int(math.Ceil(res.RetryAfter.Seconds()))
				if retrySec < 1 {
					retrySec = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySec))
				WriteError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests; please wait before trying again", map[string]int{
					"retry_after_seconds": retrySec,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
