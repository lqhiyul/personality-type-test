# Changelog

## Unreleased

- Tightened CI and Makefile quality gates for a v0.1.0-style portfolio release.
- Added API contract and migration constraint coverage for reviewer-critical behavior.
- Refactored the backend into clearer internal packages and focused app files.
- Added versioned SQLite migrations with a `schema_migrations` table.
- Added SQLite-backed hashed sessions for admin and user authentication.
- Added CSRF protection, security headers, request IDs, and structured request logging.
- Added Playwright smoke tests, stronger CI, Dependabot, and CodeQL.
- Updated developer, deployment, security, and architecture documentation.
