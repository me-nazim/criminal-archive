# Runbook

The book of how to **operate** Tansiq Information Portal in production.
For "how to develop" see the root `README.md`. For architecture see
`docs/ARCHITECTURE.md`.

---

## 1. Deployment topology (target)

```
                    Cloudflare (DNS + WAF + CDN)
                            │
              ┌─────────────┼──────────────┐
              ▼             ▼              ▼
         tansiq.org    api.tansiq.org   files.tansiq.org
              │             │              │
              ▼             ▼              ▼
         nginx (SPA)    Go API binary   Cloudflare R2 bucket
                            │
                            ▼
                Managed PostgreSQL 18 (with PITR)
                            │
                            ▼
                Nightly pg_dump → R2 archive bucket
```

- A single VPS (Hetzner / Hostinger / DO, 2 vCPU / 4 GB) runs the
  backend and nginx as docker-compose services.
- Postgres lives outside the VPS — Neon, Supabase, or Railway. The VPS
  itself stays disposable.
- Cloudflare R2 holds evidence files. Hidden / internal kinds never
  reach the public custom domain; admins fetch them via API-issued
  presigned URLs.
- The frontend SPA is served by nginx from `/usr/share/nginx/html` (the
  static build output). It is also fine to host the SPA from R2 with
  Cloudflare Pages — pick whichever you maintain elsewhere.

## 2. First-time provisioning

This sequence is run once when standing up a brand new environment.

1. **DNS** — `A`/`AAAA` records for `tansiq.org`, `api.tansiq.org`,
   `files.tansiq.org` pointing at the VPS (proxied through Cloudflare).
   Add a custom domain on the R2 bucket for `files.tansiq.org`.
2. **Database** — provision a managed Postgres 18 instance, enable
   point-in-time recovery and an outbound IP allow-list. Capture
   `DATABASE_URL` (use `sslmode=require`).
3. **Storage** — create the R2 bucket `tansiq-archive`. Generate an
   API token with read+write+delete scope to that bucket only. Capture
   `S3_ACCESS_KEY` / `S3_SECRET_KEY` / `S3_ENDPOINT`. The endpoint will
   be `https://<account-id>.r2.cloudflarestorage.com`.
4. **Email** — sign up for a transactional provider (Resend / Postmark)
   and capture the API key. Used for password resets.
5. **VPS** — `apt install docker.io docker-compose-plugin`. Create a
   non-root deploy user, drop your SSH public key, disable password
   auth.
6. **Secrets** — copy `.env.example` to `.env` on the VPS, edit:
   - `APP_ENV=production`
   - `APP_BASE_URL=https://api.tansiq.org`
   - `FRONTEND_BASE_URL=https://tansiq.org`
   - `DATABASE_URL=…`
   - `JWT_SECRET=` — 64 random hex chars (`openssl rand -hex 32`)
   - `COOKIE_SECURE=true`
   - `COOKIE_DOMAIN=.tansiq.org`
   - `S3_*` — R2 credentials
   - `S3_PUBLIC_BASE_URL=https://files.tansiq.org/tansiq-archive`
   - `CORS_ALLOWED_ORIGINS=https://tansiq.org`
7. **Bring it up:**
   ```bash
   docker compose -f docker-compose.prod.yml up -d
   docker compose exec backend /app/api migrate up
   docker compose exec backend /app/api seed
   docker compose exec backend /app/api admin bootstrap \
     --email super@tansiq.org \
     --name  "First admin" \
     --password "$(openssl rand -base64 24)" \
     --role  super_admin
   ```
   Save the generated password into your password manager. The first
   admin account exists only to create everyone else from `/admin/users`.

## 3. Day-to-day operations

### Deploy a new release
```bash
# On the VPS
git -C /opt/tansiq pull
docker compose -f docker-compose.prod.yml up -d --build
docker compose exec backend /app/api migrate up   # idempotent
```
Traffic is unaffected because the API process is restarted while nginx
keeps serving the SPA. With multiple API replicas behind nginx the
deploy can be made zero-downtime via rolling restart.

