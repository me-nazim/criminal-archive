package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testJWTConfig(t *testing.T, ttl time.Duration) JWTConfig {
	t.Helper()
	return JWTConfig{
		Secret:     []byte("super-secret-test-key-must-be-long-enough"),
		Issuer:     "tansiq-test",
		AccessTTL:  ttl,
		RefreshTTL: 24 * time.Hour,
	}
}

func TestIssueAndParseAccessToken(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, 5*time.Minute)
	uid := uuid.New()
	tok, exp, err := IssueAccessToken(cfg, uid, "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !exp.After(time.Now()) {
		t.Fatalf("expected future expiry, got %s", exp)
	}
	if tok == "" || strings.Count(tok, ".") != 2 {
		t.Fatalf("expected 3-part JWT, got %q", tok)
	}
	claims, err := ParseAccessToken(cfg, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != uid.String() {
		t.Errorf("subject mismatch: got %s, want %s", claims.Subject, uid.String())
	}
	if claims.Role != "admin" {
		t.Errorf("role mismatch: got %s, want admin", claims.Role)
	}
}

func TestParseAccessToken_RejectsWrongIssuer(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, 5*time.Minute)
	tok, _, err := IssueAccessToken(cfg, uuid.New(), "viewer")
	if err != nil {
		t.Fatal(err)
	}
	bad := cfg
	bad.Issuer = "someone-else"
	if _, err := ParseAccessToken(bad, tok); err == nil {
		t.Fatal("expected issuer mismatch to fail validation")
	}
}

func TestParseAccessToken_RejectsExpired(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, -1*time.Second) // already expired
	tok, _, err := IssueAccessToken(cfg, uuid.New(), "viewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseAccessToken(cfg, tok); err == nil {
		t.Fatal("expected expired token to fail validation")
	} else if !IsExpired(err) {
		t.Errorf("expected IsExpired=true, got err=%v", err)
	}
}

func TestParseAccessToken_RejectsTampering(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, 5*time.Minute)
	tok, _, err := IssueAccessToken(cfg, uuid.New(), "viewer")
	if err != nil {
		t.Fatal(err)
	}
	// Truncate the signature segment by one character; this guarantees an
	// invalid HMAC regardless of the random bytes the original signature
	// happened to contain.
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if len(parts[2]) < 4 {
		t.Fatalf("signature too short: %q", parts[2])
	}
	parts[2] = parts[2][:len(parts[2])-1]
	tampered := strings.Join(parts, ".")
	if _, err := ParseAccessToken(cfg, tampered); err == nil {
		t.Fatal("expected tampered token to fail validation")
	}
}

func TestRefreshTokenHashStable(t *testing.T) {
	t.Parallel()
	raw, hash1, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	hash2 := HashRefreshToken(raw)
	if hash1 != hash2 {
		t.Errorf("hash mismatch: %q vs %q", hash1, hash2)
	}
	if HashRefreshToken("other") == hash1 {
		t.Error("different inputs produced the same hash")
	}
}
