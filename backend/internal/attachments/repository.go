// Package attachments owns the case_attachments table: the mapping
// between cases, R2 objects, and metadata (kind, sequence, captions).
package attachments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Attachment mirrors the case_attachments table.
type Attachment struct {
	ID               uuid.UUID  `json:"id"`
	CaseID           uuid.UUID  `json:"case_id"`
	Kind             string     `json:"kind"` // public | hidden | internal
	SequenceNo       int        `json:"sequence_no"`
	OriginalFilename string     `json:"original_filename"`
	StoredFilename   string     `json:"stored_filename"`
	StorageKey       string     `json:"storage_key"`
	PublicURL        *string    `json:"public_url,omitempty"`
	MimeType         string     `json:"mime_type"`
	SizeBytes        int64      `json:"size_bytes"`
	ChecksumSHA256   *string    `json:"checksum_sha256,omitempty"`
	Width            *int       `json:"width,omitempty"`
	Height           *int       `json:"height,omitempty"`
	DurationSeconds  *int       `json:"duration_seconds,omitempty"`
	CaptionBN        *string    `json:"caption_bn,omitempty"`
	CaptionEN        *string    `json:"caption_en,omitempty"`
	UploadedBy       *uuid.UUID `json:"uploaded_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// Repository wraps SQL access for attachments.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// AuditExec lets the handler layer write audit rows that need to be
// transactionally close to an attachment mutation without leaking the
// pool out of this package.
func (r *Repository) AuditExec(ctx context.Context, sql string, args ...any) error {
	_, err := r.pool.Exec(ctx, sql, args...)
	return err
}

// Sentinel errors.
var ErrNotFound = errors.New("attachment not found")

// AllocateSequence reserves the next sequence_no for a (case_id, kind)
// pair. We do this by holding a transaction-scoped advisory lock and
// then computing MAX(sequence_no)+1 in the same transaction.
//
// Returning the slot via SELECT...FOR UPDATE on the cases row would also
// work, but advisory locks let us serialise allocation only on the
// (case_id, kind) we actually care about.
func (r *Repository) AllocateSequence(ctx context.Context, caseID uuid.UUID, kind string) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Hash the (case_id, kind) pair into an int64 for pg_advisory_xact_lock.
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0), hashtextextended($2, 0))`,
		caseID.String(), kind); err != nil {
		return 0, fmt.Errorf("advisory lock: %w", err)
	}
	var seq int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(sequence_no), 0) + 1
		 FROM   case_attachments
		 WHERE  case_id = $1 AND kind = $2`,
		caseID, kind).Scan(&seq); err != nil {
		return 0, fmt.Errorf("compute seq: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return seq, nil
}

// CreateParams collects fields needed to record a finalised attachment.
type CreateParams struct {
	CaseID           uuid.UUID
	Kind             string
	SequenceNo       int
	OriginalFilename string
	StoredFilename   string
	StorageKey       string
	PublicURL        *string
	MimeType         string
	SizeBytes        int64
	ChecksumSHA256   *string
	UploadedBy       *uuid.UUID
}

// Create inserts a new attachment row.
func (r *Repository) Create(ctx context.Context, p CreateParams) (*Attachment, error) {
	const stmt = `
INSERT INTO case_attachments (
  case_id, kind, sequence_no, original_filename, stored_filename, storage_key,
  public_url, mime_type, size_bytes, checksum_sha256, uploaded_by
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING ` + columns
	row := r.pool.QueryRow(ctx, stmt,
		p.CaseID, p.Kind, p.SequenceNo, p.OriginalFilename, p.StoredFilename, p.StorageKey,
		p.PublicURL, p.MimeType, p.SizeBytes, p.ChecksumSHA256, p.UploadedBy,
	)
	return scan(row)
}

// GetByID returns an attachment.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Attachment, error) {
	const stmt = `SELECT ` + columns + ` FROM case_attachments WHERE id = $1`
	return scan(r.pool.QueryRow(ctx, stmt, id))
}

// ListByCase returns all attachments for a case. When publicOnly is true,
// only kind='public' rows are returned (used by anonymous viewers).
func (r *Repository) ListByCase(ctx context.Context, caseID uuid.UUID, publicOnly bool) ([]Attachment, error) {
	stmt := `SELECT ` + columns + ` FROM case_attachments WHERE case_id = $1`
	if publicOnly {
		stmt += ` AND kind = 'public'`
	}
	stmt += ` ORDER BY kind, sequence_no`
	rows, err := r.pool.Query(ctx, stmt, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// UpdateMetadata sets caption + kind on an existing attachment. Used by
// admin edits.
func (r *Repository) UpdateMetadata(
	ctx context.Context, id uuid.UUID, kind *string, captionBN, captionEN *string,
) (*Attachment, error) {
	sets := []string{}
	args := []any{id}
	idx := 2
	add := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	if kind != nil {
		add("kind", *kind)
	}
	if captionBN != nil {
		add("caption_bn", *captionBN)
	}
	if captionEN != nil {
		add("caption_en", *captionEN)
	}
	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}
	stmt := "UPDATE case_attachments SET " + strings.Join(sets, ", ") +
		" WHERE id = $1 RETURNING " + columns
	return scan(r.pool.QueryRow(ctx, stmt, args...))
}

// Delete removes the attachment row. The R2 object is deleted by the
// caller (service layer) so we do not hold an SDK dependency here.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) (*Attachment, error) {
	const stmt = `DELETE FROM case_attachments WHERE id = $1 RETURNING ` + columns
	return scan(r.pool.QueryRow(ctx, stmt, id))
}

// --------- helpers ---------

const columns = `id, case_id, kind, sequence_no, original_filename, stored_filename,
  storage_key, public_url, mime_type, size_bytes, checksum_sha256,
  width, height, duration_seconds, caption_bn, caption_en, uploaded_by, created_at`

func scan(row pgx.Row) (*Attachment, error) {
	a := &Attachment{}
	if err := row.Scan(
		&a.ID, &a.CaseID, &a.Kind, &a.SequenceNo, &a.OriginalFilename, &a.StoredFilename,
		&a.StorageKey, &a.PublicURL, &a.MimeType, &a.SizeBytes, &a.ChecksumSHA256,
		&a.Width, &a.Height, &a.DurationSeconds, &a.CaptionBN, &a.CaptionEN,
		&a.UploadedBy, &a.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}
