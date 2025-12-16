#!/bin/bash
#
# build-container-tools.sh - Build the container-tools distribution package
#
# Creates a tarball that can be installed alongside other MCPs in /opt/container-tools/
# Follows the Container Tools MCP Integration Guide specification.
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Get version from VERSION file
VERSION=$(cat "${REPO_ROOT}/VERSION" | tr -d '\n')
if [ -z "$VERSION" ]; then
    echo "ERROR: VERSION file is empty or missing"
    exit 1
fi

MCP_NAME="image-tools-mcp"
PACKAGE_NAME="container-tools-${MCP_NAME}-v${VERSION}"
BUILD_DIR="${REPO_ROOT}/builds/container-tools"
PACKAGE_DIR="${BUILD_DIR}/${PACKAGE_NAME}"

# Build metadata
BUILD_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "=============================================="
echo "Building ${MCP_NAME} Container Tools Package"
echo "=============================================="
echo "Version:    ${VERSION}"
echo "Commit:     ${GIT_COMMIT}"
echo "Build Date: ${BUILD_DATE}"
echo ""

# Ensure tessdata is available for embedding (Linux builds only)
echo "Checking tessdata for embedding..."
"${SCRIPT_DIR}/ensure-tessdata.sh"
echo ""

# Clean and create build directory
# Structure per Container Tools Integration Guide:
#   package-name/
#   ├── image-tools-mcp/        # MCP's territory
#   │   ├── bin/
#   │   │   ├── image-tools-mcp (launcher)
#   │   │   └── platforms/
#   │   ├── install.sh
#   │   └── README.md
#   ├── etc/
#   │   ├── hooks-dispatcher.sh
#   │   └── hooks.d/
#   │       └── app-start/
#   │           └── image-tools-mcp.sh
#   ├── install.sh              # Top-level installer
#   └── VERSION_MANIFEST.txt
rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}/${MCP_NAME}/bin/platforms"
mkdir -p "${PACKAGE_DIR}/etc/hooks.d/app-start"

# Build function for non-CGO platforms (macOS, Windows)
build_binary_nocgo() {
    local os=$1
    local arch=$2
    local suffix=$3
    local output_name="${MCP_NAME}-v${VERSION}-${os}-${arch}${suffix}"

    echo "Building ${output_name} (no CGO - CLI fallback for OCR)..."

    LDFLAGS="-s -w"
    LDFLAGS="${LDFLAGS} -X 'main.Version=${VERSION}'"
    LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_DATE}'"
    LDFLAGS="${LDFLAGS} -X 'main.GitCommit=${GIT_COMMIT}'"

    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build \
        -ldflags="${LDFLAGS}" \
        -o "${PACKAGE_DIR}/${MCP_NAME}/bin/platforms/${output_name}" \
        "${REPO_ROOT}/cmd/image-mcp"
}

# Build function for Linux with CGO (native OCR via gosseract)
build_binary_cgo() {
    local os=$1
    local arch=$2
    local suffix=$3
    local output_name="${MCP_NAME}-v${VERSION}-${os}-${arch}${suffix}"

    echo "Building ${output_name} (CGO enabled - native OCR)..."

    LDFLAGS="-s -w"
    LDFLAGS="${LDFLAGS} -X 'main.Version=${VERSION}'"
    LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_DATE}'"
    LDFLAGS="${LDFLAGS} -X 'main.GitCommit=${GIT_COMMIT}'"

    CGO_ENABLED=1 GOOS=${os} GOARCH=${arch} go build \
        -ldflags="${LDFLAGS}" \
        -o "${PACKAGE_DIR}/${MCP_NAME}/bin/platforms/${output_name}" \
        "${REPO_ROOT}/cmd/image-mcp"
}

# Build all platforms
echo "Building platform binaries..."

# macOS and Windows: no CGO, uses tesseract CLI fallback
build_binary_nocgo "darwin" "amd64" ""
build_binary_nocgo "darwin" "arm64" ""
build_binary_nocgo "windows" "amd64" ".exe"

