package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"

	"github.com/me-nazim/criminal-archive/backend/internal/settings"
)

// smtpDriver speaks to an SMTP relay. We avoid third-party libraries to
// keep the dependency set lean — net/smtp + a small MIME builder is
// enough for HTML+text multipart messages.
type smtpDriver struct {
	cfg settings.EmailConfig
}

func newSMTP(cfg settings.EmailConfig) *smtpDriver { return &smtpDriver{cfg: cfg} }

func (d *smtpDriver) Name() string { return DriverSMTP }

func (d *smtpDriver) Send(ctx context.Context, m Message) (string, error) {
	host := d.cfg.SMTP.Host
	if host == "" {
		return "", errors.New("smtp: host is empty")
	}
	port := d.cfg.SMTP.Port
	if port == 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	body, err := buildMime(d.cfg, m)
	if err != nil {
		return "", err
	}

	from := d.cfg.FromAddress
	rcpts := append([]string{m.To}, m.CC...)
	rcpts = append(rcpts, m.BCC...)

	auth := smtp.PlainAuth("", d.cfg.SMTP.Username, d.cfg.SMTP.Password, host)
	dialer := &smtpDialer{addr: addr, host: host, useTLS: d.cfg.SMTP.UseTLS, startTLS: d.cfg.SMTP.StartTLS, auth: auth}

	errCh := make(chan error, 1)
	go func() { errCh <- dialer.send(from, rcpts, body) }()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errCh:
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("smtp-%d", time.Now().UnixNano()), nil
	}
}

// smtpDialer wraps net/smtp with TLS / STARTTLS modes.
type smtpDialer struct {
	addr     string
	host     string
	useTLS   bool
	startTLS bool
	auth     smtp.Auth
}

func (d *smtpDialer) send(from string, to []string, body []byte) error {
	var c *smtp.Client
	var err error
	if d.useTLS {
		conn, err := tls.Dial("tcp", d.addr, &tls.Config{ServerName: d.host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		c, err = smtp.NewClient(conn, d.host)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
	} else {
		c, err = smtp.Dial(d.addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
	}
	defer c.Close()

	if err := c.Hello("tansiq.local"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if d.startTLS && !d.useTLS {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: d.host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}
	if d.auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(d.auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	for _, r := range to {
		if r == "" {
			continue
		}
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", r, err)
		}
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := wc.Write(body); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return c.Quit()
}

// buildMime renders a multipart/alternative message with the right
// headers. We deliberately keep encoding logic small; for richer cases
// (attachments, calendar invites) consider go-mail later.
func buildMime(cfg settings.EmailConfig, m Message) ([]byte, error) {
	if m.To == "" {
		return nil, errors.New("smtp: recipient required")
	}
	from := cfg.FromAddress
	if cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", encodeHeader(cfg.FromName), cfg.FromAddress)
	}
	to := m.To
	if m.ToName != "" {
		to = fmt.Sprintf("%s <%s>", encodeHeader(m.ToName), m.To)
	}
	subject := encodeHeader(m.Subject)
	boundary := fmt.Sprintf("tansiq-bnd-%d", time.Now().UnixNano())

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	if len(m.CC) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(m.CC, ", "))
	}
	if cfg.ReplyTo != "" {
		fmt.Fprintf(&buf, "Reply-To: %s\r\n", cfg.ReplyTo)
	}
	if m.ReplyTo != "" {
		fmt.Fprintf(&buf, "Reply-To: %s\r\n", m.ReplyTo)
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)

	if m.TextBody != "" {
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n\r\n")
		buf.WriteString(b64(m.TextBody))
		buf.WriteString("\r\n")
	}
	if m.HTMLBody != "" {
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n\r\n")
		buf.WriteString(b64(m.HTMLBody))
		buf.WriteString("\r\n")
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes(), nil
}

func b64(s string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	// Wrap at 76 chars per RFC 2045.
	var w strings.Builder
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		w.WriteString(enc[i:end])
		w.WriteString("\r\n")
	}
	return w.String()
}

func encodeHeader(s string) string {
	if isASCII(s) {
		return s
	}
	return mime.QEncoding.Encode("UTF-8", s)
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}
