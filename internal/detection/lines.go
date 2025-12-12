package detection

import (
	"image"
	"math"
	"sort"
)

// Line represents a detected line segment with metadata.
//
// Lines are detected using the Hough line transform, which finds lines
// by voting in polar coordinate space (rho, theta).
type Line struct {
	// Start is one endpoint of the line segment.
	Start Point `json:"start"`

	// End is the other endpoint of the line segment.
	End Point `json:"end"`

	// Length is the Euclidean distance between Start and End in pixels.
	// Rounded to 1 decimal place.
	Length float64 `json:"length"`

	// AngleDegrees is the angle of the line from horizontal.
	// 0° = horizontal right, 90° = vertical down, -90° = vertical up.
	// Range: -180° to 180°, rounded to 1 decimal place.
	AngleDegrees float64 `json:"angle_degrees"`

	// Color is the hex color (#RRGGBB) sampled at the line's midpoint.
	Color string `json:"color"`

	// ThicknessApprox is an estimated line thickness in pixels.
	// Measured by sampling perpendicular to the line at its midpoint.
	ThicknessApprox int `json:"thickness_approx"`

	// HasArrowStart indicates if an arrowhead was detected at the Start point.
	// Only populated if detectArrows was true in DetectLines.
	HasArrowStart bool `json:"has_arrow_start"`

	// HasArrowEnd indicates if an arrowhead was detected at the End point.
	HasArrowEnd bool `json:"has_arrow_end"`
}

// LinesResult contains all lines detected in an image.
type LinesResult struct {
	// Lines is the list of detected line segments.
	// Limited to 50 lines maximum, sorted by vote count (strongest first).
	Lines []Line `json:"lines"`

	// Count is the number of lines detected.
	Count int `json:"count"`
}

