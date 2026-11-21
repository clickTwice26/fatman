package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	reporedis "github.com/clickTwice26/fatman/services/api/internal/repository/redis"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)

// UserRepository defines the persistence contract required by AuthService.
type UserRepository interface {
	UpsertVerifiedByEmail(ctx context.Context, email string) (domain.User, bool, error)
	FindByID(ctx context.Context, userID string) (domain.User, error)
	IsUsernameAvailable(ctx context.Context, username string) (bool, error)
	CompleteOnboarding(ctx context.Context, in domain.CompleteOnboardingInput) (domain.User, error)
}

// OTPStore defines the Redis validation contract required by AuthService.
type OTPStore interface {
	IssueEmailOTP(ctx context.Context, email, code string) error
	VerifyEmailOTP(ctx context.Context, email, code string) error
	RemainingCooldown(ctx context.Context, email string) (int, error)
	AcquireUsernameLock(ctx context.Context, username, userID string, ttl time.Duration) (bool, error)
	ReleaseUsernameLock(ctx context.Context, username, userID string) error
}

// SessionStore defines the Redis authentication session contract required by AuthService.
type SessionStore interface {
	CreateSession(ctx context.Context, userID, userAgent, ip string) (reporedis.Session, error)
	RotateSession(ctx context.Context, rawRefreshToken string) (reporedis.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	DenyAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	IsAccessTokenDenied(ctx context.Context, jti string) (bool, error)
}

// EmailSender delivers onboarding OTP emails.
type EmailSender interface {
	SendOTP(ctx context.Context, toEmail, code string, expiresIn time.Duration) error
}

// AuthServiceDeps bundles dependencies for AuthService.
type AuthServiceDeps struct {
	Users             UserRepository
	OTPStore          OTPStore
	SessionStore      SessionStore
	Tokens            *TokenService
	Mailer            EmailSender
	OTPTTL            time.Duration
	OTPResendCooldown time.Duration
}

// OTPRequestResult reports delivery metadata to the client.
type OTPRequestResult struct {
	ExpiresInSeconds   int
	ResendAfterSeconds int
}

// AuthResult contains issued credentials and member state after OTP verify or refresh.
type AuthResult struct {
	AccessToken         string
	RefreshToken        string
	RefreshExpiresAt    time.Time
	User                domain.User
	IsNewUser           bool
	OnboardingCompleted bool
}

// AuthService orchestrates passwordless email-OTP authentication and member onboarding.
type AuthService struct {
	deps AuthServiceDeps
}

// NewAuthService creates a new AuthService.
func NewAuthService(deps AuthServiceDeps) *AuthService {
	return &AuthService{deps: deps}
}

// RequestEmailOTP generates a 6-digit OTP, stores its digest in Redis, and sends it via email.
func (s *AuthService) RequestEmailOTP(ctx context.Context, email string) (OTPRequestResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return OTPRequestResult{}, domain.ErrInvalidInput
	}

	code, err := generateSixDigitOTP()
	if err != nil {
		return OTPRequestResult{}, err
	}

	if err := s.deps.OTPStore.IssueEmailOTP(ctx, email, code); err != nil {
		return OTPRequestResult{}, err
	}

	if err := s.deps.Mailer.SendOTP(ctx, email, code, s.deps.OTPTTL); err != nil {
		return OTPRequestResult{}, fmt.Errorf("deliver otp email: %w", err)
	}

	return OTPRequestResult{
		ExpiresInSeconds:   int(s.deps.OTPTTL.Seconds()),
		ResendAfterSeconds: int(s.deps.OTPResendCooldown.Seconds()),
	}, nil
}

// RemainingOTPCooldown returns remaining seconds before email may request another OTP.
func (s *AuthService) RemainingOTPCooldown(ctx context.Context, email string) int {
	sec, err := s.deps.OTPStore.RemainingCooldown(ctx, email)
	if err != nil {
		return int(s.deps.OTPResendCooldown.Seconds())
	}
	return sec
}

