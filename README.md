# user-analytics-service

A small Go + PostgreSQL service that ingests user login events and answers daily/monthly
unique active user queries.

## Live deployment

Deployed on [Render](https://render.com):

```
https://shield-ai-user-analytics-service.onrender.com
```

Render's free tier spins the instance down when idle — the first request after a period of
inactivity can take 30–60s to respond while it cold-starts.

## Tech stack

- Go, stdlib `net/http` (no router framework)
- PostgreSQL via [`jackc/pgx`](https://github.com/jackc/pgx) (`pgxpool`)
- [`golang-migrate`](https://github.com/golang-migrate/migrate) for schema migrations, run on startup
- [`joho/godotenv`](https://github.com/joho/godotenv) for local `.env` loading

## Project structure

```
cmd/server/           entrypoint: config → migrations → DB pool → wiring → HTTP server
internal/domain/      LoginEvent entity + day/month → UTC boundary resolution
internal/repository/  Repository interface + pgx-backed Postgres implementation
internal/service/     ingestion (validation + persist) and query (aggregate) services
internal/api/         HTTP handlers and request/response DTOs
migrations/           SQL schema migrations
```

Each package under `internal/` (plus `cmd/server`) has its own `constants.go` for
package-local constants (routes, env var names/defaults, date layouts, table name, timeouts).

## Data model

```sql
CREATE TABLE user_activity_log (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, timestamp)
);

CREATE INDEX idx_user_activity_log_timestamp_user_id
    ON user_activity_log (timestamp, user_id);
```

Timestamps are stored as `TIMESTAMPTZ` (i.e. as UTC instants). A single storage timezone keeps
the schema simple; calendar-day/month interpretation for queries is handled in the service layer
via `SERVICE_TIMEZONE` (defaults to `UTC`) — see `internal/domain/timeutil.go`.

## Running locally

1. Copy `.env.example` to `.env` and point `DATABASE_URL` at a local/dev Postgres instance.
2. Run the service — migrations apply automatically on startup:

   ```bash
   go run ./cmd/server
   ```

3. Run the test suite:

   ```bash
   go test ./...
   ```

   The Postgres integration test (`internal/repository/postgres_test.go`) is skipped unless
   `TEST_DATABASE_URL` is set:

   ```bash
   TEST_DATABASE_URL=postgres://user:pass@localhost:5432/analytics_test?sslmode=disable \
     go test ./internal/repository/...
   ```

## API

Base URL used below is the live Render deployment; swap in `http://localhost:8080` for local
testing.

### 1. Record a login event

`POST /api/v1/events/login`

`timestamp` is optional (RFC3339) — defaults to the server's current time when omitted.

```bash
curl -X POST https://shield-ai-user-analytics-service.onrender.com/api/v1/events/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "timestamp": "2026-09-01T10:15:00Z"
  }'
```

Response — `201 Created`:

```json
{
  "user_id": "user-123",
  "timestamp": "2026-09-01T10:15:00Z"
}
```

### 2. Daily unique active users

`GET /api/v1/analytics/daily-active-users?date=YYYY-MM-DD`

```bash
curl "https://shield-ai-user-analytics-service.onrender.com/api/v1/analytics/daily-active-users?date=2026-09-01"
```

Response — `200 OK`:

```json
{
  "date": "2026-09-01",
  "unique_active_users": 1
}
```

### 3. Monthly unique active users

`GET /api/v1/analytics/monthly-active-users?month=YYYY-MM`

```bash
curl "https://shield-ai-user-analytics-service.onrender.com/api/v1/analytics/monthly-active-users?month=2026-09"
```

Response — `200 OK`:

```json
{
  "month": "2026-09",
  "unique_active_users": 1
}
```

### Errors

Non-2xx responses are JSON: `{"error": "<message>"}`. A malformed/missing `user_id`,
`timestamp`, `date`, or `month` returns `400`; a downstream failure returns `500`.

## AI usage

See [AI_USAGE.md](AI_USAGE.md) for what was AI-generated versus human-authored in this project.
