package imaging

import (
	"image"
	"image/color"
	"testing"
)

// createInMemoryImage creates an in-memory test image
func createInMemoryImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// createPatternImage creates an image with different colors in each quadrant
func createPatternImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var c color.Color
			if x < width/2 && y < height/2 {
				c = color.RGBA{255, 0, 0, 255} // Red top-left
			} else if x >= width/2 && y < height/2 {
				c = color.RGBA{0, 255, 0, 255} // Green top-right
			} else if x < width/2 && y >= height/2 {
				c = color.RGBA{0, 0, 255, 255} // Blue bottom-left
			} else {
				c = color.RGBA{255, 255, 255, 255} // White bottom-right
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func TestSampleColor(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 128, 64, 255})

	result, err := SampleColor(img, 50, 50)
	if err != nil {
		t.Fatalf("SampleColor failed: %v", err)
	}

	// Check hex
	if result.Hex != "#FF8040" {
		t.Errorf("Hex: got %s, want #FF8040", result.Hex)
	}

	// Check RGB
	if result.RGB.R != 255 || result.RGB.G != 128 || result.RGB.B != 64 {
		t.Errorf("RGB: got (%d,%d,%d), want (255,128,64)", result.RGB.R, result.RGB.G, result.RGB.B)
	}

	// Check RGBA
	if result.RGBA.R != 255 || result.RGBA.G != 128 || result.RGBA.B != 64 || result.RGBA.A != 255 {
		t.Errorf("RGBA: got (%d,%d,%d,%d), want (255,128,64,255)",
			result.RGBA.R, result.RGBA.G, result.RGBA.B, result.RGBA.A)
	}
}

func TestSampleColor_KnownColors(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		wantHex  string
		wantHue  int // approximate
	}{
		{"pure red", color.RGBA{255, 0, 0, 255}, "#FF0000", 0},
		{"pure green", color.RGBA{0, 255, 0, 255}, "#00FF00", 120},
		{"pure blue", color.RGBA{0, 0, 255, 255}, "#0000FF", 240},
		{"white", color.RGBA{255, 255, 255, 255}, "#FFFFFF", 0},
		{"black", color.RGBA{0, 0, 0, 255}, "#000000", 0},
		{"gray", color.RGBA{128, 128, 128, 255}, "#808080", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := createInMemoryImage(10, 10, tt.color)
			result, err := SampleColor(img, 5, 5)
			if err != nil {
				t.Fatalf("SampleColor failed: %v", err)
			}

			if result.Hex != tt.wantHex {
				t.Errorf("Hex: got %s, want %s", result.Hex, tt.wantHex)
			}
		})
	}
}

func TestSampleColor_OutOfBounds(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	tests := []struct {
		name string
		x, y int
	}{
		{"negative x", -1, 50},
		{"negative y", 50, -1},
		{"x too large", 100, 50},
		{"y too large", 50, 100},
		{"both too large", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SampleColor(img, tt.x, tt.y)
			if err == nil {
				t.Error("SampleColor should fail for out-of-bounds coordinates")
			}
		})
	}
}

func TestSampleColor_EdgeCoordinates(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	// Test edge coordinates (should succeed)
	tests := []struct {
		name string
		x, y int
	}{
		{"top-left", 0, 0},
		{"top-right", 99, 0},
		{"bottom-left", 0, 99},
		{"bottom-right", 99, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SampleColor(img, tt.x, tt.y)
			if err != nil {
				t.Errorf("SampleColor failed for valid edge coordinate (%d,%d): %v", tt.x, tt.y, err)
			}
		})
	}
}

func TestSampleColorsMulti(t *testing.T) {
	img := createPatternImage(100, 100)

	points := []LabeledPoint{
		{X: 25, Y: 25, Label: "red"},
		{X: 75, Y: 25, Label: "green"},
		{X: 25, Y: 75, Label: "blue"},
		{X: 75, Y: 75, Label: "white"},
	}

	result, err := SampleColorsMulti(img, points)
	if err != nil {
		t.Fatalf("SampleColorsMulti failed: %v", err)
	}

	if len(result.Samples) != 4 {
		t.Fatalf("expected 4 samples, got %d", len(result.Samples))
	}

	// Check labels preserved
	for i, sample := range result.Samples {
		if sample.Label != points[i].Label {
			t.Errorf("sample %d label: got %s, want %s", i, sample.Label, points[i].Label)
		}
	}

	// Check colors
	expectedHex := []string{"#FF0000", "#00FF00", "#0000FF", "#FFFFFF"}
	for i, sample := range result.Samples {
		if sample.Color.Hex != expectedHex[i] {
			t.Errorf("sample %d (%s) hex: got %s, want %s",
				i, sample.Label, sample.Color.Hex, expectedHex[i])
		}
	}
}

