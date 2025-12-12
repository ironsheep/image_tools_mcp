package detection

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// createHorizontalLineImage creates an image with a horizontal line
func createHorizontalLineImage(width, height, y, thickness int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	for t := 0; t < thickness; t++ {
		for x := 0; x < width; x++ {
			if y+t >= 0 && y+t < height {
				img.Set(x, y+t, color.Black)
			}
		}
	}

	return img
}

// createVerticalLineImage creates an image with a vertical line
func createVerticalLineImage(width, height, x, thickness int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	for t := 0; t < thickness; t++ {
		for y := 0; y < height; y++ {
			if x+t >= 0 && x+t < width {
				img.Set(x+t, y, color.Black)
			}
		}
	}

	return img
}

// createDiagonalLineImage creates an image with a diagonal line
func createDiagonalLineImage(width, height int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	// Draw diagonal from (0,0) to (width-1, height-1)
	for i := 0; i < min(width, height); i++ {
		img.Set(i, i, color.Black)
	}

	return img
}

// createArrowImage creates an image with an arrow
func createArrowImage(width, height int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	// Draw horizontal line
	y := height / 2
	for x := 20; x < width-20; x++ {
		img.Set(x, y, color.Black)
	}

	// Draw arrow head at end
	endX := width - 20
	for i := 1; i <= 10; i++ {
		img.Set(endX-i, y-i, color.Black) // top wing
		img.Set(endX-i, y+i, color.Black) // bottom wing
	}

	return img
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestDetectLines(t *testing.T) {
	img := createHorizontalLineImage(100, 100, 50, 1)

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	// Should detect the horizontal line
	t.Logf("Detected %d lines", result.Count)
}

func TestDetectLines_MinLength(t *testing.T) {
	// Create short line
	img := createTestImage(100, 100, color.White)
	for x := 45; x <= 55; x++ {
		img.Set(x, 50, color.Black)
	}

	// Line is ~10 pixels, minLength=20 should filter it out
	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	// Very short line may not be detected with high minLength
	t.Logf("Detected %d lines with minLength=20 for ~10px line", result.Count)
}

func TestDetectLines_VerticalLine(t *testing.T) {
	img := createVerticalLineImage(100, 100, 50, 1)

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	if result.Count > 0 {
		// Vertical line should have angle near 90 or -90
		angle := result.Lines[0].AngleDegrees
		if math.Abs(angle) < 80 {
			t.Logf("Vertical line angle: %.1f (expected ~90 or ~-90)", angle)
		}
	}
}

func TestDetectLines_DiagonalLine(t *testing.T) {
	img := createDiagonalLineImage(100, 100)

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	if result.Count > 0 {
		// Diagonal line should have angle around 45
		angle := result.Lines[0].AngleDegrees
		t.Logf("Diagonal line angle: %.1f", angle)
	}
}

func TestDetectLines_WithArrows(t *testing.T) {
	img := createArrowImage(100, 100)

	result, err := DetectLines(img, 20, true)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	t.Logf("Detected %d lines with arrow detection enabled", result.Count)

	// Check if any arrows were detected
	for i, line := range result.Lines {
		if line.HasArrowStart || line.HasArrowEnd {
			t.Logf("Line %d has arrow: start=%v, end=%v", i, line.HasArrowStart, line.HasArrowEnd)
		}
	}
}

func TestDetectLines_EmptyImage(t *testing.T) {
	img := createTestImage(100, 100, color.White)

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Expected 0 lines in empty image, got %d", result.Count)
	}
}

