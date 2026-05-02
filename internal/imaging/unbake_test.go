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
	// 97% threshold: tolerates the slight expansion at the disk boundary that
	// cell-level recovery introduces (a tradeoff for correctly handling icons
	// that contain pure-white or near-checker-colored regions; see
	// TestUnbakeTransparency_RecoversWhiteRegions).
	if matchPct < 97.0 {
		t.Errorf("alpha agreement %.2f%% < 97%%", matchPct)
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

// makeJpegLikeIcon synthesizes a checker + colored disk where the disk has
// realistic per-pixel color jitter (simulating JPEG noise). Used to verify
// that PreserveSourceColors keeps that jitter intact.
func makeJpegLikeIcon(w, h, period int, light, dark, fg color.NRGBA, radius int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	cx, cy := w/2, h/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cellX := x / period
			cellY := y / period
			bg := light
			if (cellX+cellY)%2 != 0 {
				bg = dark
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy < radius*radius {
				// Add deterministic per-pixel jitter — pseudo-noise that's stable across runs.
				jitter := (x*7 + y*13) % 11 - 5 // -5..+5
				img.SetNRGBA(x, y, color.NRGBA{
					R: clampU8(int(fg.R) + jitter),
					G: clampU8(int(fg.G) + jitter),
					B: clampU8(int(fg.B) + jitter),
					A: 255,
				})
			} else {
				img.SetNRGBA(x, y, bg)
			}
		}
	}
	return img
}

func clampU8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func TestUnbakeTransparency_PreserveSourceColors(t *testing.T) {
	light := color.NRGBA{R: 254, G: 254, B: 254, A: 255}
	dark := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	fg := color.NRGBA{R: 228, G: 183, B: 99, A: 255}
	img := makeJpegLikeIcon(400, 400, 25, light, dark, fg, 80)

	// Default: preservation on. Source color jitter must be visible in output.
	tmpDir := t.TempDir()
	resPreserve, err := UnbakeTransparency(img, filepath.Join(tmpDir, "preserve.png"), UnbakeOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Snap mode: preservation off. Output should be uniformly the palette color.
	off := false
	resSnap, err := UnbakeTransparency(img, filepath.Join(tmpDir, "snap.png"), UnbakeOptions{PreserveSourceColors: &off})
	if err != nil {
		t.Fatal(err)
	}

	// Read both back and count distinct opaque colors (excluding pure transparent).
	countDistinct := func(path string) int {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		decoded, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[uint32]struct{}{}
		bounds := decoded.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := decoded.At(x, y).RGBA()
				if a < 32768 {
					continue
				}
				key := uint32(uint8(r>>8))<<16 | uint32(uint8(g>>8))<<8 | uint32(uint8(b>>8))
				seen[key] = struct{}{}
			}
		}
		return len(seen)
	}

	preserveColors := countDistinct(resPreserve.OutputPath)
	snapColors := countDistinct(resSnap.OutputPath)
	if preserveColors <= snapColors {
		t.Errorf("preserve mode should show more distinct opaque colors than snap mode; got preserve=%d snap=%d", preserveColors, snapColors)
	}
	if snapColors > 5 {
		t.Errorf("snap mode should yield very few distinct colors (palette+small leakage); got %d", snapColors)
	}
}

func TestUnbakeTransparency_AmbiguousToForeground(t *testing.T) {
	// Use the real fixture image — its JPEG noise reliably produces ambiguous
	// pixels, which is what this option controls.
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
	on := true
	off := false
	resOn, err := UnbakeTransparency(img, filepath.Join(tmpDir, "ambig_on.png"), UnbakeOptions{AmbiguousToForeground: &on})
	if err != nil {
		t.Fatal(err)
	}
	resOff, err := UnbakeTransparency(img, filepath.Join(tmpDir, "ambig_off.png"), UnbakeOptions{AmbiguousToForeground: &off})
	if err != nil {
		t.Fatal(err)
	}

	// Count opaque pixels in each output PNG; default mode should have more
	// (it converts ambiguous pixels to opaque foreground).
	countOpaque := func(path string) int {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		decoded, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		bounds := decoded.Bounds()
		opaque := 0
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := decoded.At(x, y).RGBA()
				if a > 32768 {
					opaque++
				}
			}
		}
		return opaque
	}
	onOpaque := countOpaque(resOn.OutputPath)
	offOpaque := countOpaque(resOff.OutputPath)
	if onOpaque <= offOpaque {
		t.Errorf("ambiguous-to-fg=true should produce more opaque pixels: got on=%d off=%d", onOpaque, offOpaque)
	}
}

