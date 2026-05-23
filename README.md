# Tansiq Information Portal (criminal-archive)

A community-driven, public archive that documents crimes — particularly sensitive incidents such as sexual violence — with verifiable evidence, victim/accused profiles, and case timelines. Inspired by the Internet Archive: anyone can browse and download published material; submissions go through a verification + admin review workflow before publication.

> ⚠️ This is a **work-in-progress scaffold**. APIs, schema and UI are still being built. See `docs/ARCHITECTURE.md` for the design.

## Tech stack

| Layer       | Choice                                  |
| ----------- | --------------------------------------- |
| Backend     | Go 1.25 (`net/http` + chi router, pgx)  |
| Database    | PostgreSQL 18                           |
| Storage     | Cloudflare R2 (S3-compatible)           |
| Frontend    | React 18 + Vite + TypeScript + Tailwind |
| i18n        | Bengali (primary) + English             |
| Deployment  | Docker / docker-compose                 |

## Quick start (local development)

```bash
# 1. Copy env file and edit secrets
cp .env.example .env

# 2. Start everything (postgres + minio + backend + frontend)
docker compose up --build

# 3. Open
#   Frontend  → http://localhost:5173
#   API       → http://localhost:8080/health
#   MinIO UI  → http://localhost:9001  (S3-compatible R2 substitute)
```

### Run services individually

```bash
# Backend
cd backend
go run ./cmd/api

# Frontend
cd frontend
npm install
npm run dev
```

## Repository layout

```
backend/    Go API, migrations, seed data
frontend/   React + Vite + Tailwind SPA
docs/       Architecture & design notes
```

## Documentation

The planning suite lives in [`docs/`](./docs/). Read in this order:

1. [`docs/REQUIREMENTS.md`](./docs/REQUIREMENTS.md) — vision, personas, user stories, functional + non-functional requirements.
2. [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) — system architecture, data model, sequence diagrams, deployment topology.
3. [`docs/API_SPEC.md`](./docs/API_SPEC.md) — REST API contract for `v1`.
4. [`docs/UI_DESIGN.md`](./docs/UI_DESIGN.md) — design language, component library, page-by-page specifications.
5. [`docs/ROADMAP.md`](./docs/ROADMAP.md) — phased implementation plan with PR breakdown.

## Roles

`super_admin` · `admin` · `moderator` · `contributor` · `viewer` (+ anonymous public).

## License

TBD.
