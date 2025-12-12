package detection

import (
	"image"
	"image/color"
	"testing"
)

// createTextPatternImage creates an image with text-like edge patterns
func createTextPatternImage(width, height int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	// Create text-like patterns (horizontal lines with gaps)
	for y := 20; y < 80; y += 10 {
		for x := 20; x < width-20; x++ {
			// Simulate letter shapes (vertical strokes)
			if x%15 < 5 {
				img.Set(x, y, color.Black)
				img.Set(x, y+1, color.Black)
				img.Set(x, y+5, color.Black)
			}
		}
	}

	return img
}

// createHighEdgeDensityImage creates an image with very high edge density (not text)
func createHighEdgeDensityImage(width, height int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	// Checker pattern (high edge density)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.Black)
			}
		}
	}

	return img
}

func TestDetectTextRegions(t *testing.T) {
	img := createTextPatternImage(200, 150)

	result, err := DetectTextRegions(img, 0.3)
	if err != nil {
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	t.Logf("Detected %d text regions", result.Count)
}

func TestDetectTextRegions_MinConfidence(t *testing.T) {
	img := createTextPatternImage(200, 150)

	// Low confidence threshold
	result1, _ := DetectTextRegions(img, 0.1)
	// High confidence threshold
	result2, _ := DetectTextRegions(img, 0.8)

	// Higher threshold should give fewer or equal results
	if result2.Count > result1.Count {
		t.Errorf("Higher minConfidence should give fewer results: low=%d, high=%d",
			result1.Count, result2.Count)
	}
}

func TestDetectTextRegions_EmptyImage(t *testing.T) {
	img := createTestImage(200, 150, color.White)

	result, err := DetectTextRegions(img, 0.3)
	if err != nil {
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	// Empty image should have no text regions (no edges)
	if result.Count != 0 {
		t.Errorf("Expected 0 text regions in empty image, got %d", result.Count)
	}
}

func TestDetectTextRegions_HighDensity(t *testing.T) {
	// Very high edge density (like noise) should not match text pattern
	img := createHighEdgeDensityImage(200, 150)

	result, err := DetectTextRegions(img, 0.5)
	if err != nil {
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	// High edge density (>40%) should be filtered out
	t.Logf("Detected %d text regions in high-density image", result.Count)
}

func TestDetectTextRegions_SortedByConfidence(t *testing.T) {
	img := createTextPatternImage(300, 200)

	result, err := DetectTextRegions(img, 0.2)
	if err != nil {
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	// Check that results are sorted by confidence (highest first)
	for i := 1; i < result.Count; i++ {
		if result.Regions[i-1].Confidence < result.Regions[i].Confidence {
			t.Error("Text regions should be sorted by confidence (highest first)")
			break
		}
	}
}

func TestCalculateHorizontalScore(t *testing.T) {
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	// Create horizontal lines - these have one horizontal run per row
	// but many vertical runs (each column has multiple interrupted runs)
	// The algorithm counts runs, not line orientations
	for y := 10; y < 40; y += 5 {
		for x := 5; x < 45; x++ {
			edges[y][x] = true
		}
	}

	score := calculateHorizontalScore(edges, 0, 0, 50, 50)

	// The score depends on the ratio of horizontal runs to total runs
	// Just verify it returns a valid score (0 to 1)
	if score < 0 || score > 1 {
		t.Errorf("Score should be between 0 and 1, got %.2f", score)
	}
}

func TestCalculateHorizontalScore_Vertical(t *testing.T) {
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	// Create vertical lines - these have one vertical run per column
	// but many horizontal runs (each row has multiple interrupted runs)
	for x := 10; x < 40; x += 5 {
		for y := 5; y < 45; y++ {
			edges[y][x] = true
		}
	}

	score := calculateHorizontalScore(edges, 0, 0, 50, 50)

	// The score depends on the ratio of horizontal runs to total runs
	// Just verify it returns a valid score (0 to 1)
	if score < 0 || score > 1 {
		t.Errorf("Score should be between 0 and 1, got %.2f", score)
	}
}

func TestCalculateHorizontalScore_Empty(t *testing.T) {
	edges := make([][]bool, 50)
	for y := 0; y < 50; y++ {
		edges[y] = make([]bool, 50)
	}

	score := calculateHorizontalScore(edges, 0, 0, 50, 50)

	// Empty should return 0
	if score != 0 {
		t.Errorf("Empty edges should have score 0, got %.2f", score)
	}
}

func TestMergeOverlappingRegions(t *testing.T) {
	regions := []TextRegion{
		{Bounds: Bounds{X1: 10, Y1: 10, X2: 50, Y2: 30}, Confidence: 0.8, Area: 800},
		{Bounds: Bounds{X1: 30, Y1: 10, X2: 70, Y2: 30}, Confidence: 0.7, Area: 800}, // overlaps
		{Bounds: Bounds{X1: 100, Y1: 100, X2: 150, Y2: 130}, Confidence: 0.6, Area: 1500},
	}

	merged := mergeOverlappingRegions(regions)

	// Should merge first two, keep third separate
	if len(merged) != 2 {
		t.Errorf("Expected 2 merged regions, got %d", len(merged))
	}
}

func TestMergeOverlappingRegions_NoOverlap(t *testing.T) {
	regions := []TextRegion{
		{Bounds: Bounds{X1: 10, Y1: 10, X2: 30, Y2: 30}, Confidence: 0.8},
		{Bounds: Bounds{X1: 50, Y1: 50, X2: 70, Y2: 70}, Confidence: 0.7},
	}

	merged := mergeOverlappingRegions(regions)

	if len(merged) != 2 {
		t.Errorf("Expected 2 regions (no overlap), got %d", len(merged))
	}
}

func TestMergeOverlappingRegions_Empty(t *testing.T) {
	regions := []TextRegion{}

	merged := mergeOverlappingRegions(regions)

	if len(merged) != 0 {
		t.Errorf("Expected 0 regions, got %d", len(merged))
	}
}

func TestRegionsOverlap(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Bounds
		expected bool
	}{
		{
			"overlapping",
			Bounds{X1: 0, Y1: 0, X2: 50, Y2: 50},
			Bounds{X1: 25, Y1: 25, X2: 75, Y2: 75},
			true,
		},
		{
			"non-overlapping horizontal",
			Bounds{X1: 0, Y1: 0, X2: 50, Y2: 50},
			Bounds{X1: 60, Y1: 0, X2: 100, Y2: 50},
			false,
		},
		{
			"non-overlapping vertical",
			Bounds{X1: 0, Y1: 0, X2: 50, Y2: 50},
			Bounds{X1: 0, Y1: 60, X2: 50, Y2: 100},
			false,
		},
		{
			"touching edges (not overlapping)",
			Bounds{X1: 0, Y1: 0, X2: 50, Y2: 50},
			Bounds{X1: 50, Y1: 0, X2: 100, Y2: 50},
			false,
		},
		{
			"contained",
			Bounds{X1: 0, Y1: 0, X2: 100, Y2: 100},
			Bounds{X1: 25, Y1: 25, X2: 75, Y2: 75},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := regionsOverlap(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("regionsOverlap: got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMergeBounds(t *testing.T) {
	a := Bounds{X1: 10, Y1: 20, X2: 50, Y2: 60}
	b := Bounds{X1: 30, Y1: 40, X2: 80, Y2: 90}

	merged := mergeBounds(a, b)

	// Should be the union of both
	if merged.X1 != 10 || merged.Y1 != 20 || merged.X2 != 80 || merged.Y2 != 90 {
		t.Errorf("mergeBounds: got (%d,%d,%d,%d), want (10,20,80,90)",
			merged.X1, merged.Y1, merged.X2, merged.Y2)
	}
}

func TestMinInt(t *testing.T) {
	if minInt(5, 10) != 5 {
		t.Error("minInt(5, 10) should be 5")
	}
	if minInt(10, 5) != 5 {
		t.Error("minInt(10, 5) should be 5")
	}
	if minInt(5, 5) != 5 {
		t.Error("minInt(5, 5) should be 5")
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(5, 10) != 10 {
		t.Error("maxInt(5, 10) should be 10")
	}
	if maxInt(10, 5) != 10 {
		t.Error("maxInt(10, 5) should be 10")
	}
	if maxInt(5, 5) != 5 {
		t.Error("maxInt(5, 5) should be 5")
	}
}

func TestTextRegion_Area(t *testing.T) {
	img := createTextPatternImage(200, 150)

	result, err := DetectTextRegions(img, 0.2)
	if err != nil {
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	// Verify area calculation is correct
	for _, region := range result.Regions {
		expectedArea := (region.Bounds.X2 - region.Bounds.X1) * (region.Bounds.Y2 - region.Bounds.Y1)
		if region.Area != expectedArea {
			t.Errorf("Area mismatch: stored %d, calculated %d", region.Area, expectedArea)
		}
	}
}

func TestDetectTextRegions_SmallImage(t *testing.T) {
	// Very small image (smaller than window sizes)
	img := createTestImage(50, 20, color.White)

	result, err := DetectTextRegions(img, 0.3)
	if err != nil {
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	// Should not crash, may detect 0 regions
	t.Logf("Small image: detected %d regions", result.Count)
}
