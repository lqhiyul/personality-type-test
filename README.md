# Personality Type Test

[![CI](https://github.com/lqhiyul/personality-type-test/actions/workflows/go.yml/badge.svg)](https://github.com/lqhiyul/personality-type-test/actions/workflows/go.yml)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![Vanilla JS](https://img.shields.io/badge/Frontend-Vanilla%20JS-F7DF1E?logo=javascript&logoColor=111)

A lightweight full-stack personality type test with a Go HTTP backend, modular vanilla JavaScript frontend, local JSON result storage, and an admin/export panel.

**Live demo:** not verified yet. A deployment link should be added only after the hosted app is checked manually.

**Quick links:** [Preview](#preview) · [Features](#features) · [Quick Start](#quick-start) · [Docker](#docker) · [Quality Checks](#quality-checks) · [API](#api-overview)

## Preview

The repository currently includes a lightweight preview asset, but no real browser screenshots yet.

![Project preview](assets/preview.svg)

Real screenshots should be added manually after running the app locally:

```text
assets/screenshots/home.png
assets/screenshots/quiz.png
assets/screenshots/result.png
assets/screenshots/types.png
assets/screenshots/admin.png
```

Manual screenshot workflow:

1. Run the app locally with the commands in [Quick Start](#quick-start).
2. Open `http://localhost:8080` in a browser.
3. Capture the home, quiz, result, types, and admin screens.
4. Save the files into `assets/screenshots/`.
5. Commit the images.
6. Replace this note with a screenshot gallery once the files exist.

## Features

- 28-question MBTI-style quiz with progress tracking and draft saving in `localStorage`.
- Result page with type breakdown, confidence notes, similar types, copy/share actions, and Telegram CTA.
- Searchable catalog of all 16 personality types with localized profile content.
- Type compatibility comparison for friendship, relationships, and work.
- Hidden admin panel with login, logout, search, delete, clear, CSV export, JSON export, and demo/autopass mode.
- In-memory login rate limiting for repeated failed admin login attempts by IP address.
- Local JSON persistence with safer temp-file writes and rename.
- Embedded static assets, Docker support, GitHub Actions, Go tests, and JavaScript syntax checks.

## Tech Stack

- **Backend:** Go 1.22, standard `net/http`, embedded static files.
- **Frontend:** HTML, CSS, modular vanilla JavaScript.
- **Storage:** local JSON file at `data/results.json` by default.
- **Tooling:** Docker, Makefile, GitHub Actions, Node-based JavaScript syntax check.

## What This Project Demonstrates

- Building a small full-stack web app without a heavy frontend framework.
- Designing REST-style JSON endpoints with validation and clear error responses.
- Managing frontend state with simple JavaScript modules.
- Persisting data locally while keeping the storage layer easy to review.
- Implementing basic admin sessions, result exports, and failed-login rate limiting.
- Keeping the app portable with Docker and environment-based configuration.
- Maintaining confidence with `go vet`, Go tests, race-test CI, build checks, and frontend syntax checks.

## Quick Start

Set the environment variables directly, or copy values from [.env.example](.env.example) into your local environment manager.

PowerShell:

```powershell
$env:HOST = "127.0.0.1"
$env:PORT = "8080"
$env:ADMIN_PASSWORD = "change-me"
go run .
```

macOS/Linux shell:

```bash
HOST=127.0.0.1 PORT=8080 ADMIN_PASSWORD=change-me go run .
```

Open `http://localhost:8080`.

Admin tools are intentionally subtle in the public UI. Locally, click the small `QA` control or open `http://localhost:8080/?admin=1`, then log in with `ADMIN_PASSWORD`.

For phone testing on the same Wi-Fi network, bind to all interfaces:

```powershell
$env:HOST = "0.0.0.0"
$env:PORT = "8080"
$env:COOKIE_SECURE = "false"
go run .
```

Then open `http://LOCAL_PC_IP:8080` from the phone. See [docs/local-network-testing.md](docs/local-network-testing.md) for the checklist.

## Docker

```bash
docker build -t personality-type-test .
docker run --rm -p 8080:8080 -e ADMIN_PASSWORD="strong-password" personality-type-test
```

The image runs as a non-root user, exposes port `8080`, and includes a `/healthz` healthcheck.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `HOST` | empty | Optional bind host. Use `127.0.0.1` locally or `0.0.0.0` for LAN/container runs. |
| `PORT` | `8080` | HTTP port. Both `8080` and `:8080` are accepted. |
| `ADDR` | empty | Exact bind address override, for example `127.0.0.1:8080`. Wins over `HOST` and `PORT`. |
| `ADMIN_PASSWORD` | `change-me` | Password for the admin panel. Change it before public deploys. |
| `DATA_FILE` | `data/results.json` | Path for saved quiz submissions. |
| `COOKIE_SECURE` | `false` | Keep `false` locally. Set to `true` only behind HTTPS. |

Runtime data is ignored by Git. The `data/results.json` file is created automatically after the first saved result.

## Quality Checks

Direct commands:

```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
node scripts/js-check.mjs
```

Optional Makefile targets, when `make` is available:

```bash
make fmt
make vet
make test
make race
make build
make js-check
make check
```

GitHub Actions runs Go formatting, JavaScript syntax checks, `go vet`, `go test`, `go test -race`, and `go build`.

## Project Structure

```text
.
+-- .github/workflows/go.yml
+-- assets/
|   +-- preview.svg
|   +-- screenshots/
+-- docs/local-network-testing.md
+-- scripts/js-check.mjs
+-- web/static/
|   +-- index.html
|   +-- style.css
|   +-- js/
|   |   +-- api.js
|   |   +-- app.js
|   |   +-- admin.js
|   |   +-- compatibility.js
|   |   +-- dom.js
|   |   +-- events.js
|   |   +-- i18n.js
|   |   +-- quiz.js
|   |   +-- results.js
|   |   +-- share.js
|   |   +-- state.js
|   |   +-- types.js
|   |   +-- ui.js
|   |   +-- utils.js
|   +-- assets/share-cards/
|   +-- compatibility-engine.js
|   +-- content-*.js
|   +-- result-insights.js
|   +-- types-data.js
+-- config.go
+-- handlers.go
+-- login_rate_limiter.go
+-- main.go
+-- scoring.go
+-- sessions.go
+-- store.go
+-- *_test.go
+-- Dockerfile
+-- Makefile
```

## API Overview

| Method | Route | Description |
| --- | --- | --- |
| `GET` | `/` | Main page. |
| `GET` | `/healthz` | Healthcheck endpoint. |
| `POST` | `/api/submit` | Save a completed test and return the result profile. |
| `POST` | `/api/login` | Admin login with failed-attempt rate limiting. |
| `POST` | `/api/logout` | Admin logout and session cookie cleanup. |
| `GET` | `/api/results` | List saved results. |
| `GET` | `/api/results/export` | Export saved results as CSV. |
| `GET` | `/api/results/export?format=json` | Export saved results as JSON. |
| `DELETE` | `/api/results` | Delete all results. |
| `DELETE` | `/api/results/{id}` | Delete one result. |
| `GET` | `/api/stats` | Return saved result statistics. |

## Security Notes

- Admin access uses a single password configured through `ADMIN_PASSWORD`.
- Failed admin login attempts are rate-limited in memory per IP address.
- Session cookies are `HttpOnly` and `SameSite=Lax`.
- Set `COOKIE_SECURE=true` only when the app is served behind HTTPS.
- The app is not intended to store sensitive personal data.

## Limitations and Trade-offs

- This is an educational/self-reflection tool, not a medical, psychological, or scientific diagnosis.
- JSON file storage is simple and reviewable, but it is not ideal for multi-instance deployments.
- Sessions and login rate limits are in memory, so they reset when the process restarts.
- No verified live demo is currently linked.
- Real screenshots still need to be captured manually before the README has a full visual gallery.

## Roadmap

- Add verified live deployment link after manual testing.
- Add real screenshots or a short GIF from the running app.
- Add a lightweight browser smoke test when it is worth the extra tooling.
- Add pagination for admin results if the JSON file grows.
- Move to a database only if the project becomes a public multi-user app.

## Author

Built as a junior full-stack portfolio project by [lqhiyul](https://github.com/lqhiyul).
