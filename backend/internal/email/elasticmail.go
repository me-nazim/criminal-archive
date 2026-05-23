package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/me-nazim/criminal-archive/backend/internal/settings"
)

// elasticMailDriver targets Elastic Email's v4 transactional API
// (https://elasticemail.com/developers/api-documentation/rest-api).
type elasticMailDriver struct {
	cfg    settings.EmailConfig
	client *http.Client
}

func newElasticMail(cfg settings.EmailConfig) *elasticMailDriver {
	return &elasticMailDriver{cfg: cfg, client: &http.Client{Timeout: 20 * time.Second}}
}

func (d *elasticMailDriver) Name() string { return DriverElasticMail }

// Elastic Email v4 schema (transactional emails endpoint).
type elasticEmailReq struct {
	Recipients []elasticRecipient `json:"Recipients"`
	Content    elasticContent     `json:"Content"`
}

type elasticRecipient struct {
	Email string `json:"Email"`
	Field map[string]string `json:"Fields,omitempty"`
}

type elasticContent struct {
	From         string         `json:"From"`
	ReplyTo      string         `json:"ReplyTo,omitempty"`
	Subject      string         `json:"Subject"`
	Body         []elasticBody  `json:"Body"`
	EnvelopeFrom string         `json:"EnvelopeFrom,omitempty"`
}

type elasticBody struct {
	ContentType string `json:"ContentType"` // HTML | PlainText
	Content     string `json:"Content"`
	Charset     string `json:"Charset"`
}

type elasticResp struct {
	MessageID string `json:"MessageID"`
	Error     string `json:"Error"`
}

func (d *elasticMailDriver) Send(ctx context.Context, m Message) (string, error) {
	if d.cfg.ElasticMail.APIKey == "" {
		return "", errors.New("elastic_mail: api key is empty")
	}
	from := d.cfg.FromAddress
	if d.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", d.cfg.FromName, d.cfg.FromAddress)
	}
	body := elasticEmailReq{
		Recipients: []elasticRecipient{{Email: m.To}},
		Content: elasticContent{
			From:    from,
			ReplyTo: nonEmpty(m.ReplyTo, d.cfg.ReplyTo),
			Subject: m.Subject,
			Body:    []elasticBody{},
		},
	}
	if m.HTMLBody != "" {
		body.Content.Body = append(body.Content.Body, elasticBody{ContentType: "HTML", Content: m.HTMLBody, Charset: "UTF-8"})
	}
	if m.TextBody != "" {
		body.Content.Body = append(body.Content.Body, elasticBody{ContentType: "PlainText", Content: m.TextBody, Charset: "UTF-8"})
	}
	for _, cc := range m.CC {
		body.Recipients = append(body.Recipients, elasticRecipient{Email: cc})
	}
	for _, bcc := range m.BCC {
		body.Recipients = append(body.Recipients, elasticRecipient{Email: bcc})
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(d.cfg.ElasticMail.BaseURL, "/")
	if base == "" {
		base = "https://api.elasticemail.com/v4"
	}
	url := base + "/emails/transactional"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-ElasticEmail-ApiKey", d.cfg.ElasticMail.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("elastic_mail http: %w", err)
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(res.Body)
	var parsed elasticResp
	_ = json.Unmarshal(respBody, &parsed)
	if res.StatusCode >= 400 {
		msg := parsed.Error
		if msg == "" {
			msg = string(respBody)
		}
		return "", fmt.Errorf("elastic_mail %d: %s", res.StatusCode, msg)
	}
	return parsed.MessageID, nil
}
