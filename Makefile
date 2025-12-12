.PHONY: build test clean docker docker-universal install all help lint dist

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

# Binary name
BINARY := image-tools-mcp

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

# Default target
all: build

# Help
help:
	@echo "Image MCP Build System"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build binary for current platform"
	@echo "  test           Run tests"
	@echo "  lint           Run linters"
	@echo "  clean          Remove build artifacts"
	@echo "  docker         Build Docker image for current platform"
	@echo "  docker-universal  Build universal Docker image (requires binaries in dist/)"
	@echo "  dist           Build binaries for all platforms"
	@echo "  install        Install binary to /usr/local/bin"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION        Version string (default: git tag or 'dev')"

# Build for current platform
build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY) ./cmd/image-mcp

# Run tests
test:
	go test -v -race ./...

# Run linters
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
	docker rmi image-tools-mcp:latest image-tools-mcp:$(VERSION) 2>/dev/null || true

# Build Docker image
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		-t image-tools-mcp:$(VERSION) \
		-t image-tools-mcp:latest \
		.

# Build universal Docker image (requires dist/ with all binaries)
docker-universal: dist
	docker build \
		--build-arg VERSION=$(VERSION) \
		-f Dockerfile.universal \
		-t image-tools-mcp:$(VERSION)-universal \
		-t image-tools-mcp:universal \
		.

# Cross-compile for all platforms
dist:
	@mkdir -p dist
	@echo "Building for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="dist/$(BINARY)-$${GOOS}-$${GOARCH}"; \
		if [ "$${GOOS}" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $${output}..."; \
		if [ "$${GOOS}" = "linux" ]; then \
			CGO_ENABLED=1 GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(LDFLAGS) -o $${output} ./cmd/image-mcp || echo "Warning: CGO build failed for $${platform}, trying without CGO"; \
		else \
			CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(LDFLAGS) -o $${output} ./cmd/image-mcp; \
		fi; \
	done
	@echo "Done! Binaries in dist/"

# Build Linux binaries only (for quick testing)
dist-linux:
	@mkdir -p dist
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/image-mcp
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/image-mcp

# Install to system
install: build
	sudo cp $(BINARY) /usr/local/bin/

# Run the server (for testing)
run: build
	./$(BINARY)

# Test MCP protocol
test-mcp: build
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./$(BINARY)
	@echo ""
	@echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./$(BINARY)

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Generate mocks (if needed)
generate:
	go generate ./...

# Check for vulnerabilities
vuln:
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Update dependencies
deps:
	go mod tidy
	go mod download
