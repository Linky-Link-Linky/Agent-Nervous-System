BINARY   := ans
CMD      := ./cmd/ans
BIN      := bin

.PHONY: build build-all test test-race test-cover bench vet clean install checksums

build:
	@mkdir -p $(BIN)
	go build -ldflags="-s -w" -trimpath -o $(BIN)/$(BINARY) $(CMD)

build-all:
	@mkdir -p $(BIN)
	GOOS=linux   GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN)/$(BINARY)_linux_amd64   $(CMD)
	GOOS=linux   GOARCH=arm64  CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN)/$(BINARY)_linux_arm64   $(CMD)
	GOOS=darwin  GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN)/$(BINARY)_darwin_amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN)/$(BINARY)_darwin_arm64  $(CMD)
	GOOS=windows GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN)/$(BINARY)_windows_amd64.exe $(CMD)

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN) coverage.out coverage.html

install: build
	@if [ -w /usr/local/bin ]; then \
		cp $(BIN)/$(BINARY) /usr/local/bin/$(BINARY); \
	else \
		sudo cp $(BIN)/$(BINARY) /usr/local/bin/$(BINARY); \
	fi
	@echo "Installed: /usr/local/bin/$(BINARY)"

checksums: build-all
	@cd $(BIN) && \
		if command -v sha256sum >/dev/null 2>&1; then \
			sha256sum $(BINARY)_* > checksums.txt; \
		elif command -v shasum >/dev/null 2>&1; then \
			shasum -a 256 $(BINARY)_* > checksums.txt; \
		else \
			echo "no sha256sum or shasum found" >&2; exit 1; \
		fi && \
		cat checksums.txt
