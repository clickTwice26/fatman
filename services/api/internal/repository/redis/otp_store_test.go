package redis_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	reporedis "github.com/clickTwice26/fatman/services/api/internal/repository/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *infraredis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		password = "fatman_redis_dev_password"
	}
	c, err := infraredis.New(context.Background(), addr, password, 0)
	require.NoError(t, err, "Docker Redis must be running (make infra-up)")
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestOTPStore_IssueVerifySingleUseAndCooldown(t *testing.T) {
	rdb := newTestRedis(t)
	kb := infraredis.NewKeyBuilder("test")
	store := reporedis.NewOTPStore(rdb.Raw, kb, "test-otp-hmac-secret-32-chars-minimum", 2*time.Minute, 2*time.Second, 3)

	ctx := context.Background()
	email := fmt.Sprintf("otp.%s@example.com", uuid.NewString()[:8])

	// Issue OTP
	require.NoError(t, store.IssueEmailOTP(ctx, email, "482910"))

	// Immediate resend hits cooldown
	err := store.IssueEmailOTP(ctx, email, "111111")
	require.ErrorIs(t, err, domain.ErrOTPCooldownActive)

	// Wrong code returns ErrOTPInvalid
	err = store.VerifyEmailOTP(ctx, email, "000000")
	require.ErrorIs(t, err, domain.ErrOTPInvalid)

	// Correct code succeeds (case-insensitive email)
	require.NoError(t, store.VerifyEmailOTP(ctx, "OTP."+email[4:], "482910"))

	// Replay of the same code fails because it was atomically consumed
	err = store.VerifyEmailOTP(ctx, email, "482910")
	require.ErrorIs(t, err, domain.ErrOTPInvalid)
}

func TestOTPStore_LockoutAfterMaxAttempts(t *testing.T) {
	rdb := newTestRedis(t)
	kb := infraredis.NewKeyBuilder("test")
	store := reporedis.NewOTPStore(rdb.Raw, kb, "test-otp-hmac-secret-32-chars-minimum", 2*time.Minute, 100*time.Millisecond, 3)

	ctx := context.Background()
	email := fmt.Sprintf("lockout.%s@example.com", uuid.NewString()[:8])

	require.NoError(t, store.IssueEmailOTP(ctx, email, "991234"))

	require.ErrorIs(t, store.VerifyEmailOTP(ctx, email, "111111"), domain.ErrOTPInvalid)
	require.ErrorIs(t, store.VerifyEmailOTP(ctx, email, "222222"), domain.ErrOTPInvalid)
	// 3rd wrong attempt triggers lockout & deletes OTP
	require.ErrorIs(t, store.VerifyEmailOTP(ctx, email, "333333"), domain.ErrOTPAttemptsExceeded)
	// Even the originally valid code is now rejected
	require.ErrorIs(t, store.VerifyEmailOTP(ctx, email, "991234"), domain.ErrOTPInvalid)
}

func TestOTPStore_UsernameReservationLock(t *testing.T) {
	rdb := newTestRedis(t)
	kb := infraredis.NewKeyBuilder("test")
	store := reporedis.NewOTPStore(rdb.Raw, kb, "test-otp-hmac-secret-32-chars-minimum", 2*time.Minute, time.Second, 5)

	ctx := context.Background()
	handle := "runner_" + uuid.NewString()[:6]

	ok, err := store.AcquireUsernameLock(ctx, handle, "user-1", 5*time.Second)
	require.NoError(t, err)
	assert.True(t, ok)

	// Another user cannot acquire the same username lock
	ok2, err := store.AcquireUsernameLock(ctx, handle, "user-2", 5*time.Second)
	require.NoError(t, err)
	assert.False(t, ok2)

	// Releasing frees it
	require.NoError(t, store.ReleaseUsernameLock(ctx, handle, "user-1"))
	ok3, err := store.AcquireUsernameLock(ctx, handle, "user-2", 5*time.Second)
	require.NoError(t, err)
	assert.True(t, ok3)
}