# Linux: CGO enabled for native gosseract OCR (no external dependencies needed)
#
# CI builds: Both linux-amd64 and linux-arm64 are built natively with CGO on
#            their respective GitHub Actions runners (ubuntu-latest for amd64,
#            ubuntu-24.04-arm for arm64). Both have embedded OCR.
#
# Local builds: Native arch gets CGO, cross-compiled arch falls back to CLI.
#               This is fine for development; production releases use CI.
if [ "$(uname -s)" = "Linux" ]; then
    HOST_ARCH=$(uname -m)
    case "$HOST_ARCH" in
        x86_64|amd64)
            build_binary_cgo "linux" "amd64" ""
            # Cross-compile arm64 without CGO (CI builds this natively with CGO)
            echo "Note: linux-arm64 built without CGO locally; CI builds with CGO"
            build_binary_nocgo "linux" "arm64" ""
            ;;
        aarch64|arm64)
            build_binary_cgo "linux" "arm64" ""
            # Cross-compile amd64 without CGO (CI builds this natively with CGO)
            echo "Note: linux-amd64 built without CGO locally; CI builds with CGO"
            build_binary_nocgo "linux" "amd64" ""
            ;;
        *)
            # Unknown arch, build both without CGO
            build_binary_nocgo "linux" "amd64" ""
            build_binary_nocgo "linux" "arm64" ""
            ;;
    esac
else
    # Not on Linux, can't use CGO for Linux builds
    echo "Note: Not building on Linux, Linux binaries will use CLI fallback for OCR"
    build_binary_nocgo "linux" "amd64" ""
    build_binary_nocgo "linux" "arm64" ""
fi

echo ""
echo "Creating universal launcher..."

# Create universal launcher script
cat > "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}" << 'LAUNCHER'
#!/usr/bin/env bash
#
# Universal launcher for image-tools-mcp
# Auto-detects OS, architecture, and container environment
#
set -e

LAUNCHER_VERSION="__VERSION__"
BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORMS_DIR="${BIN_DIR}/platforms"
MCP_NAME="image-tools-mcp"

# Debug mode
debug() {
    if [ -n "${IMAGE_TOOLS_MCP_DEBUG}" ]; then
        echo "[DEBUG] $*" >&2
    fi
}

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

# Binary selection
select_binary() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    local suffix=""

    # Always use Linux binary in containers
    if is_container; then
        debug "Container detected, using Linux binary"
        os="linux"
    fi

    if [ "$os" = "windows" ]; then
        suffix=".exe"
    fi

    local binary="${PLATFORMS_DIR}/${MCP_NAME}-v${LAUNCHER_VERSION}-${os}-${arch}${suffix}"

    debug "OS: ${os}, Arch: ${arch}, Container: $(is_container && echo yes || echo no)"
    debug "Selected binary: ${binary}"

    if [ ! -f "$binary" ]; then
        echo "ERROR: Binary not found: ${binary}" >&2
        echo "Available binaries:" >&2
        ls -1 "${PLATFORMS_DIR}/" >&2
        exit 1
    fi

    echo "$binary"
}

# Execute
exec "$(select_binary)" "$@"
LAUNCHER

# Replace version placeholder
sed -i "s/__VERSION__/${VERSION}/g" "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}"
chmod +x "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}"

echo "Creating hooks dispatcher..."

# Create hooks-dispatcher.sh (per integration guide)
cat > "${PACKAGE_DIR}/etc/hooks-dispatcher.sh" << 'DISPATCHER'
#!/bin/bash
#
# Container Tools Hook Dispatcher
# Runs all hook scripts for a given hook type
#
# Usage: hooks-dispatcher.sh <hook-type>
# Example: hooks-dispatcher.sh app-start
#

set -e

HOOK_TYPE="$1"
HOOKS_DIR="/opt/container-tools/etc/hooks.d/${HOOK_TYPE}"

if [ -z "$HOOK_TYPE" ]; then
    echo "Usage: $0 <hook-type>" >&2
    exit 1
fi

if [ ! -d "$HOOKS_DIR" ]; then
    # No hooks registered for this type - that's okay
    exit 0
fi

