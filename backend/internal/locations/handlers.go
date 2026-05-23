package locations

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers wires Repository to chi routes.
type Handlers struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandlers constructs a new Handlers instance.
func NewHandlers(repo *Repository, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, logger: logger}
}

// Mount registers the public read-only location endpoints.
func (h *Handlers) Mount(r chi.Router) {
	r.With(cacheLong).Get("/locations/countries", h.listCountries)
	r.With(cacheLong).Get("/locations/divisions", h.listDivisions)
	r.With(cacheLong).Get("/locations/districts", h.listDistricts)
	r.With(cacheLong).Get("/locations/upazilas", h.listUpazilas)
}

func (h *Handlers) listCountries(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.ListCountries(r.Context())
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) listDivisions(w http.ResponseWriter, r *http.Request) {
	id, err := requireIntQuery(r, "country_id")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListDivisions(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) listDistricts(w http.ResponseWriter, r *http.Request) {
	id, err := requireIntQuery(r, "division_id")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListDistricts(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) listUpazilas(w http.ResponseWriter, r *http.Request) {
	id, err := requireIntQuery(r, "district_id")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListUpazilas(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func requireIntQuery(r *http.Request, name string) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return 0, httpx.ValidationError(map[string]string{name: "required"}, "")
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, httpx.ValidationError(map[string]string{name: "must be an integer"}, "")
	}
	if n <= 0 {
		return 0, httpx.ValidationError(map[string]string{name: "must be positive"}, "")
	}
	return n, nil
}

// cacheLong adds a long Cache-Control on responses for static reference data.
func cacheLong(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
		next.ServeHTTP(w, r)
	})
}
