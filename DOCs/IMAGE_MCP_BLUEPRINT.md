# Image Analysis MCP - Complete Build Specification

## Project Overview

This document provides complete specifications for building an MCP (Model Context Protocol) server that provides image analysis tools for Claude. The primary use case is enabling Claude to accurately recreate diagrams as TikZ code by giving it precise measurement, color, text extraction, and region examination capabilities.

**Target User**: Claude Code instances working on diagram recreation tasks.

**Core Problem Solved**: Claude can see images but cannot measure them, zoom into regions, extract precise colors, or reliably read small text. This MCP bridges that gap.

---

## MCP Tool Definitions

### Tool Category 1: Basic Image Information

#### `image_load`
Load an image and return basic metadata. This should be called first to establish the working image.

```json
{
  "name": "image_load",
  "description": "Load an image file and return its dimensions and format. Sets this as the active image for subsequent operations.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      }
    },
    "required": ["path"]
  }
}
```

**Returns:**
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

#### `image_dimensions`
Get dimensions of an image without loading it as active.

```json
{
  "name": "image_dimensions",
  "description": "Get the width and height of an image file.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      }
    },
    "required": ["path"]
  }
}
```

**Returns:**
```json
{
  "width": 800,
  "height": 600
}
```

---

### Tool Category 2: Region Operations

#### `image_crop`
Extract a rectangular region and return it as a new image (base64 encoded).

```json
{
  "name": "image_crop",
  "description": "Crop a rectangular region from an image and return it as base64-encoded PNG. Use this to zoom into areas that need detailed examination.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "x1": {
        "type": "integer",
        "description": "Left edge X coordinate (0-based)"
      },
      "y1": {
        "type": "integer",
        "description": "Top edge Y coordinate (0-based)"
      },
      "x2": {
        "type": "integer",
        "description": "Right edge X coordinate (exclusive)"
      },
      "y2": {
        "type": "integer",
        "description": "Bottom edge Y coordinate (exclusive)"
      },
      "scale": {
        "type": "number",
        "description": "Optional scale factor (e.g., 2.0 to double size). Default 1.0",
        "default": 1.0
      }
    },
    "required": ["path", "x1", "y1", "x2", "y2"]
  }
}
```

**Returns:**
```json
{
  "width": 200,
  "height": 150,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA...",
  "mime_type": "image/png"
}
```

#### `image_crop_quadrant`
Convenience method to crop by named region.

```json
{
  "name": "image_crop_quadrant",
  "description": "Crop a named region of the image (top-left, top-right, bottom-left, bottom-right, top-half, bottom-half, left-half, right-half, center).",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "region": {
        "type": "string",
        "enum": ["top-left", "top-right", "bottom-left", "bottom-right", "top-half", "bottom-half", "left-half", "right-half", "center"],
        "description": "Named region to extract"
      },
      "scale": {
        "type": "number",
        "description": "Optional scale factor. Default 1.0",
        "default": 1.0
      }
    },
    "required": ["path", "region"]
  }
}
```

---

### Tool Category 3: Color Operations

#### `image_sample_color`
Get the exact color at a specific pixel.

```json
{
  "name": "image_sample_color",
  "description": "Get the exact color value at a specific pixel coordinate.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "x": {
        "type": "integer",
        "description": "X coordinate (0-based, from left)"
      },
      "y": {
        "type": "integer",
        "description": "Y coordinate (0-based, from top)"
      }
    },
    "required": ["path", "x", "y"]
  }
}
```

**Returns:**
```json
{
  "hex": "#336699",
  "rgb": {"r": 51, "g": 102, "b": 153},
  "rgba": {"r": 51, "g": 102, "b": 153, "a": 255},
  "hsl": {"h": 210, "s": 50, "l": 40}
}
```

#### `image_sample_colors_multi`
Sample colors at multiple points in one call.

