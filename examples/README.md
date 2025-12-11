# Example Images and Usage

This directory contains example images and expected outputs for testing and demonstration purposes.

## Test Images

### simple_diagram.png
A basic flowchart with three boxes connected by arrows.

**Expected tool outputs:**

```bash
# Get dimensions
image_dimensions("examples/simple_diagram.png")
# → {"width": 600, "height": 400}

# Detect rectangles
image_detect_rectangles("examples/simple_diagram.png")
# → 3 rectangles with positions and fill colors

# Detect lines with arrows
image_detect_lines("examples/simple_diagram.png", detect_arrows=true)
# → 2 lines, both with arrow_end=true

# OCR text extraction
image_ocr_full("examples/simple_diagram.png")
# → "Input", "Process", "Output" with bounding boxes
```

### color_palette.png
An image with known color swatches for testing color sampling accuracy.

**Color positions:**
| Position | Expected Color |
|----------|---------------|
| (25, 25) | #FF0000 (Red) |
| (75, 25) | #00FF00 (Green) |
| (125, 25) | #0000FF (Blue) |
| (25, 75) | #FFFF00 (Yellow) |
| (75, 75) | #FF00FF (Magenta) |
| (125, 75) | #00FFFF (Cyan) |

```bash
# Sample multiple colors
image_sample_colors_multi("examples/color_palette.png", points=[
  {"x": 25, "y": 25, "label": "red"},
  {"x": 75, "y": 25, "label": "green"},
  {"x": 125, "y": 25, "label": "blue"}
])
```

### text_sample.png
An image with various text sizes and fonts for OCR testing.

**Expected text:**
- "Heading" (large, bold)
- "Subheading" (medium)
- "Body text paragraph" (small)
- "12345" (numbers)

### shapes_test.png
An image containing various geometric shapes for shape detection testing.

**Expected shapes:**
- 3 rectangles (various sizes)
- 2 circles (different radii)
- 4 lines (horizontal, vertical, diagonal)

## Creating Test Images

You can create test images programmatically or use image editing software. For best results:

1. **Use solid colors** - Gradients can affect color sampling accuracy
2. **Clear backgrounds** - White or transparent backgrounds work best for shape detection
3. **High contrast** - Better edge detection with high contrast between elements
4. **Standard fonts** - Sans-serif fonts work best for OCR

## Running Examples

### Using Docker

```bash
# Mount the examples directory
docker run --rm -i \
  -v $(pwd)/examples:/images/examples:ro \
  ghcr.io/ironsheep/image-tools-mcp:latest
```

### Using Binary

```bash
# Direct path access
./image-tools-mcp
# Then send JSON-RPC requests with paths to examples/
```

## Example Workflows

### Workflow 1: Diagram Analysis

```
1. Load image and get metadata
   → image_load("/images/examples/simple_diagram.png")

2. Add grid overlay for reference
   → image_grid_overlay(..., grid_spacing=50)

3. Detect all shapes
   → image_detect_rectangles(...)
   → image_detect_lines(...)

4. Extract text labels
   → image_ocr_full(...)

5. Sample specific colors
   → image_sample_colors_multi(...)
```

### Workflow 2: Color Extraction

```
1. Get dominant colors
   → image_dominant_colors(..., count=10)

2. Sample specific positions for accuracy
   → image_sample_color(..., x=100, y=150)

3. Compare to expected values
```

### Workflow 3: Text Extraction

```
1. Detect text regions first (faster)
   → image_detect_text_regions(...)

2. Run OCR on specific regions
   → image_ocr_region(..., x1, y1, x2, y2)

3. Or extract all text at once
   → image_ocr_full(...)
```

## Generating Expected Outputs

To generate expected outputs for new test images:

```bash
# Create a test script
cat << 'EOF' > test_image.sh
#!/bin/bash
IMAGE="/images/examples/your_image.png"

# Dimensions
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"image_dimensions","arguments":{"path":"'$IMAGE'"}}}' | ./image-tools-mcp

# Rectangles
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"image_detect_rectangles","arguments":{"path":"'$IMAGE'"}}}' | ./image-tools-mcp

# OCR
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"image_ocr_full","arguments":{"path":"'$IMAGE'"}}}' | ./image-tools-mcp
EOF
```

## Notes

- All coordinates are 0-based with origin at top-left
- Colors are returned in hex, RGB, RGBA, and HSL formats
- Cropped images are returned as base64-encoded PNG
- OCR requires Tesseract with appropriate language data
