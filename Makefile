.PHONY: build test lint fmt run clean coverage

VERSION ?= $(shell git describe --tags --always --dirty)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o ccost ./cmd/ccost

test:
	go test ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

run:
	go run ./cmd/ccost

clean:
	rm -f ccost
	rm -f coverage.out

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
