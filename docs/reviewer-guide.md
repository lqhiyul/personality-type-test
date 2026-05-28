# Reviewer Guide

This guide is for a backend reviewer who wants to assess the project quickly after `git clone`.

## First Files To Read

- `cmd/server/main.go`: server entrypoint.
- `internal/app/server.go`: app construction, DB/session/store wiring, HTTP server timeouts.
- `internal/app/handlers.go`: route groups.
- `internal/app/user_auth_handlers.go`: user registration/login/logout/current user.
- `internal/app/admin_handlers.go`: admin login, anonymous result admin API, exports.
- `internal/app/admin_audit.go`: admin audit writes.
- `internal/sessions/store.go`: hashed SQLite sessions.
- `internal/storage/sqlite/sqlite.go`: SQLite opening, PRAGMAs, migration runner.
- `migrations/`: reviewable SQL schema.
- `docs/security.md`: security model and limitations.
- `docs/api.md`: endpoint contract.

## Run Locally

```bash
cp .env.example .env
go run ./cmd/server
```

Open `http://localhost:8080`.

Admin panel:

```text
http://localhost:8080/?admin=1
```

Use `ADMIN_PASSWORD` from `.env`.

## Check Auth Quickly

1. Register a user from the UI or `POST /api/auth/register`.
2. Confirm `/api/auth/me` returns the user with the session cookie.
3. Submit the quiz while logged in.
4. Confirm `/api/me/results` contains the saved result.
5. Logout and confirm `/api/auth/me` returns `401`.

## Check Admin Quickly

1. Submit one anonymous quiz result.
2. Open `/?admin=1`.
3. Login with `ADMIN_PASSWORD`.
4. Check result list, stats, CSV/JSON export.
5. Delete one result or clear results.
6. Inspect `admin_audit_logs` in SQLite to confirm admin actions are recorded.

## Quality Gate

```bash
go fmt ./...
go mod tidy
go test ./...
go vet ./...
staticcheck ./...
npm run js-check
```

Full local gate when Docker and Playwright browsers are available:

```bash
go test -race ./...
npm run e2e
docker build -t personality-type-test .
```

With `make`:

```bash
make check
make full-check
```

## Main Trade-Offs To Notice

- SQLite is intentional for easy clone-and-run review, not high-write multi-replica production.
- Admin auth is a demo model: one configured password, admin sessions, and audit logs. Production should use per-admin accounts, RBAC, and MFA.
- `internal/app` remains broad, but route registration and handlers are grouped by domain to keep review clear without a risky package split.
- The frontend is vanilla JS to keep the dependency surface small and the backend as the main review target.