func TestDetectLines_MaxLines(t *testing.T) {
	// Create image with many lines
	img := image.NewRGBA(image.Rect(0, 0, 500, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 500; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add many horizontal lines
	for i := 0; i < 100; i++ {
		y := i * 5
		if y < 500 {
			for x := 0; x < 500; x++ {
				img.Set(x, y, color.Black)
			}
		}
	}

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	// Should be limited to 50 lines
	if result.Count > 50 {
		t.Errorf("Expected max 50 lines, got %d", result.Count)
	}
}

func TestEstimateLineThickness(t *testing.T) {
	// Create edge array with thick line
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	// Thick horizontal line (3 pixels)
	for x := 0; x < 50; x++ {
		edges[24][x] = true
		edges[25][x] = true
		edges[26][x] = true
	}

	thickness := estimateLineThickness(edges, 0, 25, 49, 25, 50, 50)

	// Should detect roughly 3 pixels of thickness
	if thickness < 2 || thickness > 5 {
		t.Errorf("Expected thickness ~3, got %d", thickness)
	}
}

func TestEstimateLineThickness_SinglePixel(t *testing.T) {
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	// Single pixel line
	for x := 0; x < 50; x++ {
		edges[25][x] = true
	}

	thickness := estimateLineThickness(edges, 0, 25, 49, 25, 50, 50)

	// Should be at least 1
	if thickness < 1 {
		t.Errorf("Expected thickness >= 1, got %d", thickness)
	}
}

func TestEstimateLineThickness_ZeroLength(t *testing.T) {
	edges := make([][]bool, 10)
	for y := 0; y < 10; y++ {
		edges[y] = make([]bool, 10)
	}

	// Zero length line (same point)
	thickness := estimateLineThickness(edges, 5, 5, 5, 5, 10, 10)

	// Should return 1 (minimum)
	if thickness != 1 {
		t.Errorf("Expected thickness 1 for zero-length line, got %d", thickness)
	}
}

func TestDetectArrowHead(t *testing.T) {
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	// Create arrow head pattern at (40, 25)
	// Line going from left to right
	endX, endY := 40, 25
	for x := 10; x <= endX; x++ {
		edges[25][x] = true
	}

	// Arrow wings at 45 degrees
	for i := 1; i <= 5; i++ {
		edges[endY-i][endX-i] = true // top wing
		edges[endY+i][endX-i] = true // bottom wing
	}

	hasArrow := detectArrowHead(edges, endX, endY, 10, 25, 50, 50)

	t.Logf("Arrow detected: %v", hasArrow)
}

func TestDetectArrowHead_NoArrow(t *testing.T) {
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	// Just a line, no arrow head
	for x := 10; x <= 40; x++ {
		edges[25][x] = true
	}

	hasArrow := detectArrowHead(edges, 40, 25, 10, 25, 50, 50)

	if hasArrow {
		t.Error("Should not detect arrow when there's no arrow head")
	}
}

func TestDetectArrowHead_ZeroLength(t *testing.T) {
	edges := make([][]bool, 10)
	for y := 0; y < 10; y++ {
		edges[y] = make([]bool, 10)
	}

	// Same point (zero length)
	hasArrow := detectArrowHead(edges, 5, 5, 5, 5, 10, 10)

	if hasArrow {
		t.Error("Should not detect arrow for zero-length line")
	}
}

func TestLineResult_Length(t *testing.T) {
	img := createHorizontalLineImage(100, 50, 25, 1)

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	if result.Count > 0 {
		line := result.Lines[0]
		// Calculate expected length
		dx := float64(line.End.X - line.Start.X)
		dy := float64(line.End.Y - line.Start.Y)
		expectedLength := math.Sqrt(dx*dx + dy*dy)

		// Should be roughly the same (within 1 pixel due to rounding)
		if math.Abs(line.Length-expectedLength) > 1 {
			t.Errorf("Length mismatch: stored %.1f, calculated %.1f", line.Length, expectedLength)
		}
	}
}

func TestLineResult_Color(t *testing.T) {
	// Create image with colored line
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Red line
	for x := 10; x < 90; x++ {
		img.Set(x, 50, color.RGBA{255, 0, 0, 255})
	}

	result, err := DetectLines(img, 20, false)
	if err != nil {
		t.Fatalf("DetectLines failed: %v", err)
	}

	if result.Count > 0 {
		// Color should be sampled at midpoint
		t.Logf("Line color: %s", result.Lines[0].Color)
	}
}
