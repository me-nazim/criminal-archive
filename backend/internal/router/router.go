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

	"github.com/me-nazim/criminal-archive/backend/internal/attachments"
	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/cases"
	"github.com/me-nazim/criminal-archive/backend/internal/config"
	"github.com/me-nazim/criminal-archive/backend/internal/crimetypes"
	"github.com/me-nazim/criminal-archive/backend/internal/feeds"
	"github.com/me-nazim/criminal-archive/backend/internal/locations"
	"github.com/me-nazim/criminal-archive/backend/internal/persons"
	"github.com/me-nazim/criminal-archive/backend/internal/search"
	"github.com/me-nazim/criminal-archive/backend/internal/stats"
	"github.com/me-nazim/criminal-archive/backend/internal/storage"
	"github.com/me-nazim/criminal-archive/backend/internal/users"
	"github.com/me-nazim/criminal-archive/backend/internal/verification"
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

	personsRepo := persons.NewRepository(pool)
	personsSvc := persons.NewService(pool, personsRepo, logger)
	personsHandlers := persons.NewHandlers(personsRepo, personsSvc, logger)

	casesRepo := cases.NewRepository(pool)
	casesSvc := cases.NewService(pool, casesRepo, logger)
	casesHandlers := cases.NewHandlers(casesRepo, casesSvc, logger)

	verifRepo := verification.NewRepository(pool)
	verifHandlers := verification.NewHandlers(verifRepo, logger)

	auditRepo := audit.NewRepository(pool)
	auditHandlers := audit.NewHandlers(auditRepo, logger)

	statsHandlers := stats.NewHandlers(pool, logger)

	searchHandlers := search.NewHandlers(casesRepo, personsRepo, logger)
	feedsHandlers := feeds.NewHandlers(casesRepo, personsRepo, cfg.AppBaseURL, logger)

	// Object storage is optional: if config is missing we boot without it
	// and skip the attachment routes. This lets the rest of the API keep
	// working in a dev environment without R2/MinIO ready.
	var attachHandlers *attachments.Handlers
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" && cfg.S3Bucket != "" {
		store, err := storage.NewClient(context.Background(), storage.Config{
			Endpoint:       cfg.S3Endpoint,
			Region:         cfg.S3Region,
			AccessKey:      cfg.S3AccessKey,
			SecretKey:      cfg.S3SecretKey,
			Bucket:         cfg.S3Bucket,
			PublicBaseURL:  cfg.S3PublicBaseURL,
			ForcePathStyle: cfg.S3ForcePathStyle,
		})
		if err != nil {
			logger.Warn("storage init failed; attachment routes will be unavailable", "err", err)
		} else {
			// Best-effort: ensure the bucket exists (helpful for local minio).
			if cfg.AppEnv != "production" {
				if err := store.EnsureBucket(context.Background()); err != nil {
					logger.Warn("storage: ensure bucket failed", "err", err)
				}
			}
			attachRepo := attachments.NewRepository(pool)
			attachHandlers = attachments.NewHandlers(attachRepo, casesRepo, store, []byte(cfg.JWTSecret), logger)
			attachHandlers.SetAudit(pool, logger)
		}
	} else {
		logger.Info("storage not configured; attachment routes disabled")
	}

	// ---------- mount ----------
	feedsHandlers.Mount(r) // /feed.xml + /sitemap.xml at the root

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/version", versionHandler)
		api.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"message": "pong"})
		})

		// ---------- public ----------
		api.Group(func(p chi.Router) {
			p.Use(auth.OptionalAuthenticator(jwtCfg)) // light context awareness
			locHandlers.Mount(p)
			crimeHandlers.Mount(p)
			personsHandlers.MountPublic(p)
			casesHandlers.MountPublic(p)
			searchHandlers.Mount(p)
		})

		// ---------- public auth (register / login / refresh) ----------
		authHandlers.MountPublic(api)

		// ---------- authenticated ----------
		api.Group(func(p chi.Router) {
			p.Use(auth.Authenticator(jwtCfg))
			authHandlers.MountAuthenticated(p)
			personsHandlers.MountAuthenticated(p)
			casesHandlers.MountAuthenticated(p)
			verifHandlers.MountAuthenticated(p)
			if attachHandlers != nil {
				attachHandlers.MountAuthenticated(p)
			}

			// ---------- admin / super-admin ----------
			p.Route("/admin", func(a chi.Router) {
				a.Use(auth.RequireMinRole(auth.RoleAdmin))
				usersHandlers.Mount(a)
				personsHandlers.MountAdmin(a)
				casesHandlers.MountAdmin(a)
				verifHandlers.MountAdmin(a)
				auditHandlers.Mount(a)
				statsHandlers.Mount(a)
				if attachHandlers != nil {
					attachHandlers.MountAdmin(a)
				}
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
