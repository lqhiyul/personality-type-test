# Personality Type Test

A small Go + vanilla JavaScript web app for taking a 28-question personality test, getting an MBTI-style result, and browsing all 16 personality types with socionics codes.

The project is intentionally lightweight: Go serves the API and embedded static files, the frontend is plain HTML/CSS/JavaScript, and submissions are stored in a local JSON file. The goal is to keep the code easy to read, review, and run as a portfolio project.

![Project preview](assets/preview.svg)

## What This Project Demonstrates

- Building a small full-stack app without a heavy frontend framework.
- Clean HTTP handlers in Go with validation, admin sessions, JSON storage, CSV/JSON export, and tests.
- Accessible UI details: keyboard tabs, focus states, skip link, modal focus trap, live states, and reduced-motion support.
- Product polish: draft saving, floating language switcher, gentler trust copy, shareable result text, static PNG share cards with Canvas fallback, hidden admin tools, and a calm dark UI.
- Practical DevOps basics: Docker, non-root container user, healthcheck, and GitHub Actions.

## Features

- 28 situational questions with live progress and clear selected-answer states.
- MBTI-style result with E/I, S/N, T/F, J/P breakdown.
- Draft saving in `localStorage`, so refresh does not wipe the test.
- Searchable and filterable catalog of all 16 types.
- English socionics codes such as `LII`, `LSI`, `ILI`, and `EIE`.
- UA/RU/EN content layer with situational questions, author type names, localized summaries, long-form profiles, and share-card text.
- Editorial long-form profile modules for all 16 types in Ukrainian, Russian, and English.
- Result page with compact summary, confidence, axis-by-axis reasoning, similar types, Telegram CTA, copy result, and share result card.
- Type detail pages with readable summary cards and collapsible full-profile sections.
- Type Compatibility / Interaction Dynamics: compare any two types in friendship, relationships, or work with a transparent heuristic score, strengths, tensions, tips, and mini-scales.
- Admin panel with login, logout, search, delete, clear, CSV export, JSON export, and a local demo/autopass mode for previewing result pages without saving test data.
- Privacy/disclaimer note and non-diagnostic Socionics orientation copy.
- Responsive dark UI with mobile Safari-friendly controls and reduced-motion support.

## Tech Stack

- Go 1.22
- HTML, CSS, vanilla JavaScript
- Embedded static files via `embed.FS`
- Local JSON file storage
- Docker
- GitHub Actions

## Run Locally

```powershell
$env:HOST = "127.0.0.1"
$env:PORT = "8080"
$env:ADMIN_PASSWORD = "change-me"
go run .
```

Open `http://localhost:8080`.

For a cleaner local setup, copy `.env.example` into your local environment manager or set the variables manually before running the app.

Admin tools are intentionally subtle in the public UI. Locally, click the small `QA` control or open `http://localhost:8080/?admin=1`, then log in with `ADMIN_PASSWORD`. The demo/autopass block can preview any of the 16 result pages without writing to the saved-results store.

## Local Network / iPhone Testing

For phone testing, the iPhone and computer must be on the same Wi-Fi network. Run the server on all local interfaces:

```powershell
$env:HOST = "0.0.0.0"
$env:PORT = "8080"
$env:COOKIE_SECURE = "false"
go run .
```

Find the computer's Wi-Fi IPv4 address with `ipconfig`, then open this on the iPhone:

```text
http://LOCAL_PC_IP:8080
```

Use `http`, not `https`, for local development. Do not use `127.0.0.1` on the phone, because that points to the phone itself. If Windows Firewall asks, allow access on the private network. A longer checklist lives in [docs/local-network-testing.md](docs/local-network-testing.md).

## Environment Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `HOST` | empty | Optional bind host. Use `127.0.0.1` for computer-only local testing or `0.0.0.0` for local network testing. |
| `PORT` | `8080` | HTTP port. Both `8080` and `:8080` are accepted. |
| `ADDR` | empty | Exact bind address override, for example `127.0.0.1:8080`. When set, it wins over `HOST` and `PORT`. |
| `ADMIN_PASSWORD` | `change-me` | Password for the admin panel. Change it before any public deploy. |
| `DATA_FILE` | `data/results.json` | Path for saved results. |
| `COOKIE_SECURE` | `false` | Keep `false` locally. Set to `true` only when running behind HTTPS. |

Runtime data is ignored by Git. The `data/results.json` file is created automatically after the first saved result.

## Project Structure

```text
.
+-- .github/workflows/go.yml
+-- .vscode/settings.json
+-- assets/preview.svg
+-- docs/local-network-testing.md
+-- web/static/
|   +-- app.js
|   +-- assets/share-cards/
|   |   +-- intj.png ... esfp.png
|   +-- compatibility-engine.js
|   +-- result-insights.js
|   +-- content-author.js
|   +-- content-en.js
|   +-- content-profiles-en.js
|   +-- content-profiles-ru.js
|   +-- content-profiles-uk.js
|   +-- content-ru.js
|   +-- content-uk.js
|   +-- index.html
|   +-- style.css
|   +-- types-data.js
+-- .dockerignore
+-- .editorconfig
+-- .env.example
+-- .gitignore
+-- Dockerfile
+-- config.go
+-- handlers.go
+-- main.go
+-- main_test.go
+-- scoring.go
+-- sessions.go
+-- store.go
+-- store_test.go
```

