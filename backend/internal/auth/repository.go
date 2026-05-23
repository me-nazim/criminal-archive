package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User mirrors a row in the users table for auth purposes.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	FullName     string     `json:"full_name"`
	DisplayName  *string    `json:"display_name"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	Phone        *string    `json:"phone"`
	AvatarURL    *string    `json:"avatar_url"`
	Bio          *string    `json:"bio"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Session mirrors a row in the sessions table.
type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshHash  string
	UserAgent    *string
	IPAddress    *string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

// Repository is the SQL access layer for auth.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ErrEmailTaken is returned when CreateUser hits a unique-email conflict.
var ErrEmailTaken = errors.New("email already registered")

// CreateUserParams bundles inputs for CreateUser.
type CreateUserParams struct {
	Email        string
	PasswordHash string
	FullName     string
	Phone        *string
	Role         string // Defaults to "contributor" if empty.
}

// CreateUser inserts a new user with status='pending'. It returns the
// created row.
func (r *Repository) CreateUser(ctx context.Context, p CreateUserParams) (*User, error) {
	role := p.Role
	if role == "" {
		role = "contributor"
	}
	const stmt = `
INSERT INTO users (email, password_hash, full_name, phone, role, status)
VALUES ($1, $2, $3, $4, $5, 'pending')
RETURNING id, email, full_name, display_name, role, status, phone, avatar_url, bio, last_login_at, created_at, updated_at`
	u := &User{}
	err := r.pool.QueryRow(ctx, stmt,
		p.Email, p.PasswordHash, p.FullName, p.Phone, role,
	).Scan(
		&u.ID, &u.Email, &u.FullName, &u.DisplayName, &u.Role, &u.Status,
		&u.Phone, &u.AvatarURL, &u.Bio, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return u, nil
}

// GetUserByID fetches a user by id.
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const stmt = `
SELECT id, email, full_name, display_name, role, status, phone, avatar_url, bio,
       last_login_at, created_at, updated_at
FROM   users WHERE id = $1`
	return scanUserRow(r.pool.QueryRow(ctx, stmt, id))
}

// GetUserByEmail fetches a user by email (case insensitive thanks to citext).
// It also returns the password hash, which is never serialised in responses.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, string, error) {
	const stmt = `
SELECT id, email, full_name, display_name, role, status, phone, avatar_url, bio,
       last_login_at, created_at, updated_at, password_hash
FROM   users WHERE email = $1`
	row := r.pool.QueryRow(ctx, stmt, email)
	var u User
	var hash string
	err := row.Scan(
		&u.ID, &u.Email, &u.FullName, &u.DisplayName, &u.Role, &u.Status,
		&u.Phone, &u.AvatarURL, &u.Bio, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		&hash,
	)
	if err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

// UpdatePassword sets a new password hash and bumps updated_at.
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, hash string) error {
	const stmt = `UPDATE users SET password_hash = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, stmt, hash, userID)
	return err
}

// MarkLoggedIn updates last_login_at to now().
func (r *Repository) MarkLoggedIn(ctx context.Context, userID uuid.UUID) error {
	const stmt = `UPDATE users SET last_login_at = now() WHERE id = $1`
	_, err := r.pool.Exec(ctx, stmt, userID)
	return err
}

// CreateSession persists a hashed refresh token.
func (r *Repository) CreateSession(ctx context.Context, s Session) (*Session, error) {
	const stmt = `
INSERT INTO sessions (user_id, refresh_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, NULLIF($4,'')::inet, $5)
RETURNING id, created_at`
	ip := ""
	if s.IPAddress != nil {
		ip = *s.IPAddress
	}
	if err := r.pool.QueryRow(ctx, stmt,
		s.UserID, s.RefreshHash, s.UserAgent, ip, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindActiveSession returns a non-revoked, non-expired session whose
// refresh_hash matches.
func (r *Repository) FindActiveSession(ctx context.Context, refreshHash string) (*Session, error) {
	const stmt = `
SELECT id, user_id, refresh_hash, user_agent, host(ip_address)::text, expires_at, revoked_at, created_at
FROM   sessions
WHERE  refresh_hash = $1
  AND  revoked_at IS NULL
  AND  expires_at > now()`
	row := r.pool.QueryRow(ctx, stmt, refreshHash)
	var s Session
	if err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshHash, &s.UserAgent, &s.IPAddress,
		&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

// RevokeSession marks a single session as revoked.
func (r *Repository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	const stmt = `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, stmt, sessionID)
	return err
}

// RevokeAllSessionsForUser invalidates every active session for a user
// (used when we suspect a token was reused or when a password changes).
func (r *Repository) RevokeAllSessionsForUser(ctx context.Context, userID uuid.UUID) error {
	const stmt = `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, stmt, userID)
	return err
}

func scanUserRow(row pgx.Row) (*User, error) {
	u := &User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.FullName, &u.DisplayName, &u.Role, &u.Status,
		&u.Phone, &u.AvatarURL, &u.Bio, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}
