# API Specification — Tansiq Information Portal

> Status: **DRAFT (planning round 1)** · Version: `v1` · Last updated: 2026-05-23

This is the contract between the React frontend, third-party integrators, and
the Go backend. It is paired with [`REQUIREMENTS.md`](./REQUIREMENTS.md) and
[`ARCHITECTURE.md`](./ARCHITECTURE.md).

The API is intentionally **REST + JSON**. There is no GraphQL layer. Endpoints
that take or return collections always paginate using cursor-based pagination
unless explicitly noted.

---

## 1. Conventions

### 1.1 Base URL & versioning

```
https://api.tansiq.org/api/v1
```

The path prefix `/api/v1` is the only versioning we expose. Breaking changes
ship as `/api/v2`. Additive changes happen in place.

### 1.2 Content type

- All requests and responses are `application/json; charset=utf-8` unless the
  endpoint explicitly serves binary (e.g. zip downloads).
- Date / time fields are RFC 3339 UTC strings (`2026-05-23T11:42:09Z`).
- Date-only fields are ISO 8601 calendar dates (`2026-05-23`).

### 1.3 Authentication

- **Public endpoints** (case browsing, person profiles, public file URLs) are
  unauthenticated. They may be served from CDN cache.
- **Authenticated endpoints** require a `Bearer <access_token>` header.
- The **refresh token** lives in an `httpOnly`, `Secure`, `SameSite=Lax`
  cookie named `tip_refresh`. The frontend never reads it.
- Access tokens are JWTs signed with HS256, TTL 15 min. Claims:

  ```json
  {
    "sub": "<user uuid>",
    "role": "admin",
    "iat": 1716461129,
    "exp": 1716462029,
    "jti": "<random>"
  }
  ```

### 1.4 Errors

Errors always follow this shape:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Field 'incident_date' is required.",
    "fields": {
      "incident_date": "required"
    },
    "request_id": "req_01HRX..."
  }
}
```

| HTTP | `code` examples |
| ---: | --- |
| 400 | `validation_error`, `bad_request` |
| 401 | `unauthenticated`, `token_expired` |
| 403 | `forbidden`, `account_pending`, `account_suspended` |
| 404 | `not_found` |
| 409 | `conflict`, `state_transition_invalid` |
| 422 | `unprocessable_entity` |
| 429 | `rate_limited` |
| 500 | `internal_error` |
| 503 | `service_unavailable` |

### 1.5 Pagination

Cursor-based:

```
GET /api/v1/cases?limit=20&cursor=eyJpZCI6IjAxSFJYLi4uIn0
```

Response:

```json
{
  "data": [ /* ... 20 items ... */ ],
  "page": {
    "next_cursor": "eyJpZCI6IjAxSFJZLi4uIn0",
    "has_more": true,
    "limit": 20
  }
}
```

`limit` defaults to 20 and is capped at 100.

### 1.6 Rate limits

Public read: 60 req/min/IP. Mutations: 20 req/min/user. Login attempts: 5
req/min/IP and 10/hour/email.

Headers on every response:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 58
X-RateLimit-Reset: 1716462129
```

### 1.7 Idempotency

POST endpoints that create resources accept an optional
`Idempotency-Key: <uuid>` header. Replays within 24 h return the original
response.

---

## 2. Endpoint catalogue

Legend for permissions:
🌐 public · 👤 any logged-in user · ✏️ contributor+ · 🔍 moderator+ · 🛠 admin+ · 👑 super-admin

### 2.1 Health

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`  | `/health` | 🌐 | Liveness + DB ping. |
| `GET`  | `/api/v1/version` | 🌐 | Build SHA, version, server time. |

### 2.2 Auth

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `POST` | `/api/v1/auth/register` | 🌐 | Create a `pending` account. |
| `POST` | `/api/v1/auth/login`    | 🌐 | Issue access JWT + refresh cookie. |
| `POST` | `/api/v1/auth/refresh`  | 🌐 (cookie) | Rotate refresh + new access JWT. |
| `POST` | `/api/v1/auth/logout`   | 👤 | Revoke current refresh token. |
| `GET`  | `/api/v1/auth/me`       | 👤 | Current user. |
| `POST` | `/api/v1/auth/password/change` | 👤 | Change password. |
| `POST` | `/api/v1/auth/password/reset/request` | 🌐 | Email reset link. |
| `POST` | `/api/v1/auth/password/reset/confirm` | 🌐 | Set new password using token. |

#### `POST /auth/register`
```json
// req
{ "email": "...", "password": "...", "full_name": "...", "phone": "+8801..." }

