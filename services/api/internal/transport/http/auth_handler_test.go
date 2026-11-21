package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	infrapg "github.com/clickTwice26/fatman/services/api/internal/infrastructure/postgres"
	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	repopg "github.com/clickTwice26/fatman/services/api/internal/repository/postgres"
	reporedis "github.com/clickTwice26/fatman/services/api/internal/repository/redis"
	"github.com/clickTwice26/fatman/services/api/internal/service"
	transporthttp "github.com/clickTwice26/fatman/services/api/internal/transport/http"
	"github.com/clickTwice26/fatman/services/api/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureMailer struct {
	mu    sync.Mutex
	codes map[string]string
}

func newCaptureMailer() *captureMailer {
	return &captureMailer{codes: make(map[string]string)}
}

func (m *captureMailer) SendOTP(_ context.Context, toEmail, code string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[toEmail] = code
	return nil
}

func (m *captureMailer) LastCode(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codes[email]
}

func TestEmailOTPOnboardingLifecycle_EndToEnd(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://fatman:fatman_dev_password@localhost:5432/fatman?sslmode=disable"
	}
	pg, err := infrapg.New(ctx, dbURL)
	require.NoError(t, err)
	t.Cleanup(pg.Close)
	require.NoError(t, migrations.Up(ctx, pg.Pool))

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPass := os.Getenv("REDIS_PASSWORD")
	if redisPass == "" {
		redisPass = "fatman_redis_dev_password"
	}
	rdb, err := infraredis.New(ctx, redisAddr, redisPass, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	kb := infraredis.NewKeyBuilder("test")
	userRepo := repopg.NewUserRepository(pg.Pool)
	otpStore := reporedis.NewOTPStore(rdb.Raw, kb, "test-otp-hmac-secret-32-chars-minimum", 10*time.Minute, 60*time.Second, 5)
	sessionStore := reporedis.NewSessionStore(rdb.Raw, kb, "test-session-secret-32-chars-minimum", 24*time.Hour)
	tokenSvc := service.NewTokenService("test-jwt-secret-32-chars-minimum-len", 15*time.Minute)
	mailSpy := newCaptureMailer()
	limiter := infraredis.NewSlidingWindowLimiter(rdb.Raw, kb)

	authSvc := service.NewAuthService(service.AuthServiceDeps{
		Users:             userRepo,
		OTPStore:          otpStore,
		SessionStore:      sessionStore,
		Tokens:            tokenSvc,
		Mailer:            mailSpy,
		OTPTTL:            10 * time.Minute,
		OTPResendCooldown: 60 * time.Second,
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Logger:         logger,
		AllowedOrigins: []string{"http://localhost:3000"},
		DBChecker:      pg,
		RedisChecker:   rdb,
		RateLimiter:    limiter,
		AuthService:    authSvc,
		TokenService:   tokenSvc,
		Denylist:       sessionStore,
		CookieSecure:   false,
	})

	email := fmt.Sprintf("journey.%s@example.com", uuid.NewString()[:8])
	clientIP := "198.51.100." + fmt.Sprintf("%d", time.Now().UnixNano()%200+1)

	// 1. Request Email OTP
	reqOTP := func(body any) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/request", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP + ":4321"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec1 := reqOTP(map[string]string{"email": email})
	require.Equal(t, http.StatusAccepted, rec1.Code)
	code := mailSpy.LastCode(email)
	require.Len(t, code, 6)

	// 2. Immediate resend hits cooldown
	recCooldown := reqOTP(map[string]string{"email": email})
	require.Equal(t, http.StatusTooManyRequests, recCooldown.Code)

	// 3. Wrong OTP returns 401
	verifyOTP := func(submitCode string) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(map[string]string{"email": email, "code": submitCode})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/verify", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP + ":4321"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	recWrong := verifyOTP("000000")
	require.Equal(t, http.StatusUnauthorized, recWrong.Code)

	// 4. Correct OTP creates user, sets HttpOnly refresh cookie, returns JWT + onboarding_completed=false
	recVerify := verifyOTP(code)
	require.Equal(t, http.StatusOK, recVerify.Code)

	var verifyResp struct {
		Data struct {
			AccessToken         string `json:"access_token"`
			IsNewUser           bool   `json:"is_new_user"`
			OnboardingCompleted bool   `json:"onboarding_completed"`
			User                struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recVerify.Body.Bytes(), &verifyResp))
	assert.NotEmpty(t, verifyResp.Data.AccessToken)
	assert.True(t, verifyResp.Data.IsNewUser)
	assert.False(t, verifyResp.Data.OnboardingCompleted)
	assert.Equal(t, email, verifyResp.Data.User.Email)

	var refreshCookie *http.Cookie
	for _, c := range recVerify.Result().Cookies() {
		if c.Name == "fatman_refresh" {
			refreshCookie = c
		}
	}
	require.NotNil(t, refreshCookie)
	assert.True(t, refreshCookie.HttpOnly)

	accessToken := verifyResp.Data.AccessToken

	// 5. GET /api/v1/auth/me returns authenticated member
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+accessToken)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	require.Equal(t, http.StatusOK, meRec.Code)

	// 6. Check username availability
	handle := "stride_" + uuid.NewString()[:6]
	availReq := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/username-available?username="+handle, nil)
	availReq.Header.Set("Authorization", "Bearer "+accessToken)
	availRec := httptest.NewRecorder()
	router.ServeHTTP(availRec, availReq)
	require.Equal(t, http.StatusOK, availRec.Code)
	assert.Contains(t, availRec.Body.String(), `"available":true`)

	// 7. Complete guided onboarding
	startW := 98500
	goalW := 82000
	onboardPayload, _ := json.Marshal(map[string]any{
		"username":             handle,
		"display_name":         "Jordan Stride",
		"bio":                  "Ready for 30 days of walking quests!",
		"unit_pref":            "kg",
		"start_weight_grams":   startW,
		"current_weight_grams": startW,
		"goal_weight_grams":    goalW,
		"weight_privacy":       "squads_only",
		"journey_focus":        []string{"daily_steps", "mindful_meals"},
		"timezone":             "UTC",
	})
	onboardReq := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", bytes.NewReader(onboardPayload))
	onboardReq.Header.Set("Content-Type", "application/json")
	onboardReq.Header.Set("Authorization", "Bearer "+accessToken)
	onboardRec := httptest.NewRecorder()
	router.ServeHTTP(onboardRec, onboardReq)
	require.Equal(t, http.StatusOK, onboardRec.Code)
	assert.Contains(t, onboardRec.Body.String(), `"onboarding_completed":true`)
	assert.Contains(t, onboardRec.Body.String(), handle)

	// 8. Refresh rotates session cookie and issues a new access token
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	router.ServeHTTP(refreshRec, refreshReq)
	require.Equal(t, http.StatusOK, refreshRec.Code)

	var refreshBody struct {
		Data struct {
			AccessToken         string `json:"access_token"`
			OnboardingCompleted bool   `json:"onboarding_completed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(refreshRec.Body.Bytes(), &refreshBody))
	assert.NotEmpty(t, refreshBody.Data.AccessToken)
	assert.True(t, refreshBody.Data.OnboardingCompleted)

	// 9. Logout revokes Redis session & denies the access token JTI immediately
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+refreshBody.Data.AccessToken)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)
	require.Equal(t, http.StatusOK, logoutRec.Code)

	// Reusing the revoked access token is rejected by the Redis JTI denylist
	deniedReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	deniedReq.Header.Set("Authorization", "Bearer "+refreshBody.Data.AccessToken)
	deniedRec := httptest.NewRecorder()
	router.ServeHTTP(deniedRec, deniedReq)
	require.Equal(t, http.StatusUnauthorized, deniedRec.Code)
}
