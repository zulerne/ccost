.PHONY: build test clean install lint fmt run

build:
	go build -o ccost ./cmd/ccost

test:
	go test ./...

clean:
	rm -f ccost

install:
	go install ./cmd/ccost

lint:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./ccost