```json
{
  "name": "image_sample_colors_multi",
  "description": "Get color values at multiple pixel coordinates in a single call.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "points": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "x": {"type": "integer"},
            "y": {"type": "integer"},
            "label": {"type": "string", "description": "Optional label for this point"}
          },
          "required": ["x", "y"]
        },
        "description": "Array of points to sample"
      }
    },
    "required": ["path", "points"]
  }
}
```

#### `image_dominant_colors`
Extract the dominant colors from an image or region.

```json
{
  "name": "image_dominant_colors",
  "description": "Analyze an image and return the N most dominant colors (color palette extraction).",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "count": {
        "type": "integer",
        "description": "Number of dominant colors to return (default 5)",
        "default": 5
      },
      "region": {
        "type": "object",
        "properties": {
          "x1": {"type": "integer"},
          "y1": {"type": "integer"},
          "x2": {"type": "integer"},
          "y2": {"type": "integer"}
        },
        "description": "Optional region to analyze. If omitted, analyzes entire image."
      }
    },
    "required": ["path"]
  }
}
```

**Returns:**
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

### Tool Category 4: Measurement Operations

#### `image_measure_distance`
Measure pixel distance between two points.

```json
{
  "name": "image_measure_distance",
  "description": "Measure the distance in pixels between two points.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "x1": {"type": "integer", "description": "First point X"},
      "y1": {"type": "integer", "description": "First point Y"},
      "x2": {"type": "integer", "description": "Second point X"},
      "y2": {"type": "integer", "description": "Second point Y"}
    },
    "required": ["path", "x1", "y1", "x2", "y2"]
  }
}
```

**Returns:**
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

#### `image_grid_overlay`
Generate a version of the image with a coordinate grid overlay.

```json
{
  "name": "image_grid_overlay",
  "description": "Return a version of the image with a coordinate grid overlay for precise positioning reference.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "grid_spacing": {
        "type": "integer",
        "description": "Pixels between grid lines (default 50)",
        "default": 50
      },
      "show_coordinates": {
        "type": "boolean",
        "description": "Whether to label grid intersections with coordinates",
        "default": true
      },
      "grid_color": {
        "type": "string",
        "description": "Grid line color as hex (default #FF000080 - semi-transparent red)",
        "default": "#FF000080"
      }
    },
    "required": ["path"]
  }
}
```

---

### Tool Category 5: Text Extraction (OCR)

#### `image_ocr_full`
Extract all text from the entire image.

```json
{
  "name": "image_ocr_full",
  "description": "Extract all text from the image using OCR. Returns text with approximate bounding boxes.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "language": {
        "type": "string",
        "description": "OCR language hint (default 'eng')",
        "default": "eng"
      }
    },
    "required": ["path"]
  }
}
```

**Returns:**
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

#### `image_ocr_region`
Extract text from a specific region.

```json
{
  "name": "image_ocr_region",
  "description": "Extract text from a specific rectangular region of the image.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "x1": {"type": "integer"},
      "y1": {"type": "integer"},
      "x2": {"type": "integer"},
      "y2": {"type": "integer"},
      "language": {
        "type": "string",
        "default": "eng"
      }
    },
    "required": ["path", "x1", "y1", "x2", "y2"]
  }
}
```

#### `image_detect_text_regions`
Find all regions containing text (without reading them).

```json
{
  "name": "image_detect_text_regions",
  "description": "Detect all regions in the image that contain text. Returns bounding boxes without performing full OCR.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "min_confidence": {
        "type": "number",
        "description": "Minimum confidence threshold (0-1, default 0.5)",
        "default": 0.5
      }
    },
    "required": ["path"]
  }
}
```

---

### Tool Category 6: Shape Detection

#### `image_detect_rectangles`
Find rectangular shapes in the image.

```json
{
  "name": "image_detect_rectangles",
  "description": "Detect rectangular shapes in the image. Useful for finding boxes in diagrams.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "min_area": {
        "type": "integer",
        "description": "Minimum area in pixels to consider (default 100)",
        "default": 100
      },
      "tolerance": {
        "type": "number",
        "description": "How close to rectangular a shape must be (0-1, default 0.9)",
        "default": 0.9
      }
    },
    "required": ["path"]
  }
}
```

