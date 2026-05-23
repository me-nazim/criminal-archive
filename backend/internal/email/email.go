// Package email provides a pluggable mailer interface (SMTP, Resend,
// Elastic Mail) keyed off live admin settings. Outbound mail is queued
// in `email_outbox` for durability and a small worker drains it.
//
// Templates live in ./templates and are pre-rendered HTML emitted by the
// react-email build pipeline (see emails/ at the repo root). At runtime
// the package fills `{{ . }}` Go-template placeholders in the HTML and
// produces a plain-text fallback.
package email

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/me-nazim/criminal-archive/backend/internal/settings"
)

// Driver names recognised by the manager.
const (
	DriverSMTP        = "smtp"
	DriverResend      = "resend"
	DriverElasticMail = "elastic_mail"
)

// Message is a render-ready email document.
type Message struct {
	To       string
	ToName   string
	CC       []string
	BCC      []string
	Subject  string
	HTMLBody string
	TextBody string
	ReplyTo  string

	// Used by the outbox worker for retries / audit.
	Template string
	Payload  map[string]any
}

// Mailer is implemented by every driver.
type Mailer interface {
	Name() string
	Send(ctx context.Context, m Message) (providerMessageID string, err error)
}

// FromConfig instantiates a Mailer based on cfg.Provider.
func FromConfig(cfg settings.EmailConfig) (Mailer, error) {
	switch cfg.Provider {
	case DriverSMTP:
		return newSMTP(cfg), nil
	case DriverResend:
		return newResend(cfg), nil
	case DriverElasticMail:
		return newElasticMail(cfg), nil
	default:
		return nil, fmt.Errorf("email: unknown provider %q", cfg.Provider)
	}
}

// ErrDisabled is returned when the configuration disables email sending.
// Callers (e.g. password-reset) should treat this as soft success — the
// recipient simply won't get a notification.
var ErrDisabled = errors.New("email: provider disabled")

// embeddedTemplates holds every template HTML file so the binary remains
// self-contained.
//
//go:embed templates/*.html
var embeddedTemplates embed.FS

// loadTemplate returns the bytes for templates/<name>.html.
func loadTemplate(name string) ([]byte, error) {
	b, err := fs.ReadFile(embeddedTemplates, "templates/"+name+".html")
	if err != nil {
		return nil, fmt.Errorf("email: template %q not found: %w", name, err)
	}
	return b, nil
}
