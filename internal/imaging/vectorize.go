package imaging

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/dennwc/gotrace"
)

// VectorizeResult holds the SVG produced by tracing a low-color raster image.
type VectorizeResult struct {
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	ColorsUsed     int      `json:"colors_used"`
	Palette        []string `json:"palette"`
	SVG            string   `json:"svg"`
	SVGBase64      string   `json:"svg_base64"`
	MimeType       string   `json:"mime_type"`
	AutoDetected   bool     `json:"auto_detected,omitempty"`
	Quantize       int      `json:"quantize"`
	QuantizeAuto   bool     `json:"quantize_auto,omitempty"`
	AlphaThreshold int      `json:"alpha_threshold"`
}

// VectorizeOptions tunes the raster-to-vector conversion.
//
// MaxColors clamps the output palette size. Pass 0 to auto-detect via CountColors
// (clamped to MaxDiscreteColors). Quantize controls how aggressively similar source
// colors are merged before palette selection. TurdSize suppresses speckles smaller than
// the given pixel area; AlphaMax is potrace's corner-rounding parameter (0 = polygons,
// 1.0 = smooth, 1.3334 = max smoothing). AlphaThreshold is the minimum alpha (0-255)
// for a pixel to be considered part of the icon — pixels below this are treated as
// transparent background. Pass 0 for DefaultAlphaThreshold (128); without this, the
// near-white RGB stored under anti-aliased fringe pixels leaks into the palette and
// can dominate the trace.
type VectorizeOptions struct {
	MaxColors      int
	Quantize       int
	TurdSize       int
	AlphaMax       float64
	AlphaThreshold int
}

