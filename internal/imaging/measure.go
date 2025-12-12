package imaging

import (
	"image"
	"math"
)

// Point represents a 2D coordinate in pixel space.
//
// Points use the standard image coordinate system where (0,0) is at the top-left,
// X increases rightward, and Y increases downward.
type Point struct {
	X int `json:"x"` // Horizontal position (0 = leftmost)
	Y int `json:"y"` // Vertical position (0 = topmost)
}

// DistanceResult contains comprehensive measurement data between two points.
//
// All distances are in pixels. Percentage values are relative to image dimensions,
// useful for resolution-independent comparisons.
type DistanceResult struct {
	// DistancePixels is the Euclidean distance: sqrt(dx² + dy²).
	// Rounded to 2 decimal places for readability.
	DistancePixels float64 `json:"distance_pixels"`

	// DeltaX is the horizontal displacement: x2 - x1.
	// Positive = rightward, Negative = leftward.
	DeltaX int `json:"delta_x"`

	// DeltaY is the vertical displacement: y2 - y1.
	// Positive = downward, Negative = upward.
	DeltaY int `json:"delta_y"`

	// AngleDegrees is the angle from point 1 to point 2.
	// 0° = horizontal right, 90° = straight down, -90° = straight up.
	// Range: -180° to 180°, rounded to 1 decimal place.
	AngleDegrees float64 `json:"angle_degrees"`

	// DistancePercentWidth is the distance as a percentage of image width.
	// Useful for resolution-independent measurements.
	DistancePercentWidth float64 `json:"distance_percent_width"`

	// DistancePercentHeight is the distance as a percentage of image height.
	DistancePercentHeight float64 `json:"distance_percent_height"`
}

// MeasureDistance calculates the Euclidean distance and angle between two points.
//
// This function is useful for precise measurement in diagrams, determining
// element spacing, or validating layout dimensions.
//
// Parameters:
//   - img: Image context (used only for percentage calculations).
//   - x1, y1: First point coordinates.
//   - x2, y2: Second point coordinates.
//
// Returns:
//   - *DistanceResult: Comprehensive measurement data.
//   - error: Currently always nil (coordinates are not bounds-checked).
//
// # Distance Formula
//
//	distance = sqrt((x2-x1)² + (y2-y1)²)
//
// # Angle Convention
//
// The angle uses standard mathematical convention with Y-axis inverted for
// screen coordinates:
//   - 0° points right (positive X direction)
//   - 90° points down (positive Y direction)
//   - -90° points up (negative Y direction)
//   - ±180° points left (negative X direction)
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

// AlignmentResult contains the results of checking point alignment.
//
// Points are considered aligned if their variance (standard deviation) in
// one axis is within the specified tolerance.
type AlignmentResult struct {
	// HorizontallyAligned is true if all points share approximately the same Y coordinate.
	// Determined by: VerticalVariance (std dev of Y) <= tolerance.
	HorizontallyAligned bool `json:"horizontally_aligned"`

	// VerticallyAligned is true if all points share approximately the same X coordinate.
	// Determined by: HorizontalVariance (std dev of X) <= tolerance.
	VerticallyAligned bool `json:"vertically_aligned"`

	// HorizontalVariance is the standard deviation of Y coordinates (not X!).
	// Named for the horizontal *alignment* it indicates, not the axis measured.
	// Lower values mean better horizontal alignment.
	HorizontalVariance float64 `json:"horizontal_variance"`

	// VerticalVariance is the standard deviation of X coordinates (not Y!).
	// Named for the vertical *alignment* it indicates, not the axis measured.
	// Lower values mean better vertical alignment.
	VerticalVariance float64 `json:"vertical_variance"`

	// AverageY is the mean Y coordinate of all points.
	// Useful as a reference line for horizontal alignment.
	AverageY float64 `json:"average_y"`

	// AverageX is the mean X coordinate of all points.
	// Useful as a reference line for vertical alignment.
	AverageX float64 `json:"average_x"`
}