**Returns:**
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

#### `image_detect_lines`
Find line segments in the image.

```json
{
  "name": "image_detect_lines",
  "description": "Detect line segments in the image. Useful for finding connections between elements.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "min_length": {
        "type": "integer",
        "description": "Minimum line length in pixels (default 20)",
        "default": 20
      },
      "detect_arrows": {
        "type": "boolean",
        "description": "Whether to detect arrow heads at line endpoints",
        "default": true
      }
    },
    "required": ["path"]
  }
}
```

**Returns:**
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

#### `image_detect_circles`
Find circular shapes in the image.

```json
{
  "name": "image_detect_circles",
  "description": "Detect circular shapes in the image. Useful for finding nodes, connectors, or bullets.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "min_radius": {
        "type": "integer",
        "description": "Minimum radius in pixels (default 5)",
        "default": 5
      },
      "max_radius": {
        "type": "integer",
        "description": "Maximum radius in pixels (default 500)",
        "default": 500
      }
    },
    "required": ["path"]
  }
}
```

#### `image_edge_detect`
Generate an edge-detected version of the image.

```json
{
  "name": "image_edge_detect",
  "description": "Return an edge-detected version of the image, showing only structural lines. Useful for understanding diagram structure without color fills.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "threshold_low": {
        "type": "integer",
        "description": "Low threshold for Canny edge detection (default 50)",
        "default": 50
      },
      "threshold_high": {
        "type": "integer",
        "description": "High threshold for Canny edge detection (default 150)",
        "default": 150
      }
    },
    "required": ["path"]
  }
}
```

---

### Tool Category 7: Analysis Helpers

#### `image_check_alignment`
Check if elements are aligned.

```json
{
  "name": "image_check_alignment",
  "description": "Check if multiple points or regions are horizontally or vertically aligned.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "points": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "x": {"type": "integer"},
            "y": {"type": "integer"}
          },
          "required": ["x", "y"]
        },
        "description": "Points to check for alignment"
      },
      "tolerance": {
        "type": "integer",
        "description": "Pixel tolerance for alignment (default 5)",
        "default": 5
      }
    },
    "required": ["path", "points"]
  }
}
```

**Returns:**
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

#### `image_compare_regions`
Compare two regions for similarity.

```json
{
  "name": "image_compare_regions",
  "description": "Compare two regions of an image to determine if they contain similar content (useful for detecting repeated elements).",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Absolute path to the image file"
      },
      "region1": {
        "type": "object",
        "properties": {
          "x1": {"type": "integer"},
          "y1": {"type": "integer"},
          "x2": {"type": "integer"},
          "y2": {"type": "integer"}
        },
        "required": ["x1", "y1", "x2", "y2"]
      },
      "region2": {
        "type": "object",
        "properties": {
          "x1": {"type": "integer"},
          "y1": {"type": "integer"},
          "x2": {"type": "integer"},
          "y2": {"type": "integer"}
        },
        "required": ["x1", "y1", "x2", "y2"]
      }
    },
    "required": ["path", "region1", "region2"]
  }
}
```

---

## Project Structure

```
image-mcp/
├── cmd/
│   └── image-mcp/
│       └── main.go                 # Entry point
├── internal/
│   ├── server/
│   │   ├── server.go               # MCP protocol handler
│   │   ├── tools.go                # Tool definitions
│   │   └── handlers.go             # Tool execution handlers
│   ├── imaging/
│   │   ├── loader.go               # Image loading/caching
│   │   ├── crop.go                 # Crop operations
│   │   ├── color.go                # Color sampling, palette extraction
│   │   ├── measure.go              # Distance, alignment
│   │   ├── grid.go                 # Grid overlay generation
│   │   └── edge.go                 # Edge detection
│   ├── detection/
│   │   ├── shapes.go               # Rectangle, circle detection
│   │   ├── lines.go                # Line segment detection
│   │   └── text.go                 # Text region detection
│   └── ocr/
│       └── tesseract.go            # Tesseract integration
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── mcp-config.json
├── README.md
└── Makefile
```

