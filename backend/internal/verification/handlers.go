package verification

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers expose the moderator-facing verification routes.
type Handlers struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(repo *Repository, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, logger: logger}
}

// MountAuthenticated registers /verification routes for any moderator+.
func (h *Handlers) MountAuthenticated(r chi.Router) {
	r.With(auth.RequireMinRole(auth.RoleModerator)).Get("/verification/queue", h.listMine)
	r.With(auth.RequireMinRole(auth.RoleModerator)).Post("/verification/{id}/start", h.start)
	r.With(auth.RequireMinRole(auth.RoleModerator)).Post("/verification/{id}/notes", h.appendNote)
}

// MountAdmin registers admin-only verification overview routes.
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/verification", h.listAll)
}

// ---------- handlers ----------

func (h *Handlers) listMine(w http.ResponseWriter, r *http.Request) {
	actor := auth.MustIdentity(r.Context())
	openOnly := r.URL.Query().Get("open") != "false"
	rows, err := h.repo.ListMine(r.Context(), actor.UserID, openOnly)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) listAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := h.repo.ListAll(r.Context(), q.Get("status"), 50)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) start(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	if err := h.repo.MarkInProgress(r.Context(), id, actor.UserID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, r, h.logger, httpx.NotFound("Assignment not found or not yours."))
			return
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handlers) appendNote(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if body.Note == "" {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"note": "required"}, ""))
		return
	}
	actor := auth.MustIdentity(r.Context())
	if err := h.repo.AppendNote(r.Context(), id, actor.UserID, body.Note); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, r, h.logger, httpx.NotFound("Assignment not found or not yours."))
			return
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}

func parseID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.ValidationError(map[string]string{"id": "must be a uuid"}, "")
	}
	return id, nil
}
