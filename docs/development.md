# Development

## First Run

Prerequisites: Go 1.25+, Node.js 20+ for JS/E2E checks, and optional Docker.

```bash
cp .env.example .env
make run
```

Open `http://localhost:8080`.

## Common Commands

```bash
make fmt
make vet
make staticcheck
make test
make race
make coverage
make js-check
make e2e
make docker-build
make full-check
```

`make check` runs formatting, vet, staticcheck, Go tests, build, and JS checks. `make full-check` adds race tests, coverage, Playwright smoke tests, and Docker build. `make e2e` starts a local server on port `18080`.

## Migrations

SQL migrations live in `migrations/` for reviewers and are embedded from `internal/storage/sqlite/migrations/` for deployment. The runner records applied versions in `schema_migrations`.

The app runs migrations on startup. To run them explicitly:

```bash
make migrate
```

## Seed Data

For local development only:

```bash
make seed
```

This creates demo accounts with fake data. The seed command refuses to run in production mode.
