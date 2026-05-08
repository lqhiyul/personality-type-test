.PHONY: fmt vet test race build js-check check

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

build:
	go build ./...

js-check:
	node scripts/js-check.mjs

check: fmt vet test race build js-check
