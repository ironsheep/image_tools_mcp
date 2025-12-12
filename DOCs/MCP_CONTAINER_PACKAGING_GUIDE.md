# MCP Container Packaging Guide

A comprehensive guide for packaging MCP servers for cross-platform distribution with Docker/container support.

## Overview

This packaging system creates a self-contained, multi-platform distribution that:
- Works on macOS (Intel & Apple Silicon), Linux (AMD64 & ARM64), and Windows (via WSL2)
- Auto-detects host vs container environment
- Uses a universal launcher script to select the correct binary
- Includes MCP configuration for Claude Code integration
- **Optionally** provides init tooling and templates (see [Optional: Init Infrastructure](#optional-init-infrastructure))

---

# PART 1: Core Package (Required)

This section covers the minimum required components for any MCP container package.

## Minimal Package Structure

```
{mcp-name}-container-tools-v{version}/
├── etc/
│   └── mcp.json                      # MCP server configuration for Claude
├── opt/{mcp-name}/
│   ├── bin/
│   │   ├── {mcp-name}                # Universal launcher script
│   │   └── platforms/                # Platform-specific binaries
│   │       ├── {mcp-name}-v{version}-darwin-amd64
│   │       ├── {mcp-name}-v{version}-darwin-arm64
│   │       ├── {mcp-name}-v{version}-linux-amd64
│   │       ├── {mcp-name}-v{version}-linux-arm64
│   │       └── {mcp-name}-v{version}-windows-amd64.exe
│   ├── test-platforms.sh             # Verification script
│   └── README.md                     # Package documentation
└── VERSION_MANIFEST.txt              # Build metadata (optional but recommended)
```

This is all you need for a working MCP distribution.

## Installation Location

Standard installation path: `/opt/container-tools/`

This location is:
- Accessible to both host and container environments
- Mounted read-only into containers
- Outside of user home directories (avoids permission issues)

## Core Components

### 1. Universal Launcher Script

The launcher (`opt/{mcp-name}/bin/{mcp-name}`) auto-selects the correct binary:

```bash
#!/usr/bin/env bash
set -e

DIST_VERSION="{version}"
BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORMS_DIR="${BIN_DIR}/platforms"

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
        *) echo "unknown" ;;
    esac
}

# Binary selection logic
select_binary() {
    local os=$(detect_os)
    local arch=$(detect_arch)

    if is_container; then
        # Always use Linux binary in containers
        echo "${PLATFORMS_DIR}/{mcp-name}-v${DIST_VERSION}-linux-${arch}"
    else
        echo "${PLATFORMS_DIR}/{mcp-name}-v${DIST_VERSION}-${os}-${arch}"
    fi
}

# Execute selected binary
exec "$(select_binary)" "$@"
```

### 2. MCP Configuration (`etc/mcp.json`)

```json
{
  "mcpServers": {
    "{mcp-name}": {
      "command": "/opt/container-tools/opt/{mcp-name}/bin/{mcp-name}",
      "args": ["--mode", "stdio"]
    }
  },
  "hooks": {
    "app-start": "/opt/container-tools/templates/hooks/app-start-hook.sh"
  }
}
```

### 3. Platform Binaries

Binary naming convention: `{mcp-name}-v{version}-{os}-{arch}[.exe]`

| Platform | Binary Name |
|----------|-------------|
| macOS Intel | `{mcp-name}-v{version}-darwin-amd64` |
| macOS Apple Silicon | `{mcp-name}-v{version}-darwin-arm64` |
| Linux AMD64 | `{mcp-name}-v{version}-linux-amd64` |
| Linux ARM64 | `{mcp-name}-v{version}-linux-arm64` |
| Windows | `{mcp-name}-v{version}-windows-amd64.exe` |

## Build Script Structure

### Source File Layout

```
your-mcp-repo/
├── VERSION                          # Version number (e.g., "0.6.8.2")
├── cmd/{mcp-name}/                  # Go main package
├── scripts/
│   └── build-container-tools.sh     # Main build script
├── container-tools-templates/       # Source templates
│   ├── bin/
│   │   └── {tool}-init              # Init command template
│   └── hooks/
│       ├── app-start-hook.sh
│       └── ...
└── content/                         # Optional content bundles
    └── mastery/
```

### Build Script Template

```bash
#!/bin/bash
set -e

# Get version from VERSION file
VERSION=$(cat VERSION | tr -d 'v\n')

PACKAGE_NAME="{mcp-name}-container-tools-v${VERSION}"
PACKAGE_DIR="builds/container-tools/${PACKAGE_NAME}"

# Create structure
mkdir -p "${PACKAGE_DIR}/opt/{mcp-name}/bin/platforms"
mkdir -p "${PACKAGE_DIR}/etc"
mkdir -p "${PACKAGE_DIR}/bin"
mkdir -p "${PACKAGE_DIR}/internal"
mkdir -p "${PACKAGE_DIR}/templates/hooks"

# Build function with version embedding
build_binary() {
    local platform=$1
    local arch=$2
    local output_name=$3

    BUILD_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")

    LDFLAGS="-s -w"
    LDFLAGS="${LDFLAGS} -X 'main.Version=${VERSION}'"
    LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_DATE}'"
    LDFLAGS="${LDFLAGS} -X 'main.GitCommit=${COMMIT_ID}'"

    GOOS=${platform} GOARCH=${arch} go build \
        -ldflags="${LDFLAGS}" \
        -o "${PACKAGE_DIR}/opt/{mcp-name}/bin/platforms/${output_name}" \
        ./cmd/{mcp-name}
}

# Build all platforms
build_binary "darwin" "amd64" "{mcp-name}-v${VERSION}-darwin-amd64"
build_binary "darwin" "arm64" "{mcp-name}-v${VERSION}-darwin-arm64"
build_binary "linux" "amd64" "{mcp-name}-v${VERSION}-linux-amd64"
build_binary "linux" "arm64" "{mcp-name}-v${VERSION}-linux-arm64"
build_binary "windows" "amd64" "{mcp-name}-v${VERSION}-windows-amd64.exe"

# Create launcher (see template above)
# Create mcp.json
# Copy templates
# Create tarball

cd builds/container-tools
tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}"
```

## Version Embedding (Go)

In your Go code, define these variables for `ldflags` injection:

```go
// cmd/{mcp-name}/main.go
package main

var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

// Also in internal packages if needed
// internal/server/version.go
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)
```

Build command injects values via:
```
-X 'main.Version=${VERSION}'
-X 'main.BuildTime=${BUILD_DATE}'
-X 'main.GitCommit=${COMMIT_ID}'
```

## Container Usage

### Docker Run
```bash
docker run -it \
  -v /opt/container-tools:/opt/container-tools:ro \
  your-image \
  claude --mcp-config /opt/container-tools/etc/mcp.json
```

### Docker Compose
```yaml
services:
  claude-dev:
    image: your-image
    volumes:
      - /opt/container-tools:/opt/container-tools:ro
      - ~/.{mcp-name}:/root/.{mcp-name}  # Data persistence
    command: claude --mcp-config /opt/container-tools/etc/mcp.json
```

### DevContainer
```json
{
  "mounts": [
    "source=/opt/container-tools,target=/opt/container-tools,type=bind,readonly"
  ],
  "remoteEnv": {
    "CLAUDE_MCP_CONFIG": "/opt/container-tools/etc/mcp.json"
  }
}
```

## Installation Workflow

```bash
# Extract
tar -xzf {mcp-name}-container-tools-v{version}.tar.gz

# Backup existing (if any)
[ -d /opt/container-tools ] && sudo mv /opt/container-tools /opt/container-tools-prior

# Install
sudo mv {mcp-name}-container-tools-v{version} /opt/container-tools

# Verify
/opt/container-tools/opt/{mcp-name}/test-platforms.sh

# Test launcher
/opt/container-tools/opt/{mcp-name}/bin/{mcp-name} --version

# Rollback if needed
sudo mv /opt/container-tools /opt/container-tools-failed
sudo mv /opt/container-tools-prior /opt/container-tools
```

## Debug Mode

Enable launcher debugging:
```bash
{MCP_NAME}_DEBUG=1 /opt/container-tools/opt/{mcp-name}/bin/{mcp-name} --version
```

Output shows:
- Detected OS and architecture
- Container detection result
- Selected binary path

## Checklist for New MCP (Core Only)

1. **Repository Setup**
   - [ ] `VERSION` file at root
   - [ ] Go code in `cmd/{mcp-name}/`
   - [ ] Version variables defined for ldflags

2. **Build System**
   - [ ] `scripts/build-container-tools.sh` created
   - [ ] All 5 platform targets defined
   - [ ] Launcher script generation
   - [ ] MCP config generation

3. **Testing**
   - [ ] `test-platforms.sh` script
   - [ ] Debug mode in launcher
   - [ ] Container test command

4. **Documentation**
   - [ ] README in package
   - [ ] Installation instructions
   - [ ] Rollback procedure

**Note:** For init infrastructure (hooks, init commands, templates), see [PART 2](#part-2-optional-init-infrastructure).

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Version in binary filename | Allows multiple versions to coexist, clear identification |
| Universal launcher | Single entry point, abstracts platform selection |
| `/opt/container-tools` path | Standard location, works for both host and containers |
| Read-only container mount | Security best practice |
| Separate `platforms/` directory | Clean organization, launcher simplicity |
| tarball distribution | Universal, no special tools needed |

## Common Adaptations

### Adding Environment Variables
```bash
# In launcher, before exec
export {MCP_NAME}_DATA_DIR="${HOME}/.{mcp-name}"
```

### Multiple MCP Servers
```json
{
  "mcpServers": {
    "mcp-one": { "command": "/opt/container-tools/opt/mcp-one/bin/mcp-one" },
    "mcp-two": { "command": "/opt/container-tools/opt/mcp-two/bin/mcp-two" }
  }
}
```

### Custom Data Paths
The launcher can set environment variables before executing the binary to configure data directories per-environment.

---

# PART 2: Optional Init Infrastructure

This section covers **optional** components for MCPs that need:
- Project initialization commands
- Claude Code hooks
- Documentation/template deployment
- Internal tooling commands

**Skip this entire section if your MCP just needs the binary distribution.**

## Extended Package Structure

If your MCP has init infrastructure, add these directories:

```
{mcp-name}-container-tools-v{version}/
├── etc/
│   └── mcp.json                      # (Core) MCP config
├── opt/{mcp-name}/                   # (Core) Binary distribution
│   └── ...
│
│ # === OPTIONAL ADDITIONS BELOW ===
│
├── bin/                              # User-facing init commands
│   └── {tool}-init                   # Initialize project with best practices
├── internal/                         # Internal tooling commands
│   ├── {prefix}                      # Main command menu (shows available tools)
│   ├── {prefix}-discover             # Browse available add-ons
│   ├── {prefix}-install              # Install add-ons
│   ├── {prefix}-status               # Show system status
│   └── {prefix}-init-*               # Domain-specific init commands
├── templates/
│   ├── hooks/                        # Claude Code hook templates
│   │   ├── app-start-hook.sh
│   │   ├── compact-start-hook.sh
│   │   ├── compact-end-hook.sh
│   │   └── init-hook.sh
│   ├── learning/                     # Optional documentation bundles
│   └── VERSION_MANIFEST.txt          # Build metadata
├── content/                          # Deployable content bundles
│   └── mastery/                      # Documentation for init deployment
└── .claude/
    └── commands/                     # Slash command templates
        └── *.md
```

## Init Command Pattern

An init command sets up project-specific configuration. Example structure:

```bash
#!/bin/bash
# {tool}-init - Initialize {mcp-name} best practices

set -e

INIT_VERSION="v{version}"
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

# Check for existing installation
if [ -d ".{mcp-name}" ]; then
    echo "Already initialized. Use --force to reinitialize."
    exit 0
fi

# Create project structure
mkdir -p .{mcp-name}/best-practices
mkdir -p .{mcp-name}/hooks

# Record version
echo "$INIT_VERSION" > .{mcp-name}/.init-version

# Copy templates from container-tools
if [ -d "/opt/container-tools/templates/hooks" ]; then
    cp /opt/container-tools/templates/hooks/*.sh .{mcp-name}/hooks/
    chmod +x .{mcp-name}/hooks/*.sh
fi

# Create best practices documentation
cat > .{mcp-name}/README.md << 'EOF'
# {MCP Name} Best Practices
...
EOF

echo "✅ {mcp-name} initialized!"
```

## Internal Commands Pattern

Internal commands provide a suite of management tools:

```bash
# internal/{prefix} - Main menu
#!/bin/bash
echo "{MCP Name} Container Tools v${VERSION}"
echo ""
echo "Available commands:"
echo "  {prefix}-discover     Browse available add-ons"
echo "  {prefix}-install      Install add-ons"
echo "  {prefix}-status       Show system status"
echo ""
echo "Run any command with --help for details"
```

## Hook Templates

Claude Code hooks fire at specific lifecycle events. Place templates in `templates/hooks/`:

### app-start-hook.sh
```bash
#!/bin/bash
# Fires when Claude Code starts
echo "{mcp-name} integration active"
# Could: check for updates, initialize state, etc.
```

### compact-start-hook.sh
```bash
#!/bin/bash
# Fires before context compaction
echo "Context compaction starting - save state"
# Could: trigger context save, checkpoint creation
```

### compact-end-hook.sh
```bash
#!/bin/bash
# Fires after context compaction
echo "Context compaction complete - restore state"
# Could: trigger context restoration
```

## MCP Config with Hooks

When hooks are included, extend `etc/mcp.json`:

```json
{
  "mcpServers": {
    "{mcp-name}": {
      "command": "/opt/container-tools/opt/{mcp-name}/bin/{mcp-name}",
      "args": ["--mode", "stdio"]
    }
  },
  "hooks": {
    "app-start": "/opt/container-tools/templates/hooks/app-start-hook.sh",
    "compact-start": "/opt/container-tools/templates/hooks/compact-start-hook.sh",
    "compact-end": "/opt/container-tools/templates/hooks/compact-end-hook.sh"
  },
  "commands": {
    "{prefix}": "/opt/container-tools/internal/{prefix}",
    "{prefix}-discover": "/opt/container-tools/internal/{prefix}-discover",
    "{prefix}-status": "/opt/container-tools/internal/{prefix}-status"
  }
}
```

## Adding Internal Tools to PATH

The launcher can auto-add internal tools to PATH:

```bash
# Add to launcher script, before exec
add_internal_tools_to_path() {
    local CONTAINER_TOOLS_ROOT="$(cd "${BIN_DIR}/../../.." && pwd)"
    local INTERNAL_DIR="${CONTAINER_TOOLS_ROOT}/internal"

    if [ -d "${INTERNAL_DIR}" ]; then
        export PATH="${INTERNAL_DIR}:${PATH}"
    fi
}
```

## Build Script Additions for Init Infrastructure

Add these sections to your build script:

```bash
# After building binaries...

# === OPTIONAL: Init Infrastructure ===

# Create directories for init components
mkdir -p "${PACKAGE_DIR}/bin"
mkdir -p "${PACKAGE_DIR}/internal"
mkdir -p "${PACKAGE_DIR}/templates/hooks"
mkdir -p "${PACKAGE_DIR}/content"

# Copy init command
if [ -f "container-tools-templates/bin/{tool}-init" ]; then
    cp "container-tools-templates/bin/{tool}-init" "${PACKAGE_DIR}/bin/"
    chmod +x "${PACKAGE_DIR}/bin/{tool}-init"
fi

# Copy hook templates
if [ -d "container-tools-templates/hooks" ]; then
    cp container-tools-templates/hooks/*.sh "${PACKAGE_DIR}/templates/hooks/"
    chmod +x "${PACKAGE_DIR}/templates/hooks"/*.sh
fi

# Copy internal tools
for tool in {prefix} {prefix}-discover {prefix}-install {prefix}-status; do
    if [ -f "container-tools-templates/internal/${tool}" ]; then
        cp "container-tools-templates/internal/${tool}" "${PACKAGE_DIR}/internal/"
        chmod +x "${PACKAGE_DIR}/internal/${tool}"
    fi
done

# Copy content bundles
if [ -d "content/mastery" ]; then
    mkdir -p "${PACKAGE_DIR}/content/mastery"
    cp -r content/mastery/* "${PACKAGE_DIR}/content/mastery/"
fi

# Create extended mcp.json with hooks
cat > "${PACKAGE_DIR}/etc/mcp.json" << CONFIG
{
  "mcpServers": {
    "{mcp-name}": {
      "command": "/opt/container-tools/opt/{mcp-name}/bin/{mcp-name}",
      "args": ["--mode", "stdio"]
    }
  },
  "hooks": {
    "app-start": "/opt/container-tools/templates/hooks/app-start-hook.sh"
  }
}
CONFIG
```

## Source Template Directory

For init infrastructure, create this source structure:

```
your-mcp-repo/
├── container-tools-templates/
│   ├── bin/
│   │   └── {tool}-init              # Project init command
│   ├── hooks/
│   │   ├── app-start-hook.sh
│   │   ├── compact-start-hook.sh
│   │   └── compact-end-hook.sh
│   └── internal/
│       ├── {prefix}
│       ├── {prefix}-discover
│       └── {prefix}-status
└── content/
    └── mastery/                     # Documentation bundles
        └── *.md
```

## Checklist for Init Infrastructure (Optional)

Only if your MCP needs project initialization:

1. **Source Templates**
   - [ ] `container-tools-templates/` directory created
   - [ ] Init command in `bin/`
   - [ ] Hook templates in `hooks/`
   - [ ] Internal tools in `internal/` (if needed)

2. **Build Script Updates**
   - [ ] Copy init command to `bin/`
   - [ ] Copy hooks to `templates/hooks/`
   - [ ] Copy internal tools to `internal/`
   - [ ] Extended `mcp.json` with hooks config

3. **Testing**
   - [ ] Init command creates expected structure
   - [ ] Hooks execute without errors
   - [ ] Internal tools are accessible

4. **Documentation**
   - [ ] Init command usage in README
   - [ ] Hook behavior documented
   - [ ] Upgrade/migration path documented
