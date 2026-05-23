package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessClaims is what we sign into an access JWT.
type AccessClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// JWTConfig holds runtime config for token issuance.
type JWTConfig struct {
	Secret     []byte
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// IssueAccessToken signs a short-lived JWT for the given user.
func IssueAccessToken(cfg JWTConfig, userID uuid.UUID, role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(cfg.AccessTTL)
	claims := AccessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(cfg.Secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

// ParseAccessToken validates and parses an access JWT. The returned
// AccessClaims is only populated on success.
func ParseAccessToken(cfg JWTConfig, raw string) (*AccessClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(cfg.Issuer),
	)
	tok, err := parser.ParseWithClaims(raw, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		return cfg.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// IsExpired reports whether err returned by ParseAccessToken indicates
// expiration vs. some other validation failure.
func IsExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}

// ----- Refresh tokens -----

// NewRefreshToken returns a fresh, opaque token plus its sha256 hash.
// Only the hash is stored in the database; the raw value lives in the
// httpOnly cookie on the client.
func NewRefreshToken() (raw, hashed string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("rand: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hashed = HashRefreshToken(raw)
	return raw, hashed, nil
}

// HashRefreshToken returns the canonical sha256 hex digest of raw.
// We use sha256 (not bcrypt) because refresh tokens are 256-bit random
// values, so we don't need slow hashing — we need indexable equality.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
