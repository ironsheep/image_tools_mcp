# MCP Coexistence Installation Guide

How to package your MCP server to install alongside other MCPs in `/opt/container-tools/`.

## Overview

This guide shows how to create a distribution package that:
- Installs into the shared `/opt/container-tools/` directory
- Coexists with other MCP servers (like todo-mcp)
- Merges its configuration into the existing `mcp.json`
- Uses a universal launcher for cross-platform support

## Directory Structure

The `/opt/container-tools/` directory is shared by multiple MCPs:

```
/opt/container-tools/
├── etc/
│   └── mcp.json                    # Shared config - each MCP adds its entry
├── opt/
│   ├── todo-mcp/                   # Another MCP's namespace
│   │   └── ...
│   └── your-mcp/                   # YOUR MCP's namespace
│       ├── bin/
│       │   ├── your-mcp            # Universal launcher script
│       │   └── platforms/          # Platform-specific binaries
│       │       ├── your-mcp-v1.0.0-darwin-amd64
│       │       ├── your-mcp-v1.0.0-darwin-arm64
│       │       ├── your-mcp-v1.0.0-linux-amd64
│       │       ├── your-mcp-v1.0.0-linux-arm64
│       │       └── your-mcp-v1.0.0-windows-amd64.exe
│       ├── test-platforms.sh
│       └── README.md
├── bin/                            # Shared user commands (optional)
├── internal/                       # Shared internal tools (optional)
└── templates/                      # Shared templates (optional)
```

**Key principle:** Each MCP owns only its `opt/{mcp-name}/` directory. The `etc/mcp.json` is shared and must be merged, not replaced.

## What Your Package Should Contain

Your distribution tarball structure:

```
your-mcp-v1.0.0/
├── opt/
│   └── your-mcp/
│       ├── bin/
│       │   ├── your-mcp              # Universal launcher
│       │   └── platforms/
│       │       └── (5 platform binaries)
│       ├── test-platforms.sh
│       └── README.md
├── install.sh                        # Handles merging into mcp.json
└── VERSION_MANIFEST.txt
```

## Step 1: Create VERSION File

At your repository root, create a `VERSION` file:

```
1.0.0
```

## Step 2: Create the Build Script

Create `scripts/build-container-tools.sh`:

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Configuration - CHANGE THESE FOR YOUR MCP
MCP_NAME="your-mcp"
MAIN_PACKAGE="./cmd/your-mcp"    # Path to your Go main package

# Get version
VERSION=$(cat "${REPO_ROOT}/VERSION" | tr -d '\n')
PACKAGE_NAME="${MCP_NAME}-v${VERSION}"
BUILD_DIR="${REPO_ROOT}/builds/container-tools"
PACKAGE_DIR="${BUILD_DIR}/${PACKAGE_NAME}"

# Build metadata
BUILD_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "Building ${MCP_NAME} v${VERSION}..."

# Clean and create directories
rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/platforms"

# Build function
build_binary() {
    local os=$1
    local arch=$2
    local suffix=$3
    local output="${PACKAGE_DIR}/opt/${MCP_NAME}/bin/platforms/${MCP_NAME}-v${VERSION}-${os}-${arch}${suffix}"

    echo "  Building ${os}-${arch}..."

    LDFLAGS="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_DATE}' -X 'main.GitCommit=${GIT_COMMIT}'"
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -ldflags="${LDFLAGS}" -o "${output}" "${MAIN_PACKAGE}"
}

# Build all platforms
build_binary "darwin" "amd64" ""
build_binary "darwin" "arm64" ""
build_binary "linux" "amd64" ""
build_binary "linux" "arm64" ""
build_binary "windows" "amd64" ".exe"

# Create universal launcher (see next section)
# Create install.sh (see below)
# Create tarball

cd "${BUILD_DIR}"
tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}"
echo "Created: ${BUILD_DIR}/${PACKAGE_NAME}.tar.gz"
```

## Step 3: Create the Universal Launcher

The launcher auto-detects the platform and runs the correct binary:

```bash
#!/usr/bin/env bash
set -e

LAUNCHER_VERSION="__VERSION__"  # Replaced during build
BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORMS_DIR="${BIN_DIR}/platforms"
MCP_NAME="your-mcp"  # CHANGE THIS

# Container detection
is_container() {
    [ -f /.dockerenv ] && return 0
    [ -n "${CONTAINER}" ] && return 0
    [ -f /proc/1/cgroup ] && grep -qE 'docker|containerd|podman|kubernetes' /proc/1/cgroup 2>/dev/null && return 0
    return 1
}

# Architecture detection
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}

# OS detection
detect_os() {
    case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
        darwin) echo "darwin" ;;
        linux) echo "linux" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

# Select and execute binary
select_binary() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    local suffix=""

    # Always use Linux in containers
    is_container && os="linux"
    [ "$os" = "windows" ] && suffix=".exe"

    echo "${PLATFORMS_DIR}/${MCP_NAME}-v${LAUNCHER_VERSION}-${os}-${arch}${suffix}"
}