---

## Implementation Guide

### Dependencies (go.mod)

```go
module github.com/yourusername/image-mcp

go 1.22

require (
    github.com/disintegration/imaging v1.6.2
    github.com/anthonynsimon/bild v0.14.0
    github.com/otiai10/gosseract/v2 v2.4.1
    github.com/lucasb-eyer/go-colorful v1.2.0
)
```

### Core MCP Server Pattern

Use the same MCP server pattern as todo-mcp. Key elements:

```go
// internal/server/server.go
package server

import (
    "encoding/json"
    "bufio"
    "os"
)

type Server struct {
    // image cache, config, etc.
}

type MCPRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   *MCPError   `json:"error,omitempty"`
}

func (s *Server) Run() error {
    scanner := bufio.NewScanner(os.Stdin)
    encoder := json.NewEncoder(os.Stdout)

    for scanner.Scan() {
        var req MCPRequest
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            continue
        }

        resp := s.handleRequest(&req)
        encoder.Encode(resp)
    }
    return nil
}
```

### Image Loading Pattern

```go
// internal/imaging/loader.go
package imaging

import (
    "image"
    "image/png"
    "image/jpeg"
    _ "image/gif"
    "os"
    "sync"
)

type ImageCache struct {
    mu     sync.RWMutex
    images map[string]image.Image
}

func (c *ImageCache) Load(path string) (image.Image, error) {
    c.mu.RLock()
    if img, ok := c.images[path]; ok {
        c.mu.RUnlock()
        return img, nil
    }
    c.mu.RUnlock()

    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    img, _, err := image.Decode(f)
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.images[path] = img
    c.mu.Unlock()

    return img, nil
}
```

### Color Sampling Pattern

```go
// internal/imaging/color.go
package imaging

import (
    "image"
    "fmt"
)

type ColorResult struct {
    Hex  string   `json:"hex"`
    RGB  RGBColor `json:"rgb"`
    RGBA RGBAColor `json:"rgba"`
    HSL  HSLColor `json:"hsl"`
}

type RGBColor struct {
    R uint8 `json:"r"`
    G uint8 `json:"g"`
    B uint8 `json:"b"`
}

func SampleColor(img image.Image, x, y int) (*ColorResult, error) {
    bounds := img.Bounds()
    if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
        return nil, fmt.Errorf("coordinates (%d,%d) outside image bounds", x, y)
    }

    r, g, b, a := img.At(x, y).RGBA()
    // Convert from 16-bit to 8-bit
    r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)

    return &ColorResult{
        Hex: fmt.Sprintf("#%02X%02X%02X", r8, g8, b8),
        RGB: RGBColor{R: r8, G: g8, B: b8},
        RGBA: RGBAColor{R: r8, G: g8, B: b8, A: a8},
        HSL: rgbToHSL(r8, g8, b8),
    }, nil
}
```

### Cropping with Scale

```go
// internal/imaging/crop.go
package imaging

import (
    "bytes"
    "encoding/base64"
    "image"
    "image/png"

    "github.com/disintegration/imaging"
)

type CropResult struct {
    Width       int    `json:"width"`
    Height      int    `json:"height"`
    ImageBase64 string `json:"image_base64"`
    MimeType    string `json:"mime_type"`
}

func Crop(img image.Image, x1, y1, x2, y2 int, scale float64) (*CropResult, error) {
    cropped := imaging.Crop(img, image.Rect(x1, y1, x2, y2))

    if scale != 1.0 {
        newWidth := int(float64(cropped.Bounds().Dx()) * scale)
        newHeight := int(float64(cropped.Bounds().Dy()) * scale)
        cropped = imaging.Resize(cropped, newWidth, newHeight, imaging.Lanczos)
    }

    var buf bytes.Buffer
    if err := png.Encode(&buf, cropped); err != nil {
        return nil, err
    }

    return &CropResult{
        Width:       cropped.Bounds().Dx(),
        Height:      cropped.Bounds().Dy(),
        ImageBase64: base64.StdEncoding.EncodeToString(buf.Bytes()),
        MimeType:    "image/png",
    }, nil
}
```

