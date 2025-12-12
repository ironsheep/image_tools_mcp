package detection

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// Bounds represents a rectangular bounding box in pixel coordinates.
//
// The coordinate convention follows standard image bounds:
//   - (X1, Y1) is the top-left corner (inclusive)
//   - (X2, Y2) is the bottom-right corner (exclusive for iteration, inclusive for bounds)
type Bounds struct {
	X1 int `json:"x1"` // Left edge (inclusive)
	Y1 int `json:"y1"` // Top edge (inclusive)
	X2 int `json:"x2"` // Right edge
	Y2 int `json:"y2"` // Bottom edge
}

// Point represents a 2D coordinate in pixel space.
type Point struct {
	X int `json:"x"` // Horizontal position (0 = leftmost)
	Y int `json:"y"` // Vertical position (0 = topmost)
}

// Rectangle represents a detected rectangular shape with metadata.
//
// Rectangles are detected using edge detection and contour analysis.
// The confidence score indicates how closely the shape matches a true rectangle.
type Rectangle struct {
	// Bounds is the bounding box enclosing the rectangle.
	Bounds Bounds `json:"bounds"`

	// Center is the center point of the rectangle.
	Center Point `json:"center"`

	// Width is the horizontal extent in pixels (X2 - X1).
	Width int `json:"width"`

	// Height is the vertical extent in pixels (Y2 - Y1).
	Height int `json:"height"`

	// Area is the rectangle's area in square pixels (Width × Height).
	Area int `json:"area"`

	// FillColor is the hex color sampled at the center of the rectangle.
	// May be empty if sampling fails.
	FillColor string `json:"fill_color,omitempty"`

	// BorderColor is the hex color sampled at the top-left corner.
	// May be empty if sampling fails.
	BorderColor string `json:"border_color,omitempty"`

	// Confidence indicates how rectangular the shape is (0.0 to 1.0).
	// Based on comparing contour length to expected rectangle perimeter.
	Confidence float64 `json:"confidence"`
}

// RectanglesResult contains all rectangles detected in an image.
type RectanglesResult struct {
	// Rectangles is the list of detected rectangles, sorted by area (largest first).
	Rectangles []Rectangle `json:"rectangles"`

	// Count is the number of rectangles detected.
	Count int `json:"count"`
}

// DetectRectangles finds rectangular shapes in an image using edge and contour analysis.
//
// This function is useful for detecting boxes, frames, UI elements, and other
// rectangular shapes in diagrams and screenshots.
//
// Parameters:
//   - img: Source image to analyze.
//   - minArea: Minimum area in square pixels for a rectangle to be included.
//     Use higher values to filter out small noise. Typical: 100-1000.
//   - tolerance: Rectangularity threshold (0.0 to 1.0). Higher values require
//     shapes to be more perfectly rectangular. Typical: 0.8-0.95.
//
// Returns:
//   - *RectanglesResult: Detected rectangles sorted by area (largest first).
//   - error: Currently always nil.
//
// # Algorithm
//
//  1. Edge Detection: Compute gradients and threshold to find edge pixels
//  2. Contour Finding: Use flood-fill to group connected edge pixels
//  3. Bounding Box: Calculate the bounding rectangle of each contour
//  4. Rectangularity Check: Compare contour perimeter to expected rectangle
//     perimeter. Score = 1 - |contour_length - expected_perimeter| / expected_perimeter
//  5. Filtering: Remove shapes below minArea or with score < tolerance
//  6. Color Sampling: Sample fill color at center, border color at corner
//
// # Rectangularity Score
//
// A perfect rectangle has a contour length exactly equal to 2*(width + height).
// The rectangularity score measures deviation from this:
//   - 1.0 = Perfect rectangle (contour matches perimeter exactly)
//   - Lower values indicate non-rectangular shapes (circles, irregular polygons)
//
// # Limitations
//
//   - Only detects axis-aligned rectangles (not rotated)
//   - May detect nested rectangles separately
//   - Rounded corners reduce rectangularity score
//   - Very thin rectangles may have low confidence
func DetectRectangles(img image.Image, minArea int, tolerance float64) (*RectanglesResult, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Convert to grayscale and detect edges
	edges := detectEdges(img, width, height)

	// Find contours (connected components of edge pixels)
	contours := findContours(edges, width, height)

	// Filter and analyze contours for rectangles
	rectangles := make([]Rectangle, 0)

	for _, contour := range contours {
		if len(contour) < 4 {
			continue
		}

		// Get bounding box of contour
		minX, minY := width, height
		maxX, maxY := 0, 0
		for _, p := range contour {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}

		rectWidth := maxX - minX
		rectHeight := maxY - minY
		area := rectWidth * rectHeight

		if area < minArea {
			continue
		}

		// Calculate how rectangular the shape is
		contourArea := len(contour)
		expectedPerimeter := 2 * (rectWidth + rectHeight)
		rectangularity := 1.0 - math.Abs(float64(contourArea-expectedPerimeter))/float64(expectedPerimeter)

		if rectangularity < tolerance {
			continue
		}

		// Sample colors
		centerX := (minX + maxX) / 2
		centerY := (minY + maxY) / 2

		fillColor := sampleColorHex(img, centerX, centerY)
		borderColor := sampleColorHex(img, minX, minY)

		rectangles = append(rectangles, Rectangle{
			Bounds: Bounds{
				X1: minX + bounds.Min.X,
				Y1: minY + bounds.Min.Y,
				X2: maxX + bounds.Min.X,
				Y2: maxY + bounds.Min.Y,
			},
			Center: Point{
				X: centerX + bounds.Min.X,
				Y: centerY + bounds.Min.Y,
			},
			Width:       rectWidth,
			Height:      rectHeight,
			Area:        area,
			FillColor:   fillColor,
			BorderColor: borderColor,
			Confidence:  rectangularity,
		})
	}

	// Sort by area descending
	sort.Slice(rectangles, func(i, j int) bool {
		return rectangles[i].Area > rectangles[j].Area
	})

	return &RectanglesResult{
		Rectangles: rectangles,
		Count:      len(rectangles),
	}, nil
}

