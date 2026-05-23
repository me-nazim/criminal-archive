package persons

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Service orchestrates person mutations and writes audit events.
type Service struct {
	pool   *pgxpool.Pool
	repo   *Repository
	logger *slog.Logger
}

// NewService constructs a Service.
func NewService(pool *pgxpool.Pool, repo *Repository, logger *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, logger: logger}
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// makeSlug converts arbitrary text into a URL-safe slug. Bengali characters
// fall back to a stable random suffix when no Latin name is available.
func makeSlug(en, bn string) string {
	src := strings.TrimSpace(strings.ToLower(en))
	if src == "" {
		src = strings.TrimSpace(bn)
	}
	src = slugRE.ReplaceAllString(src, "-")
	src = strings.Trim(src, "-")
	if src == "" {
		src = "person"
	}
	if len(src) > 80 {
		src = src[:80]
	}
	return src
}

func (s *Service) ensureUniqueSlug(ctx context.Context, base string) (string, error) {
	slug := base
	for i := 0; i < 8; i++ {
		taken, err := s.repo.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, time.Now().UnixNano()%10000+int64(i))
	}
	return "", errors.New("could not allocate unique slug")
}

var validPrimaryTypes = map[string]bool{
	"victim": true, "accused": true, "witness": true, "other": true,
}

// CreateInput is the unvalidated user-facing payload.
type CreateInput struct {
	FullNameBN   *string    `json:"full_name_bn"`
	FullNameEN   *string    `json:"full_name_en"`
	Aliases      []string   `json:"aliases"`
	PrimaryType  string     `json:"primary_type"`
	Gender       *string    `json:"gender"`
	DateOfBirth  *time.Time `json:"date_of_birth"`
	PhotoURL     *string    `json:"photo_url"`
	Occupation   *string    `json:"occupation"`
	Organization *string    `json:"organization"`
	Designation  *string    `json:"designation"`
	CountryID    *int       `json:"country_id"`
	DivisionID   *int       `json:"division_id"`
	DistrictID   *int       `json:"district_id"`
	UpazilaID    *int       `json:"upazila_id"`
	AddressLine  *string    `json:"address_line"`
	PublicBioBN  *string    `json:"public_bio_bn"`
	PublicBioEN  *string    `json:"public_bio_en"`
	IsAnonymous  bool       `json:"is_anonymous"`
}

// Create validates input and inserts a person row.
func (s *Service) Create(ctx context.Context, actor auth.Identity, in CreateInput, ip, ua string) (*Person, error) {
	if !validPrimaryTypes[in.PrimaryType] {
		return nil, httpx.ValidationError(
			map[string]string{"primary_type": "must be one of victim, accused, witness, other"}, "")
	}
	en := ""
	if in.FullNameEN != nil {
		en = *in.FullNameEN
	}
	bn := ""
	if in.FullNameBN != nil {
		bn = *in.FullNameBN
	}
	if strings.TrimSpace(en) == "" && strings.TrimSpace(bn) == "" && !in.IsAnonymous {
		return nil, httpx.ValidationError(
			map[string]string{"full_name_en": "required unless person is anonymous"}, "")
	}
	base := makeSlug(en, bn)
	slug, err := s.ensureUniqueSlug(ctx, base)
	if err != nil {
		return nil, err
	}
	uid := actor.UserID
	p, err := s.repo.Create(ctx, CreateParams{
		Slug:         slug,
		FullNameBN:   in.FullNameBN,
		FullNameEN:   in.FullNameEN,
		Aliases:      sanitizeAliases(in.Aliases),
		PrimaryType:  in.PrimaryType,
		Gender:       in.Gender,
		DateOfBirth:  in.DateOfBirth,
		PhotoURL:     in.PhotoURL,
		Occupation:   in.Occupation,
		Organization: in.Organization,
		Designation:  in.Designation,
		CountryID:    in.CountryID,
		DivisionID:   in.DivisionID,
		DistrictID:   in.DistrictID,
		UpazilaID:    in.UpazilaID,
		AddressLine:  in.AddressLine,
		PublicBioBN:  in.PublicBioBN,
		PublicBioEN:  in.PublicBioEN,
		IsAnonymous:  in.IsAnonymous,
		SubmittedBy:  &uid,
	})
	if err != nil {
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID:     &actor.UserID,
		Action:     audit.ActionPersonCreate,
		TargetType: "person",
		TargetID:   p.ID.String(),
		IP:         ip, UserAgent: ua,
		Metadata: map[string]any{"slug": p.Slug, "primary_type": p.PrimaryType},
	}, s.logger)
	return p, nil
}

