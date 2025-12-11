package imaging

import (
	"fmt"
	"image"
	"sort"
)

// RGBColor represents RGB color values
type RGBColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// RGBAColor represents RGBA color values
type RGBAColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// HSLColor represents HSL color values
type HSLColor struct {
	H int `json:"h"` // 0-360
	S int `json:"s"` // 0-100
	L int `json:"l"` // 0-100
}

// ColorResult contains color information in multiple formats
type ColorResult struct {
	Hex  string    `json:"hex"`
	RGB  RGBColor  `json:"rgb"`
	RGBA RGBAColor `json:"rgba"`
	HSL  HSLColor  `json:"hsl"`
}

// SampleColor gets the color at a specific pixel
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

// LabeledPoint represents a point with an optional label
type LabeledPoint struct {
	X     int
	Y     int
	Label string
}

// LabeledColorResult is a color result with a label
type LabeledColorResult struct {
	Label string      `json:"label,omitempty"`
	X     int         `json:"x"`
	Y     int         `json:"y"`
	Color ColorResult `json:"color"`
}

// MultiColorResult contains results for multiple sample points
type MultiColorResult struct {
	Samples []LabeledColorResult `json:"samples"`
}

// SampleColorsMulti samples colors at multiple points
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

// Region represents a rectangular region
type Region struct {
	X1 int
	Y1 int
	X2 int
	Y2 int
}

// ColorFrequency represents a color and its frequency
type ColorFrequency struct {
	Hex        string   `json:"hex"`
	Percentage float64  `json:"percentage"`
	RGB        RGBColor `json:"rgb"`
}

// DominantColorsResult contains the dominant colors in an image
type DominantColorsResult struct {
	Colors []ColorFrequency `json:"colors"`
}

// DominantColors extracts the most common colors from an image or region
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
		fmt.Sscanf(hex, "#%02X%02X%02X", &r, &g, &b)

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

// rgbToHSL converts RGB values to HSL
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
