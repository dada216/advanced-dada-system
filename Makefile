.PHONY: build lint test clean tidy install install-konsole

all: lint test build

build:
	go build -mod=vendor -tags sqlite_fts5 -o bin/ads ./cmd/ads
	go build -mod=vendor -tags sqlite_fts5 -o bin/ads-shell ./cmd/ads-shell
	go build -mod=vendor -tags sqlite_fts5 -o bin/ads-plugin-search ./cmd/ads-plugin-search
	go build -mod=vendor -tags sqlite_fts5 -o bin/ads-plugin-llm ./cmd/ads-plugin-llm

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

test:
	go test -mod=vendor -tags sqlite_fts5 -v ./...

test-integration:
	bash integration.sh

release:
	@echo "Creating Git tag for release..."
	@VERSION=$$(grep -Eo 'Version = "[^"]+"' cmd/ads/version.go | cut -d'"' -f2) && \
	git tag -a $$VERSION -m "Release $$VERSION" && \
	echo "Successfully tagged $$VERSION. Push with 'git push origin $$VERSION'"

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
	@if ! install -m 755 bin/ads-shell /usr/bin/ads-shell 2>/dev/null; then \
		echo "WARNING: Failed to install 'ads-shell' to /usr/bin/. It might be running or permission denied. Please run 'sudo make install'."; \
	else \
		echo "Successfully installed 'ads-shell' to /usr/bin/ads-shell"; \
	fi
	@if ! install -m 755 bin/ads-plugin-search /usr/bin/ads-plugin-search 2>/dev/null; then \
		echo "WARNING: Failed to install 'ads-plugin-search'."; \
	else \
		echo "Successfully installed 'ads-plugin-search'"; \
	fi
	@if ! install -m 755 bin/ads-plugin-llm /usr/bin/ads-plugin-llm 2>/dev/null; then \
		echo "WARNING: Failed to install 'ads-plugin-llm'."; \
	else \
		echo "Successfully installed 'ads-plugin-llm'"; \
	fi

install-konsole:
	@echo "Installing Konsole profile..."
	@bash scripts/install-konsole.sh
