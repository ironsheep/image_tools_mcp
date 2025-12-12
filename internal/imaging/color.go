package imaging

import (
	"fmt"
	"image"
	"sort"
)

// RGBColor represents an RGB color with 8-bit components.
//
// Each component ranges from 0 to 255, where:
//   - 0 represents no intensity (black for all components)
//   - 255 represents full intensity (white for all components)
type RGBColor struct {
	R uint8 `json:"r"` // Red component (0-255)
	G uint8 `json:"g"` // Green component (0-255)
	B uint8 `json:"b"` // Blue component (0-255)
}

// RGBAColor represents an RGBA color with 8-bit components including alpha.
//
// The alpha component represents opacity:
//   - 0 = fully transparent
//   - 255 = fully opaque
type RGBAColor struct {
	R uint8 `json:"r"` // Red component (0-255)
	G uint8 `json:"g"` // Green component (0-255)
	B uint8 `json:"b"` // Blue component (0-255)
	A uint8 `json:"a"` // Alpha/opacity component (0-255)
}

// HSLColor represents a color in HSL (Hue, Saturation, Lightness) color space.
//
// HSL is often more intuitive for color manipulation than RGB:
//   - Hue represents the color type (red, green, blue, etc.)
//   - Saturation represents color intensity (gray to vivid)
//   - Lightness represents brightness (black to white)
type HSLColor struct {
	H int `json:"h"` // Hue: 0-360 degrees (0=red, 120=green, 240=blue)
	S int `json:"s"` // Saturation: 0-100 percent (0=gray, 100=vivid)
	L int `json:"l"` // Lightness: 0-100 percent (0=black, 50=normal, 100=white)
}

// ColorResult contains a color value in multiple representations.
//
// This struct provides the same color in four formats to suit different use cases:
//   - Hex: Compact string format for CSS/web usage
//   - RGB: Standard 8-bit components without alpha
//   - RGBA: 8-bit components with alpha for transparency
//   - HSL: Perceptual color space for intuitive color operations
type ColorResult struct {
	Hex  string    `json:"hex"`  // Hex format "#RRGGBB" (no alpha)
	RGB  RGBColor  `json:"rgb"`  // RGB components
	RGBA RGBAColor `json:"rgba"` // RGBA components with alpha
	HSL  HSLColor  `json:"hsl"`  // HSL representation
}

// SampleColor extracts the color value at a specific pixel coordinate.
//
// Parameters:
//   - img: The source image to sample from.
//   - x: X coordinate (0-based, 0 = leftmost pixel).
//   - y: Y coordinate (0-based, 0 = topmost pixel).
//
// Returns:
//   - *ColorResult: The color at (x, y) in multiple formats.
//   - error: Non-nil if coordinates are outside the image bounds.
//
// # Coordinate System
//
// Coordinates are 0-based with origin at top-left:
//   - Valid X range: 0 to width-1
//   - Valid Y range: 0 to height-1
//
// # Color Conversion
//
// The function reads the native color from the image and converts it to 8-bit
// components. For 16-bit images, values are scaled down by right-shifting 8 bits.
// The Hex format excludes alpha; use RGBA.A to get transparency information.
func SampleColor(img image.Image, x, y int) (*ColorResult, error) {
	bounds := img.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return nil, fmt.Errorf("coordinates (%d,%d) outside image bounds", x, y)
	}

	r, g, b, a := img.At(x, y).RGBA()
	// Convert from 16-bit to 8-bit
	r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)

	return &ColorResult{
		Hex:  fmt.Sprintf("#%02X%02X%02X", r8, g8, b8),
		RGB:  RGBColor{R: r8, G: g8, B: b8},
		RGBA: RGBAColor{R: r8, G: g8, B: b8, A: a8},
		HSL:  rgbToHSL(r8, g8, b8),
	}, nil
}

// LabeledPoint represents a pixel coordinate with an optional descriptive label.
//
// Labels are useful for identifying specific points in the results, such as
// "button_background" or "header_text". If Label is empty, the point will
// still be sampled but won't have an identifying label in the output.
type LabeledPoint struct {
	X     int    // X coordinate (0-based)
	Y     int    // Y coordinate (0-based)
	Label string // Optional descriptive label for this point
}

// LabeledColorResult combines a color sample with its location and optional label.
type LabeledColorResult struct {
	Label string      `json:"label,omitempty"` // Optional label (empty if not provided)
	X     int         `json:"x"`               // X coordinate that was sampled
	Y     int         `json:"y"`               // Y coordinate that was sampled
	Color ColorResult `json:"color"`           // The color at this location
}

// MultiColorResult contains color samples from multiple points.
//
// Results are returned in the same order as the input points.
type MultiColorResult struct {
	Samples []LabeledColorResult `json:"samples"` // Color samples in input order
}

// SampleColorsMulti extracts colors at multiple pixel coordinates in a single call.
//
// This is more efficient than calling SampleColor multiple times when sampling
// many points, as it processes all points in one pass.
//
// Parameters:
//   - img: The source image to sample from.
//   - points: Slice of coordinates to sample. Each point may have an optional label.
//
// Returns:
//   - *MultiColorResult: Colors at all requested points, in the same order as input.
//   - error: Non-nil if any coordinate is outside the image bounds. On error, no
//     partial results are returned.
//
// # Example
//
//	points := []imaging.LabeledPoint{
//	    {X: 10, Y: 20, Label: "background"},
//	    {X: 50, Y: 100, Label: "text"},
//	}
//	result, err := imaging.SampleColorsMulti(img, points)
func SampleColorsMulti(img image.Image, points []LabeledPoint) (*MultiColorResult, error) {
	results := make([]LabeledColorResult, 0, len(points))

	for _, p := range points {
		color, err := SampleColor(img, p.X, p.Y)
		if err != nil {
			return nil, fmt.Errorf("failed to sample point (%d,%d): %w", p.X, p.Y, err)
		}
		results = append(results, LabeledColorResult{
			Label: p.Label,
			X:     p.X,
			Y:     p.Y,
			Color: *color,
		})
	}

	return &MultiColorResult{Samples: results}, nil
}

