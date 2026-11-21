package redis

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// Session represents an active Redis-backed refresh session.
type Session struct {
	SessionID    string
	UserID       string
	RefreshToken string
	ExpiresAt    time.Time
}

// SessionStore manages rotating refresh token sessions and access token JTI denylists in Redis.
type SessionStore struct {
	rdb        *goredis.Client
	kb         infraredis.KeyBuilder
	hmacSecret []byte
	ttl        time.Duration
}

// NewSessionStore constructs a Redis auth session store.
func NewSessionStore(rdb *goredis.Client, kb infraredis.KeyBuilder, hmacSecret string, ttl time.Duration) *SessionStore {
	return &SessionStore{
		rdb:        rdb,
		kb:         kb,
		hmacSecret: []byte(hmacSecret),
		ttl:        ttl,
	}
}

// CreateSession generates a new sessionID + cryptographically random refresh secret and stores its digest in Redis.
func (s *SessionStore) CreateSession(ctx context.Context, userID, userAgent, ip string) (Session, error) {
	sessionID := uuid.NewString()
	secret, err := randomSecret(32)
	if err != nil {
		return Session{}, err
	}
	digest := s.hashSecret(sessionID, secret)
	key := s.kb.AuthSession(sessionID)

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key,
		"user_id", userID,
		"secret_digest", digest,
		"user_agent", userAgent,
		"ip", ip,
		"version", 1,
		"created_at", time.Now().UTC().Unix(),
	)
	pipe.Expire(ctx, key, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return Session{}, fmt.Errorf("create redis session: %w", err)
	}

	return Session{
		SessionID:    sessionID,
		UserID:       userID,
		RefreshToken: sessionID + "." + secret,
		ExpiresAt:    time.Now().UTC().Add(s.ttl),
	}, nil
}

// RotateSession validates rawRefreshToken, detects replay/reuse, and rotates to a fresh secret.
func (s *SessionStore) RotateSession(ctx context.Context, rawRefreshToken string) (Session, error) {
	sessionID, secret, ok := strings.Cut(strings.TrimSpace(rawRefreshToken), ".")
	if !ok || sessionID == "" || secret == "" {
		return Session{}, domain.ErrSessionRevoked
	}

	key := s.kb.AuthSession(sessionID)
	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return Session{}, fmt.Errorf("load redis session: %w", err)
	}
	storedDigest, hasDigest := vals["secret_digest"]
	userID, hasUser := vals["user_id"]
	if !hasDigest || !hasUser || storedDigest == "" || userID == "" {
		return Session{}, domain.ErrSessionRevoked
	}

	providedDigest := s.hashSecret(sessionID, secret)
	if !hmac.Equal([]byte(storedDigest), []byte(providedDigest)) {
		// Refresh token reuse or forgery detected -> immediately revoke entire session family.
		_ = s.rdb.Del(ctx, key).Err()
		return Session{}, domain.ErrSessionReuseDetected
	}

	newSecret, err := randomSecret(32)
	if err != nil {
		return Session{}, err
	}
	newDigest := s.hashSecret(sessionID, newSecret)

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, "secret_digest", newDigest)
	pipe.HIncrBy(ctx, key, "version", 1)
	pipe.Expire(ctx, key, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return Session{}, fmt.Errorf("rotate redis session: %w", err)
	}

	return Session{
		SessionID:    sessionID,
		UserID:       userID,
		RefreshToken: sessionID + "." + newSecret,
		ExpiresAt:    time.Now().UTC().Add(s.ttl),
	}, nil
}

// RevokeSession deletes a refresh session from Redis.
func (s *SessionStore) RevokeSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return s.rdb.Del(ctx, s.kb.AuthSession(sessionID)).Err()
}

// DenyAccessToken adds an access token's JTI to the Redis denylist until its natural expiration.
func (s *SessionStore) DenyAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	if strings.TrimSpace(jti) == "" || ttl <= 0 {
		return nil
	}
	return s.rdb.Set(ctx, s.kb.AuthDenyJTI(jti), "1", ttl).Err()
}

// IsAccessTokenDenied returns true if the access token's JTI has been revoked.
func (s *SessionStore) IsAccessTokenDenied(ctx context.Context, jti string) (bool, error) {
	n, err := s.rdb.Exists(ctx, s.kb.AuthDenyJTI(jti)).Result()
	if err != nil {
		return false, fmt.Errorf("check access token denylist: %w", err)
	}
	return n > 0, nil
}

func (s *SessionStore) hashSecret(sessionID, secret string) string {
	mac := hmac.New(sha256.New, s.hmacSecret)
	_, _ = mac.Write([]byte(sessionID + ":" + secret))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomSecret(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
