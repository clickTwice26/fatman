package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	infrapg "github.com/clickTwice26/fatman/services/api/internal/infrastructure/postgres"
	repopg "github.com/clickTwice26/fatman/services/api/internal/repository/postgres"
	"github.com/clickTwice26/fatman/services/api/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPG(t *testing.T) *infrapg.Client {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://fatman:fatman_dev_password@localhost:5432/fatman?sslmode=disable"
	}
	client, err := infrapg.New(context.Background(), dbURL)
	require.NoError(t, err, "Docker PostgreSQL must be running for PG integration tests (make infra-up)")
	t.Cleanup(func() {
		client.Close()
	})
	require.NoError(t, migrations.Up(context.Background(), client.Pool))
	return client
}

func TestMigrations_ReversibleUpAndDown(t *testing.T) {
	pg := newTestPG(t)
	ctx := context.Background()

	require.NoError(t, migrations.Down(ctx, pg.Pool))
	require.NoError(t, migrations.Up(ctx, pg.Pool))
}

func TestUserRepository_EmailOTPUpsertAndOnboarding(t *testing.T) {
	pg := newTestPG(t)
	repo := repopg.NewUserRepository(pg.Pool)
	ctx := context.Background()

	uniqueEmail := fmt.Sprintf("Member.%s@Example.com", uuid.NewString()[:8])

	// First verified OTP creates the user.
	user1, isNew, err := repo.UpsertVerifiedByEmail(ctx, uniqueEmail)
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.NotEmpty(t, user1.ID)
	assert.Equal(t, uniqueEmail, user1.Email)
	assert.False(t, user1.OnboardingCompleted())
	assert.Nil(t, user1.Username)

	// Second verified OTP with different casing finds the same user.
	user2, isNew2, err := repo.UpsertVerifiedByEmail(ctx, "member."+uniqueEmail[7:])
	require.NoError(t, err)
	assert.False(t, isNew2)
	assert.Equal(t, user1.ID, user2.ID)

	// Username availability check before onboarding.
	handle := "walker_" + uuid.NewString()[:6]
	avail, err := repo.IsUsernameAvailable(ctx, handle)
	require.NoError(t, err)
	assert.True(t, avail)

	startW := 105000
	goalW := 85000
	completed, err := repo.CompleteOnboarding(ctx, domain.CompleteOnboardingInput{
		UserID:             user1.ID,
		Username:           handle,
		DisplayName:        "Sam Walker",
		Bio:                "Walking 8k steps every day!",
		UnitPref:           domain.WeightUnitKG,
		StartWeightGrams:   &startW,
		CurrentWeightGrams: &startW,
		GoalWeightGrams:    &goalW,
		WeightPrivacy:      domain.WeightPrivacySquadsOnly,
		JourneyFocus:       []string{"daily_steps", "hydration"},
		Timezone:           "America/New_York",
	})
	require.NoError(t, err)
	assert.True(t, completed.OnboardingCompleted())
	require.NotNil(t, completed.Username)
	assert.Equal(t, handle, *completed.Username)
	require.NotNil(t, completed.Profile)
	assert.Equal(t, "Sam Walker", completed.Profile.DisplayName)
	assert.Equal(t, domain.WeightPrivacySquadsOnly, completed.Profile.WeightPrivacy)
	assert.Equal(t, []string{"daily_steps", "hydration"}, completed.Profile.JourneyFocus)

	// Username is no longer available (case-insensitive).
	availAfter, err := repo.IsUsernameAvailable(ctx, "WALKER_"+handle[7:])
	require.NoError(t, err)
	assert.False(t, availAfter)

	// Second user attempting to claim the same username receives ErrUsernameTaken.
	otherEmail := fmt.Sprintf("other.%s@example.com", uuid.NewString()[:8])
	otherUser, _, err := repo.UpsertVerifiedByEmail(ctx, otherEmail)
	require.NoError(t, err)

	_, err = repo.CompleteOnboarding(ctx, domain.CompleteOnboardingInput{
		UserID:        otherUser.ID,
		Username:      handle,
		DisplayName:   "Other User",
		UnitPref:      domain.WeightUnitLB,
		WeightPrivacy: domain.WeightPrivacyPrivate,
		Timezone:      "UTC",
	})
	require.ErrorIs(t, err, domain.ErrUsernameTaken)
}
