# Installation Guide

This guide covers installing Image Tools MCP Server on all supported platforms.

## Platform Support

| Platform | Architecture | OCR Support | Additional Setup |
|----------|-------------|-------------|------------------|
| Linux | AMD64 | Embedded | None required |
| Linux | ARM64 | CLI fallback | Tesseract (optional) |
| macOS | AMD64 (Intel) | CLI fallback | Tesseract (optional) |
| macOS | ARM64 (Apple Silicon) | CLI fallback | Tesseract (optional) |
| Windows | AMD64 | CLI fallback | Tesseract (optional) |
| Windows | ARM64 | CLI fallback | Tesseract (optional) |

**Note**: All platforms support full functionality for non-OCR tools (image loading, cropping, color sampling, measurements, shape detection, etc.) without any additional setup. Tesseract is only needed for the three OCR tools: `image_ocr_full`, `image_ocr_region`, and `image_detect_text_regions`.

## Quick Start

### 1. Download the Binary

Download the appropriate binary for your platform from the [Releases](https://github.com/ironsheep/image_tools_mcp/releases) page:

- `image-tools-mcp-v1.0.0-linux-amd64` - Linux 64-bit (recommended, includes embedded OCR)
- `image-tools-mcp-v1.0.0-linux-arm64` - Linux ARM64 (Raspberry Pi, etc.)
- `image-tools-mcp-v1.0.0-darwin-amd64` - macOS Intel
- `image-tools-mcp-v1.0.0-darwin-arm64` - macOS Apple Silicon
- `image-tools-mcp-v1.0.0-windows-amd64.exe` - Windows 64-bit
- `image-tools-mcp-v1.0.0-windows-arm64.exe` - Windows ARM64

### 2. Make Executable (Linux/macOS)

```bash
chmod +x image-tools-mcp-*
```

### 3. Configure MCP Client

See [MCP Client Configuration](#mcp-client-configuration) below.

## Installing Tesseract (Optional)

For full OCR functionality on platforms other than Linux AMD64, install Tesseract OCR:

### macOS

**Using Homebrew** (recommended):
```bash
brew install tesseract
```

**Using MacPorts**:
```bash
sudo port install tesseract
```

### Linux (ARM64 or if not using embedded OCR)

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

For adding image analysis to existing Docker containers, use the container-tools package:

### 1. Download Container Tools Package

Download `container-tools-v1.0.0.tar.gz` from the [Releases](https://github.com/ironsheep/image_tools_mcp/releases) page.

### 2. Extract and Install

```bash
tar -xzf container-tools-v1.0.0.tar.gz
cd container-tools-v1.0.0

# Copy binary to your container or image
# The package includes the Linux AMD64 binary with embedded OCR
```

### 3. Dockerfile Example

```dockerfile
# Add to an existing container
COPY container-tools-v1.0.0/opt/image-tools-mcp/bin/image-tools-mcp /usr/local/bin/

# Or in a multi-stage build
COPY --from=image-tools /opt/image-tools-mcp/bin/image-tools-mcp /usr/local/bin/
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
2. Use Linux AMD64 binary which has embedded OCR
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
