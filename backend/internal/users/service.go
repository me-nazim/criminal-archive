package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Service wraps Repository with business rules around state transitions
// and writes audit-log rows for every privileged action.
type Service struct {
	pool   *pgxpool.Pool
	repo   *Repository
	auth   *auth.Repository // used to revoke sessions on suspend
	logger *slog.Logger
}

// NewService constructs a Service.
func NewService(pool *pgxpool.Pool, repo *Repository, authRepo *auth.Repository, logger *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, auth: authRepo, logger: logger}
}

// List exposes Repository.List behind the service layer.
func (s *Service) List(ctx context.Context, p ListParams) ([]User, error) {
	return s.repo.List(ctx, p)
}

// Get exposes Repository.Get.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, httpx.NotFound("User not found.")
		}
		return nil, err
	}
	return u, nil
}

// allowedTransitions enumerates the legal status transitions.
var allowedTransitions = map[string]map[string]bool{
	"pending":   {"approved": true, "rejected": true},
	"approved":  {"suspended": true},
	"suspended": {"approved": true},
}

func (s *Service) transition(
	ctx context.Context,
	actor auth.Identity,
	targetID uuid.UUID,
	to string,
	action audit.Action,
	ip, ua string,
) (*User, error) {
	target, err := s.Get(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if target.ID == actor.UserID {
		return nil, httpx.Forbidden("self_action_forbidden", "You cannot perform this action on your own account.")
	}
	// Admin cannot act on super-admin.
	if target.Role == string(auth.RoleSuperAdmin) && actor.Role != auth.RoleSuperAdmin {
		return nil, httpx.Forbidden("forbidden", "Only a super-admin can act on a super-admin.")
	}
	allowed, ok := allowedTransitions[target.Status]
	if !ok || !allowed[to] {
		return nil, httpx.StateInvalid(fmt.Sprintf("Cannot move user from %q to %q.", target.Status, to))
	}
	approver := &actor.UserID
	if to != "approved" {
		approver = nil
	}
	if err := s.repo.SetStatus(ctx, targetID, to, approver); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, httpx.NotFound("User not found.")
		}
		return nil, err
	}
	// Suspended users lose all sessions.
	if to == "suspended" && s.auth != nil {
		_ = s.auth.RevokeAllSessionsForUser(ctx, targetID)
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID:     &actor.UserID,
		Action:     action,
		TargetType: "user",
		TargetID:   targetID.String(),
		IP:         ip,
		UserAgent:  ua,
		Metadata: map[string]any{
			"from": target.Status, "to": to,
		},
	}, s.logger)
	return s.repo.Get(ctx, targetID)
}

// Approve moves pending → approved.
func (s *Service) Approve(ctx context.Context, actor auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
	return s.transition(ctx, actor, id, "approved", audit.ActionUserApprove, ip, ua)
}

// Reject moves pending → rejected.
func (s *Service) Reject(ctx context.Context, actor auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
	return s.transition(ctx, actor, id, "rejected", audit.ActionUserReject, ip, ua)
}

// Suspend moves approved → suspended and revokes all sessions.
func (s *Service) Suspend(ctx context.Context, actor auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
	return s.transition(ctx, actor, id, "suspended", audit.ActionUserSuspend, ip, ua)
}

// Reactivate moves suspended → approved.
func (s *Service) Reactivate(ctx context.Context, actor auth.Identity, id uuid.UUID, ip, ua string) (*User, error) {
	return s.transition(ctx, actor, id, "approved", audit.ActionUserReactivate, ip, ua)
}

// SetRole changes the role of a user. Only super-admin can grant or
// revoke the super-admin role.
func (s *Service) SetRole(ctx context.Context, actor auth.Identity, id uuid.UUID, newRole string, ip, ua string) (*User, error) {
	target, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if target.ID == actor.UserID {
		return nil, httpx.Forbidden("self_action_forbidden", "You cannot change your own role.")
	}
	switch newRole {
	case string(auth.RoleSuperAdmin), string(auth.RoleAdmin), string(auth.RoleModerator),
		string(auth.RoleContributor), string(auth.RoleViewer):
		// ok
	default:
		return nil, httpx.ValidationError(map[string]string{"role": "invalid role"}, "")
	}
	// Only super-admin can promote to or demote from super-admin.
	if (newRole == string(auth.RoleSuperAdmin) || target.Role == string(auth.RoleSuperAdmin)) &&
		actor.Role != auth.RoleSuperAdmin {
		return nil, httpx.Forbidden("forbidden", "Only a super-admin can grant or revoke the super-admin role.")
	}
	if err := s.repo.SetRole(ctx, id, newRole); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, httpx.NotFound("User not found.")
		}
		return nil, err
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID:     &actor.UserID,
		Action:     audit.ActionUserRoleChange,
		TargetType: "user",
		TargetID:   id.String(),
		IP:         ip,
		UserAgent:  ua,
		Metadata:   map[string]any{"from": target.Role, "to": newRole},
	}, s.logger)
	return s.repo.Get(ctx, id)
}