// Circle represents a detected circular shape with metadata.
//
// Circles are detected using the Hough circle transform, which votes for
// potential circle centers at each edge pixel.
type Circle struct {
	// Center is the detected center point of the circle.
	Center Point `json:"center"`

	// Radius is the detected radius in pixels.
	Radius int `json:"radius"`

	// Diameter is 2 × Radius for convenience.
	Diameter int `json:"diameter"`

	// FillColor is the hex color sampled at the center of the circle.
	FillColor string `json:"fill_color,omitempty"`

	// Confidence indicates detection quality (0.0 to 1.0).
	// Based on the ratio of edge votes to expected circumference.
	Confidence float64 `json:"confidence"`
}

// CirclesResult contains all circles detected in an image.
type CirclesResult struct {
	// Circles is the list of detected circles, sorted by confidence (highest first).
	Circles []Circle `json:"circles"`

	// Count is the number of circles detected.
	Count int `json:"count"`
}

// DetectCircles finds circular shapes in an image using the Hough circle transform.
//
// This function is useful for detecting nodes, bullets, connectors, and other
// circular elements in diagrams.
//
// Parameters:
//   - img: Source image to analyze.
//   - minRadius: Minimum circle radius to detect in pixels. Use higher values
//     to filter out small dots. Typical: 5-20.
//   - maxRadius: Maximum circle radius to detect in pixels. Limits search space
//     for performance. Typical: 50-500.
//
// Returns:
//   - *CirclesResult: Detected circles sorted by confidence (highest first).
//   - error: Currently always nil.
//
// # Algorithm (Hough Circle Transform)
//
//  1. Edge Detection: Find edge pixels using gradient thresholds
//  2. Accumulator Voting: For each radius from minRadius to maxRadius:
//     - For each edge pixel, vote for potential centers by drawing a
//     voting circle around the pixel
//     - Votes are cast every 10° around the edge pixel
//  3. Peak Detection: Find local maxima in the accumulator that exceed
//     threshold (60% of expected circumference points)
//  4. Duplicate Removal: Merge circles with overlapping centers
//  5. Color Sampling: Sample fill color at detected center
//
// # Confidence Score
//
// Confidence is calculated as: votes / (2 × radius)
//
// This represents the fraction of the circumference where edge pixels voted
// for this center. Capped at 1.0.
//   - 1.0 = Every expected edge point voted for this center
//   - 0.6 = Threshold for detection (sparse but detectable circle)
//
// # Performance
//
// Time complexity is O(width × height × (maxRadius - minRadius) × 36), where 36
// comes from voting every 10°. Large radius ranges significantly increase time.
//
// # Limitations
//
//   - Only detects filled or outlined circles, not arcs
//   - Overlapping circles may be detected as single circles
//   - Ellipses are not detected (only true circles)
//   - Large maxRadius values slow detection significantly
func DetectCircles(img image.Image, minRadius, maxRadius int) (*CirclesResult, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Detect edges
	edges := detectEdges(img, width, height)

	// Simple circle detection using accumulator
	circles := make([]Circle, 0)

	// For each radius, accumulate votes
	for radius := minRadius; radius <= maxRadius; radius++ {
		accumulator := make([][]int, height)
		for y := 0; y < height; y++ {
			accumulator[y] = make([]int, width)
		}

		// Vote for circle centers
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if edges[y][x] {
					// Vote in a circle around this edge point
					for angle := 0; angle < 360; angle += 10 {
						rad := float64(angle) * math.Pi / 180
						cx := x - int(float64(radius)*math.Cos(rad))
						cy := y - int(float64(radius)*math.Sin(rad))
						if cx >= 0 && cx < width && cy >= 0 && cy < height {
							accumulator[cy][cx]++
						}
					}
				}
			}
		}

		// Find local maxima in accumulator
		threshold := int(float64(2*radius) * 0.6) // Require ~60% of circumference
		for y := radius; y < height-radius; y++ {
			for x := radius; x < width-radius; x++ {
				if accumulator[y][x] >= threshold {
					// Check if local maximum
					isMax := true
					for dy := -5; dy <= 5 && isMax; dy++ {
						for dx := -5; dx <= 5 && isMax; dx++ {
							if dy == 0 && dx == 0 {
								continue
							}
							ny, nx := y+dy, x+dx
							if ny >= 0 && ny < height && nx >= 0 && nx < width {
								if accumulator[ny][nx] > accumulator[y][x] {
									isMax = false
								}
							}
						}
					}

					if isMax {
						confidence := float64(accumulator[y][x]) / float64(2*radius)
						fillColor := sampleColorHex(img, x, y)

						circles = append(circles, Circle{
							Center: Point{
								X: x + bounds.Min.X,
								Y: y + bounds.Min.Y,
							},
							Radius:     radius,
							Diameter:   radius * 2,
							FillColor:  fillColor,
							Confidence: math.Min(confidence, 1.0),
						})
					}
				}
			}
		}
	}

	// Remove duplicate detections (circles with very close centers)
	filtered := filterDuplicateCircles(circles)

	// Sort by confidence descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Confidence > filtered[j].Confidence
	})

	return &CirclesResult{
		Circles: filtered,
		Count:   len(filtered),
	}, nil
}

