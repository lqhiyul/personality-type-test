.PHONY: fmt vet staticcheck test race coverage build run dev migrate seed js-check e2e docker-build check full-check

fmt:
	go fmt ./...

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

test:
	go test ./...

race:
	go test -race ./...

coverage:
	go test ./... -coverprofile=coverage.out

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

check: fmt vet staticcheck test build js-check

full-check: check race coverage e2e docker-build