## API

| Method | Route | Description |
| --- | --- | --- |
| `GET` | `/` | Main page. |
| `GET` | `/healthz` | Healthcheck endpoint. |
| `POST` | `/api/submit` | Save a completed test and return the result profile. |
| `POST` | `/api/login` | Admin login. |
| `POST` | `/api/logout` | Admin logout and session cookie cleanup. |
| `GET` | `/api/results` | List saved results. |
| `GET` | `/api/results/export` | Export saved results as CSV. |
| `GET` | `/api/results/export?format=json` | Export saved results as JSON. |
| `DELETE` | `/api/results` | Delete all results. |
| `DELETE` | `/api/results/{id}` | Delete one result. |
| `GET` | `/api/stats` | Return saved result statistics. |

## Quality Checks

```powershell
gofmt -w *.go
go vet ./...
go test ./...
go test -race ./...
go build .
```

For a quick static frontend syntax check when Node.js is available:

```powershell
node -e "const fs=require('fs'); ['web/static/types-data.js','web/static/content-uk.js','web/static/content-ru.js','web/static/content-en.js','web/static/content-author.js','web/static/content-profiles-uk.js','web/static/content-profiles-ru.js','web/static/content-profiles-en.js','web/static/result-insights.js','web/static/compatibility-engine.js','web/static/app.js'].forEach(f=>new Function(fs.readFileSync(f,'utf8'))); console.log('JS_OK')"
```

## Docker

```powershell
docker build -t personality-type-test .
docker run --rm -p 8080:8080 -e ADMIN_PASSWORD="strong-password" personality-type-test
```

The container runs as a non-root user and includes a `/healthz` healthcheck.

The Docker image defaults to `HOST=0.0.0.0` so the container can receive traffic from the published port.

## Author Content Layer

The app keeps the technical i18n foundation in `content-uk.js`, `content-ru.js`, and `content-en.js`. The current editorial pass is applied in `web/static/content-author.js`, which overrides the public-facing content with:

- calmer, more situational questions instead of direct "are you logical or emotional" prompts;
- custom type names that avoid copying common MBTI websites;
- localized UA/RU/EN type summaries with strengths, difficulties, work style, communication style, and growth notes;
- per-type short and deeper share-card text for UA/RU/EN;
- a non-diagnostic tone that treats typology as self-reflection, not as a medical or scientific assessment.

Full profiles live in separate profile modules:

- `web/static/content-profiles-uk.js`
- `web/static/content-profiles-ru.js`
- `web/static/content-profiles-en.js`

This keeps deep editorial content separate from the lighter UI/i18n foundation. Each profile uses the same structured schema: metadata, short profile, 13-section full profile, and share-card copy.

Each profile is modular and uses the same 13-section structure:

1. Short image.
2. Inner logic.
3. Thinking style.
4. Decision making.
5. Work and study.
6. Communication.
7. Motivation.
8. Drains.
9. Strengths.
10. Possible difficulties.
11. Stress behavior.
12. Development.
13. Short summary.

## Share Result Card

The result page includes both copy-result text and a `Share result` flow. The primary share-card path uses final type-specific PNG assets in `web/static/assets/share-cards/`.

Current format:

- `1200x630` PNG for general social and messenger sharing.
- One static PNG per type, named `intj.png`, `intp.png`, ..., `esfp.png`.
- Native Web Share API is used when available.
- Fallbacks: preview modal, PNG download, copy-link fallback, and the older Canvas renderer if a type-specific PNG cannot be loaded.

## Type Compatibility

The compatibility feature is frontend-only and lives in `web/static/compatibility-engine.js`. It compares two MBTI-style type codes across three contexts:

- friendship;
- relationship;
- work/team.

The score is a heuristic interaction score, not a scientific measurement or guarantee. It uses the four dichotomies to estimate social rhythm, information style, decision style, and planning rhythm, then renders context-specific strengths, tensions, practical tips, and four mini-scales. The result page includes a `Compare my type with another` CTA that opens the compatibility tab with the user's result prefilled.

## Result Insights

The scoring model stays unchanged, but the result page adds a clearer explanation layer from `web/static/result-insights.js`:

- result confidence based on the margins between both sides of each dichotomy;
- a short "why this type" explanation for `E/I`, `S/N`, `T/F`, and `J/P`;
- four similar one-letter-neighbor types with short differences;
- privacy-friendly type permalinks such as `/?tab=types&type=INTJ`.

## Scoring V1 and Socionics Orientation

The scoring model is intentionally v1: 28 questions are balanced across four MBTI-style dichotomies, with 7 prompts each for `E/I`, `S/N`, `T/F`, and `J/P`.

Socionics codes are shown as an orientation, not as a precise function-aware diagnosis. The current test does not score Model A or cognitive functions, so the UI avoids presenting the Socionics code as an exact conversion.

