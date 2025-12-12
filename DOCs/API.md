# API Reference

Complete reference for all 18 Image Tools MCP Server tools.

## Table of Contents

- [Basic Image Information](#basic-image-information)
  - [image_load](#image_load)
  - [image_dimensions](#image_dimensions)
- [Region Operations](#region-operations)
  - [image_crop](#image_crop)
  - [image_crop_quadrant](#image_crop_quadrant)
- [Color Operations](#color-operations)
  - [image_sample_color](#image_sample_color)
  - [image_sample_colors_multi](#image_sample_colors_multi)
  - [image_dominant_colors](#image_dominant_colors)
- [Measurement Operations](#measurement-operations)
  - [image_measure_distance](#image_measure_distance)
  - [image_grid_overlay](#image_grid_overlay)
- [OCR Operations](#ocr-operations)
  - [image_ocr_full](#image_ocr_full)
  - [image_ocr_region](#image_ocr_region)
  - [image_detect_text_regions](#image_detect_text_regions)
- [Shape Detection](#shape-detection)
  - [image_detect_rectangles](#image_detect_rectangles)
  - [image_detect_lines](#image_detect_lines)
  - [image_detect_circles](#image_detect_circles)
  - [image_edge_detect](#image_edge_detect)
- [Analysis Helpers](#analysis-helpers)
  - [image_check_alignment](#image_check_alignment)
  - [image_compare_regions](#image_compare_regions)

---

## Basic Image Information

### image_load

Load an image file and return its dimensions and format.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Absolute path to the image file |

**Returns:**

```json
{
  "width": 800,
  "height": 600,
  "format": "png"
}
```

**Example:**

```json
{
  "name": "image_load",
  "arguments": {
    "path": "/path/to/diagram.png"
  }
}
```

---

### image_dimensions

Get the width and height of an image file.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Absolute path to the image file |

**Returns:**

```json
{
  "width": 800,
  "height": 600
}
```

---

## Region Operations

### image_crop

Crop a rectangular region from an image and return it as base64-encoded PNG.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `x1` | integer | Yes | - | Left edge X coordinate (0-based) |
| `y1` | integer | Yes | - | Top edge Y coordinate (0-based) |
| `x2` | integer | Yes | - | Right edge X coordinate (exclusive) |
| `y2` | integer | Yes | - | Bottom edge Y coordinate (exclusive) |
| `scale` | number | No | 1.0 | Scale factor (e.g., 2.0 to double size) |

**Returns:**

```json
{
  "width": 200,
  "height": 150,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA..."
}
```

**Example:**

```json
{
  "name": "image_crop",
  "arguments": {
    "path": "/path/to/image.png",
    "x1": 100,
    "y1": 50,
    "x2": 300,
    "y2": 200,
    "scale": 1.5
  }
}
```

---

### image_crop_quadrant

Crop a named region of the image.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `region` | string | Yes | - | Named region (see below) |
| `scale` | number | No | 1.0 | Scale factor |

**Valid regions:**

- `top-left`, `top-right`, `bottom-left`, `bottom-right` - Quarter regions
- `top-half`, `bottom-half`, `left-half`, `right-half` - Half regions
- `center` - Center region (50% of each dimension)

**Returns:**

Same as `image_crop`.

---

## Color Operations

### image_sample_color

Get the exact color value at a specific pixel coordinate.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Absolute path to the image file |
| `x` | integer | Yes | X coordinate (0-based, from left) |
| `y` | integer | Yes | Y coordinate (0-based, from top) |

**Returns:**

```json
{
  "hex": "#FF5733",
  "rgb": {"r": 255, "g": 87, "b": 51},
  "rgba": {"r": 255, "g": 87, "b": 51, "a": 255},
  "hsl": {"h": 11, "s": 100, "l": 60}
}
```

---

### image_sample_colors_multi

Get color values at multiple pixel coordinates in a single call.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Absolute path to the image file |
| `points` | array | Yes | Array of point objects |

**Point object:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `x` | integer | Yes | X coordinate |
| `y` | integer | Yes | Y coordinate |
| `label` | string | No | Optional label for this point |

**Returns:**

```json
{
  "samples": [
    {
      "label": "background",
      "x": 10,
      "y": 10,
      "hex": "#FFFFFF",
      "rgb": {"r": 255, "g": 255, "b": 255}
    },
    {
      "label": "text",
      "x": 100,
      "y": 50,
      "hex": "#000000",
      "rgb": {"r": 0, "g": 0, "b": 0}
    }
  ]
}
```

---

### image_dominant_colors

Analyze an image and return the N most dominant colors (color palette extraction).

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `count` | integer | No | 5 | Number of dominant colors to return |
| `region` | object | No | - | Optional region to analyze |

**Region object (optional):**

| Name | Type | Required |
|------|------|----------|
| `x1` | integer | Yes |
| `y1` | integer | Yes |
| `x2` | integer | Yes |
| `y2` | integer | Yes |

**Returns:**

```json
{
  "colors": [
    {"hex": "#FFFFFF", "percentage": 45.2, "rgb": {"r": 255, "g": 255, "b": 255}},
    {"hex": "#000000", "percentage": 30.1, "rgb": {"r": 0, "g": 0, "b": 0}},
    {"hex": "#FF0000", "percentage": 15.5, "rgb": {"r": 255, "g": 0, "b": 0}}
  ]
}
```

---

## Measurement Operations

### image_measure_distance

Measure the distance in pixels between two points.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Absolute path to the image file |
| `x1` | integer | Yes | First point X |
| `y1` | integer | Yes | First point Y |
| `x2` | integer | Yes | Second point X |
| `y2` | integer | Yes | Second point Y |

**Returns:**

```json
{
  "distance_pixels": 141.42,
  "delta_x": 100,
  "delta_y": 100,
  "percent_of_width": 12.5,
  "percent_of_height": 16.7
}
```

---

### image_grid_overlay

Return a version of the image with a coordinate grid overlay.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `grid_spacing` | integer | No | 50 | Pixels between grid lines |
| `show_coordinates` | boolean | No | true | Label grid intersections |
| `grid_color` | string | No | #FF000080 | Grid color as hex (with optional alpha) |

**Returns:**

```json
{
  "width": 800,
  "height": 600,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA..."
}
```

---

## OCR Operations

> **Note:** OCR tools require Tesseract on most platforms. See [INSTALL.md](../INSTALL.md) for setup instructions. Linux AMD64 binaries include embedded OCR.

### image_ocr_full

Extract all text from the image using OCR.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `language` | string | No | eng | OCR language code |

**Returns:**

```json
{
  "full_text": "Hello World\nThis is a test",
  "regions": [
    {
      "text": "Hello",
      "confidence": 0.95,
      "bounds": {"x1": 10, "y1": 20, "x2": 80, "y2": 45}
    },
    {
      "text": "World",
      "confidence": 0.93,
      "bounds": {"x1": 90, "y1": 20, "x2": 160, "y2": 45}
    }
  ]
}
```

---

### image_ocr_region

Extract text from a specific rectangular region of the image.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `x1` | integer | Yes | - | Left edge |
| `y1` | integer | Yes | - | Top edge |
| `x2` | integer | Yes | - | Right edge |
| `y2` | integer | Yes | - | Bottom edge |
| `language` | string | No | eng | OCR language code |

**Returns:**

Same structure as `image_ocr_full`.

---

### image_detect_text_regions

Detect regions containing text without performing full OCR.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `min_confidence` | number | No | 0.5 | Minimum confidence (0-1) |

**Returns:**

```json
{
  "regions": [
    {"bounds": {"x1": 10, "y1": 20, "x2": 200, "y2": 50}, "confidence": 0.92},
    {"bounds": {"x1": 10, "y1": 60, "x2": 180, "y2": 90}, "confidence": 0.88}
  ],
  "count": 2
}
```

---

## Shape Detection

### image_detect_rectangles

Detect rectangular shapes in the image.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `min_area` | integer | No | 100 | Minimum area in pixels |
| `tolerance` | number | No | 0.9 | How rectangular (0-1) |

**Returns:**

```json
{
  "rectangles": [
    {
      "x": 50,
      "y": 100,
      "width": 200,
      "height": 150,
      "area": 30000,
      "center": {"x": 150, "y": 175}
    }
  ],
  "count": 1
}
```

---

### image_detect_lines

Detect line segments in the image.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `min_length` | integer | No | 20 | Minimum line length in pixels |
| `detect_arrows` | boolean | No | true | Detect arrow heads |

**Returns:**

```json
{
  "lines": [
    {
      "x1": 100,
      "y1": 50,
      "x2": 300,
      "y2": 50,
      "length": 200,
      "angle": 0,
      "has_arrow_start": false,
      "has_arrow_end": true,
      "color": "#000000",
      "thickness": 2
    }
  ],
  "count": 1
}
```

---

### image_detect_circles

Detect circular shapes in the image.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `min_radius` | integer | No | 5 | Minimum radius in pixels |
| `max_radius` | integer | No | 500 | Maximum radius in pixels |

**Returns:**

```json
{
  "circles": [
    {
      "center": {"x": 200, "y": 150},
      "radius": 50,
      "diameter": 100
    }
  ],
  "count": 1
}
```

---

### image_edge_detect

Return an edge-detected version of the image using Canny edge detection.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `threshold_low` | integer | No | 50 | Low threshold for Canny |
| `threshold_high` | integer | No | 150 | High threshold for Canny |

**Returns:**

```json
{
  "width": 800,
  "height": 600,
  "image_base64": "iVBORw0KGgoAAAANSUhEUgAA..."
}
```

---

## Analysis Helpers

### image_check_alignment

Check if multiple points are horizontally or vertically aligned.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to the image file |
| `points` | array | Yes | - | Array of point objects |
| `tolerance` | integer | No | 5 | Pixel tolerance |

**Point object:**

| Name | Type | Required |
|------|------|----------|
| `x` | integer | Yes |
| `y` | integer | Yes |

**Returns:**

```json
{
  "horizontally_aligned": true,
  "vertically_aligned": false,
  "average_x": 150,
  "average_y": 100,
  "max_x_deviation": 3,
  "max_y_deviation": 45
}
```

---

### image_compare_regions

Compare two regions of an image for similarity.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Absolute path to the image file |
| `region1` | object | Yes | First region bounds |
| `region2` | object | Yes | Second region bounds |

**Region object:**

| Name | Type | Required |
|------|------|----------|
| `x1` | integer | Yes |
| `y1` | integer | Yes |
| `x2` | integer | Yes |
| `y2` | integer | Yes |

**Returns:**

```json
{
  "similarity": 0.95,
  "identical": false,
  "region1_size": {"width": 100, "height": 80},
  "region2_size": {"width": 100, "height": 80}
}
```

---

## Coordinate System

All coordinates in this API use:

- **Origin**: Top-left corner of the image is (0, 0)
- **X-axis**: Increases to the right
- **Y-axis**: Increases downward
- **Exclusive bounds**: For regions, `x2` and `y2` are exclusive (not included in the region)

```
(0,0) ─────────────────────► X
  │
  │    Image Area
  │
  │
  ▼
  Y
```

## Error Handling

All tools return errors in standard MCP format:

```json
{
  "error": {
    "code": -32602,
    "message": "Invalid params: path is required"
  }
}
```

Common error codes:

- `-32602`: Invalid parameters
- `-32603`: Internal error (e.g., file not found, invalid image)
