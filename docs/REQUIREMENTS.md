# Requirements — Tansiq Information Portal

> Status: **DRAFT (planning round 1)** · Owner: project lead · Last updated: 2026-05-23

This document captures **what** we are building and **why**. The "how" lives in
[`ARCHITECTURE.md`](./ARCHITECTURE.md), [`API_SPEC.md`](./API_SPEC.md), and
[`UI_DESIGN.md`](./UI_DESIGN.md). Sequencing of work lives in
[`ROADMAP.md`](./ROADMAP.md).

---

## 1. Vision

> A free, public, community-curated archive that preserves the truth of
> serious crimes in Bangladesh and beyond — verified evidence, named
> perpetrators, supported victims — when official narratives fail.

The portal is modelled on the spirit of the Internet Archive: anything that is
published is permanently accessible to everyone, with no login wall, and
downloadable as raw evidence.

## 2. Mission & guiding values

| Value             | What it means in practice                                                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------- |
| **Truth-first**   | No claim becomes public until a verification team has reviewed evidence.                                  |
| **Open by default** | Everything published is free to read and download. No paywalls, no signups for reading.                 |
| **Victim-respecting** | Submitter and victim identities are never public unless they choose to be; admins know, the world doesn't. |
| **Auditable**     | Every privileged action is logged and replayable. Disputes can be investigated.                           |
| **Lean**          | The system must run on modest infrastructure; we optimise for read-heavy traffic.                         |
| **Bilingual**     | Bengali is the default language; English is a first-class peer, not an afterthought.                      |

## 3. Personas

### 3.1 Visitor (anonymous public)
Wants to read about a specific case or person, browse recent incidents,
download evidence files, and verify claims independently. **No account.**

### 3.2 Contributor
A registered, admin-approved user who submits cases. Can attach evidence,
edit their own drafts, see the verification status of their submissions.

### 3.3 Verifier (member of verification team)
A registered, admin-approved user with the `moderator` role. Reviews
submitted cases against attached evidence, flags inconsistencies, marks
verification status (`verified`, `rejected`, `needs more info`).

### 3.4 Admin
Approves new user accounts, assigns verifiers to cases, publishes verified
cases, edits internal notes, manages crime types and locations metadata,
and views audit logs.

### 3.5 Super-admin
Same as admin **plus** can promote/demote admins themselves and access
sensitive system settings (storage credentials, retention policies).

## 4. User stories

