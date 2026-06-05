.PHONY: build lint test clean tidy

all: lint test build

build:
	go build -o bin/ads ./cmd/ads
	go build -o bin/ads-recorder ./cmd/ads-recorder

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

test:
	go test -v ./...

test-integration:
	@echo "Running integration tests..."

clean:
	rm -rf bin/

tidy:
	go mod tidy
