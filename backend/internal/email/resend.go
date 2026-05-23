package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/me-nazim/criminal-archive/backend/internal/settings"
)

// resendDriver speaks to Resend's REST API
// (https://resend.com/docs/api-reference/emails/send-email).
type resendDriver struct {
	cfg    settings.EmailConfig
	client *http.Client
}

func newResend(cfg settings.EmailConfig) *resendDriver {
	return &resendDriver{cfg: cfg, client: &http.Client{Timeout: 20 * time.Second}}
}

func (d *resendDriver) Name() string { return DriverResend }

type resendReq struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Cc      []string `json:"cc,omitempty"`
	Bcc     []string `json:"bcc,omitempty"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

type resendResp struct {
	ID    string `json:"id"`
	Error struct {
		Message string `json:"message"`
		Name    string `json:"name"`
	} `json:"error"`
	Message string `json:"message"` // older shape for some errors
}

func (d *resendDriver) Send(ctx context.Context, m Message) (string, error) {
	if d.cfg.Resend.APIKey == "" {
		return "", errors.New("resend: api key is empty")
	}
	from := d.cfg.FromAddress
	if d.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", d.cfg.FromName, d.cfg.FromAddress)
	}
	body, err := json.Marshal(resendReq{
		From: from, To: []string{m.To}, Cc: m.CC, Bcc: m.BCC,
		Subject: m.Subject, HTML: m.HTMLBody, Text: m.TextBody,
		ReplyTo: nonEmpty(m.ReplyTo, d.cfg.ReplyTo),
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+d.cfg.Resend.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resend http: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var parsed resendResp
	_ = json.Unmarshal(raw, &parsed)
	if res.StatusCode >= 400 {
		msg := parsed.Error.Message
		if msg == "" {
			msg = parsed.Message
		}
		if msg == "" {
			msg = string(raw)
		}
		return "", fmt.Errorf("resend %d: %s", res.StatusCode, msg)
	}
	return parsed.ID, nil
}

func nonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