Stories are written as `As <persona>, I want <capability>, so that <benefit>.`
Each is tagged for traceability against [Functional Requirements](#5-functional-requirements).

### 4.1 Visitor
- **US-V-1** As a visitor, I want to browse a paginated, filterable list of
  published cases so that I can find incidents in my area or category.
  *(→ FR-CASE-LIST, FR-SEARCH)*
- **US-V-2** As a visitor, I want to read a case's full details — timeline,
  evidence, linked persons — without an account, so that I can verify the
  story for myself. *(→ FR-CASE-DETAIL, FR-FILE-PUBLIC)*
- **US-V-3** As a visitor, I want to download all public attachments of a
  case as a single zip so that I can preserve them locally.
  *(→ FR-FILE-BULK-DOWNLOAD)*
- **US-V-4** As a visitor, I want to click on an accused person's name and
  see every other case they appear in. *(→ FR-PERSON-PROFILE)*
- **US-V-5** As a visitor, I want to switch between Bengali and English
  with a single click and have my choice remembered.
  *(→ FR-I18N)*

### 4.2 Contributor
- **US-C-1** As a contributor, I want to register an account so that I can
  start submitting information. *(→ FR-AUTH-REGISTER)*
- **US-C-2** As a contributor, I want to submit a new case with title,
  description, location, date, linked persons, and evidence files, and save
  it as a draft to come back later. *(→ FR-CASE-CREATE, FR-FILE-UPLOAD)*
- **US-C-3** As a contributor, I want to see the status of every case I
  submitted (`draft`, `pending_review`, `in_verification`, `approved`,
  `published`, `rejected`). *(→ FR-CASE-MY-LIST)*
- **US-C-4** As a contributor, I want to attach links to news sources and
  optionally an archive.org snapshot URL. *(→ FR-NEWS-SOURCE)*

### 4.3 Verifier
- **US-VR-1** As a verifier, I want a queue of cases assigned to me, sorted
  by oldest-first. *(→ FR-VERIF-QUEUE)*
- **US-VR-2** As a verifier, I want to view all evidence (including hidden
  attachments) and add internal review notes that the public never sees.
  *(→ FR-VERIF-NOTES)*
- **US-VR-3** As a verifier, I want to mark a case `verified` or `rejected`
  with a reason. *(→ FR-VERIF-DECISION)*

### 4.4 Admin
- **US-A-1** As an admin, I want to see all pending user registrations and
  approve / reject them with one click. *(→ FR-USER-APPROVE)*
- **US-A-2** As an admin, I want to assign a verified case to a verifier
  and track verification progress. *(→ FR-VERIF-ASSIGN)*
- **US-A-3** As an admin, I want to publish a verified case, schedule its
  publication, or send it back for revision. *(→ FR-CASE-PUBLISH)*
- **US-A-4** As an admin, I want to attach internal notes and hidden
  attachments that no public user can see, so that we can preserve
  sensitive context. *(→ FR-INTERNAL-NOTES, FR-FILE-HIDDEN)*
- **US-A-5** As an admin, I want to see who submitted a piece of content
  even if it's published anonymously. *(→ FR-SUBMITTER-VISIBILITY)*
- **US-A-6** As an admin, I want a chronological audit log of every
  privileged action with filters by user / type / date.
  *(→ FR-AUDIT-LOG)*
- **US-A-7** As an admin, I want to manage the list of crime types
  (categories) used for tagging cases. *(→ FR-CRIME-TYPE-MGMT)*

### 4.5 Super-admin
- **US-SA-1** As a super-admin, I want to promote another user to admin or
  demote them. *(→ FR-USER-ROLE-MGMT)*

## 5. Functional requirements

> Identifiers are stable; never renumber them.

### Auth & users
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-AUTH-REGISTER** | Email + password registration; account starts in `pending` status until admin approval. | P0 |
| **FR-AUTH-LOGIN** | Email + password login returns a short-lived access JWT (15 min) and an httpOnly refresh cookie (30 days). | P0 |
| **FR-AUTH-REFRESH** | Refresh endpoint rotates the refresh token (single-use, hashed in DB). | P0 |
| **FR-AUTH-LOGOUT** | Logout revokes the current refresh token row in `sessions`. | P0 |
| **FR-AUTH-ME** | `/api/v1/me` returns the authenticated user. | P0 |
| **FR-USER-APPROVE** | Admins can list and approve / reject pending users. | P0 |
| **FR-USER-ROLE-MGMT** | Super-admins can change another user's role within constraints (only super-admin can create super-admins). | P1 |
| **FR-USER-SUSPEND** | Admins can suspend or reactivate any non-admin account. | P1 |

### Cases
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-CASE-CREATE** | Authenticated contributors can create a case in `draft` status. | P0 |
| **FR-CASE-UPDATE** | Authors can update their own drafts; admins can edit any case. Edits to a `published` case generate an audit-log entry. | P0 |
| **FR-CASE-SUBMIT** | A draft can be transitioned to `pending_review` (locks editing for the contributor). | P0 |
| **FR-CASE-LIST** | Public list of `published` cases with pagination, filtering by country/division/district/upazila/crime_type/year/tag, sortable by date. | P0 |
| **FR-CASE-DETAIL** | Public detail page for a `published` case including title, description, timeline, public attachments, linked persons, news sources. | P0 |
| **FR-CASE-MY-LIST** | Authenticated contributors see all of their own cases regardless of status. | P0 |
| **FR-CASE-PUBLISH** | Admins can publish a verified case (sets `status='published'`, `published_at=now()`). | P0 |
| **FR-CASE-UNPUBLISH** | Admins can move a published case back to `archived` (no longer publicly listed but URL still resolves with a "no longer published" notice). | P1 |
| **FR-CASE-NUMBER** | Each case is assigned a globally unique, human-readable number `TIP-{YYYY}-{seq:05d}` at the moment its first attachment is uploaded (or at submission, whichever comes first). | P0 |

### Persons
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-PERSON-CREATE** | Persons (victim / accused / witness) can be created and edited by contributors and above; published only after admin approval. | P0 |
| **FR-PERSON-PROFILE** | Public profile shows public bio, photo, occupation, location, and a list of cases the person is linked to. Internal notes are admin-only. | P0 |
| **FR-PERSON-LINK** | A person can be linked to multiple cases via `case_persons` with a role (`victim`/`accused`/`witness`/`other`). | P0 |
| **FR-PERSON-ANONYMISE** | Victims can be marked `is_anonymous = true`; their name and photo never render publicly even after publication. | P0 |
| **FR-PERSON-MERGE** | Admins can merge duplicate person records. | P2 |

### Files / evidence
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-FILE-UPLOAD** | Files upload directly to object storage via presigned PUT URL. The API never proxies large file bytes. | P0 |
| **FR-FILE-NAMING** | Stored filename matches `TIP-{YYYY}-{seq:05d}_{kind}_{idx:02d}.{ext}` and is deterministic per case. Original filename is preserved separately. | P0 |
| **FR-FILE-PUBLIC** | `kind=public` files are served from the public CDN with long cache headers. | P0 |
| **FR-FILE-HIDDEN** | `kind=hidden` files are NOT listed for the public; admins fetch them via short-lived presigned download URLs. | P0 |
| **FR-FILE-INTERNAL** | `kind=internal` files are admin-only; never linked from any public response. | P0 |
| **FR-FILE-BULK-DOWNLOAD** | Public endpoint returns a streamed zip of all public attachments of a case, named after the case number. | P1 |
| **FR-FILE-VIRUS-SCAN** | Uploaded files are scanned by ClamAV (or equivalent) before becoming public. Failed scans are quarantined. | P1 |

### Locations & taxonomy
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-LOCATION-BD** | Bangladesh location hierarchy (8 divisions / 64 districts / ~495 upazilas) is seeded with bilingual names. | P0 |
| **FR-LOCATION-INTL** | Other countries are represented at country-level only with ISO codes; sub-units captured as free text. | P0 |
| **FR-LOCATION-CASCADE** | UI dropdowns cascade: country → division → district → upazila. | P0 |
| **FR-CRIME-TYPE-MGMT** | Admins can create / edit crime type categories with bilingual names and severity. | P0 |

### Verification & moderation
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-VERIF-ASSIGN** | Admins can assign a case in `pending_review` to a verifier; status moves to `in_verification`. | P0 |
| **FR-VERIF-QUEUE** | Verifiers see only cases assigned to them. | P0 |
| **FR-VERIF-NOTES** | Verifiers can add internal notes per case. | P0 |
| **FR-VERIF-DECISION** | Verifiers mark `verified` or `rejected` with a reason; rejection sends the case back to the contributor. | P0 |
| **FR-INTERNAL-NOTES** | Cases and persons carry an admin-only `internal_notes` field. | P0 |

### Search
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-SEARCH-BASIC** | Full-text-ish search over case title and person name (`pg_trgm`). | P0 |
| **FR-SEARCH-FILTERS** | Combine free-text query with structured filters (location, crime type, date range). | P0 |
| **FR-SEARCH-RANK** | Results ranked by recency + match quality. | P1 |

### Audit & logging
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-AUDIT-LOG** | Every privileged action (approve user, publish case, edit someone else's content, change role, hard-delete) writes an immutable `audit_logs` row. | P0 |
| **FR-AUDIT-VIEW** | Admins can view, filter, and export audit logs. | P1 |
| **FR-SUBMITTER-VISIBILITY** | Admins can always see the submitter of a piece of content; public never sees it unless the submitter explicitly chose a public attribution. | P0 |

### Internationalisation
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-I18N** | All UI strings translatable; default Bengali, English secondary. Language switcher persists across sessions. | P0 |
| **FR-I18N-CONTENT** | Case content fields exist in both `*_bn` and `*_en`; either may be empty. | P0 |
| **FR-I18N-SEO** | Per-language URLs and `<html lang>` updated correctly. | P1 |

### Misc
| ID | Requirement | Priority |
| -- | -- | :-: |
| **FR-COVER-IMAGE** | Each case may have a cover image (one of its public attachments). | P1 |
| **FR-TAGS** | Cases support free-form tags (string array). | P1 |
| **FR-RSS** | A public RSS feed of recently published cases. | P2 |
| **FR-SITEMAP** | Auto-generated `sitemap.xml`. | P1 |

## 6. Non-functional requirements

| ID | Requirement | Target |
| -- | -- | -- |
| **NFR-PERF-LCP** | Public list and detail pages must achieve Largest Contentful Paint < 2.5 s on a 3G connection from Bangladesh. | P0 |
| **NFR-PERF-API** | p95 read API latency under 200 ms (excluding cold cache). | P0 |
| **NFR-SCALE** | The system handles 200 concurrent read requests / second on a single 2 vCPU / 4 GB backend instance with Postgres on a separate node. | P0 |
| **NFR-AVAIL** | Target 99.5 % monthly uptime for the read path. | P1 |
| **NFR-DR-RPO** | Recovery Point Objective ≤ 24 h (daily backups). | P0 |
| **NFR-DR-RTO** | Recovery Time Objective ≤ 4 h. | P1 |
| **NFR-SECURITY-OWASP** | The API meets OWASP API Top 10 guidelines: rate limiting, input validation, no IDOR, no excessive data exposure. | P0 |
| **NFR-PRIVACY-PII** | No raw NID is ever stored. Only a salted hash of NID is stored if needed. Submitter identity is encrypted at rest? (open question; default: not encrypted but never publicly exposed). | P0 |
| **NFR-A11Y** | WCAG 2.1 AA compliance for all public pages. | P1 |
| **NFR-OBS-LOGS** | All API requests log a JSON line with request_id, user_id (if any), method, path, status, duration. | P0 |
| **NFR-OBS-METRICS** | Backend exports Prometheus metrics: request count, duration histogram, DB pool stats. | P1 |

## 7. Out of scope (v1)

- Real-time chat / messaging between users.
- Native mobile apps (iOS / Android). PWA install is acceptable for v1.
- Payment, donations, paid tiers.
- AI-generated case summaries or transcription.
- Multi-tenant deployments / white-label.
- A separate moderator-facing native desktop app.

## 8. Success metrics

We will declare v1 successful if, **6 months after public launch**, we have:

1. **≥ 100 published cases** across at least 30 districts.
2. **≥ 50 approved contributors** with at least one published submission.
3. **<24 h median time-to-verification** from submission to verifier decision.
4. **<7 day median time-to-publication** from submission to publish.
5. **0 verified takedowns** caused by inaccurate / unverified content
   reaching publication.
6. **p95 read API latency < 200 ms** sustained over a 7-day window.

## 9. Risks & mitigations

| Risk | Likelihood | Impact | Mitigation |
| -- | -- | -- | -- |
| Defamation lawsuits | High | High | Mandatory verification step; clear takedown policy; admin reviews; rate-limited submissions; never publish unverified accusations. |
| Coordinated disinformation campaign uses the platform | Medium | High | Verification team; admin approval before publish; submitter identity logged. |
| State-level censorship / DNS block | High | Medium | Cloudflare in front; mirror site / Tor onion as a fallback; data exportable. |
| Doxxing of submitters | Medium | High | Submitter PII never appears in public responses; tested with automated checks. |
| Storage cost runaway from large videos | Medium | Medium | R2 has no egress fees; lifecycle policy moves cold attachments to archive class. |
| Single-maintainer bus factor | High | High | All knowledge in this repo (`docs/`); deployment runbooks; secrets in a managed vault with break-glass access. |

## 10. Open questions

These are deliberately listed so we close them before the relevant phase
ships, not before this document is approved.

1. **Comments / corrections on cases** — open them up to the public, or
   keep the archive read-only? *(Decision needed before Phase 7.)*
2. **NID storage** — store salted hash, or drop entirely? *(Before Phase 3.)*
3. **Cross-references between cases** (same accused, same location) —
   build a graph view, or only "see other cases"? *(Before Phase 7.)*
4. **Submitter encryption-at-rest** — column-level encryption for
   `submitted_by` once published? *(Before Phase 5.)*
5. **Tor / .onion mirror** — in scope for v1 or later?
6. **Account types for orgs** — should journalist organisations have a
   verified-org badge? *(Post v1.)*
