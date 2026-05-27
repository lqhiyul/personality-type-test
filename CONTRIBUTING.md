# Contributing

This is a portfolio project, so changes should stay focused, reviewable, and easy to explain.

## Local Checks

Run:

```bash
make check
npm run e2e
make docker-build
```

Use small pull requests. Include tests for backend behavior, security changes, storage changes, and user-visible workflows.

## Style

- Keep the Go backend on `net/http` and the frontend on vanilla JavaScript.
- Do not add framework rewrites for small UI changes.
- Keep handlers thin and move storage/security/domain logic into focused packages.
- Do not commit local data, database files, `.env` files, coverage HTML, Playwright traces, or generated binaries.
