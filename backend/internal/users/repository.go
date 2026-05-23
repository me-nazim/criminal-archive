// Package users contains admin-side user management:
// listing, approving, rejecting, suspending, and changing roles.
package users

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User mirrors the public-facing fields we expose to admins.
type User struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	DisplayName *string    `json:"display_name,omitempty"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	Phone       *string    `json:"phone,omitempty"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	Bio         *string    `json:"bio,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListParams holds optional filters for List.
type ListParams struct {
	Status  string
	Role    string
	Search  string
	Limit   int
	Cursor  *time.Time // pagination cursor: created_at less than
	CursorID *uuid.UUID
}

// Repository wraps SQL access for user admin endpoints.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ErrUserNotFound is returned when a target user does not exist.
var ErrUserNotFound = errors.New("user not found")

// List returns users matching the filters. Pagination is by (created_at desc, id desc).
func (r *Repository) List(ctx context.Context, p ListParams) ([]User, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}

	args := make([]any, 0, 6)
	conds := make([]string, 0, 4)
	idx := 1

	if p.Status != "" {
		conds = append(conds, "status = $"+itoa(idx))
		args = append(args, p.Status)
		idx++
	}
	if p.Role != "" {
		conds = append(conds, "role = $"+itoa(idx))
		args = append(args, p.Role)
		idx++
	}
	if p.Search != "" {
		conds = append(conds, "(email ILIKE $"+itoa(idx)+" OR full_name ILIKE $"+itoa(idx)+")")
		args = append(args, "%"+p.Search+"%")
		idx++
	}
	if p.Cursor != nil && p.CursorID != nil {
		conds = append(conds, "(created_at, id) < ($"+itoa(idx)+", $"+itoa(idx+1)+")")
		args = append(args, *p.Cursor, *p.CursorID)
		idx += 2
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	stmt := `
SELECT id, email, full_name, display_name, role, status, phone, avatar_url, bio,
       last_login_at, approved_at, created_at, updated_at
FROM   users ` + where + `
ORDER  BY created_at DESC, id DESC
LIMIT  $` + itoa(idx)
	args = append(args, p.Limit)

	rows, err := r.pool.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]User, 0, p.Limit)
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FullName, &u.DisplayName, &u.Role, &u.Status,
			&u.Phone, &u.AvatarURL, &u.Bio, &u.LastLoginAt, &u.ApprovedAt,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Get returns a single user.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	const stmt = `
SELECT id, email, full_name, display_name, role, status, phone, avatar_url, bio,
       last_login_at, approved_at, created_at, updated_at
FROM   users WHERE id = $1`
	row := r.pool.QueryRow(ctx, stmt, id)
	var u User
	if err := row.Scan(
		&u.ID, &u.Email, &u.FullName, &u.DisplayName, &u.Role, &u.Status,
		&u.Phone, &u.AvatarURL, &u.Bio, &u.LastLoginAt, &u.ApprovedAt,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// SetStatus moves the user into the given status. Allowed transitions
// are enforced by the service layer.
func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, status string, approvedBy *uuid.UUID) error {
	const stmt = `
UPDATE users
SET    status      = $2,
       approved_by = COALESCE($3, approved_by),
       approved_at = CASE WHEN $2 = 'approved' AND approved_at IS NULL THEN now() ELSE approved_at END
WHERE  id = $1`
	tag, err := r.pool.Exec(ctx, stmt, id, status, approvedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetRole changes the user's role.
func (r *Repository) SetRole(ctx context.Context, id uuid.UUID, role string) error {
	const stmt = `UPDATE users SET role = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, stmt, id, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Helper. We avoid pulling in strconv here just for one digit.
func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