// 201
{ "id": "<uuid>", "email": "...", "status": "pending", "role": "contributor" }
```

#### `POST /auth/login`
```json
// req
{ "email": "...", "password": "..." }

// 200
{
  "access_token": "<jwt>",
  "expires_in": 900,
  "user": { "id": "...", "email": "...", "role": "contributor", "status": "approved" }
}
// + Set-Cookie: tip_refresh=...; HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth
```

### 2.3 Users (admin)

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`    | `/api/v1/admin/users` | 🛠 | List users (filter by role / status / search). |
| `GET`    | `/api/v1/admin/users/{id}` | 🛠 | Get a user. |
| `POST`   | `/api/v1/admin/users/{id}/approve` | 🛠 | Move pending → approved. |
| `POST`   | `/api/v1/admin/users/{id}/reject`  | 🛠 | Move pending → rejected. |
| `POST`   | `/api/v1/admin/users/{id}/suspend` | 🛠 | approved → suspended. |
| `POST`   | `/api/v1/admin/users/{id}/reactivate` | 🛠 | suspended → approved. |
| `PATCH`  | `/api/v1/admin/users/{id}/role` | 👑 | Change role. Only super-admin can grant super-admin. |

### 2.4 Cases — public read

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`  | `/api/v1/cases` | 🌐 | Paginated list of `published` cases. |
| `GET`  | `/api/v1/cases/{slug-or-number}` | 🌐 | Case detail (published only for public). |
| `GET`  | `/api/v1/cases/{id}/timeline` | 🌐 | Public timeline events. |
| `GET`  | `/api/v1/cases/{id}/attachments` | 🌐 | Public attachments only. |
| `GET`  | `/api/v1/cases/{id}/attachments.zip` | 🌐 | Streamed zip of public attachments. |
| `GET`  | `/api/v1/cases/{id}/news-sources` | 🌐 | News links. |

Filters supported by `GET /cases`:
```
country_id, division_id, district_id, upazila_id,
crime_type_id, year, tag, q (free text), sort=published_desc|incident_desc
```

### 2.5 Cases — authenticated

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `POST`   | `/api/v1/cases` | ✏️ | Create draft case. |
| `GET`    | `/api/v1/cases/mine` | ✏️ | List my submissions (any status). |
| `PATCH`  | `/api/v1/cases/{id}` | ✏️ (own) / 🛠 (any) | Update case fields. |
| `DELETE` | `/api/v1/cases/{id}` | 🛠 | Hard delete (audit logged). |
| `POST`   | `/api/v1/cases/{id}/submit` | ✏️ (own) | draft → pending_review. |
| `POST`   | `/api/v1/cases/{id}/assign` | 🛠 | Assign verifier. |
| `POST`   | `/api/v1/cases/{id}/verify` | 🔍 (assigned) | verified / rejected with reason. |
| `POST`   | `/api/v1/cases/{id}/publish` | 🛠 | approved → published. |
| `POST`   | `/api/v1/cases/{id}/unpublish` | 🛠 | published → archived. |

#### `POST /cases`
```json
// req
{
  "title_bn": "...", "title_en": null,
  "summary_bn": "...", "description_bn": "...",
  "incident_date": "2026-04-19",
  "country_id": 18, "division_id": 1, "district_id": 1, "upazila_id": 1,
  "location_text": null,
  "crime_type_id": 5,
  "tags": ["protest", "rapeculture"],
  "person_links": [
    { "person_id": "<uuid>", "role": "victim" },
    { "person_id": "<uuid>", "role": "accused" }
  ]
}

