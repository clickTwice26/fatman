package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	reporedis "github.com/clickTwice26/fatman/services/api/internal/repository/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStore_CreateRotateReuseDetectionAndDenylist(t *testing.T) {
	rdb := newTestRedis(t)
	kb := infraredis.NewKeyBuilder("test")
	store := reporedis.NewSessionStore(rdb.Raw, kb, "test-session-secret-at-least-32-bytes", 24*time.Hour)

	ctx := context.Background()
	userID := uuid.NewString()

	sess, err := store.CreateSession(ctx, userID, "FatmanTest/1.0", "127.0.0.1")
	require.NoError(t, err)
	assert.NotEmpty(t, sess.SessionID)
	assert.NotEmpty(t, sess.RefreshToken)

	// Rotate valid token -> yields new refresh token
	rotated, err := store.RotateSession(ctx, sess.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, sess.SessionID, rotated.SessionID)
	assert.Equal(t, userID, rotated.UserID)
	assert.NotEqual(t, sess.RefreshToken, rotated.RefreshToken)

	// Replaying the OLD refresh token triggers reuse detection and revokes the entire session
	_, err = store.RotateSession(ctx, sess.RefreshToken)
	require.ErrorIs(t, err, domain.ErrSessionReuseDetected)

	// Even the newer rotated token is now revoked because the family was compromised
	_, err = store.RotateSession(ctx, rotated.RefreshToken)
	require.ErrorIs(t, err, domain.ErrSessionRevoked)

	// Access token JTI denylist
	jti := uuid.NewString()
	denied, err := store.IsAccessTokenDenied(ctx, jti)
	require.NoError(t, err)
	assert.False(t, denied)

	require.NoError(t, store.DenyAccessToken(ctx, jti, 10*time.Second))
	denied, err = store.IsAccessTokenDenied(ctx, jti)
	require.NoError(t, err)
	assert.True(t, denied)
}
