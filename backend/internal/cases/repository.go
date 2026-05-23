package cases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Case is the wire-shape of a case record. Localised text is exposed as
// nested objects ({bn, en}) at handler-layer; this struct stays flat for
// SQL convenience.
type Case struct {
	ID            uuid.UUID  `json:"id"`
	CaseNumber    string     `json:"case_number"`
	Slug          string     `json:"slug"`
	TitleBN       string     `json:"title_bn"`
	TitleEN       *string    `json:"title_en,omitempty"`
	SummaryBN     *string    `json:"summary_bn,omitempty"`
	SummaryEN     *string    `json:"summary_en,omitempty"`
	DescriptionBN *string    `json:"description_bn,omitempty"`
	DescriptionEN *string    `json:"description_en,omitempty"`
	InternalNotes *string    `json:"internal_notes,omitempty"`

	IncidentDate *time.Time `json:"incident_date,omitempty"`
	IncidentTime *string    `json:"incident_time,omitempty"`

	CountryID    *int    `json:"country_id,omitempty"`
	DivisionID   *int    `json:"division_id,omitempty"`
	DistrictID   *int    `json:"district_id,omitempty"`
	UpazilaID    *int    `json:"upazila_id,omitempty"`
	LocationText *string `json:"location_text,omitempty"`

	CrimeTypeID *int    `json:"crime_type_id,omitempty"`
	CaseStatus  *string `json:"case_status,omitempty"`
	Severity    *int16  `json:"severity,omitempty"`

	CoverImageURL *string  `json:"cover_image_url,omitempty"`
	Tags          []string `json:"tags"`

	Status      string     `json:"status"`
	SubmittedBy *uuid.UUID `json:"submitted_by,omitempty"`
	ApprovedBy  *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`

	ViewCount     int64 `json:"view_count"`
	DownloadCount int64 `json:"download_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CasePersonLink is the join row in case_persons enriched with a couple
// of person fields the UI needs without an extra fetch.
type CasePersonLink struct {
	PersonID    uuid.UUID `json:"person_id"`
	Slug        string    `json:"person_slug"`
	Role        string    `json:"role"`
	IsAnonymous bool      `json:"is_anonymous"`
	NameBN      *string   `json:"name_bn,omitempty"`
	NameEN      *string   `json:"name_en,omitempty"`
	PhotoURL    *string   `json:"photo_url,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
}

// TimelineEvent mirrors a row in case_timeline.
type TimelineEvent struct {
	ID            uuid.UUID  `json:"id"`
	CaseID        uuid.UUID  `json:"case_id"`
	EventDate     time.Time  `json:"event_date"`
	EventTime     *string    `json:"event_time,omitempty"`
	TitleBN       string     `json:"title_bn"`
	TitleEN       *string    `json:"title_en,omitempty"`
	DescriptionBN *string    `json:"description_bn,omitempty"`
	DescriptionEN *string    `json:"description_en,omitempty"`
	SourceURL     *string    `json:"source_url,omitempty"`
	IsInternal    bool       `json:"is_internal"`
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// NewsSource mirrors a row in news_sources.
type NewsSource struct {
	ID          uuid.UUID  `json:"id"`
	CaseID      uuid.UUID  `json:"case_id"`
	URL         string     `json:"url"`
	Title       *string    `json:"title,omitempty"`
	SourceName  *string    `json:"source_name,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	ArchivedURL *string    `json:"archived_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Repository wraps SQL access for cases and their related rows.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Sentinel errors.
var (
	ErrNotFound = errors.New("case not found")
)

// CreateParams is what the service passes to Create.
type CreateParams struct {
	TitleBN       string
	TitleEN       *string
	SummaryBN     *string
	SummaryEN     *string
	DescriptionBN *string
	DescriptionEN *string
	IncidentDate  *time.Time
	IncidentTime  *string
	CountryID     *int
	DivisionID    *int
	DistrictID    *int
	UpazilaID     *int
	LocationText  *string
	CrimeTypeID   *int
	CaseStatus    *string
	Severity      *int16
	Tags          []string
	SubmittedBy   uuid.UUID
}

// Create inserts a draft case with an allocated case_number + slug.
func (r *Repository) Create(ctx context.Context, p CreateParams) (*Case, error) {
	year := time.Now().UTC().Year()
	if p.IncidentDate != nil {
		year = p.IncidentDate.Year()
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, caseNumber, err := AllocateCaseNumber(ctx, tx, year)
	if err != nil {
		return nil, err
	}
	slug := strings.ToLower(caseNumber)

	const stmt = `
INSERT INTO cases (
  case_number, slug, title_bn, title_en,
  summary_bn, summary_en, description_bn, description_en,
  incident_date, incident_time,
  country_id, division_id, district_id, upazila_id, location_text,
  crime_type_id, case_status, severity, tags,
  submitted_by, status
)
VALUES (
  $1,$2,$3,$4,
  $5,$6,$7,$8,
  $9,$10,
  $11,$12,$13,$14,$15,
  $16,$17,$18,$19,
  $20,'draft'
)
RETURNING ` + caseColumns
	row := tx.QueryRow(ctx, stmt,
		caseNumber, slug, p.TitleBN, p.TitleEN,
		p.SummaryBN, p.SummaryEN, p.DescriptionBN, p.DescriptionEN,
		p.IncidentDate, p.IncidentTime,
		p.CountryID, p.DivisionID, p.DistrictID, p.UpazilaID, p.LocationText,
		p.CrimeTypeID, p.CaseStatus, p.Severity, p.Tags,
		p.SubmittedBy,
	)
	c, err := scanCase(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateFieldsParams describes a partial update of plain case columns.
// Linked rows (persons, timeline, news) are managed via separate methods.
type UpdateFieldsParams struct {
	Set map[string]any
}

// UpdateFields applies a sparse field update by column name.
func (r *Repository) UpdateFields(ctx context.Context, id uuid.UUID, set map[string]any) (*Case, error) {
	if len(set) == 0 {
		return r.GetByID(ctx, id)
	}
	cols := make([]string, 0, len(set))
	args := []any{id}
	idx := 2
	for k, v := range set {
		cols = append(cols, fmt.Sprintf("%s = $%d", k, idx))
		args = append(args, v)
		idx++
	}
	stmt := "UPDATE cases SET " + strings.Join(cols, ", ") + " WHERE id = $1 RETURNING " + caseColumns
	row := r.pool.QueryRow(ctx, stmt, args...)
	return scanCase(row)
}

// SetStatus moves a case to a new status with bookkeeping (approved_by /
// approved_at / published_at).
func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, status string, approver *uuid.UUID) (*Case, error) {
	const stmt = `
UPDATE cases
SET    status       = $2,
       approved_by  = COALESCE($3, approved_by),
       approved_at  = CASE WHEN $2 IN ('approved','published') AND approved_at IS NULL THEN now() ELSE approved_at END,
       published_at = CASE WHEN $2 = 'published' AND published_at IS NULL THEN now() ELSE published_at END
WHERE  id = $1
RETURNING ` + caseColumns
	row := r.pool.QueryRow(ctx, stmt, id, status, approver)
	return scanCase(row)
}

// GetByID returns a case by its id.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Case, error) {
	const stmt = `SELECT ` + caseColumns + ` FROM cases WHERE id = $1`
	row := r.pool.QueryRow(ctx, stmt, id)
	return scanCase(row)
}

// GetByKey resolves a case by either uuid, slug, or case_number.
func (r *Repository) GetByKey(ctx context.Context, key string) (*Case, error) {
	if id, err := uuid.Parse(key); err == nil {
		return r.GetByID(ctx, id)
	}
	const stmt = `SELECT ` + caseColumns + ` FROM cases WHERE slug = $1 OR case_number = $1`
	row := r.pool.QueryRow(ctx, stmt, key)
	return scanCase(row)
}

// Delete removes a case (cascade clears case_persons, attachments, etc.).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM cases WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// IncrementViewCount bumps view_count by 1. Errors are non-fatal.
func (r *Repository) IncrementViewCount(ctx context.Context, id uuid.UUID) {
	_, _ = r.pool.Exec(ctx, `UPDATE cases SET view_count = view_count + 1 WHERE id = $1`, id)
}

// LookupPersonIDBySlug resolves a published person's id from its slug.
// Returns ErrNotFound when the slug is unknown or the person is not yet
// published.
func (r *Repository) LookupPersonIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM persons WHERE slug = $1 AND status = 'published'`, slug,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return id, nil
}

// AssignVerification creates a verification_assignments row.
func (r *Repository) AssignVerification(ctx context.Context, caseID, assignedTo, assignedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO verification_assignments (case_id, assigned_to, assigned_by, status)
		 VALUES ($1, $2, $3, 'assigned')`, caseID, assignedTo, assignedBy)
	return err
}

// CompleteVerification marks the active verification row for a case as
// finished with the given status and optional notes.
func (r *Repository) CompleteVerification(
	ctx context.Context, caseID, assignee uuid.UUID, status string, notes *string,
) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE verification_assignments
		 SET    status = $2, completed_at = now(), notes = $3
		 WHERE  case_id = $1 AND assigned_to = $4 AND completed_at IS NULL`,
		caseID, status, notes, assignee)
	return err
}

// AuditExec exposes a thin INSERT helper so the handler layer can write
// audit rows that need to live next to a case mutation. It exists only
// to avoid leaking the *pgxpool.Pool out of this package.
func (r *Repository) AuditExec(ctx context.Context, sql string, args ...any) error {
	_, err := r.pool.Exec(ctx, sql, args...)
	return err
}

// ListParams holds the supported filters for ListPublic.
type ListParams struct {
	Status        string // empty = published only
	CountryID     int
	DivisionID    int
	DistrictID    int
	UpazilaID     int
	CrimeTypeID   int
	Year          int
	Search        string
	Tag           string
	SortByDate    bool // true = sort by incident_date; default = published_at
	Limit         int
}

// ListPublic returns cases matching the filters, ordered by recency.
func (r *Repository) ListPublic(ctx context.Context, p ListParams) ([]Case, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 24
	}
	conds := []string{}
	args := []any{}
	idx := 1
	add := func(cond string, vals ...any) {
		conds = append(conds, cond)
		args = append(args, vals...)
		idx += len(vals)
	}
	if p.Status == "" {
		add("status = 'published'")
	} else {
		add(fmt.Sprintf("status = $%d", idx), p.Status)
	}
	if p.CountryID > 0 {
		add(fmt.Sprintf("country_id = $%d", idx), p.CountryID)
	}
	if p.DivisionID > 0 {
		add(fmt.Sprintf("division_id = $%d", idx), p.DivisionID)
	}
	if p.DistrictID > 0 {
		add(fmt.Sprintf("district_id = $%d", idx), p.DistrictID)
	}
	if p.UpazilaID > 0 {
		add(fmt.Sprintf("upazila_id = $%d", idx), p.UpazilaID)
	}
	if p.CrimeTypeID > 0 {
		add(fmt.Sprintf("crime_type_id = $%d", idx), p.CrimeTypeID)
	}
	if p.Year > 0 {
		add(fmt.Sprintf("EXTRACT(year FROM incident_date) = $%d", idx), p.Year)
	}
	if p.Search != "" {
		add(fmt.Sprintf("(title_bn ILIKE $%d OR title_en ILIKE $%d OR case_number ILIKE $%d)", idx, idx, idx), "%"+p.Search+"%")
	}
	if p.Tag != "" {
		add(fmt.Sprintf("$%d = ANY(tags)", idx), p.Tag)
	}
	order := "published_at DESC NULLS LAST, created_at DESC"
	if p.SortByDate {
		order = "incident_date DESC NULLS LAST, created_at DESC"
	}
	stmt := `SELECT ` + caseColumns + ` FROM cases WHERE ` + strings.Join(conds, " AND ") +
		` ORDER BY ` + order + ` LIMIT $` + fmt.Sprint(idx)
	args = append(args, p.Limit)

	rows, err := r.pool.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Case, 0, p.Limit)
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ListByOwner returns every case submitted by the given user (any status).
func (r *Repository) ListByOwner(ctx context.Context, userID uuid.UUID) ([]Case, error) {
	const stmt = `SELECT ` + caseColumns + ` FROM cases WHERE submitted_by = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, stmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Case{}
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// --------------- case_persons ---------------

// AddPerson links a person to a case in the given role. Idempotent.
func (r *Repository) AddPerson(ctx context.Context, caseID, personID uuid.UUID, role string, notes *string) error {
	const stmt = `
INSERT INTO case_persons (case_id, person_id, role, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (case_id, person_id, role) DO UPDATE SET notes = EXCLUDED.notes`
	_, err := r.pool.Exec(ctx, stmt, caseID, personID, role, notes)
	return err
}

// RemovePerson unlinks a person from a case.
func (r *Repository) RemovePerson(ctx context.Context, caseID, personID uuid.UUID, role string) error {
	const stmt = `DELETE FROM case_persons WHERE case_id = $1 AND person_id = $2 AND role = $3`
	_, err := r.pool.Exec(ctx, stmt, caseID, personID, role)
	return err
}

// ListPersonsForCase returns person links enriched with display fields.
func (r *Repository) ListPersonsForCase(ctx context.Context, caseID uuid.UUID) ([]CasePersonLink, error) {
	const stmt = `
SELECT cp.person_id, p.slug, cp.role, p.is_anonymous,
       p.full_name_bn, p.full_name_en, p.photo_url, cp.notes
FROM   case_persons cp
JOIN   persons p ON p.id = cp.person_id
WHERE  cp.case_id = $1
ORDER  BY cp.created_at`
	rows, err := r.pool.Query(ctx, stmt, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CasePersonLink{}
	for rows.Next() {
		var l CasePersonLink
		if err := rows.Scan(&l.PersonID, &l.Slug, &l.Role, &l.IsAnonymous,
			&l.NameBN, &l.NameEN, &l.PhotoURL, &l.Notes); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ListCasesForPerson returns the cases linked to a person, optionally
// restricted to published.
func (r *Repository) ListCasesForPerson(ctx context.Context, personID uuid.UUID, publishedOnly bool) ([]Case, error) {
	stmt := `
SELECT ` + caseColumnsAliased("c") + `
FROM   cases c
JOIN   case_persons cp ON cp.case_id = c.id
WHERE  cp.person_id = $1`
	if publishedOnly {
		stmt += ` AND c.status = 'published'`
	}
	stmt += ` ORDER BY c.published_at DESC NULLS LAST, c.created_at DESC`
	rows, err := r.pool.Query(ctx, stmt, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Case{}
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// --------------- timeline + news ---------------

// AddTimelineEvent inserts a timeline row.
func (r *Repository) AddTimelineEvent(ctx context.Context, e TimelineEvent) (*TimelineEvent, error) {
	const stmt = `
INSERT INTO case_timeline (
  case_id, event_date, event_time, title_bn, title_en,
  description_bn, description_en, source_url, is_internal, created_by
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, created_at`
	if err := r.pool.QueryRow(ctx, stmt,
		e.CaseID, e.EventDate, e.EventTime, e.TitleBN, e.TitleEN,
		e.DescriptionBN, e.DescriptionEN, e.SourceURL, e.IsInternal, e.CreatedBy,
	).Scan(&e.ID, &e.CreatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}

// ListTimeline returns the timeline of a case, optionally filtering out
// internal events.
func (r *Repository) ListTimeline(ctx context.Context, caseID uuid.UUID, includeInternal bool) ([]TimelineEvent, error) {
	stmt := `
SELECT id, case_id, event_date, event_time, title_bn, title_en,
       description_bn, description_en, source_url, is_internal, created_by, created_at
FROM   case_timeline
WHERE  case_id = $1`
	if !includeInternal {
		stmt += ` AND is_internal = FALSE`
	}
	stmt += ` ORDER BY event_date, created_at`
	rows, err := r.pool.Query(ctx, stmt, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TimelineEvent{}
	for rows.Next() {
		var e TimelineEvent
		var et *string
		if err := rows.Scan(&e.ID, &e.CaseID, &e.EventDate, &et, &e.TitleBN, &e.TitleEN,
			&e.DescriptionBN, &e.DescriptionEN, &e.SourceURL, &e.IsInternal, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.EventTime = et
		out = append(out, e)
	}
	return out, rows.Err()
}

// AddNewsSource inserts a news source row.
func (r *Repository) AddNewsSource(ctx context.Context, n NewsSource) (*NewsSource, error) {
	const stmt = `
INSERT INTO news_sources (case_id, url, title, source_name, published_at, archived_url)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id, created_at`
	if err := r.pool.QueryRow(ctx, stmt,
		n.CaseID, n.URL, n.Title, n.SourceName, n.PublishedAt, n.ArchivedURL,
	).Scan(&n.ID, &n.CreatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

// ListNewsSources returns news source links for a case.
func (r *Repository) ListNewsSources(ctx context.Context, caseID uuid.UUID) ([]NewsSource, error) {
	const stmt = `
SELECT id, case_id, url, title, source_name, published_at, archived_url, created_at
FROM   news_sources WHERE case_id = $1 ORDER BY published_at DESC NULLS LAST, created_at`
	rows, err := r.pool.Query(ctx, stmt, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NewsSource{}
	for rows.Next() {
		var n NewsSource
		if err := rows.Scan(&n.ID, &n.CaseID, &n.URL, &n.Title, &n.SourceName, &n.PublishedAt, &n.ArchivedURL, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// --------------- helpers ---------------

const caseColumns = `id, case_number, slug, title_bn, title_en,
       summary_bn, summary_en, description_bn, description_en, internal_notes,
       incident_date, incident_time,
       country_id, division_id, district_id, upazila_id, location_text,
       crime_type_id, case_status, severity, cover_image_url, tags,
       status, submitted_by, approved_by, approved_at, published_at,
       view_count, download_count, created_at, updated_at`

func caseColumnsAliased(alias string) string {
	parts := strings.Split(caseColumns, ", ")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

func scanCase(row pgx.Row) (*Case, error) {
	c := &Case{}
	var et *string
	if err := row.Scan(
		&c.ID, &c.CaseNumber, &c.Slug, &c.TitleBN, &c.TitleEN,
		&c.SummaryBN, &c.SummaryEN, &c.DescriptionBN, &c.DescriptionEN, &c.InternalNotes,
		&c.IncidentDate, &et,
		&c.CountryID, &c.DivisionID, &c.DistrictID, &c.UpazilaID, &c.LocationText,
		&c.CrimeTypeID, &c.CaseStatus, &c.Severity, &c.CoverImageURL, &c.Tags,
		&c.Status, &c.SubmittedBy, &c.ApprovedBy, &c.ApprovedAt, &c.PublishedAt,
		&c.ViewCount, &c.DownloadCount, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IncidentTime = et
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return c, nil
}
