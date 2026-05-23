package crimetypes

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers wires Repository to HTTP.
type Handlers struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandlers constructs Handlers.
func NewHandlers(repo *Repository, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, logger: logger}
}

// Mount registers public crime-type endpoints.
func (h *Handlers) Mount(r chi.Router) {
	r.With(cacheMedium).Get("/crime-types", h.list)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.ListActive(r.Context())
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func cacheMedium(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=86400")
		next.ServeHTTP(w, r)
	})
}
