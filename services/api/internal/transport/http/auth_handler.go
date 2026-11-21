package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	"github.com/clickTwice26/fatman/services/api/internal/service"
	"github.com/go-playground/validator/v10"
)

const (
	refreshCookieName = "fatman_refresh"
	maxJSONBodyBytes  = 64 * 1024
)

// AuthHandler exposes passwordless Email-OTP auth and onboarding HTTP endpoints.
type AuthHandler struct {
	auth         *service.AuthService
	validate     *validator.Validate
	cookieSecure bool
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(auth *service.AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		auth:         auth,
		validate:     validator.New(),
		cookieSecure: cookieSecure,
	}
}

type requestOTPBody struct {
	Email string `json:"email" validate:"required,email,max=254"`
}

type verifyOTPBody struct {
	Email string `json:"email" validate:"required,email,max=254"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}

type completeOnboardingBody struct {
	Username           string   `json:"username" validate:"required,min=3,max=24"`
	DisplayName        string   `json:"display_name" validate:"required,min=2,max=60"`
	Bio                string   `json:"bio" validate:"max=280"`
	UnitPref           string   `json:"unit_pref" validate:"required,oneof=kg lb"`
	StartWeightGrams   *int     `json:"start_weight_grams" validate:"omitempty,gt=20000,lt=500000"`
	CurrentWeightGrams *int     `json:"current_weight_grams" validate:"omitempty,gt=20000,lt=500000"`
	GoalWeightGrams    *int     `json:"goal_weight_grams" validate:"omitempty,gt=20000,lt=500000"`
	WeightPrivacy      string   `json:"weight_privacy" validate:"required,oneof=public squads_only private"`
	JourneyFocus       []string `json:"journey_focus" validate:"max=8,dive,min=2,max=40"`
	Timezone           string   `json:"timezone" validate:"max=64"`
}

type userProfileDTO struct {
	DisplayName        string   `json:"display_name"`
	Bio                string   `json:"bio"`
	AvatarURL          string   `json:"avatar_url"`
	UnitPref           string   `json:"unit_pref"`
	StartWeightGrams   *int     `json:"start_weight_grams,omitempty"`
	CurrentWeightGrams *int     `json:"current_weight_grams,omitempty"`
	GoalWeightGrams    *int     `json:"goal_weight_grams,omitempty"`
	WeightPrivacy      string   `json:"weight_privacy"`
	JourneyFocus       []string `json:"journey_focus"`
	Timezone           string   `json:"timezone"`
}

type userDTO struct {
	ID                  string          `json:"id"`
	Email               string          `json:"email"`
	Username            *string         `json:"username,omitempty"`
	OnboardingCompleted bool            `json:"onboarding_completed"`
	CreatedAt           time.Time       `json:"created_at"`
	Profile             *userProfileDTO `json:"profile,omitempty"`
}

func toUserDTO(u domain.User) userDTO {
	dto := userDTO{
		ID:                  u.ID,
		Email:               u.Email,
		Username:            u.Username,
		OnboardingCompleted: u.OnboardingCompleted(),
		CreatedAt:           u.CreatedAt,
	}
	if u.Profile != nil {
		dto.Profile = &userProfileDTO{
			DisplayName:        u.Profile.DisplayName,
			Bio:                u.Profile.Bio,
			AvatarURL:          u.Profile.AvatarURL,
			UnitPref:           string(u.Profile.UnitPref),
			StartWeightGrams:   u.Profile.StartWeightGrams,
			CurrentWeightGrams: u.Profile.CurrentWeightGrams,
			GoalWeightGrams:    u.Profile.GoalWeightGrams,
			WeightPrivacy:      string(u.Profile.WeightPrivacy),
			JourneyFocus:       u.Profile.JourneyFocus,
			Timezone:           u.Profile.Timezone,
		}
	}
	return dto
}

// RequestOTP handles POST /api/v1/auth/otp/request.
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var body requestOTPBody
	if !decodeAndValidate(w, r, h.validate, &body) {
		return
	}

	res, err := h.auth.RequestEmailOTP(r.Context(), body.Email)
	if err != nil {
		if errors.Is(err, domain.ErrOTPCooldownActive) {
			retryAfter := h.auth.RemainingOTPCooldown(r.Context(), body.Email)
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			WriteError(w, http.StatusTooManyRequests, "OTP_COOLDOWN", "a verification code was just sent; please wait before requesting another", map[string]int{
				"retry_after_seconds": retryAfter,
			})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "please enter a valid email address", nil)
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to send verification code", nil)
		return
	}

	WriteJSON(w, http.StatusAccepted, map[string]any{
		"message":              "verification code sent",
		"expires_in_seconds":   res.ExpiresInSeconds,
		"resend_after_seconds": res.ResendAfterSeconds,
	})
}

// VerifyOTP handles POST /api/v1/auth/otp/verify.
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var body verifyOTPBody
	if !decodeAndValidate(w, r, h.validate, &body) {
		return
	}

	res, err := h.auth.VerifyEmailOTP(r.Context(), body.Email, body.Code, r.UserAgent(), ClientIPKey(r))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOTPAttemptsExceeded):
			WriteError(w, http.StatusTooManyRequests, "OTP_ATTEMPTS_EXCEEDED", "too many incorrect attempts; please request a new code", nil)
		case errors.Is(err, domain.ErrOTPInvalid):
			WriteError(w, http.StatusUnauthorized, "OTP_INVALID", "verification code is invalid or expired", nil)
		default:
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to verify code", nil)
		}
		return
	}

	h.setRefreshCookie(w, res.RefreshToken, res.RefreshExpiresAt)
	WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":         res.AccessToken,
		"is_new_user":          res.IsNewUser,
		"onboarding_completed": res.OnboardingCompleted,
		"user":                 toUserDTO(res.User),
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing refresh session", nil)
		return
	}

	res, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		if errors.Is(err, domain.ErrSessionReuseDetected) {
			WriteError(w, http.StatusUnauthorized, "SESSION_COMPROMISED", "session invalidated for security; please sign in again", nil)
			return
		}
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session expired; please sign in again", nil)
		return
	}

	h.setRefreshCookie(w, res.RefreshToken, res.RefreshExpiresAt)
	WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":         res.AccessToken,
		"onboarding_completed": res.OnboardingCompleted,
		"user":                 toUserDTO(res.User),
	})
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := AuthClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	if err := h.auth.Logout(r.Context(), claims); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to complete logout", nil)
		return
	}
	h.clearRefreshCookie(w)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// Me handles GET /api/v1/auth/me.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := AuthClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	user, err := h.auth.GetCurrentUser(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "account not found", nil)
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to load member profile", nil)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"user": toUserDTO(user),
	})
}

// CheckUsernameAvailable handles GET /api/v1/onboarding/username-available.
func (h *AuthHandler) CheckUsernameAvailable(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	available, err := h.auth.CheckUsernameAvailable(r.Context(), username)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "username must be 3-24 characters using letters, numbers, or underscores", nil)
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to check username", nil)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"username":  username,
		"available": available,
	})
}

// CompleteOnboarding handles POST /api/v1/onboarding/complete.
func (h *AuthHandler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	claims, ok := AuthClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}

	var body completeOnboardingBody
	if !decodeAndValidate(w, r, h.validate, &body) {
		return
	}

	user, err := h.auth.CompleteOnboarding(r.Context(), domain.CompleteOnboardingInput{
		UserID:             claims.UserID,
		Username:           body.Username,
		DisplayName:        body.DisplayName,
		Bio:                body.Bio,
		UnitPref:           domain.WeightUnit(body.UnitPref),
		StartWeightGrams:   body.StartWeightGrams,
		CurrentWeightGrams: body.CurrentWeightGrams,
		GoalWeightGrams:    body.GoalWeightGrams,
		WeightPrivacy:      domain.WeightPrivacy(body.WeightPrivacy),
		JourneyFocus:       body.JourneyFocus,
		Timezone:           body.Timezone,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUsernameTaken):
			WriteError(w, http.StatusConflict, "USERNAME_TAKEN", "that username is already taken", nil)
		case errors.Is(err, domain.ErrInvalidInput):
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "please review your onboarding details", nil)
		default:
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to save onboarding profile", nil)
		}
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"onboarding_completed": user.OnboardingCompleted(),
		"user":                 toUserDTO(user),
	})
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func decodeAndValidate(w http.ResponseWriter, r *http.Request, validate *validator.Validate, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "request body must be valid JSON", nil)
		return false
	}
	if err := validate.Struct(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "one or more fields failed validation", formatValidationErrors(err))
		return false
	}
	return true
}

func formatValidationErrors(err error) map[string]string {
	out := make(map[string]string)
	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		for _, fe := range vErrs {
			out[strings.ToLower(fe.Field())] = fe.Tag()
		}
	}
	return out
}
