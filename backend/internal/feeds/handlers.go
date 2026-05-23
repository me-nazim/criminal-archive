// Package feeds renders the public RSS feed and sitemap.xml for the
// archive. They are served at the root (not under /api/v1) so search
// engines and feed readers find them at the conventional locations.
package feeds

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/cases"
	"github.com/me-nazim/criminal-archive/backend/internal/persons"
)

// Handlers wraps the feed dependencies.
type Handlers struct {
	cases       *cases.Repository
	persons     *persons.Repository
	siteBaseURL string
	logger      *slog.Logger
}

// NewHandlers constructs a Handlers. siteBaseURL should NOT have a
// trailing slash and is the public origin used for canonical URLs (e.g.
// `https://tansiq.org`).
func NewHandlers(c *cases.Repository, p *persons.Repository, siteBaseURL string, logger *slog.Logger) *Handlers {
	return &Handlers{
		cases:       c,
		persons:     p,
		siteBaseURL: strings.TrimRight(siteBaseURL, "/"),
		logger:      logger,
	}
}

// Mount registers /feed.xml and /sitemap.xml directly on the root mux.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/feed.xml", h.rss)
	r.Get("/sitemap.xml", h.sitemap)
}

// ---------------------------------------------------------- RSS

type rssRoot struct {
	XMLName xml.Name  `xml:"rss"`
	Version string    `xml:"version,attr"`
	Channel rssChan   `xml:"channel"`
}

type rssChan struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	PubDate     string    `xml:"pubDate"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

func (h *Handlers) rss(w http.ResponseWriter, r *http.Request) {
	rows, err := h.cases.ListPublic(r.Context(), cases.ListParams{Limit: 30})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC().Format(time.RFC1123Z)
	channel := rssChan{
		Title:       "Tansiq Information Portal",
		Link:        h.siteBaseURL,
		Description: "Verified, public documentation of crimes.",
		Language:    "bn",
		PubDate:     now,
	}
	for _, c := range rows {
		title := c.TitleBN
		if title == "" && c.TitleEN != nil {
			title = *c.TitleEN
		}
		desc := ""
		if c.SummaryBN != nil {
			desc = *c.SummaryBN
		} else if c.SummaryEN != nil {
			desc = *c.SummaryEN
		}
		pub := c.CreatedAt
		if c.PublishedAt != nil {
			pub = *c.PublishedAt
		}
		channel.Items = append(channel.Items, rssItem{
			Title:       title,
			Link:        h.siteBaseURL + "/cases/" + c.Slug,
			GUID:        c.CaseNumber,
			PubDate:     pub.UTC().Format(time.RFC1123Z),
			Description: desc,
		})
	}
	root := rssRoot{Version: "2.0", Channel: channel}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(root); err != nil {
		h.logger.Warn("rss encode", "err", err)
	}
}

// ---------------------------------------------------------- Sitemap

type smURLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []smURL  `xml:"url"`
}

type smURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
}

func (h *Handlers) sitemap(w http.ResponseWriter, r *http.Request) {
	caseRows, err := h.cases.ListPublic(r.Context(), cases.ListParams{Limit: 100})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	personRows, err := h.persons.ListPublic(r.Context(), persons.ListParams{Limit: 100})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	urls := []smURL{
		{Loc: h.siteBaseURL + "/", ChangeFreq: "daily"},
		{Loc: h.siteBaseURL + "/cases", ChangeFreq: "hourly"},
		{Loc: h.siteBaseURL + "/persons", ChangeFreq: "daily"},
	}
	for _, c := range caseRows {
		urls = append(urls, smURL{
			Loc:     fmt.Sprintf("%s/cases/%s", h.siteBaseURL, c.Slug),
			LastMod: c.UpdatedAt.UTC().Format("2006-01-02"),
		})
	}
	for _, p := range personRows {
		urls = append(urls, smURL{
			Loc:     fmt.Sprintf("%s/persons/%s", h.siteBaseURL, p.Slug),
			LastMod: p.UpdatedAt.UTC().Format("2006-01-02"),
		})
	}
	body := smURLSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(body); err != nil {
		h.logger.Warn("sitemap encode", "err", err)
	}
}
