package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Role is the canonical role string. We don't use a real enum because
// chi handlers compare against the string anyway.
type Role string

const (
	RoleSuperAdmin  Role = "super_admin"
	RoleAdmin       Role = "admin"
	RoleModerator   Role = "moderator"
	RoleContributor Role = "contributor"
	RoleViewer      Role = "viewer"
)

// Identity is what we put into the request context after authenticating.
type Identity struct {
	UserID uuid.UUID
	Role   Role
}

type ctxKey int

const identityKey ctxKey = 1

// ContextWithIdentity returns a derived context that carries id.
func ContextWithIdentity(parent context.Context, id Identity) context.Context {
	return context.WithValue(parent, identityKey, id)
}

// IdentityFrom extracts the Identity stored in ctx, if any.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// MustIdentity panics if the context has no Identity. Use only in
// handlers that are wrapped by Authenticator (otherwise prefer IdentityFrom).
func MustIdentity(ctx context.Context) Identity {
	id, ok := IdentityFrom(ctx)
	if !ok {
		panic("auth: identity missing from context")
	}
	return id
}

// Authenticator returns a middleware that requires a valid access JWT.
// On success it puts the Identity into request context. On failure it
// writes a 401 and stops the chain.
func Authenticator(cfg JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				httpx.WriteError(w, r, nil, httpx.Unauthenticated(""))
				return
			}
			claims, err := ParseAccessToken(cfg, raw)
			if err != nil {
				if IsExpired(err) {
					httpx.WriteError(w, r, nil, httpx.TokenExpired())
					return
				}
				httpx.WriteError(w, r, nil, httpx.Unauthenticated("Invalid access token."))
				return
			}
			uid, err := uuid.Parse(claims.Subject)
			if err != nil {
				httpx.WriteError(w, r, nil, httpx.Unauthenticated("Invalid token subject."))
				return
			}
			id := Identity{UserID: uid, Role: Role(claims.Role)}
			next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), id)))
		})
	}
}

// OptionalAuthenticator is like Authenticator but does not reject the
// request when the header is missing or invalid; it just doesn't set an
// identity. Useful for public endpoints that have a "logged-in" extra.
func OptionalAuthenticator(cfg JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw != "" {
				if claims, err := ParseAccessToken(cfg, raw); err == nil {
					if uid, err := uuid.Parse(claims.Subject); err == nil {
						id := Identity{UserID: uid, Role: Role(claims.Role)}
						r = r.WithContext(ContextWithIdentity(r.Context(), id))
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns a middleware that 403s if the request identity
// does not have one of the allowed roles. Super-admin always passes.
func RequireRole(allowed ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := IdentityFrom(r.Context())
			if !ok {
				httpx.WriteError(w, r, nil, httpx.Unauthenticated(""))
				return
			}
			if id.Role == RoleSuperAdmin {
				next.ServeHTTP(w, r)
				return
			}
			for _, a := range allowed {
				if id.Role == a {
					next.ServeHTTP(w, r)
					return
				}
			}
			httpx.WriteError(w, r, nil, httpx.Forbidden("forbidden", ""))
		})
	}
}

// RequireMinRole gates by role hierarchy (super_admin > admin > moderator
// > contributor > viewer). It is shorthand for RequireRole(... all roles
// at or above min ...).
func RequireMinRole(min Role) func(http.Handler) http.Handler {
	rank := map[Role]int{
		RoleViewer:      1,
		RoleContributor: 2,
		RoleModerator:   3,
		RoleAdmin:       4,
		RoleSuperAdmin:  5,
	}
	required, ok := rank[min]
	if !ok {
		// Fail closed for unknown roles.
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				httpx.WriteError(w, r, nil, httpx.Forbidden("forbidden", ""))
			})
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := IdentityFrom(r.Context())
			if !ok {
				httpx.WriteError(w, r, nil, httpx.Unauthenticated(""))
				return
			}
			if rank[id.Role] >= required {
				next.ServeHTTP(w, r)
				return
			}
			httpx.WriteError(w, r, nil, httpx.Forbidden("forbidden", ""))
		})
	}
}

func bearerToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if v == "" {
		return ""
	}
	parts := strings.SplitN(v, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// Sentinel error so service code can distinguish "no token" from
// network / DB issues if they ever check. Currently unused but kept
// for symmetry with other packages.
var ErrNotAuthenticated = errors.New("not authenticated")
