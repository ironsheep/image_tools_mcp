package imaging

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/disintegration/imaging"
)

// CropResult contains a cropped image encoded as base64 PNG.
//
// This result type is designed for transmitting cropped images through
// JSON-based protocols like MCP, where binary data must be encoded as text.
type CropResult struct {
	// Width of the cropped (and optionally scaled) image in pixels.
	Width int `json:"width"`

	// Height of the cropped (and optionally scaled) image in pixels.
	Height int `json:"height"`

	// ImageBase64 is the cropped image encoded as base64 PNG.
	// Decode with base64.StdEncoding.DecodeString() to get raw PNG bytes.
	ImageBase64 string `json:"image_base64"`

	// MimeType is always "image/png" for crop results.
	MimeType string `json:"mime_type"`
}

// Crop extracts a rectangular region from an image and returns it as base64 PNG.
//
// This function is useful for zooming into specific areas of an image for
// detailed examination or for extracting sub-images for further processing.
//
// Parameters:
//   - img: Source image to crop from.
//   - x1, y1: Top-left corner of the crop region (inclusive).
//   - x2, y2: Bottom-right corner of the crop region (exclusive).
//   - scale: Scaling factor to apply after cropping. Use 1.0 for no scaling,
//     2.0 to double the size, 0.5 to halve it, etc. Must be > 0.
//
// Returns:
//   - *CropResult: The cropped image data with dimensions and base64 encoding.
//   - error: Non-nil if:
//   - Crop region is outside image bounds
//   - Crop region is invalid (x1 >= x2 or y1 >= y2)
//   - PNG encoding fails
//
// # Coordinate System
//
// Coordinates are 0-based with (0,0) at top-left:
//   - x1, y1 specify the inclusive top-left corner
//   - x2, y2 specify the exclusive bottom-right corner
//   - The cropped width is (x2 - x1), height is (y2 - y1)
//
// # Scaling
//
// When scale != 1.0, the cropped region is resized using Lanczos interpolation,
// which provides high-quality results for both upscaling and downscaling.
// The final dimensions are:
//
//	finalWidth = int(cropWidth * scale)
//	finalHeight = int(cropHeight * scale)
func Crop(img image.Image, x1, y1, x2, y2 int, scale float64) (*CropResult, error) {
	bounds := img.Bounds()

	// Validate coordinates
	if x1 < bounds.Min.X || y1 < bounds.Min.Y || x2 > bounds.Max.X || y2 > bounds.Max.Y {
		return nil, fmt.Errorf("crop region (%d,%d)-(%d,%d) outside image bounds (%d,%d)-(%d,%d)",
			x1, y1, x2, y2, bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)
	}
	if x1 >= x2 || y1 >= y2 {
		return nil, fmt.Errorf("invalid crop region: x1 must be < x2, y1 must be < y2")
	}

	cropped := imaging.Crop(img, image.Rect(x1, y1, x2, y2))

	if scale != 1.0 && scale > 0 {
		newWidth := int(float64(cropped.Bounds().Dx()) * scale)
		newHeight := int(float64(cropped.Bounds().Dy()) * scale)
		cropped = imaging.Resize(cropped, newWidth, newHeight, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, cropped); err != nil {
		return nil, fmt.Errorf("failed to encode cropped image: %w", err)
	}

	return &CropResult{
		Width:       cropped.Bounds().Dx(),
		Height:      cropped.Bounds().Dy(),
		ImageBase64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		MimeType:    "image/png",
	}, nil
}

// CropQuadrant extracts a named region from an image using predefined positions.
//
// This function provides a convenient way to extract common image regions without
// calculating exact coordinates. It's useful for quick navigation or when analyzing
// images with predictable layouts.
//
// Parameters:
//   - img: Source image to crop from.
//   - region: Named region to extract. Must be one of:
//   - "top-left": Upper-left quadrant (0,0 to midX,midY)
//   - "top-right": Upper-right quadrant (midX,0 to width,midY)
//   - "bottom-left": Lower-left quadrant (0,midY to midX,height)
//   - "bottom-right": Lower-right quadrant (midX,midY to width,height)
//   - "top-half": Upper half (0,0 to width,midY)
//   - "bottom-half": Lower half (0,midY to width,height)
//   - "left-half": Left half (0,0 to midX,height)
//   - "right-half": Right half (midX,0 to width,height)
//   - "center": Center 50% (quarter margins on all sides)
//   - scale: Scaling factor to apply after cropping (same as Crop).
//
// Returns:
//   - *CropResult: The cropped image data.
//   - error: Non-nil if region name is invalid or cropping fails.
//
// # Region Definitions
//
// For an image of size W x H:
//   - midX = W / 2 (integer division)
//   - midY = H / 2 (integer division)
//   - "center" uses 25% margins: (W/4, H/4) to (W-W/4, H-H/4)
//
// Due to integer division, odd-sized images may have slightly asymmetric regions.
func CropQuadrant(img image.Image, region string, scale float64) (*CropResult, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	midX := w / 2
	midY := h / 2

	var x1, y1, x2, y2 int

	switch region {
	case "top-left":
		x1, y1, x2, y2 = 0, 0, midX, midY
	case "top-right":
		x1, y1, x2, y2 = midX, 0, w, midY
	case "bottom-left":
		x1, y1, x2, y2 = 0, midY, midX, h
	case "bottom-right":
		x1, y1, x2, y2 = midX, midY, w, h
	case "top-half":
		x1, y1, x2, y2 = 0, 0, w, midY
	case "bottom-half":
		x1, y1, x2, y2 = 0, midY, w, h
	case "left-half":
		x1, y1, x2, y2 = 0, 0, midX, h
	case "right-half":
		x1, y1, x2, y2 = midX, 0, w, h
	case "center":
		// Center 50% of the image
		qW := w / 4
		qH := h / 4
		x1, y1, x2, y2 = qW, qH, w-qW, h-qH
	default:
		return nil, fmt.Errorf("unknown region: %s", region)
	}

	return Crop(img, x1, y1, x2, y2, scale)
}