// VerifyEmailOTP verifies the 6-digit OTP, upserts the verified user, and issues tokens.
func (s *AuthService) VerifyEmailOTP(ctx context.Context, email, code, userAgent, ip string) (AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)
	if email == "" || len(code) != 6 {
		return AuthResult{}, domain.ErrOTPInvalid
	}

	if err := s.deps.OTPStore.VerifyEmailOTP(ctx, email, code); err != nil {
		return AuthResult{}, err
	}

	user, isNew, err := s.deps.Users.UpsertVerifiedByEmail(ctx, email)
	if err != nil {
		return AuthResult{}, err
	}

	sess, err := s.deps.SessionStore.CreateSession(ctx, user.ID, userAgent, ip)
	if err != nil {
		return AuthResult{}, err
	}

	accessToken, _, err := s.deps.Tokens.IssueAccessToken(user.ID, sess.SessionID)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:         accessToken,
		RefreshToken:        sess.RefreshToken,
		RefreshExpiresAt:    sess.ExpiresAt,
		User:                user,
		IsNewUser:           isNew,
		OnboardingCompleted: user.OnboardingCompleted(),
	}, nil
}

// Refresh rotates the refresh session in Redis and issues a fresh access token.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (AuthResult, error) {
	sess, err := s.deps.SessionStore.RotateSession(ctx, rawRefreshToken)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.deps.Users.FindByID(ctx, sess.UserID)
	if err != nil {
		return AuthResult{}, err
	}

	accessToken, _, err := s.deps.Tokens.IssueAccessToken(user.ID, sess.SessionID)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:         accessToken,
		RefreshToken:        sess.RefreshToken,
		RefreshExpiresAt:    sess.ExpiresAt,
		User:                user,
		OnboardingCompleted: user.OnboardingCompleted(),
	}, nil
}

// Logout revokes the Redis session and adds the active access token JTI to the Redis denylist.
func (s *AuthService) Logout(ctx context.Context, claims AccessTokenClaims) error {
	if err := s.deps.SessionStore.RevokeSession(ctx, claims.SessionID); err != nil {
		return err
	}
	remaining := time.Until(claims.ExpiresAt)
	if remaining > 0 {
		if err := s.deps.SessionStore.DenyAccessToken(ctx, claims.JTI, remaining); err != nil {
			return err
		}
	}
	return nil
}

// GetCurrentUser loads the authenticated member by ID.
func (s *AuthService) GetCurrentUser(ctx context.Context, userID string) (domain.User, error) {
	return s.deps.Users.FindByID(ctx, userID)
}

// CheckUsernameAvailable validates username format and checks availability in PostgreSQL.
func (s *AuthService) CheckUsernameAvailable(ctx context.Context, username string) (bool, error) {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) {
		return false, domain.ErrInvalidInput
	}
	return s.deps.Users.IsUsernameAvailable(ctx, username)
}

// CompleteOnboarding claims a username (guarded by a Redis lock) and persists the member's initial journey profile.
func (s *AuthService) CompleteOnboarding(ctx context.Context, in domain.CompleteOnboardingInput) (domain.User, error) {
	in.Username = strings.TrimSpace(in.Username)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if !usernamePattern.MatchString(in.Username) || len(in.DisplayName) < 2 || len(in.DisplayName) > 60 {
		return domain.User{}, domain.ErrInvalidInput
	}

	acquired, err := s.deps.OTPStore.AcquireUsernameLock(ctx, in.Username, in.UserID, 30*time.Second)
	if err != nil {
		return domain.User{}, err
	}
	if !acquired {
		return domain.User{}, domain.ErrUsernameTaken
	}
	defer func() {
		_ = s.deps.OTPStore.ReleaseUsernameLock(ctx, in.Username, in.UserID)
	}()

	return s.deps.Users.CompleteOnboarding(ctx, in)
}

func generateSixDigitOTP() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate secure otp: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
