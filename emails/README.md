# Tansiq email templates (React Email)

This workspace contains the **source** for every transactional email
sent by the Tansiq Information Portal. Templates are authored as React
components using [`react.email`](https://react.email) and compiled to
plain HTML files that are embedded into the Go binary.

The compiled output lives at
[`backend/internal/email/templates/`](../backend/internal/email/templates/).
The Go side fills `{{ .Foo }}` placeholders in those HTML files at
runtime, so authoring rule #1 is **leave Go template tokens intact**
when you edit a JSX file.

## Workflow

```bash
# 1. install once
npm install

# 2. preview locally with hot reload
npm run dev          # opens http://localhost:3300

# 3. export to the backend tree
npm run export       # writes ../backend/internal/email/templates/
```

After exporting, run `go build ./...` in the backend; the templates are
embedded via `go:embed` and ship inside the binary.

## Templates

| Name                       | Triggered when                               |
|----------------------------|----------------------------------------------|
| `user.welcome.html`        | A new user registers (account pending)       |
| `user.approved.html`       | An admin approves a pending account          |
| `user.rejected.html`       | An admin rejects a pending account           |
| `password.reset.html`      | The user requests a password reset           |
| `case.published.html`      | A case the user submitted has been published |
| `case.rejected.html`       | A case the user submitted needs changes      |
| `test.html`                | "Send test email" from the admin settings    |

## Variables

Every template receives:

- `SiteName` — branded site name (English)
- `SiteURL` — public frontend URL
- `Year` — current year (footer)
- `LogoURL` — optional logo URL configured by the admin
- `PrimaryColor` — branded primary colour, used for the CTA buttons

Per-template variables are documented in each `.tsx` file.

## Bilingual content

Subjects are computed in Go (see `email/manager.go::subjectFor`). Body
copy is currently authored in English; localised variants per recipient
language are on the roadmap.
