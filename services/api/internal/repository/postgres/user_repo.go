package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/clickTwice26/fatman/services/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository persists users and onboarding profiles in PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a PostgreSQL-backed UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// UpsertVerifiedByEmail finds an existing user by email (case-insensitive) or inserts a new verified user.
func (r *UserRepository) UpsertVerifiedByEmail(ctx context.Context, email string) (domain.User, bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return domain.User{}, false, domain.ErrInvalidInput
	}

	now := time.Now().UTC()
	newID := uuid.NewString()

	// xmax = 0 indicates the row was newly inserted rather than updated on conflict.
	const query = `
		INSERT INTO users (id, email, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $3, $3)
		ON CONFLICT (email) DO UPDATE
		SET email_verified_at = COALESCE(users.email_verified_at, EXCLUDED.email_verified_at),
		    updated_at = EXCLUDED.updated_at
		RETURNING id, email, username, email_verified_at, onboarding_completed_at, created_at, updated_at, (xmax = 0) AS inserted
	`

	var u domain.User
	var inserted bool
	err := r.pool.QueryRow(ctx, query, newID, email, now).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.EmailVerifiedAt,
		&u.OnboardingCompletedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&inserted,
	)
	if err != nil {
		return domain.User{}, false, fmt.Errorf("upsert verified user: %w", err)
	}

	profile, err := r.findProfile(ctx, u.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, false, err
	}
	if err == nil {
		u.Profile = &profile
	}

	return u, inserted, nil
}

// FindByID loads a user and their profile (if created).
func (r *UserRepository) FindByID(ctx context.Context, userID string) (domain.User, error) {
	const query = `
		SELECT id, email, username, email_verified_at, onboarding_completed_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.EmailVerifiedAt,
		&u.OnboardingCompletedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}

	profile, err := r.findProfile(ctx, u.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}
	if err == nil {
		u.Profile = &profile
	}
	return u, nil
}

// IsUsernameAvailable checks whether username is currently unclaimed (case-insensitive).
func (r *UserRepository) IsUsernameAvailable(ctx context.Context, username string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, domain.ErrInvalidInput
	}
	const query = `SELECT NOT EXISTS (SELECT 1 FROM users WHERE username = $1)`
	var available bool
	if err := r.pool.QueryRow(ctx, query, username).Scan(&available); err != nil {
		return false, fmt.Errorf("check username availability: %w", err)
	}
	return available, nil
}

// CompleteOnboarding atomically assigns username, marks onboarding completed, and upserts the member's profile.
func (r *UserRepository) CompleteOnboarding(ctx context.Context, in domain.CompleteOnboardingInput) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin onboarding tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	const updateUserSQL = `
		UPDATE users
		SET username = $2,
		    onboarding_completed_at = COALESCE(onboarding_completed_at, $3),
		    updated_at = $3
		WHERE id = $1
		RETURNING id, email, username, email_verified_at, onboarding_completed_at, created_at, updated_at
	`
	var u domain.User
	err = tx.QueryRow(ctx, updateUserSQL, in.UserID, strings.TrimSpace(in.Username), now).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.EmailVerifiedAt,
		&u.OnboardingCompletedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, domain.ErrUsernameTaken
		}
		return domain.User{}, fmt.Errorf("update user onboarding: %w", err)
	}

	focus := in.JourneyFocus
	if focus == nil {
		focus = []string{}
	}
	unit := in.UnitPref
	if unit == "" {
		unit = domain.WeightUnitKG
	}
	privacy := in.WeightPrivacy
	if privacy == "" {
		privacy = domain.WeightPrivacyPrivate
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		tz = "UTC"
	}

	const upsertProfileSQL = `
		INSERT INTO user_profiles (
			user_id, display_name, bio, unit_pref,
			start_weight_grams, current_weight_grams, goal_weight_grams,
			weight_privacy, journey_focus, timezone, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    bio = EXCLUDED.bio,
		    unit_pref = EXCLUDED.unit_pref,
		    start_weight_grams = EXCLUDED.start_weight_grams,
		    current_weight_grams = EXCLUDED.current_weight_grams,
		    goal_weight_grams = EXCLUDED.goal_weight_grams,
		    weight_privacy = EXCLUDED.weight_privacy,
		    journey_focus = EXCLUDED.journey_focus,
		    timezone = EXCLUDED.timezone,
		    updated_at = EXCLUDED.updated_at
		RETURNING user_id, display_name, bio, avatar_url, unit_pref,
		          start_weight_grams, current_weight_grams, goal_weight_grams,
		          weight_privacy, journey_focus, timezone, updated_at
	`
	var p domain.UserProfile
	err = tx.QueryRow(ctx, upsertProfileSQL,
		in.UserID,
		strings.TrimSpace(in.DisplayName),
		strings.TrimSpace(in.Bio),
		string(unit),
		in.StartWeightGrams,
		in.CurrentWeightGrams,
		in.GoalWeightGrams,
		string(privacy),
		focus,
		tz,
		now,
	).Scan(
		&p.UserID,
		&p.DisplayName,
		&p.Bio,
		&p.AvatarURL,
		&p.UnitPref,
		&p.StartWeightGrams,
		&p.CurrentWeightGrams,
		&p.GoalWeightGrams,
		&p.WeightPrivacy,
		&p.JourneyFocus,
		&p.Timezone,
		&p.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("commit onboarding tx: %w", err)
	}

	u.Profile = &p
	return u, nil
}

func (r *UserRepository) findProfile(ctx context.Context, userID string) (domain.UserProfile, error) {
	const query = `
		SELECT user_id, display_name, bio, avatar_url, unit_pref,
		       start_weight_grams, current_weight_grams, goal_weight_grams,
		       weight_privacy, journey_focus, timezone, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`
	var p domain.UserProfile
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&p.UserID,
		&p.DisplayName,
		&p.Bio,
		&p.AvatarURL,
		&p.UnitPref,
		&p.StartWeightGrams,
		&p.CurrentWeightGrams,
		&p.GoalWeightGrams,
		&p.WeightPrivacy,
		&p.JourneyFocus,
		&p.Timezone,
		&p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserProfile{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.UserProfile{}, fmt.Errorf("find user profile: %w", err)
	}
	return p, nil
}
