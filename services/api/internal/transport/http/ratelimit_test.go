package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	transporthttp "github.com/clickTwice26/fatman/services/api/internal/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLimiter struct {
	results []infraredis.RateLimitResult
	idx     int
}

func (s *stubLimiter) Allow(_ context.Context, _, _ string, _ int, _ time.Duration) (infraredis.RateLimitResult, error) {
	res := s.results[s.idx]
	if s.idx < len(s.results)-1 {
		s.idx++
	}
	return res, nil
}

func TestRateLimitMiddleware_SetsHeadersAndBlocksOn429(t *testing.T) {
	limiter := &stubLimiter{
		results: []infraredis.RateLimitResult{
			{Allowed: true, Limit: 5, Remaining: 4, ResetAt: time.Now().Add(time.Minute)},
			{Allowed: false, Limit: 5, Remaining: 0, RetryAfter: 42 * time.Second, ResetAt: time.Now().Add(42 * time.Second)},
		},
	}

	mw := transporthttp.RateLimitMiddleware(limiter, transporthttp.RateLimitPolicy{
		Route:  "otp_request",
		Limit:  5,
		Window: time.Minute,
	}, transporthttp.ClientIPKey)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		transporthttp.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	}))

	// First request allowed
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/request", nil)
	req1.RemoteAddr = "203.0.113.10:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	require.Equal(t, http.StatusOK, rec1.Code)
	assert.Equal(t, "5", rec1.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", rec1.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec1.Header().Get("X-RateLimit-Reset"))

	// Second request blocked with 429
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/request", nil)
	req2.RemoteAddr = "203.0.113.10:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusTooManyRequests, rec2.Code)
	assert.Equal(t, "0", rec2.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "42", rec2.Header().Get("Retry-After"))

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &body))
	assert.Equal(t, "RATE_LIMITED", body.Error.Code)
}
