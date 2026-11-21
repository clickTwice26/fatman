package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/service"
	"github.com/go-chi/chi/v5"
)

// HealthChecker defines a readiness dependency probe.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// RouterDeps holds all dependencies required to construct the HTTP router.
type RouterDeps struct {
	Logger         *slog.Logger
	AllowedOrigins []string
	DBChecker      HealthChecker
	RedisChecker   HealthChecker
	RateLimiter    RateLimiter
	AuthService    *service.AuthService
	TokenService   *service.TokenService
	Denylist       AccessTokenDenylist
	CookieSecure   bool
}

// NewRouter constructs the root Chi HTTP router with core middleware, health routes, and API v1 routes.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(RequestIDMiddleware)
	r.Use(RecoveryMiddleware(deps.Logger))
	r.Use(LoggingMiddleware(deps.Logger))
	r.Use(CORSMiddleware(deps.AllowedOrigins))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		pgStatus := "up"
		redisStatus := "up"
		healthy := true

		if deps.DBChecker == nil || deps.DBChecker.HealthCheck(ctx) != nil {
			pgStatus = "down"
			healthy = false
		}
		if deps.RedisChecker == nil || deps.RedisChecker.HealthCheck(ctx) != nil {
			redisStatus = "down"
			healthy = false
		}

		if !healthy {
			WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "one or more dependencies are unavailable", map[string]string{
				"postgres": pgStatus,
				"redis":    redisStatus,
			})
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{
			"status":   "ready",
			"postgres": pgStatus,
			"redis":    redisStatus,
		})
	})

	if deps.AuthService != nil && deps.TokenService != nil {
		authHandler := NewAuthHandler(deps.AuthService, deps.CookieSecure)
		authMW := AuthMiddleware(deps.TokenService, deps.Denylist)

		otpRequestRL := RateLimitMiddleware(deps.RateLimiter, RateLimitPolicy{
			Route:  "otp_request",
			Limit:  5,
			Window: 15 * time.Minute,
		}, ClientIPKey)

		otpVerifyRL := RateLimitMiddleware(deps.RateLimiter, RateLimitPolicy{
			Route:  "otp_verify",
			Limit:  10,
			Window: 15 * time.Minute,
		}, ClientIPKey)

		r.Route("/api/v1", func(v1 chi.Router) {
			v1.Route("/auth", func(auth chi.Router) {
				auth.With(otpRequestRL).Post("/otp/request", authHandler.RequestOTP)
				auth.With(otpVerifyRL).Post("/otp/verify", authHandler.VerifyOTP)
				auth.Post("/refresh", authHandler.Refresh)
				auth.With(authMW).Post("/logout", authHandler.Logout)
				auth.With(authMW).Get("/me", authHandler.Me)
			})

			v1.Route("/onboarding", func(onb chi.Router) {
				onb.Use(authMW)
				onb.Get("/username-available", authHandler.CheckUsernameAvailable)
				onb.Post("/complete", authHandler.CompleteOnboarding)
			})
		})
	}

	return r
}
