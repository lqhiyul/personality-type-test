.PHONY: help fmt fmt-check vet staticcheck test race coverage build run dev migrate seed js-check e2e docker-build check full-check

help:
	@printf "%s\n" \
		"Targets:" \
		"  fmt           format Go code" \
		"  fmt-check     fail if Go files need formatting" \
		"  vet           run go vet" \
		"  staticcheck   run pinned staticcheck" \
		"  test          run Go tests" \
		"  race          run Go race tests (requires CGO)" \
		"  coverage      run Go coverage and print function summary" \
		"  build         build all Go packages" \
		"  run           start the server" \
		"  js-check      check frontend JavaScript" \
		"  e2e           run Playwright smoke tests" \
		"  docker-build  build the Docker image" \
		"  check         regular local quality gate" \
		"  full-check    check plus slower E2E/Docker/race checks"

fmt:
	go fmt ./...

fmt-check:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...

test:
	go test ./...

race:
	go test -race ./...

coverage:
	go test ./... "-coverprofile=coverage.out"
	go tool cover "-func=coverage.out"

build:
	go build ./...

run:
	go run ./cmd/server

dev: run

migrate:
	go run ./cmd/migrate

seed:
	go run ./cmd/seed

js-check:
	npm run js-check

e2e:
	npm run e2e

docker-build:
	docker build -t personality-type-test .

check: fmt-check vet staticcheck test build js-check

full-check: check race coverage e2e docker-build