// Vectorize converts a low-color raster image (with optional transparent background)
// to SVG by tracing each color layer separately with potrace.
//
// The algorithm:
//  1. Build a palette of up to MaxColors via quantization + frequency selection.
//  2. For each palette color, build a binary mask of pixels that match (and are opaque).
//  3. Trace each mask with gotrace and emit one <path fill="..."> per color.
//  4. Concatenate into a single SVG with the original viewBox.
//
// Fully transparent pixels stay transparent in the output (they are absent from every layer).
func Vectorize(img image.Image, opts VectorizeOptions) (*VectorizeResult, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if opts.AlphaThreshold == 0 {
		opts.AlphaThreshold = DefaultAlphaThreshold
	}
	if opts.AlphaThreshold < 0 {
		opts.AlphaThreshold = 0
	}
	if opts.AlphaThreshold > 255 {
		opts.AlphaThreshold = 255
	}

	quantizeAuto := false
	if opts.Quantize <= 0 {
		opts.Quantize = ChooseQuantize(img, opts.AlphaThreshold)
		quantizeAuto = true
	}
	if opts.TurdSize <= 0 {
		opts.TurdSize = 2
	}
	if opts.AlphaMax <= 0 {
		opts.AlphaMax = 1.0
	}

	autoDetected := false
	if opts.MaxColors <= 0 {
		count, err := CountColors(img, opts.Quantize, opts.AlphaThreshold)
		if err != nil {
			return nil, err
		}
		opts.MaxColors = count.DistinctCount
		if opts.MaxColors < 1 {
			opts.MaxColors = 1
		}
		autoDetected = true
	}
	if opts.MaxColors > MaxDiscreteColors {
		opts.MaxColors = MaxDiscreteColors
	}

	palette, assignment, err := buildPalette(img, opts.MaxColors, opts.Quantize, opts.AlphaThreshold)
	if err != nil {
		return nil, err
	}

	traceParams := &gotrace.Params{
		TurdSize:     opts.TurdSize,
		TurnPolicy:   gotrace.TurnMinority,
		AlphaMax:     opts.AlphaMax,
		OptiCurve:    true,
		OptTolerance: 0.2,
	}

	var svgBody strings.Builder
	hexPalette := make([]string, len(palette))
	for idx, col := range palette {
		hexPalette[idx] = fmt.Sprintf("#%02X%02X%02X", col.R, col.G, col.B)

		mask := newColorMask(w, h, idx, assignment)
		paths, err := gotrace.Trace(mask, traceParams)
		if err != nil {
			return nil, fmt.Errorf("trace color %d: %w", idx, err)
		}
		if len(paths) == 0 {
			continue
		}

		fmt.Fprintf(&svgBody, `<path fill="%s" fill-rule="evenodd" d="`, hexPalette[idx])
		for _, p := range paths {
			svgBody.WriteString(p.ToSvgPath())
		}
		svgBody.WriteString(`"/>`)
	}

	var svg strings.Builder
	fmt.Fprintf(&svg,
		`<?xml version="1.0" encoding="UTF-8"?>`+"\n"+
			`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		w, h, w, h)
	svg.WriteString(svgBody.String())
	svg.WriteString(`</svg>`)

	svgStr := svg.String()
	return &VectorizeResult{
		Width:          w,
		Height:         h,
		ColorsUsed:     len(palette),
		Palette:        hexPalette,
		SVG:            svgStr,
		SVGBase64:      base64.StdEncoding.EncodeToString([]byte(svgStr)),
		MimeType:       "image/svg+xml",
		AutoDetected:   autoDetected,
		Quantize:       opts.Quantize,
		QuantizeAuto:   quantizeAuto,
		AlphaThreshold: opts.AlphaThreshold,
	}, nil
}

// buildPalette picks up to maxColors representative colors and returns, for each pixel,
// the index of the closest palette entry — or -1 if the pixel is below alphaThreshold
// and should be skipped by every layer.
func buildPalette(img image.Image, maxColors, quantize, alphaThreshold int) ([]RGBColor, []int, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	type bucket struct {
		key     uint32
		count   int
		r, g, b uint8
	}
	bucketMap := make(map[uint32]*bucket)
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
			bk, ok := bucketMap[key]
			if !ok {
				bk = &bucket{key: key, r: r8, g: g8, b: b8}
				bucketMap[key] = bk
			}
			bk.count++
		}
	}

	buckets := make([]*bucket, 0, len(bucketMap))
	for _, b := range bucketMap {
		buckets = append(buckets, b)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].count > buckets[j].count })

	if len(buckets) > maxColors {
		buckets = buckets[:maxColors]
	}
	if len(buckets) == 0 {
		return nil, nil, fmt.Errorf("image has no pixels with alpha >= %d", alphaThreshold)
	}

	palette := make([]RGBColor, len(buckets))
	for i, b := range buckets {
		palette[i] = RGBColor{R: b.r, G: b.g, B: b.b}
	}

	assignment := make([]int, w*h)
	for i := range assignment {
		assignment[i] = -1
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if int(a>>8) < alphaThreshold {
				continue
			}
			assignment[(y-bounds.Min.Y)*w+(x-bounds.Min.X)] = nearestColor(uint8(r>>8), uint8(g>>8), uint8(b>>8), palette)
		}
	}
	return palette, assignment, nil
}

func nearestColor(r, g, b uint8, palette []RGBColor) int {
	best := 0
	bestDist := 1 << 30
	for i, p := range palette {
		dr := int(r) - int(p.R)
		dg := int(g) - int(p.G)
		db := int(b) - int(p.B)
		d := dr*dr + dg*dg + db*db
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}

// newColorMask returns a gotrace bitmap whose set pixels are exactly those assigned to
// the given palette index. We render via an in-memory NRGBA image so we can lean on
// gotrace's existing alpha-threshold path.
func newColorMask(w, h, idx int, assignment []int) *gotrace.Bitmap {
	mask := image.NewNRGBA(image.Rect(0, 0, w, h))
	on := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	off := color.NRGBA{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if assignment[y*w+x] == idx {
				mask.SetNRGBA(x, y, on)
			} else {
				mask.SetNRGBA(x, y, off)
			}
		}
	}
	return gotrace.NewBitmapFromImage(mask, nil)
}
