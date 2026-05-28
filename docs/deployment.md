# Deployment

## Docker

```bash
docker build -t personality-type-test .
docker run --rm -p 8080:8080 \
  -e HOST=0.0.0.0 \
  -e ADMIN_PASSWORD="replace-with-a-strong-password" \
  personality-type-test
```

Mount persistent storage for `/app/data` in real deployments.

## Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

The compose file maps `8080:8080` and stores `/app/data` in a named volume.

## Environment

- `HOST`, `PORT`, or `ADDR`: bind address.
- `ADMIN_PASSWORD`: admin panel password.
- `DATA_FILE`: JSON file for anonymous submissions.
- `DATABASE_PATH`: SQLite database path.
- `COOKIE_SECURE`: set `true` behind HTTPS.
- `PRODUCTION` or `APP_ENV=production`: enables production validation.
- `TRUSTED_PROXY_CIDRS`: comma-separated proxy CIDRs allowed to provide `X-Forwarded-For`.

## Production Checklist

- Set a strong `ADMIN_PASSWORD`.
- Set `PRODUCTION=true` or `APP_ENV=production`.
- Set `COOKIE_SECURE=true`.
- Run migrations with `make migrate` or allow startup migration.
- Use a persistent disk for SQLite and JSON storage.
- Back up `DATABASE_PATH` and `DATA_FILE`.
- Do not use local seed data.
- Configure trusted proxy CIDRs only for infrastructure you control.
- Review logs for request IDs, status codes, and unexpected auth failures.
- Review `admin_audit_logs` for admin login/export/delete/report-review actions.
- Rotate secrets if anything sensitive is committed or exposed.

## Known Deployment Limitation

SQLite is appropriate for a small portfolio/demo app. For high write volume, multiple app replicas, or multi-region deployment, move authenticated storage to a managed relational database.
