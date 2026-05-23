// Package persons handles victim, accused, and witness profiles. A
// person can be linked to multiple cases via case_persons.
package persons

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

// Person mirrors the persons table.
type Person struct {
	ID            uuid.UUID  `json:"id"`
	Slug          string     `json:"slug"`
	FullNameBN    *string    `json:"full_name_bn,omitempty"`
	FullNameEN    *string    `json:"full_name_en,omitempty"`
	Aliases       []string   `json:"aliases"`
	PrimaryType   string     `json:"primary_type"` // victim | accused | witness | other
	Gender        *string    `json:"gender,omitempty"`
	DateOfBirth   *time.Time `json:"date_of_birth,omitempty"`
	PhotoURL      *string    `json:"photo_url,omitempty"`
	Occupation    *string    `json:"occupation,omitempty"`
	Organization  *string    `json:"organization,omitempty"`
	Designation   *string    `json:"designation,omitempty"`
	CountryID     *int       `json:"country_id,omitempty"`
	DivisionID    *int       `json:"division_id,omitempty"`
	DistrictID    *int       `json:"district_id,omitempty"`
	UpazilaID     *int       `json:"upazila_id,omitempty"`
	AddressLine   *string    `json:"address_line,omitempty"`
	PublicBioBN   *string    `json:"public_bio_bn,omitempty"`
	PublicBioEN   *string    `json:"public_bio_en,omitempty"`
	InternalNotes *string    `json:"internal_notes,omitempty"`
	IsAnonymous   bool       `json:"is_anonymous"`
	Status        string     `json:"status"`
	SubmittedBy   *uuid.UUID `json:"submitted_by,omitempty"`
	ApprovedBy    *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CaseCount     int        `json:"case_count,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Repository wraps SQL access for persons.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Sentinel errors.
var (
	ErrNotFound  = errors.New("person not found")
	ErrSlugTaken = errors.New("slug already taken")
)

// CreateParams collects inputs for Create.
type CreateParams struct {
	Slug         string
	FullNameBN   *string
	FullNameEN   *string
	Aliases      []string
	PrimaryType  string
	Gender       *string
	DateOfBirth  *time.Time
	PhotoURL     *string
	Occupation   *string
	Organization *string
	Designation  *string
	CountryID    *int
	DivisionID   *int
	DistrictID   *int
	UpazilaID    *int
	AddressLine  *string
	PublicBioBN  *string
	PublicBioEN  *string
	IsAnonymous  bool
	SubmittedBy  *uuid.UUID
}

// Create inserts a new person row in pending_review status.
func (r *Repository) Create(ctx context.Context, p CreateParams) (*Person, error) {
	const stmt = `
INSERT INTO persons (
  slug, full_name_bn, full_name_en, aliases, primary_type,
  gender, date_of_birth, photo_url, occupation, organization, designation,
  country_id, division_id, district_id, upazila_id, address_line,
  public_bio_bn, public_bio_en, is_anonymous, submitted_by, status
)
VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10, $11,
  $12, $13, $14, $15, $16,
  $17, $18, $19, $20, 'pending_review'
)
RETURNING id, slug, full_name_bn, full_name_en, aliases, primary_type,
          gender, date_of_birth, photo_url, occupation, organization, designation,
          country_id, division_id, district_id, upazila_id, address_line,
          public_bio_bn, public_bio_en, internal_notes, is_anonymous,
          status, submitted_by, approved_by, approved_at, published_at,
          created_at, updated_at`
	row := r.pool.QueryRow(ctx, stmt,
		p.Slug, p.FullNameBN, p.FullNameEN, p.Aliases, p.PrimaryType,
		p.Gender, p.DateOfBirth, p.PhotoURL, p.Occupation, p.Organization, p.Designation,
		p.CountryID, p.DivisionID, p.DistrictID, p.UpazilaID, p.AddressLine,
		p.PublicBioBN, p.PublicBioEN, p.IsAnonymous, p.SubmittedBy,
	)
	out, err := scanPerson(row, false)
	if err != nil {
		if isUniqueViolation(err, "persons_slug_key") {
			return nil, ErrSlugTaken
		}
		return nil, err
	}
	return out, nil
}

// UpdateParams describes a partial update. Nil pointers leave fields unchanged.
type UpdateParams struct {
	FullNameBN    *string
	FullNameEN    *string
	Aliases       []string
	HasAliases    bool
	Gender        *string
	HasGender     bool
	DateOfBirth   *time.Time
	HasDOB        bool
	PhotoURL      *string
	HasPhoto      bool
	Occupation    *string
	HasOccupation bool
	Organization  *string
	HasOrg        bool
	Designation   *string
	HasDesg       bool
	CountryID     *int
	HasCountry    bool
	DivisionID    *int
	HasDivision   bool
	DistrictID    *int
	HasDistrict   bool
	UpazilaID     *int
	HasUpazila    bool
	AddressLine   *string
	HasAddress    bool
	PublicBioBN   *string
	HasPubBN      bool
	PublicBioEN   *string
	HasPubEN      bool
	InternalNotes *string
	HasInternal   bool
	HasAnonymous  bool
	IsAnonymous   *bool
}

// Update applies a partial update to a person. The caller decides RBAC.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, p UpdateParams) (*Person, error) {
	sets := make([]string, 0, 12)
	args := []any{id}
	idx := 2
	add := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	if p.FullNameBN != nil {
		add("full_name_bn", p.FullNameBN)
	}
	if p.FullNameEN != nil {
		add("full_name_en", p.FullNameEN)
	}
	if p.HasAliases {
		add("aliases", p.Aliases)
	}
	if p.HasGender {
		add("gender", p.Gender)
	}
	if p.HasDOB {
		add("date_of_birth", p.DateOfBirth)
	}
	if p.HasPhoto {
		add("photo_url", p.PhotoURL)
	}
	if p.HasOccupation {
		add("occupation", p.Occupation)
	}
	if p.HasOrg {
		add("organization", p.Organization)
	}
	if p.HasDesg {
		add("designation", p.Designation)
	}
	if p.HasCountry {
		add("country_id", p.CountryID)
	}
	if p.HasDivision {
		add("division_id", p.DivisionID)
	}
	if p.HasDistrict {
		add("district_id", p.DistrictID)
	}
	if p.HasUpazila {
		add("upazila_id", p.UpazilaID)
	}
	if p.HasAddress {
		add("address_line", p.AddressLine)
	}
	if p.HasPubBN {
		add("public_bio_bn", p.PublicBioBN)
	}
	if p.HasPubEN {
		add("public_bio_en", p.PublicBioEN)
	}
	if p.HasInternal {
		add("internal_notes", p.InternalNotes)
	}
	if p.HasAnonymous {
		add("is_anonymous", p.IsAnonymous)
	}
	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}
	stmt := "UPDATE persons SET " + strings.Join(sets, ", ") + " WHERE id = $1 RETURNING " +
		personColumns
	row := r.pool.QueryRow(ctx, stmt, args...)
	return scanPerson(row, false)
}

// SetStatus moves a person to a new submission status. Used by approve/publish.
func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, status string, approver *uuid.UUID) (*Person, error) {
	const stmt = `
