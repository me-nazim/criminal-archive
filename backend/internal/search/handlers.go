// Package search exposes a single cross-resource search endpoint that
// fans out to cases and persons in parallel and merges the results. The
// endpoint is anonymous-readable; only published rows are returned.
package search

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/cases"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
	"github.com/me-nazim/criminal-archive/backend/internal/persons"
)

// Handlers wraps the search dependencies.
type Handlers struct {
	cases   *cases.Repository
	persons *persons.Repository
	logger  *slog.Logger
}

// NewHandlers constructs a Handlers.
func NewHandlers(c *cases.Repository, p *persons.Repository, logger *slog.Logger) *Handlers {
	return &Handlers{cases: c, persons: p, logger: logger}
}

// Mount registers GET /search.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/search", h.search)
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"q":       "",
			"cases":   []any{},
			"persons": []any{},
		})
		return
	}
	typ := r.URL.Query().Get("type") // empty = both, "case" or "person" narrow

	type caseRes struct {
		rows []cases.Case
		err  error
	}
	type personRes struct {
		rows []persons.Person
		err  error
	}
	cr := caseRes{}
	pr := personRes{}

	var wg sync.WaitGroup
	if typ == "" || typ == "case" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cr.rows, cr.err = h.cases.ListPublic(r.Context(), cases.ListParams{Search: q, Limit: 20})
		}()
	}
	if typ == "" || typ == "person" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pr.rows, pr.err = h.persons.ListPublic(r.Context(), persons.ListParams{Search: q, Limit: 20})
		}()
	}
	wg.Wait()

	if cr.err != nil || pr.err != nil {
		err := cr.err
		if err == nil {
			err = pr.err
		}
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"q":       q,
		"cases":   redactCases(cr.rows),
		"persons": redactPersons(pr.rows),
	})
}

func redactCases(rows []cases.Case) []cases.Case {
	out := make([]cases.Case, len(rows))
	for i, c := range rows {
		c.InternalNotes = nil
		c.SubmittedBy = nil
		c.ApprovedBy = nil
		out[i] = c
	}
	return out
}

func redactPersons(rows []persons.Person) []persons.Person {
	out := make([]persons.Person, len(rows))
	for i, p := range rows {
		p.InternalNotes = nil
		p.SubmittedBy = nil
		p.ApprovedBy = nil
		if p.IsAnonymous {
			p.FullNameBN = nil
			p.FullNameEN = nil
			p.PhotoURL = nil
			p.AddressLine = nil
			p.Aliases = nil
		}
		out[i] = p
	}
	return out
}

// silence: ensure context import is used in case we add ctx.WithTimeout later
var _ = context.Background
