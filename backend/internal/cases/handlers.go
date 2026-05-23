package cases

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers expose the cases REST API.
type Handlers struct {
	repo   *Repository
	svc    *Service
	logger *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(repo *Repository, svc *Service, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, svc: svc, logger: logger}
}

// MountPublic mounts public read endpoints.
func (h *Handlers) MountPublic(r chi.Router) {
	r.Get("/cases", h.list)
	r.Get("/cases/{key}", h.detail)
	r.Get("/cases/{key}/persons", h.detailPersons)
	r.Get("/cases/{key}/timeline", h.detailTimeline)
	r.Get("/cases/{key}/news-sources", h.detailNews)
	r.Get("/persons/{slugOrID}/cases", h.casesForPerson)
}

// MountAuthenticated mounts authenticated routes (contributors+).
func (h *Handlers) MountAuthenticated(r chi.Router) {
	r.Post("/cases", h.create)
	r.Get("/me/cases", h.listMine)
	r.Get("/me/cases/{id}", h.detailAdmin) // an owner needs full case details too
	r.Patch("/cases/{id}", h.patch)
	r.Post("/cases/{id}/submit", h.submit)
	r.Post("/cases/{id}/persons", h.addPerson)
	r.Delete("/cases/{id}/persons/{personId}/{role}", h.removePerson)
	r.Post("/cases/{id}/timeline", h.addTimeline)
	r.Post("/cases/{id}/news-sources", h.addNews)
}

// MountAdmin mounts admin/moderator routes.
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/cases", h.adminList)
	r.Get("/cases/{id}", h.detailAdmin)
	r.Post("/cases/{id}/assign", h.assign)
	r.Post("/cases/{id}/verify", h.verify)
	r.Post("/cases/{id}/publish", h.publish)
	r.Post("/cases/{id}/unpublish", h.unpublish)
	r.Delete("/cases/{id}", h.delete)
}

