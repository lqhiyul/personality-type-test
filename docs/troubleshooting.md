# Troubleshooting

## Admin Login Fails In Production

Production mode rejects `ADMIN_PASSWORD=change-me` and requires `COOKIE_SECURE=true`.

## Login Works Locally But Not Behind A Proxy

Check that HTTPS is terminating correctly and that `COOKIE_SECURE=true` is set only when the browser reaches the site over HTTPS. Configure `TRUSTED_PROXY_CIDRS` only for your proxy addresses.

## E2E Tests Cannot Find Chromium

Run:

```bash
npx playwright install chromium
```

CI installs the browser automatically.

## Race Tests Fail With CGO Disabled

`go test -race ./...` requires CGO. Linux CI uses the default CGO-enabled toolchain. On Windows, install a compatible C compiler and run with CGO enabled, or use CI as the race-test source of truth.

## Docker Build Cannot Connect To The Daemon

Start Docker Desktop or another Docker daemon before running:

```bash
docker build -t personality-type-test .
```

## Static Files Missing In Docker

Rebuild the image after frontend changes. The Dockerfile copies `web/static` into the runtime image.
