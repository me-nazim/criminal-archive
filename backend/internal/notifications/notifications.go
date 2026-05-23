// Package notifications stores in-app notifications: a per-user feed
// rendered as a "bell" UI in the frontend. Email notifications are a
// separate concern handled by the email package; both can be triggered
// from the same upstream events.
package notifications

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Notification mirrors a row in `notifications`.
type Notification struct {
	ID        uuid.UUID       `json:"id"`
	Kind      string          `json:"kind"`
	Title     string          `json:"title"`
	Body      *string         `json:"body,omitempty"`
	Link      *string         `json:"link,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Repository is the SQL access layer.
type Repository struct{ pool *pgxpool.Pool }

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Create inserts a notification for a single user.
func (r *Repository) Create(ctx context.Context, userID uuid.UUID, kind, title, body, link string, metadata map[string]any) error {
	var meta []byte
	if len(metadata) > 0 {
		var err error
		if meta, err = json.Marshal(metadata); err != nil {
			return err
		}
	}
	const stmt = `
INSERT INTO notifications (user_id, kind, title, body, link, metadata)
VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), COALESCE($6, '{}'::jsonb))`
	_, err := r.pool.Exec(ctx, stmt, userID, kind, title, body, link, meta)
	return err
}

// List returns the user's most recent notifications, newest first.
func (r *Repository) List(ctx context.Context, userID uuid.UUID, limit int, unreadOnly bool) ([]Notification, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var args []any
	args = append(args, userID, limit)
	q := `
SELECT id, kind, title, body, link, metadata, read_at, created_at
FROM   notifications
WHERE  user_id = $1`
	if unreadOnly {
		q += ` AND read_at IS NULL`
	}
	q += ` ORDER BY created_at DESC LIMIT $2`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		var meta []byte
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.Link, &meta, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		if len(meta) > 0 {
			n.Metadata = meta
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var unread int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		userID).Scan(&unread); err != nil {
		return nil, 0, err
	}
	return out, unread, nil
}

// MarkRead marks one or all notifications for a user as read.
func (r *Repository) MarkRead(ctx context.Context, userID uuid.UUID, id *uuid.UUID) error {
	if id != nil {
		_, err := r.pool.Exec(ctx,
			`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND id = $2 AND read_at IS NULL`,
			userID, *id)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`,
		userID)
	return err
}

// Handlers exposes /api/v1/notifications endpoints.
type Handlers struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandlers constructs Handlers.
func NewHandlers(repo *Repository, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, logger: logger}
}

// Mount mounts the authenticated routes.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/notifications", h.list)
	r.Post("/notifications/read-all", h.readAll)
	r.Post("/notifications/{id}/read", h.read)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	id := auth.MustIdentity(r.Context())
	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	unreadOnly := r.URL.Query().Get("unread") == "true"
	rows, unread, err := h.repo.List(r.Context(), id.UserID, limit, unreadOnly)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows, "unread": unread})
}

func (h *Handlers) read(w http.ResponseWriter, r *http.Request) {
	id := auth.MustIdentity(r.Context())
	notifID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("invalid id"))
		return
	}
	if err := h.repo.MarkRead(r.Context(), id.UserID, &notifID); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handlers) readAll(w http.ResponseWriter, r *http.Request) {
	id := auth.MustIdentity(r.Context())
	if err := h.repo.MarkRead(r.Context(), id.UserID, nil); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}