func TestSampleColorsMulti_EmptyPoints(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	result, err := SampleColorsMulti(img, []LabeledPoint{})
	if err != nil {
		t.Fatalf("SampleColorsMulti failed: %v", err)
	}

	if len(result.Samples) != 0 {
		t.Errorf("expected 0 samples, got %d", len(result.Samples))
	}
}

func TestSampleColorsMulti_OutOfBounds(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	points := []LabeledPoint{
		{X: 50, Y: 50, Label: "valid"},
		{X: 200, Y: 50, Label: "invalid"},
	}

	_, err := SampleColorsMulti(img, points)
	if err == nil {
		t.Error("SampleColorsMulti should fail when any point is out of bounds")
	}
}

func TestDominantColors(t *testing.T) {
	// Create an image with mostly red, some green
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if x < 80 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255}) // 80% red
			} else {
				img.Set(x, y, color.RGBA{0, 255, 0, 255}) // 20% green
			}
		}
	}

	result, err := DominantColors(img, 5, nil)
	if err != nil {
		t.Fatalf("DominantColors failed: %v", err)
	}

	if len(result.Colors) == 0 {
		t.Fatal("expected at least one color")
	}

	// Red should be the most dominant
	// Note: colors are quantized, so exact values may differ
	if result.Colors[0].Percentage < 50 {
		t.Errorf("dominant color percentage too low: %f", result.Colors[0].Percentage)
	}
}

func TestDominantColors_WithRegion(t *testing.T) {
	img := createPatternImage(100, 100)

	// Sample only the top-left quadrant (red)
	region := &Region{X1: 0, Y1: 0, X2: 50, Y2: 50}
	result, err := DominantColors(img, 5, region)
	if err != nil {
		t.Fatalf("DominantColors with region failed: %v", err)
	}

	if len(result.Colors) == 0 {
		t.Fatal("expected at least one color")
	}

	// Should be predominantly red (quantized)
	// The quantized red is #F00000 (255/16*16 = 240 -> F0)
	if result.Colors[0].Percentage < 90 {
		t.Errorf("expected red to dominate in top-left region, got %f%%", result.Colors[0].Percentage)
	}
}

func TestDominantColors_SingleColor(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := DominantColors(img, 3, nil)
	if err != nil {
		t.Fatalf("DominantColors failed: %v", err)
	}

	// Should have exactly 1 color since image is uniform
	if len(result.Colors) != 1 {
		t.Errorf("expected 1 color for uniform image, got %d", len(result.Colors))
	}

	// That color should be 100%
	if result.Colors[0].Percentage != 100 {
		t.Errorf("expected 100%% for single color, got %f%%", result.Colors[0].Percentage)
	}
}

func TestRgbToHSL(t *testing.T) {
	tests := []struct {
		name     string
		r, g, b  uint8
		wantH    int
		wantS    int
		wantL    int
	}{
		{"red", 255, 0, 0, 0, 100, 50},
		{"green", 0, 255, 0, 120, 100, 50},
		{"blue", 0, 0, 255, 240, 100, 50},
		{"white", 255, 255, 255, 0, 0, 100},
		{"black", 0, 0, 0, 0, 0, 0},
		{"gray", 128, 128, 128, 0, 0, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hsl := rgbToHSL(tt.r, tt.g, tt.b)

			// Allow some tolerance for rounding
			if abs(hsl.H-tt.wantH) > 1 {
				t.Errorf("H: got %d, want %d", hsl.H, tt.wantH)
			}
			if abs(hsl.S-tt.wantS) > 1 {
				t.Errorf("S: got %d, want %d", hsl.S, tt.wantS)
			}
			if abs(hsl.L-tt.wantL) > 1 {
				t.Errorf("L: got %d, want %d", hsl.L, tt.wantL)
			}
		})
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
