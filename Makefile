.PHONY: build test clean docker docker-universal install all help lint dist dist-static

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

# Binary name
BINARY := image-tools-mcp

# Host architecture in Go's GOARCH vocabulary (used to name the static binary).
HOST_ARCH := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

# Static build target arch. Leave STATIC_PLATFORM empty for a host-native build
# (no emulation). Set STATIC_PLATFORM=linux/amd64 or linux/arm64 to force the
# other architecture via buildx --platform (accepts qemu, local-only use).
STATIC_PLATFORM ?=
STATIC_ARCH := $(if $(STATIC_PLATFORM),$(notdir $(STATIC_PLATFORM)),$(HOST_ARCH))

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
	@echo "  dist-static    Build a fully static Linux binary via Dockerfile.static"
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

# Build a fully static, zero-dependency Linux binary via Dockerfile.static and
# extract it to dist/. Host-native by default (no emulation). Cross-arch builds
# are the release workflow's job on native runners; for a local-only cross build
# pass STATIC_PLATFORM=linux/<arch> (uses buildx --platform + qemu).
#   make dist-static                          # host arch
#   make dist-static STATIC_PLATFORM=linux/arm64
dist-static:
	@mkdir -p dist
	@echo "Building static linux/$(STATIC_ARCH) binary via Dockerfile.static..."
	docker buildx build \
		-f Dockerfile.static \
		--target export-binaries \
		$(if $(STATIC_PLATFORM),--platform $(STATIC_PLATFORM),) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		-o type=local,dest=dist/static-tmp \
		.
	@mv dist/static-tmp/image-tools-mcp dist/$(BINARY)-linux-$(STATIC_ARCH)
	@rm -rf dist/static-tmp
	@echo "Done! Static binary: dist/$(BINARY)-linux-$(STATIC_ARCH)"

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
