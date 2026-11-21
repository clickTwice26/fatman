package redis

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	goredis "github.com/redis/go-redis/v9"
)

var releaseLockScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

// OTPStore implements Redis-backed email OTP validation, resend cooldowns, and username reservation locks.
type OTPStore struct {
	rdb         *goredis.Client
	kb          infraredis.KeyBuilder
	hmacSecret  []byte
	ttl         time.Duration
	cooldown    time.Duration
	maxAttempts int
}

// NewOTPStore constructs a Redis validation store for email OTP onboarding.
func NewOTPStore(
	rdb *goredis.Client,
	kb infraredis.KeyBuilder,
	hmacSecret string,
	ttl time.Duration,
	cooldown time.Duration,
	maxAttempts int,
) *OTPStore {
	return &OTPStore{
		rdb:         rdb,
		kb:          kb,
		hmacSecret:  []byte(hmacSecret),
		ttl:         ttl,
		cooldown:    cooldown,
		maxAttempts: maxAttempts,
	}
}

// IssueEmailOTP stores an HMAC-SHA256 digest of the OTP in Redis if the resend cooldown is not active.
func (s *OTPStore) IssueEmailOTP(ctx context.Context, email, code string) error {
	normEmail := normalizeEmail(email)
	emailHash := s.hashValue("email", normEmail)
	cooldownKey := s.kb.ValidationOTPCooldown(emailHash)

	acquired, err := s.rdb.SetNX(ctx, cooldownKey, "1", s.cooldown).Result()
	if err != nil {
		return fmt.Errorf("check otp cooldown: %w", err)
	}
	if !acquired {
		return domain.ErrOTPCooldownActive
	}

	otpKey := s.kb.ValidationEmailOTP(emailHash)
	digest := s.hashValue("otp", normEmail+":"+strings.TrimSpace(code))

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, otpKey, "digest", digest, "attempts", 0)
	pipe.Expire(ctx, otpKey, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store email otp: %w", err)
	}
	return nil
}

// VerifyEmailOTP performs constant-time HMAC verification, enforces attempt caps, and consumes the OTP on success.
func (s *OTPStore) VerifyEmailOTP(ctx context.Context, email, code string) error {
	normEmail := normalizeEmail(email)
	emailHash := s.hashValue("email", normEmail)
	otpKey := s.kb.ValidationEmailOTP(emailHash)

	vals, err := s.rdb.HGetAll(ctx, otpKey).Result()
	if err != nil {
		return fmt.Errorf("fetch email otp: %w", err)
	}
	storedDigest, ok := vals["digest"]
	if !ok || storedDigest == "" {
		return domain.ErrOTPInvalid
	}

	expectedDigest := s.hashValue("otp", normEmail+":"+strings.TrimSpace(code))
	if hmac.Equal([]byte(storedDigest), []byte(expectedDigest)) {
		deleted, delErr := s.rdb.Del(ctx, otpKey).Result()
		if delErr != nil {
			return fmt.Errorf("consume email otp: %w", delErr)
		}
		if deleted == 0 {
			return domain.ErrOTPInvalid
		}
		return nil
	}

	attempts, err := s.rdb.HIncrBy(ctx, otpKey, "attempts", 1).Result()
	if err != nil {
		return fmt.Errorf("increment otp attempts: %w", err)
	}
	if int(attempts) >= s.maxAttempts {
		_ = s.rdb.Del(ctx, otpKey).Err()
		return domain.ErrOTPAttemptsExceeded
	}
	return domain.ErrOTPInvalid
}

// AcquireUsernameLock reserves username briefly for userID during onboarding completion.
func (s *OTPStore) AcquireUsernameLock(ctx context.Context, username, userID string, ttl time.Duration) (bool, error) {
	key := s.kb.ValidationUsernameLock(strings.ToLower(strings.TrimSpace(username)))
	ok, err := s.rdb.SetNX(ctx, key, userID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("acquire username lock: %w", err)
	}
	return ok, nil
}

// ReleaseUsernameLock releases username only if still held by userID.
func (s *OTPStore) ReleaseUsernameLock(ctx context.Context, username, userID string) error {
	key := s.kb.ValidationUsernameLock(strings.ToLower(strings.TrimSpace(username)))
	if err := releaseLockScript.Run(ctx, s.rdb, []string{key}, userID).Err(); err != nil {
		return fmt.Errorf("release username lock: %w", err)
	}
	return nil
}

// RemainingCooldown returns remaining resend cooldown seconds for email (0 if ready).
func (s *OTPStore) RemainingCooldown(ctx context.Context, email string) (int, error) {
	emailHash := s.hashValue("email", normalizeEmail(email))
	ttl, err := s.rdb.TTL(ctx, s.kb.ValidationOTPCooldown(emailHash)).Result()
	if err != nil {
		return 0, err
	}
	if ttl <= 0 {
		return 0, nil
	}
	return int(ttl.Seconds()) + 1, nil
}

func (s *OTPStore) hashValue(namespace, val string) string {
	mac := hmac.New(sha256.New, s.hmacSecret)
	_, _ = mac.Write([]byte(namespace + ":" + val))
	return hex.EncodeToString(mac.Sum(nil))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