// DetectLines finds line segments in an image using the Hough line transform.
//
// This function is useful for detecting connections, borders, separators, and
// structural lines in diagrams. It can optionally detect arrow heads at line
// endpoints.
//
// Parameters:
//   - img: Source image to analyze.
//   - minLength: Minimum line length in pixels. Lines shorter than this are
//     filtered out. Also affects the voting threshold (threshold = minLength/2).
//     Typical: 20-100.
//   - detectArrows: If true, check both endpoints for arrow head patterns.
//     This adds processing time but identifies directed connections.
//
// Returns:
//   - *LinesResult: Detected lines (max 50), sorted by detection confidence.
//   - error: Currently always nil.
//
// # Algorithm (Hough Line Transform)
//
//  1. Edge Detection: Find edge pixels using gradient thresholds
//  2. Hough Space Voting: For each edge pixel, vote for all lines passing
//     through it by iterating theta from 0° to 179° and computing rho:
//     rho = x*cos(theta) + y*sin(theta)
//  3. Peak Detection: Find local maxima in the accumulator with votes >= threshold
//  4. Line Extraction: For each peak (rho, theta):
//     - Find all edge pixels within 2 pixels of the line
//     - Determine endpoints from the extreme points
//  5. Length Filtering: Remove lines shorter than minLength
//  6. Arrow Detection (optional): Check endpoints for arrow head patterns
//
// # Arrow Detection
//
// Arrow heads are detected by looking for edge pixels at ±45° angles from the
// line direction, extending back from the endpoint. Both left and right "wings"
// must have at least 3 edge pixels within 10 pixels of the endpoint.
//
// # Thickness Estimation
//
// Line thickness is estimated by sampling perpendicular to the line at its
// midpoint, counting edge pixels within ±10 pixels.
//
// # Limitations
//
//   - Maximum 50 lines returned (strongest by vote count)
//   - Curved lines are not detected
//   - Very thick lines may be detected as multiple parallel lines
//   - Dashed/dotted lines may be detected as multiple segments
//   - Arrow detection only works for ~45° arrow heads
func DetectLines(img image.Image, minLength int, detectArrows bool) (*LinesResult, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Detect edges
	edges := detectEdges(img, width, height)

	// Hough transform parameters
	maxDist := int(math.Sqrt(float64(width*width + height*height)))
	numAngles := 180
	accumulator := make([][]int, maxDist*2)
	for i := range accumulator {
		accumulator[i] = make([]int, numAngles)
	}

	// Vote in Hough space
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !edges[y][x] {
				continue
			}
			for theta := 0; theta < numAngles; theta++ {
				angle := float64(theta) * math.Pi / 180.0
				rho := float64(x)*math.Cos(angle) + float64(y)*math.Sin(angle)
				rhoIdx := int(rho) + maxDist
				if rhoIdx >= 0 && rhoIdx < maxDist*2 {
					accumulator[rhoIdx][theta]++
				}
			}
		}
	}

	// Find peaks in accumulator
	type Peak struct {
		rho   int
		theta int
		votes int
	}
	peaks := make([]Peak, 0)
	threshold := minLength / 2

	for rhoIdx := 0; rhoIdx < maxDist*2; rhoIdx++ {
		for theta := 0; theta < numAngles; theta++ {
			if accumulator[rhoIdx][theta] >= threshold {
				// Check if local maximum
				isMax := true
				for dr := -2; dr <= 2 && isMax; dr++ {
					for dt := -2; dt <= 2 && isMax; dt++ {
						if dr == 0 && dt == 0 {
							continue
						}
						nr := rhoIdx + dr
						nt := (theta + dt + numAngles) % numAngles
						if nr >= 0 && nr < maxDist*2 {
							if accumulator[nr][nt] > accumulator[rhoIdx][theta] {
								isMax = false
							}
						}
					}
				}
				if isMax {
					peaks = append(peaks, Peak{
						rho:   rhoIdx - maxDist,
						theta: theta,
						votes: accumulator[rhoIdx][theta],
					})
				}
			}
		}
	}

	// Sort peaks by votes
	sort.Slice(peaks, func(i, j int) bool {
		return peaks[i].votes > peaks[j].votes
	})

	// Convert peaks to line segments
	lines := make([]Line, 0)

	for _, peak := range peaks {
		if len(lines) >= 50 { // Limit number of lines
			break
		}

		angle := float64(peak.theta) * math.Pi / 180.0
		rho := float64(peak.rho)

		// Find actual line endpoints by tracing along the line
		var startX, startY, endX, endY int
		found := false

		cosA := math.Cos(angle)
		sinA := math.Sin(angle)

		// Find points on this line in the edge image
		linePoints := make([]Point, 0)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if !edges[y][x] {
					continue
				}
				// Check if point is on the line (within tolerance)
				dist := math.Abs(float64(x)*cosA + float64(y)*sinA - rho)
				if dist < 2.0 {
					linePoints = append(linePoints, Point{X: x, Y: y})
				}
			}
		}

		if len(linePoints) < minLength {
			continue
		}

		// Find endpoints
		minDist := math.MaxFloat64
		maxDist := 0.0
		for _, p := range linePoints {
			d := float64(p.X)*cosA + float64(p.Y)*sinA
			if d < minDist {
				minDist = d
				startX, startY = p.X, p.Y
				found = true
			}
			if d > maxDist {
				maxDist = d
				endX, endY = p.X, p.Y
			}
		}

		if !found {
			continue
		}

		// Calculate length
		dx := float64(endX - startX)
		dy := float64(endY - startY)
		length := math.Sqrt(dx*dx + dy*dy)

		if length < float64(minLength) {
			continue
		}

		// Calculate angle in degrees
		angleDeg := math.Atan2(dy, dx) * 180 / math.Pi

		// Sample color at midpoint
		midX := (startX + endX) / 2
		midY := (startY + endY) / 2
		color := sampleColorHex(img, midX+bounds.Min.X, midY+bounds.Min.Y)

		// Estimate thickness
		thickness := estimateLineThickness(edges, startX, startY, endX, endY, width, height)

		// Detect arrows if requested
		hasArrowStart := false
		hasArrowEnd := false
		if detectArrows {
			hasArrowStart = detectArrowHead(edges, startX, startY, endX, endY, width, height)
			hasArrowEnd = detectArrowHead(edges, endX, endY, startX, startY, width, height)
		}

		lines = append(lines, Line{
			Start:          Point{X: startX + bounds.Min.X, Y: startY + bounds.Min.Y},
			End:            Point{X: endX + bounds.Min.X, Y: endY + bounds.Min.Y},
			Length:         math.Round(length*10) / 10,
			AngleDegrees:   math.Round(angleDeg*10) / 10,
			Color:          color,
			ThicknessApprox: thickness,
			HasArrowStart:  hasArrowStart,
			HasArrowEnd:    hasArrowEnd,
		})
	}

	return &LinesResult{
		Lines: lines,
		Count: len(lines),
	}, nil
}

