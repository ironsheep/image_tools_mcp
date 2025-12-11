package detection

import (
	"image"
	"math"
	"sort"
)

// TextRegion represents a detected text region
type TextRegion struct {
	Bounds     Bounds  `json:"bounds"`
	Confidence float64 `json:"confidence"`
	Area       int     `json:"area"`
}

// TextRegionsResult contains detected text regions
type TextRegionsResult struct {
	Regions []TextRegion `json:"regions"`
	Count   int          `json:"count"`
}

// DetectTextRegions finds regions likely to contain text
// This is a heuristic-based approach that looks for areas with high edge density
// and appropriate aspect ratios typical of text
func DetectTextRegions(img image.Image, minConfidence float64) (*TextRegionsResult, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Detect edges
	edges := detectEdges(img, width, height)

	// Use sliding window to find regions with high edge density
	windowSizes := []struct{ w, h int }{
		{100, 30}, // Small text
		{150, 40}, // Medium text
		{200, 50}, // Large text
		{80, 25},  // Very small text
	}

	candidates := make([]TextRegion, 0)

	for _, ws := range windowSizes {
		stepX := ws.w / 2
		stepY := ws.h / 2

		for y := 0; y <= height-ws.h; y += stepY {
			for x := 0; x <= width-ws.w; x += stepX {
				// Count edge pixels in window
				edgeCount := 0
				for wy := 0; wy < ws.h; wy++ {
					for wx := 0; wx < ws.w; wx++ {
						if edges[y+wy][x+wx] {
							edgeCount++
						}
					}
				}

				// Calculate edge density
				area := ws.w * ws.h
				density := float64(edgeCount) / float64(area)

				// Text typically has medium edge density (not too sparse, not too dense)
				if density >= 0.05 && density <= 0.4 {
					// Check horizontal edge distribution (text is usually horizontal)
					horizontalScore := calculateHorizontalScore(edges, x, y, ws.w, ws.h)

					confidence := horizontalScore * (1.0 - math.Abs(density-0.2)/0.2)

					if confidence >= minConfidence {
						candidates = append(candidates, TextRegion{
							Bounds: Bounds{
								X1: x + bounds.Min.X,
								Y1: y + bounds.Min.Y,
								X2: x + ws.w + bounds.Min.X,
								Y2: y + ws.h + bounds.Min.Y,
							},
							Confidence: math.Round(confidence*1000) / 1000,
							Area:       area,
						})
					}
				}
			}
		}
	}

	// Merge overlapping regions
	merged := mergeOverlappingRegions(candidates)

	// Sort by confidence
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Confidence > merged[j].Confidence
	})

	return &TextRegionsResult{
		Regions: merged,
		Count:   len(merged),
	}, nil
}

// calculateHorizontalScore calculates how "horizontal" the edge distribution is
func calculateHorizontalScore(edges [][]bool, x, y, w, h int) float64 {
	horizontalRuns := 0
	verticalRuns := 0

	// Count horizontal edge runs
	for row := y; row < y+h; row++ {
		inRun := false
		for col := x; col < x+w; col++ {
			if edges[row][col] {
				if !inRun {
					horizontalRuns++
					inRun = true
				}
			} else {
				inRun = false
			}
		}
	}

	// Count vertical edge runs
	for col := x; col < x+w; col++ {
		inRun := false
		for row := y; row < y+h; row++ {
			if edges[row][col] {
				if !inRun {
					verticalRuns++
					inRun = true
				}
			} else {
				inRun = false
			}
		}
	}

	// Text typically has more horizontal structure
	if horizontalRuns+verticalRuns == 0 {
		return 0
	}
	return float64(horizontalRuns) / float64(horizontalRuns+verticalRuns)
}

// mergeOverlappingRegions combines overlapping text regions
func mergeOverlappingRegions(regions []TextRegion) []TextRegion {
	if len(regions) == 0 {
		return regions
	}

	merged := make([]TextRegion, 0)

	for _, r := range regions {
		foundMerge := false
		for i := range merged {
			if regionsOverlap(r.Bounds, merged[i].Bounds) {
				// Merge into existing region
				merged[i].Bounds = mergeBounds(r.Bounds, merged[i].Bounds)
				merged[i].Confidence = math.Max(r.Confidence, merged[i].Confidence)
				merged[i].Area = (merged[i].Bounds.X2 - merged[i].Bounds.X1) *
					(merged[i].Bounds.Y2 - merged[i].Bounds.Y1)
				foundMerge = true
				break
			}
		}
		if !foundMerge {
			merged = append(merged, r)
		}
	}

	return merged
}

// regionsOverlap checks if two bounds overlap
func regionsOverlap(a, b Bounds) bool {
	return a.X1 < b.X2 && a.X2 > b.X1 && a.Y1 < b.Y2 && a.Y2 > b.Y1
}

// mergeBounds combines two bounds into their union
func mergeBounds(a, b Bounds) Bounds {
	return Bounds{
		X1: minInt(a.X1, b.X1),
		Y1: minInt(a.Y1, b.Y1),
		X2: maxInt(a.X2, b.X2),
		Y2: maxInt(a.Y2, b.Y2),
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
