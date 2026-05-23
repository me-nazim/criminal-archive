package users

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers wires Service to chi routes.
type Handlers struct {
	svc    *Service
	logger *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(svc *Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// Mount registers admin user-management endpoints under /admin/users.
// The caller is responsible for adding RequireMinRole(admin).
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/users", h.list)
	r.Get("/users/{id}", h.get)
	r.Post("/users/{id}/approve", h.approve)
	r.Post("/users/{id}/reject", h.reject)
	r.Post("/users/{id}/suspend", h.suspend)
	r.Post("/users/{id}/reactivate", h.reactivate)
	r.Patch("/users/{id}/role", h.setRole)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	rows, err := h.svc.List(r.Context(), ListParams{
		Status: q.Get("status"),
		Role:   q.Get("role"),
		Search: q.Get("q"),
		Limit:  limit,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"data": rows,
		"page": map[string]any{"limit": len(rows)},
	})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

func (h *Handlers) transitionHandler(action func(svc *Service, actor auth.Identity, id uuid.UUID, ip, ua string) (*User, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err != nil {
			httpx.WriteError(w, r, h.logger, err)
			return
		}
		actor := auth.MustIdentity(r.Context())
		ip, ua := audit.FromRequest(r)
		u, err := action(h.svc, actor, id, ip, ua)
		if err != nil {
			httpx.WriteError(w, r, h.logger, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, u)
	}
}

func (h *Handlers) approve(w http.ResponseWriter, r *http.Request) {
	h.transitionHandler(func(s *Service, a auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
		return s.Approve(r.Context(), a, id, ip, ua)
	})(w, r)
}
func (h *Handlers) reject(w http.ResponseWriter, r *http.Request) {
	h.transitionHandler(func(s *Service, a auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
		return s.Reject(r.Context(), a, id, ip, ua)
	})(w, r)
}
func (h *Handlers) suspend(w http.ResponseWriter, r *http.Request) {
	h.transitionHandler(func(s *Service, a auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
		return s.Suspend(r.Context(), a, id, ip, ua)
	})(w, r)
}
func (h *Handlers) reactivate(w http.ResponseWriter, r *http.Request) {
	h.transitionHandler(func(s *Service, a auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
		return s.Reactivate(r.Context(), a, id, ip, ua)
	})(w, r)
}

func (h *Handlers) setRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	u, err := h.svc.SetRole(r.Context(), actor, id, req.Role, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

func parseID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.ValidationError(map[string]string{"id": "must be a uuid"}, "")
	}
	return id, nil
}
