// Package auth implements password hashing, JWT issuance, refresh token
// management, and the HTTP middleware that gates routes on role and
// approval status.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength enforces a baseline password length on registration.
// We deliberately don't check for "complexity" — length + a strong hash
// beats character-class theatre.
const MinPasswordLength = 10

// ErrPasswordTooShort is returned by ValidatePassword if the input is
// below MinPasswordLength.
var ErrPasswordTooShort = errors.New("password must be at least 10 characters")

// HashPassword returns a bcrypt hash of plain at the given cost.
func HashPassword(plain string, cost int) (string, error) {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	if err := ValidatePassword(plain); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword reports whether plain matches a previously hashed value.
// It returns nil on match, bcrypt.ErrMismatchedHashAndPassword on mismatch,
// or another error on infrastructure failures.
func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// ValidatePassword applies the project's password policy.
func ValidatePassword(plain string) error {
	if len([]rune(plain)) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}