// 201
{ "id": "<uuid>", "case_number": "TIP-2026-00045", "status": "draft", ... }
```

### 2.6 Persons

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`    | `/api/v1/persons` | 🌐 | Public list (published persons). |
| `GET`    | `/api/v1/persons/{slug-or-id}` | 🌐 | Profile. |
| `GET`    | `/api/v1/persons/{id}/cases` | 🌐 | Cases the person is linked to. |
| `POST`   | `/api/v1/persons` | ✏️ | Create person (status `pending_review`). |
| `PATCH`  | `/api/v1/persons/{id}` | ✏️ (own) / 🛠 (any) | Update. |
| `POST`   | `/api/v1/persons/{id}/approve` | 🛠 | Move pending → approved → published. |
| `POST`   | `/api/v1/persons/{id}/merge` | 🛠 | Merge with another person id. |
| `POST`   | `/api/v1/cases/{id}/persons` | ✏️ / 🛠 | Link a person to a case with a role. |
| `DELETE` | `/api/v1/cases/{caseId}/persons/{personId}/{role}` | ✏️ / 🛠 | Unlink. |

### 2.7 Files / attachments

The API never proxies large file bytes for upload. Flow:

1. Client calls `POST /cases/{id}/attachments/presign` with the planned
   `kind`, MIME type, byte size, original filename.
2. Server resolves the next `sequence_no`, computes the canonical
   `stored_filename`, and returns a presigned PUT URL plus the canonical
   key. **Nothing is recorded in DB yet.**
3. Client PUTs the bytes directly to R2 / MinIO using that URL.
4. Client calls `POST /cases/{id}/attachments/finalize` with the
   `presign_token` returned in step 2 and the actual `etag`/checksum.
5. Server verifies the object exists in R2, records the
   `case_attachments` row, kicks off async virus scan + image
   transcoding, and returns the attachment record.

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `POST`   | `/api/v1/cases/{id}/attachments/presign` | ✏️ / 🛠 | Get a presigned PUT URL + canonical key. |
| `POST`   | `/api/v1/cases/{id}/attachments/finalize` | ✏️ / 🛠 | Confirm upload, persist row. |
| `GET`    | `/api/v1/cases/{id}/attachments/admin` | 🛠 | List all (incl. hidden / internal). |
| `POST`   | `/api/v1/attachments/{id}/download-url` | 🛠 | Issue 5-min presigned GET for hidden / internal kinds. |
| `PATCH`  | `/api/v1/attachments/{id}` | 🛠 | Update caption, kind. |
| `DELETE` | `/api/v1/attachments/{id}` | 🛠 | Delete attachment + R2 object. |

### 2.8 Locations

All location endpoints are 🌐 public, cacheable for hours.

| Method | Path | Description |
| -- | -- | -- |
| `GET` | `/api/v1/locations/countries` | All countries. |
| `GET` | `/api/v1/locations/divisions?country_id=` | Divisions of a country (BD has 8). |
| `GET` | `/api/v1/locations/districts?division_id=` | Districts of a division. |
| `GET` | `/api/v1/locations/upazilas?district_id=` | Upazilas of a district. |

### 2.9 Crime types

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`    | `/api/v1/crime-types` | 🌐 | All active categories. |
| `POST`   | `/api/v1/admin/crime-types` | 🛠 | Create. |
| `PATCH`  | `/api/v1/admin/crime-types/{id}` | 🛠 | Update. |
| `DELETE` | `/api/v1/admin/crime-types/{id}` | 🛠 | Soft-delete (sets `is_active=false`). |

### 2.10 Verification

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`  | `/api/v1/verification/queue` | 🔍 | My assignments. |
| `POST` | `/api/v1/verification/{id}/start` | 🔍 (assigned) | unassigned → in_progress. |
| `POST` | `/api/v1/verification/{id}/decide` | 🔍 (assigned) | verified / rejected with reason. |
| `POST` | `/api/v1/verification/{id}/notes` | 🔍 (assigned) / 🛠 | Append internal note. |

### 2.11 Audit log

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`  | `/api/v1/admin/audit` | 🛠 | Filter by user / action / target / date range. Cursor paginated. |
| `GET`  | `/api/v1/admin/audit.csv` | 🛠 | Streamed CSV export. |

### 2.12 Search

| Method | Path | Auth | Description |
| -- | -- | :-: | -- |
| `GET`  | `/api/v1/search?q=...&type=case|person&filters=...` | 🌐 | Cross-resource search. |

### 2.13 RSS / sitemap

| Method | Path | Description |
| -- | -- | -- |
| `GET`  | `/feed.xml` | Recent published cases (RSS 2.0). |
| `GET`  | `/sitemap.xml` | Auto-generated sitemap. |

---

## 3. Resource shapes

### 3.1 `User`

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "full_name": "...",
  "display_name": "...",
  "role": "contributor",
  "status": "approved",
  "phone": null,
  "avatar_url": null,
  "bio": null,
  "created_at": "2026-05-23T10:11:00Z",
  "approved_at": "2026-05-23T10:30:00Z",
  "last_login_at": "2026-05-23T11:00:00Z"
}
```

