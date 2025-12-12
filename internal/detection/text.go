package detection

import (
	"image"
	"math"
	"sort"
)

// TextRegion represents a detected region likely to contain text.
//
// This is a heuristic-based detection without OCR. The confidence score
// indicates how likely the region contains text based on edge patterns.
type TextRegion struct {
	// Bounds is the bounding box of the text region.
	Bounds Bounds `json:"bounds"`

	// Confidence indicates how likely this region contains text (0.0 to 1.0).
	// Based on edge density and horizontal structure.
	Confidence float64 `json:"confidence"`

	// Area is the region size in square pixels.
	Area int `json:"area"`
}

// TextRegionsResult contains all text regions detected in an image.
type TextRegionsResult struct {
	// Regions is the list of detected text regions, sorted by confidence (highest first).
	Regions []TextRegion `json:"regions"`

	// Count is the number of text regions detected.
	Count int `json:"count"`
}

// DetectTextRegions finds regions likely to contain text using edge density heuristics.
//
// This function identifies areas that have characteristics typical of text:
// medium edge density (not too sparse, not too dense) and predominantly
// horizontal edge structure. It does NOT perform OCR or read the actual text.
//
// Parameters:
//   - img: Source image to analyze.
//   - minConfidence: Minimum confidence threshold (0.0 to 1.0) for including
//     a region. Higher values return fewer, more certain regions. Typical: 0.3-0.7.
//
// Returns:
//   - *TextRegionsResult: Detected text regions sorted by confidence.
//   - error: Currently always nil.
//
// # Algorithm
//
//  1. Edge Detection: Find edge pixels using gradient thresholds
//  2. Sliding Window: Scan the image with multiple window sizes:
//     - 100×30 (small text)
//     - 150×40 (medium text)
//     - 200×50 (large text)
//     - 80×25 (very small text)
//  3. Edge Density Check: For each window position:
//     - Calculate edge pixel density (edges / total pixels)
//     - Text typically has 5-40% edge density
//  4. Horizontal Score: Calculate ratio of horizontal to vertical edge runs
//     - Text tends to have more horizontal structure
//  5. Confidence Calculation:
//     confidence = horizontalScore × (1 - |density - 0.2| / 0.2)
//     This peaks when density is ~20% and horizontal score is high
//  6. Region Merging: Combine overlapping regions, keeping highest confidence
//
// # Edge Density for Text
//
// Text regions typically have medium edge density:
//   - Too low (<5%): Likely blank or solid-colored area
//   - Optimal (15-25%): Typical for text characters
//   - Too high (>40%): Likely a complex graphic or texture
//
// # Horizontal Structure
//
// Latin text is predominantly horizontal, so regions with more horizontal
// edge runs than vertical runs are more likely to contain text.
//
// # Limitations
//
//   - Only detects horizontal text (not rotated or vertical)
//   - May detect non-text regions with similar edge patterns (barcodes, patterns)
//   - Does not read or recognize the text (use OCR for that)
//   - Window sizes are fixed; very large or small text may be missed
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

// calculateHorizontalScore measures how horizontally oriented the edge distribution is.
//
// Counts horizontal and vertical "runs" of consecutive edge pixels.
// Returns the ratio of horizontal runs to total runs.
// A higher score (closer to 1.0) indicates more horizontal structure, typical of text.
// Returns 0 if no edge runs are found.
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

// mergeOverlappingRegions combines overlapping text regions into larger regions.
//
// When two regions overlap, they are merged into a single region with:
//   - Bounds: Union of both bounding boxes
//   - Confidence: Maximum of both confidences
//   - Area: Recalculated from merged bounds
//
// This reduces fragmentation from the sliding window approach.
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

// regionsOverlap checks if two bounding boxes overlap (share any area).
func regionsOverlap(a, b Bounds) bool {
	return a.X1 < b.X2 && a.X2 > b.X1 && a.Y1 < b.Y2 && a.Y2 > b.Y1
}

// mergeBounds returns the smallest bounding box that contains both input bounds.
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
