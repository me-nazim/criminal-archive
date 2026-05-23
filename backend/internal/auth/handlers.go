package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

const refreshCookieName = "tip_refresh"

// Handlers wires Service to HTTP endpoints.
type Handlers struct {
	svc      *Service
	repo     *Repository
	logger   *slog.Logger
	cfg      JWTConfig
	cookiePath   string
	cookieDomain string
	cookieSecure bool
}

// HandlersConfig is what the entrypoint passes to NewHandlers.
type HandlersConfig struct {
	JWT          JWTConfig
	CookiePath   string
	CookieDomain string
	CookieSecure bool
}

// NewHandlers constructs Handlers with explicit cookie attributes so we
// can flip Secure on in production and off in dev.
func NewHandlers(svc *Service, repo *Repository, hc HandlersConfig, logger *slog.Logger) *Handlers {
	if hc.CookiePath == "" {
		hc.CookiePath = "/api/v1/auth"
	}
	return &Handlers{
		svc:          svc,
		repo:         repo,
		logger:       logger,
		cfg:          hc.JWT,
		cookiePath:   hc.CookiePath,
		cookieDomain: hc.CookieDomain,
		cookieSecure: hc.CookieSecure,
	}
}

// MountPublic mounts the unauthenticated subset of auth routes.
func (h *Handlers) MountPublic(r chi.Router) {
	r.Route("/auth", func(a chi.Router) {
		a.Post("/register", h.register)
		a.Post("/login", h.login)
		a.Post("/refresh", h.refresh)
	})
}

// MountAuthenticated mounts routes that require a valid access token.
// Caller is responsible for wrapping with the Authenticator middleware.
func (h *Handlers) MountAuthenticated(r chi.Router) {
	r.Get("/auth/me", h.me)
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/password/change", h.changePassword)
}

// ------- DTOs -------

type registerReq struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	FullName string  `json:"full_name"`
	Phone    *string `json:"phone,omitempty"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type publicUser struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	FullName    string  `json:"full_name"`
	DisplayName *string `json:"display_name,omitempty"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	Phone       *string `json:"phone,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	CreatedAt   string  `json:"created_at"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
}

func toPublicUser(u *User) publicUser {
	pu := publicUser{
		ID:          u.ID.String(),
		Email:       u.Email,
		FullName:    u.FullName,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Status:      u.Status,
		Phone:       u.Phone,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		CreatedAt:   u.CreatedAt.UTC().Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		s := u.LastLoginAt.UTC().Format(time.RFC3339)
		pu.LastLoginAt = &s
	}
	return pu
}

// ------- handlers -------

func (h *Handlers) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	ip, ua := audit.FromRequest(r)
	user, err := h.svc.Register(r.Context(), RegisterParams{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
	}, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toPublicUser(user))
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	ip, ua := audit.FromRequest(r)
	res, err := h.svc.Login(r.Context(), LoginParams{Email: req.Email, Password: req.Password}, ip, ua)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	h.setRefreshCookie(w, res.RefreshRaw, res.RefreshExpires)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token": res.AccessToken,
		"expires_in":   int(time.Until(res.AccessExpires).Seconds()),
		"user":         toPublicUser(res.User),
	})
}

func (h *Handlers) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c == nil || c.Value == "" {
		httpx.WriteError(w, r, h.logger, httpx.Unauthenticated("Missing refresh cookie."))
		return
	}
	ip, ua := audit.FromRequest(r)
	res, err := h.svc.Refresh(r.Context(), c.Value, ip, ua)
	if err != nil {
		// Clear the bad cookie so subsequent requests don't keep retrying.
		h.clearRefreshCookie(w)
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	h.setRefreshCookie(w, res.RefreshRaw, res.RefreshExpires)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token": res.AccessToken,
		"expires_in":   int(time.Until(res.AccessExpires).Seconds()),
		"user":         toPublicUser(res.User),
	})
}

func (h *Handlers) me(w http.ResponseWriter, r *http.Request) {
	id := MustIdentity(r.Context())
	u, err := h.repo.GetUserByID(r.Context(), id.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, h.logger, httpx.NotFound(""))
			return
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPublicUser(u))
}

func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil && c != nil {
		h.svc.Logout(r.Context(), c.Value)
	}
	h.clearRefreshCookie(w)

	id := MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	audit.Write(r.Context(), h.svc.pool, audit.Entry{
		UserID:     &id.UserID,
		Action:     audit.ActionUserLogout,
		TargetType: "user",
		TargetID:   id.UserID.String(),
		IP:         ip,
		UserAgent:  ua,
	}, h.logger)

	httpx.NoContent(w)
}

func (h *Handlers) changePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	id := MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	if err := h.svc.ChangePassword(r.Context(), id.UserID, req.OldPassword, req.NewPassword, ip, ua); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.NoContent(w)
}

// ------- cookie helpers -------

func (h *Handlers) setRefreshCookie(w http.ResponseWriter, raw string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    raw,
		Path:     h.cookiePath,
		Domain:   h.cookieDomain,
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handlers) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     h.cookiePath,
		Domain:   h.cookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