# Run all executable scripts in alphabetical order
for script in "$HOOKS_DIR"/*.sh; do
    if [ -f "$script" ] && [ -x "$script" ]; then
        if [ -n "$CONTAINER_TOOLS_DEBUG" ]; then
            echo "[hooks-dispatcher] Running: $script" >&2
        fi

        # Run hook, capture errors but don't stop other hooks
        if ! "$script"; then
            echo "[hooks-dispatcher] Warning: $script failed" >&2
        fi
    fi
done

exit 0
DISPATCHER
chmod +x "${PACKAGE_DIR}/etc/hooks-dispatcher.sh"

echo "Creating app-start hook..."

# Create image-tools-mcp app-start hook
cat > "${PACKAGE_DIR}/etc/hooks.d/app-start/${MCP_NAME}.sh" << 'HOOK'
#!/bin/bash
#
# image-tools-mcp app-start hook
# Called when Claude Code starts
#
# This hook can be used for:
# - Verifying dependencies are available
# - Initializing cache directories
# - Logging startup events
#

# Currently a placeholder - image-tools-mcp doesn't require initialization
# but having the hook in place follows the container-tools pattern

if [ -n "$CONTAINER_TOOLS_DEBUG" ]; then
    echo "[image-tools-mcp] App start hook executed" >&2
fi

exit 0
HOOK
chmod +x "${PACKAGE_DIR}/etc/hooks.d/app-start/${MCP_NAME}.sh"

echo "Creating test script..."

# Create test-platforms.sh
cat > "${PACKAGE_DIR}/${MCP_NAME}/test-platforms.sh" << 'TESTSCRIPT'
#!/bin/bash
#
# Test that all platform binaries are valid executables
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORMS_DIR="${SCRIPT_DIR}/bin/platforms"

echo "Testing platform binaries..."
echo ""

for binary in "${PLATFORMS_DIR}"/*; do
    name=$(basename "$binary")
    # Skip tessdata directory
    if [ -d "$binary" ]; then
        continue
    fi
    printf "  %-45s " "$name"

    if [ -x "$binary" ] || [[ "$name" == *.exe ]]; then
        # Check if it's a valid executable (file magic)
        file_type=$(file -b "$binary" 2>/dev/null || echo "unknown")
        if echo "$file_type" | grep -qiE 'executable|mach-o|pe32|elf'; then
            echo "OK (${file_type:0:30}...)"
        else
            echo "WARN: ${file_type:0:40}"
        fi
    else
        echo "FAIL: not executable"
    fi
done

echo ""
echo "Testing launcher..."
"${SCRIPT_DIR}/bin/image-tools-mcp" --version && echo "Launcher: OK" || echo "Launcher: FAIL"
TESTSCRIPT

chmod +x "${PACKAGE_DIR}/${MCP_NAME}/test-platforms.sh"

echo "Creating README..."

# Create README (with correct paths per integration guide)
cat > "${PACKAGE_DIR}/${MCP_NAME}/README.md" << README
# Image Tools MCP v${VERSION}

MCP server providing image analysis tools for Claude.

## Features

- Image loading and metadata extraction
- Color sampling (single point and multi-point)
- Dominant color extraction
- Image cropping (coordinates and named regions)
- Distance measurement
- Grid overlay generation
- Edge detection
- Shape detection (rectangles, circles, lines)
- Text region detection
- OCR (built-in on Linux; macOS/Windows require Tesseract CLI)

## Installation

Run the installer from the extracted package:

\`\`\`bash
./install.sh
\`\`\`

Or specify a custom target:

\`\`\`bash
./install.sh --target /custom/path
\`\`\`

## Usage

After installation, the universal launcher is available at:

\`\`\`bash
/opt/container-tools/image-tools-mcp/bin/image-tools-mcp --version
\`\`\`

Or via the symlink:

\`\`\`bash
/opt/container-tools/bin/image-tools-mcp --version
\`\`\`

## MCP Configuration

The installer automatically configures \`/opt/container-tools/etc/mcp.json\`.

Manual configuration (if needed):

\`\`\`json
{
  "mcpServers": {
    "image-tools-mcp": {
      "command": "/opt/container-tools/image-tools-mcp/bin/image-tools-mcp",
      "args": []
    }
  }
}
\`\`\`

## Uninstall / Rollback

To uninstall or rollback to a prior version:

\`\`\`bash
./install.sh --uninstall
\`\`\`

If a prior installation exists, it will be restored. Otherwise, the MCP is fully removed.

## Build Info

- Version: ${VERSION}
- Git Commit: ${GIT_COMMIT}
- Build Date: ${BUILD_DATE}
README

echo "Creating install script..."

# Create install.sh (following Container Tools Integration Guide template)
cat > "${PACKAGE_DIR}/install.sh" << 'INSTALL'
#!/bin/bash
#
# image-tools-mcp installer for container-tools
#
# Usage:
#   ./install.sh [OPTIONS] [target-dir]
#
# Options:
#   --target DIR    Install to DIR (default: /opt/container-tools)
#   --uninstall     Remove/rollback image-tools-mcp from container-tools
#   --help          Show this help
#
# Default target: /opt/container-tools
#

set -e

YOUR_MCP="image-tools-mcp"
VERSION="__VERSION__"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Parse arguments
UNINSTALL=false
TARGET="/opt/container-tools"

while [[ $# -gt 0 ]]; do
    case $1 in
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        --target)
            TARGET="$2"
            shift 2
            ;;
        --help|-h)
            head -20 "$0" | tail -15
            exit 0
            ;;
        *)
            TARGET="$1"
            shift
            ;;
    esac
done

# Check for sudo if needed
need_sudo() {
    if [ -w "$TARGET" ] 2>/dev/null || [ -w "$(dirname "$TARGET")" ] 2>/dev/null; then
        echo ""
    else
        echo "sudo"
    fi
}
SUDO=$(need_sudo)

# Get script directory (where the package was extracted)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

#
# PLATFORM DETECTION
#
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) arch="unknown" ;;
    esac
    echo "${os}-${arch}"
}

#
# UNINSTALL / ROLLBACK
#
if [ "$UNINSTALL" = true ]; then
    info "Uninstalling $YOUR_MCP from $TARGET..."

    # Check if we have a prior installation to roll back to
    if [ -d "$TARGET/$YOUR_MCP-prior" ]; then
        info "Prior installation found - performing rollback..."

        # 1. Remove current installation
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # 2. Restore prior installation
        $SUDO mv "$TARGET/$YOUR_MCP-prior" "$TARGET/$YOUR_MCP"
        info "Restored prior installation"

        # 3. Rollback mcp.json entry (merge prior entry into current mcp.json)
        PRIOR_MCP_JSON="$TARGET/$YOUR_MCP/backup/mcp.json-prior"
        CURRENT_MCP_JSON="$TARGET/etc/mcp.json"

        if [ -f "$PRIOR_MCP_JSON" ] && command -v jq &> /dev/null; then
            # Extract our entry from the prior mcp.json
            PRIOR_ENTRY=$($SUDO cat "$PRIOR_MCP_JSON" | jq ".mcpServers[\"$YOUR_MCP\"]")

            if [ "$PRIOR_ENTRY" != "null" ]; then
                # Merge prior entry into current mcp.json (preserves other MCPs)
                $SUDO jq --argjson entry "$PRIOR_ENTRY" \
                   ".mcpServers[\"$YOUR_MCP\"] = \$entry" \
                   "$CURRENT_MCP_JSON" > "/tmp/mcp.json.tmp"
                $SUDO mv "/tmp/mcp.json.tmp" "$CURRENT_MCP_JSON"
                info "Rolled back mcp.json entry to prior version"
            fi
        else
            warn "Could not rollback mcp.json entry (jq not found or no prior backup)"
        fi

        # 4. Update symlink to point to restored version
        case "$OSTYPE" in
            msys*|cygwin*|win32*) ;;
            *)
                $SUDO rm -f "$TARGET/bin/$YOUR_MCP"
                $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
                ;;
        esac

        info "Rollback complete - restored prior version"
    else
        info "No prior installation - performing full removal..."

        # Remove our directory
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # Remove our symlink
        $SUDO rm -f "$TARGET/bin/$YOUR_MCP"

        # Remove our hooks
        $SUDO find "$TARGET/etc/hooks.d" -name "$YOUR_MCP.sh" -delete 2>/dev/null || true

        # Remove our entry from mcp.json
        if command -v jq &> /dev/null && [ -f "$TARGET/etc/mcp.json" ]; then
            $SUDO jq "del(.mcpServers[\"$YOUR_MCP\"])" \
               "$TARGET/etc/mcp.json" > "/tmp/mcp.json.tmp"
            $SUDO mv "/tmp/mcp.json.tmp" "$TARGET/etc/mcp.json"
            info "Removed $YOUR_MCP from mcp.json"
        else
            warn "Please manually remove '$YOUR_MCP' from $TARGET/etc/mcp.json"
        fi

        info "Uninstall complete"
    fi
    exit 0
fi

#
# INSTALL
#

# Check if already up to date (skip-if-identical optimization)
PLATFORM=$(detect_platform)
SOURCE_BIN=$(find "$SCRIPT_DIR/$YOUR_MCP/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)
DEST_BIN=$(find "$TARGET/$YOUR_MCP/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)

if [ -n "$SOURCE_BIN" ] && [ -n "$DEST_BIN" ]; then
    SOURCE_MD5=$(md5sum "$SOURCE_BIN" 2>/dev/null | awk '{print $1}')
    DEST_MD5=$(md5sum "$DEST_BIN" 2>/dev/null | awk '{print $1}')
    if [ -n "$SOURCE_MD5" ] && [ "$SOURCE_MD5" = "$DEST_MD5" ]; then
        info "Already up to date (${PLATFORM} binary unchanged)"
        exit 0
    fi
fi

echo ""
echo "========================================="
echo -e "${BLUE}Installing $YOUR_MCP v${VERSION}${NC}"
echo "========================================="
echo ""
info "Target: $TARGET"

# 1. Create directory structure if first-time install
if [ ! -d "$TARGET" ]; then
    info "Creating container-tools directory structure..."
    $SUDO mkdir -p "$TARGET/bin"
    $SUDO mkdir -p "$TARGET/etc/hooks.d/app-start"
    $SUDO mkdir -p "$TARGET/etc/hooks.d/compact-start"
    $SUDO mkdir -p "$TARGET/etc/hooks.d/compact-end"
fi

# Ensure subdirectories exist
$SUDO mkdir -p "$TARGET/bin"
$SUDO mkdir -p "$TARGET/etc/hooks.d/app-start"

# 2. Backup existing mcp.json to our territory
if [ -f "$TARGET/etc/mcp.json" ]; then
    $SUDO mkdir -p "$TARGET/$YOUR_MCP/backup"
    $SUDO cp "$TARGET/etc/mcp.json" "$TARGET/$YOUR_MCP/backup/mcp.json-prior"
    info "Backed up mcp.json to $YOUR_MCP/backup/"
fi

# 3. Backup existing installation with -prior suffix (depth of 1)
if [ -d "$TARGET/$YOUR_MCP" ]; then
    info "Backing up previous installation..."
    $SUDO rm -rf "$TARGET/$YOUR_MCP-prior"
    $SUDO mv "$TARGET/$YOUR_MCP" "$TARGET/$YOUR_MCP-prior"
fi

# 4. Install MCP directory
info "Installing $YOUR_MCP..."
$SUDO cp -r "$SCRIPT_DIR/$YOUR_MCP" "$TARGET/"

# Ensure binaries are executable
$SUDO chmod +x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP"
$SUDO chmod +x "$TARGET/$YOUR_MCP/bin/platforms"/* 2>/dev/null || true
$SUDO chmod +x "$TARGET/$YOUR_MCP/test-platforms.sh" 2>/dev/null || true

# 5. Install hooks dispatcher if missing
if [ ! -f "$TARGET/etc/hooks-dispatcher.sh" ]; then
    info "Installing hooks dispatcher..."
    $SUDO cp "$SCRIPT_DIR/etc/hooks-dispatcher.sh" "$TARGET/etc/"
    $SUDO chmod +x "$TARGET/etc/hooks-dispatcher.sh"
fi

# 6. Install our hooks
info "Installing hooks..."
if [ -f "$SCRIPT_DIR/etc/hooks.d/app-start/$YOUR_MCP.sh" ]; then
    $SUDO cp "$SCRIPT_DIR/etc/hooks.d/app-start/$YOUR_MCP.sh" "$TARGET/etc/hooks.d/app-start/"
    $SUDO chmod +x "$TARGET/etc/hooks.d/app-start/$YOUR_MCP.sh"
fi

# 7. Create symlink (Linux/macOS only)
case "$OSTYPE" in
    msys*|cygwin*|win32*)
        warn "Windows detected - skipping symlink"
        warn "Add $TARGET/$YOUR_MCP/bin to your PATH"
        ;;
    *)
        $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
        info "Created symlink: $TARGET/bin/$YOUR_MCP"
        ;;
esac

# 8. Update mcp.json
MCP_JSON="$TARGET/etc/mcp.json"
YOUR_COMMAND="$TARGET/$YOUR_MCP/bin/$YOUR_MCP"
DISPATCHER="$TARGET/etc/hooks-dispatcher.sh"

if [ ! -f "$MCP_JSON" ]; then
    info "Creating mcp.json..."
    $SUDO tee "$MCP_JSON" > /dev/null << MCPEOF
{
  "mcpServers": {
    "$YOUR_MCP": {
      "command": "$YOUR_COMMAND",
      "args": []
    }
  },
  "hooks": {
    "app-start": "$DISPATCHER app-start",
    "compact-start": "$DISPATCHER compact-start",
    "compact-end": "$DISPATCHER compact-end"
  }
}
MCPEOF
elif command -v jq &> /dev/null; then
    info "Merging into mcp.json..."
    $SUDO jq --arg name "$YOUR_MCP" \
       --arg cmd "$YOUR_COMMAND" \
       --arg dispatcher "$DISPATCHER" \
       '.mcpServers[$name] = {"command": $cmd, "args": []} |
        .hooks["app-start"] = "\($dispatcher) app-start" |
        .hooks["compact-start"] = "\($dispatcher) compact-start" |
        .hooks["compact-end"] = "\($dispatcher) compact-end"' \
       "$MCP_JSON" > "/tmp/mcp.json.tmp"
    $SUDO mv "/tmp/mcp.json.tmp" "$MCP_JSON"
else
    warn "jq not found - please manually configure mcp.json"
    warn "Add $YOUR_MCP entry pointing to: $YOUR_COMMAND"
fi

# 9. Verify installation
echo ""
info "Verifying installation..."
if [ -x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP" ]; then
    VERSION_OUTPUT=$("$TARGET/$YOUR_MCP/bin/$YOUR_MCP" --version 2>&1 | head -1)
    info "Installed: $VERSION_OUTPUT"
else
    error "Installation verification failed - launcher not executable"
fi

# Summary
echo ""
echo "========================================="
echo -e "${GREEN}Installation complete!${NC}"
echo "========================================="
echo ""
echo "Installed to: $TARGET/$YOUR_MCP/"
echo ""
echo -e "${BLUE}Next steps:${NC}"
case "$OSTYPE" in
    msys*|cygwin*|win32*)
        echo "  1. Add $TARGET/$YOUR_MCP/bin to your PATH"
        ;;
    *)
        echo "  1. Add $TARGET/bin to your PATH (if not already)"
        ;;
esac
echo "  2. Restart Claude Code to load the new MCP"
echo ""
echo -e "${BLUE}Test:${NC}"
echo "  $TARGET/$YOUR_MCP/bin/$YOUR_MCP --version"
echo ""
echo -e "${BLUE}Rollback (if needed):${NC}"
echo "  ./install.sh --uninstall"
echo ""
INSTALL

# Replace version placeholder in install.sh
sed -i "s/__VERSION__/${VERSION}/g" "${PACKAGE_DIR}/install.sh"
chmod +x "${PACKAGE_DIR}/install.sh"

echo "Creating VERSION_MANIFEST.txt..."

# Create VERSION_MANIFEST.txt
cat > "${PACKAGE_DIR}/VERSION_MANIFEST.txt" << MANIFEST
Package: ${MCP_NAME}
Version: ${VERSION}
Git Commit: ${GIT_COMMIT}
Build Date: ${BUILD_DATE}
Build Host: $(hostname 2>/dev/null || echo "unknown")

Directory Structure:
  ${MCP_NAME}/           - MCP installation directory
  ${MCP_NAME}/bin/       - Universal launcher and platform binaries
  etc/                   - Shared configuration (hooks dispatcher)

Platforms:
  - darwin-amd64
  - darwin-arm64
  - linux-amd64
  - linux-arm64
  - windows-amd64

Container Tools Integration Guide Compliant: Yes
MANIFEST

echo ""
echo "Creating tarball..."

# Create tarball
cd "${BUILD_DIR}"
tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}"

echo ""
echo "=============================================="
echo "Build complete!"
echo "=============================================="
echo ""
echo "Package: ${BUILD_DIR}/${PACKAGE_NAME}.tar.gz"
echo ""
echo "Contents:"
find "${PACKAGE_NAME}" -type f | sed 's/^/  /'
echo ""
echo "To install:"
echo "  tar -xzf ${PACKAGE_NAME}.tar.gz"
echo "  cd ${PACKAGE_NAME}"
echo "  ./install.sh"
echo ""
