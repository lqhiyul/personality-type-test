# Changelog

## Unreleased

- Refactored the backend into clearer internal packages and focused app files.
- Added versioned SQLite migrations with a `schema_migrations` table.
- Added SQLite-backed hashed sessions for admin and user authentication.
- Added CSRF protection, security headers, request IDs, and structured request logging.
- Added Playwright smoke tests, stronger CI, Dependabot, and CodeQL.
- Updated developer, deployment, security, and architecture documentation.