// estimateLineThickness estimates line thickness by sampling perpendicular to the line.
//
// At the line's midpoint, samples perpendicular to the line direction for ±10 pixels,
// counting edge pixels. Returns a minimum of 1 even if no edges are found.
func estimateLineThickness(edges [][]bool, x1, y1, x2, y2, width, height int) int {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return 1
	}

	// Perpendicular direction
	perpX := -dy / length
	perpY := dx / length

	// Sample at midpoint
	midX := float64(x1+x2) / 2
	midY := float64(y1+y2) / 2

	// Count edge pixels along perpendicular
	thickness := 0
	for d := -10; d <= 10; d++ {
		px := int(midX + float64(d)*perpX)
		py := int(midY + float64(d)*perpY)
		if px >= 0 && px < width && py >= 0 && py < height && edges[py][px] {
			thickness++
		}
	}

	if thickness < 1 {
		thickness = 1
	}
	return thickness
}

// detectArrowHead checks if there's an arrow head pattern at a line endpoint.
//
// Looks for edge pixels forming a "V" shape pointing away from the line direction.
// The arrow wings are expected at ±45° from the line direction.
//
// Parameters:
//   - endX, endY: The endpoint to check for an arrow
//   - otherX, otherY: The other endpoint (defines line direction)
//
// Returns true if both left and right wings have at least 3 edge pixels
// within 10 pixels of the endpoint.
func detectArrowHead(edges [][]bool, endX, endY, otherX, otherY, width, height int) bool {
	// Direction from other end to this end
	dx := float64(endX - otherX)
	dy := float64(endY - otherY)
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return false
	}
	dx /= length
	dy /= length

	// Check for edge pixels in arrow head pattern
	// Look for pixels at ~45 degrees from line direction
	checkDist := 10
	arrowAngle := math.Pi / 4 // 45 degrees

	// Rotate direction by +/- 45 degrees for arrow wings
	cos45 := math.Cos(arrowAngle)
	sin45 := math.Sin(arrowAngle)

	// Left wing direction
	leftX := dx*cos45 - dy*sin45
	leftY := dx*sin45 + dy*cos45

	// Right wing direction
	rightX := dx*cos45 + dy*sin45
	rightY := -dx*sin45 + dy*cos45

	// Count edge pixels along potential arrow wings
	leftCount := 0
	rightCount := 0

	for d := 1; d <= checkDist; d++ {
		px := endX - int(float64(d)*leftX)
		py := endY - int(float64(d)*leftY)
		if px >= 0 && px < width && py >= 0 && py < height && edges[py][px] {
			leftCount++
		}

		px = endX - int(float64(d)*rightX)
		py = endY - int(float64(d)*rightY)
		if px >= 0 && px < width && py >= 0 && py < height && edges[py][px] {
			rightCount++
		}
	}

	// Arrow head if both wings have sufficient edge pixels
	return leftCount >= 3 && rightCount >= 3
}
