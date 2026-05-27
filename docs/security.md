# Security Model

## Auth

User accounts use normalized usernames/emails and bcrypt password hashes. Login errors are generic to reduce account enumeration. Registration duplicate errors are intentionally generic across username/email.

## Sessions

Admin and user sessions are stored in SQLite in the `sessions` table. Raw session tokens are only sent to the browser in HttpOnly cookies; the database stores SHA-256 token hashes, expiry timestamps, and revocation timestamps.

Session cookies use `HttpOnly` and `SameSite=Lax`. Set `COOKIE_SECURE=true` in production so cookies are sent only over HTTPS.

## CSRF

The server sets a readable `csrf_token` cookie. Unsafe methods (`POST`, `PUT`, `PATCH`, `DELETE`, and similar) must send the same value in `X-CSRF-Token`. The frontend API wrapper handles this automatically.

## Rate Limiting

Admin and user login attempts are rate-limited by client IP. By default, the app uses `RemoteAddr`. `X-Forwarded-For` is trusted only when the direct peer IP is inside `TRUSTED_PROXY_CIDRS`.

## Security Headers

Responses include:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy`
- `Permissions-Policy`
- `X-Request-ID`

## Admin Limitations

`ADMIN_PASSWORD` is intentionally simple for portfolio/demo deployment. Production mode refuses the default admin password and requires secure cookies. For serious multi-admin use, replace this with per-admin accounts and audit logging.

## Production Checklist

- Set a strong `ADMIN_PASSWORD`.
- Set `PRODUCTION=true` or `APP_ENV=production`.
- Set `COOKIE_SECURE=true` behind HTTPS.
- Configure `TRUSTED_PROXY_CIDRS` only for proxies you control.
- Persist `DATABASE_PATH` and `DATA_FILE` on durable storage.
- Back up SQLite and JSON storage.
- Run migrations before serving traffic.
- Do not run `make seed` in production.
- Rotate secrets if `.env`, cookies, or database files are exposed.
