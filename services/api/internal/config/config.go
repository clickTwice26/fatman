package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds validated runtime configuration for the Fatman API service.
type Config struct {
	AppEnv             string
	APIHost            string
	APIPort            string
	CORSAllowedOrigins []string

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	SMTPHost string
	SMTPPort int
	SMTPFrom string

	JWTAccessSecret   string
	JWTAccessTTL      time.Duration
	RefreshTokenTTL   time.Duration
	OTPHMACSecret     string
	OTPTTL            time.Duration
	OTPResendCooldown time.Duration
	OTPMaxAttempts    int
	CookieSecure      bool
}

// ListenAddr returns the host:port address for the HTTP server.
func (c Config) ListenAddr() string {
	return net.JoinHostPort(c.APIHost, c.APIPort)
}

// Load reads configuration from environment variables and validates required fields.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:             envOrDefault("APP_ENV", "dev"),
		APIHost:            envOrDefault("API_HOST", "0.0.0.0"),
		APIPort:            envOrDefault("API_PORT", "8180"),
		CORSAllowedOrigins: splitCSV(envOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisAddr:          envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		SMTPHost:           envOrDefault("SMTP_HOST", "localhost"),
		SMTPFrom:           envOrDefault("SMTP_FROM", "no-reply@fatman.local"),
		JWTAccessSecret:    strings.TrimSpace(os.Getenv("JWT_ACCESS_SECRET")),
		OTPHMACSecret:      strings.TrimSpace(os.Getenv("OTP_HMAC_SECRET")),
	}

	var err error
	if cfg.RedisDB, err = envInt("REDIS_DB", 0); err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	if cfg.SMTPPort, err = envInt("SMTP_PORT", 1025); err != nil {
		return Config{}, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}
	if cfg.JWTAccessTTL, err = envDuration("JWT_ACCESS_TTL", 15*time.Minute); err != nil {
		return Config{}, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	if cfg.RefreshTokenTTL, err = envDuration("REFRESH_TOKEN_TTL", 336*time.Hour); err != nil {
		return Config{}, fmt.Errorf("invalid REFRESH_TOKEN_TTL: %w", err)
	}
	if cfg.OTPTTL, err = envDuration("OTP_TTL", 10*time.Minute); err != nil {
		return Config{}, fmt.Errorf("invalid OTP_TTL: %w", err)
	}
	if cfg.OTPResendCooldown, err = envDuration("OTP_RESEND_COOLDOWN", 60*time.Second); err != nil {
		return Config{}, fmt.Errorf("invalid OTP_RESEND_COOLDOWN: %w", err)
	}
	if cfg.OTPMaxAttempts, err = envInt("OTP_MAX_ATTEMPTS", 5); err != nil {
		return Config{}, fmt.Errorf("invalid OTP_MAX_ATTEMPTS: %w", err)
	}
	if cfg.CookieSecure, err = envBool("COOKIE_SECURE", cfg.AppEnv == "prod"); err != nil {
		return Config{}, fmt.Errorf("invalid COOKIE_SECURE: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks mandatory fields and security minimums.
func (c Config) Validate() error {
	var errs []error
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.RedisAddr == "" {
		errs = append(errs, errors.New("REDIS_ADDR is required"))
	}
	if len(c.JWTAccessSecret) < 32 {
		errs = append(errs, errors.New("JWT_ACCESS_SECRET must be at least 32 characters"))
	}
	if len(c.OTPHMACSecret) < 32 {
		errs = append(errs, errors.New("OTP_HMAC_SECRET must be at least 32 characters"))
	}
	if c.OTPMaxAttempts < 1 {
		errs = append(errs, errors.New("OTP_MAX_ATTEMPTS must be at least 1"))
	}
	return errors.Join(errs...)
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	return strconv.Atoi(v)
}

func envBool(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	return strconv.ParseBool(v)
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	return time.ParseDuration(v)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