// CheckAlignment determines if a set of points are horizontally or vertically aligned.
//
// This function is useful for verifying layout correctness, checking if diagram
// elements are properly aligned, or finding alignment guides.
//
// Parameters:
//   - points: Slice of points to check. With fewer than 2 points, alignment is
//     trivially true.
//   - tolerance: Maximum allowed standard deviation (in pixels) for points to be
//     considered aligned. Common values: 1-5 for strict alignment, 10+ for loose.
//
// Returns:
//   - *AlignmentResult: Alignment status and statistics.
//   - error: Currently always nil.
//
// # Alignment Detection
//
// Points are horizontally aligned if they lie on (approximately) the same
// horizontal line, meaning their Y coordinates are similar. Conversely, points
// are vertically aligned if their X coordinates are similar.
//
// The variance is calculated as standard deviation, not variance in the
// statistical sense:
//
//	stdDev = sqrt(sum((x - mean)²) / n)
//
// # Special Cases
//
//   - 0 points: Both alignments true, variances 0
//   - 1 point: Both alignments true, variances 0
//   - 2+ points: Alignment determined by tolerance comparison
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

// CompareRegionsResult contains the results of comparing two image regions.
//
// This is useful for detecting repeated elements, finding differences between
// similar regions, or validating that two areas match.
type CompareRegionsResult struct {
	// SimilarityScore ranges from 0.0 to 1.0, where:
	//   - 1.0 = regions are identical
	//   - 0.0 = every pixel differs by more than threshold
	// Calculated as: 1 - (pixels_different / total_pixels)
	SimilarityScore float64 `json:"similarity_score"`

	// PixelsDifferent is the count of pixels with color difference > 10.
	// The threshold of 10 (per-channel average) filters out minor noise.
	PixelsDifferent int `json:"pixels_different"`

	// TotalPixels is the number of pixels compared.
	// For different-sized regions, this is min(width1,width2) * min(height1,height2).
	TotalPixels int `json:"total_pixels"`

	// SameSize is true if both regions have identical dimensions.
	SameSize bool `json:"same_size"`

	// Region1Size contains the dimensions of the first region as (width, height).
	// Uses Point type for convenience (X=width, Y=height).
	Region1Size Point `json:"region1_size"`

	// Region2Size contains the dimensions of the second region as (width, height).
	Region2Size Point `json:"region2_size"`

	// AverageColorDiff is the mean color difference across all compared pixels.
	// Calculated as average of: (|r1-r2| + |g1-g2| + |b1-b2|) / 3
	// Range: 0 (identical) to 255 (maximum difference).
	AverageColorDiff float64 `json:"average_color_diff"`
}

// CompareRegions compares two rectangular regions of an image for similarity.
//
// This function is useful for detecting repeated elements (icons, buttons),
// finding differences between similar UI states, or validating visual consistency.
//
// Parameters:
//   - img: Source image containing both regions.
//   - r1: First region to compare (coordinates as Region type).
//   - r2: Second region to compare.
//
// Returns:
//   - *CompareRegionsResult: Detailed comparison statistics.
//   - error: Currently always nil (no bounds validation).
//
// # Comparison Method
//
// Regions are compared pixel-by-pixel, starting from the top-left of each region.
// If regions have different sizes, only the overlapping area (minimum dimensions)
// is compared.
//
// Color difference for each pixel is calculated as:
//
//	diff = (|r1-r2| + |g1-g2| + |b1-b2|) / 3
//
// A pixel is counted as "different" if diff > 10 (threshold to ignore minor
// compression artifacts or anti-aliasing differences).
//
// # Performance
//
// Time complexity is O(width × height) for the smaller region dimensions.
// Large regions may take noticeable time to compare.
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

// absDiff returns the absolute difference between two uint8 values.
// Used for color channel comparison without overflow issues.
func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
