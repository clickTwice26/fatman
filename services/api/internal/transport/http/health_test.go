package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	transporthttp "github.com/clickTwice26/fatman/services/api/internal/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChecker struct {
	err error
}

func (s stubChecker) HealthCheck(context.Context) error {
	return s.err
}

func TestHealthz_ReturnsOKAndRequestID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Logger:         logger,
		AllowedOrigins: []string{"http://localhost:3000"},
		DBChecker:      stubChecker{},
		RedisChecker:   stubChecker{},
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))

	var body struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Data.Status)
}

func TestReadyz_ReturnsReadyWhenDependenciesHealthy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Logger:         logger,
		AllowedOrigins: []string{"http://localhost:3000"},
		DBChecker:      stubChecker{},
		RedisChecker:   stubChecker{},
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Status   string `json:"status"`
			Postgres string `json:"postgres"`
			Redis    string `json:"redis"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ready", body.Data.Status)
	assert.Equal(t, "up", body.Data.Postgres)
	assert.Equal(t, "up", body.Data.Redis)
}

func TestReadyz_Returns503WhenRedisDown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Logger:         logger,
		AllowedOrigins: []string{"http://localhost:3000"},
		DBChecker:      stubChecker{},
		RedisChecker:   stubChecker{err: errors.New("connection refused")},
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "SERVICE_UNAVAILABLE", body.Error.Code)
}
