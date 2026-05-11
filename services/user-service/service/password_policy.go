package service

import (
	"errors"
	"strings"
)

var (
	ErrPasswordTooShort = errors.New("password_too_short")
	ErrPasswordTooLong  = errors.New("password_too_long")
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
)

// NormalizePassword trims leading/trailing whitespace before policy checks and hashing.
func NormalizePassword(password string) string {
	return strings.TrimSpace(password)
}

// ValidatePasswordLength enforces the end-user password policy (8–72 chars).
func ValidatePasswordLength(password string) error {
	n := len([]rune(password))
	if n < minPasswordLength {
		return ErrPasswordTooShort
	}
	if n > maxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}
