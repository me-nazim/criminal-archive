# Migrations

SQL migrations are plain `*.up.sql` / `*.down.sql` files, applied in lexicographic order.

## Apply locally

```bash
# from repo root, with docker compose running
docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  < backend/migrations/0001_init.up.sql
```

A Go-based migrator (using `golang-migrate` or a small in-house runner) will be
wired into the API process in a follow-up PR.

## Adding a new migration

1. Create a pair `NNNN_short_name.up.sql` / `NNNN_short_name.down.sql`.
2. Keep each migration **transactional** (`BEGIN; ... COMMIT;`).
3. Never edit a migration after it has been applied to a shared environment;
   create a new one instead.
