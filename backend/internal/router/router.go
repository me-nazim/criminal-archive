// Package router wires HTTP routes for the API. The router itself owns
// no business logic — it instantiates Service / Repository / Handlers
// objects and mounts them at the right paths with the right middleware.
package router

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/config"
	"github.com/me-nazim/criminal-archive/backend/internal/crimetypes"
	"github.com/me-nazim/criminal-archive/backend/internal/locations"
	"github.com/me-nazim/criminal-archive/backend/internal/users"
)

// New constructs the top-level HTTP handler. When pool is nil, only
// /health is mounted — the API will respond but every data-touching
// endpoint will return 500. Useful for local boot before the DB is up.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler(pool))

	if pool == nil {
		// Without a DB nothing else can usefully exist.
		return r
	}

	// ---------- repositories & services ----------
	authRepo := auth.NewRepository(pool)
	jwtCfg := auth.JWTConfig{
		Secret:     []byte(cfg.JWTSecret),
		Issuer:     cfg.JWTIssuer,
		AccessTTL:  cfg.JWTAccessTTL,
		RefreshTTL: cfg.JWTRefreshTTL,
	}
	authSvc := auth.NewService(pool, authRepo, jwtCfg, cfg.BcryptCost, logger)
	authHandlers := auth.NewHandlers(authSvc, authRepo, auth.HandlersConfig{
		JWT:          jwtCfg,
		CookieDomain: cfg.CookieDomain,
		CookieSecure: cfg.CookieSecure,
	}, logger)

	usersRepo := users.NewRepository(pool)
	usersSvc := users.NewService(pool, usersRepo, authRepo, logger)
	usersHandlers := users.NewHandlers(usersSvc, logger)

	locRepo := locations.NewRepository(pool)
	locHandlers := locations.NewHandlers(locRepo, logger)

	crimeRepo := crimetypes.NewRepository(pool)
	crimeHandlers := crimetypes.NewHandlers(crimeRepo, logger)

	// ---------- mount ----------
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/version", versionHandler)

		// Public ping retained as a smoke endpoint.
		api.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"message": "pong"})
		})

		// Public reference data.
		locHandlers.Mount(api)
		crimeHandlers.Mount(api)

		// Public auth (register, login, refresh).
		authHandlers.MountPublic(api)

		// Authenticated routes.
		api.Group(func(p chi.Router) {
			p.Use(auth.Authenticator(jwtCfg))
			authHandlers.MountAuthenticated(p)

			// Admin-only routes.
			p.Route("/admin", func(a chi.Router) {
				a.Use(auth.RequireMinRole(auth.RoleAdmin))
				usersHandlers.Mount(a)
			})
		})
	})

	return r
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		}
		if pool == nil {
			status["db"] = "unconfigured"
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				status["db"] = "down"
				writeJSON(w, http.StatusServiceUnavailable, status)
				return
			}
			status["db"] = "up"
		}
		writeJSON(w, http.StatusOK, status)
	}
}

// versionHandler returns a small payload identifying the running build.
// The build sha and version are wired in at build time via -ldflags.
func versionHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version": Version,
		"commit":  Commit,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Version and Commit can be overridden by the linker.
var (
	Version = "dev"
	Commit  = "unknown"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
