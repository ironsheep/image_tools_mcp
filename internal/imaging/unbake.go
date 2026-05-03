package imaging

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CheckerInfo describes a detected fake-transparency checker pattern.
//
// Pixel (x, y) belongs to a "light" cell when ((x-OriginX)/Period + (y-OriginY)/Period) % 2 == LightParity.
// Color1 is the color of light cells; Color2 is the color of dark cells.
type CheckerInfo struct {
	Color1       RGBColor `json:"color1"`        // light-cell color
	Color2       RGBColor `json:"color2"`        // dark-cell color
	Period       int      `json:"period"`        // pixels per square
	OriginX      int      `json:"origin_x"`      // x at which cell (0,0) starts
	OriginY      int      `json:"origin_y"`      // y at which cell (0,0) starts
	LightParity  int      `json:"light_parity"`  // 0 or 1; cell parity that maps to Color1
	Amplitude    int      `json:"amplitude"`     // |luma(Color1) - luma(Color2)| measured in clean corners
	Confidence   float64  `json:"confidence"`    // 0..1, autocorrelation strength at the chosen period
}

// UnbakeResult describes the reconstructed image.
type UnbakeResult struct {
	OutputPath          string         `json:"output_path"`
	Width               int            `json:"width"`
	Height              int            `json:"height"`
	DetectedChecker     *CheckerInfo   `json:"detected_checker,omitempty"`
	DetectedForeground  []CountedColor `json:"detected_foreground"`
	PixelStats          UnbakeStats    `json:"pixel_stats"`
	PreviewBase64       string         `json:"preview_base64"`
	PreviewMimeType     string         `json:"preview_mime_type"`
	Notes               []string       `json:"notes,omitempty"`
}

// UnbakeStats reports per-pixel classification counts.
type UnbakeStats struct {
	Total                    int `json:"total"`
	PureBackground           int `json:"pure_background"`
	PureForeground           int `json:"pure_foreground"`
	EdgeBlend                int `json:"edge_blend"`                 // recovered partial alpha
	Ambiguous                int `json:"ambiguous"`                  // could not classify cleanly
	AntiFringeAdded          int `json:"anti_fringe_added"`          // pixels forced to α=0 by edge enhancement
	CellsRecovered           int `json:"cells_recovered"`            // pixels recovered by parity-pair cell check
	EnclosedBackgroundFilled int `json:"enclosed_background_filled"` // pixels recovered by border flood-fill
}

// UnbakeOptions tunes the reconstruction pipeline.
type UnbakeOptions struct {
	// Checker, if non-nil, skips auto-detection and uses these values.
	Checker *CheckerInfo
	// ForegroundColors, if non-empty, overrides auto-detection. Hex strings ("#RRGGBB").
	ForegroundColors []string
	// BgTolerance is the max RGB Euclidean distance from a checker color to still
	// count a pixel as "pure background." Wider catches JPEG ringing halos. Default 28.
	BgTolerance int
	// FgTolerance is the max RGB distance from a foreground color to still count
	// a pixel as "pure foreground." Default 18.
	FgTolerance int
	// EdgeBlendTolerance is the max perpendicular distance from the line FG→BG
	// in RGB space for a pixel to count as an edge blend. Default 12.
	EdgeBlendTolerance float64
	// AntiFringeRadius extends an α=0 ring around the icon by this many pixels
	// to absorb leftover halo. Default 1. Pass -1 to disable.
	AntiFringeRadius int
	// MaxPreviewDim caps the longest side of the embedded base64 preview. Default 512.
	MaxPreviewDim int
	// PreserveSourceColors keeps the original source RGB on pure-fg and ambiguous-fg
	// pixels (only alpha is changed). Default true. When false, fg pixels are snapped
	// to the nearest detected palette entry — destructive but produces a "cleaned"
	// version of the icon with no JPEG color jitter. The vectorize pipeline benefits
	// from preservation: it can do its own quantization with full information.
	// Edge-blend pixels are always set to the canonical palette color (the alpha
	// recovery formula assumes you know the true foreground color).
	PreserveSourceColors *bool
	// AmbiguousToForeground treats pixels that don't fit any clean category as
	// foreground (preserving their source color, with α=255) rather than transparent.
	// Default true. Eliminates "pinhole" speckle inside the icon body that JPEG noise
	// produces. When false, the old conservative behavior applies (ambiguous pixels
	// become transparent unless much closer to fg than bg).
	AmbiguousToForeground *bool
	// RecoverColorMatchedIcon enables WHITE-region recovery for icons whose
	// content includes pure-white (or near-checker-colored) pixels. When TRUE:
	//   - Switches isBackgroundLike to predicted-color matching (more discriminating).
	//   - Runs a cell-level parity-pair pass that flips background cells with
	//     ≥2 foreground opposite-parity neighbors back to foreground with source
	//     RGB restored.
	// Default FALSE — opt-in. The parity-pair pass produces visible perimeter
	// artifacts on icons with irregular boundaries (every cell just outside the
	// icon adjacent to ≥2 fg opposite-parity cells gets flipped, creating a
	// blocky halo). Only enable when the source genuinely contains white or
	// other near-checker-colored regions you want preserved.
	RecoverColorMatchedIcon *bool
	// FillEnclosedBackground fills regions of "background" pixels that aren't
	// connected (4-connectivity) to the image border. These are necessarily
	// holes inside foreground regions and should be opaque (e.g. the white of an
	// eye in a face logo). Default true.
	FillEnclosedBackground *bool
}

