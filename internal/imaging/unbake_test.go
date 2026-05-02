package imaging

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// makeFakeTransparentIcon synthesizes a test image with known ground truth:
// a colored disk on a checker background of given period and colors. Returns
// the synthetic image plus a "ground truth" version (the disk on a true
// transparent background) for comparison.
func makeFakeTransparentIcon(w, h, period int, light, dark, fg color.NRGBA, radius int) (synth, truth *image.NRGBA) {
	synth = image.NewNRGBA(image.Rect(0, 0, w, h))
	truth = image.NewNRGBA(image.Rect(0, 0, w, h))
	cx, cy := w/2, h/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Background color from checker pattern
			cellX := x / period
			cellY := y / period
			bg := light
			if (cellX+cellY)%2 != 0 {
				bg = dark
			}
			// Is this pixel inside the disk?
			dx := x - cx
			dy := y - cy
			dist2 := dx*dx + dy*dy
			r2 := radius * radius
			if dist2 < r2 {
				synth.SetNRGBA(x, y, fg)
				truth.SetNRGBA(x, y, fg)
			} else {
				synth.SetNRGBA(x, y, bg)
				truth.SetNRGBA(x, y, color.NRGBA{}) // transparent
			}
		}
	}
	return
}

func TestDetectChecker_KnownPattern(t *testing.T) {
	light := color.NRGBA{R: 254, G: 254, B: 254, A: 255}
	dark := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	fg := color.NRGBA{R: 228, G: 183, B: 99, A: 255}
	img, _ := makeFakeTransparentIcon(400, 400, 25, light, dark, fg, 80)

	ck, err := DetectChecker(img)
	if err != nil {
		t.Fatalf("DetectChecker: %v", err)
	}
	if ck.Period != 25 {
		t.Errorf("period: got %d want 25", ck.Period)
	}
	if absInt(int(ck.Color1.R)-254) > 2 {
		t.Errorf("Color1.R: got %d want ~254", ck.Color1.R)
	}
	if absInt(int(ck.Color2.R)-240) > 2 {
		t.Errorf("Color2.R: got %d want ~240", ck.Color2.R)
	}
	if ck.Confidence < 1.0 {
		t.Errorf("confidence too low: %f", ck.Confidence)
	}
}

func TestUnbakeTransparency_KnownPattern(t *testing.T) {
	light := color.NRGBA{R: 254, G: 254, B: 254, A: 255}
	dark := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	fg := color.NRGBA{R: 228, G: 183, B: 99, A: 255}
	img, truth := makeFakeTransparentIcon(400, 400, 25, light, dark, fg, 80)

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.png")
	res, err := UnbakeTransparency(img, outPath, UnbakeOptions{})
	if err != nil {
		t.Fatalf("UnbakeTransparency: %v", err)
	}

	// Detected foreground should match (within tolerance) the synthetic gold.
	if len(res.DetectedForeground) == 0 {
		t.Fatal("no foreground detected")
	}
	gotFg := res.DetectedForeground[0].RGB
	if absInt(int(gotFg.R)-228) > 8 || absInt(int(gotFg.G)-183) > 8 || absInt(int(gotFg.B)-99) > 8 {
		t.Errorf("detected fg: got %v want ~(228,183,99)", gotFg)
	}

	// Output file must exist
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file: %v", err)
	}

	// Compare reconstructed image to truth pixel-by-pixel; require >= 99%
	// agreement on the alpha channel (some edge pixels may differ due to
	// classification subtleties — that's fine for a perfect synthetic).
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bounds := got.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	matchAlpha := 0
	totalAlpha := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, _, _, gotA := got.At(x, y).RGBA()
			_, _, _, truthA := truth.At(x, y).RGBA()
			gotOpaque := gotA > 32768
			truthOpaque := truthA > 32768
			totalAlpha++
			if gotOpaque == truthOpaque {
				matchAlpha++
			}
		}
	}
	matchPct := float64(matchAlpha) / float64(totalAlpha) * 100
	if matchPct < 99.0 {
		t.Errorf("alpha agreement %.2f%% < 99%%", matchPct)
	}
}

func TestUnbakeTransparency_RealImage(t *testing.T) {
	// Regression test against the actual fake-transparent fixture. Asserts
	// detected checker matches measured ground truth and reconstructed file is
	// produced with the expected foreground palette.
	path := "/workspaces/image_tools_mcp/testdata/fake-transparent-image.png"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("fixture not present")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "real.png")
	res, err := UnbakeTransparency(img, outPath, UnbakeOptions{})
	if err != nil {
		t.Fatalf("UnbakeTransparency: %v", err)
	}

	// Measured ground truth from probe: period=25, colors ~#FDFDFD/#F0F0F0
	if res.DetectedChecker.Period != 25 {
		t.Errorf("period: got %d want 25", res.DetectedChecker.Period)
	}
	if absInt(int(res.DetectedChecker.Color1.R)-253) > 3 ||
		absInt(int(res.DetectedChecker.Color2.R)-240) > 3 {
		t.Errorf("checker colors: got %v / %v", res.DetectedChecker.Color1, res.DetectedChecker.Color2)
	}

	// Should detect 2 distinct foreground colors (the two golds).
	if len(res.DetectedForeground) < 2 {
		t.Errorf("expected >= 2 foreground colors, got %d: %v", len(res.DetectedForeground), res.DetectedForeground)
	}
	// Light gold ~#E8B868 must be present.
	foundLight := false
	foundDark := false
	for _, c := range res.DetectedForeground {
		if absInt(int(c.RGB.R)-232) <= 16 && absInt(int(c.RGB.G)-184) <= 16 && absInt(int(c.RGB.B)-104) <= 16 {
			foundLight = true
		}
		if absInt(int(c.RGB.R)-168) <= 16 && absInt(int(c.RGB.G)-120) <= 16 && absInt(int(c.RGB.B)-40) <= 16 {
			foundDark = true
		}
	}
	if !foundLight {
		t.Errorf("light gold (~#E8B868) not in detected foreground: %v", res.DetectedForeground)
	}
	if !foundDark {
		t.Errorf("dark gold (~#A87828) not in detected foreground: %v", res.DetectedForeground)
	}

	// Background should be the largest classification bucket.
	if res.PixelStats.PureBackground < res.PixelStats.Total/2 {
		t.Errorf("pure_background %d not majority of total %d", res.PixelStats.PureBackground, res.PixelStats.Total)
	}

	// Output PNG should exist and be smaller than the source (transparency makes PNG much smaller).
	stat, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	srcStat, _ := os.Stat(path)
	if stat.Size() >= srcStat.Size() {
		t.Errorf("output size %d not smaller than source %d", stat.Size(), srcStat.Size())
	}
}

func TestParseHex(t *testing.T) {
	c, err := parseHex("#E4B763")
	if err != nil {
		t.Fatal(err)
	}
	if c.R != 0xE4 || c.G != 0xB7 || c.B != 0x63 {
		t.Errorf("got %v", c)
	}
	c, err = parseHex("a87828")
	if err != nil {
		t.Fatal(err)
	}
	if c.R != 0xA8 || c.G != 0x78 || c.B != 0x28 {
		t.Errorf("got %v", c)
	}
	if _, err := parseHex("nope"); err == nil {
		t.Error("expected error on bad hex")
	}
}
