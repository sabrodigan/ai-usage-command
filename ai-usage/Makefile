BINARY_NAME=usage
BUILD_DIR=dist

.PHONY: all build build-all test clean install

all: test build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/usage/main.go
	@cp $(BUILD_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	@echo "✔ Binary built: ./$(BINARY_NAME)"

build-all: test
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/usage/main.go
	@echo "Building for Linux arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 cmd/usage/main.go
	@echo "Building for Windows amd64..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/usage/main.go
	@echo "✔ Cross-platform binaries created in $(BUILD_DIR)/"

test:
	go test -v -race ./...

install: build
	@mkdir -p $(HOME)/.local/bin $(HOME)/go/bin
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/go/bin/$(BINARY_NAME) 2>/dev/null || true
	@echo "✔ Installed $(BINARY_NAME) to ~/.local/bin/$(BINARY_NAME)"

clean:
	rm -rf $(BUILD_DIR) $(BINARY_NAME)