UPDATE persons
SET    status        = $2,
       approved_by   = COALESCE($3, approved_by),
       approved_at   = CASE WHEN $2 IN ('approved','published') AND approved_at IS NULL THEN now() ELSE approved_at END,
       published_at  = CASE WHEN $2 = 'published' AND published_at IS NULL THEN now() ELSE published_at END
WHERE  id = $1
RETURNING ` + personColumns
	row := r.pool.QueryRow(ctx, stmt, id, status, approver)
	return scanPerson(row, false)
}

// GetByID returns a single person by id, regardless of status.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Person, error) {
	const stmt = `SELECT ` + personColumns + ` FROM persons WHERE id = $1`
	row := r.pool.QueryRow(ctx, stmt, id)
	return scanPerson(row, false)
}

// GetBySlug returns a published person by slug. Pass requirePublished=false to
// also resolve persons in pending_review (e.g. for admin views).
func (r *Repository) GetBySlug(ctx context.Context, slug string, requirePublished bool) (*Person, error) {
	stmt := `SELECT ` + personColumns + ` FROM persons WHERE slug = $1`
	if requirePublished {
		stmt += ` AND status = 'published'`
	}
	row := r.pool.QueryRow(ctx, stmt, slug)
	return scanPerson(row, false)
}

// ListParams enumerates supported public-list filters.
type ListParams struct {
	Status      string // empty = published only
	PrimaryType string
	Search      string
	Limit       int
}

// ListPublic returns persons matching the filters.
func (r *Repository) ListPublic(ctx context.Context, p ListParams) ([]Person, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 24
	}
	conds := []string{}
	args := []any{}
	idx := 1
	if p.Status == "" {
		conds = append(conds, "status = 'published'")
	} else {
		conds = append(conds, fmt.Sprintf("status = $%d", idx))
		args = append(args, p.Status)
		idx++
	}
	if p.PrimaryType != "" {
		conds = append(conds, fmt.Sprintf("primary_type = $%d", idx))
		args = append(args, p.PrimaryType)
		idx++
	}
	if p.Search != "" {
		conds = append(conds, fmt.Sprintf("(full_name_en ILIKE $%d OR full_name_bn ILIKE $%d OR slug ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+p.Search+"%")
		idx++
	}
	stmt := `SELECT ` + personColumns + `,
       (SELECT count(*) FROM case_persons cp WHERE cp.person_id = persons.id) AS case_count