This is a product honesty choice: the MBTI-style result is calculated, while the Socionics code is a lightweight comparison marker that should be refined only after a separate function-aware question model exists.

## Design and Product Notes

This is not a medical, psychological, or scientific diagnostic tool. The result is informational and educational-entertainment oriented.

The app can store a submitted name, answers, result type, duration, and submission date in the configured local JSON file. Browser progress is stored in `localStorage` so users can refresh the page without losing answers.

A short Telegram CTA points to `https://t.me/+H1RfT8lJFYA0MDI6` for compact examples and follow-up notes about type behavior.

## Trade-offs

- **Vanilla JS instead of React:** the UI is small enough that a framework would add more setup than value. Plain JavaScript keeps the project transparent for review.
- **JSON storage instead of a database:** local JSON is enough for a portfolio demo and makes the app easy to run. A real public product would eventually need a database and migrations.
- **Simple admin password instead of user accounts:** the project only needs a lightweight local admin panel. Full auth would be overengineering right now.
- **Simple content modules instead of a CMS:** full UA/RU/EN profiles are plain static JS modules. That keeps review and deployment easy for the current project size.


## Content Roadmap

The multilingual editorial profile layer is now in place. Next content work should focus on human rereading, deploy proof, and sharper examples rather than expanding scope.

Recommended order:

1. Run a final human editorial read for all 16 long profiles in UA/RU/EN, with Ukrainian as the canonical source.
2. Deploy a live URL and verify the share card on the real domain.
3. Add real screenshots or a short GIF after browser QA on the deployed URL.
4. Review the question model with a typology-first lens and remove any remaining prompts that feel too self-image based.
5. Turn the existing metadata into a function-aware layer only after the question model is stable.
6. Add optional Telegram integration after the public content flow is proven.
7. Consider SEO/type pages later, without adding SSR until there is a clear need.

## Question Model Roadmap

The current scoring is v1: straightforward MBTI dichotomy scoring across `E/I`, `S/N`, `T/F`, and `J/P`. That keeps the test explainable and easy to verify.

Future versions can evolve gradually:

1. v2: add function-aware metadata to questions while keeping the current scoring stable.
2. v3: map questions to socionics aspects and Model A hypotheses after a separate design pass.
3. v4: produce richer result explanations from both dichotomy scores and function/aspect patterns.

The frontend already keeps optional question metadata fields such as `axis`, `function`, `socionicsAspect`, `weight`, `reverse`, and `tags`. They are intentionally descriptive for now; the backend scoring should only use them after the question model is reviewed carefully.

## Pre-deploy Checklist

- Change `ADMIN_PASSWORD` from the default.
- Set `HOST=0.0.0.0` in containers or local-network testing; keep `HOST=127.0.0.1` for computer-only local work if preferred.
- Set `COOKIE_SECURE=true` only behind HTTPS.
- Confirm the Telegram CTA points to `https://t.me/+H1RfT8lJFYA0MDI6`.
- Run `gofmt`, `go vet ./...`, `go test ./...`, and `go build`.
- Run the JS syntax and content integrity check, including `content-profiles-uk.js`, `content-profiles-ru.js`, `content-profiles-en.js`, and `compatibility-engine.js`.
- Smoke-test `/`, static assets, `/healthz`, submit, result page, type detail page, compatibility tab, floating language switching, share-card preview, copy result, hidden admin access, admin demo/autopass, admin login/logout, CSV export, and JSON export.
- Manually review desktop and mobile UI for Safari/iOS link colors, selected answers, result layout, type cards, full profile accordions, floating controls, hidden admin control, Telegram CTA, and PNG share card.
- For phone QA, verify `http://LOCAL_PC_IP:PORT` from a real iPhone on the same Wi-Fi network.

## Deployment Notes

The app is deployable as a single Go service with embedded static assets and local JSON storage. For a public deployment, use a persistent volume for `DATA_FILE` or replace the store with a database in a later version.

Recommended production environment:

```powershell
$env:ADMIN_PASSWORD = "strong-password"
$env:HOST = "0.0.0.0"
$env:COOKIE_SECURE = "true"
$env:DATA_FILE = "/data/results.json"
```

Production HTTPS should be provided by the deploy platform or domain. The app does not create local HTTPS certificates. After the live URL is chosen, verify the share card visually once on the deployed domain.

## Screenshots / Demo

The repository includes `assets/preview.svg` as a lightweight preview. For a stronger portfolio presentation, add real screenshots or a short GIF after the next visual QA pass.

Suggested files:

```text
assets/screenshot-test.png
assets/screenshot-result.png
assets/screenshot-types.png
```

## Future Improvements

- Deploy the live site and verify share-card URLs on the public domain.
- Do a final human wording pass over UA/RU/EN long profiles after testing them in the real UI.
- Add end-to-end browser tests for the quiz and admin flows.
- Add pagination for admin results if the JSON file grows large.
- Add platform-specific notes for Render, Fly.io, or Railway after choosing the host.
- Continue polishing the Telegram content flow once the channel has a stable editorial format.

