.PHONY: build lint test clean tidy install

all: lint test build

build:
	go build -mod=vendor -tags sqlite_fts5 -o bin/ads ./cmd/ads
	go build -mod=vendor -tags sqlite_fts5 -o bin/ads-recorder ./cmd/ads-recorder

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

test:
	go test -mod=vendor -v ./...

test-integration:
	bash integration.sh

clean:
	rm -rf bin/

tidy:
	go mod tidy

install: build
	@echo "Installing binaries to /usr/bin/..."
	@if ! install -m 755 bin/ads /usr/bin/ads 2>/dev/null; then \
		echo "WARNING: Failed to install 'ads' to /usr/bin/. Permission denied. Please run 'sudo make install'."; \
	else \
		echo "Successfully installed 'ads' to /usr/bin/ads"; \
	fi
	@if ! install -m 755 bin/ads-recorder /usr/bin/ads-recorder 2>/dev/null; then \
		echo "WARNING: Failed to install 'ads-recorder' to /usr/bin/. It might be running or permission denied. Please run 'sudo make install'."; \
	else \
		echo "Successfully installed 'ads-recorder' to /usr/bin/ads-recorder"; \
	fi