// ------------------------------------------------------------ public list

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	year, _ := strconv.Atoi(q.Get("year"))
	rows, err := h.repo.ListPublic(r.Context(), ListParams{
		CountryID:   atoi(q.Get("country_id")),
		DivisionID:  atoi(q.Get("division_id")),
		DistrictID:  atoi(q.Get("district_id")),
		UpazilaID:   atoi(q.Get("upazila_id")),
		CrimeTypeID: atoi(q.Get("crime_type_id")),
		Year:        year,
		Search:      q.Get("q"),
		Tag:         q.Get("tag"),
		SortByDate:  q.Get("sort") == "incident_desc",
		Limit:       limit,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": redactList(rows, false)})
}

func (h *Handlers) detail(w http.ResponseWriter, r *http.Request) {
	c, isAdmin, err := h.resolveCase(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if !isAdmin && c.Status != "published" {
		httpx.WriteError(w, r, h.logger, httpx.NotFound("Case not found."))
		return
	}
	go h.repo.IncrementViewCount(r.Context(), c.ID)

	persons, _ := h.repo.ListPersonsForCase(r.Context(), c.ID)
	timeline, _ := h.repo.ListTimeline(r.Context(), c.ID, isAdmin)
	news, _ := h.repo.ListNewsSources(r.Context(), c.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"case":         redactOne(c, isAdmin),
		"persons":      persons,
		"timeline":     timeline,
		"news_sources": news,
	})
}

func (h *Handlers) detailPersons(w http.ResponseWriter, r *http.Request) {
	c, _, err := h.resolveCase(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListPersonsForCase(r.Context(), c.ID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) detailTimeline(w http.ResponseWriter, r *http.Request) {
	c, isAdmin, err := h.resolveCase(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListTimeline(r.Context(), c.ID, isAdmin)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) detailNews(w http.ResponseWriter, r *http.Request) {
	c, _, err := h.resolveCase(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListNewsSources(r.Context(), c.ID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) casesForPerson(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "slugOrID")
	pid, perr := uuid.Parse(key)
	if perr != nil {
		resolved, err := h.repo.LookupPersonIDBySlug(r.Context(), key)
		if err != nil {
			httpx.WriteError(w, r, h.logger, httpx.NotFound("Person not found."))
			return
		}
		pid = resolved
	}
	rows, err := h.repo.ListCasesForPerson(r.Context(), pid, true)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": redactList(rows, false)})
}

// ------------------------------------------------------------ auth

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	c, err := h.svc.Create(r.Context(), actor, in, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, redactOne(c, true))
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

func (h *Handlers) patch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	c, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err, "Case not found."))
		return
	}
	actor := auth.MustIdentity(r.Context())
	owner := c.SubmittedBy != nil && *c.SubmittedBy == actor.UserID
	rank := roleRank(actor.Role)
	if !owner && rank < roleRank(auth.RoleModerator) {
		httpx.WriteError(w, r, h.logger, httpx.Forbidden("forbidden", ""))
		return
	}

	var raw map[string]any
	if err := httpx.DecodeJSON(r, &raw); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	allowed := map[string]bool{
		"title_bn": true, "title_en": true,
		"summary_bn": true, "summary_en": true,
		"description_bn": true, "description_en": true,
		"incident_date": true, "incident_time": true,
		"country_id": true, "division_id": true, "district_id": true, "upazila_id": true,
		"location_text": true, "crime_type_id": true, "case_status": true, "severity": true,
		"tags": true, "cover_image_url": true,
	}
	if rank >= roleRank(auth.RoleAdmin) {
		allowed["internal_notes"] = true
	}
	set := map[string]any{}
	for k, v := range raw {
		if allowed[k] {
			set[k] = v
		}
	}
	if len(set) == 0 {
		httpx.WriteJSON(w, http.StatusOK, redactOne(c, rank >= roleRank(auth.RoleAdmin)))
		return
	}
	ip, ua := audit.FromRequest(r)
	updated, err := h.svc.PatchFields(r.Context(), actor, id, set, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(updated, rank >= roleRank(auth.RoleAdmin)))
}

func (h *Handlers) submit(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	c, err := h.svc.Transition(r.Context(), actor, id, "pending_review", audit.ActionCaseSubmit, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(c, true))
}

func (h *Handlers) addPerson(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		PersonID string  `json:"person_id"`
		Role     string  `json:"role"`
		Notes    *string `json:"notes"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	pid, err := uuid.Parse(body.PersonID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"person_id": "uuid"}, ""))
		return
	}
	if body.Role == "" {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"role": "required"}, ""))
		return
	}
	if err := h.repo.AddPerson(r.Context(), caseID, pid, body.Role, body.Notes); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handlers) removePerson(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	pid, err := uuid.Parse(chi.URLParam(r, "personId"))
	if err != nil {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"personId": "uuid"}, ""))
		return
	}
	role := chi.URLParam(r, "role")
	if err := h.repo.RemovePerson(r.Context(), caseID, pid, role); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handlers) addTimeline(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		EventDate     time.Time `json:"event_date"`
		EventTime     *string   `json:"event_time"`
		TitleBN       string    `json:"title_bn"`
		TitleEN       *string   `json:"title_en"`
		DescriptionBN *string   `json:"description_bn"`
		DescriptionEN *string   `json:"description_en"`
		SourceURL     *string   `json:"source_url"`
		IsInternal    bool      `json:"is_internal"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	if body.IsInternal && roleRank(actor.Role) < roleRank(auth.RoleAdmin) {
		httpx.WriteError(w, r, h.logger, httpx.Forbidden("forbidden", "Only admins can add internal timeline events."))
		return
	}
	ev, err := h.repo.AddTimelineEvent(r.Context(), TimelineEvent{
		CaseID: caseID, EventDate: body.EventDate, EventTime: body.EventTime,
		TitleBN: body.TitleBN, TitleEN: body.TitleEN,
		DescriptionBN: body.DescriptionBN, DescriptionEN: body.DescriptionEN,
		SourceURL: body.SourceURL, IsInternal: body.IsInternal, CreatedBy: &actor.UserID,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, ev)
}

func (h *Handlers) addNews(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		URL         string     `json:"url"`
		Title       *string    `json:"title"`
		SourceName  *string    `json:"source_name"`
		PublishedAt *time.Time `json:"published_at"`
		ArchivedURL *string    `json:"archived_url"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if body.URL == "" {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"url": "required"}, ""))
		return
	}
	src, err := h.repo.AddNewsSource(r.Context(), NewsSource{
		CaseID: caseID, URL: body.URL, Title: body.Title, SourceName: body.SourceName,
		PublishedAt: body.PublishedAt, ArchivedURL: body.ArchivedURL,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, src)
}

// ------------------------------------------------------------ admin

func (h *Handlers) adminList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	year, _ := strconv.Atoi(q.Get("year"))
	status := q.Get("status")
	if status == "" {
		status = "pending_review"
	}
	rows, err := h.repo.ListPublic(r.Context(), ListParams{
		Status:      status,
		CountryID:   atoi(q.Get("country_id")),
		DivisionID:  atoi(q.Get("division_id")),
		DistrictID:  atoi(q.Get("district_id")),
		UpazilaID:   atoi(q.Get("upazila_id")),
		CrimeTypeID: atoi(q.Get("crime_type_id")),
		Year:        year,
		Search:      q.Get("q"),
		Tag:         q.Get("tag"),
		Limit:       limit,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": redactList(rows, true)})
}

func (h *Handlers) detailAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	c, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err, "Case not found."))
		return
	}
	persons, _ := h.repo.ListPersonsForCase(r.Context(), c.ID)
	timeline, _ := h.repo.ListTimeline(r.Context(), c.ID, true)
	news, _ := h.repo.ListNewsSources(r.Context(), c.ID)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"case":         redactOne(c, true),
		"persons":      persons,
		"timeline":     timeline,
		"news_sources": news,
	})
}

func (h *Handlers) assign(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		AssigneeID string `json:"assignee_id"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	assignee, err := uuid.Parse(body.AssigneeID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"assignee_id": "uuid"}, ""))
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	if err := h.repo.AssignVerification(r.Context(), id, assignee, actor.UserID); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	c, err := h.svc.Transition(r.Context(), actor, id, "in_verification", audit.ActionCaseAssign, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(c, true))
}

