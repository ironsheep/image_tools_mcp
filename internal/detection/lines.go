package detection

import (
	"image"
	"math"
	"sort"
)

// Line represents a detected line segment
type Line struct {
	Start          Point   `json:"start"`
	End            Point   `json:"end"`
	Length         float64 `json:"length"`
	AngleDegrees   float64 `json:"angle_degrees"`
	Color          string  `json:"color"`
	ThicknessApprox int    `json:"thickness_approx"`
	HasArrowStart  bool    `json:"has_arrow_start"`
	HasArrowEnd    bool    `json:"has_arrow_end"`
}

// LinesResult contains detected lines
type LinesResult struct {
	Lines []Line `json:"lines"`
	Count int    `json:"count"`
}

// DetectLines finds line segments in an image using Hough transform
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

// estimateLineThickness estimates line thickness by sampling perpendicular to line
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

// detectArrowHead checks if there's an arrow head at the given end of a line
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
