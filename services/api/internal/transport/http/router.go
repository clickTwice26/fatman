package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

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
}

// NewRouter constructs the root Chi HTTP router with core middleware and health routes.
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

	return r
}
