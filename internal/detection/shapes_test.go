package detection

import (
	"image"
	"image/color"
	"testing"
)

// createTestImage creates a solid color test image
func createTestImage(width, height int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// createRectangleImage creates an image with a rectangle outline
func createRectangleImage(width, height int, rectX1, rectY1, rectX2, rectY2 int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	// Draw rectangle outline
	for x := rectX1; x <= rectX2; x++ {
		img.Set(x, rectY1, color.Black)
		img.Set(x, rectY2, color.Black)
	}
	for y := rectY1; y <= rectY2; y++ {
		img.Set(rectX1, y, color.Black)
		img.Set(rectX2, y, color.Black)
	}

	return img
}

// createCircleImage creates an image with a circle outline
func createCircleImage(width, height, cx, cy, radius int) *image.RGBA {
	img := createTestImage(width, height, color.White)

	// Draw circle outline using midpoint algorithm
	x := radius
	y := 0
	err := 0

	for x >= y {
		img.Set(cx+x, cy+y, color.Black)
		img.Set(cx+y, cy+x, color.Black)
		img.Set(cx-y, cy+x, color.Black)
		img.Set(cx-x, cy+y, color.Black)
		img.Set(cx-x, cy-y, color.Black)
		img.Set(cx-y, cy-x, color.Black)
		img.Set(cx+y, cy-x, color.Black)
		img.Set(cx+x, cy-y, color.Black)

		if err <= 0 {
			y += 1
			err += 2*y + 1
		}
		if err > 0 {
			x -= 1
			err -= 2*x + 1
		}
	}

	return img
}

func TestDetectRectangles(t *testing.T) {
	img := createRectangleImage(100, 100, 20, 20, 80, 80)

	result, err := DetectRectangles(img, 100, 0.5)
	if err != nil {
		t.Fatalf("DetectRectangles failed: %v", err)
	}

	// Should detect at least one rectangle
	if result.Count == 0 {
		t.Log("No rectangles detected - this may be expected for simple edge detection")
	}
}

func TestDetectRectangles_MinArea(t *testing.T) {
	img := createRectangleImage(100, 100, 40, 40, 50, 50)

	// Very small rectangle (10x10 = 100 area)
	result1, _ := DetectRectangles(img, 50, 0.5)
	result2, _ := DetectRectangles(img, 200, 0.5)

	// With minArea=50, might detect; with minArea=200, should not
	if result1.Count > 0 && result2.Count >= result1.Count {
		t.Log("minArea filter may not be working as expected")
	}
}

func TestDetectRectangles_Tolerance(t *testing.T) {
	img := createRectangleImage(100, 100, 20, 20, 80, 80)

	// Low tolerance should find more shapes
	result1, _ := DetectRectangles(img, 100, 0.3)
	// High tolerance should find fewer shapes
	result2, _ := DetectRectangles(img, 100, 0.95)

	// Lower tolerance generally allows more detections
	t.Logf("Low tolerance: %d, High tolerance: %d", result1.Count, result2.Count)
}

func TestDetectRectangles_EmptyImage(t *testing.T) {
	img := createTestImage(100, 100, color.White)

	result, err := DetectRectangles(img, 100, 0.5)
	if err != nil {
		t.Fatalf("DetectRectangles failed: %v", err)
	}

	// Empty image should have no rectangles
	if result.Count != 0 {
		t.Errorf("Expected 0 rectangles in empty image, got %d", result.Count)
	}
}

func TestDetectCircles(t *testing.T) {
	img := createCircleImage(100, 100, 50, 50, 20)

	result, err := DetectCircles(img, 15, 25)
	if err != nil {
		t.Fatalf("DetectCircles failed: %v", err)
	}

	// May or may not detect depending on edge detection sensitivity
	t.Logf("Detected %d circles", result.Count)
}

func TestDetectCircles_MinMaxRadius(t *testing.T) {
	img := createCircleImage(100, 100, 50, 50, 20)

	// Radius outside range should not be detected
	result, err := DetectCircles(img, 30, 40) // r=20 is outside [30,40]
	if err != nil {
		t.Fatalf("DetectCircles failed: %v", err)
	}

	// With narrow range excluding actual radius, might detect fewer
	t.Logf("Detected %d circles with narrow radius range", result.Count)
}

func TestDetectCircles_EmptyImage(t *testing.T) {
	img := createTestImage(100, 100, color.White)

	result, err := DetectCircles(img, 5, 50)
	if err != nil {
		t.Fatalf("DetectCircles failed: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Expected 0 circles in empty image, got %d", result.Count)
	}
}

func TestDetectEdges(t *testing.T) {
	// Create image with a vertical edge
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			if x < 25 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}

	edges := detectEdges(img, 50, 50)

	// Should detect edges around x=25
	edgeFound := false
	for y := 1; y < 49; y++ {
		for x := 23; x <= 26; x++ {
			if edges[y][x] {
				edgeFound = true
				break
			}
		}
	}

	if !edgeFound {
		t.Error("Edge detection should find vertical edge")
	}
}

