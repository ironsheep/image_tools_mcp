package imaging

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/disintegration/imaging"
)

// CropResult contains the cropped image data
type CropResult struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ImageBase64 string `json:"image_base64"`
	MimeType    string `json:"mime_type"`
}

// Crop extracts a rectangular region from an image
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

// CropQuadrant extracts a named region from an image
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
