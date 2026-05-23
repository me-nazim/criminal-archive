# Roadmap — Tansiq Information Portal

> Status: **DRAFT (planning round 1)** · Last updated: 2026-05-23

This roadmap turns the requirements and design documents into an ordered
sequence of pull requests. It is the **single place** to look when asking
"what are we doing next?". Update it as we ship.

Companion documents:
- [`REQUIREMENTS.md`](./REQUIREMENTS.md) — what we are building.
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — how it is built.
- [`API_SPEC.md`](./API_SPEC.md) — the API contract.
- [`UI_DESIGN.md`](./UI_DESIGN.md) — UI components and pages.

---

## 1. Working principles

- **Small PRs.** Each PR ships one phase milestone or a slice of one. We
  prefer 5 × 200-line PRs over 1 × 1000-line PR.
- **Vertical slices.** Where possible, a phase ends in a feature visible
  end-to-end (DB → API → UI), not a backend-only or frontend-only scaffold.
- **Tests on the critical path.** We do **not** add a test for every line.
  We do add tests for: auth flow, RBAC, file naming, state transitions,
  publish/unpublish, audit log writes.
- **Migrations are forward-only in practice.** Down migrations exist for
  emergency rollbacks but are never used for routine schema changes.
- **Docs are part of the change.** Any PR that changes the API or schema
  also updates `API_SPEC.md` / `ARCHITECTURE.md` in the same commit.
- **No feature flags in v1.** We ship behind branches; flags add complexity
  before we have the deployment maturity to use them.

## 2. Status legend

`☐` not started · `⏳` in progress · `🔍` in review · `✅` shipped

## 3. Phases

The roadmap is organised in 12 phases. Phase 0 is already shipped (the
scaffold PR). Phase X (this PR) ships the planning docs.

```
Ph0 ─▶ PhX ─▶ Ph1 ─▶ Ph2 ─▶ Ph3 ─▶ Ph4 ─▶ Ph5 ─▶ Ph6 ─▶ Ph7 ─▶ Ph8 ─▶ Ph9 ─▶ Ph10 ─▶ Ph11
 ✅      ⏳     ☐     ☐     ☐     ☐     ☐     ☐     ☐     ☐     ☐      ☐      ☐
```

