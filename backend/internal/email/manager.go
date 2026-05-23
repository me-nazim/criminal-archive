package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/settings"
)

// Manager owns the live mailer, refreshes it when settings change, and
// coordinates outbox enqueue + drain.
type Manager struct {
	pool   *pgxpool.Pool
	store  *settings.Store
	logger *slog.Logger

	branding settings.Branding
	siteURL  string

	mu          sync.RWMutex
	mailer      Mailer
	cfgVersion  uint64
	cfgSnapshot settings.EmailConfig
}

// NewManager constructs a Manager. siteURL is the public frontend URL
// used to build action links inside templates.
func NewManager(pool *pgxpool.Pool, store *settings.Store, siteURL string, logger *slog.Logger) *Manager {
	return &Manager{pool: pool, store: store, siteURL: siteURL, logger: logger}
}

// Refresh reloads config from settings and rebuilds the mailer.
func (m *Manager) Refresh(ctx context.Context) error {
	cfg, err := m.store.GetEmail(ctx)
	if err != nil && !errors.Is(err, settings.ErrNotFound) {
		return err
	}
	br, _ := m.store.GetBranding(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfgSnapshot = cfg
	m.branding = br
	m.cfgVersion = m.store.Version()
	if !cfg.Enabled {
		m.mailer = nil
		return nil
	}
	mailer, err := FromConfig(cfg)
	if err != nil {
		m.mailer = nil
		return err
	}
	m.mailer = mailer
	return nil
}

// snapshot returns the current config (and ensures the mailer is fresh
// when settings have changed since the last call).
func (m *Manager) snapshot(ctx context.Context) (settings.EmailConfig, settings.Branding, Mailer) {
	m.mu.RLock()
	cur := m.cfgVersion
	live := m.store.Version()
	m.mu.RUnlock()
	if cur != live {
		_ = m.Refresh(ctx)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfgSnapshot, m.branding, m.mailer
}

// SendNow renders the template and sends synchronously. Returns a
// provider message ID on success.
func (m *Manager) SendNow(ctx context.Context, to, template string, data map[string]any) (string, error) {
	cfg, br, mailer := m.snapshot(ctx)
	if !cfg.Enabled || mailer == nil {
		return "", ErrDisabled
	}
	html, text, subject, err := m.render(template, data, br)
	if err != nil {
		return "", err
	}
	return mailer.Send(ctx, Message{
		To: to, Subject: subject, HTMLBody: html, TextBody: text,
		Template: template, Payload: data,
	})
}

// Enqueue persists an outbox row scheduled for now. The worker drains it.
func (m *Manager) Enqueue(ctx context.Context, to, subject, template string, data map[string]any) error {
	_, br, _ := m.snapshot(ctx)
	html, text, computedSubject, err := m.render(template, data, br)
	if err != nil {
		return err
	}
	if subject == "" {
		subject = computedSubject
	}
	payload, _ := json.Marshal(data)
	const stmt = `
INSERT INTO email_outbox (to_address, subject, template, payload, html_body, text_body)
VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = m.pool.Exec(ctx, stmt, to, subject, template, payload, html, text)
	return err
}

// StartWorker launches a goroutine that drains queued emails until ctx
// is cancelled.
func (m *Manager) StartWorker(ctx context.Context) {
	go m.workerLoop(ctx)
}

func (m *Manager) workerLoop(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := m.drainOnce(ctx); err != nil && m.logger != nil {
				m.logger.Warn("email worker: drain failed", "err", err)
			}
		}
	}
}

func (m *Manager) drainOnce(ctx context.Context) error {
	cfg, _, mailer := m.snapshot(ctx)
	if !cfg.Enabled || mailer == nil {
		return nil
	}
	const fetch = `
UPDATE email_outbox
SET    status = 'sending', updated_at = now()
WHERE  id = (
  SELECT id FROM email_outbox
  WHERE  status IN ('queued', 'failed')
    AND  attempts < 5
    AND  scheduled_for <= now()
  ORDER BY scheduled_for
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING id, to_address, subject, template, payload, html_body, text_body, attempts`
	for {
		var (
			id       uuid.UUID
			to, sub  string
			tmpl     string
			payload  []byte
			html, tx string
			attempts int
		)
		err := m.pool.QueryRow(ctx, fetch).Scan(&id, &to, &sub, &tmpl, &payload, &html, &tx, &attempts)
		if err != nil {
			if isNoRows(err) {
				return nil
			}
			return err
		}
		mid, sendErr := mailer.Send(ctx, Message{
			To: to, Subject: sub, HTMLBody: html, TextBody: tx,
			Template: tmpl,
		})
		if sendErr != nil {
			delay := time.Duration(1<<attempts) * 30 * time.Second
			_, _ = m.pool.Exec(ctx, `
UPDATE email_outbox SET status = 'failed', last_error = $2, attempts = attempts + 1,
       scheduled_for = now() + $3::interval, updated_at = now() WHERE id = $1`,
				id, sendErr.Error(), fmt.Sprintf("%d seconds", int(delay.Seconds())))
			if m.logger != nil {
				m.logger.Warn("email send failed", "id", id, "err", sendErr)
			}
			continue
		}
		_, _ = m.pool.Exec(ctx, `
UPDATE email_outbox SET status = 'sent', sent_at = now(), provider = $2, provider_msg_id = $3,
       attempts = attempts + 1, updated_at = now() WHERE id = $1`,
			id, mailer.Name(), mid)
	}
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// ---- rendering -------------------------------------------------------

func (m *Manager) render(name string, data map[string]any, br settings.Branding) (html, text, subject string, err error) {
	if data == nil {
		data = map[string]any{}
	}
	// Standard fields available to every template.
	if _, ok := data["SiteName"]; !ok {
		data["SiteName"] = chooseSiteName(br)
	}
	if _, ok := data["SiteURL"]; !ok {
		data["SiteURL"] = m.siteURL
	}
	if _, ok := data["Year"]; !ok {
		data["Year"] = time.Now().Year()
	}
	if _, ok := data["LogoURL"]; !ok {
		data["LogoURL"] = br.LogoURL
	}
	if _, ok := data["PrimaryColor"]; !ok {
		if br.PrimaryColor != "" {
			data["PrimaryColor"] = br.PrimaryColor
		} else {
			data["PrimaryColor"] = "#e8501f"
		}
	}
	html, text, err = Render(name, data)
	if err != nil {
		return "", "", "", err
	}
	subject = subjectFor(name, br, data)
	return
}

func chooseSiteName(br settings.Branding) string {
	if br.SiteNameEN != "" {
		return br.SiteNameEN
	}
	if br.SiteNameBN != "" {
		return br.SiteNameBN
	}
	return "Tansiq Information Portal"
}

// subjectFor produces a default subject line per template. If callers
// pass `Subject` in data, that wins.
func subjectFor(name string, br settings.Branding, data map[string]any) string {
	if s, ok := data["Subject"].(string); ok && s != "" {
		return s
	}
	site := chooseSiteName(br)
	switch name {
	case "user.welcome":
		return fmt.Sprintf("Welcome to %s — pending review", site)
	case "user.approved":
		return fmt.Sprintf("Your %s account is now active", site)
	case "user.rejected":
		return fmt.Sprintf("Your %s registration was not approved", site)
	case "password.reset":
		return fmt.Sprintf("Reset your %s password", site)
	case "case.published":
		return fmt.Sprintf("A case you submitted has been published on %s", site)
	case "case.rejected":
		return fmt.Sprintf("Your case submission on %s needs changes", site)
	case "test":
		return fmt.Sprintf("[Test] %s email configuration", site)
	}
	return site + " — notification"
}

// ---- ProviderTester implementation ----------------------------------

// TestEmail satisfies settings.ProviderTester. The handler builds an
// ad-hoc mailer from the supplied config and sends a fixed template.
func (m *Manager) TestEmail(_ *http.Request, to string, cfg settings.EmailConfig) error {
	if to == "" {
		return errors.New("recipient is required")
	}
	cfg.Enabled = true
	mailer, err := FromConfig(cfg)
	if err != nil {
		return err
	}
	br, _ := m.store.GetBranding(context.Background())
	html, text, _, err := m.render("test", map[string]any{
		"Provider": cfg.Provider,
		"Subject":  fmt.Sprintf("Test from %s", chooseSiteName(br)),
	}, br)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = mailer.Send(ctx, Message{
		To: to, Subject: subjectFor("test", br, nil),
		HTMLBody: html, TextBody: text, Template: "test",
	})
	return err
}
