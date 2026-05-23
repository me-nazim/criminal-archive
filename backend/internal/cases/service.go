package cases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Service orchestrates case mutations: validation, state transitions,
// and audit logging.
type Service struct {
	pool   *pgxpool.Pool
	repo   *Repository
	logger *slog.Logger
}

// NewService constructs a Service.
func NewService(pool *pgxpool.Pool, repo *Repository, logger *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, logger: logger}
}

// WriteAudit forwards an audit entry. Exposed so package-external
// handlers can write audit rows without holding a *pgxpool.Pool.
func (s *Service) WriteAudit(ctx context.Context, e audit.Entry) {
	audit.Write(ctx, s.pool, e, s.logger)
}

// CreateInput is the payload from POST /cases.
type CreateInput struct {
	TitleBN       string     `json:"title_bn"`
	TitleEN       *string    `json:"title_en"`
	SummaryBN     *string    `json:"summary_bn"`
	SummaryEN     *string    `json:"summary_en"`
	DescriptionBN *string    `json:"description_bn"`
	DescriptionEN *string    `json:"description_en"`
	IncidentDate  *time.Time `json:"incident_date"`
	IncidentTime  *string    `json:"incident_time"`
	CountryID     *int       `json:"country_id"`
	DivisionID    *int       `json:"division_id"`
	DistrictID    *int       `json:"district_id"`
	UpazilaID     *int       `json:"upazila_id"`
	LocationText  *string    `json:"location_text"`
	CrimeTypeID   *int       `json:"crime_type_id"`
	CaseStatus    *string    `json:"case_status"`
	Severity      *int16     `json:"severity"`
	Tags          []string   `json:"tags"`
}

// Create validates the input and inserts a draft case owned by actor.
func (s *Service) Create(ctx context.Context, actor auth.Identity, in CreateInput, ip, ua string) (*Case, error) {
	title := strings.TrimSpace(in.TitleBN)
	if title == "" {
		return nil, httpx.ValidationError(map[string]string{"title_bn": "required"}, "")
	}
	c, err := s.repo.Create(ctx, CreateParams{
		TitleBN:       title,
		TitleEN:       in.TitleEN,
		SummaryBN:     in.SummaryBN,
		SummaryEN:     in.SummaryEN,
		DescriptionBN: in.DescriptionBN,
		DescriptionEN: in.DescriptionEN,
		IncidentDate:  in.IncidentDate,
		IncidentTime:  in.IncidentTime,
		CountryID:     in.CountryID,
		DivisionID:    in.DivisionID,
		DistrictID:    in.DistrictID,
		UpazilaID:     in.UpazilaID,
		LocationText:  in.LocationText,
		CrimeTypeID:   in.CrimeTypeID,
		CaseStatus:    in.CaseStatus,
		Severity:      in.Severity,
		Tags:          in.Tags,
		SubmittedBy:   actor.UserID,
	})
	if err != nil {
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID: &actor.UserID, Action: audit.ActionCaseCreate,
		TargetType: "case", TargetID: c.ID.String(),
		IP: ip, UserAgent: ua,
		Metadata: map[string]any{"case_number": c.CaseNumber},
	}, s.logger)
	return c, nil
}

// PatchFields applies a partial update by column name. Caller does RBAC.
func (s *Service) PatchFields(
	ctx context.Context, actor auth.Identity, id uuid.UUID, set map[string]any, ip, ua string,
) (*Case, error) {
	if len(set) == 0 {
		return s.repo.GetByID(ctx, id)
	}
	c, err := s.repo.UpdateFields(ctx, id, set)
	if err != nil {
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID: &actor.UserID, Action: audit.ActionCaseUpdate,
		TargetType: "case", TargetID: id.String(),
		IP: ip, UserAgent: ua,
	}, s.logger)
	return c, nil
}

// Transition moves the case from its current status to `to`. Used by
// submit / publish / unpublish / archive endpoints.
func (s *Service) Transition(
	ctx context.Context, actor auth.Identity, id uuid.UUID, to string, action audit.Action, ip, ua string,
) (*Case, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("Case not found.")
		}
		return nil, err
	}
	if !CanTransition(current.Status, to) {
		return nil, httpx.StateInvalid(fmt.Sprintf("Cannot move case from %q to %q.", current.Status, to))
	}
	var approver *uuid.UUID
	if to == "approved" || to == "published" {
		approver = &actor.UserID
	}
	updated, err := s.repo.SetStatus(ctx, id, to, approver)
	if err != nil {
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID: &actor.UserID, Action: action,
		TargetType: "case", TargetID: id.String(),
		IP: ip, UserAgent: ua,
		Metadata: map[string]any{"from": current.Status, "to": to, "case_number": current.CaseNumber},
	}, s.logger)
	return updated, nil
}
