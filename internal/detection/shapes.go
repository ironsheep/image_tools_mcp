package detection

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// Bounds represents a bounding box
type Bounds struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

// Point represents a 2D point
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Rectangle represents a detected rectangle
type Rectangle struct {
	Bounds      Bounds  `json:"bounds"`
	Center      Point   `json:"center"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Area        int     `json:"area"`
	FillColor   string  `json:"fill_color,omitempty"`
	BorderColor string  `json:"border_color,omitempty"`
	Confidence  float64 `json:"confidence"`
}

// RectanglesResult contains detected rectangles
type RectanglesResult struct {
	Rectangles []Rectangle `json:"rectangles"`
	Count      int         `json:"count"`
}

// DetectRectangles finds rectangular shapes in an image
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

// Circle represents a detected circle
type Circle struct {
	Center     Point   `json:"center"`
	Radius     int     `json:"radius"`
	Diameter   int     `json:"diameter"`
	FillColor  string  `json:"fill_color,omitempty"`
	Confidence float64 `json:"confidence"`
}

// CirclesResult contains detected circles
type CirclesResult struct {
	Circles []Circle `json:"circles"`
	Count   int      `json:"count"`
}

// DetectCircles finds circular shapes in an image using Hough circle transform
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

// detectEdges performs simple edge detection
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

// findContours finds connected components in edge image
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

// floodFill performs flood fill to find connected components
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

// grayValue returns grayscale value of a pixel
func grayValue(img image.Image, x, y int) uint8 {
	r, g, b, _ := img.At(x, y).RGBA()
	return uint8((float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114))
}

// sampleColorHex returns hex color at a pixel
func sampleColorHex(img image.Image, x, y int) string {
	r, g, b, _ := img.At(x, y).RGBA()
	return fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// filterDuplicateCircles removes circles with very close centers
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