// boolDefault returns *p if non-nil, else def. Used for tri-state options that
// distinguish "unset" (use default) from "explicitly false."
func boolDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// UnbakeTransparency reconstructs a transparent-background PNG from an image
// whose original transparency was flattened onto a checkerboard "fake transparency"
// pattern. Returns the result and writes the reconstructed image to outputPath.
//
// Pipeline: detect checker → identify foreground colors → classify each pixel
// (pure-bg / pure-fg / edge-blend / ambiguous) → solve for α on edge blends via
// inverse alpha compositing → optional anti-fringe extension → encode PNG.
func UnbakeTransparency(img image.Image, outputPath string, opts UnbakeOptions) (*UnbakeResult, error) {
	if opts.BgTolerance == 0 {
		// Default 28: matches v1.2.4-v1.2.6 permissive tolerance. Wide enough to
		// absorb JPEG halo (~7 luma units of distortion in the ring around the
		// icon) and slight color jitter on flattened checker patterns. When
		// RecoverColorMatchedIcon is enabled, this same tolerance is used with
		// predicted-color matching, which is more discriminating in practice
		// because it requires matching the *specific* color expected at each (x, y).
		opts.BgTolerance = 28
	}
	if opts.FgTolerance == 0 {
		opts.FgTolerance = 32
	}
	if opts.EdgeBlendTolerance == 0 {
		opts.EdgeBlendTolerance = 12
	}
	if opts.MaxPreviewDim == 0 {
		opts.MaxPreviewDim = 512
	}
	preserveColors := boolDefault(opts.PreserveSourceColors, true)
	ambiguousToFg := boolDefault(opts.AmbiguousToForeground, true)
	// Backward-compat: these flags no longer affect the pipeline. The new
	// "outer-bg-only" rule (applyOuterBackgroundOnly) subsumes their behavior.
	_ = opts.RecoverColorMatchedIcon
	_ = opts.FillEnclosedBackground
	_ = opts.AntiFringeRadius

	notes := []string{}

	// Stage 1: detect checker (or use supplied)
	checker := opts.Checker
	if checker == nil {
		c, err := DetectChecker(img)
		if err != nil {
			return nil, fmt.Errorf("checker detection failed: %w", err)
		}
		checker = c
		notes = append(notes, fmt.Sprintf("auto-detected checker: period=%d colors=#%02X%02X%02X/#%02X%02X%02X confidence=%.2f",
			checker.Period,
			checker.Color1.R, checker.Color1.G, checker.Color1.B,
			checker.Color2.R, checker.Color2.G, checker.Color2.B,
			checker.Confidence))
	}

	// Stage 2: identify foreground colors
	fgPalette, err := identifyForeground(img, checker, opts, false)
	if err != nil {
		return nil, err
	}
	if len(fgPalette) == 0 {
		return nil, fmt.Errorf("no foreground colors found above background; image may be empty")
	}

	// Stage 3 + 4: classify and reconstruct
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	stats := UnbakeStats{Total: w * h}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			px := RGBColor{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}

			outPx, cat := classifyPixel(px, x, y, checker, fgPalette, opts, preserveColors, ambiguousToFg, false)
			out.SetNRGBA(x, y, outPx)

			switch cat {
			case catPureBg:
				stats.PureBackground++
			case catPureFg:
				stats.PureForeground++
			case catEdgeBlend:
				stats.EdgeBlend++
			case catAmbiguous:
				stats.Ambiguous++
			}
		}
	}

	// Stage 3.5: "no transforms inside the icon" cleanup.
	//
	// The previous per-pixel classifier produced a draft that's correct in the
	// large but loses two classes of pixels: (a) icon interior regions that
	// happen to match a checker color (e.g. white eyes in a face logo, white
	// shirt under a rider's collar), and (b) checker pixels in any concave
	// pocket of the icon's silhouette that should be transparent.
	//
	// We fix this with one principled rule: only the "outer" background — the
	// connected region of checker pixels reachable from the image border —
	// should be transparent. Everything else keeps its source RGB at α=255.
	//
	// Two refinements:
	//   1. Per-edge gap closure (1D convex hull on each border): if foreground
	//      pixels touch a border at multiple disjoint segments, fill the gaps
	//      between the outermost fg pixels on that border so flood-fill can't
	//      escape through the gap. Handles the "white-extends-to-image-border"
	//      case (e.g. white shirt under collar reaching the right edge).
	//   2. Border flood-fill on the bg mask (4-connectivity). Pixels not
	//      reached are necessarily enclosed inside the icon → kept opaque.
	transformed := applyOuterBackgroundOnly(out, img, bounds, preserveColors, fgPalette)
	stats.EnclosedBackgroundFilled = transformed.enclosedRetained
	stats.CellsRecovered = transformed.borderGapsClosed

	// Encode PNG to disk
	if err := writePNG(outputPath, out); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	// Build base64 preview (downsampled if necessary)
	previewB64, err := encodePreview(out, opts.MaxPreviewDim)
	if err != nil {
		return nil, fmt.Errorf("failed to encode preview: %w", err)
	}

	// Build foreground palette report
	fgReport := make([]CountedColor, len(fgPalette))
	for i, c := range fgPalette {
		fgReport[i] = CountedColor{
			Hex: fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B),
			RGB: c,
		}
	}

	return &UnbakeResult{
		OutputPath:         outputPath,
		Width:              w,
		Height:             h,
		DetectedChecker:    checker,
		DetectedForeground: fgReport,
		PixelStats:         stats,
		PreviewBase64:      previewB64,
		PreviewMimeType:    "image/png",
		Notes:              notes,
	}, nil
}

