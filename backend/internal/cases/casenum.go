package cases

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AllocateCaseNumber atomically increments the per-year counter and
// returns the canonical "TIP-YYYY-NNNNN" string. Caller passes a tx so
// the increment commits with the related case insert/update.
func AllocateCaseNumber(ctx context.Context, tx pgx.Tx, year int) (int, string, error) {
	const stmt = `
INSERT INTO case_number_counters (year, seq)
VALUES ($1, 1)
ON CONFLICT (year) DO UPDATE
SET seq = case_number_counters.seq + 1
RETURNING seq`
	var seq int
	if err := tx.QueryRow(ctx, stmt, year).Scan(&seq); err != nil {
		return 0, "", fmt.Errorf("allocate case_number: %w", err)
	}
	return seq, fmt.Sprintf("TIP-%04d-%05d", year, seq), nil
}
