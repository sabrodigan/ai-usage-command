BINARY_NAME=usage
BUILD_DIR=dist
# Single source of truth for the version is cmd/usage/main.go.
VERSION := $(shell sed -n 's/^const Version = "\(.*\)"/\1/p' cmd/usage/main.go)
LDFLAGS=-s -w
PLUGIN_DIR ?= $(HOME)/Projects/omarchy-ai-usage

.PHONY: all build build-all test check fmt-check vet plugin-test version release clean install

all: check build

version:
	@echo $(VERSION)

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/usage
	@cp $(BUILD_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	@echo "✔ Binary built: ./$(BINARY_NAME) (v$(VERSION))"

build-all:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/usage
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/usage
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/usage
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/usage
	cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > SHA256SUMS
	@echo "✔ v$(VERSION) binaries and SHA256SUMS in $(BUILD_DIR)/"

fmt-check:
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

vet:
	go vet ./...

test:
	go test -race ./...

# Everything CI runs for this repo.
check: fmt-check vet test

# Run the Omarchy plugin's tests (including its end-to-end contract test) against
# a freshly built binary, so a CLI change can't silently break the widget.
plugin-test: build
	@test -d $(PLUGIN_DIR) || { echo "Plugin checkout not found at $(PLUGIN_DIR) (set PLUGIN_DIR)"; exit 1; }
	cd $(PLUGIN_DIR) && USAGE_BIN=$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) python3 -m unittest discover -s tests

# Tag v$(VERSION) and push it; the release workflow builds and publishes binaries.
# Bump `const Version` in cmd/usage/main.go and commit before running this.
release: check plugin-test
	@test -z "$$(git status --porcelain --untracked-files=no)" || { echo "Commit your changes first"; exit 1; }
	@! git rev-parse -q --verify "refs/tags/v$(VERSION)" >/dev/null || { echo "Tag v$(VERSION) already exists: bump const Version"; exit 1; }
	git tag -a v$(VERSION) -m "usage v$(VERSION)"
	git push origin HEAD v$(VERSION)

install: build
	@mkdir -p $(HOME)/.local/bin $(HOME)/go/bin
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/go/bin/$(BINARY_NAME) 2>/dev/null || true
	@echo "✔ Installed $(BINARY_NAME) v$(VERSION) to ~/.local/bin/$(BINARY_NAME)"

clean:
	rm -rf $(BUILD_DIR) $(BINARY_NAME)
