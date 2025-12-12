package imaging

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestGridOverlay(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := GridOverlay(img, 25, false, "#FF0000")
	if err != nil {
		t.Fatalf("GridOverlay failed: %v", err)
	}

	if result.Width != 100 || result.Height != 100 {
		t.Errorf("dimensions: got %dx%d, want 100x100", result.Width, result.Height)
	}

	if result.GridSpacing != 25 {
		t.Errorf("GridSpacing: got %d, want 25", result.GridSpacing)
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

func TestGridOverlay_GridLines(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{0, 0, 0, 255})

	result, err := GridOverlay(img, 25, false, "#FF0000FF")
	if err != nil {
		t.Fatalf("GridOverlay failed: %v", err)
	}

	// Decode and verify grid lines
	decoded, _ := base64.StdEncoding.DecodeString(result.ImageBase64)
	gridImg, _ := png.Decode(strings.NewReader(string(decoded)))

	// Check that grid line at x=25 is red
	r, g, b, _ := gridImg.At(25, 50).RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	if r8 != 255 || g8 != 0 || b8 != 0 {
		t.Errorf("grid line color at (25,50): got (%d,%d,%d), want (255,0,0)", r8, g8, b8)
	}

	// Check that non-grid position is still black (background)
	r, g, b, _ = gridImg.At(15, 15).RGBA()
	r8, g8, b8 = uint8(r>>8), uint8(g>>8), uint8(b>>8)

	if r8 != 0 || g8 != 0 || b8 != 0 {
		t.Errorf("non-grid position at (15,15): got (%d,%d,%d), want (0,0,0)", r8, g8, b8)
	}
}

func TestGridOverlay_WithCoordinates(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := GridOverlay(img, 50, true, "#FF0000")
	if err != nil {
		t.Fatalf("GridOverlay failed: %v", err)
	}

	// Just verify it produces valid output (coordinate rendering is complex to verify)
	if result.ImageBase64 == "" {
		t.Error("ImageBase64 is empty")
	}
}

func TestGridOverlay_DifferentSpacings(t *testing.T) {
	img := createInMemoryImage(200, 200, color.RGBA{128, 128, 128, 255})

	tests := []struct {
		spacing int
	}{
		{10},
		{25},
		{50},
		{100},
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.spacing/10)), func(t *testing.T) {
			result, err := GridOverlay(img, tt.spacing, false, "#FF0000")
			if err != nil {
				t.Fatalf("GridOverlay failed: %v", err)
			}

			if result.GridSpacing != tt.spacing {
				t.Errorf("GridSpacing: got %d, want %d", result.GridSpacing, tt.spacing)
			}
		})
	}
}

func TestGridOverlay_InvalidColor(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{128, 128, 128, 255})

	// Should use default color (semi-transparent red)
	result, err := GridOverlay(img, 50, false, "invalid")
	if err != nil {
		t.Fatalf("GridOverlay failed: %v", err)
	}

	// Just verify it produces valid output
	if result.ImageBase64 == "" {
		t.Error("ImageBase64 is empty")
	}
}

func TestGridOverlay_EmptyColor(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{128, 128, 128, 255})

	// Should use default color
	result, err := GridOverlay(img, 50, false, "")
	if err != nil {
		t.Fatalf("GridOverlay failed: %v", err)
	}

	if result.ImageBase64 == "" {
		t.Error("ImageBase64 is empty")
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		hex     string
		wantR   uint8
		wantG   uint8
		wantB   uint8
		wantA   uint8
		wantErr bool
	}{
		{"#FF0000", 255, 0, 0, 255, false},
		{"#00FF00", 0, 255, 0, 255, false},
		{"#0000FF", 0, 0, 255, 255, false},
		{"#FFFFFF", 255, 255, 255, 255, false},
		{"#000000", 0, 0, 0, 255, false},
		{"FF0000", 255, 0, 0, 255, false},      // without #
		{"#FF000080", 255, 0, 0, 128, false},   // with alpha
		{"FF000080", 255, 0, 0, 128, false},    // without # with alpha
		{"", 0, 0, 0, 0, true},                 // empty
		{"#FFF", 0, 0, 0, 0, true},             // invalid length
		{"#GGGGGG", 0, 0, 0, 0, true},          // invalid hex
	}

	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			c, err := parseHexColor(tt.hex)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if c.R != tt.wantR || c.G != tt.wantG || c.B != tt.wantB || c.A != tt.wantA {
				t.Errorf("got (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					c.R, c.G, c.B, c.A, tt.wantR, tt.wantG, tt.wantB, tt.wantA)
			}
		})
	}
}

func TestDrawLabel(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Draw a label
	fg := color.RGBA{255, 255, 255, 255}
	bg := color.RGBA{0, 0, 0, 180}
	drawLabel(img, 10, 10, "50,50", fg, bg)

	// Verify something was drawn (not empty)
	hasWhite := false
	hasBlack := false
	for y := 9; y < 20; y++ {
		for x := 9; x < 40; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 200<<8 {
				hasWhite = true
			}
			if r < 50<<8 {
				hasBlack = true
			}
		}
	}

	if !hasWhite {
		t.Error("label should have white pixels (text)")
	}
	if !hasBlack {
		t.Error("label should have dark pixels (background)")
	}
}

func TestDrawLabel_BoundsCheck(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))

	// Draw near edge - should not panic
	fg := color.RGBA{255, 255, 255, 255}
	bg := color.RGBA{0, 0, 0, 180}

	// These should not panic even if label extends past bounds
	drawLabel(img, 15, 15, "100,100", fg, bg)
	drawLabel(img, 0, 0, "0,0", fg, bg)
	drawLabel(img, -5, -5, "test", fg, bg)
}

func TestDrawLabel_EmptyString(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))

	fg := color.RGBA{255, 255, 255, 255}
	bg := color.RGBA{0, 0, 0, 180}

	// Should not panic on empty string
	drawLabel(img, 10, 10, "", fg, bg)
}

func TestDrawLabel_UnknownChars(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))

	fg := color.RGBA{255, 255, 255, 255}
	bg := color.RGBA{0, 0, 0, 180}

	// Unknown characters should be skipped
	drawLabel(img, 10, 10, "abc123", fg, bg) // 'a', 'b', 'c' are unknown
}
