package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/config"
	"github.com/clickTwice26/fatman/services/api/internal/infrastructure/mailer"
	"github.com/clickTwice26/fatman/services/api/internal/infrastructure/postgres"
	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	repopg "github.com/clickTwice26/fatman/services/api/internal/repository/postgres"
	reporedis "github.com/clickTwice26/fatman/services/api/internal/repository/redis"
	"github.com/clickTwice26/fatman/services/api/internal/service"
	transporthttp "github.com/clickTwice26/fatman/services/api/internal/transport/http"
	"github.com/clickTwice26/fatman/services/api/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pgClient, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer pgClient.Close()

	if err := migrations.Up(ctx, pgClient.Pool); err != nil {
		logger.Error("failed to apply database migrations", slog.Any("error", err))
		os.Exit(1)
	}

	redisClient, err := infraredis.New(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("failed to connect to redis", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Error("failed to close redis client", slog.Any("error", closeErr))
		}
	}()

	keyBuilder := infraredis.NewKeyBuilder(cfg.AppEnv)
	rateLimiter := infraredis.NewSlidingWindowLimiter(redisClient.Raw, keyBuilder)
	userRepo := repopg.NewUserRepository(pgClient.Pool)
	otpStore := reporedis.NewOTPStore(
		redisClient.Raw,
		keyBuilder,
		cfg.OTPHMACSecret,
		cfg.OTPTTL,
		cfg.OTPResendCooldown,
		cfg.OTPMaxAttempts,
	)
	sessionStore := reporedis.NewSessionStore(
		redisClient.Raw,
		keyBuilder,
		cfg.JWTAccessSecret,
		cfg.RefreshTokenTTL,
	)
	tokenSvc := service.NewTokenService(cfg.JWTAccessSecret, cfg.JWTAccessTTL)
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)

	authSvc := service.NewAuthService(service.AuthServiceDeps{
		Users:             userRepo,
		OTPStore:          otpStore,
		SessionStore:      sessionStore,
		Tokens:            tokenSvc,
		Mailer:            smtpMailer,
		OTPTTL:            cfg.OTPTTL,
		OTPResendCooldown: cfg.OTPResendCooldown,
	})

	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Logger:         logger,
		AllowedOrigins: cfg.CORSAllowedOrigins,
		DBChecker:      pgClient,
		RedisChecker:   redisClient,
		RateLimiter:    rateLimiter,
		AuthService:    authSvc,
		TokenService:   tokenSvc,
		Denylist:       sessionStore,
		CookieSecure:   cfg.CookieSecure,
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("fatman api server starting", slog.String("addr", cfg.ListenAddr()), slog.String("env", cfg.AppEnv))
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case serveErr := <-errCh:
		if serveErr != nil {
			logger.Error("server exited with error", slog.Any("error", serveErr))
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("fatman api server stopped cleanly")
}
