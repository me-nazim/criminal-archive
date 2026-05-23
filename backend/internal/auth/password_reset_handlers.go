package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// EmailEnqueuer is the smallest contract the password-reset endpoint
// needs: enqueue an email by template name. The email package's Manager
// satisfies this interface.
type EmailEnqueuer interface {
	Enqueue(ctx context.Context, to, subject, template string, data map[string]any) error
}

// ResetHandlers wires the forgot-password / reset-password endpoints.
type ResetHandlers struct {
	svc      *Service
	repo     *Repository
	mailer   EmailEnqueuer
	frontend string
	logger   *slog.Logger
}

// NewResetHandlers constructs ResetHandlers. mailer may be nil — in
// that case the email step is skipped and the endpoint still responds
// 204 (so attackers can't probe for valid emails).
func NewResetHandlers(svc *Service, repo *Repository, mailer EmailEnqueuer, frontendURL string, logger *slog.Logger) *ResetHandlers {
	return &ResetHandlers{svc: svc, repo: repo, mailer: mailer, frontend: strings.TrimRight(frontendURL, "/"), logger: logger}
}

// Mount mounts the public endpoints.
func (h *ResetHandlers) Mount(r chi.Router) {
	r.Post("/auth/password/forgot", h.forgot)
	r.Post("/auth/password/reset", h.reset)
}

type forgotReq struct {
	Email string `json:"email"`
}

func (h *ResetHandlers) forgot(w http.ResponseWriter, r *http.Request) {
	var req forgotReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	ip, ua := audit.FromRequest(r)
	raw, user, err := h.repo.CreateResetToken(r.Context(), req.Email, ip, ua)
	if err != nil {
		if errors.Is(err, ErrNoSuchUser) {
			// Account enumeration guard: silently succeed.
			httpx.NoContent(w)
			return
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if h.mailer != nil {
		resetURL := h.frontend + "/reset-password?token=" + raw
		if err := h.mailer.Enqueue(r.Context(), user.Email, "", "password.reset", map[string]any{
			"FullName":         user.FullName,
			"ResetURL":         resetURL,
			"ExpiresInMinutes": int(ResetTokenTTL.Minutes()),
		}); err != nil && h.logger != nil {
			h.logger.Warn("password reset: enqueue failed", "err", err)
		}
	}
	audit.Write(r.Context(), h.svc.pool, audit.Entry{
		UserID:     &user.ID,
		Action:     audit.Action("user.password_reset_request"),
		TargetType: "user",
		TargetID:   user.ID.String(),
		IP:         ip, UserAgent: ua,
	}, h.logger)
	httpx.NoContent(w)
}

type resetReq struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *ResetHandlers) reset(w http.ResponseWriter, r *http.Request) {
	var req resetReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if req.Token == "" {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("token is required"))
		return
	}
	if err := ValidatePassword(req.NewPassword); err != nil {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"new_password": "must be at least 10 characters"}, ""))
		return
	}
	rec, err := h.repo.FindActiveResetToken(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, h.logger, httpx.BadRequest("Reset link is invalid or has expired."))
			return
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	hash, err := HashPassword(req.NewPassword, h.svc.bcryptCost)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.repo.UpdatePassword(r.Context(), rec.UserID, hash); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.repo.MarkResetTokenUsed(r.Context(), rec.ID); err != nil && h.logger != nil {
		h.logger.Warn("password reset: mark used failed", "err", err)
	}
	if err := h.repo.RevokeAllSessionsForUser(r.Context(), rec.UserID); err != nil && h.logger != nil {
		h.logger.Warn("password reset: revoke sessions failed", "err", err)
	}
	ip, ua := audit.FromRequest(r)
	uid := rec.UserID
	audit.Write(r.Context(), h.svc.pool, audit.Entry{
		UserID:     &uid,
		Action:     audit.ActionUserPasswordSet,
		TargetType: "user",
		TargetID:   rec.UserID.String(),
		IP:         ip, UserAgent: ua,
		Metadata: map[string]any{"via": "reset_token"},
	}, h.logger)
	httpx.NoContent(w)
}
