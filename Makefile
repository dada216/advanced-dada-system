.PHONY: build lint test clean tidy

all: lint test build

build:
	go build -tags sqlite_fts5 -o bin/ads ./cmd/ads
	go build -tags sqlite_fts5 -o bin/ads-recorder ./cmd/ads-recorder

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

test:
	go test -v ./...

test-integration:
	bash integration.sh

clean:
	rm -rf bin/

tidy:
	go mod tidy