// Region represents a rectangular region within an image.
//
// Coordinates follow the standard image convention:
//   - (X1, Y1) is the top-left corner (inclusive)
//   - (X2, Y2) is the bottom-right corner (exclusive)
//   - Width = X2 - X1, Height = Y2 - Y1
type Region struct {
	X1 int // Left edge X coordinate (inclusive)
	Y1 int // Top edge Y coordinate (inclusive)
	X2 int // Right edge X coordinate (exclusive)
	Y2 int // Bottom edge Y coordinate (exclusive)
}

// ColorFrequency represents a color and its occurrence frequency in an image.
type ColorFrequency struct {
	Hex        string   `json:"hex"`        // Hex color "#RRGGBB" (quantized)
	Percentage float64  `json:"percentage"` // Percentage of pixels with this color (0-100)
	RGB        RGBColor `json:"rgb"`        // RGB components (quantized)
}

// DominantColorsResult contains the most frequently occurring colors in an image.
//
// Colors are sorted by frequency in descending order (most common first).
type DominantColorsResult struct {
	Colors []ColorFrequency `json:"colors"` // Colors sorted by frequency (descending)
}

// DominantColors extracts the N most common colors from an image or region.
//
// This function analyzes pixel colors and returns the most frequently occurring
// colors, useful for palette extraction or understanding an image's color scheme.
//
// Parameters:
//   - img: The source image to analyze.
//   - count: Maximum number of colors to return. If the image has fewer distinct
//     colors (after quantization), fewer results may be returned.
//   - region: Optional rectangular region to analyze. If nil, the entire image
//     is analyzed.
//
// Returns:
//   - *DominantColorsResult: The dominant colors sorted by frequency.
//   - error: Currently always returns nil (reserved for future validation).
//
// # Color Quantization
//
// To group similar colors, the function quantizes RGB values by dividing each
// component by 16 and rounding down. This means colors within 16 units of each
// other (per component) are grouped together. The quantization formula is:
//
//	quantized = (original / 16) * 16
//
// For example, colors #F0F0F0 and #FAFAFA would both be quantized to #F0F0F0.
//
// # Performance
//
// The function iterates over every pixel in the region, so large images may
// take longer to process. Consider using a smaller region for quick analysis.
func DominantColors(img image.Image, count int, region *Region) (*DominantColorsResult, error) {
	bounds := img.Bounds()
	if region != nil {
		bounds = image.Rect(region.X1, region.Y1, region.X2, region.Y2)
	}

	colorCounts := make(map[string]int)
	totalPixels := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Quantize to reduce color space (group similar colors)
			r8 := uint8((r >> 8) / 16 * 16)
			g8 := uint8((g >> 8) / 16 * 16)
			b8 := uint8((b >> 8) / 16 * 16)
			key := fmt.Sprintf("#%02X%02X%02X", r8, g8, b8)
			colorCounts[key]++
			totalPixels++
		}
	}

	// Convert to slice and sort by frequency
	colors := make([]ColorFrequency, 0, len(colorCounts))
	for hex, cnt := range colorCounts {
		// Parse hex back to RGB
		var r, g, b uint8
		_, _ = fmt.Sscanf(hex, "#%02X%02X%02X", &r, &g, &b)

		colors = append(colors, ColorFrequency{
			Hex:        hex,
			Percentage: float64(cnt) / float64(totalPixels) * 100,
			RGB:        RGBColor{R: r, G: g, B: b},
		})
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].Percentage > colors[j].Percentage
	})

	if len(colors) > count {
		colors = colors[:count]
	}

	return &DominantColorsResult{Colors: colors}, nil
}

// rgbToHSL converts 8-bit RGB values to HSL color space.
//
// The conversion follows the standard algorithm:
//  1. Normalize RGB to 0-1 range
//  2. Find min and max components
//  3. Calculate Lightness as (max + min) / 2
//  4. Calculate Saturation based on lightness
//  5. Calculate Hue based on which component is max
//
// Parameters:
//   - r, g, b: 8-bit color components (0-255)
//
// Returns HSLColor with:
//   - H: 0-360 (degrees on color wheel)
//   - S: 0-100 (percentage)
//   - L: 0-100 (percentage)
func rgbToHSL(r, g, b uint8) HSLColor {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := rf
	if gf > max {
		max = gf
	}
	if bf > max {
		max = bf
	}

	min := rf
	if gf < min {
		min = gf
	}
	if bf < min {
		min = bf
	}

	l := (max + min) / 2.0

	if max == min {
		return HSLColor{H: 0, S: 0, L: int(l * 100)}
	}

	var s float64
	if l < 0.5 {
		s = (max - min) / (max + min)
	} else {
		s = (max - min) / (2.0 - max - min)
	}

	var h float64
	switch max {
	case rf:
		h = (gf - bf) / (max - min)
		if gf < bf {
			h += 6
		}
	case gf:
		h = 2.0 + (bf-rf)/(max-min)
	case bf:
		h = 4.0 + (rf-gf)/(max-min)
	}
	h *= 60

	return HSLColor{
		H: int(h),
		S: int(s * 100),
		L: int(l * 100),
	}
}
