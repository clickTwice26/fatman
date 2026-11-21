package redis_test

import (
	"testing"

	infraredis "github.com/clickTwice26/fatman/services/api/internal/infrastructure/redis"
	"github.com/stretchr/testify/assert"
)

func TestKeyBuilder_FollowsCanonicalConvention(t *testing.T) {
	kb := infraredis.NewKeyBuilder("dev")

	assert.Equal(t, "fatman:dev:auth:session:sess-123", kb.AuthSession("sess-123"))
	assert.Equal(t, "fatman:dev:auth:deny:jti-456", kb.AuthDenyJTI("jti-456"))
	assert.Equal(t, "fatman:dev:val:otp:email:hash1", kb.ValidationEmailOTP("hash1"))
	assert.Equal(t, "fatman:dev:val:otp_cooldown:email:hash1", kb.ValidationOTPCooldown("hash1"))
	assert.Equal(t, "fatman:dev:val:lock:username:alex", kb.ValidationUsernameLock("alex"))
	assert.Equal(t, "fatman:dev:val:idempotency:checkin:key1", kb.ValidationIdempotency("checkin", "key1"))
	assert.Equal(t, "fatman:dev:rl:otp_request:127.0.0.1", kb.RateLimit("otp_request", "127.0.0.1"))
	assert.Equal(t, "fatman:dev:cache:profile:user-1", kb.CacheProfile("user-1"))
	assert.Equal(t, "fatman:dev:cache:quest_lb:quest-1", kb.CacheQuestLeaderboard("quest-1"))
}
