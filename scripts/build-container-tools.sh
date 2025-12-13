#!/bin/bash
#
# build-container-tools.sh - Build the container-tools distribution package
#
# Creates a tarball that can be installed alongside other MCPs in /opt/container-tools/
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
PACKAGE_NAME="${MCP_NAME}-v${VERSION}"
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
rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/platforms"

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
        -o "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/platforms/${output_name}" \
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
        -o "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/platforms/${output_name}" \
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
cat > "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/${MCP_NAME}" << 'LAUNCHER'
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
sed -i "s/__VERSION__/${VERSION}/g" "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/${MCP_NAME}"
chmod +x "${PACKAGE_DIR}/opt/${MCP_NAME}/bin/${MCP_NAME}"

echo "Creating test script..."

# Create test-platforms.sh
cat > "${PACKAGE_DIR}/opt/${MCP_NAME}/test-platforms.sh" << 'TESTSCRIPT'
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

chmod +x "${PACKAGE_DIR}/opt/${MCP_NAME}/test-platforms.sh"

echo "Creating README..."

# Create README
cat > "${PACKAGE_DIR}/opt/${MCP_NAME}/README.md" << README
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

## Usage

The universal launcher automatically selects the correct binary:

\`\`\`bash
/opt/container-tools/opt/image-tools-mcp/bin/image-tools-mcp --version
\`\`\`

## MCP Configuration

Add to your Claude MCP config:

\`\`\`json
{
  "mcpServers": {
    "image-tools-mcp": {
      "command": "/opt/container-tools/opt/image-tools-mcp/bin/image-tools-mcp",
      "args": []
    }
  }
}
\`\`\`

## Build Info

- Version: ${VERSION}
- Git Commit: ${GIT_COMMIT}
- Build Date: ${BUILD_DATE}
README

echo "Creating install script..."

# Create install.sh
cat > "${PACKAGE_DIR}/install.sh" << 'INSTALL'
#!/bin/bash
#
# Install image-tools-mcp into /opt/container-tools/
# Merges with existing MCP installations
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MCP_NAME="image-tools-mcp"
INSTALL_ROOT="/opt/container-tools"
MCP_JSON="${INSTALL_ROOT}/etc/mcp.json"

echo "Installing ${MCP_NAME}..."
echo ""

# Check for root/sudo
if [ "$EUID" -ne 0 ] && [ ! -w "$INSTALL_ROOT" ]; then
    echo "This script requires sudo to install to ${INSTALL_ROOT}"
    echo "Re-running with sudo..."
    exec sudo "$0" "$@"
fi

# Create base directory if it doesn't exist
if [ ! -d "$INSTALL_ROOT" ]; then
    echo "Creating ${INSTALL_ROOT}..."
    mkdir -p "${INSTALL_ROOT}/opt"
    mkdir -p "${INSTALL_ROOT}/etc"
fi

# Backup existing installation if present
if [ -d "${INSTALL_ROOT}/opt/${MCP_NAME}" ]; then
    echo "Backing up existing ${MCP_NAME}..."
    mv "${INSTALL_ROOT}/opt/${MCP_NAME}" "${INSTALL_ROOT}/opt/${MCP_NAME}.backup.$(date +%Y%m%d%H%M%S)"
fi

# Copy our files
echo "Copying ${MCP_NAME} files..."
cp -r "${SCRIPT_DIR}/opt/${MCP_NAME}" "${INSTALL_ROOT}/opt/"

# Make binaries executable
chmod +x "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}"
chmod +x "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/platforms"/*
chmod +x "${INSTALL_ROOT}/opt/${MCP_NAME}/test-platforms.sh"

# Merge into mcp.json
echo "Updating MCP configuration..."

if [ -f "$MCP_JSON" ]; then
    # Check if jq is available
    if command -v jq &> /dev/null; then
        # Merge our entry into existing config
        TEMP_JSON=$(mktemp)
        jq --arg cmd "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}" \
           '.mcpServers["image-tools-mcp"] = {"command": $cmd, "args": []}' \
           "$MCP_JSON" > "$TEMP_JSON"
        mv "$TEMP_JSON" "$MCP_JSON"
        echo "  Merged into existing mcp.json"
    else
        echo ""
        echo "WARNING: jq not installed. Please manually add to ${MCP_JSON}:"
        echo ""
        echo '  "image-tools-mcp": {'
        echo "    \"command\": \"${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}\","
        echo '    "args": []'
        echo '  }'
        echo ""
    fi
else
    # Create new mcp.json
    cat > "$MCP_JSON" << MCPJSON
{
  "mcpServers": {
    "image-tools-mcp": {
      "command": "${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME}",
      "args": []
    }
  }
}
MCPJSON
    echo "  Created new mcp.json"
fi

echo ""
echo "Installation complete!"
echo ""
echo "Verify with:"
echo "  ${INSTALL_ROOT}/opt/${MCP_NAME}/test-platforms.sh"
echo ""
echo "Test the binary:"
echo "  ${INSTALL_ROOT}/opt/${MCP_NAME}/bin/${MCP_NAME} --version"
INSTALL

chmod +x "${PACKAGE_DIR}/install.sh"

echo "Creating VERSION_MANIFEST.txt..."

# Create VERSION_MANIFEST.txt
cat > "${PACKAGE_DIR}/VERSION_MANIFEST.txt" << MANIFEST
Package: ${MCP_NAME}
Version: ${VERSION}
Git Commit: ${GIT_COMMIT}
Build Date: ${BUILD_DATE}
Build Host: $(hostname 2>/dev/null || echo "unknown")

Platforms:
  - darwin-amd64
  - darwin-arm64
  - linux-amd64
  - linux-arm64
  - windows-amd64
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
