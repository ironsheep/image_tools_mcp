# Image Analysis MCP Server

## Project Overview

This is an MCP (Model Context Protocol) server that provides image analysis tools for Claude. It enables precise measurement, color extraction, OCR, and shape detection capabilities that Claude lacks when viewing images directly.

**Primary Use Case**: Enabling Claude to accurately recreate diagrams as TikZ code by providing precise measurement and analysis tools.

## Tech Stack

- **Language**: Go 1.22+
- **OCR**: Tesseract via gosseract/v2
- **Image Processing**: disintegration/imaging, anthonynsimon/bild
- **Color Utilities**: lucasb-eyer/go-colorful
- **Protocol**: MCP over stdio (JSON-RPC 2.0)

## Project Structure

```
image-tools-mcp/
├── cmd/image-mcp/          # Entry point (builds image-tools-mcp binary)
│   └── main.go
├── internal/
│   ├── server/             # MCP protocol handling
│   │   ├── server.go       # Main server loop
│   │   ├── tools.go        # Tool definitions
│   │   └── handlers.go     # Tool execution
│   ├── imaging/            # Image operations
│   │   ├── loader.go       # Image loading/caching
│   │   ├── crop.go         # Crop operations
│   │   ├── color.go        # Color sampling
│   │   ├── measure.go      # Distance measurement
│   │   ├── grid.go         # Grid overlay
│   │   └── edge.go         # Edge detection
│   ├── detection/          # Shape detection
│   │   ├── shapes.go       # Rectangle/circle detection
│   │   ├── lines.go        # Line detection
│   │   └── text.go         # Text region detection
│   └── ocr/                # OCR integration
│       └── tesseract.go    # Tesseract wrapper
├── testdata/               # Test images
├── .devcontainer/          # VSCode dev container
├── Dockerfile              # Production container
├── docker-compose.yml
├── Makefile
└── go.mod
```

## MCP Tools (19 total)

### Basic Info
- `image_load` - Load image and get metadata
- `image_dimensions` - Get width/height

### Region Operations
- `image_crop` - Extract rectangular region
- `image_crop_quadrant` - Crop by named region (top-left, center, etc.)

### Color Operations
- `image_sample_color` - Get color at pixel
- `image_sample_colors_multi` - Sample multiple points
- `image_dominant_colors` - Extract color palette

### Measurement
- `image_measure_distance` - Distance between points
- `image_grid_overlay` - Add coordinate grid

### OCR
- `image_ocr_full` - Extract all text
- `image_ocr_region` - Extract text from region
- `image_detect_text_regions` - Find text bounding boxes

### Shape Detection
- `image_detect_rectangles` - Find rectangular shapes
- `image_detect_lines` - Find line segments (with arrow detection)
- `image_detect_circles` - Find circular shapes
- `image_edge_detect` - Canny edge detection

### Analysis
- `image_check_alignment` - Check if points are aligned
- `image_compare_regions` - Compare two regions

## Development Commands

```bash
# Build the binary
make build

# Run tests
make test

# Build Docker image
make docker

# Run the MCP server directly (for testing)
./image-mcp

# Test with a sample MCP request
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./image-mcp
```

## Key Implementation Notes

1. **Image Caching**: The `ImageCache` in `internal/imaging/loader.go` caches loaded images to avoid repeated disk reads.

2. **Coordinate System**: All coordinates are 0-based, with (0,0) at top-left. X increases rightward, Y increases downward.

3. **Color Format**: Colors are returned in multiple formats (hex, RGB, RGBA, HSL) for flexibility.

4. **OCR Language**: Default is English (`eng`). Tesseract must have the language data installed.

5. **Shape Detection**: Uses edge detection and contour analysis. May need tuning via `tolerance` and `min_area` parameters.

6. **Base64 Output**: Cropped images are returned as base64-encoded PNG for direct use by Claude.

## Testing

Test images should be placed in `testdata/`:
- `simple_boxes.png` - Known rectangles
- `text_sample.png` - Known text for OCR
- `color_palette.png` - Known colors

## MCP Configuration

Add to `~/.claude/mcp.json`:

```json
{
  "mcpServers": {
    "image-tools-mcp": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-v", "${HOME}:/home:ro", "ghcr.io/ironsheep/image-tools-mcp:latest"]
    }
  }
}
```

## Reference

See `IMAGE_MCP_BLUEPRINT.md` for the complete specification including all JSON schemas and implementation patterns.
