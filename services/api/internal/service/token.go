package service

import (
	"fmt"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const tokenIssuer = "fatman-api"

// AccessTokenClaims represents validated claims extracted from a Fatman JWT access token.
type AccessTokenClaims struct {
	UserID    string
	SessionID string
	JTI       string
	ExpiresAt time.Time
}

type fatmanJWTClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// TokenService signs and verifies short-lived HS256 JWT access tokens.
type TokenService struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenService constructs a TokenService.
func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// IssueAccessToken signs a new JWT access token bound to userID and sessionID.
func (t *TokenService) IssueAccessToken(userID, sessionID string) (string, AccessTokenClaims, error) {
	now := time.Now().UTC()
	exp := now.Add(t.ttl)
	jti := uuid.NewString()

	claims := fatmanJWTClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", AccessTokenClaims{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		JTI:       jti,
		ExpiresAt: exp,
	}, nil
}

// ParseAccessToken validates signature, HS256 algorithm, issuer, and expiry.
func (t *TokenService) ParseAccessToken(raw string) (AccessTokenClaims, error) {
	var parsed fatmanJWTClaims
	tok, err := jwt.ParseWithClaims(raw, &parsed, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrUnauthorized
		}
		return t.secret, nil
	}, jwt.WithIssuer(tokenIssuer), jwt.WithExpirationRequired())
	if err != nil || !tok.Valid || parsed.Subject == "" || parsed.ID == "" {
		return AccessTokenClaims{}, domain.ErrUnauthorized
	}

	return AccessTokenClaims{
		UserID:    parsed.Subject,
		SessionID: parsed.SessionID,
		JTI:       parsed.ID,
		ExpiresAt: parsed.ExpiresAt.Time,
	}, nil
}