### 3.2 `Case` (public detail)

```json
{
  "id": "uuid",
  "case_number": "TIP-2026-00045",
  "slug": "rape-incident-savar-2026-04",
  "title": { "bn": "...", "en": null },
  "summary": { "bn": "...", "en": null },
  "description": { "bn": "...", "en": null },
  "incident_date": "2026-04-19",
  "incident_time": "21:30:00",
  "location": {
    "country": { "id": 18, "iso2": "BD", "name": { "bn": "বাংলাদেশ", "en": "Bangladesh" } },
    "division": { "id": 1, "name": { "bn": "ঢাকা", "en": "Dhaka" } },
    "district": { "id": 12, "name": { "bn": "...", "en": "..." } },
    "upazila":  { "id": 88, "name": { "bn": "...", "en": "..." } },
    "text": null
  },
  "crime_type": { "id": 5, "slug": "rape", "name": { "bn": "ধর্ষণ", "en": "Rape" } },
  "tags": ["..."],
  "case_status": "under_investigation",
  "severity": 4,
  "cover_image_url": "https://files.tansiq.org/.../TIP-2026-00045_evidence_01.jpg",
  "persons": [
    { "id": "uuid", "role": "victim",   "is_anonymous": true, "name": null, "photo_url": null },
    { "id": "uuid", "role": "accused",  "is_anonymous": false, "name": { "bn": "...", "en": "..." }, "photo_url": "..." }
  ],
  "attachments": [
    {
      "id": "uuid",
      "kind": "public",
      "sequence_no": 1,
      "stored_filename": "TIP-2026-00045_evidence_01.jpg",
      "mime_type": "image/jpeg",
      "size_bytes": 123456,
      "public_url": "https://files.tansiq.org/.../TIP-2026-00045_evidence_01.jpg",
      "caption": { "bn": "...", "en": null }
    }
  ],
  "timeline": [
    {
      "event_date": "2026-04-19",
      "event_time": "21:30:00",
      "title": { "bn": "...", "en": null },
      "description": { "bn": "...", "en": null },
      "source_url": null
    }
  ],
  "news_sources": [
    { "url": "...", "title": "...", "source_name": "...", "published_at": "...", "archived_url": "..." }
  ],
  "stats": { "view_count": 4231, "download_count": 87 },
  "published_at": "2026-04-25T08:00:00Z"
}
```

### 3.3 `Person` (public profile)

```json
{
  "id": "uuid",
  "slug": "...",
  "primary_type": "accused",
  "name": { "bn": "...", "en": "..." },
  "aliases": ["..."],
  "gender": "male",
  "date_of_birth": "1980-01-15",
  "photo_url": "...",
  "occupation": "...",
  "organization": "...",
  "designation": "...",
  "location": { /* same shape as case.location */ },
  "public_bio": { "bn": "...", "en": null },
  "case_count": 3,
  "cases": [ /* short case summaries */ ]
}
```

### 3.4 `Attachment (admin view)`

Same as public, plus:

```json
{
  "kind": "hidden",            // public | hidden | internal
  "uploaded_by": { "id": "...", "display_name": "..." },
  "checksum_sha256": "...",
  "virus_scan_status": "clean" // pending | clean | infected
}
```

---

## 4. Webhooks (post-v1, listed here for foresight)

- `case.published`
- `case.unpublished`
- `attachment.virus_scan_failed`

---

## 5. Caching guidance

| Endpoint | Cache-Control |
| -- | -- |
| `GET /cases` | `public, max-age=60, stale-while-revalidate=300` |
| `GET /cases/{slug}` | `public, max-age=300, stale-while-revalidate=3600` |
| `GET /persons/{slug}` | same |
| `GET /locations/*` | `public, max-age=86400, immutable` |
| `GET /crime-types` | `public, max-age=3600` |
| `GET /admin/*` | `private, no-store` |
| `GET /auth/me` | `private, no-store` |

The CDN may serve stale content while the API revalidates. Mutations on
admin endpoints are followed by an explicit purge call to the CDN for
affected paths.
