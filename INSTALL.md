# Installation Guide

This guide covers installing Image Tools MCP Server on all supported platforms.

## Platform Support

| Platform | Architecture | OCR Support | Additional Setup |
|----------|-------------|-------------|------------------|
| Linux | AMD64 | Embedded | None required |
| Linux | ARM64 | Embedded | None required |
| macOS | AMD64 (Intel) | CLI fallback | Tesseract (optional) |
| macOS | ARM64 (Apple Silicon) | CLI fallback | Tesseract (optional) |
| Windows | AMD64 | CLI fallback | Tesseract (optional) |
| Windows | ARM64 | CLI fallback | Tesseract (optional) |

**Note**: All platforms support full functionality for non-OCR tools (image loading, cropping, color sampling, measurements, shape detection, etc.) without any additional setup. Tesseract is only needed for the three OCR tools: `image_ocr_full`, `image_ocr_region`, and `image_detect_text_regions`.

## Quick Start

### 1. Download the Binary

Download the appropriate binary for your platform from the [Releases](https://github.com/ironsheep/image_tools_mcp/releases) page:

- `image-tools-mcp-v*-linux-amd64` - Linux 64-bit (includes embedded OCR)
- `image-tools-mcp-v*-linux-arm64` - Linux ARM64 (Raspberry Pi, etc., includes embedded OCR)
- `image-tools-mcp-v*-darwin-amd64` - macOS Intel
- `image-tools-mcp-v*-darwin-arm64` - macOS Apple Silicon
- `image-tools-mcp-v*-windows-amd64.exe` - Windows 64-bit
- `image-tools-mcp-v*-windows-arm64.exe` - Windows ARM64

### 2. Make Executable (Linux/macOS)

```bash
chmod +x image-tools-mcp-*
```

### 3. Configure MCP Client

See [MCP Client Configuration](#mcp-client-configuration) below.

## Installing Tesseract (Optional)

For full OCR functionality on macOS and Windows, install Tesseract OCR. Linux binaries (both AMD64 and ARM64) include embedded OCR and do not require Tesseract installation.

### macOS

**Using Homebrew** (recommended):
```bash
brew install tesseract
```

**Using MacPorts**:
```bash
sudo port install tesseract
```

### Linux (only if building from source without CGO)

The official Linux binaries include embedded OCR. These instructions are only needed if you build from source without CGO:

**Ubuntu/Debian**:
```bash
sudo apt-get update
sudo apt-get install -y tesseract-ocr tesseract-ocr-eng
```

**Fedora/RHEL**:
```bash
sudo dnf install tesseract tesseract-langpack-eng
```

**Arch Linux**:
```bash
sudo pacman -S tesseract tesseract-data-eng
```

### Windows

1. Download the installer from [UB Mannheim Tesseract](https://github.com/UB-Mannheim/tesseract/wiki)
2. Run the installer and follow the prompts
3. Add Tesseract to your PATH:
   - Default install location: `C:\Program Files\Tesseract-OCR`
   - Add this to your system PATH environment variable

**Using Chocolatey**:
```powershell
choco install tesseract
```

**Using Scoop**:
```powershell
scoop install tesseract
```

### Verify Tesseract Installation

```bash
tesseract --version
```

You should see version information if installed correctly.

## Container Deployment

For adding image analysis to existing Docker containers, use the container-tools package.

**Note:** The container-tools package includes both Linux AMD64 and ARM64 binaries with embedded OCR. No additional Tesseract installation is required for OCR functionality on Linux containers.

### 1. Download Container Tools Package

Download `container-tools-image-tools-mcp-v*.tar.gz` from the [Releases](https://github.com/ironsheep/image_tools_mcp/releases) page.

### 2. Extract and Install

**Option A: Using the install script (recommended)**

The package includes an install script that safely installs alongside other MCP tools following the [Container Tools Integration Guide](DOCs/CONTAINER_TOOLS_MCP_INTEGRATION_GUIDE.md):

```bash
tar -xzf container-tools-image-tools-mcp-v1.2.0.tar.gz
cd container-tools-image-tools-mcp-v1.2.0
sudo ./image-tools-mcp/install.sh
```

The install script will:
- Install to `/opt/container-tools/image-tools-mcp/`
- Create a symlink at `/opt/container-tools/bin/image-tools-mcp`
- Back up any existing installation to `backup/prior/` inside the MCP folder
- Merge into existing `/opt/container-tools/etc/mcp.json` (preserves other MCP entries)
- Install hooks dispatcher and app-start hook
- Skip installation if the binary is already up-to-date (MD5 comparison)

After installation, verify with:
```bash
/opt/container-tools/image-tools-mcp/bin/image-tools-mcp --version
# Or via symlink:
/opt/container-tools/bin/image-tools-mcp --version
```

**Custom installation location:**
```bash
./image-tools-mcp/install.sh --target /custom/path
```

**Uninstall or rollback:**
```bash
./image-tools-mcp/install.sh --uninstall
# If a prior installation exists, it will be restored
# Otherwise, the MCP is fully removed
```

**Option B: Manual copy to custom location**

```bash
tar -xzf container-tools-image-tools-mcp-v1.2.0.tar.gz
cd container-tools-image-tools-mcp-v1.2.0

# Copy binary to your preferred location
cp image-tools-mcp/bin/image-tools-mcp /usr/local/bin/
```

### 3. Dockerfile Examples

**Basic (Linux AMD64 or ARM64 - embedded OCR, no dependencies):**
```dockerfile
# Add to an existing container - OCR works out of the box on both architectures
COPY image-tools-mcp-v*/image-tools-mcp/bin/image-tools-mcp /usr/local/bin/
```

The universal launcher automatically selects the correct binary for your architecture. Both Linux AMD64 and ARM64 binaries include embedded OCR with no external dependencies.

**Note:** The following examples show how to install Tesseract if you need additional language packs beyond English, or if building from source without CGO:

**Debian/Ubuntu-based containers (additional languages):**
```dockerfile
# Only needed for additional OCR languages beyond English
RUN apt-get update && apt-get install -y --no-install-recommends \
    tesseract-ocr-deu \
    tesseract-ocr-fra \
    && rm -rf /var/lib/apt/lists/*

COPY image-tools-mcp-v*/image-tools-mcp/bin/image-tools-mcp /usr/local/bin/
```

**Alpine-based containers (additional languages):**
```dockerfile
# Only needed for additional OCR languages beyond English
RUN apk add --no-cache tesseract-ocr-data-deu tesseract-ocr-data-fra

COPY image-tools-mcp-v*/image-tools-mcp/bin/image-tools-mcp /usr/local/bin/
```

## MCP Client Configuration

### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "image-tools": {
      "command": "/path/to/image-tools-mcp"
    }
  }
}
```

Replace `/path/to/image-tools-mcp` with the actual path to the binary.

### Claude Code (CLI)

Add to `~/.claude/mcp.json`:

```json
{
  "mcpServers": {
    "image-tools": {
      "command": "/path/to/image-tools-mcp"
    }
  }
}
```

### Docker-based Configuration

If running the MCP server in Docker with volume mounts:

```json
{
  "mcpServers": {
    "image-tools": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-v", "/path/to/images:/images:ro",
        "your-image-with-mcp:latest",
        "/usr/local/bin/image-tools-mcp"
      ]
    }
  }
}
```

## Verifying Installation

### Test the Binary

```bash
# Check version
./image-tools-mcp --version