// detectEdges performs simple gradient-based edge detection.
//
// Uses a simple gradient threshold: pixels where |current - neighbor| > 30
// (in grayscale) are marked as edges. Checks both horizontal and vertical
// neighbors.
//
// Returns a 2D boolean array where true indicates an edge pixel.
// Border pixels (x=0, y=0, x=width-1, y=height-1) are never edges.
func detectEdges(img image.Image, width, height int) [][]bool {
	bounds := img.Bounds()
	edges := make([][]bool, height)
	threshold := 30.0

	for y := 0; y < height; y++ {
		edges[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			if x == 0 || y == 0 || x == width-1 || y == height-1 {
				continue
			}

			// Get grayscale values
			c := grayValue(img, x+bounds.Min.X, y+bounds.Min.Y)
			cx := grayValue(img, x+1+bounds.Min.X, y+bounds.Min.Y)
			cy := grayValue(img, x+bounds.Min.X, y+1+bounds.Min.Y)

			// Simple gradient
			dx := math.Abs(float64(c) - float64(cx))
			dy := math.Abs(float64(c) - float64(cy))

			if dx > threshold || dy > threshold {
				edges[y][x] = true
			}
		}
	}

	return edges
}

// findContours finds connected components (contours) in a binary edge image.
//
// Uses flood-fill to group connected edge pixels into contours.
// Connectivity is 8-connected (includes diagonals).
//
// Contours smaller than 10 pixels are discarded as noise.
// Returns a slice of contours, where each contour is a slice of Points.
func findContours(edges [][]bool, width, height int) [][]Point {
	visited := make([][]bool, height)
	for y := 0; y < height; y++ {
		visited[y] = make([]bool, width)
	}

	contours := make([][]Point, 0)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if edges[y][x] && !visited[y][x] {
				contour := make([]Point, 0)
				floodFill(edges, visited, x, y, width, height, &contour)
				if len(contour) >= 10 { // Minimum contour size
					contours = append(contours, contour)
				}
			}
		}
	}

	return contours
}

// floodFill performs iterative flood-fill from a starting point.
//
// Uses a stack-based approach (not recursive) to avoid stack overflow
// on large contours. Marks visited pixels and appends them to the contour.
// Uses 8-connectivity (includes diagonal neighbors).
func floodFill(edges, visited [][]bool, startX, startY, width, height int, contour *[]Point) {
	stack := []Point{{X: startX, Y: startY}}

	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if p.X < 0 || p.X >= width || p.Y < 0 || p.Y >= height {
			continue
		}
		if visited[p.Y][p.X] || !edges[p.Y][p.X] {
			continue
		}

		visited[p.Y][p.X] = true
		*contour = append(*contour, p)

		// 8-connected neighbors
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				stack = append(stack, Point{X: p.X + dx, Y: p.Y + dy})
			}
		}
	}
}

// grayValue converts a pixel to grayscale using ITU-R BT.601 luminance weights.
// Formula: Y = 0.299*R + 0.587*G + 0.114*B
func grayValue(img image.Image, x, y int) uint8 {
	r, g, b, _ := img.At(x, y).RGBA()
	return uint8((float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114))
}

// sampleColorHex returns the hex color (#RRGGBB) of a pixel.
// No bounds checking is performed; caller must ensure coordinates are valid.
func sampleColorHex(img image.Image, x, y int) string {
	r, g, b, _ := img.At(x, y).RGBA()
	return fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// filterDuplicateCircles removes circles with overlapping centers.
//
// Two circles are considered duplicates if the distance between their centers
// is less than the average of their radii. In such cases, only the first
// circle (typically higher confidence due to sorting) is kept.
func filterDuplicateCircles(circles []Circle) []Circle {
	if len(circles) == 0 {
		return circles
	}

	filtered := make([]Circle, 0)
	for _, c := range circles {
		isDuplicate := false
		for _, f := range filtered {
			dx := c.Center.X - f.Center.X
			dy := c.Center.Y - f.Center.Y
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			if dist < float64(c.Radius+f.Radius)/2 {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
