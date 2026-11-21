package redis_test

import (
	"context"
	"os"
	"testing"
	"time"

	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisClient(t *testing.T) *infraredis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		password = "fatman_redis_dev_password"
	}
	client, err := infraredis.New(context.Background(), addr, password, 0)
	require.NoError(t, err, "Docker Redis must be running for Redis integration tests (make infra-up)")
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestSlidingWindowLimiter_AllowsUpToLimitAndRecovers(t *testing.T) {
	client := newTestRedisClient(t)
	kb := infraredis.NewKeyBuilder("test")
	limiter := infraredis.NewSlidingWindowLimiter(client.Raw, kb)

	subject := "ip-" + uuid.NewString()
	window := 300 * time.Millisecond
	ctx := context.Background()

	res1, err := limiter.Allow(ctx, "otp_request", subject, 2, window)
	require.NoError(t, err)
	assert.True(t, res1.Allowed)
	assert.Equal(t, 2, res1.Limit)
	assert.Equal(t, 1, res1.Remaining)

	res2, err := limiter.Allow(ctx, "otp_request", subject, 2, window)
	require.NoError(t, err)
	assert.True(t, res2.Allowed)
	assert.Equal(t, 0, res2.Remaining)

	res3, err := limiter.Allow(ctx, "otp_request", subject, 2, window)
	require.NoError(t, err)
	assert.False(t, res3.Allowed)
	assert.Equal(t, 0, res3.Remaining)
	assert.Greater(t, res3.RetryAfter, time.Duration(0))

	// Wait for sliding window to expire and confirm traffic is allowed again.
	time.Sleep(window + 60*time.Millisecond)

	res4, err := limiter.Allow(ctx, "otp_request", subject, 2, window)
	require.NoError(t, err)
	assert.True(t, res4.Allowed)
	assert.Equal(t, 1, res4.Remaining)
}
