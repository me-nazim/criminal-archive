package persons

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers expose the persons API.
type Handlers struct {
	repo   *Repository
	svc    *Service
	logger *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(repo *Repository, svc *Service, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, svc: svc, logger: logger}
}

// MountPublic registers anonymous-readable endpoints.
func (h *Handlers) MountPublic(r chi.Router) {
	r.Get("/persons", h.list)
	r.Get("/persons/{slugOrID}", h.detail)
}

// MountAuthenticated registers endpoints that require a logged-in user.
func (h *Handlers) MountAuthenticated(r chi.Router) {
	r.Post("/persons", h.create)
	r.Patch("/persons/{id}", h.update)
	r.Get("/me/persons", h.listMine)
}

// MountAdmin registers admin-only routes.
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/persons", h.adminList)
	r.Post("/persons/{id}/approve", h.approve)
}

// ---------------------------------------------------------------- public

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	rows, err := h.repo.ListPublic(r.Context(), ListParams{
		PrimaryType: q.Get("primary_type"),
		Search:      q.Get("q"),
		Limit:       limit,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"data": redactList(rows, false),
	})
}

func (h *Handlers) detail(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "slugOrID")
	identity, _ := auth.IdentityFrom(r.Context())
	isAdmin := identity.Role == auth.RoleAdmin || identity.Role == auth.RoleSuperAdmin

	var p *Person
	var err error
	if id, parseErr := uuid.Parse(key); parseErr == nil {
		p, err = h.repo.GetByID(r.Context(), id)
	} else {
		p, err = h.repo.GetBySlug(r.Context(), key, !isAdmin)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, r, h.logger, httpx.NotFound("Person not found."))
			return
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if !isAdmin && p.Status != "published" {
		httpx.WriteError(w, r, h.logger, httpx.NotFound("Person not found."))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(p, isAdmin))
}

// ---------------------------------------------------------------- auth

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	p, err := h.svc.Create(r.Context(), actor, in, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, redactOne(p, true))
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	in, err := decodeUpdateInput(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	p, err := h.svc.Update(r.Context(), actor, id, in, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	isAdmin := actor.Role == auth.RoleAdmin || actor.Role == auth.RoleSuperAdmin
	httpx.WriteJSON(w, http.StatusOK, redactOne(p, isAdmin))
}

func (h *Handlers) listMine(w http.ResponseWriter, r *http.Request) {
	actor := auth.MustIdentity(r.Context())
	rows, err := h.repo.ListByOwner(r.Context(), actor.UserID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": redactList(rows, true)})
}

// ---------------------------------------------------------------- admin

func (h *Handlers) adminList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	status := q.Get("status")
	if status == "" {
		status = "pending_review"
	}
	rows, err := h.repo.ListPublic(r.Context(), ListParams{
		Status:      status,
		PrimaryType: q.Get("primary_type"),
		Search:      q.Get("q"),
		Limit:       limit,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": redactList(rows, true)})
}

func (h *Handlers) approve(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	p, err := h.svc.Approve(r.Context(), actor, id, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(p, true))
}

// ---------------------------------------------------------------- helpers

// redactOne returns a copy of the person with submitter / internal_notes
// removed for non-admin viewers, and identity blanked when is_anonymous.
func redactOne(p *Person, isAdmin bool) *Person {
	out := *p
	if !isAdmin {
		out.InternalNotes = nil
		out.SubmittedBy = nil
		out.ApprovedBy = nil
		if out.IsAnonymous {
			out.FullNameBN = nil
			out.FullNameEN = nil
			out.PhotoURL = nil
			out.DateOfBirth = nil
			out.AddressLine = nil
			out.Aliases = nil
		}
	}
	return &out
}

func redactList(rows []Person, isAdmin bool) []Person {
	out := make([]Person, len(rows))
	for i := range rows {
		out[i] = *redactOne(&rows[i], isAdmin)
	}
	return out
}

func parseID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.ValidationError(map[string]string{"id": "must be a uuid"}, "")
	}
	return id, nil
}

// decodeUpdateInput preserves the "field present vs absent" distinction
// that JSON.Unmarshal alone cannot express for pointers.
func decodeUpdateInput(r *http.Request) (UpdateInput, error) {
	// Decode into a generic map first, then carefully populate the typed
	// struct so we know which fields the client actually sent.
	var raw map[string]any
	if err := httpx.DecodeJSON(r, &raw); err != nil {
		return UpdateInput{}, err
	}
	in := UpdateInput{}

	get := func(k string) (any, bool) { v, ok := raw[k]; return v, ok }

	if v, ok := get("full_name_bn"); ok {
		in.HasFullNameBN = true
		in.FullNameBN = strPtr(v)
	}
	if v, ok := get("full_name_en"); ok {
		in.HasFullNameEN = true
		in.FullNameEN = strPtr(v)
	}
	if v, ok := get("aliases"); ok {
		if arr, ok2 := v.([]any); ok2 {
			s := make([]string, 0, len(arr))
			for _, a := range arr {
				if str, ok3 := a.(string); ok3 {
					s = append(s, str)
				}
			}
			in.Aliases = &s
		}
	}
	if v, ok := get("gender"); ok {
		in.HasGender = true
		in.Gender = strPtr(v)
	}
	if v, ok := get("photo_url"); ok {
		in.HasPhoto = true
		in.PhotoURL = strPtr(v)
	}
	if v, ok := get("occupation"); ok {
		in.HasOccupation = true
		in.Occupation = strPtr(v)
	}
	if v, ok := get("organization"); ok {
		in.HasOrg = true
		in.Organization = strPtr(v)
	}
	if v, ok := get("designation"); ok {
		in.HasDesg = true
		in.Designation = strPtr(v)
	}
	if v, ok := get("country_id"); ok {
		in.HasCountry = true
		in.CountryID = intPtr(v)
	}
	if v, ok := get("division_id"); ok {
		in.HasDivision = true
		in.DivisionID = intPtr(v)
	}
	if v, ok := get("district_id"); ok {
		in.HasDistrict = true
		in.DistrictID = intPtr(v)
	}
	if v, ok := get("upazila_id"); ok {
		in.HasUpazila = true
		in.UpazilaID = intPtr(v)
	}
	if v, ok := get("address_line"); ok {
		in.HasAddress = true
		in.AddressLine = strPtr(v)
	}
	if v, ok := get("public_bio_bn"); ok {
		in.HasPubBN = true
		in.PublicBioBN = strPtr(v)
	}
	if v, ok := get("public_bio_en"); ok {
		in.HasPubEN = true
		in.PublicBioEN = strPtr(v)
	}
	if v, ok := get("internal_notes"); ok {
		in.HasInternal = true
		in.InternalNotes = strPtr(v)
	}
	if v, ok := get("is_anonymous"); ok {
		if b, ok2 := v.(bool); ok2 {
			in.IsAnonymous = &b
		}
	}
	return in, nil
}

func strPtr(v any) *string {
	switch s := v.(type) {
	case string:
		return &s
	case nil:
		return nil
	default:
		return nil
	}
}

func intPtr(v any) *int {
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case nil:
		return nil
	default:
		return nil
	}
}