### Phase 0 — Scaffolding ✅
**Status:** shipped (PR #1).
- Repo layout (backend/, frontend/, docs/, docker-compose.yml).
- Go API skeleton with `/health`, pgxpool, slog.
- React+Vite+Tailwind+i18n SPA shell.
- Initial PostgreSQL 18 schema (16 tables) verified up + down.

### Phase X — Planning docs ⏳
**This PR.** Adds `REQUIREMENTS.md`, `API_SPEC.md`, `UI_DESIGN.md`,
`ROADMAP.md`, and updates the root `README.md` to link them.
**Acceptance:** maintainers approve the planning round 1 documents.

### Phase 1 — Foundations: locations, crime types, migrator
**Goal:** all reference data is queryable and the migrations runner is part
of the API binary.

- [ ] PR-1.1 — Embed `golang-migrate` (or a small in-house runner) into
  the API binary; `cmd/api migrate up|down|status` subcommand.
- [ ] PR-1.2 — Seed data files: `seed/countries.json` (full ISO 3166),
  `seed/bd_locations.json` (8 / 64 / ~495 with bilingual names), and
  `seed/crime_types.json`.
- [ ] PR-1.3 — Seed loader command `cmd/api seed [--reset]`. Idempotent.
- [ ] PR-1.4 — Public read-only handlers: `GET /locations/countries`,
  `/divisions`, `/districts`, `/upazilas`, `/crime-types`. Cache headers.
- [ ] PR-1.5 — Frontend `LocationCascade` component wired against these
  endpoints. Storybook page for the component.

**Acceptance:** `docker compose up` followed by `seed` produces a database
ready to take case submissions; the `LocationCascade` works end-to-end on
the demo page.

### Phase 2 — Authentication & RBAC
**Goal:** real users, real sessions, real role enforcement.

- [ ] PR-2.1 — `auth` package: bcrypt hashing, JWT issuance + parsing,
  refresh-token table writes, `Authenticator` middleware.
- [ ] PR-2.2 — Endpoints: `register`, `login`, `refresh`, `logout`, `me`,
  `password/change`. Password reset wired but emails are stubbed (logged).
- [ ] PR-2.3 — `RequireRole(role…)` middleware; per-route gates.
- [ ] PR-2.4 — Admin user-management endpoints (`approve`, `reject`,
  `suspend`, `reactivate`, `role`).
- [ ] PR-2.5 — Frontend: `Login`, `Register`, `ForgotPassword`,
  `ResetPassword`, `/me` profile page, auth context + token refresh.
- [ ] PR-2.6 — Admin Approvals queue page (read-only list + 1-click
  approve/reject).

**Acceptance:** A new user can register, sit in `pending`, be approved by
an admin, log in, and call `me`. RBAC denies a `viewer` from any write
endpoint.

### Phase 3 — Persons (victim / accused / witness)
**Goal:** standalone person profiles that don't yet appear in any case.

- [ ] PR-3.1 — Repository + service for persons; create / update /
  approve. Slug generation strategy.
- [ ] PR-3.2 — Public endpoints: list, detail, `/persons/:id/cases`
  (returns `[]` for now).
- [ ] PR-3.3 — Admin person editor with internal notes, anonymise toggle,
  merge stub.
- [ ] PR-3.4 — Frontend: `/persons` index, `/persons/:slug` profile,
  `/admin/persons` table + editor.
- [ ] PR-3.5 — `PersonCombobox` async-search component (used later by the
  case form).

**Acceptance:** An admin can publish a person profile and visit
`/persons/:slug` anonymously. Anonymous victims render correctly.

### Phase 4 — Cases CRUD without files
**Goal:** end-to-end case submission and publication, minus evidence files.

- [ ] PR-4.1 — `cases` repository + service; case-number generator
  using `case_number_counters`; submission state machine.
- [ ] PR-4.2 — Endpoints: create, list (mine), detail, list (public),
  patch, submit, assign, verify, publish, unpublish.
- [ ] PR-4.3 — Linking persons to cases (`case_persons`).
- [ ] PR-4.4 — `case_timeline` and `news_sources` endpoints.
- [ ] PR-4.5 — Frontend submit wizard (steps 1, 2, 4 — files come in
  Phase 5).
- [ ] PR-4.6 — Frontend case detail page (read-only, no attachments
  section yet).
- [ ] PR-4.7 — Admin case list + editor.

**Acceptance:** A contributor can submit a case, an admin can publish it,
the public URL works, and the case shows up in `/cases`.

### Phase 5 — File storage (R2 / MinIO)
**Goal:** evidence files end-to-end with deterministic naming.

- [ ] PR-5.1 — `storage` package wrapping AWS SDK v2 with R2-compatible
  config. Health check on boot.
- [ ] PR-5.2 — Presigned PUT URL endpoint; sequence allocator that
  reserves an `(case_id, kind, sequence_no)` triple.
- [ ] PR-5.3 — Finalise endpoint that verifies the object, stores the
  attachment row, kicks an async checksum job.
- [ ] PR-5.4 — Public attachment listing on case detail; admin-only
  presigned download for `hidden` and `internal`.
- [ ] PR-5.5 — Bulk download zip stream for public attachments.
- [ ] PR-5.6 — Frontend submit wizard step 3 (FileDropzone + progress).
- [ ] PR-5.7 — Public EvidenceGallery + lightbox.
- [ ] PR-5.8 — Optional ClamAV integration behind a feature toggle.

**Acceptance:** A contributor uploads three images on a draft case; on
publish they appear in the public gallery with the canonical names.

### Phase 6 — Verification workflow
**Goal:** verifier role does its job through the UI.

- [ ] PR-6.1 — Verification queue endpoint; assignment endpoint.
- [ ] PR-6.2 — Decision endpoint (`verified`/`rejected` with reason).
- [ ] PR-6.3 — Admin-only "view as verifier" mode in the case editor.
- [ ] PR-6.4 — Frontend `/admin/verification` page + per-case verifier
  view.

**Acceptance:** Cases that fail verification go back to the submitter
with the reason visible in `/me/cases`.

### Phase 7 — Public site polish
**Goal:** make the public read experience competitive.

- [ ] PR-7.1 — `/cases` filters in URL, cursor pagination, empty/loading
  states.
- [ ] PR-7.2 — `/search?q=` cross-resource search.
- [ ] PR-7.3 — Sticky ToC + "report inaccuracy" floating button on case
  detail.
- [ ] PR-7.4 — `/methodology`, `/about`, `/contact` static pages
  (markdown-driven).
- [ ] PR-7.5 — `feed.xml` (RSS) and `sitemap.xml`.
- [ ] PR-7.6 — Open Graph / Twitter Card meta on case + person pages.

**Acceptance:** `web.dev/measure` scores ≥ 95 perf / 95 a11y / 100 best
practices on a published case URL.

### Phase 8 — Audit log + admin dashboard
**Goal:** every privileged action is logged and inspectable.

- [ ] PR-8.1 — `audit` package; helpers used by every privileged action
  in services.
- [ ] PR-8.2 — `GET /admin/audit` with filters; CSV export.
- [ ] PR-8.3 — Frontend `/admin/audit-log` page.
- [ ] PR-8.4 — `/admin/dashboard` metric cards.

**Acceptance:** Approving a user, publishing a case, deleting an
attachment, and changing a role each appear in the audit log with the
correct actor, target, and metadata.

### Phase 9 — Hardening
**Goal:** the system is safe to put on the open internet.

- [ ] PR-9.1 — Rate limiting middleware (per-IP, per-user, per-endpoint).
- [ ] PR-9.2 — Input validation pass with consistent error envelope.
- [ ] PR-9.3 — CSRF protection on cookie-based endpoints.
- [ ] PR-9.4 — Security headers (CSP, HSTS, X-Frame-Options) at the edge
  + nginx.
- [ ] PR-9.5 — Backups: scheduled `pg_dump` to R2 cold class; documented
  restore procedure.
- [ ] PR-9.6 — Observability: structured logs verified end-to-end,
  Prometheus `/metrics` endpoint, dashboard JSON checked into `ops/`.
- [ ] PR-9.7 — Load test (k6 or vegeta) with documented baseline.

**Acceptance:** OWASP API Top 10 self-checklist passes; load test sustains
NFR-SCALE numbers.

### Phase 10 — Production launch
**Goal:** real domain, real users, real traffic.

- [ ] PR-10.1 — Production `docker-compose.prod.yml` and Caddy / nginx
  edge config.
- [ ] PR-10.2 — CI/CD: GitHub Actions builds + pushes images on tag,
  deploys to the prod host.
- [ ] PR-10.3 — DNS, Cloudflare proxy, R2 bucket, managed Postgres set
  up; runbook in `ops/RUNBOOK.md`.
- [ ] PR-10.4 — First batch of seed content; "soft launch" announcement.

**Acceptance:** the live URL serves verified content; a smoke-test
contributor flow completes successfully.

### Phase 11 — Post-launch backlog (post-v1)
- Comments / corrections from public.
- Cross-reference graph view.
- PWA install prompt + offline reading of bookmarked cases.
- Tor / .onion mirror.
- Map view of cases by district.
- Verified-organisation badges.
- Webhook system for partners.
- Native mobile apps.

## 4. Dependency graph

```
Ph1 (locations, migrator)
  └─▶ Ph2 (auth)
        ├─▶ Ph3 (persons)
        │     └─▶ Ph4 (cases without files)
        │           └─▶ Ph5 (files) ──▶ Ph6 (verification)
        │                                    └─▶ Ph7 (public polish)
        └─▶ Ph8 (audit + dashboard)  ──────────┘
                                               └─▶ Ph9 (hardening) ─▶ Ph10 (launch)
```

Phase 8 can run in parallel with Phase 6/7 if we have bandwidth.

## 5. Risk register (active)

The full risk register lives in `REQUIREMENTS.md §9`. The risks tracked
during execution are:

- **R-DEF-1** Defamation lawsuit triggered by an unverified case slipping
  through. Mitigation: enforce `published` requires both `approved` and
  a `verified` assignment row.
- **R-DOX-1** Submitter PII leak via API response. Mitigation: serializer
  tests in CI for every endpoint, asserting submitter fields are absent.
- **R-FILE-1** Malicious upload. Mitigation: ClamAV behind a feature
  toggle in Phase 5.8; quarantine bucket.
- **R-LEGAL-1** Government takedown. Mitigation: data export tool;
  off-jurisdiction R2; documented mirror procedure.

## 6. Definition of done (per PR)

A PR may be merged once **all** of the following are true:

1. CI green (lint + test + build for both backend and frontend, where
   applicable).
2. The relevant docs are updated in the same PR (no "docs to follow").
3. Manual smoke test of the affected user-visible flow has been performed
   (and a 1-line note saying so is in the PR description).
4. No new linter warnings.
5. No `TODO` comments without a tracking issue or follow-up PR linked.
