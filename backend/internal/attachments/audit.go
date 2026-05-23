package attachments

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
)

// auditWriter is a Handlers helper that holds the pool needed by audit.Write.
type auditWriter struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// writeAudit forwards an Entry. Kept on Handlers (set during construction)
// so the package never exposes the pool publicly.
func (h *Handlers) writeAudit(ctx context.Context, e audit.Entry) {
	if h.audit == nil {
		return
	}
	audit.Write(ctx, h.audit.pool, e, h.audit.logger)
}

// SetAudit lets cmd/api wire the audit writer at construction time.
func (h *Handlers) SetAudit(pool *pgxpool.Pool, logger *slog.Logger) {
	h.audit = &auditWriter{pool: pool, logger: logger}
}
