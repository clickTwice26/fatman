package domain

import "errors"

var (
	ErrNotFound             = errors.New("resource not found")
	ErrConflict             = errors.New("resource conflict")
	ErrUsernameTaken        = errors.New("username is already taken")
	ErrInvalidInput         = errors.New("invalid input")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrOTPInvalid           = errors.New("invalid or expired verification code")
	ErrOTPAttemptsExceeded  = errors.New("verification code attempts exceeded")
	ErrOTPCooldownActive    = errors.New("please wait before requesting a new verification code")
	ErrSessionRevoked       = errors.New("session revoked or expired")
	ErrSessionReuseDetected = errors.New("refresh token reuse detected")
)
