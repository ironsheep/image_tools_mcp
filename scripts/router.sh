#!/bin/sh
# Router script for universal container
# Detects architecture and runs the appropriate binary

set -e

BINARY_DIR="/app/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# Normalize OS names
case "$OS" in
    linux)
        OS="linux"
        ;;
    darwin)
        OS="darwin"
        ;;
    mingw*|msys*|cygwin*|windows*)
        OS="windows"
        ;;
    *)
        echo "Unsupported OS: $OS" >&2
        exit 1
        ;;
esac

# Construct binary name
BINARY_NAME="image-mcp-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

BINARY_PATH="${BINARY_DIR}/${BINARY_NAME}"

if [ ! -f "$BINARY_PATH" ]; then
    echo "Binary not found: $BINARY_PATH" >&2
    echo "Available binaries:" >&2
    ls -la "$BINARY_DIR" >&2
    exit 1
fi

# Debug output (only if log level is debug)
if [ "$IMAGE_MCP_LOG_LEVEL" = "debug" ]; then
    echo "Detected: OS=$OS ARCH=$ARCH" >&2
    echo "Running: $BINARY_PATH" >&2
fi

# Execute the appropriate binary
exec "$BINARY_PATH" "$@"
