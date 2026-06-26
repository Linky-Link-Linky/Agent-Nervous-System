BINARY      := ans
TUI_BINARY  := ans-tui
CMD         := ./cmd/ans
TUI_CMD     := ./ans-tui/cmd/ans-tui
BIN         := bin
VERSION     := $(shell git describe --tags --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X main.version=$(VERSION)

.PHONY: build build-tui build-all test test-race test-cover bench vet clean install checksums

build:
	@mkdir -p $(BIN)
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(BINARY) $(CMD)

build-tui:
	@mkdir -p $(BIN)
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(TUI_BINARY) $(TUI_CMD)

build-all:
	@mkdir -p $(BIN)
	GOOS=linux   GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(BINARY)_linux_amd64      $(CMD)
	GOOS=linux   GOARCH=arm64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(BINARY)_linux_arm64      $(CMD)
	GOOS=darwin  GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(BINARY)_darwin_amd64     $(CMD)
	GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(BINARY)_darwin_arm64     $(CMD)
	GOOS=windows GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(BINARY)_windows_amd64.exe $(CMD)
	GOOS=linux   GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(TUI_BINARY)_linux_amd64      $(TUI_CMD)
	GOOS=linux   GOARCH=arm64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(TUI_BINARY)_linux_arm64      $(TUI_CMD)
	GOOS=darwin  GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(TUI_BINARY)_darwin_amd64     $(TUI_CMD)
	GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(TUI_BINARY)_darwin_arm64     $(TUI_CMD)
	GOOS=windows GOARCH=amd64  CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BIN)/$(TUI_BINARY)_windows_amd64.exe $(TUI_CMD)

test:
	go test ./... ./ans-tui/...

test-race:
	go test -race ./... ./ans-tui/...

test-cover:
	go test -coverprofile=coverage.out ./... ./ans-tui/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	go test -bench=. -benchmem ./... ./ans-tui/...

vet:
	go vet ./... ./ans-tui/...

clean:
	rm -rf $(BIN) coverage.out coverage.html

install: build build-tui
	@if [ -w /usr/local/bin ]; then \
		cp $(BIN)/$(BINARY) /usr/local/bin/$(BINARY); \
		cp $(BIN)/$(TUI_BINARY) /usr/local/bin/$(TUI_BINARY); \
	else \
		sudo cp $(BIN)/$(BINARY) /usr/local/bin/$(BINARY); \
		sudo cp $(BIN)/$(TUI_BINARY) /usr/local/bin/$(TUI_BINARY); \
	fi
	@echo "Installed: /usr/local/bin/$(BINARY) and /usr/local/bin/$(TUI_BINARY)"

checksums: build-all
	@cd $(BIN) && \
		if command -v sha256sum >/dev/null 2>&1; then \
			sha256sum $(BINARY)_* $(TUI_BINARY)_* > checksums.txt; \
		elif command -v shasum >/dev/null 2>&1; then \
			shasum -a 256 $(BINARY)_* $(TUI_BINARY)_* > checksums.txt; \
		else \
			echo "no sha256sum or shasum found" >&2; exit 1; \
		fi && \
		cat checksums.txt

