.PHONY: build lint test clean tidy install

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

install: build
	@echo "Installing binaries to /usr/bin/..."
	@if ! cp bin/ads /usr/bin/ads 2>/dev/null; then \
		echo "WARNING: Failed to copy 'ads' to /usr/bin/. Permission denied. Please run 'sudo make install'."; \
	else \
		echo "Successfully installed 'ads' to /usr/bin/ads"; \
	fi
	@if ! cp bin/ads-recorder /usr/bin/ads-recorder 2>/dev/null; then \
		echo "WARNING: Failed to copy 'ads-recorder' to /usr/bin/. Permission denied. Please run 'sudo make install'."; \
	else \
		echo "Successfully installed 'ads-recorder' to /usr/bin/ads-recorder"; \
	fi