### Approve a registration
1. New user signs up on the public site.
2. Admin opens `/admin/approvals`, clicks **Approve**.
3. The user is notified by email (TODO: email is currently logged-only;
   wire to the configured provider in Phase 10).

### Investigate a request
Every API request is logged as a single JSON line with `request_id`,
`user_id`, `path`, `status`, `duration_ms`. Cross-reference with the
audit log at `/admin/audit-log` to see which privileged action happened
in which request.

### Emergency: revoke all sessions
```sql
UPDATE sessions SET revoked_at = now() WHERE revoked_at IS NULL;
```
Every user is forced to re-login at the next request. Used when a
secret leaks.

## 4. Backups

- **Postgres** — managed provider keeps PITR for 7 days. Additionally,
  a nightly cron dumps with `pg_dump --format=custom` and uploads to a
  separate R2 bucket (`tansiq-backups`). Retention: 30 daily + 12
  monthly snapshots.
- **Object storage** — R2 lifecycle: hot for 90 days then infrequent
  access. Cross-region replication on `tansiq-archive` for the
  public-attachments bucket.
- **Configuration** — repo-tracked (`.env.example`). The actual
  `.env` is mirrored to a password manager whenever it changes.

### Restore drill (run quarterly)
1. Provision a fresh Postgres instance.
2. `pg_restore --create --dbname=postgres /path/to/dump.pgcustom`.
3. Re-point the deployment's `DATABASE_URL` to the restored instance
   (or the new one for the drill).
4. Verify: `/health` returns `db: up`; `/api/v1/cases` returns the
   expected `published_at` count.

## 5. Observability

- **Logs** — `journalctl -u docker-compose@tansiq -f` aggregates the
  Docker stdout JSON. In the absence of a log shipper, weekly
  `journalctl --since "7 days ago" > weekly.log` is fine for a
  one-person operation.
- **Metrics** — `GET /metrics` returns Prometheus exposition format:
  - `http_requests_total{method,route,status}`
  - `http_request_duration_seconds_bucket{method,route}`
  - Standard Go runtime + process collectors.
  Scraping target: bind to a private port behind the VPS firewall and
  scrape from a hosted Grafana Cloud (or run a sidecar Prometheus).
- **Alerts** — at minimum:
  - 5xx rate > 1% over 5 minutes
  - p95 read latency > 500 ms over 10 minutes
  - DB connection error count > 0
  - Disk usage > 85%

## 6. Incident response cheat sheet

| Symptom | First check | Fix |
| -- | -- | -- |
| 5xx spike | `/metrics`, `journalctl` last 200 lines | Roll back the last image. |
| DB unreachable | Managed provider status page | Switch to read-replica DSN; ride out. |
| R2 returns 5xx | Cloudflare status | Disable uploads via feature flag; reads continue from cache. |
| Token leak | `UPDATE sessions SET revoked_at = now()` | Rotate `JWT_SECRET`, redeploy, invalidate all access tokens. |
| Spam registrations | Inspect `audit_logs WHERE action='user.register'` | Tighten the rate limiter (env), enable hCaptcha (Phase 11). |

## 7. Decommissioning a contributor

When a contributor turns out to be acting in bad faith:
1. From `/admin/users`, **suspend** the account → all sessions revoked.
2. Audit-log search for everything they touched: `target_user_id =`.
3. For any of their cases that are `published`, decide whether to
   `unpublish` → `archived` and add an admin internal note explaining
   the reason.
4. The audit log entry is permanent — no need to "clean it up".

## 8. Open questions tracked here

- Email delivery is currently logged-only. Phase 10 should wire the
  `EMAIL_PROVIDER_API_KEY` env var to actually send password reset
  links and approval notifications.
- ClamAV virus scanning of uploads is not enabled in v1. We rely on
  R2 + Cloudflare bot mitigation; revisit if abuse appears.
- Bulk download (zip of a case's public attachments) is mentioned in
  the API spec but not yet implemented.
