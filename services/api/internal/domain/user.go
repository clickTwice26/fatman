package domain

import "time"

// WeightUnit represents a member's preferred display unit.
type WeightUnit string

const (
	WeightUnitKG WeightUnit = "kg"
	WeightUnitLB WeightUnit = "lb"
)

// WeightPrivacy governs who can view a member's weight metrics.
type WeightPrivacy string

const (
	WeightPrivacyPublic     WeightPrivacy = "public"
	WeightPrivacySquadsOnly WeightPrivacy = "squads_only"
	WeightPrivacyPrivate    WeightPrivacy = "private"
)

// User represents a Fatman community account authenticated via email OTP.
type User struct {
	ID                    string
	Email                 string
	Username              *string
	EmailVerifiedAt       time.Time
	OnboardingCompletedAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Profile               *UserProfile
}

// OnboardingCompleted returns true when the member has finished the onboarding wizard.
func (u User) OnboardingCompleted() bool {
	return u.OnboardingCompletedAt != nil
}

// UserProfile stores a member's customizable profile and weight-loss journey settings.
type UserProfile struct {
	UserID             string
	DisplayName        string
	Bio                string
	AvatarURL          string
	UnitPref           WeightUnit
	StartWeightGrams   *int
	CurrentWeightGrams *int
	GoalWeightGrams    *int
	WeightPrivacy      WeightPrivacy
	JourneyFocus       []string
	Timezone           string
	UpdatedAt          time.Time
}

// CompleteOnboardingInput carries validated onboarding fields submitted by a new member.
type CompleteOnboardingInput struct {
	UserID             string
	Username           string
	DisplayName        string
	Bio                string
	UnitPref           WeightUnit
	StartWeightGrams   *int
	CurrentWeightGrams *int
	GoalWeightGrams    *int
	WeightPrivacy      WeightPrivacy
	JourneyFocus       []string
	Timezone           string
}
