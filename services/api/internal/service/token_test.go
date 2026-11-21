package service_test

import (
	"testing"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	"github.com/clickTwice26/fatman/services/api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenService_IssueAndVerifyAccessToken(t *testing.T) {
	svc := service.NewTokenService("jwt-access-secret-at-least-32-characters-long", 15*time.Minute)

	signed, claims, err := svc.IssueAccessToken("user-42", "sess-99")
	require.NoError(t, err)
	assert.NotEmpty(t, signed)
	assert.Equal(t, "user-42", claims.UserID)
	assert.Equal(t, "sess-99", claims.SessionID)
	assert.NotEmpty(t, claims.JTI)

	parsed, err := svc.ParseAccessToken(signed)
	require.NoError(t, err)
	assert.Equal(t, claims.UserID, parsed.UserID)
	assert.Equal(t, claims.SessionID, parsed.SessionID)
	assert.Equal(t, claims.JTI, parsed.JTI)

	// Tampered token is rejected
	_, err = svc.ParseAccessToken(signed + "tamper")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}
