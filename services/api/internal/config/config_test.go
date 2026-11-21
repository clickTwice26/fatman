package config_test

import (
	"testing"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsAndValidation(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("DATABASE_URL", "postgres://fatman:pass@localhost:5432/fatman?sslmode=disable")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("JWT_ACCESS_SECRET", "super-secret-jwt-key-at-least-32-chars")
	t.Setenv("OTP_HMAC_SECRET", "super-secret-otp-key-at-least-32-chars")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "dev", cfg.AppEnv)
	assert.Equal(t, "0.0.0.0:8180", cfg.ListenAddr())
	assert.Equal(t, 15*time.Minute, cfg.JWTAccessTTL)
	assert.Equal(t, 336*time.Hour, cfg.RefreshTokenTTL)
	assert.Equal(t, 10*time.Minute, cfg.OTPTTL)
	assert.Equal(t, 60*time.Second, cfg.OTPResendCooldown)
	assert.Equal(t, 5, cfg.OTPMaxAttempts)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.CORSAllowedOrigins)
}

func TestLoad_RejectsMissingRequiredFields(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_ACCESS_SECRET", "")
	t.Setenv("OTP_HMAC_SECRET", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoad_RejectsShortSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://fatman:pass@localhost:5432/fatman?sslmode=disable")
	t.Setenv("JWT_ACCESS_SECRET", "short")
	t.Setenv("OTP_HMAC_SECRET", "super-secret-otp-key-at-least-32-chars")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_ACCESS_SECRET")
}
