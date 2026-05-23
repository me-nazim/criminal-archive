package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Entry is a row from audit_logs (read view; the write side lives in
// audit.go). Kept on the same package so the JSON shape stays consistent.
type Row struct {
	ID         int64           `json:"id"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	Action     string          `json:"action"`
	TargetType *string         `json:"target_type,omitempty"`
	TargetID   *string         `json:"target_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	IPAddress  *string         `json:"ip_address,omitempty"`
	UserAgent  *string         `json:"user_agent,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Repository wraps SQL access for the audit_logs table.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListParams holds the supported filters for List.
type ListParams struct {
	UserID     *uuid.UUID
	Action     string // exact match; empty = any
	TargetType string
	TargetID   string
	Since      *time.Time
	Until      *time.Time
	Limit      int
	Cursor     *int64 // id < cursor
}

// List returns audit rows matching p, newest first, paginated by id.
func (r *Repository) List(ctx context.Context, p ListParams) ([]Row, error) {
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	conds := []string{}
	args := []any{}
	idx := 1
	add := func(cond string, val any) {
		conds = append(conds, cond)
		args = append(args, val)
		idx++
	}
	if p.UserID != nil {
		add("user_id = $"+itoa(idx), *p.UserID)
	}
	if p.Action != "" {
		add("action = $"+itoa(idx), p.Action)
	}
	if p.TargetType != "" {
		add("target_type = $"+itoa(idx), p.TargetType)
	}
	if p.TargetID != "" {
		add("target_id = $"+itoa(idx), p.TargetID)
	}
	if p.Since != nil {
		add("created_at >= $"+itoa(idx), *p.Since)
	}
	if p.Until != nil {
		add("created_at <= $"+itoa(idx), *p.Until)
	}
	if p.Cursor != nil {
		add("id < $"+itoa(idx), *p.Cursor)
	}
	stmt := `
SELECT id, user_id, action, target_type, target_id, metadata,
       host(ip_address)::text, user_agent, created_at
FROM   audit_logs`
	if len(conds) > 0 {
		stmt += " WHERE " + strings.Join(conds, " AND ")
	}
	stmt += " ORDER BY id DESC LIMIT $" + itoa(idx)
	args = append(args, p.Limit)

	rows, err := r.pool.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Row{}
	for rows.Next() {
		var rr Row
		var meta []byte
		var ip, ua *string
		if err := rows.Scan(&rr.ID, &rr.UserID, &rr.Action, &rr.TargetType, &rr.TargetID,
			&meta, &ip, &ua, &rr.CreatedAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			rr.Metadata = meta
		}
		rr.IPAddress = ip
		rr.UserAgent = ua
		out = append(out, rr)
	}
	return out, rows.Err()
}

// Handlers exposes admin-facing audit log endpoints.
type Handlers struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(repo *Repository, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, logger: logger}
}

// Mount registers /admin/audit (caller is responsible for the RBAC
// wrapper).
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/audit", h.list)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	var cursor *int64
	if c := q.Get("cursor"); c != "" {
		if n, err := strconv.ParseInt(c, 10, 64); err == nil {
			cursor = &n
		}
	}
	var userID *uuid.UUID
	if uid := q.Get("user_id"); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			userID = &id
		}
	}
	var since, until *time.Time
	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = &t
		}
	}
	if u := q.Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = &t
		}
	}
	rows, err := h.repo.List(r.Context(), ListParams{
		UserID:     userID,
		Action:     q.Get("action"),
		TargetType: q.Get("target_type"),
		TargetID:   q.Get("target_id"),
		Since:      since,
		Until:      until,
		Limit:      limit,
		Cursor:     cursor,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var nextCursor *int64
	if len(rows) > 0 {
		c := rows[len(rows)-1].ID
		nextCursor = &c
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"data": rows,
		"page": map[string]any{
			"limit":       len(rows),
			"next_cursor": nextCursor,
		},
	})
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	return string(buf[pos:])
}
