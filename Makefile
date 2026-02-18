.PHONY: build test lint fmt run

build:
	go build -o ccost ./cmd/ccost

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

run:
	go run ./cmd/ccost