exec "$(select_binary)" "$@"
```

## Step 4: Create the Install Script

The install script is **critical** - it merges into the existing `mcp.json` instead of replacing it:

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MCP_NAME="your-mcp"  # CHANGE THIS
INSTALL_ROOT="/opt/container-tools"
MCP_JSON="${INSTALL_ROOT}/etc/mcp.json"

echo "Installing ${MCP_NAME}..."

# Check for root/sudo
if [ "$EUID" -ne 0 ] && [ ! -w "$INSTALL_ROOT" ]; then
    exec sudo "$0" "$@"
fi

# Create base directories if needed
mkdir -p "${INSTALL_ROOT}/opt"
mkdir -p "${INSTALL_ROOT}/etc"

# Backup existing if present
if [ -d "${INSTALL_ROOT}/opt/${MCP_NAME}" ]; then
    mv "${INSTALL_ROOT}/opt/${MCP_NAME}" "${INSTALL_ROOT}/opt/${MCP_NAME}.backup.$(date +%Y%m%d%H%M%S)"
fi

# Copy our files
cp -r "${SCRIPT_DIR}/opt/${MCP_NAME}" "${INSTALL_ROOT}/opt/"
chmod +x "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}"
chmod +x "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/platforms"/*

# CRITICAL: Merge into mcp.json (don't replace!)
if [ -f "$MCP_JSON" ]; then
    if command -v jq &> /dev/null; then
        # Use jq to merge
        TEMP_JSON=$(mktemp)
        jq --arg cmd "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}" \
           ".mcpServers[\"${MCP_NAME}\"] = {\"command\": \$cmd, \"args\": []}" \
           "$MCP_JSON" > "$TEMP_JSON"
        mv "$TEMP_JSON" "$MCP_JSON"
    else
        echo "WARNING: jq not installed. Manually add to ${MCP_JSON}:"
        echo "  \"${MCP_NAME}\": {\"command\": \"${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}\", \"args\": []}"
    fi
else
    # Create new mcp.json
    cat > "$MCP_JSON" << EOF
{
  "mcpServers": {
    "${MCP_NAME}": {
      "command": "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}",
      "args": []
    }
  }
}
EOF
fi

echo "Installation complete!"
echo "Test: ${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME} --version"
```

## Step 5: Version Variables in Go

Your Go code must have these variables for ldflags injection:

```go
// cmd/your-mcp/main.go
package main

var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func main() {
    // Handle --version flag
    if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
        fmt.Printf("%s %s\n", "your-mcp", Version)
        fmt.Printf("  Build time: %s\n", BuildTime)
        fmt.Printf("  Git commit: %s\n", GitCommit)
        return
    }

    // ... rest of your server
}
```

## Step 6: GitHub Actions Release Workflow

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  build-container-tools:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build container-tools package
        run: ./scripts/build-container-tools.sh

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: container-tools-package
          path: builds/container-tools/*.tar.gz

  create-release:
    needs: build-container-tools
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/download-artifact@v4
        with:
          name: container-tools-package
          path: release/

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: release/*.tar.gz
          body: |
            ## Installation

            ```bash
            tar -xzf your-mcp-v${{ github.ref_name }}.tar.gz
            cd your-mcp-v${{ github.ref_name }}
            ./install.sh
            ```
```

## Installation Flow for Users

```bash
# Download the release
curl -LO https://github.com/you/your-mcp/releases/download/v1.0.0/your-mcp-v1.0.0.tar.gz

# Extract
tar -xzf your-mcp-v1.0.0.tar.gz

# Install (merges with existing MCPs)
cd your-mcp-v1.0.0
./install.sh

# Verify
/opt/container-tools/opt/your-mcp/bin/your-mcp --version
```

## Result: Coexisting MCPs

After installation, `mcp.json` contains both MCPs:

```json
{
  "mcpServers": {
    "todo-mcp": {
      "command": "/opt/container-tools/opt/todo-mcp/bin/todo-mcp",
      "args": ["--mode", "stdio"]
    },
    "your-mcp": {
      "command": "/opt/container-tools/opt/your-mcp/bin/your-mcp",
      "args": []
    }
  },
  "hooks": { ... },
  "commands": { ... }
}
```

Both work together, each in their own namespace.

## Checklist

- [ ] `VERSION` file at repo root
- [ ] Version variables in Go code (`Version`, `BuildTime`, `GitCommit`)
- [ ] `--version` flag handling in main()
- [ ] `scripts/build-container-tools.sh` builds all 5 platforms
- [ ] Universal launcher script with container detection
- [ ] `install.sh` **merges** into mcp.json (uses `jq`)
- [ ] GitHub Actions workflow creates tarball release
- [ ] Test installation on clean system
- [ ] Test installation alongside existing MCP

## Key Principles

1. **Own only your namespace**: `opt/{mcp-name}/` is yours, everything else is shared
2. **Merge, don't replace**: The `mcp.json` belongs to everyone
3. **Universal launcher**: One entry point, works everywhere
4. **Self-contained binaries**: CGO_ENABLED=0 for maximum portability
5. **Version in filenames**: Allows rollback and multiple versions
