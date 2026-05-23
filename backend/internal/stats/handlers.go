// Package stats serves a small aggregate-counts endpoint that powers
// the admin dashboard. The numbers are intentionally cheap to compute —
// COUNT(*) over indexed columns — so we can call this on every dashboard
// load without instrumenting a real metrics pipeline.
package stats

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers wraps the stats dependency.
type Handlers struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(pool *pgxpool.Pool, logger *slog.Logger) *Handlers {
	return &Handlers{pool: pool, logger: logger}
}

// Mount registers admin-only stats routes.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/stats", h.stats)
}

func (h *Handlers) stats(w http.ResponseWriter, r *http.Request) {
	caseCounts, err := groupCount(r.Context(), h.pool, "cases", "status")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	personCounts, err := groupCount(r.Context(), h.pool, "persons", "status")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	userCounts, err := groupCount(r.Context(), h.pool, "users", "status")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	var attachments int64
	_ = h.pool.QueryRow(r.Context(),
		`SELECT count(*) FROM case_attachments WHERE kind = 'public'`,
	).Scan(&attachments)

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"cases":            caseCounts,
		"persons":          personCounts,
		"users":            userCounts,
		"public_attachments": attachments,
	})
}

func groupCount(ctx context.Context, pool *pgxpool.Pool, table, col string) (map[string]int64, error) {
	stmt := "SELECT " + col + ", count(*) FROM " + table + " GROUP BY " + col
	rows, err := pool.Query(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] = n
	}
	return out, rows.Err()
}
