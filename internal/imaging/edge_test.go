package imaging

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestEdgeDetect(t *testing.T) {
	// Create an image with a clear edge (black rectangle on white background)
	img := createEdgeTestImage(100, 100)

	result, err := EdgeDetect(img, 50, 150)
	if err != nil {
		t.Fatalf("EdgeDetect failed: %v", err)
	}

	if result.Width != 100 || result.Height != 100 {
		t.Errorf("dimensions: got %dx%d, want 100x100", result.Width, result.Height)
	}

	if result.MimeType != "image/png" {
		t.Errorf("MimeType: got %s, want image/png", result.MimeType)
	}

	// Verify base64 can be decoded
	decoded, err := base64.StdEncoding.DecodeString(result.ImageBase64)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}

	// Verify it's a valid PNG
	edgeImg, err := png.Decode(strings.NewReader(string(decoded)))
	if err != nil {
		t.Fatalf("failed to decode PNG: %v", err)
	}

	// Verify the edge image has the same bounds
	if edgeImg.Bounds().Dx() != 100 || edgeImg.Bounds().Dy() != 100 {
		t.Errorf("decoded image dimensions: got %dx%d, want 100x100",
			edgeImg.Bounds().Dx(), edgeImg.Bounds().Dy())
	}
}

func TestEdgeDetect_DifferentThresholds(t *testing.T) {
	img := createEdgeTestImage(50, 50)

	tests := []struct {
		name         string
		low, high    int
	}{
		{"low thresholds", 10, 50},
		{"medium thresholds", 50, 150},
		{"high thresholds", 100, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EdgeDetect(img, tt.low, tt.high)
			if err != nil {
				t.Fatalf("EdgeDetect failed: %v", err)
			}

			// Just verify it produces valid output
			if result.ImageBase64 == "" {
				t.Error("ImageBase64 is empty")
			}
		})
	}
}

func TestEdgeDetect_UniformImage(t *testing.T) {
	// Uniform image should have no edges
	img := createInMemoryImage(50, 50, color.RGBA{128, 128, 128, 255})

	result, err := EdgeDetect(img, 50, 150)
	if err != nil {
		t.Fatalf("EdgeDetect failed: %v", err)
	}

	// Decode and verify mostly black (no edges)
	decoded, _ := base64.StdEncoding.DecodeString(result.ImageBase64)
	edgeImg, _ := png.Decode(strings.NewReader(string(decoded)))

	// Sample center - should be black (no edge in uniform image)
	r, g, b, _ := edgeImg.At(25, 25).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("uniform image should have no edges at center, got (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func TestEdgeDetect_StrongEdge(t *testing.T) {
	// Create image with strong contrast edge
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if x < 50 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}

	result, err := EdgeDetect(img, 50, 150)
	if err != nil {
		t.Fatalf("EdgeDetect failed: %v", err)
	}

	// Decode result
	decoded, _ := base64.StdEncoding.DecodeString(result.ImageBase64)
	edgeImg, _ := png.Decode(strings.NewReader(string(decoded)))

	// Check near the edge (around x=50)
	// The edge should be detected around x=50
	edgeFound := false
	for x := 48; x <= 52; x++ {
		r, _, _, _ := edgeImg.At(x, 50).RGBA()
		if r > 0 {
			edgeFound = true
			break
		}
	}

	if !edgeFound {
		t.Error("strong vertical edge was not detected")
	}
}

func TestEdgeDetect_SmallImage(t *testing.T) {
	// Very small image (edge cases for convolution)
	img := createInMemoryImage(5, 5, color.RGBA{128, 128, 128, 255})

	result, err := EdgeDetect(img, 50, 150)
	if err != nil {
		t.Fatalf("EdgeDetect failed: %v", err)
	}

	if result.Width != 5 || result.Height != 5 {
		t.Errorf("dimensions: got %dx%d, want 5x5", result.Width, result.Height)
	}
}

func TestGaussianBlur(t *testing.T) {
	// Create a simple test image as float array
	width, height := 10, 10
	img := make([][]float64, height)
	for y := 0; y < height; y++ {
		img[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			img[y][x] = 0.5 // uniform gray
		}
	}

	blurred := gaussianBlur(img, width, height)

	// Uniform image should remain uniform after blur
	for y := 2; y < height-2; y++ {
		for x := 2; x < width-2; x++ {
			if absFloat(blurred[y][x]-0.5) > 0.01 {
				t.Errorf("blurred[%d][%d]: got %.3f, want ~0.5", y, x, blurred[y][x])
			}
		}
	}
}

func TestGaussianBlur_WithSpot(t *testing.T) {
	// Create image with a bright spot
	width, height := 11, 11
	img := make([][]float64, height)
	for y := 0; y < height; y++ {
		img[y] = make([]float64, width)
	}
	img[5][5] = 1.0 // bright spot in center

	blurred := gaussianBlur(img, width, height)

	// Center should be reduced (spread to neighbors)
	if blurred[5][5] >= 1.0 {
		t.Error("bright spot should be reduced after blur")
	}

	// Neighbors should receive some of the brightness
	if blurred[5][4] == 0 || blurred[5][6] == 0 || blurred[4][5] == 0 || blurred[6][5] == 0 {
		t.Error("neighbors should receive some brightness from blur")
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		val, min, max, want int
	}{
		{5, 0, 10, 5},    // within range
		{-1, 0, 10, 0},   // below min
		{15, 0, 10, 10},  // above max
		{0, 0, 10, 0},    // at min
		{10, 0, 10, 10},  // at max
	}

	for _, tt := range tests {
		got := clamp(tt.val, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clamp(%d, %d, %d): got %d, want %d",
				tt.val, tt.min, tt.max, got, tt.want)
		}
	}
}

// Helper functions

// createEdgeTestImage creates an image with a black rectangle on white background
// to create clear edges for testing
func createEdgeTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Black rectangle in center (creates 4 edges)
	for y := height/4; y < 3*height/4; y++ {
		for x := width/4; x < 3*width/4; x++ {
			img.Set(x, y, color.Black)
		}
	}

	return img
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