func (h *Handlers) verify(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		Decision string  `json:"decision"`
		Reason   *string `json:"reason"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	to := "approved"
	verifStatus := "verified"
	if body.Decision == "rejected" {
		to = "rejected"
		verifStatus = "rejected"
	} else if body.Decision != "verified" {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"decision": "verified|rejected"}, ""))
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	if err := h.repo.CompleteVerification(r.Context(), id, actor.UserID, verifStatus, body.Reason); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	c, err := h.svc.Transition(r.Context(), actor, id, to, audit.ActionCaseVerify, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(c, true))
}

func (h *Handlers) publish(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	c, err := h.svc.Transition(r.Context(), actor, id, "published", audit.ActionCasePublish, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(c, true))
}

func (h *Handlers) unpublish(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	c, err := h.svc.Transition(r.Context(), actor, id, "archived", audit.ActionCaseUnpublish, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redactOne(c, true))
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	c, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err, "Case not found."))
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	h.svc.WriteAudit(r.Context(), audit.Entry{
		UserID: &actor.UserID, Action: audit.ActionCaseDelete,
		TargetType: "case", TargetID: id.String(),
		IP: ip, UserAgent: ua,
		Metadata: map[string]any{"case_number": c.CaseNumber},
	})
	httpx.NoContent(w)
}

// ------------------------------------------------------------ helpers

func (h *Handlers) resolveCase(r *http.Request) (*Case, bool, error) {
	key := chi.URLParam(r, "key")
	c, err := h.repo.GetByKey(r.Context(), key)
	if err != nil {
		return nil, false, mapNotFound(err, "Case not found.")
	}
	id, _ := auth.IdentityFrom(r.Context())
	isAdmin := id.Role == auth.RoleAdmin || id.Role == auth.RoleSuperAdmin
	return c, isAdmin, nil
}

func parseID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.ValidationError(map[string]string{"id": "must be a uuid"}, "")
	}
	return id, nil
}

func mapNotFound(err error, msg string) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound(msg)
	}
	return err
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func roleRank(r auth.Role) int {
	switch r {
	case auth.RoleSuperAdmin:
		return 5
	case auth.RoleAdmin:
		return 4
	case auth.RoleModerator:
		return 3
	case auth.RoleContributor:
		return 2
	case auth.RoleViewer:
		return 1
	}
	return 0
}

// redactOne returns a copy of the case with admin-only fields removed
// for non-admin viewers.
func redactOne(c *Case, isAdmin bool) *Case {
	out := *c
	if !isAdmin {
		out.InternalNotes = nil
		out.SubmittedBy = nil
		out.ApprovedBy = nil
	}
	return &out
}

func redactList(rows []Case, isAdmin bool) []Case {
	out := make([]Case, len(rows))
	for i := range rows {
		out[i] = *redactOne(&rows[i], isAdmin)
	}
	return out
}
