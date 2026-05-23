package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	t.Parallel()
	const pwd = "correct horse battery staple"
	hash, err := HashPassword(pwd, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == pwd {
		t.Fatal("hash equals plaintext")
	}
	if err := VerifyPassword(hash, pwd); err != nil {
		t.Fatalf("verify good password: %v", err)
	}
	if err := VerifyPassword(hash, "wrong-password"); err == nil {
		t.Fatal("verify accepted a wrong password")
	}
}

func TestHashPassword_RejectsShort(t *testing.T) {
	t.Parallel()
	_, err := HashPassword("short", bcrypt.MinCost)
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()
	cases := map[string]error{
		"":                     ErrPasswordTooShort,
		strings.Repeat("a", 9): ErrPasswordTooShort,
		"abcdefghij":           nil, // 10 runes
		"বাংলাঅক্ষর":           nil, // unicode runes count
		strings.Repeat("a", 200): nil,
	}
	for in, want := range cases {
		got := ValidatePassword(in)
		if (got == nil) != (want == nil) {
			t.Errorf("ValidatePassword(%q) = %v, want %v", in, got, want)
		}
	}
}
