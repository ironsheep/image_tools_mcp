package imaging

import (
	"encoding/base64"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestCrop(t *testing.T) {
	img := createPatternImage(100, 100)

	result, err := Crop(img, 0, 0, 50, 50, 1.0)
	if err != nil {
		t.Fatalf("Crop failed: %v", err)
	}

	if result.Width != 50 || result.Height != 50 {
		t.Errorf("dimensions: got %dx%d, want 50x50", result.Width, result.Height)
	}

	if result.MimeType != "image/png" {
		t.Errorf("MimeType: got %s, want image/png", result.MimeType)
	}

	// Verify base64 can be decoded
	_, err = base64.StdEncoding.DecodeString(result.ImageBase64)
	if err != nil {
		t.Errorf("failed to decode base64: %v", err)
	}
}

func TestCrop_WithScale(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	// Scale up 2x
	result, err := Crop(img, 0, 0, 50, 50, 2.0)
	if err != nil {
		t.Fatalf("Crop with scale failed: %v", err)
	}

	if result.Width != 100 || result.Height != 100 {
		t.Errorf("scaled dimensions: got %dx%d, want 100x100", result.Width, result.Height)
	}
}

func TestCrop_ScaleDown(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	// Scale down 0.5x
	result, err := Crop(img, 0, 0, 100, 100, 0.5)
	if err != nil {
		t.Fatalf("Crop with scale down failed: %v", err)
	}

	if result.Width != 50 || result.Height != 50 {
		t.Errorf("scaled dimensions: got %dx%d, want 50x50", result.Width, result.Height)
	}
}

func TestCrop_OutOfBounds(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	tests := []struct {
		name           string
		x1, y1, x2, y2 int
	}{
		{"x1 negative", -1, 0, 50, 50},
		{"y1 negative", 0, -1, 50, 50},
		{"x2 too large", 0, 0, 101, 50},
		{"y2 too large", 0, 0, 50, 101},
		{"all out of bounds", -1, -1, 200, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Crop(img, tt.x1, tt.y1, tt.x2, tt.y2, 1.0)
			if err == nil {
				t.Error("Crop should fail for out-of-bounds coordinates")
			}
		})
	}
}

func TestCrop_InvalidRegion(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	tests := []struct {
		name           string
		x1, y1, x2, y2 int
	}{
		{"x1 >= x2", 50, 0, 50, 50},
		{"x1 > x2", 60, 0, 50, 50},
		{"y1 >= y2", 0, 50, 50, 50},
		{"y1 > y2", 0, 60, 50, 50},
		{"zero area", 50, 50, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Crop(img, tt.x1, tt.y1, tt.x2, tt.y2, 1.0)
			if err == nil {
				t.Error("Crop should fail for invalid region")
			}
		})
	}
}

func TestCrop_FullImage(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	result, err := Crop(img, 0, 0, 100, 100, 1.0)
	if err != nil {
		t.Fatalf("Crop full image failed: %v", err)
	}

	if result.Width != 100 || result.Height != 100 {
		t.Errorf("dimensions: got %dx%d, want 100x100", result.Width, result.Height)
	}
}

func TestCrop_VerifyContent(t *testing.T) {
	img := createPatternImage(100, 100)

	// Crop top-left quadrant (should be red)
	result, err := Crop(img, 0, 0, 50, 50, 1.0)
	if err != nil {
		t.Fatalf("Crop failed: %v", err)
	}

	// Decode the result and verify color
	decoded, err := base64.StdEncoding.DecodeString(result.ImageBase64)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}

	croppedImg, err := png.Decode(strings.NewReader(string(decoded)))
	if err != nil {
		t.Fatalf("failed to decode PNG: %v", err)
	}

	// Sample center pixel - should be red
	r, g, b, _ := croppedImg.At(25, 25).RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	if r8 != 255 || g8 != 0 || b8 != 0 {
		t.Errorf("cropped image color: got (%d,%d,%d), want (255,0,0)", r8, g8, b8)
	}
}

func TestCropQuadrant(t *testing.T) {
	img := createPatternImage(100, 100)

	tests := []struct {
		region      string
		wantW, wantH int
	}{
		{"top-left", 50, 50},
		{"top-right", 50, 50},
		{"bottom-left", 50, 50},
		{"bottom-right", 50, 50},
		{"top-half", 100, 50},
		{"bottom-half", 100, 50},
		{"left-half", 50, 100},
		{"right-half", 50, 100},
		{"center", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result, err := CropQuadrant(img, tt.region, 1.0)
			if err != nil {
				t.Fatalf("CropQuadrant(%s) failed: %v", tt.region, err)
			}

			if result.Width != tt.wantW || result.Height != tt.wantH {
				t.Errorf("dimensions: got %dx%d, want %dx%d",
					result.Width, result.Height, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestCropQuadrant_InvalidRegion(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	invalidRegions := []string{"invalid", "TOP-LEFT", "middle", "", "center-left"}

	for _, region := range invalidRegions {
		t.Run(region, func(t *testing.T) {
			_, err := CropQuadrant(img, region, 1.0)
			if err == nil {
				t.Errorf("CropQuadrant should fail for invalid region %q", region)
			}
		})
	}
}

func TestCropQuadrant_WithScale(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	result, err := CropQuadrant(img, "top-left", 2.0)
	if err != nil {
		t.Fatalf("CropQuadrant with scale failed: %v", err)
	}

	// top-left is 50x50, scaled 2x should be 100x100
	if result.Width != 100 || result.Height != 100 {
		t.Errorf("scaled dimensions: got %dx%d, want 100x100", result.Width, result.Height)
	}
}

func TestCropQuadrant_VerifyContent(t *testing.T) {
	img := createPatternImage(100, 100)

	tests := []struct {
		region   string
		wantHex  string
	}{
		{"top-left", "#FF0000"},     // red
		{"top-right", "#00FF00"},    // green
		{"bottom-left", "#0000FF"},  // blue
		{"bottom-right", "#FFFFFF"}, // white
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result, err := CropQuadrant(img, tt.region, 1.0)
			if err != nil {
				t.Fatalf("CropQuadrant(%s) failed: %v", tt.region, err)
			}

			// Decode and check color
			decoded, _ := base64.StdEncoding.DecodeString(result.ImageBase64)
			croppedImg, _ := png.Decode(strings.NewReader(string(decoded)))

			// Sample center
			r, g, b, _ := croppedImg.At(result.Width/2, result.Height/2).RGBA()
			gotHex := "#" + toHex(uint8(r>>8)) + toHex(uint8(g>>8)) + toHex(uint8(b>>8))

			if gotHex != tt.wantHex {
				t.Errorf("color in %s: got %s, want %s", tt.region, gotHex, tt.wantHex)
			}
		})
	}
}

func toHex(b uint8) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0xf]})
}

func TestCropQuadrant_OddDimensions(t *testing.T) {
	// Test with odd dimensions to verify integer division handling
	img := createInMemoryImage(101, 101, color.RGBA{255, 0, 0, 255})

	result, err := CropQuadrant(img, "top-left", 1.0)
	if err != nil {
		t.Fatalf("CropQuadrant with odd dimensions failed: %v", err)
	}

	// 101/2 = 50 (integer division)
	if result.Width != 50 || result.Height != 50 {
		t.Errorf("dimensions: got %dx%d, want 50x50", result.Width, result.Height)
	}
}