FROM   persons
WHERE  ` + strings.Join(conds, " AND ") + `
ORDER  BY published_at DESC NULLS LAST, created_at DESC
LIMIT  $` + itoa(idx)
	args = append(args, p.Limit)

	rows, err := r.pool.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Person, 0, p.Limit)
	for rows.Next() {
		p, err := scanPerson(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListByOwner returns every person submitted by the given user (any status).
func (r *Repository) ListByOwner(ctx context.Context, userID uuid.UUID) ([]Person, error) {
	const stmt = `SELECT ` + personColumns + `,
       (SELECT count(*) FROM case_persons cp WHERE cp.person_id = persons.id) AS case_count
FROM   persons
WHERE  submitted_by = $1
ORDER  BY created_at DESC`
	rows, err := r.pool.Query(ctx, stmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Person{}
	for rows.Next() {
		p, err := scanPerson(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// SlugExists reports whether the slug is already used by any person.
func (r *Repository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM persons WHERE slug = $1`, slug).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ---------- internal helpers ----------

const personColumns = `id, slug, full_name_bn, full_name_en, aliases, primary_type,
       gender, date_of_birth, photo_url, occupation, organization, designation,
       country_id, division_id, district_id, upazila_id, address_line,
       public_bio_bn, public_bio_en, internal_notes, is_anonymous,
       status, submitted_by, approved_by, approved_at, published_at,
       created_at, updated_at`

// scanPerson maps a row to a Person. If withCount is true the row is expected
// to carry an extra trailing case_count column.
func scanPerson(row pgx.Row, withCount bool) (*Person, error) {
	p := &Person{}
	dest := []any{
		&p.ID, &p.Slug, &p.FullNameBN, &p.FullNameEN, &p.Aliases, &p.PrimaryType,
		&p.Gender, &p.DateOfBirth, &p.PhotoURL, &p.Occupation, &p.Organization, &p.Designation,
		&p.CountryID, &p.DivisionID, &p.DistrictID, &p.UpazilaID, &p.AddressLine,
		&p.PublicBioBN, &p.PublicBioEN, &p.InternalNotes, &p.IsAnonymous,
		&p.Status, &p.SubmittedBy, &p.ApprovedBy, &p.ApprovedAt, &p.PublishedAt,
		&p.CreatedAt, &p.UpdatedAt,
	}
	if withCount {
		dest = append(dest, &p.CaseCount)
	}
	if err := row.Scan(dest...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	return string(buf[pos:])
}

func isUniqueViolation(err error, _ string) bool {
	// pgx wraps PG errors; we keep this loose because we don't always know the
	// constraint name. Caller may inspect the message if it matters.
	return strings.Contains(err.Error(), "23505")
}
