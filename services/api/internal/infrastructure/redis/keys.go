package redis

import "fmt"

// KeyBuilder constructs canonical Redis keys following:
// fatman:{env}:{scope}:{entity}:{identifier}
type KeyBuilder struct {
	env string
}

// NewKeyBuilder creates a KeyBuilder scoped to an environment (dev, test, prod).
func NewKeyBuilder(env string) KeyBuilder {
	if env == "" {
		env = "dev"
	}
	return KeyBuilder{env: env}
}

func (k KeyBuilder) key(scope, entity, identifier string) string {
	return fmt.Sprintf("fatman:%s:%s:%s:%s", k.env, scope, entity, identifier)
}

// AuthSession stores a rotating refresh session hash.
func (k KeyBuilder) AuthSession(sessionID string) string {
	return k.key("auth", "session", sessionID)
}

// AuthDenyJTI stores a revoked JWT access token ID until its natural expiry.
func (k KeyBuilder) AuthDenyJTI(jti string) string {
	return k.key("auth", "deny", jti)
}

// ValidationEmailOTP stores a hashed 6-digit email OTP and attempt counter.
func (k KeyBuilder) ValidationEmailOTP(emailHash string) string {
	return k.key("val", "otp:email", emailHash)
}

// ValidationOTPCooldown enforces resend cooldown per email address.
func (k KeyBuilder) ValidationOTPCooldown(emailHash string) string {
	return k.key("val", "otp_cooldown:email", emailHash)
}

// ValidationUsernameLock reserves a username briefly during onboarding submission.
func (k KeyBuilder) ValidationUsernameLock(username string) string {
	return k.key("val", "lock:username", username)
}

// ValidationIdempotency prevents duplicate writes for actions with Idempotency-Key.
func (k KeyBuilder) ValidationIdempotency(action, key string) string {
	return k.key("val", "idempotency:"+action, key)
}

// RateLimit constructs a sliding-window rate-limit bucket key.
func (k KeyBuilder) RateLimit(route, subject string) string {
	return k.key("rl", route, subject)
}

// CacheProfile caches a user's public/self profile summary.
func (k KeyBuilder) CacheProfile(userID string) string {
	return k.key("cache", "profile", userID)
}

// CacheQuestLeaderboard stores a quest's Redis ZSET leaderboard.
func (k KeyBuilder) CacheQuestLeaderboard(questID string) string {
	return k.key("cache", "quest_lb", questID)
}
