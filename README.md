# Image Tools MCP

[![CI](https://github.com/ironsheep/image-tools-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/ironsheep/image-tools-mcp/actions/workflows/ci.yml)

[![Release](https://github.com/ironsheep/image-tools-mcp/actions/workflows/release.yml/badge.svg)](https://github.com/ironsheep/image-tools-mcp/actions/workflows/release.yml)

[![Go Version](https://img.shields.io/github/go-mod/go-version/ironsheep/image-tools-mcp)](https://go.dev/)

[![License](https://img.shields.io/github/license/ironsheep/image-tools-mcp)](LICENSE)

[![GitHub release](https://img.shields.io/github/v/release/ironsheep/image-tools-mcp)](https://github.com/ironsheep/image-tools-mcp/releases)

A Model Context Protocol (MCP) server providing precise image analysis tools for Claude. This server bridges the gap between Claude's visual understanding and the precise measurements required for tasks such as diagram recreation, UI analysis, and technical documentation.

## Why This Exists

When Claude views an image, it can understand what's in it, but cannot:
- Measure exact pixel distances between elements
- Extract precise color values (hex, RGB, HSL)
- Reliably read small or stylized text
- Detect and locate shapes with exact coordinates
- Zoom into specific regions for detailed examination

This MCP server addresses these problems by providing 19 specialized tools that give Claude precise numerical data about images.

**Primary Use Case**: Enabling Claude to accurately recreate diagrams as TikZ/LaTeX code by providing exact measurements, colors, text content, and shape positions.

## Features

- **Precise Measurements** - Measure pixel distances, check element alignment, compare regions
- **Color Analysis** - Sample exact colors at any pixel, extract dominant color palettes
- **Region Operations** - Crop and zoom into specific areas with optional scaling
- **OCR** - Extract text with word-level bounding boxes and confidence scores
- **Shape Detection** - Find rectangles, circles, and lines (with arrow detection)
- **Edge Detection** - Canny edge detection for structural analysis
- **Grid Overlay** - Add coordinate grids for visual reference

## Quick Start

### Installation

**Docker (Recommended)**
```bash
docker pull ghcr.io/ironsheep/image-tools-mcp:latest
```

**From Source** (requires Go 1.22+ and Tesseract OCR)
```bash
git clone https://github.com/ironsheep/image-tools-mcp.git
cd image-tools-mcp
make build
```

### Configuration

Add to your Claude Code MCP settings (`~/.claude/mcp.json`):

**Docker:**
```json
{
  "mcpServers": {
    "image-tools": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-v", "${HOME}/Pictures:/images/pictures:ro",
        "-v", "${HOME}/Downloads:/images/downloads:ro",
        "-v", "${PWD}:/images/project:ro",
        "ghcr.io/ironsheep/image-tools-mcp:latest"
      ]
    }
  }
}
```

**Binary:**
```json
{
  "mcpServers": {
    "image-tools": {
      "command": "/path/to/image-tools-mcp"
    }
  }
}
```

---

## API Reference

All tools accept a `path` parameter with the absolute path to the image file. When using Docker, paths must be within mounted volumes (e.g., `/images/project/diagram.png`).

### Basic Image Information

#### `image_load`
Load an image and retrieve comprehensive metadata.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute path to image file |

**Response:**
```json
{
  "width": 800,
  "height": 600,
  "format": "png",
  "color_depth": "8-bit",
  "has_alpha": true,
  "file_size_bytes": 45230
}
```

---

#### `image_dimensions`
Get image dimensions quickly without full metadata.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute path to image file |

**Response:**
```json
{
  "width": 800,
  "height": 600
}
```

---

### Region Operations

#### `image_crop`
Extract a rectangular region as a base64-encoded PNG. Useful for zooming into areas that need detailed examination.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `x1` | integer | Yes | - | Left edge X coordinate (0-based) |
| `y1` | integer | Yes | - | Top edge Y coordinate (0-based) |
| `x2` | integer | Yes | - | Right edge X coordinate (exclusive) |
| `y2` | integer | Yes | - | Bottom edge Y coordinate (exclusive) |
| `scale` | number | No | 1.0 | Scale factor (e.g., 2.0 doubles size) |

**Response:**
```json
{
  "width": 200,
  "height": 150,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA...",
  "mime_type": "image/png"
}
```

---

#### `image_crop_quadrant`
Crop a named region of the image for quick access to common areas.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `region` | string | Yes | - | One of: `top-left`, `top-right`, `bottom-left`, `bottom-right`, `top-half`, `bottom-half`, `left-half`, `right-half`, `center` |
| `scale` | number | No | 1.0 | Scale factor |

**Response:** Same as `image_crop`

---

### Color Operations

#### `image_sample_color`
Get the exact color at a specific pixel coordinate.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute path to image file |
| `x` | integer | Yes | X coordinate (0-based, from left) |
| `y` | integer | Yes | Y coordinate (0-based, from top) |

**Response:**
```json
{
  "hex": "#336699",
  "rgb": {"r": 51, "g": 102, "b": 153},
  "rgba": {"r": 51, "g": 102, "b": 153, "a": 255},
  "hsl": {"h": 210, "s": 50, "l": 40}
}
```

---

#### `image_sample_colors_multi`
Sample colors at multiple points in a single call for efficiency.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute path to image file |
| `points` | array | Yes | Array of `{x, y, label?}` objects |

**Example Input:**
```json
{
  "path": "/images/diagram.png",
  "points": [
    {"x": 100, "y": 50, "label": "header"},
    {"x": 200, "y": 150, "label": "box1"},
    {"x": 300, "y": 150, "label": "box2"}
  ]
}
```

**Response:**
```json
{
  "samples": [
    {"label": "header", "x": 100, "y": 50, "hex": "#FFFFFF", "rgb": {...}},
    {"label": "box1", "x": 200, "y": 150, "hex": "#336699", "rgb": {...}},
    {"label": "box2", "x": 300, "y": 150, "hex": "#996633", "rgb": {...}}
  ]
}
```

---

#### `image_dominant_colors`
Extract the N most dominant colors from an image or region (color palette extraction).

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `count` | integer | No | 5 | Number of colors to return |
| `region` | object | No | entire image | Optional `{x1, y1, x2, y2}` bounds |

**Response:**
```json
{
  "colors": [
    {"hex": "#FFFFFF", "percentage": 45.2, "rgb": {"r": 255, "g": 255, "b": 255}},
    {"hex": "#336699", "percentage": 22.1, "rgb": {"r": 51, "g": 102, "b": 153}},
    {"hex": "#333333", "percentage": 15.8, "rgb": {"r": 51, "g": 51, "b": 51}}
  ]
}
```

---

### Measurement Operations

#### `image_measure_distance`
Measure the pixel distance between two points with additional metrics.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute path to image file |
| `x1` | integer | Yes | First point X |
| `y1` | integer | Yes | First point Y |
| `x2` | integer | Yes | Second point X |
| `y2` | integer | Yes | Second point Y |

**Response:**
```json
{
  "distance_pixels": 127.3,
  "delta_x": 100,
  "delta_y": 79,
  "angle_degrees": 38.3,
  "distance_percent_width": 15.9,
  "distance_percent_height": 13.2
}
```

---

#### `image_grid_overlay`
Generate a version of the image with a coordinate grid overlay for precise positioning reference.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `grid_spacing` | integer | No | 50 | Pixels between grid lines |
| `show_coordinates` | boolean | No | true | Label grid intersections |
| `grid_color` | string | No | `#FF000080` | Grid color (hex with optional alpha) |

**Response:**
```json
{
  "width": 800,
  "height": 600,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA...",
  "mime_type": "image/png"
}
```

---

#### `image_check_alignment`
Check if multiple points are horizontally or vertically aligned.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `points` | array | Yes | - | Array of `{x, y}` objects |
| `tolerance` | integer | No | 5 | Pixel tolerance for alignment |

**Response:**
```json
{
  "horizontally_aligned": true,
  "vertically_aligned": false,
  "horizontal_variance": 3,
  "vertical_variance": 245,
  "average_y": 142,
  "average_x": 325
}
```

---

#### `image_compare_regions`
Compare two regions to determine similarity (useful for detecting repeated elements).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute path to image file |
| `region1` | object | Yes | `{x1, y1, x2, y2}` bounds |
| `region2` | object | Yes | `{x1, y1, x2, y2}` bounds |

**Response:**
```json
{
  "similarity_score": 0.95,
  "pixel_difference_avg": 12.3,
  "dimensions_match": true
}
```

---

### OCR (Text Extraction)

#### `image_ocr_full`
Extract all text from the image with word-level bounding boxes.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `language` | string | No | `eng` | Tesseract language code |

**Response:**
```json
{
  "full_text": "Input\nProcess\nOutput",
  "regions": [
    {
      "text": "Input",
      "confidence": 0.95,
      "bounds": {"x1": 50, "y1": 100, "x2": 120, "y2": 125}
    },
    {
      "text": "Process",
      "confidence": 0.92,
      "bounds": {"x1": 350, "y1": 280, "x2": 450, "y2": 305}
    }
  ]
}
```

---

#### `image_ocr_region`
Extract text from a specific rectangular region.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `x1` | integer | Yes | - | Left edge X |
| `y1` | integer | Yes | - | Top edge Y |
| `x2` | integer | Yes | - | Right edge X |
| `y2` | integer | Yes | - | Bottom edge Y |
| `language` | string | No | `eng` | Tesseract language code |

**Response:** Same structure as `image_ocr_full`

---

#### `image_detect_text_regions`
Find all regions containing text without performing full OCR (faster).

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `min_confidence` | number | No | 0.5 | Minimum confidence (0-1) |

**Response:**
```json
{
  "regions": [
    {"bounds": {"x1": 50, "y1": 100, "x2": 120, "y2": 125}, "confidence": 0.89},
    {"bounds": {"x1": 350, "y1": 280, "x2": 450, "y2": 305}, "confidence": 0.92}
  ]
}
```

---

### Shape Detection

#### `image_detect_rectangles`
Detect rectangular shapes in the image.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `min_area` | integer | No | 100 | Minimum area in pixels |
| `tolerance` | number | No | 0.9 | Rectangularity threshold (0-1) |

**Response:**
```json
{
  "rectangles": [
    {
      "bounds": {"x1": 50, "y1": 100, "x2": 200, "y2": 180},
      "center": {"x": 125, "y": 140},
      "width": 150,
      "height": 80,
      "area": 12000,
      "fill_color": "#336699",
      "border_color": "#000000",
      "confidence": 0.95
    }
  ]
}
```

---

#### `image_detect_lines`
Detect line segments with optional arrow head detection.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `min_length` | integer | No | 20 | Minimum line length in pixels |
| `detect_arrows` | boolean | No | true | Detect arrow heads |

**Response:**
```json
{
  "lines": [
    {
      "start": {"x": 200, "y": 140},
      "end": {"x": 350, "y": 140},
      "length": 150,
      "angle_degrees": 0,
      "color": "#000000",
      "thickness_approx": 2,
      "has_arrow_start": false,
      "has_arrow_end": true
    }
  ]
}
```

---

#### `image_detect_circles`
Detect circular shapes in the image.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `min_radius` | integer | No | 5 | Minimum radius in pixels |
| `max_radius` | integer | No | 500 | Maximum radius in pixels |

**Response:**
```json
{
  "circles": [
    {
      "center": {"x": 150, "y": 200},
      "radius": 25,
      "color": "#FF0000",
      "confidence": 0.88
    }
  ]
}
```

---

#### `image_edge_detect`
Generate an edge-detected version using Canny edge detection.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to image file |
| `threshold_low` | integer | No | 50 | Low threshold for hysteresis |
| `threshold_high` | integer | No | 150 | High threshold for hysteresis |

**Response:**
```json
{
  "width": 800,
  "height": 600,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA...",
  "mime_type": "image/png"
}
```

---

## Usage Examples

### Workflow: Recreating a Diagram as TikZ

```
1. Get dimensions and establish scale
   → image_dimensions("/images/project/diagram.png")
   → {width: 800, height: 600}

2. Extract the color palette
   → image_dominant_colors("/images/project/diagram.png", count=5)
   → [#FFFFFF (45%), #336699 (22%), #333333 (16%), ...]

3. Read all text labels
   → image_ocr_full("/images/project/diagram.png")
   → "Input → Process → Output" with bounding boxes

4. Find all boxes
   → image_detect_rectangles("/images/project/diagram.png")
   → [{bounds: {50,80,150,140}, fill: #336699}, ...]

5. Find connecting lines/arrows
   → image_detect_lines("/images/project/diagram.png")
   → [{start: {200,110}, end: {350,110}, has_arrow_end: true}, ...]

6. Sample specific colors for accuracy
   → image_sample_color("/images/project/diagram.png", x=100, y=110)
   → {hex: "#336699", rgb: {r:51, g:102, b:153}}

7. Zoom into unclear areas
   → image_crop("/images/project/diagram.png", x1=300, y1=250, x2=450, y2=350, scale=2.0)
   → Base64 PNG of zoomed region
```

### Workflow: Analyzing UI Screenshots

```
1. Add a grid overlay for coordinate reference
   → image_grid_overlay("/images/screenshot.png", grid_spacing=100)

2. Check if elements are aligned
   → image_check_alignment(points=[{x:50,y:100}, {x:50,y:200}, {x:50,y:300}])
   → {vertically_aligned: true, horizontal_variance: 2}

3. Compare repeated elements
   → image_compare_regions(region1={...}, region2={...})
   → {similarity_score: 0.98}
```

---

## Development

### Prerequisites

- Go 1.22+
- Tesseract OCR with language data (`tesseract-ocr`, `tesseract-ocr-data-eng`)
- Docker (for container builds)

### Building

```bash
# Build binary
make build

# Run tests
make test

# Build Docker image
make docker

# Build for all platforms
make dist
```

### Project Structure

```
image-tools-mcp/
├── cmd/image-mcp/          # Entry point
│   └── main.go
├── internal/
│   ├── server/             # MCP protocol handling
│   │   ├── server.go       # JSON-RPC server loop
│   │   ├── tools.go        # Tool definitions (19 tools)
│   │   └── handlers.go     # Tool execution
│   ├── imaging/            # Image operations
│   │   ├── loader.go       # Image loading with caching
│   │   ├── crop.go         # Crop operations
│   │   ├── color.go        # Color sampling & palette
│   │   ├── measure.go      # Distance & alignment
│   │   ├── grid.go         # Grid overlay
│   │   └── edge.go         # Canny edge detection
│   ├── detection/          # Shape detection
│   │   ├── shapes.go       # Rectangle & circle detection
│   │   ├── lines.go        # Line detection with arrows
│   │   └── text.go         # Text region detection
│   └── ocr/                # OCR integration
│       └── tesseract.go    # Tesseract wrapper
├── testdata/               # Test images
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

### Testing MCP Protocol

```bash
# Initialize connection
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./image-tools-mcp

# List available tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./image-tools-mcp

# Call a tool
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"image_dimensions","arguments":{"path":"/path/to/image.png"}}}' | ./image-tools-mcp
```

### Dev Container

This project includes a VS Code Dev Container with all dependencies pre-installed:

1. Open in VS Code
2. Click "Reopen in Container" when prompted
3. All tools (Go, Tesseract, linters) are ready

---

## Coordinate System

All coordinates use a standard image coordinate system:
- **Origin (0,0)**: Top-left corner
- **X-axis**: Increases rightward
- **Y-axis**: Increases downward
- **Exclusive bounds**: For regions, `x2` and `y2` are exclusive (standard rectangle convention)

```
(0,0) ────────────► X
  │
  │    ┌─────────┐
  │    │ region  │
  │    │(x1,y1)──┤
  │    └─────────┘(x2,y2)
  ▼
  Y
```

---

## Platform Support

| Platform | Architecture | OCR Support |
|----------|--------------|-------------|
| Linux | amd64, arm64 | Full |
| macOS | amd64 (Intel), arm64 (Apple Silicon) | Limited* |
| Windows | amd64, arm64 | Limited* |
| Docker | linux/amd64, linux/arm64 | Full |

*Non-Linux platforms have OCR disabled in binary builds due to Tesseract CGO requirements. Use Docker for full OCR support on all platforms.

---

## Troubleshooting

### "File not found" errors with Docker
Ensure the image path is within a mounted volume. With Docker, you access files via mount points:
- `${HOME}/Pictures` → `/images/pictures`
- `${HOME}/Downloads` → `/images/downloads`
- `${PWD}` → `/images/project`

### OCR returns empty results
- Ensure Tesseract is installed with language data
- Try increasing image contrast or cropping to just the text region
- Check that text is large enough (try `scale: 2.0` with crop)

### Shape detection misses elements
- Adjust `tolerance` parameter (lower = more permissive)
- Adjust `min_area` to include smaller shapes
- Use `image_edge_detect` first to visualize what edges are being detected

### Performance with large images
- Images are cached after first load
- Use `image_crop` to work with smaller regions
- Consider reducing image resolution before analysis

---

## Dependencies

| Package | Purpose |
|---------|---------|
| [disintegration/imaging](https://github.com/disintegration/imaging) | Image resizing, cropping |
| [anthonynsimon/bild](https://github.com/anthonynsimon/bild) | Image filters and effects |
| [otiai10/gosseract](https://github.com/otiai10/gosseract) | Tesseract OCR bindings |
| [lucasb-eyer/go-colorful](https://github.com/lucasb-eyer/go-colorful) | Color space conversions |

---

## License

MIT License - see [LICENSE](LICENSE) file.

## Contributing

Contributions welcome! Please open an issue to discuss changes before submitting PRs.
