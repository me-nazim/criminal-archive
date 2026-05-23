// Package verification owns the moderator-facing queue: which cases are
// currently assigned to the active user for evidence review, and the
// state transitions that come with their decision.
//
// The actual case status moves remain owned by the cases package
// (cases.Service.Transition); this package only manages the
// verification_assignments rows and the surface area exposed to a
// moderator.
package verification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Assignment is a flat join row of verification_assignments + the case
// fields a verifier needs to triage their queue.
type Assignment struct {
	ID           uuid.UUID  `json:"id"`
	CaseID       uuid.UUID  `json:"case_id"`
	CaseNumber   string     `json:"case_number"`
	CaseSlug     string     `json:"case_slug"`
	CaseTitleBN  string     `json:"case_title_bn"`
	CaseTitleEN  *string    `json:"case_title_en,omitempty"`
	CaseStatus   string     `json:"case_status"`
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	AssignedBy   *uuid.UUID `json:"assigned_by,omitempty"`
	Status       string     `json:"status"`
	Notes        *string    `json:"notes,omitempty"`
	AssignedAt   time.Time  `json:"assigned_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// Repository wraps SQL access for verification.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ErrNotFound is returned when an assignment lookup misses.
var ErrNotFound = errors.New("verification assignment not found")

// ListMine returns assignments owned by `userID`. When openOnly is true,
// only rows that have not yet been completed are returned.
func (r *Repository) ListMine(ctx context.Context, userID uuid.UUID, openOnly bool) ([]Assignment, error) {
	stmt := `
SELECT v.id, v.case_id, c.case_number, c.slug, c.title_bn, c.title_en, c.status,
       v.assigned_to, v.assigned_by, v.status, v.notes, v.assigned_at, v.completed_at
FROM   verification_assignments v
JOIN   cases c ON c.id = v.case_id
WHERE  v.assigned_to = $1`
	if openOnly {
		stmt += ` AND v.completed_at IS NULL`
	}
	stmt += ` ORDER BY v.assigned_at DESC`
	rows, err := r.pool.Query(ctx, stmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Assignment{}
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(
			&a.ID, &a.CaseID, &a.CaseNumber, &a.CaseSlug, &a.CaseTitleBN, &a.CaseTitleEN, &a.CaseStatus,
			&a.AssignedTo, &a.AssignedBy, &a.Status, &a.Notes, &a.AssignedAt, &a.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAll returns every assignment (admin view) optionally filtered by
// status. Useful for the admin verification dashboard.
func (r *Repository) ListAll(ctx context.Context, status string, limit int) ([]Assignment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{}
	stmt := `
SELECT v.id, v.case_id, c.case_number, c.slug, c.title_bn, c.title_en, c.status,
       v.assigned_to, v.assigned_by, v.status, v.notes, v.assigned_at, v.completed_at
FROM   verification_assignments v
JOIN   cases c ON c.id = v.case_id`
	if status != "" {
		stmt += ` WHERE v.status = $1`
		args = append(args, status)
	}
	stmt += fmt.Sprintf(` ORDER BY v.assigned_at DESC LIMIT $%d`, len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Assignment{}
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(
			&a.ID, &a.CaseID, &a.CaseNumber, &a.CaseSlug, &a.CaseTitleBN, &a.CaseTitleEN, &a.CaseStatus,
			&a.AssignedTo, &a.AssignedBy, &a.Status, &a.Notes, &a.AssignedAt, &a.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// MarkInProgress flips an assignment from `assigned` to `in_progress`.
// Idempotent: a no-op when the assignment is already in_progress.
func (r *Repository) MarkInProgress(ctx context.Context, assignmentID, assignee uuid.UUID) error {
	const stmt = `
UPDATE verification_assignments
SET    status = 'in_progress'
WHERE  id = $1
  AND  assigned_to = $2
  AND  completed_at IS NULL
  AND  status IN ('assigned', 'in_progress')`
	tag, err := r.pool.Exec(ctx, stmt, assignmentID, assignee)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Either it doesn't exist, isn't ours, or it's already completed.
		return ErrNotFound
	}
	return nil
}

// AppendNote appends a free-form note to an open assignment without
// completing it. Useful when a verifier wants to track progress.
func (r *Repository) AppendNote(ctx context.Context, assignmentID, assignee uuid.UUID, note string) error {
	const stmt = `
UPDATE verification_assignments
SET    notes = COALESCE(notes, '') ||
               CASE WHEN notes IS NULL OR notes = '' THEN '' ELSE E'\n---\n' END ||
               $3
WHERE  id = $1 AND assigned_to = $2 AND completed_at IS NULL`
	tag, err := r.pool.Exec(ctx, stmt, assignmentID, assignee, note)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Get fetches a single assignment by id.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Assignment, error) {
	const stmt = `
SELECT v.id, v.case_id, c.case_number, c.slug, c.title_bn, c.title_en, c.status,
       v.assigned_to, v.assigned_by, v.status, v.notes, v.assigned_at, v.completed_at
FROM   verification_assignments v
JOIN   cases c ON c.id = v.case_id
WHERE  v.id = $1`
	var a Assignment
	err := r.pool.QueryRow(ctx, stmt, id).Scan(
		&a.ID, &a.CaseID, &a.CaseNumber, &a.CaseSlug, &a.CaseTitleBN, &a.CaseTitleEN, &a.CaseStatus,
		&a.AssignedTo, &a.AssignedBy, &a.Status, &a.Notes, &a.AssignedAt, &a.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}
