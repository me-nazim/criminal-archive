package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResetTokenTTL is how long a password reset link remains valid.
const ResetTokenTTL = 30 * time.Minute

// ResetRecord is a row in the password_resets table.
type ResetRecord struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	UsedAt     *time.Time
	CreatedAt  time.Time
}

// CreateResetToken generates a fresh reset token, stores its hash, and
// returns the *raw* token (caller emails it to the user).
//
// The function is intentionally non-leaky about whether the email is
// real — return (rawToken, nil) on success and (zero, ErrNoSuchUser) so
// callers can decide whether to surface that or stay silent.
func (r *Repository) CreateResetToken(ctx context.Context, email, ip, userAgent string) (string, *User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, _, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrNoSuchUser
		}
		return "", nil, err
	}
	raw, hash, err := newOpaqueToken(32)
	if err != nil {
		return "", nil, err
	}
	const stmt = `
INSERT INTO password_resets (user_id, token_hash, requested_ip, user_agent, expires_at)
VALUES ($1, $2, NULLIF($3,'')::inet, $4, $5)`
	_, err = r.pool.Exec(ctx, stmt,
		user.ID, hash, ip, nullIfEmpty(userAgent), time.Now().Add(ResetTokenTTL),
	)
	if err != nil {
		return "", nil, err
	}
	return raw, user, nil
}

// FindActiveResetToken returns the row backing the given raw token, if
// it exists and is not yet used or expired.
func (r *Repository) FindActiveResetToken(ctx context.Context, rawToken string) (*ResetRecord, error) {
	hash := hashToken(rawToken)
	const stmt = `
SELECT id, user_id, token_hash, expires_at, used_at, created_at
FROM   password_resets
WHERE  token_hash = $1 AND used_at IS NULL AND expires_at > now()`
	rec := &ResetRecord{}
	err := r.pool.QueryRow(ctx, stmt, hash).Scan(
		&rec.ID, &rec.UserID, &rec.TokenHash, &rec.ExpiresAt, &rec.UsedAt, &rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// MarkResetTokenUsed flips used_at = now() so the link is single-use.
func (r *Repository) MarkResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE password_resets SET used_at = now() WHERE id = $1`, id)
	return err
}

// ErrNoSuchUser is returned when a reset is requested for an unknown
// email. Surface it carefully — most callers should swallow it to avoid
// account-enumeration via the response.
var ErrNoSuchUser = errors.New("auth: user not found")

// newOpaqueToken returns (raw, hash) where raw is a URL-safe random
// string and hash is SHA-256(raw) — exactly the same scheme used for
// refresh tokens, but kept separate so a leaked refresh hash can't be
// replayed as a reset token and vice versa.
func newOpaqueToken(byteLen int) (raw, hash string, err error) {
	buf := make([]byte, byteLen)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
