# Image Tools MCP

A Model Context Protocol (MCP) server providing precise image analysis tools for Claude. This server bridges the gap between Claude's visual understanding and the precise measurements needed for tasks like diagram recreation, UI analysis, and technical documentation.

## Why This Exists

When Claude views an image, it can understand what's in it but cannot:
- Measure exact pixel distances between elements
- Extract precise color values (hex, RGB, HSL)
- Reliably read small or stylized text
- Detect and locate shapes with exact coordinates
- Zoom into specific regions for detailed examination

This MCP server provides **18 specialized tools** that give Claude precise numerical data about images.

**Primary Use Case**: Enabling Claude to accurately recreate diagrams as TikZ/LaTeX code by providing exact measurements, colors, text content, and shape positions.

## Features

| Category | Tools |
|----------|-------|
| **Basic Info** | `image_load`, `image_dimensions` |
| **Region Ops** | `image_crop`, `image_crop_quadrant` |
| **Color** | `image_sample_color`, `image_sample_colors_multi`, `image_dominant_colors` |
| **Measurement** | `image_measure_distance`, `image_grid_overlay` |
| **OCR** | `image_ocr_full`, `image_ocr_region`, `image_detect_text_regions` |
| **Shape Detection** | `image_detect_rectangles`, `image_detect_lines`, `image_detect_circles`, `image_edge_detect` |
| **Analysis** | `image_check_alignment`, `image_compare_regions` |

## Quick Start

### 1. Download

Download the binary for your platform from the [Releases](https://github.com/ironsheep/image_tools_mcp/releases) page.

| Platform | Binary | OCR |
|----------|--------|-----|
| Linux AMD64 | `image-tools-mcp-v*-linux-amd64` | Embedded |
| Linux ARM64 | `image-tools-mcp-v*-linux-arm64` | Requires Tesseract |
| macOS Intel | `image-tools-mcp-v*-darwin-amd64` | Requires Tesseract |
| macOS Apple Silicon | `image-tools-mcp-v*-darwin-arm64` | Requires Tesseract |
| Windows | `image-tools-mcp-v*-windows-amd64.exe` | Requires Tesseract |

See [INSTALL.md](INSTALL.md) for detailed setup instructions, including Tesseract installation.

### 2. Configure MCP Client

Add to your Claude Desktop or Claude Code configuration:

```json
{
  "mcpServers": {
    "image-tools": {
      "command": "/path/to/image-tools-mcp"
    }
  }
}
```

### 3. Use with Claude

Once configured, Claude can use tools like:

```
"Load the image and tell me its dimensions"
→ image_load("/path/to/diagram.png")

"What color is the header at position (100, 50)?"
→ image_sample_color with x=100, y=50

"Find all the boxes in this diagram"
→ image_detect_rectangles

"Read the text in the image"
→ image_ocr_full
```

## Documentation

- **[INSTALL.md](INSTALL.md)** - Detailed installation for all platforms, Tesseract setup, container deployment
- **[DOCs/API.md](DOCs/API.md)** - Complete API reference for all 18 tools
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes

## Example Workflow: Recreating a Diagram

```
1. Get dimensions          → image_dimensions → {width: 800, height: 600}
2. Extract colors          → image_dominant_colors → [#FFFFFF, #336699, #333333]
3. Read text labels        → image_ocr_full → "Input → Process → Output"
4. Find boxes              → image_detect_rectangles → [{x:50, y:80, w:100, h:60}, ...]
5. Find connecting arrows  → image_detect_lines → [{start, end, has_arrow: true}, ...]
6. Zoom into details       → image_crop with scale=2.0
```

## Platform Notes

**Linux AMD64** includes embedded Tesseract OCR - full functionality with no additional setup.

**All other platforms** use CLI fallback for OCR. Install Tesseract for full OCR support:
- macOS: `brew install tesseract` or `sudo port install tesseract`
- Windows: [UB Mannheim installer](https://github.com/UB-Mannheim/tesseract/wiki) or `choco install tesseract`
- Linux ARM64: `sudo apt-get install tesseract-ocr`

Non-OCR tools (image loading, cropping, color sampling, measurements, shape detection) work on all platforms without additional setup.

## Container Deployment

For adding to existing Docker containers, download the `container-tools-*.tar.gz` package from Releases. See [INSTALL.md](INSTALL.md#container-deployment) for details.

## Development

```bash
# Clone and build
git clone https://github.com/ironsheep/image_tools_mcp.git
cd image_tools_mcp
make build

# Run tests
make test

# Test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./image-tools-mcp
```

## License

MIT License - see [LICENSE](LICENSE) file.

## Contributing

Contributions welcome! Please open an issue to discuss changes before submitting PRs.