### Tesseract OCR Integration

```go
// internal/ocr/tesseract.go
package ocr

import (
    "github.com/otiai10/gosseract/v2"
)

type OCRResult struct {
    FullText string       `json:"full_text"`
    Regions  []TextRegion `json:"regions"`
}

type TextRegion struct {
    Text       string  `json:"text"`
    Confidence float64 `json:"confidence"`
    Bounds     Bounds  `json:"bounds"`
}

func ExtractText(imagePath string, language string) (*OCRResult, error) {
    client := gosseract.NewClient()
    defer client.Close()

    client.SetLanguage(language)
    client.SetImage(imagePath)

    text, err := client.Text()
    if err != nil {
        return nil, err
    }

    // Get bounding boxes
    boxes, err := client.GetBoundingBoxes(gosseract.RIL_WORD)
    if err != nil {
        return nil, err
    }

    regions := make([]TextRegion, 0, len(boxes))
    for _, box := range boxes {
        regions = append(regions, TextRegion{
            Text:       box.Word,
            Confidence: float64(box.Confidence) / 100.0,
            Bounds: Bounds{
                X1: box.Box.Min.X,
                Y1: box.Box.Min.Y,
                X2: box.Box.Max.X,
                Y2: box.Box.Max.Y,
            },
        })
    }

    return &OCRResult{
        FullText: text,
        Regions:  regions,
    }, nil
}
```

### Dominant Color Extraction

```go
// internal/imaging/color.go (continued)
package imaging

import (
    "image"
    "sort"
)

func DominantColors(img image.Image, count int, region *image.Rectangle) ([]ColorFrequency, error) {
    bounds := img.Bounds()
    if region != nil {
        bounds = *region
    }

    colorCounts := make(map[string]int)
    totalPixels := 0

    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            r, g, b, _ := img.At(x, y).RGBA()
            // Quantize to reduce color space (group similar colors)
            r8 := uint8((r >> 8) / 16 * 16)
            g8 := uint8((g >> 8) / 16 * 16)
            b8 := uint8((b >> 8) / 16 * 16)
            key := fmt.Sprintf("#%02X%02X%02X", r8, g8, b8)
            colorCounts[key]++
            totalPixels++
        }
    }

    // Convert to slice and sort by frequency
    colors := make([]ColorFrequency, 0, len(colorCounts))
    for hex, count := range colorCounts {
        colors = append(colors, ColorFrequency{
            Hex:        hex,
            Count:      count,
            Percentage: float64(count) / float64(totalPixels) * 100,
        })
    }

    sort.Slice(colors, func(i, j int) bool {
        return colors[i].Count > colors[j].Count
    })

    if len(colors) > count {
        colors = colors[:count]
    }

    return colors, nil
}
```

---

## Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /image-mcp ./cmd/image-mcp

# Runtime stage
FROM alpine:3.19

# Install Tesseract OCR and language data
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /image-mcp /app/image-mcp

# The MCP will need access to image files - mount volumes as needed
# Default to /images as the working directory for images
RUN mkdir -p /images && chown appuser:appuser /images

USER appuser

ENTRYPOINT ["/app/image-mcp"]
```

---

## Docker Compose

```yaml
version: '3.8'

services:
  image-mcp:
    build:
      context: .
      dockerfile: Dockerfile
    image: image-mcp:latest
    container_name: image-mcp
    stdin_open: true
    volumes:
      # Mount common image directories - adjust paths as needed
      - ${HOME}/Pictures:/images/pictures:ro
      - ${HOME}/Downloads:/images/downloads:ro
      - ${HOME}/Desktop:/images/desktop:ro
      # Mount project directories for diagram work
      - ${PWD}:/images/project:ro
    environment:
      - IMAGE_MCP_LOG_LEVEL=info
