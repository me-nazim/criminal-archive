// Package crimetypes exposes the curated list of crime categories used
// to tag cases. Public list is read-only; admin mutations come later.
package crimetypes

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CrimeType mirrors a row in crime_types.
type CrimeType struct {
	ID          int     `json:"id"`
	Slug        string  `json:"slug"`
	NameEN      string  `json:"name_en"`
	NameBN      string  `json:"name_bn"`
	Description *string `json:"description,omitempty"`
	Severity    int16   `json:"severity"`
}

// Repository wraps SQL access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListActive returns active crime types ordered by severity (high first)
// and then by Bangla name.
func (r *Repository) ListActive(ctx context.Context) ([]CrimeType, error) {
	const stmt = `
SELECT id, slug, name_en, name_bn, description, severity
FROM   crime_types
WHERE  is_active = TRUE
ORDER  BY severity DESC, name_bn`
	rows, err := r.pool.Query(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CrimeType, 0, 32)
	for rows.Next() {
		var c CrimeType
		if err := rows.Scan(&c.ID, &c.Slug, &c.NameEN, &c.NameBN, &c.Description, &c.Severity); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
