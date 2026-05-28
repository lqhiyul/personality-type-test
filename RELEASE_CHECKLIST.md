# Release Checklist

Use this before tagging or sending the repository to a reviewer.

## Required Local Checks

```bash
make check
go test -race ./...
make coverage
npm run e2e
docker build -t personality-type-test .
```

If `make` is unavailable, run the equivalent commands:

```bash
gofmt -l .
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
go test ./...
go build ./...
npm run js-check
go test -race ./...
go test ./... "-coverprofile=coverage.out"
go tool cover "-func=coverage.out"
npm run e2e
docker build -t personality-type-test .
```

## Manual Review

- Confirm `.github/workflows/go.yml` is green on the target branch.
- Confirm `.github/workflows/codeql.yml` is present and enabled.
- Confirm `README.md`, `docs/api.md`, `docs/security.md`, and `docs/reviewer-guide.md` match the current behavior.
- Confirm `docs/openapi.yaml` is still described as partial unless it is expanded to cover every endpoint.
- Confirm `.env`, local databases, coverage output, Playwright reports, `node_modules`, binaries, and logs are not staged.
- Confirm the deployed demo link in `README.md` is reachable, or remove it before release.

## Environment Notes

- `go test -race ./...` requires CGO. On Windows, install a compatible C compiler and enable CGO or rely on the Linux CI race job.
- `npm run e2e` starts the Go server on `127.0.0.1:18080` and writes ignored runtime data under `.e2e-data/`.
- `docker build` requires a running Docker daemon.
