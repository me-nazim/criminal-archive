// Package router wires HTTP routes for the API. The router itself owns
// no business logic — it instantiates Service / Repository / Handlers
// objects and mounts them at the right paths with the right middleware.
package router

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/attachments"
	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/cases"
	"github.com/me-nazim/criminal-archive/backend/internal/config"
	"github.com/me-nazim/criminal-archive/backend/internal/crimetypes"
	"github.com/me-nazim/criminal-archive/backend/internal/email"
	"github.com/me-nazim/criminal-archive/backend/internal/feeds"
	"github.com/me-nazim/criminal-archive/backend/internal/locations"
	"github.com/me-nazim/criminal-archive/backend/internal/metrics"
	tipmw "github.com/me-nazim/criminal-archive/backend/internal/middleware"
	"github.com/me-nazim/criminal-archive/backend/internal/notifications"
	"github.com/me-nazim/criminal-archive/backend/internal/persons"
	"github.com/me-nazim/criminal-archive/backend/internal/search"
	"github.com/me-nazim/criminal-archive/backend/internal/settings"
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

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(tipmw.SecurityHeaders)
	r.Use(tipmw.AccessLog(logger))
	r.Use(metrics.Middleware)

	rl := tipmw.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, 0)
	r.Use(rl.Middleware())

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler(pool))
	r.Handle("/metrics", metrics.Handler())

	if pool == nil {
		return r
	}

	// ---------- settings (encrypted KV) ----------
	var cipher *settings.Cipher
	if cfg.SettingsKey != "" {
		c, err := settings.NewCipher(cfg.SettingsKey)
		if err != nil {
			logger.Warn("settings: cipher init failed; secrets will be stored in plaintext", "err", err)
		} else {
			cipher = c
		}
	} else {
		logger.Warn("settings: APP_SETTINGS_KEY is not set; secrets will be stored in plaintext. Configure it before going to production.")
	}
	settingsStore := settings.New(pool, cipher, logger)
	if err := settingsStore.Reload(context.Background()); err != nil {
		logger.Warn("settings: initial reload failed", "err", err)
	}
	// Bootstrap storage row from env on first boot when admin hasn't
	// supplied one yet — keeps existing dev workflows working.
	bootstrapStorageFromEnv(context.Background(), cfg, settingsStore, logger)

	// ---------- email + storage managers ----------
	emailMgr := email.NewManager(pool, settingsStore, cfg.FrontendBaseURL, logger)
	if err := emailMgr.Refresh(context.Background()); err != nil {
		logger.Warn("email: refresh failed", "err", err)
	}
	emailMgr.StartWorker(context.Background())

	storageMgr := storage.NewManager(settingsStore, logger)
	if err := storageMgr.Refresh(context.Background()); err != nil {
		if !errors.Is(err, storage.ErrUnconfigured) {
			logger.Warn("storage: initial refresh failed", "err", err)
		}
	} else if cfg.AppEnv != "production" {
		// Best-effort bucket creation in dev.
		if c, cErr := storageMgr.Client(context.Background()); cErr == nil {
			_ = c.EnsureBucket(context.Background())
		}
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
	resetHandlers := auth.NewResetHandlers(authSvc, authRepo, emailMgr, cfg.FrontendBaseURL, logger)

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

	notifRepo := notifications.NewRepository(pool)
	notifHandlers := notifications.NewHandlers(notifRepo, logger)

	settingsHandlers := settings.NewHandlers(settingsStore, providerTester{email: emailMgr, storage: storageMgr}, logger)

	attachRepo := attachments.NewRepository(pool)
	attachHandlers := attachments.NewHandlers(attachRepo, casesRepo, storageMgr, []byte(cfg.JWTSecret), logger)
	attachHandlers.SetAudit(pool, logger)

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
			settingsHandlers.MountPublic(p)
			locHandlers.Mount(p)
			crimeHandlers.Mount(p)
			personsHandlers.MountPublic(p)
			casesHandlers.MountPublic(p)
			searchHandlers.Mount(p)
		})

		// ---------- public auth (register / login / refresh / forgot) ----------
		authHandlers.MountPublic(api)
		resetHandlers.Mount(api)

		// ---------- authenticated ----------
		api.Group(func(p chi.Router) {
			p.Use(auth.Authenticator(jwtCfg))
			authHandlers.MountAuthenticated(p)
			personsHandlers.MountAuthenticated(p)
			casesHandlers.MountAuthenticated(p)
			verifHandlers.MountAuthenticated(p)
			notifHandlers.Mount(p)
			attachHandlers.MountAuthenticated(p)

			// ---------- admin / super-admin ----------
			p.Route("/admin", func(a chi.Router) {
				a.Use(auth.RequireMinRole(auth.RoleAdmin))
				usersHandlers.Mount(a)
				personsHandlers.MountAdmin(a)
				casesHandlers.MountAdmin(a)
				verifHandlers.MountAdmin(a)
				auditHandlers.Mount(a)
				statsHandlers.Mount(a)
				attachHandlers.MountAdmin(a)
				settingsHandlers.MountAdmin(a)
			})
		})
	})

	return r
}

// providerTester implements settings.ProviderTester by composing the
// email + storage managers.
type providerTester struct {
	email   *email.Manager
	storage *storage.Manager
}

func (p providerTester) TestEmail(r *http.Request, to string, cfg settings.EmailConfig) error {
	return p.email.TestEmail(r, to, cfg)
}

func (p providerTester) TestStorage(r *http.Request, cfg settings.StorageConfig) error {
	return p.storage.TestStorage(r, cfg)
}

// bootstrapStorageFromEnv populates the `storage` settings row from
// legacy environment variables on first boot, so existing docker-compose
// deployments keep working without manual intervention.
func bootstrapStorageFromEnv(ctx context.Context, cfg *config.Config, store *settings.Store, logger *slog.Logger) {
	if cfg.S3AccessKey == "" || cfg.S3SecretKey == "" || cfg.S3Bucket == "" {
		return
	}
	cur, err := store.GetStorage(ctx)
	if err == nil && cur.AccessKey != "" {
		return // admin already configured it
	}
	driver := "s3_compatible"
	if cfg.S3Endpoint == "" {
		driver = "aws_s3"
	} else if containsAny(cfg.S3Endpoint, "minio") {
		driver = "minio"
	} else if containsAny(cfg.S3Endpoint, "r2.cloudflarestorage.com") {
		driver = "r2"
	}
	if err := store.Set(ctx, "storage", map[string]any{
		"enabled":          true,
		"driver":           driver,
		"bucket":           cfg.S3Bucket,
		"region":           cfg.S3Region,
		"endpoint":         cfg.S3Endpoint,
		"access_key":       cfg.S3AccessKey,
		"secret_key":       cfg.S3SecretKey,
		"public_base_url":  cfg.S3PublicBaseURL,
		"force_path_style": cfg.S3ForcePathStyle,
	}, settings.StorageSecretPaths, nil); err != nil {
		logger.Warn("settings: bootstrap storage from env failed", "err", err)
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if n != "" && len(haystack) >= len(n) {
			for i := 0; i+len(n) <= len(haystack); i++ {
				if haystack[i:i+len(n)] == n {
					return true
				}
			}
		}
	}
	return false
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
