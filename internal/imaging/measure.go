package imaging

import (
	"image"
	"math"
)

// Point represents a 2D point
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// DistanceResult contains measurement information
type DistanceResult struct {
	DistancePixels       float64 `json:"distance_pixels"`
	DeltaX               int     `json:"delta_x"`
	DeltaY               int     `json:"delta_y"`
	AngleDegrees         float64 `json:"angle_degrees"`
	DistancePercentWidth float64 `json:"distance_percent_width"`
	DistancePercentHeight float64 `json:"distance_percent_height"`
}

// MeasureDistance calculates the distance between two points
func MeasureDistance(img image.Image, x1, y1, x2, y2 int) (*DistanceResult, error) {
	bounds := img.Bounds()
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())

	deltaX := x2 - x1
	deltaY := y2 - y1

	distance := math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY))

	// Calculate angle in degrees (0 = horizontal right, 90 = down)
	angle := math.Atan2(float64(deltaY), float64(deltaX)) * 180 / math.Pi

	return &DistanceResult{
		DistancePixels:       math.Round(distance*100) / 100,
		DeltaX:               deltaX,
		DeltaY:               deltaY,
		AngleDegrees:         math.Round(angle*10) / 10,
		DistancePercentWidth: math.Round(distance/width*1000) / 10,
		DistancePercentHeight: math.Round(distance/height*1000) / 10,
	}, nil
}

// AlignmentResult contains alignment check information
type AlignmentResult struct {
	HorizontallyAligned bool    `json:"horizontally_aligned"`
	VerticallyAligned   bool    `json:"vertically_aligned"`
	HorizontalVariance  float64 `json:"horizontal_variance"`
	VerticalVariance    float64 `json:"vertical_variance"`
	AverageY            float64 `json:"average_y"`
	AverageX            float64 `json:"average_x"`
}

// CheckAlignment checks if points are aligned horizontally or vertically
func CheckAlignment(points []Point, tolerance int) (*AlignmentResult, error) {
	if len(points) < 2 {
		return &AlignmentResult{
			HorizontallyAligned: true,
			VerticallyAligned:   true,
		}, nil
	}

	// Calculate averages
	var sumX, sumY float64
	for _, p := range points {
		sumX += float64(p.X)
		sumY += float64(p.Y)
	}
	avgX := sumX / float64(len(points))
	avgY := sumY / float64(len(points))

	// Calculate variance
	var varX, varY float64
	for _, p := range points {
		dx := float64(p.X) - avgX
		dy := float64(p.Y) - avgY
		varX += dx * dx
		varY += dy * dy
	}
	varX = math.Sqrt(varX / float64(len(points)))
	varY = math.Sqrt(varY / float64(len(points)))

	return &AlignmentResult{
		HorizontallyAligned: varY <= float64(tolerance),
		VerticallyAligned:   varX <= float64(tolerance),
		HorizontalVariance:  math.Round(varY*100) / 100,
		VerticalVariance:    math.Round(varX*100) / 100,
		AverageY:            math.Round(avgY*100) / 100,
		AverageX:            math.Round(avgX*100) / 100,
	}, nil
}

// CompareRegionsResult contains region comparison information
type CompareRegionsResult struct {
	SimilarityScore   float64 `json:"similarity_score"`
	PixelsDifferent   int     `json:"pixels_different"`
	TotalPixels       int     `json:"total_pixels"`
	SameSize          bool    `json:"same_size"`
	Region1Size       Point   `json:"region1_size"`
	Region2Size       Point   `json:"region2_size"`
	AverageColorDiff  float64 `json:"average_color_diff"`
}

// CompareRegions compares two regions of an image
func CompareRegions(img image.Image, r1, r2 Region) (*CompareRegionsResult, error) {
	// Calculate region sizes
	w1 := r1.X2 - r1.X1
	h1 := r1.Y2 - r1.Y1
	w2 := r2.X2 - r2.X1
	h2 := r2.Y2 - r2.Y1

	sameSize := w1 == w2 && h1 == h2

	// For comparison, use the smaller dimensions
	minW := w1
	if w2 < minW {
		minW = w2
	}
	minH := h1
	if h2 < minH {
		minH = h2
	}

	totalPixels := minW * minH
	pixelsDifferent := 0
	var totalColorDiff float64

	for dy := 0; dy < minH; dy++ {
		for dx := 0; dx < minW; dx++ {
			r1c, g1c, b1c, _ := img.At(r1.X1+dx, r1.Y1+dy).RGBA()
			r2c, g2c, b2c, _ := img.At(r2.X1+dx, r2.Y1+dy).RGBA()

			// Convert to 8-bit
			r1v, g1v, b1v := uint8(r1c>>8), uint8(g1c>>8), uint8(b1c>>8)
			r2v, g2v, b2v := uint8(r2c>>8), uint8(g2c>>8), uint8(b2c>>8)

			// Calculate color difference
			dr := absDiff(r1v, r2v)
			dg := absDiff(g1v, g2v)
			db := absDiff(b1v, b2v)
			diff := float64(dr+dg+db) / 3.0

			totalColorDiff += diff

			// Count as different if difference exceeds threshold
			if diff > 10 {
				pixelsDifferent++
			}
		}
	}

	similarity := 1.0 - float64(pixelsDifferent)/float64(totalPixels)
	avgColorDiff := totalColorDiff / float64(totalPixels)

	return &CompareRegionsResult{
		SimilarityScore:  math.Round(similarity*1000) / 1000,
		PixelsDifferent:  pixelsDifferent,
		TotalPixels:      totalPixels,
		SameSize:         sameSize,
		Region1Size:      Point{X: w1, Y: h1},
		Region2Size:      Point{X: w2, Y: h2},
		AverageColorDiff: math.Round(avgColorDiff*100) / 100,
	}, nil
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