// makeIconWithWhiteFeatures synthesizes an icon that includes pure-white regions:
// (1) a white square fully enclosed by gold (case 2: enclosed background),
// (2) a white rectangle that extends to the image's left edge (case 1: white touching border).
// The rest of the icon is gold; the rest of the image is checker.
func makeIconWithWhiteFeatures(w, h, period int, light, dark, gold color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Default: checker bg
			cx := x / period
			cy := y / period
			bg := light
			if (cx+cy)%2 != 0 {
				bg = dark
			}
			img.SetNRGBA(x, y, bg)
		}
	}

	// Gold "body" — a large rectangle in the middle
	bodyX1, bodyY1, bodyX2, bodyY2 := w/4, h/4, 3*w/4, 3*h/4
	for y := bodyY1; y < bodyY2; y++ {
		for x := bodyX1; x < bodyX2; x++ {
			img.SetNRGBA(x, y, gold)
		}
	}

	// Case 2: white "eye" enclosed inside the gold body
	eyeCx, eyeCy, eyeR := w/2+w/8, h/2, w/16
	for y := eyeCy - eyeR; y < eyeCy+eyeR; y++ {
		for x := eyeCx - eyeR; x < eyeCx+eyeR; x++ {
			dx, dy := x-eyeCx, y-eyeCy
			if dx*dx+dy*dy < eyeR*eyeR {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}

	// Case 1: white rectangle extending from gold body to the LEFT image edge
	stripeY1, stripeY2 := h/2-period*2, h/2+period*2
	for y := stripeY1; y < stripeY2; y++ {
		for x := 0; x < bodyX1+period*3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}

	return img
}

func TestUnbakeTransparency_RecoversWhiteRegions(t *testing.T) {
	light := color.NRGBA{R: 254, G: 254, B: 254, A: 255}
	dark := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	gold := color.NRGBA{R: 228, G: 183, B: 99, A: 255}
	img := makeIconWithWhiteFeatures(400, 400, 25, light, dark, gold)

	// Both recovery passes are needed for this image. Case 2 is on by default;
	// case 1 (parity-pair) is opt-in because it can produce perimeter artifacts
	// on icons with irregular boundaries.
	on := true
	tmpDir := t.TempDir()
	res, err := UnbakeTransparency(img, filepath.Join(tmpDir, "out.png"), UnbakeOptions{
		RecoverColorMatchedIcon: &on,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Both recovery paths should have triggered.
	if res.PixelStats.EnclosedBackgroundFilled == 0 {
		t.Errorf("expected enclosed background fills (case 2: enclosed white eye), got 0")
	}
	if res.PixelStats.CellsRecovered == 0 {
		t.Errorf("expected cell recoveries (case 1: white stripe to border), got 0")
	}

	// Check specific pixels in the output:
	f, err := os.Open(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	// Eye center: should be opaque white
	eyeCx, eyeCy := 400/2+400/8, 400/2
	r, g, b, a := out.At(eyeCx, eyeCy).RGBA()
	if a < 32768 {
		t.Errorf("eye center (case 2): expected opaque, got α=%d", a>>8)
	}
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Errorf("eye center: expected near-white, got #%02X%02X%02X", r>>8, g>>8, b>>8)
	}

	// Stripe center (away from border): should be opaque white
	stripeCx, stripeCy := 50, 400/2 // well into the stripe, near left border
	_, _, _, sa := out.At(stripeCx, stripeCy).RGBA()
	if sa < 32768 {
		t.Errorf("stripe interior (case 1): expected opaque, got α=%d", sa>>8)
	}

	// Outer corner: should still be transparent (genuine background)
	_, _, _, ca := out.At(2, 2).RGBA()
	if ca > 32768 {
		t.Errorf("outer corner: expected transparent, got α=%d", ca>>8)
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
