package imaging

import (
	"fmt"
	"image"
	"sort"
)

// MaxDiscreteColors is the upper bound for distinct colors reported or used for vectorization.
const MaxDiscreteColors = 10

// DefaultQuantizeFallback is the quantization step used when an image has more than
// MaxDiscreteColors exact colors (e.g. anti-aliased or noisy art) and the caller did
// not specify a value.
const DefaultQuantizeFallback = 8

// DefaultAlphaThreshold is the minimum 8-bit alpha for a pixel to count as "opaque".
// Pixels below this are treated as background and ignored. 128 sits at the perceptual
// midpoint and reliably excludes the anti-aliased fringe around solid-color icons,
// which would otherwise leak the underlying RGB (often near-white) into the palette.
const DefaultAlphaThreshold = 128

// ChooseQuantize picks a quantization step automatically.
//
// It does a single fast pass counting exact "opaque enough" colors (alpha ≥
// alphaThreshold), aborting once it exceeds MaxDiscreteColors. If the image already
// has ≤ MaxDiscreteColors distinct colors, returns 1 so the SVG palette matches the
// source PNG exactly. Otherwise returns DefaultQuantizeFallback so anti-aliasing /
// noise gets merged into the dominant colors.
func ChooseQuantize(img image.Image, alphaThreshold int) int {
	if alphaThreshold < 0 {
		alphaThreshold = 0
	}
	if alphaThreshold > 255 {
		alphaThreshold = 255
	}
	bounds := img.Bounds()
	seen := make(map[uint32]struct{}, MaxDiscreteColors+1)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if int(a>>8) < alphaThreshold {
				continue
			}
			key := uint32(uint8(r>>8))<<16 | uint32(uint8(g>>8))<<8 | uint32(uint8(b>>8))
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if len(seen) > MaxDiscreteColors {
				return DefaultQuantizeFallback
			}
		}
	}
	return 1
}

// CountedColor is one discrete color found in the image, with its share of opaque pixels.
type CountedColor struct {
	Hex        string   `json:"hex"`
	RGB        RGBColor `json:"rgb"`
	Percentage float64  `json:"percentage"`
}

// CountColorsResult reports how many discrete colors an image contains after quantization.
//
// DistinctCount is the number of colors actually returned (≤ MaxDiscreteColors). If the
// image had more, Truncated is true and OtherPercentage holds the share of opaque pixels
// that fell outside the top MaxDiscreteColors. Quantize is the value actually used —
// see ChooseQuantize for the auto-selection rule. AlphaThreshold is the minimum alpha
// (0-255) for a pixel to count; pixels below it are treated as background.
type CountColorsResult struct {
	DistinctCount   int            `json:"distinct_count"`
	Truncated       bool           `json:"truncated"`
	OtherPercentage float64        `json:"other_percentage,omitempty"`
	Quantize        int            `json:"quantize"`
	QuantizeAuto    bool           `json:"quantize_auto,omitempty"`
	AlphaThreshold  int            `json:"alpha_threshold"`
	IgnoredAlpha    bool           `json:"ignored_alpha"`
	Colors          []CountedColor `json:"colors"`
}

// CountColors returns the number of discrete colors in an image, capped at MaxDiscreteColors.
//
// Pixels with alpha < alphaThreshold are ignored — this matters for PNGs whose
// anti-aliased edges store near-white RGB under low alpha. Without thresholding, those
// faded fringe pixels would dominate the palette and crowd out the actual logo colors.
// Pass 0 for alphaThreshold to use DefaultAlphaThreshold (128); pass 1 to ignore only
// fully-transparent pixels (legacy behavior); pass 255 to require full opacity.
//
// Colors within `quantize` units of each other (per RGB channel) are grouped; quantize
// must be between 1 and 128. Pass 0 to auto-select via ChooseQuantize.
func CountColors(img image.Image, quantize, alphaThreshold int) (*CountColorsResult, error) {
	if alphaThreshold == 0 {
		alphaThreshold = DefaultAlphaThreshold
	}
	if alphaThreshold < 0 {
		alphaThreshold = 0
	}
	if alphaThreshold > 255 {
		alphaThreshold = 255
	}

	auto := false
	if quantize <= 0 {
		quantize = ChooseQuantize(img, alphaThreshold)
		auto = true
	}
	if quantize > 128 {
		quantize = 128
	}

	bounds := img.Bounds()
	counts := make(map[uint32]int)
	totalOpaque := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if int(a>>8) < alphaThreshold {
				continue
			}
			r8 := uint8((r >> 8) / uint32(quantize) * uint32(quantize))
			g8 := uint8((g >> 8) / uint32(quantize) * uint32(quantize))
			b8 := uint8((b >> 8) / uint32(quantize) * uint32(quantize))
			key := uint32(r8)<<16 | uint32(g8)<<8 | uint32(b8)
			counts[key]++
			totalOpaque++
		}
	}

	if totalOpaque == 0 {
		return &CountColorsResult{Quantize: quantize, QuantizeAuto: auto, AlphaThreshold: alphaThreshold, IgnoredAlpha: true, Colors: []CountedColor{}}, nil
	}

	all := make([]CountedColor, 0, len(counts))
	for key, cnt := range counts {
		r := uint8(key >> 16)
		g := uint8(key >> 8)
		b := uint8(key)
		all = append(all, CountedColor{
			Hex:        fmt.Sprintf("#%02X%02X%02X", r, g, b),
			RGB:        RGBColor{R: r, G: g, B: b},
			Percentage: float64(cnt) / float64(totalOpaque) * 100,
		})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Percentage > all[j].Percentage })

	res := &CountColorsResult{
		Quantize:       quantize,
		QuantizeAuto:   auto,
		AlphaThreshold: alphaThreshold,
		IgnoredAlpha:   true,
	}
	if len(all) > MaxDiscreteColors {
		res.Truncated = true
		other := 0.0
		for _, c := range all[MaxDiscreteColors:] {
			other += c.Percentage
		}
		res.OtherPercentage = other
		all = all[:MaxDiscreteColors]
	}
	res.Colors = all
	res.DistinctCount = len(all)
	return res, nil
}