```

---

## MCP Configuration for Claude Code

Add to your Claude Code MCP configuration (`~/.claude/mcp.json` or project-level):

```json
{
  "mcpServers": {
    "image-mcp": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "-v", "${HOME}/Pictures:/images/pictures:ro",
        "-v", "${HOME}/Downloads:/images/downloads:ro",
        "-v", "${HOME}/Desktop:/images/desktop:ro",
        "-v", "${PWD}:/images/project:ro",
        "image-mcp:latest"
      ]
    }
  }
}
```

**Alternative: Direct binary (if not using Docker):**

```json
{
  "mcpServers": {
    "image-mcp": {
      "command": "/path/to/image-mcp",
      "args": []
    }
  }
}
```

---

## Makefile

```makefile
.PHONY: build test docker clean

VERSION := $(shell cat VERSION 2>/dev/null || echo "0.1.0")
BUILD_TIME := $(shell date -u +'%Y-%m-%d %H:%M:%S UTC')

build:
	go build -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'" \
		-o image-mcp ./cmd/image-mcp

test:
	go test -v ./...

docker:
	docker build -t image-mcp:$(VERSION) -t image-mcp:latest .

docker-run:
	docker run --rm -i \
		-v $(HOME)/Pictures:/images/pictures:ro \
		-v $(PWD):/images/project:ro \
		image-mcp:latest

clean:
	rm -f image-mcp
	docker rmi image-mcp:latest image-mcp:$(VERSION) 2>/dev/null || true

install: build
	cp image-mcp /usr/local/bin/
```

---

## Testing Strategy

### Unit Tests

```go
// internal/imaging/color_test.go
package imaging

import (
    "image"
    "image/color"
    "testing"
)

func TestSampleColor(t *testing.T) {
    // Create a simple test image
    img := image.NewRGBA(image.Rect(0, 0, 10, 10))
    img.Set(5, 5, color.RGBA{R: 51, G: 102, B: 153, A: 255})

    result, err := SampleColor(img, 5, 5)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result.Hex != "#336699" {
        t.Errorf("expected #336699, got %s", result.Hex)
    }
}
```

### Integration Tests

Create test images in `testdata/` directory:
- `simple_boxes.png` - Known rectangles for shape detection
- `text_sample.png` - Known text for OCR testing
- `color_palette.png` - Known colors for palette extraction

---

## Usage Examples for Claude

Once this MCP is available, Claude can use it like this:

```
# Examine a diagram
1. image_dimensions("/images/project/diagram.png")
   → {width: 800, height: 600}

2. image_dominant_colors("/images/project/diagram.png", count=5)
   → [#FFFFFF (45%), #336699 (22%), #333333 (16%), ...]

3. image_ocr_full("/images/project/diagram.png")
   → Full text: "Input → Process → Output"
   → Regions: [{text: "Input", bounds: {50,100,120,125}}, ...]

4. image_detect_rectangles("/images/project/diagram.png")
   → [{bounds: {50,80,150,140}, fill: #336699}, ...]

5. image_crop("/images/project/diagram.png", x1=300, y1=250, x2=450, y2=350, scale=2.0)
   → Zoomed image of the center region for detailed examination

6. image_sample_color("/images/project/diagram.png", x=125, y=110)
   → {hex: "#336699", rgb: {r:51, g:102, b:153}}
```

This workflow transforms guesswork into precision, enabling accurate TikZ recreation.

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | Initial | Core image operations, OCR, basic shape detection |

---

## Build Instructions Summary

```bash
# Clone the empty repository
git clone <your-repo-url>
cd image-mcp

# Initialize Go module
go mod init github.com/yourusername/image-mcp

# Create the directory structure
mkdir -p cmd/image-mcp internal/{server,imaging,detection,ocr}

# Implement according to this specification...

# Build and test
make build
make test
make docker

# Configure Claude Code to use it
# Edit ~/.claude/mcp.json to add the image-mcp server
```

---

This specification is complete and self-contained. A Claude instance with access to this document can build the entire MCP from scratch.
