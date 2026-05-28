# Security Model

This document describes the security decisions that exist in the codebase today. It is honest about the demo scope and the production gaps.

## User Auth

- Registration normalizes usernames and emails before storage.
- User passwords are hashed with bcrypt in `internal/app/password.go`.
- Login returns generic `invalid credentials` errors to avoid confirming whether an account exists.
- Duplicate registration errors are generic across username/email.
- Authenticated user sessions are stored in SQLite through `internal/sessions/store.go`.

## Admin Auth

- Admin access uses the configured `ADMIN_PASSWORD`.
- Successful admin login creates a SQLite-backed session with kind `admin`.
- Admin and user sessions are separated by the `sessions.kind` column.
- Admin auth is deliberately simple for a portfolio project. It is not a multi-admin account system and does not provide RBAC or MFA.
- Production mode rejects `ADMIN_PASSWORD=change-me`.

## Sessions

- Raw session tokens are sent only to the browser in HttpOnly cookies.
- SQLite stores SHA-256 token hashes, not raw session tokens.
- Sessions include `created_at`, `expires_at`, and optional `revoked_at`.
- Logout revokes the matching session row and clears the browser cookie.
- User and admin cookies use `HttpOnly` and `SameSite=Lax`.
- `COOKIE_SECURE=true` is required in production mode so cookies are sent only over HTTPS.

## CSRF

- Middleware sets a readable `csrf_token` cookie.
- Unsafe methods (`POST`, `PUT`, `PATCH`, `DELETE`, etc.) must send the same value in the `X-CSRF-Token` header.
- The frontend API wrapper reads the readable CSRF cookie and sends the header automatically.
- Auth/session cookies remain HttpOnly and are not read by JavaScript.
- This is a same-origin double-submit design. For a larger production system, consider binding CSRF tokens to server-side session state.

## Rate Limiting And Client IPs

- Admin login and user login have separate in-memory rate limiters.
- Limit keys use the client IP.
- By default, the app trusts `RemoteAddr`.
- `X-Forwarded-For` is used only when the direct peer IP is inside `TRUSTED_PROXY_CIDRS`.
- In-memory rate limiting is fine for a single-instance portfolio app. A multi-instance deployment should use shared storage such as Redis.

## Admin Audit Logs

Admin actions are written to `admin_audit_logs`:

- login invalid JSON
- login failure
- login rate limited
- login success
- logout
- export results
- delete one result
- clear results
- update report status

Each row stores action, optional target type/id, client IP, user agent, and timestamp. Audit logging is best-effort: request handling does not fail only because an audit insert fails.

Production upgrades to consider:

- per-admin accounts
- immutable external audit sink
- MFA
- RBAC
- correlation with request IDs
- retention policy

## Security Headers

`internal/http/middleware/security.go` sets:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy`
- `Permissions-Policy`

`internal/http/middleware/request_id.go` sets/propagates `X-Request-ID`.

## Data Exposure Rules

- Public profile responses do not include email, password hashes, session data, raw answers, or scores JSON.
- Private profiles hide primary result, completed count, friends, bio, and comments.
- Deleted messages return an empty body in conversation reads.
- Blocked users cannot send friend requests, comments, or messages through the implemented flows.
- Admin export is admin-only and intended for anonymous quiz submissions, not user account secrets.

## SQL Injection Risk

The app uses parameterized SQL queries through `database/sql`. Dynamic SQL is limited to migration/DDL paths and controlled internal statements.

## Secrets And Local Files

- `.env` and `.env.*` are ignored by Git.
- `.env.example` contains only safe local defaults.
- SQLite DB files, JSON runtime data, coverage files, build artifacts, Playwright output, and `node_modules/` are ignored.

## Production Validation

`internal/config/config.go` enforces these checks when `PRODUCTION=true` or `APP_ENV=production`:

- `ADMIN_PASSWORD` must not be the default `change-me`.
- `COOKIE_SECURE=true` is required.

## Production Checklist

- Set a strong `ADMIN_PASSWORD`.
- Set `PRODUCTION=true` or `APP_ENV=production`.
- Set `COOKIE_SECURE=true` behind HTTPS.
- Configure `TRUSTED_PROXY_CIDRS` only for proxies you control.
- Persist and back up `DATABASE_PATH` and `DATA_FILE`.
- Run migrations before serving traffic or allow startup migration to run.
- Do not run `make seed` in production.
- Rotate secrets if `.env`, cookies, or database files are exposed.
- Replace single-password admin auth with per-admin accounts before serious production use.
