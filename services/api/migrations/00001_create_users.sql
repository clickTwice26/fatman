-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    email CITEXT NOT NULL UNIQUE,
    username CITEXT UNIQUE,
    password_hash TEXT NULL,
    email_verified_at TIMESTAMPTZ NOT NULL,
    onboarding_completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    unit_pref TEXT NOT NULL DEFAULT 'kg' CHECK (unit_pref IN ('kg', 'lb')),
    start_weight_grams INT NULL CHECK (start_weight_grams IS NULL OR start_weight_grams > 0),
    current_weight_grams INT NULL CHECK (current_weight_grams IS NULL OR current_weight_grams > 0),
    goal_weight_grams INT NULL CHECK (goal_weight_grams IS NULL OR goal_weight_grams > 0),
    weight_privacy TEXT NOT NULL DEFAULT 'private' CHECK (weight_privacy IN ('public', 'squads_only', 'private')),
    journey_focus TEXT[] NOT NULL DEFAULT '{}',
    timezone TEXT NOT NULL DEFAULT 'UTC',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
