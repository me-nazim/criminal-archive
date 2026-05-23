// Package admincmd implements the bootstrap-admin and promote subcommands
// used to create or elevate the very first super-admin account during
// initial deployment. Once a super-admin exists in the database, future
// admin/user changes are made through the regular HTTP API.
package admincmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/auth"
)

// validRoles caps what the CLI is willing to set, mirroring the user_role
// enum in the database.
var validRoles = map[string]bool{
	"super_admin":  true,
	"admin":        true,
	"moderator":    true,
	"contributor":  true,
	"viewer":       true,
}

// CreateOrPromoteParams describes the CLI input.
type CreateOrPromoteParams struct {
	Email    string
	FullName string
	Password string
	Role     string
	BcryptCost int
}

// Run creates the user if missing, otherwise updates their role and
// password (when a new one is provided) and approves them. Idempotent.
func Run(ctx context.Context, pool *pgxpool.Pool, p CreateOrPromoteParams, logger *slog.Logger) error {
	email := strings.ToLower(strings.TrimSpace(p.Email))
	if email == "" {
		return errors.New("--email is required")
	}
	if !validRoles[p.Role] {
		return fmt.Errorf("invalid --role %q", p.Role)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		existingID   string
		existingRole string
	)
	err = tx.QueryRow(ctx, `SELECT id, role FROM users WHERE email = $1`, email).
		Scan(&existingID, &existingRole)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if p.Password == "" {
			return errors.New("user does not exist; --password is required to create one")
		}
		if p.FullName == "" {
			return errors.New("user does not exist; --name is required to create one")
		}
		hash, herr := auth.HashPassword(p.Password, p.BcryptCost)
		if herr != nil {
			return fmt.Errorf("hash password: %w", herr)
		}
		_, err = tx.Exec(ctx, `
INSERT INTO users (email, password_hash, full_name, role, status, approved_at)
VALUES ($1, $2, $3, $4, 'approved', now())`,
			email, hash, p.FullName, p.Role)
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}
		logger.Info("admin user created", "email", email, "role", p.Role)

	case err != nil:
		return fmt.Errorf("lookup user: %w", err)

	default:
		// Existing user: bump role + status, optionally rotate password.
		if p.Password != "" {
			hash, herr := auth.HashPassword(p.Password, p.BcryptCost)
			if herr != nil {
				return fmt.Errorf("hash password: %w", herr)
			}
			if _, err := tx.Exec(ctx,
				`UPDATE users SET password_hash = $2 WHERE id = $1`, existingID, hash); err != nil {
				return fmt.Errorf("update password: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
UPDATE users
SET    role        = $2,
       status      = 'approved',
       approved_at = COALESCE(approved_at, now())
WHERE  id = $1`, existingID, p.Role); err != nil {
			return fmt.Errorf("update role/status: %w", err)
		}
		// Revoke any active sessions so the user must re-login.
		if _, err := tx.Exec(ctx,
			`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
			existingID); err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		logger.Info("admin user updated", "email", email, "from_role", existingRole, "to_role", p.Role)
	}

	return tx.Commit(ctx)
}