# List available tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./image-tools-mcp
```

### Test with an Image

```bash
# Test image loading (replace with path to any image)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"image_load","arguments":{"path":"/path/to/image.png"}}}' | ./image-tools-mcp
```

### Test OCR (if Tesseract installed)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"image_ocr_full","arguments":{"path":"/path/to/image-with-text.png"}}}' | ./image-tools-mcp
```

## Troubleshooting

### "tesseract not found" Error

This error appears when using OCR tools without Tesseract installed. Either:
1. Install Tesseract following the instructions above
2. Use a Linux binary (AMD64 or ARM64) which has embedded OCR
3. Skip OCR tools if you don't need text extraction

### macOS "Unidentified Developer" Warning

If macOS blocks the binary:
1. Right-click the binary and select "Open"
2. Or: System Preferences > Security & Privacy > General > "Open Anyway"

For signed releases (when available), this warning won't appear.

### Permission Denied

Make sure the binary is executable:
```bash
chmod +x image-tools-mcp-*
```

### Path Issues

Ensure the binary path in your MCP configuration is absolute, not relative:
- Good: `/Users/you/tools/image-tools-mcp`
- Bad: `./image-tools-mcp` or `~/tools/image-tools-mcp`

## Next Steps

- See [DOCs/API.md](DOCs/API.md) for the complete tool reference
- See [README.md](README.md) for an overview and examples
