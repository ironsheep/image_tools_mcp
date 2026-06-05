//go:build cgo && linux

package ocr

import (
	"image"
	"image/draw"
	"strings"
	"testing"
)

// TestExtractTextFromImage_MatchesFilePath proves the in-memory OCR primitive
// returns the same result as the file-path entry point for identical pixels.
// ExtractTextFromImage is what ExtractTextFromRegion now builds on instead of a
// temp-file round-trip, so it must agree with ExtractText.
func TestExtractTextFromImage_MatchesFilePath(t *testing.T) {
	const want = "HELLO WORLD"
	img := renderTextRGBA(t, want)

	path := writeTempFixture(t, "ocr-inmem-*.png", img, encPNG)
	fromPath, err := ExtractText(path, "eng")
	if err != nil {
		if isTesseractUnavailable(err) {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText: %v", err)
	}
	fromImage, err := ExtractTextFromImage(img, "eng")
	if err != nil {
		t.Fatalf("ExtractTextFromImage: %v", err)
	}

	if a, b := strings.TrimSpace(fromPath.FullText), strings.TrimSpace(fromImage.FullText); a != b {
		t.Errorf("ExtractTextFromImage text = %q, want %q (must match the file-path entry point)", b, a)
	}
	if len(fromImage.Regions) != len(fromPath.Regions) {
		t.Errorf("ExtractTextFromImage region count = %d, want %d", len(fromImage.Regions), len(fromPath.Regions))
	}
}

// TestExtractTextFromRegion_OffsetAndParity proves the temp-file-free region
// path reads the text inside the region AND reports word boxes in the ORIGINAL
// image's coordinate space (offset by the crop origin), not in crop-local
// coordinates. This is the behavior-equivalence guard for the refactor that
// dropped the temp PNG + second decode.
func TestExtractTextFromRegion_OffsetAndParity(t *testing.T) {
	const want = "REGION TEXT"
	text := renderTextRGBA(t, want)
	tb := text.Bounds()

	// Place the text block at a known offset inside a larger white canvas, so a
	// correct result must offset its boxes well away from the origin.
	const offX, offY = 200, 150
	canvas := image.NewRGBA(image.Rect(0, 0, tb.Dx()+2*offX, tb.Dy()+2*offY))
	draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(canvas, tb.Add(image.Pt(offX, offY)), text, tb.Min, draw.Src)

	x1, y1 := offX, offY
	x2, y2 := offX+tb.Dx(), offY+tb.Dy()

	region, err := ExtractTextFromRegion(canvas, x1, y1, x2, y2, "eng")
	if err != nil {
		if isTesseractUnavailable(err) {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion: %v", err)
	}

	if got := strings.TrimSpace(region.FullText); got != want {
		t.Fatalf("region text = %q, want %q", got, want)
	}
	if len(region.Regions) == 0 {
		t.Fatal("region OCR returned no word boxes")
	}

	// Every box must land inside the region rectangle in original-image
	// coordinates. If the crop origin were not added back, boxes would start
	// near (0,0) — far left/above x1,y1 — and fail this.
	for _, r := range region.Regions {
		if r.Bounds.X1 < x1 || r.Bounds.Y1 < y1 || r.Bounds.X2 > x2 || r.Bounds.Y2 > y2 {
			t.Errorf("box %+v not within region [%d,%d,%d,%d] — crop offset not applied correctly",
				r.Bounds, x1, y1, x2, y2)
		}
	}
}