// === Stage 1: checker detection ===

// DetectChecker auto-detects a regular two-color checker pattern in an image.
// Strategy:
//  1. Sample several horizontal and vertical luminance strips far from the image
//     center (where icon content is unlikely).
//  2. Compute 1D autocorrelation; identify the period as 2× the first strong
//     peak (since one period of the checker = light + dark = 2 squares).
//  3. Estimate the two band luminance centers by clustering corner pixels.
//  4. Verify parity: which cell index produces light vs dark.
func DetectChecker(img image.Image) (*CheckerInfo, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 64 || h < 64 {
		return nil, fmt.Errorf("image too small for checker detection (%dx%d)", w, h)
	}

	// Sample many rows and columns distributed across the image. The matched
	// filter scores ~2.0 for a perfect checker, ~0 for noise. We keep only
	// high-scoring detections.
	const minConfidence = 1.0
	const sampleStride = 16

	type detection struct {
		period int
		conf   float64
	}
	var detections []detection
	for y := sampleStride; y < h-sampleStride; y += sampleStride {
		strip := sampleLumaRow(img, bounds.Min.X, bounds.Min.Y+y, w)
		p, c := dominantPeriod(strip)
		if p > 0 && c >= minConfidence {
			detections = append(detections, detection{p, c})
		}
	}
	for x := sampleStride; x < w-sampleStride; x += sampleStride {
		strip := sampleLumaCol(img, bounds.Min.X+x, bounds.Min.Y, h)
		p, c := dominantPeriod(strip)
		if p > 0 && c >= minConfidence {
			detections = append(detections, detection{p, c})
		}
	}
	if len(detections) == 0 {
		return nil, fmt.Errorf("no high-confidence periodic pattern found")
	}

	// Mode of the detected periods.
	periods := make([]int, len(detections))
	confs := make([]float64, len(detections))
	for i, d := range detections {
		periods[i] = d.period
		confs[i] = d.conf
	}
	period := modeInt(periods)
	if period < 4 {
		return nil, fmt.Errorf("detected period too small: %d", period)
	}
	square := period / 2
	if square < 2 {
		return nil, fmt.Errorf("detected square size too small: %d", square)
	}
	confidence := medianFloat(confs)

	// Cluster corner pixels into 2 luminance bands.
	color1, color2, amplitude, err := clusterCornerColors(img, square)
	if err != nil {
		return nil, fmt.Errorf("could not cluster corner colors: %w", err)
	}

	// Determine origin + parity by checking which cell at (0,0) matches which color.
	// We sample the center of cell (0,0) at (square/2, square/2).
	cx, cy := bounds.Min.X+square/2, bounds.Min.Y+square/2
	r, g, b, _ := img.At(cx, cy).RGBA()
	probe := RGBColor{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
	d1 := rgbDist(probe, color1)
	d2 := rgbDist(probe, color2)
	lightParity := 0 // means cell (0,0) is color1 (light)
	if d2 < d1 {
		// Cell (0,0) matches color2; flip so color1 is still the "light" band.
		// Convention: ensure color1 has higher luma than color2.
		lightParity = 1
	}
	// Enforce convention: color1 = lighter
	if luma(color2) > luma(color1) {
		color1, color2 = color2, color1
		lightParity = 1 - lightParity
	}

	return &CheckerInfo{
		Color1:      color1,
		Color2:      color2,
		Period:      square,
		OriginX:     bounds.Min.X,
		OriginY:     bounds.Min.Y,
		LightParity: lightParity,
		Amplitude:   absInt(amplitude),
		Confidence:  confidence,
	}, nil
}

// sampleLumaRow extracts an integer luminance array along a horizontal strip.
func sampleLumaRow(img image.Image, x0, y, length int) []int {
	out := make([]int, length)
	for i := 0; i < length; i++ {
		r, g, b, _ := img.At(x0+i, y).RGBA()
		out[i] = lumaInt(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
	return out
}

func sampleLumaCol(img image.Image, x, y0, length int) []int {
	out := make([]int, length)
	for i := 0; i < length; i++ {
		r, g, b, _ := img.At(x, y0+i).RGBA()
		out[i] = lumaInt(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
	return out
}

func lumaInt(r, g, b uint8) int {
	// Rec. 709 weights, integer-scaled
	return (2126*int(r) + 7152*int(g) + 722*int(b)) / 10000
}

func luma(c RGBColor) int { return lumaInt(c.R, c.G, c.B) }

// dominantPeriod finds the dominant period in a 1D luminance signal using a
// square-wave matched filter — i.e., it directly scores how well the signal
// matches a checker pattern of each candidate period.
//
// Returns (full_period, normalized_strength) where full_period is one complete
// light→dark→light cycle (i.e. 2× the cell width for a checker).
//
// Why not pure autocorrelation: JPEG 8×8 block-boundary noise produces strong
// auto-correlation peaks at small lags that survive even harmonic verification.
// A matched filter explicitly tests "does this signal alternate between two
// sustained levels every P/2 pixels?" — which JPEG noise does not.
//
// Algorithm:
//  1. Reject strips with insufficient bimodal variance (single-cell-row strips
//     have only JPEG noise; their range from min to max is small).
//  2. For each candidate period P in [16, 256]:
//     - For each cycle, average the first half and second half separately.
//     - Score = mean absolute difference between the two halves, normalized
//       by the strip's overall stddev.
//  3. Return the period with the highest score.
func dominantPeriod(s []int) (int, float64) {
	n := len(s)
	if n < 64 {
		return 0, 0
	}
	// Compute basic stats
	var sum, sumSq, mn, mx int
	mn = s[0]
	mx = s[0]
	for _, v := range s {
		sum += v
		sumSq += v * v
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	mean := float64(sum) / float64(n)
	variance := float64(sumSq)/float64(n) - mean*mean
	if variance < 0 {
		variance = 0
	}
	stddev := math.Sqrt(variance)

	// Bimodality check: the strip must have a min/max range of at least 6 luma
	// units. Single-cell strips with only JPEG noise span maybe 3 units; real
	// checker strips span 10+.
	if mx-mn < 6 {
		return 0, 0
	}

	const minPeriod = 16  // smallest typical fake-transparency square is 8 px → period 16
	maxPeriod := n / 4
	if maxPeriod > 256 {
		maxPeriod = 256
	}

	bestPeriod := 0
	bestScore := 0.0
	for p := minPeriod; p <= maxPeriod; p += 2 { // period must be even (2× square width)
		half := p / 2
		var totalContrast float64
		var cycles int
		for start := 0; start+p <= n; start += p {
			var sumA, sumB int
			for i := 0; i < half; i++ {
				sumA += s[start+i]
				sumB += s[start+half+i]
			}
			avgA := float64(sumA) / float64(half)
			avgB := float64(sumB) / float64(half)
			totalContrast += math.Abs(avgA - avgB)
			cycles++
		}
		if cycles < 3 {
			continue // need a few cycles for confidence
		}
		avgContrast := totalContrast / float64(cycles)
		score := avgContrast / (stddev + 0.5) // normalize by signal energy
		if score > bestScore {
			bestScore = score
			bestPeriod = p
		}
	}
	return bestPeriod, bestScore
}

// clusterCornerColors looks at all four corner regions and finds the two
// dominant colors. Returns (lighter, darker, amplitude).
func clusterCornerColors(img image.Image, square int) (RGBColor, RGBColor, int, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	// Collect pixels from a margin region surrounding the image — assume the icon
	// occupies the central area.
	margin := square * 4
	if margin > w/4 {
		margin = w / 4
	}
	if margin > h/4 {
		margin = h / 4
	}
	if margin < square*2 {
		margin = square * 2
	}

	// Collect samples from the margin band
	type sample struct{ r, g, b, l int }
	samples := []sample{}
	collect := func(x, y int) {
		r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
		R, G, B := uint8(r>>8), uint8(g>>8), uint8(b>>8)
		// Only consider very-near-white desaturated pixels (typical fake-bg)
		if int(R) < 200 || int(G) < 200 || int(B) < 200 {
			return
		}
		mx := max3(R, G, B)
		mn := min3(R, G, B)
		if int(mx)-int(mn) > 12 { // saturated → not bg
			return
		}
		samples = append(samples, sample{int(R), int(G), int(B), lumaInt(R, G, B)})
	}
	for y := 0; y < margin; y++ {
		for x := 0; x < w; x++ {
			collect(x, y)
		}
	}
	for y := h - margin; y < h; y++ {
		for x := 0; x < w; x++ {
			collect(x, y)
		}
	}
	for y := margin; y < h-margin; y++ {
		for x := 0; x < margin; x++ {
			collect(x, y)
		}
		for x := w - margin; x < w; x++ {
			collect(x, y)
		}
	}

	if len(samples) < 100 {
		return RGBColor{}, RGBColor{}, 0, fmt.Errorf("not enough background samples (%d)", len(samples))
	}

	// k-means with k=2 on luminance
	sort.Slice(samples, func(i, j int) bool { return samples[i].l < samples[j].l })
	c1L := samples[len(samples)/4].l
	c2L := samples[len(samples)*3/4].l
	for iter := 0; iter < 20; iter++ {
		var s1R, s1G, s1B, s1Cnt int
		var s2R, s2G, s2B, s2Cnt int
		for _, s := range samples {
			if absInt(s.l-c1L) <= absInt(s.l-c2L) {
				s1R += s.r; s1G += s.g; s1B += s.b; s1Cnt++
			} else {
				s2R += s.r; s2G += s.g; s2B += s.b; s2Cnt++
			}
		}
		if s1Cnt == 0 || s2Cnt == 0 {
			break
		}
		newC1L := lumaInt(uint8(s1R/s1Cnt), uint8(s1G/s1Cnt), uint8(s1B/s1Cnt))
		newC2L := lumaInt(uint8(s2R/s2Cnt), uint8(s2G/s2Cnt), uint8(s2B/s2Cnt))
		if newC1L == c1L && newC2L == c2L {
			c1 := RGBColor{R: uint8(s1R / s1Cnt), G: uint8(s1G / s1Cnt), B: uint8(s1B / s1Cnt)}
			c2 := RGBColor{R: uint8(s2R / s2Cnt), G: uint8(s2G / s2Cnt), B: uint8(s2B / s2Cnt)}
			if luma(c2) > luma(c1) {
				c1, c2 = c2, c1
			}
			return c1, c2, absInt(luma(c1) - luma(c2)), nil
		}
		c1L, c2L = newC1L, newC2L
	}
	return RGBColor{}, RGBColor{}, 0, fmt.Errorf("k-means did not converge")
}

// === Stage 2: foreground identification ===

func identifyForeground(img image.Image, checker *CheckerInfo, opts UnbakeOptions, predictedOnly bool) ([]RGBColor, error) {
	if len(opts.ForegroundColors) > 0 {
		out := []RGBColor{}
		for _, h := range opts.ForegroundColors {
			c, err := parseHex(h)
			if err != nil {
				return nil, fmt.Errorf("bad foreground color %q: %w", h, err)
			}
			out = append(out, c)
		}
		return out, nil
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	// Histogram pixels that are clearly *not* background (far from both checker colors and saturated/dark).
	// Use a coarse quantize for clustering.
	quantize := 8
	type bucket struct {
		key     uint32
		count   int
		r, g, b uint8
	}
	buckets := map[uint32]*bucket{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			px := RGBColor{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
			if isBackgroundLike(px, x, y, opts.BgTolerance, checker, predictedOnly) {
				continue
			}
			rq := uint8((uint32(px.R) / uint32(quantize)) * uint32(quantize))
			gq := uint8((uint32(px.G) / uint32(quantize)) * uint32(quantize))
			bq := uint8((uint32(px.B) / uint32(quantize)) * uint32(quantize))
			key := uint32(rq)<<16 | uint32(gq)<<8 | uint32(bq)
			bk := buckets[key]
			if bk == nil {
				bk = &bucket{key: key, r: rq, g: gq, b: bq}
				buckets[key] = bk
			}
			bk.count++
		}
	}
	if len(buckets) == 0 {
		return nil, nil
	}
	all := make([]*bucket, 0, len(buckets))
	for _, b := range buckets {
		all = append(all, b)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].count > all[j].count })

	// Take top entries above 1% of foreground pixels, capped at MaxDiscreteColors.
	totalFg := 0
	for _, b := range all {
		totalFg += b.count
	}
	threshold := totalFg / 100
	if threshold < 50 {
		threshold = 50
	}
	out := []RGBColor{}
	for _, b := range all {
		if b.count < threshold {
			break
		}
		out = append(out, RGBColor{R: b.r, G: b.g, B: b.b})
		if len(out) >= MaxDiscreteColors {
			break
		}
	}
	// Cluster near-identical entries via single-link clustering. Two colors merge
	// if they're within 36 RGB units — wide enough to absorb JPEG bridge noise
	// (the "mid-gold" between light and dark gold caused by chroma subsampling),
	// narrow enough to keep genuinely distinct colors apart. Within a cluster,
	// the highest-frequency color (which is `out`'s order, since `out` came from
	// a frequency-sorted source) is chosen as the representative.
	type cluster struct {
		rep RGBColor
	}
	clusters := []cluster{}
	for _, c := range out {
		assigned := false
		for i, k := range clusters {
			if rgbDist(c, k.rep) < 36 {
				assigned = true
				_ = i
				break
			}
		}
		if !assigned {
			clusters = append(clusters, cluster{rep: c})
		}
	}
	deduped := make([]RGBColor, len(clusters))
	for i, k := range clusters {
		deduped[i] = k.rep
	}
	return deduped, nil
}

// isBackgroundLike returns true if the pixel is plausibly background.
//
// Two modes, controlled by the predictedOnly flag:
//
//   - predictedOnly=false (default): matches against EITHER checker color, plus a
//     wide luma-corridor check. Permissive — catches all background and JPEG halo.
//     Side effect: pure-white icon regions (#FFFFFF) get classified as background
//     since they're near the light checker color. The user must opt into
//     predictedOnly mode if their icon contains such regions.
//
//   - predictedOnly=true: matches only against the SPECIFIC color predicted at
//     this (x, y) cell location. Pure-white pixels in dark-cell positions then
//     classify as foreground (they're far from the dark checker color). Pure-white
//     pixels in light-cell positions still match the predicted light color and
//     classify as background — but the parity-pair cell recovery pass downstream
//     flips those by observing surrounding cells. Only sane to use together with
//     RecoverColorMatchedIcon.
func isBackgroundLike(p RGBColor, x, y int, bgTol int, ck *CheckerInfo, predictedOnly bool) bool {
	mx := max3(p.R, p.G, p.B)
	mn := min3(p.R, p.G, p.B)
	if int(mx) < 200 {
		return false
	}
	if int(mx)-int(mn) > 16 {
		return false // saturated → not bg
	}
	if predictedOnly {
		// Use a tighter tolerance for predicted-color matching — the predicted
		// color is more discriminating (no longer "either of two colors"), so
		// the tolerance can shrink. With the user's typical bgTol=28, predicted
		// matching uses bgTol-6=22, which keeps pure-white pixels (#FFFFFF) in
		// dark-cell positions (distance ~26 from #F0F0F0) classified as fg.
		tol := bgTol - 6
		if tol < 8 {
			tol = 8
		}
		predicted := bgAt(x, y, ck)
		return rgbDist(p, predicted) <= tol
	}
	// Permissive: match against either checker color, plus luma corridor.
	d1 := rgbDist(p, ck.Color1)
	d2 := rgbDist(p, ck.Color2)
	if d1 <= bgTol || d2 <= bgTol {
		return true
	}
	pl := luma(p)
	l1, l2 := luma(ck.Color1), luma(ck.Color2)
	if pl >= min(l1, l2)-bgTol && pl <= max(l1, l2)+bgTol {
		return true
	}
	return false
}

// === Stage 3: pixel classification + alpha recovery ===

type pixCategory int

const (
	catPureBg pixCategory = iota
	catPureFg
	catEdgeBlend
	catAmbiguous
)

func classifyPixel(p RGBColor, x, y int, ck *CheckerInfo, fg []RGBColor, opts UnbakeOptions, preserveColors, ambiguousToFg, predictedOnly bool) (color.NRGBA, pixCategory) {
	// 1. Background test (tolerant)
	if isBackgroundLike(p, x, y, opts.BgTolerance, ck, predictedOnly) {
		return color.NRGBA{}, catPureBg
	}

	// 2. Pure-foreground test
	bestFg := 0
	bestFgDist := math.MaxFloat64
	for i, f := range fg {
		d := float64(rgbDist(p, f))
		if d < bestFgDist {
			bestFgDist = d
			bestFg = i
		}
	}
	if bestFgDist <= float64(opts.FgTolerance) {
		out := color.NRGBA{R: fg[bestFg].R, G: fg[bestFg].G, B: fg[bestFg].B, A: 255}
		if preserveColors {
			out = color.NRGBA{R: p.R, G: p.G, B: p.B, A: 255}
		}
		return out, catPureFg
	}

	// 3. Edge-blend test: is p on the line segment FG → bg(x,y)?
	// Edge blends always use the canonical palette color — the alpha-recovery
	// formula α = ‖p−bg‖ / ‖fg−bg‖ assumes the foreground is the palette entry,
	// so the output color must be that entry for the result to be self-consistent
	// when re-composited.
	bg := bgAt(x, y, ck)
	bestF := -1
	bestAlpha := 0.0
	bestPerp := math.MaxFloat64
	for i, f := range fg {
		alpha, perp, ok := projectOnSegment(p, f, bg)
		if !ok {
			continue
		}
		if perp < bestPerp {
			bestPerp = perp
			bestF = i
			bestAlpha = alpha
		}
	}
	if bestF >= 0 && bestPerp <= opts.EdgeBlendTolerance {
		f := fg[bestF]
		a := bestAlpha
		if a < 0 {
			a = 0
		}
		if a > 1 {
			a = 1
		}
		// Tiny alphas → background to avoid leaving a foggy ring
		if a < 0.05 {
			return color.NRGBA{}, catPureBg
		}
		// Near-1 alphas → snap to opaque. Preserve source color if requested.
		if a > 0.95 {
			if preserveColors {
				return color.NRGBA{R: p.R, G: p.G, B: p.B, A: 255}, catPureFg
			}
			return color.NRGBA{R: f.R, G: f.G, B: f.B, A: 255}, catPureFg
		}
		return color.NRGBA{R: f.R, G: f.G, B: f.B, A: uint8(a * 255)}, catEdgeBlend
	}

	// 4. Ambiguous: non-bg pixels that don't fit any clean category. Default
	//    behavior (ambiguousToFg=true) treats them as foreground with α=255 and
	//    the source color preserved — they're meaningful color data even if we
	//    can't classify them precisely. Old behavior (ambiguousToFg=false) is
	//    conservative: only mark as fg if much closer to fg than bg.
	if ambiguousToFg {
		if preserveColors {
			return color.NRGBA{R: p.R, G: p.G, B: p.B, A: 255}, catAmbiguous
		}
		return color.NRGBA{R: fg[bestFg].R, G: fg[bestFg].G, B: fg[bestFg].B, A: 255}, catAmbiguous
	}
	d1 := rgbDist(p, ck.Color1)
	d2 := rgbDist(p, ck.Color2)
	closestBg := d1
	if d2 < closestBg {
		closestBg = d2
	}
	if bestFgDist+8 < float64(closestBg) {
		if preserveColors {
			return color.NRGBA{R: p.R, G: p.G, B: p.B, A: 255}, catAmbiguous
		}
		return color.NRGBA{R: fg[bestFg].R, G: fg[bestFg].G, B: fg[bestFg].B, A: 255}, catAmbiguous
	}
	return color.NRGBA{}, catAmbiguous
}

func bgAt(x, y int, ck *CheckerInfo) RGBColor {
	cx := (x - ck.OriginX) / ck.Period
	cy := (y - ck.OriginY) / ck.Period
	parity := ((cx + cy) % 2 + 2) % 2
	if parity == ck.LightParity {
		return ck.Color1
	}
	return ck.Color2
}

// projectOnSegment: given pixel p and endpoints F (foreground), B (background),
// find the fraction `alpha` along F→B such that (1-alpha)*B + alpha*F is closest to p.
// alpha=1 means pure F, alpha=0 means pure B. Returns (alpha, perpendicular_distance, ok).
// ok=false if F and B are too close to define a meaningful direction.
func projectOnSegment(p, f, b RGBColor) (float64, float64, bool) {
	fx, fy, fz := float64(f.R), float64(f.G), float64(f.B)
	bx, by, bz := float64(b.R), float64(b.G), float64(b.B)
	dx, dy, dz := fx-bx, fy-by, fz-bz
	denom := dx*dx + dy*dy + dz*dz
	if denom < 1 {
		return 0, 0, false
	}
	px, py, pz := float64(p.R)-bx, float64(p.G)-by, float64(p.B)-bz
	alpha := (px*dx + py*dy + pz*dz) / denom
	// Closest point on the line:
	cx := alpha * dx
	cy := alpha * dy
	cz := alpha * dz
	perp := math.Sqrt((px-cx)*(px-cx) + (py-cy)*(py-cy) + (pz-cz)*(pz-cz))
	return alpha, perp, true
}

// outerBgResult reports what applyOuterBackgroundOnly did.
type outerBgResult struct {
	enclosedRetained int // bg-classified pixels kept opaque because not connected to border
	borderGapsClosed int // bg pixels along image borders bridged between fg pixels
}

// applyOuterBackgroundOnly enforces the rule "only the outer background becomes
// transparent — every other pixel keeps its source RGB."
//
// Steps:
//  1. Build a foreground mask from the draft classification (any pixel with α>0).
//  2. Per-edge gap closure (1D convex hull on each image border): if foreground
//     pixels touch a border at multiple disjoint segments, fill the bg pixels
//     between the outermost fg pixels on that border so flood-fill can't escape
//     through the gap. Handles "white extends to image border" — e.g. a white
//     shirt under a rider's collar reaching the right edge.
//  3. Build the outer-bg mask via 4-connectivity flood-fill from any non-fg
//     border pixel (after gap closure).
//  4. Rewrite the output: pixels in outer-bg → fully transparent; pixels that
//     were classified as bg in the draft but are NOT outer-bg (i.e. enclosed)
//     → source RGB at α=255; everything else → unchanged from the draft (so
//     edge-blend partial alpha is preserved as a soft anti-aliased edge —
//     forcing it to α=255 was creating a near-white halo around the icon).
func applyOuterBackgroundOnly(out *image.NRGBA, src image.Image, srcBounds image.Rectangle, preserveColors bool, fg []RGBColor) outerBgResult {
	w, h := out.Bounds().Dx(), out.Bounds().Dy()

	// Snapshot the draft alpha channel BEFORE any mutation. We need to know
	// whether each pixel was classified as bg (α==0) so we can preserve
	// edge-blend partial alphas in the rewrite step instead of stomping them.
	draftAlpha := make([]uint8, w*h)
	isFg := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := out.NRGBAAt(x, y).A
			draftAlpha[y*w+x] = a
			if a > 0 {
				isFg[y*w+x] = true
			}
		}
	}

	// Per-edge gap closure: for each of the four image borders, find the
	// outermost fg pixels and mark every pixel between them as fg too.
	gapsClosed := 0
	closeAlong := func(positions []bool) int {
		first, last := -1, -1
		for i, v := range positions {
			if v {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 || last <= first {
			return 0
		}
		filled := 0
		for i := first; i <= last; i++ {
			if !positions[i] {
				positions[i] = true
				filled++
			}
		}
		return filled
	}
	{ // top edge
		row := make([]bool, w)
		for x := 0; x < w; x++ {
			row[x] = isFg[x]
		}
		gapsClosed += closeAlong(row)
		for x := 0; x < w; x++ {
			isFg[x] = row[x]
		}
	}
	{ // bottom edge
		row := make([]bool, w)
		base := (h - 1) * w
		for x := 0; x < w; x++ {
			row[x] = isFg[base+x]
		}
		gapsClosed += closeAlong(row)
		for x := 0; x < w; x++ {
			isFg[base+x] = row[x]
		}
	}
	{ // left edge
		col := make([]bool, h)
		for y := 0; y < h; y++ {
			col[y] = isFg[y*w]
		}
		gapsClosed += closeAlong(col)
		for y := 0; y < h; y++ {
			isFg[y*w] = col[y]
		}
	}
	{ // right edge
		col := make([]bool, h)
		for y := 0; y < h; y++ {
			col[y] = isFg[y*w+(w-1)]
		}
		gapsClosed += closeAlong(col)
		for y := 0; y < h; y++ {
			isFg[y*w+(w-1)] = col[y]
		}
	}

	// Flood-fill outer bg: BFS from every border pixel that is NOT fg.
	outerBg := make([]bool, w*h)
	queue := make([][2]int, 0, w*2+h*2)
	enqueueIfBg := func(x, y int) {
		if isFg[y*w+x] || outerBg[y*w+x] {
			return
		}
		outerBg[y*w+x] = true
		queue = append(queue, [2]int{x, y})
	}
	for x := 0; x < w; x++ {
		enqueueIfBg(x, 0)
		enqueueIfBg(x, h-1)
	}
	for y := 0; y < h; y++ {
		enqueueIfBg(0, y)
		enqueueIfBg(w-1, y)
	}
	for head := 0; head < len(queue); head++ {
		x, y := queue[head][0], queue[head][1]
		for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			idx := ny*w + nx
			if isFg[idx] || outerBg[idx] {
				continue
			}
			outerBg[idx] = true
			queue = append(queue, [2]int{nx, ny})
		}
	}

	// Rewrite output. Three cases per pixel:
	//   (a) outer-bg → fully transparent.
	//   (b) was-bg-but-enclosed → restore source RGB at α=255 (recovery: this
	//       pixel was misclassified as bg but is actually inside the icon).
	//   (c) was-fg with partial alpha (edge-blend) AND adjacent to outer-bg →
	//       keep the partial alpha (this is the icon's true outer edge, where
	//       a soft anti-aliased blend looks correct).
	//   (d) was-fg with partial alpha (edge-blend) NOT adjacent to outer-bg →
	//       interior boundary (e.g. between two icon colors, or the rim of an
	//       enclosed white region). The classifier mistook it for an icon→bg
	//       blend, but there's no actual bg here. Force fully opaque with
	//       source RGB so it doesn't appear as a translucent halo.
	//   (e) was-fully-opaque fg → leave unchanged.
	hasOuterBgNeighbor := func(x, y int) bool {
		for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			if outerBg[ny*w+nx] {
				return true
			}
		}
		return false
	}
	enclosed := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if outerBg[idx] {
				out.SetNRGBA(x, y, color.NRGBA{})
				continue
			}
			a := draftAlpha[idx]
			if a == 255 {
				continue // case (e): solid foreground, leave as-is
			}
			if a > 0 && a < 255 {
				// edge-blend
				if hasOuterBgNeighbor(x, y) {
					continue // case (c): legitimate outer edge, keep partial alpha
				}
				// case (d): interior edge-blend, restore source RGB at full alpha.
			}
			// cases (b) and (d): restore source RGB at α=255.
			r, g, b, _ := src.At(srcBounds.Min.X+x, srcBounds.Min.Y+y).RGBA()
			srcPx := RGBColor{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
			if !preserveColors && len(fg) > 0 {
				srcPx = nearestRGB(srcPx, fg)
			}
			if a == 0 {
				enclosed++
			}
			out.SetNRGBA(x, y, color.NRGBA{R: srcPx.R, G: srcPx.G, B: srcPx.B, A: 255})
		}
	}
	return outerBgResult{enclosedRetained: enclosed, borderGapsClosed: gapsClosed}
}

// nearestRGB returns the palette entry closest (Euclidean RGB) to p.
func nearestRGB(p RGBColor, palette []RGBColor) RGBColor {
	best := palette[0]
	bestD := math.MaxFloat64
	for _, c := range palette {
		d := float64(rgbDist(p, c))
		if d < bestD {
			bestD = d
			best = c
		}
	}
	return best
}

// === I/O helpers ===

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func encodePreview(img image.Image, maxDim int) (string, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	scaled := img
	if w > maxDim || h > maxDim {
		scale := float64(maxDim) / math.Max(float64(w), float64(h))
		nw, nh := int(float64(w)*scale), int(float64(h)*scale)
		scaled = nearestNeighborResize(img, nw, nh)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func nearestNeighborResize(src image.Image, w, h int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	bounds := src.Bounds()
	sw, sh := bounds.Dx(), bounds.Dy()
	for y := 0; y < h; y++ {
		sy := y * sh / h
		for x := 0; x < w; x++ {
			sx := x * sw / w
			r, g, b, a := src.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
			out.SetNRGBA(x, y, color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)})
		}
	}
	return out
}

// === Generic helpers ===

func parseHex(s string) (RGBColor, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return RGBColor{}, fmt.Errorf("expected #RRGGBB")
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		_, err = fmt.Sscanf(s, "%02X%02X%02X", &r, &g, &b)
	}
	if err != nil {
		return RGBColor{}, err
	}
	return RGBColor{R: r, G: g, B: b}, nil
}

func rgbDist(a, b RGBColor) int {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	d := dr*dr + dg*dg + db*db
	return int(math.Sqrt(float64(d)))
}

func max3(a, b, c uint8) uint8 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

func min3(a, b, c uint8) uint8 {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// modeInt returns the most-common value in s. Ties broken by lowest value.
// Returns 0 if s is empty.
func modeInt(s []int) int {
	if len(s) == 0 {
		return 0
	}
	counts := map[int]int{}
	for _, v := range s {
		counts[v]++
	}
	bestVal, bestCount := 0, 0
	for v, c := range counts {
		if c > bestCount || (c == bestCount && v < bestVal) {
			bestVal, bestCount = v, c
		}
	}
	return bestVal
}

func medianFloat(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	return c[len(c)/2]
}
