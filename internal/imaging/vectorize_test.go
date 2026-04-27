package imaging

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// makeIconImage creates a small NRGBA image with a transparent background and
// `colors` distinct opaque colored regions arranged as vertical stripes.
func makeIconImage(w, h int, colors []color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	// Background is fully transparent by default for NRGBA.
	if len(colors) == 0 {
		return img
	}
	// Leave a transparent border on each side to verify the background is preserved.
	stripeW := (w - 2) / len(colors)
	for i, c := range colors {
		x0 := 1 + i*stripeW
		x1 := x0 + stripeW
		if i == len(colors)-1 {
			x1 = w - 1
		}
		for y := 1; y < h-1; y++ {
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	return img
}

func TestCountColors_IgnoresTransparentBackground(t *testing.T) {
	img := makeIconImage(40, 20, []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},
		{R: 0, G: 255, B: 0, A: 255},
	})

	res, err := CountColors(img, 8)
	if err != nil {
		t.Fatalf("CountColors error: %v", err)
	}
	if res.DistinctCount != 2 {
		t.Fatalf("expected 2 colors, got %d (%v)", res.DistinctCount, res.Colors)
	}
	if !res.IgnoredAlpha {
		t.Errorf("expected IgnoredAlpha=true")
	}
}

func TestCountColors_CapsAtMax(t *testing.T) {
	// Build 12 distinct colors — should cap at MaxDiscreteColors (10) with Truncated=true.
	colors := make([]color.NRGBA, 12)
	for i := range colors {
		colors[i] = color.NRGBA{R: uint8(i * 20), G: uint8(255 - i*15), B: uint8(i * 10), A: 255}
	}
	img := makeIconImage(60, 20, colors)

	res, err := CountColors(img, 1) // quantize=1 to keep colors distinct
	if err != nil {
		t.Fatalf("CountColors error: %v", err)
	}
	if res.DistinctCount != MaxDiscreteColors {
		t.Errorf("expected DistinctCount=%d, got %d", MaxDiscreteColors, res.DistinctCount)
	}
	if !res.Truncated {
		t.Errorf("expected Truncated=true")
	}
	if res.OtherPercentage <= 0 {
		t.Errorf("expected OtherPercentage > 0, got %f", res.OtherPercentage)
	}
}

func TestVectorize_AutoDetectsTwoColors(t *testing.T) {
	img := makeIconImage(40, 20, []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},
		{R: 0, G: 0, B: 255, A: 255},
	})

	res, err := Vectorize(img, VectorizeOptions{}) // MaxColors=0 → auto
	if err != nil {
		t.Fatalf("Vectorize error: %v", err)
	}
	if !res.AutoDetected {
		t.Errorf("expected AutoDetected=true")
	}
	if res.ColorsUsed != 2 {
		t.Errorf("expected 2 colors used, got %d (palette=%v)", res.ColorsUsed, res.Palette)
	}
	if !strings.Contains(res.SVG, "<svg") || !strings.Contains(res.SVG, "</svg>") {
		t.Errorf("SVG missing wrapper tags: %s", res.SVG)
	}
	if strings.Count(res.SVG, "<path ") != 2 {
		t.Errorf("expected 2 <path> elements, SVG was: %s", res.SVG)
	}
	if res.Width != 40 || res.Height != 20 {
		t.Errorf("expected 40x20, got %dx%d", res.Width, res.Height)
	}
	if res.SVGBase64 == "" {
		t.Errorf("expected non-empty SVGBase64")
	}
}

func TestVectorize_RespectsMaxColors(t *testing.T) {
	colors := []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
		{R: 255, G: 255, A: 255},
	}
	img := makeIconImage(40, 20, colors)

	res, err := Vectorize(img, VectorizeOptions{MaxColors: 2})
	if err != nil {
		t.Fatalf("Vectorize error: %v", err)
	}
	if res.ColorsUsed != 2 {
		t.Errorf("expected ColorsUsed=2, got %d", res.ColorsUsed)
	}
	if res.AutoDetected {
		t.Errorf("expected AutoDetected=false when MaxColors specified")
	}
}

func TestChooseQuantize_FewExactColors(t *testing.T) {
	// 3 distinct exact colors → should pick quantize=1
	img := makeIconImage(40, 20, []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
	})
	if q := ChooseQuantize(img); q != 1 {
		t.Errorf("expected quantize=1 for clean 3-color icon, got %d", q)
	}
}

func TestChooseQuantize_ManyExactColors(t *testing.T) {
	// 12 distinct exact colors → should fall back to DefaultQuantizeFallback
	colors := make([]color.NRGBA, 12)
	for i := range colors {
		colors[i] = color.NRGBA{R: uint8(i * 20), G: uint8(255 - i*15), B: uint8(i * 10), A: 255}
	}
	img := makeIconImage(60, 20, colors)
	if q := ChooseQuantize(img); q != DefaultQuantizeFallback {
		t.Errorf("expected fallback quantize=%d, got %d", DefaultQuantizeFallback, q)
	}
}

func TestVectorize_AutoQuantizePreservesExactColors(t *testing.T) {
	// Source has #F0F0F0 — with default auto-quantize this is now a clean
	// 2-color icon, so quantize should be 1 and the palette should match exactly.
	img := makeIconImage(40, 20, []color.NRGBA{
		{R: 0xF0, G: 0xF0, B: 0xF0, A: 255},
		{R: 0x12, G: 0x34, B: 0x56, A: 255},
	})
	res, err := Vectorize(img, VectorizeOptions{}) // no Quantize → auto
	if err != nil {
		t.Fatalf("Vectorize error: %v", err)
	}
	if !res.QuantizeAuto {
		t.Errorf("expected QuantizeAuto=true")
	}
	if res.Quantize != 1 {
		t.Errorf("expected Quantize=1 for clean low-color icon, got %d", res.Quantize)
	}
	gotPalette := map[string]bool{}
	for _, p := range res.Palette {
		gotPalette[p] = true
	}
	if !gotPalette["#F0F0F0"] || !gotPalette["#123456"] {
		t.Errorf("expected exact source colors in palette, got %v", res.Palette)
	}
}

func TestVectorize_ExplicitQuantizeRespected(t *testing.T) {
	img := makeIconImage(40, 20, []color.NRGBA{{R: 0xF0, A: 255}, {G: 0xF0, A: 255}})
	res, err := Vectorize(img, VectorizeOptions{Quantize: 16})
	if err != nil {
		t.Fatalf("Vectorize error: %v", err)
	}
	if res.QuantizeAuto {
		t.Errorf("expected QuantizeAuto=false when explicit value passed")
	}
	if res.Quantize != 16 {
		t.Errorf("expected Quantize=16, got %d", res.Quantize)
	}
}

func TestVectorize_ClampsToMaxDiscreteColors(t *testing.T) {
	img := makeIconImage(40, 20, []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}})

	res, err := Vectorize(img, VectorizeOptions{MaxColors: 99})
	if err != nil {
		t.Fatalf("Vectorize error: %v", err)
	}
	// Only 2 distinct colors exist; should not exceed that.
	if res.ColorsUsed > MaxDiscreteColors {
		t.Errorf("ColorsUsed exceeded cap: %d", res.ColorsUsed)
	}
}
