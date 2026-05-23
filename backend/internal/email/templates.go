package email

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

// Render produces the HTML and a plain-text fallback for the named template.
//
// data is merged with the standard branding fields (`SiteName`,
// `Tagline`, `Year`) before execution so templates remain terse.
func Render(name string, data map[string]any) (htmlBody, textBody string, err error) {
	raw, err := loadTemplate(name)
	if err != nil {
		return "", "", err
	}
	t, err := template.New(name).Option("missingkey=zero").Parse(string(raw))
	if err != nil {
		return "", "", fmt.Errorf("email: parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("email: exec %s: %w", name, err)
	}
	html := buf.String()
	return html, htmlToText(html), nil
}

// htmlToText is a tiny stripper used to derive the plain-text part. It's
// intentionally simple — emails are short, and a heavy library is not
// worth the dep.
var (
	rxStyle  = regexp.MustCompile(`(?is)<(style|script)[^>]*>.*?</\s*\1\s*>`)
	rxBR     = regexp.MustCompile(`(?i)<br\s*/?>`)
	rxBlock  = regexp.MustCompile(`(?i)</(p|div|h[1-6]|tr|li)>`)
	rxTags   = regexp.MustCompile(`<[^>]+>`)
	rxWS     = regexp.MustCompile(`[ \t]+`)
	rxBlanks = regexp.MustCompile(`\n{3,}`)
)

func htmlToText(s string) string {
	s = rxStyle.ReplaceAllString(s, "")
	s = rxBR.ReplaceAllString(s, "\n")
	s = rxBlock.ReplaceAllString(s, "\n")
	s = rxTags.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = rxWS.ReplaceAllString(s, " ")
	s = rxBlanks.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