// UpdateInput mirrors CreateInput but every field is optional. We accept
// "set to empty string" semantics for nullable text fields.
type UpdateInput struct {
	FullNameBN    *string    `json:"full_name_bn,omitempty"`
	HasFullNameBN bool       `json:"-"`
	FullNameEN    *string    `json:"full_name_en,omitempty"`
	HasFullNameEN bool       `json:"-"`
	Aliases       *[]string  `json:"aliases,omitempty"`
	Gender        *string    `json:"gender,omitempty"`
	HasGender     bool       `json:"-"`
	DateOfBirth   *time.Time `json:"date_of_birth,omitempty"`
	HasDOB        bool       `json:"-"`
	PhotoURL      *string    `json:"photo_url,omitempty"`
	HasPhoto      bool       `json:"-"`
	Occupation    *string    `json:"occupation,omitempty"`
	HasOccupation bool       `json:"-"`
	Organization  *string    `json:"organization,omitempty"`
	HasOrg        bool       `json:"-"`
	Designation   *string    `json:"designation,omitempty"`
	HasDesg       bool       `json:"-"`
	CountryID     *int       `json:"country_id,omitempty"`
	HasCountry    bool       `json:"-"`
	DivisionID    *int       `json:"division_id,omitempty"`
	HasDivision   bool       `json:"-"`
	DistrictID    *int       `json:"district_id,omitempty"`
	HasDistrict   bool       `json:"-"`
	UpazilaID     *int       `json:"upazila_id,omitempty"`
	HasUpazila    bool       `json:"-"`
	AddressLine   *string    `json:"address_line,omitempty"`
	HasAddress    bool       `json:"-"`
	PublicBioBN   *string    `json:"public_bio_bn,omitempty"`
	HasPubBN      bool       `json:"-"`
	PublicBioEN   *string    `json:"public_bio_en,omitempty"`
	HasPubEN      bool       `json:"-"`
	InternalNotes *string    `json:"internal_notes,omitempty"`
	HasInternal   bool       `json:"-"`
	IsAnonymous   *bool      `json:"is_anonymous,omitempty"`
}

// Update applies a partial update. Author-only edit unless caller is mod+.
func (s *Service) Update(ctx context.Context, actor auth.Identity, id uuid.UUID, in UpdateInput, ip, ua string) (*Person, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("Person not found.")
		}
		return nil, err
	}

	rank := map[auth.Role]int{
		auth.RoleSuperAdmin: 5, auth.RoleAdmin: 4, auth.RoleModerator: 3,
		auth.RoleContributor: 2, auth.RoleViewer: 1,
	}
	isOwner := existing.SubmittedBy != nil && *existing.SubmittedBy == actor.UserID
	if !isOwner && rank[actor.Role] < rank[auth.RoleModerator] {
		return nil, httpx.Forbidden("forbidden", "")
	}
	// Only admin+ may edit internal_notes.
	if in.HasInternal && rank[actor.Role] < rank[auth.RoleAdmin] {
		return nil, httpx.Forbidden("forbidden", "Only admins can edit internal notes.")
	}

	params := UpdateParams{
		FullNameBN: in.FullNameBN, FullNameEN: in.FullNameEN,
		HasGender: in.HasGender, Gender: in.Gender,
		HasDOB: in.HasDOB, DateOfBirth: in.DateOfBirth,
		HasPhoto: in.HasPhoto, PhotoURL: in.PhotoURL,
		HasOccupation: in.HasOccupation, Occupation: in.Occupation,
		HasOrg: in.HasOrg, Organization: in.Organization,
		HasDesg: in.HasDesg, Designation: in.Designation,
		HasCountry: in.HasCountry, CountryID: in.CountryID,
		HasDivision: in.HasDivision, DivisionID: in.DivisionID,
		HasDistrict: in.HasDistrict, DistrictID: in.DistrictID,
		HasUpazila: in.HasUpazila, UpazilaID: in.UpazilaID,
		HasAddress: in.HasAddress, AddressLine: in.AddressLine,
		HasPubBN: in.HasPubBN, PublicBioBN: in.PublicBioBN,
		HasPubEN: in.HasPubEN, PublicBioEN: in.PublicBioEN,
		HasInternal: in.HasInternal, InternalNotes: in.InternalNotes,
	}
	if in.Aliases != nil {
		params.HasAliases = true
		params.Aliases = sanitizeAliases(*in.Aliases)
	}
	if in.IsAnonymous != nil {
		params.HasAnonymous = true
		params.IsAnonymous = in.IsAnonymous
	}

	updated, err := s.repo.Update(ctx, id, params)
	if err != nil {
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID: &actor.UserID, Action: audit.ActionPersonUpdate,
		TargetType: "person", TargetID: id.String(),
		IP: ip, UserAgent: ua,
	}, s.logger)
	return updated, nil
}

// Approve transitions a person from pending_review to published.
func (s *Service) Approve(ctx context.Context, actor auth.Identity, id uuid.UUID, ip, ua string) (*Person, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("Person not found.")
		}
		return nil, err
	}
	if p.Status == "published" {
		return p, nil
	}
	if p.Status != "pending_review" && p.Status != "approved" {
		return nil, httpx.StateInvalid(fmt.Sprintf("Cannot publish a person from %q.", p.Status))
	}
	updated, err := s.repo.SetStatus(ctx, id, "published", &actor.UserID)
	if err != nil {
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID: &actor.UserID, Action: "person.approve",
		TargetType: "person", TargetID: id.String(),
		IP: ip, UserAgent: ua,
		Metadata: map[string]any{"from": p.Status, "to": "published"},
	}, s.logger)
	return updated, nil
}

func sanitizeAliases(in []string) []string {
	out := make([]string, 0, len(in))
	for _, a := range in {
		if t := strings.TrimSpace(a); t != "" {
			out = append(out, t)
		}
	}
	return out
}