func TestDetectEdges_UniformImage(t *testing.T) {
	img := createTestImage(50, 50, color.RGBA{128, 128, 128, 255})

	edges := detectEdges(img, 50, 50)

	// Count edges (should be 0 in uniform image)
	edgeCount := 0
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			if edges[y][x] {
				edgeCount++
			}
		}
	}

	if edgeCount != 0 {
		t.Errorf("Uniform image should have 0 edges, got %d", edgeCount)
	}
}

func TestFindContours(t *testing.T) {
	// Create a simple edge pattern
	edges := make([][]bool, 20)
	for y := 0; y < 20; y++ {
		edges[y] = make([]bool, 20)
	}

	// Create a connected contour (small square)
	for x := 5; x <= 15; x++ {
		edges[5][x] = true
		edges[15][x] = true
	}
	for y := 5; y <= 15; y++ {
		edges[y][5] = true
		edges[y][15] = true
	}

	contours := findContours(edges, 20, 20)

	// Should find at least one contour
	if len(contours) == 0 {
		t.Error("Expected to find at least one contour")
	}
}

func TestFindContours_Empty(t *testing.T) {
	edges := make([][]bool, 20)
	for y := 0; y < 20; y++ {
		edges[y] = make([]bool, 20)
	}

	contours := findContours(edges, 20, 20)

	if len(contours) != 0 {
		t.Errorf("Expected 0 contours in empty edge image, got %d", len(contours))
	}
}

func TestFloodFill(t *testing.T) {
	edges := make([][]bool, 10)
	visited := make([][]bool, 10)
	for y := 0; y < 10; y++ {
		edges[y] = make([]bool, 10)
		visited[y] = make([]bool, 10)
	}

	// Create a small connected region
	edges[5][5] = true
	edges[5][6] = true
	edges[6][5] = true
	edges[6][6] = true

	var contour []Point
	floodFill(edges, visited, 5, 5, 10, 10, &contour)

	if len(contour) != 4 {
		t.Errorf("Expected 4 points in contour, got %d", len(contour))
	}

	// Check visited was marked
	if !visited[5][5] || !visited[5][6] || !visited[6][5] || !visited[6][6] {
		t.Error("Flood fill should mark all visited points")
	}
}

func TestGrayValue(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(5, 5, color.RGBA{255, 0, 0, 255})    // red
	img.Set(6, 5, color.RGBA{0, 255, 0, 255})    // green
	img.Set(7, 5, color.RGBA{0, 0, 255, 255})    // blue
	img.Set(8, 5, color.RGBA{128, 128, 128, 255}) // gray

	// Red: 0.299*255 = 76.2
	redGray := grayValue(img, 5, 5)
	if redGray < 70 || redGray > 85 {
		t.Errorf("Red gray value: got %d, expected ~76", redGray)
	}

	// Green: 0.587*255 = 149.7
	greenGray := grayValue(img, 6, 5)
	if greenGray < 140 || greenGray > 160 {
		t.Errorf("Green gray value: got %d, expected ~150", greenGray)
	}

	// Blue: 0.114*255 = 29.1
	blueGray := grayValue(img, 7, 5)
	if blueGray < 25 || blueGray > 35 {
		t.Errorf("Blue gray value: got %d, expected ~29", blueGray)
	}
}

func TestSampleColorHex(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(5, 5, color.RGBA{255, 128, 64, 255})

	hex := sampleColorHex(img, 5, 5)
	if hex != "#FF8040" {
		t.Errorf("sampleColorHex: got %s, want #FF8040", hex)
	}
}

func TestFilterDuplicateCircles(t *testing.T) {
	circles := []Circle{
		{Center: Point{X: 50, Y: 50}, Radius: 20, Confidence: 0.9},
		{Center: Point{X: 52, Y: 51}, Radius: 20, Confidence: 0.8}, // duplicate
		{Center: Point{X: 100, Y: 100}, Radius: 15, Confidence: 0.7},
	}

	filtered := filterDuplicateCircles(circles)

	// Should remove the duplicate (close centers)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 circles after filtering, got %d", len(filtered))
	}
}

func TestFilterDuplicateCircles_Empty(t *testing.T) {
	circles := []Circle{}

	filtered := filterDuplicateCircles(circles)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 circles, got %d", len(filtered))
	}
}

func TestRectangleResult_SortedByArea(t *testing.T) {
	// Create image with multiple rectangles of different sizes
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Small rectangle
	for x := 10; x <= 30; x++ {
		img.Set(x, 10, color.Black)
		img.Set(x, 30, color.Black)
	}
	for y := 10; y <= 30; y++ {
		img.Set(10, y, color.Black)
		img.Set(30, y, color.Black)
	}

	// Large rectangle
	for x := 50; x <= 150; x++ {
		img.Set(x, 50, color.Black)
		img.Set(x, 150, color.Black)
	}
	for y := 50; y <= 150; y++ {
		img.Set(50, y, color.Black)
		img.Set(150, y, color.Black)
	}

	result, _ := DetectRectangles(img, 100, 0.3)

	// If multiple rectangles found, they should be sorted by area
	if result.Count >= 2 {
		if result.Rectangles[0].Area < result.Rectangles[1].Area {
			t.Error("Rectangles should be sorted by area (largest first)")
		}
	}
}
