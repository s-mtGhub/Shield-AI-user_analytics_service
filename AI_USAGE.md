# AI Usage

This project's boilerplate was generated with AI assistance. This file documents what was
AI-generated, what was human-decided, and how to trace the work back to the originating session.

## Session

- Tool: Claude Code, running Claude Sonnet 5 (`claude-sonnet-5`)
- Session / thread ID: `7e5157d7-df76-49ec-b2eb-9a199a6d11ed`
- Dates: 2026-09-01 – 2026-09-02

## Human-authored, before AI involvement

- The overall module layout (`cmd/server`, `internal/{api,config,domain,repository,service}`)
  and the `user_activity_log` entity design (columns, uniqueness on `(user_id, timestamp)`,
  intended indexing) were designed by the developer first. The AI was handed an already-scaffolded,
  empty package skeleton and asked to fill it in.
- `Dockerfile` and `.dockerignore` were added directly by the developer, outside the AI session.

## Decisions confirmed with the developer before code was written

The AI proposed a design and paused for explicit sign-off on the choices that shape the codebase,
rather than assuming defaults:

- HTTP routing: stdlib `net/http` (Go 1.22+ method+path `ServeMux` patterns) over chi/gin.
- Postgres access: `jackc/pgx` (via `pgxpool`) + `golang-migrate` running migrations on startup,
  over `database/sql`+`lib/pq` or manually-applied SQL.
- API shape: versioned REST paths (`/api/v1/events/login`,
  `/api/v1/analytics/daily-active-users`, `/api/v1/analytics/monthly-active-users`).
- Default service timezone: UTC, overridable via `SERVICE_TIMEZONE` — day/month boundaries are
  resolved in that timezone and converted to UTC instants for the `timestamptz` range query.

## What the AI generated

- `internal/domain/event.go`, `internal/domain/timeutil.go` — `LoginEvent`, and
  `DayBounds`/`MonthBounds` helpers that turn a calendar day or month, interpreted in a given
  `time.Location`, into a `[start, end)` UTC instant range.
- `internal/repository/interface.go`, `internal/repository/postgres.go` — a `Repository`
  interface with a single `CountDistinctUsers(start, end)` method backing both the daily and
  monthly endpoints, and a pgx-backed implementation (`INSERT ... ON CONFLICT DO NOTHING` for
  idempotent writes).
- `internal/service/ingestion.go`, `internal/service/query.go` — validation and orchestration,
  with an `ErrInvalidInput` sentinel so the HTTP layer can distinguish 400s from 500s.
- `internal/api/dto.go`, `internal/api/handlers.go` — the three HTTP handlers and their
  request/response payloads.
- `cmd/server/main.go` — wiring: load env/`.env` → run migrations → connect `pgxpool` → build
  repository/services/handlers → start the HTTP server. Also fixed a pre-existing bug in the
  skeleton: `main.go` declared `package server`, which isn't buildable as an entrypoint; it now
  declares `package main`.
- `internal/config/config.go` — env-var-driven configuration (`DATABASE_URL`, `PORT`,
  `SERVICE_TIMEZONE`, `MIGRATIONS_PATH`).
- `migrations/0001_init.up.sql` / `.down.sql` — table plus a `(timestamp, user_id)` index sized
  for the count queries specifically (they filter by timestamp range only, never by `user_id`);
  the `UNIQUE(user_id, timestamp)` constraint already provides the `(user_id, timestamp)` index
  for write-side dedup.
- Tests for every layer: table-driven unit tests for `ingestion`/`query` against a fake in-memory
  repository, `httptest`-based handler tests, and a Postgres integration test
  (`internal/repository/postgres_test.go`) that skips cleanly unless `TEST_DATABASE_URL` is set.
- A `constants.go` per package (`api`, `config`, `domain`, `repository`, `cmd/server`), added on
  request, replacing inline magic strings/numbers — route paths, query param names, error
  messages, env var names/defaults, date layout strings, the table name, and HTTP timeouts.
- `.env` / `.env.example` support via `godotenv`, added on request — `main.go` loads `.env`
  optionally (a missing file is not an error) before reading configuration, so unset variables
  still fall through to the real environment in non-local deployments.

Every generated change was verified with `gofmt`, `go build ./...`, `go vet ./...`, and
`go test ./...` before being presented as done.

## Not done by the AI

No README, no graceful shutdown handling, no request-timeout middleware beyond the server's
`ReadTimeout`/`WriteTimeout`, and no containerization (the `Dockerfile`/`.dockerignore` are
human-authored, listed above).

## A note on `.env`

`.env` is gitignored and dockerignored, but it currently holds live database credentials in
plaintext. Treat it as a secret: don't commit it, don't paste its contents into shared
channels/tickets/AI sessions, and rotate the credential if it's ever exposed.
