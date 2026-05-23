// Package router wires HTTP routes for the API.
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

	"github.com/me-nazim/criminal-archive/backend/internal/config"
)

// New constructs the top-level HTTP handler.
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

	r.Route("/api/v1", func(api chi.Router) {
		// TODO: mount auth, cases, persons, files, locations, admin handlers here.
		api.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"message": "pong"})
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
