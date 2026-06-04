//go:build cgo && linux

package ocr

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"
)

// renderTextRGBA builds a white-background RGBA image with black text rendered
// large enough for reliable OCR. It is the common source for the edge-format
// fixtures below so every format encodes the same pixels.
func renderTextRGBA(t *testing.T, text string) *image.RGBA {
	t.Helper()
	const scale = 4
	w := len(text)*7*scale + 40*scale
	h := 40 * scale

	small := image.NewRGBA(image.Rect(0, 0, w/scale, h/scale))
	draw.Draw(small, small.Bounds(), image.White, image.Point{}, draw.Src)
	drawText(small, 20, 25, text, color.Black)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h/scale; y++ {
		for x := 0; x < w/scale; x++ {
			c := small.At(x, y)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.Set(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	return img
}

// writeTempFixture encodes img with enc into a temp file and returns its path.
func writeTempFixture(t *testing.T, pattern string, img image.Image, enc func(*os.File, image.Image) error) string {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("create temp fixture: %v", err)
	}
	if err := enc(f, img); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("encode fixture %s: %v", pattern, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("close fixture %s: %v", pattern, err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func encPNG(f *os.File, img image.Image) error { return png.Encode(f, img) }
func encJPEG(f *os.File, img image.Image) error {
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 95})
}
func encGIF(f *os.File, img image.Image) error { return gif.Encode(f, img, nil) }

func encGray(f *os.File, img image.Image) error {
	b := img.Bounds()
	g := image.NewGray(b)
	draw.Draw(g, b, img, b.Min, draw.Src)
	return png.Encode(f, g)
}

// samePixels reports whether two images have identical bounds and identical
// RGBA values at every pixel.
func samePixels(a, b image.Image) (bool, string) {
	if a.Bounds() != b.Bounds() {
		return false, "bounds differ"
	}
	bnds := a.Bounds()
	for y := bnds.Min.Y; y < bnds.Max.Y; y++ {
		for x := bnds.Min.X; x < bnds.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				return false, fmt.Sprintf("pixel mismatch at (%d,%d)", x, y)
			}
		}
	}
	return true, ""
}

// TestLoadImageAsPNG_PixelIdentical is the fidelity-invariant regression test:
// re-encoding a known PNG fixture through the OCR decode path must yield pixels
// byte-identical to the source. PNG is lossless — any divergence here means the
// decode path is silently altering the image OCR sees.
func TestLoadImageAsPNG_PixelIdentical(t *testing.T) {
	const fixture = "../../testdata/simple_diagram.png"

	srcBytes, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	src, err := png.Decode(bytes.NewReader(srcBytes))
	if err != nil {
		t.Fatalf("decode source: %v", err)
	}

	pngBytes, err := loadImageAsPNG(fixture)
	if err != nil {
		t.Fatalf("loadImageAsPNG: %v", err)
	}
	got, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode re-encoded PNG: %v", err)
	}

	if ok, why := samePixels(src, got); !ok {
		t.Fatalf("re-encoded PNG is not pixel-identical to source: %s", why)
	}
}

// TestOCR_FormatParity proves the loader-decode path gives OCR every format the
// loader supports, and that lossless formats reproduce the PNG reference exactly.
// The PNG result is the reference "baseline"; JPEG/GIF/grayscale must match its
// recognized text.
func TestOCR_FormatParity(t *testing.T) {
	const want = "HELLO WORLD"
	img := renderTextRGBA(t, want)

	pngPath := writeTempFixture(t, "ocr-parity-*.png", img, encPNG)
	ref, err := ExtractText(pngPath, "eng")
	if err != nil {
		if isTesseractUnavailable(err) {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("PNG reference OCR failed: %v", err)
	}
	refText := strings.TrimSpace(ref.FullText)
	if refText != want {
		t.Fatalf("PNG reference text = %q, want %q (fixture/OCR setup problem)", refText, want)
	}

	cases := []struct {
		name     string
		path     string
		lossless bool // lossless formats must also reproduce the reference boxes
	}{
		{"jpeg", writeTempFixture(t, "ocr-parity-*.jpg", img, encJPEG), false},
		{"gif", writeTempFixture(t, "ocr-parity-*.gif", img, encGIF), true},
		{"grayscale", writeTempFixture(t, "ocr-parity-*.png", img, encGray), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ExtractText(tc.path, "eng")
			if err != nil {
				t.Fatalf("%s OCR failed: %v", tc.name, err)
			}
			if got := strings.TrimSpace(res.FullText); got != refText {
				t.Errorf("%s text = %q, want %q (matches PNG reference)", tc.name, got, refText)
			}
			if tc.lossless {
				if len(res.Regions) != len(ref.Regions) {
					t.Errorf("%s region count = %d, want %d", tc.name, len(res.Regions), len(ref.Regions))
				}
			}
		})
	}
}

// TestOCR_LargeImageNoDownsample OCRs a multi-megapixel image and asserts the
// decode path preserves full resolution — the re-encoded PNG must keep the
// source dimensions, and OCR must still read the text. This guards the fidelity
// invariant against any accidental downsampling.
func TestOCR_LargeImageNoDownsample(t *testing.T) {
	const w, h = 1600, 1200 // ~1.9 MP
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
	text := renderTextRGBA(t, "LARGE IMAGE")
	draw.Draw(canvas, text.Bounds().Add(image.Pt(40, 40)), text, text.Bounds().Min, draw.Src)

	path := writeTempFixture(t, "ocr-large-*.png", canvas, encPNG)

	// Re-encode through the OCR path and confirm dimensions are preserved.
	pngBytes, err := loadImageAsPNG(path)
	if err != nil {
		t.Fatalf("loadImageAsPNG: %v", err)
	}
	got, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode re-encoded PNG: %v", err)
	}
	if got.Bounds().Dx() != w || got.Bounds().Dy() != h {
		t.Fatalf("re-encoded dimensions = %dx%d, want %dx%d (downsampling detected)",
			got.Bounds().Dx(), got.Bounds().Dy(), w, h)
	}

	res, err := ExtractText(path, "eng")
	if err != nil {
		if isTesseractUnavailable(err) {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("large-image OCR failed: %v", err)
	}
	if !strings.Contains(res.FullText, "LARGE") || !strings.Contains(res.FullText, "IMAGE") {
		t.Errorf("large-image OCR text = %q, want it to contain LARGE and IMAGE", strings.TrimSpace(res.FullText))
	}
}

// TestOCR_UndecodableBytes confirms that feeding non-image bytes yields a clean
// error rather than a panic — the decode happens in Go (loader) before Tesseract
// is ever touched.
func TestOCR_UndecodableBytes(t *testing.T) {
	f, err := os.CreateTemp("", "ocr-garbage-*.png")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString("this is not a valid image"); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	f.Close()

	if _, err := ExtractText(f.Name(), "eng"); err == nil {
		t.Error("ExtractText should return an error for undecodable bytes")
	}
	if _, err := DetectTextRegions(f.Name(), 0.5); err == nil {
		t.Error("DetectTextRegions should return an error for undecodable bytes")
	}
}

// isTesseractUnavailable mirrors the skip guard used across this package so the
// format tests degrade gracefully where Tesseract is not installed.
func isTesseractUnavailable(err error) bool {
	return strings.Contains(err.Error(), "tesseract") || strings.Contains(err.Error(), "library")
}
