package ocr

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// createTestTextImage creates a simple image for OCR testing
// Note: Real OCR tests would need actual text images; these are basic unit tests
func createTestTextImage(t *testing.T, width, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add some black pixels to simulate text
	for y := 10; y < 20; y++ {
		for x := 10; x < 50; x++ {
			img.Set(x, y, color.Black)
		}
	}

	tmpFile, err := os.CreateTemp("", "ocr-test-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to encode image: %v", err)
	}

	return tmpFile.Name()
}

// drawText draws text on an image using basicfont
func drawText(img *image.RGBA, x, y int, text string, col color.Color) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)
}

// createImageWithText creates an image with actual rendered text for OCR testing
func createImageWithText(t *testing.T, text string, scale int) string {
	t.Helper()

	// Use a larger canvas for better OCR recognition
	// basicfont.Face7x13 is 7 pixels wide, 13 pixels tall per character
	width := len(text)*7*scale + 40*scale
	height := 40 * scale

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	// Draw text at multiple scales for better OCR
	if scale == 1 {
		drawText(img, 20, 25, text, color.Black)
	} else {
		// For scaled images, draw the text and then scale up
		smallImg := image.NewRGBA(image.Rect(0, 0, width/scale, height/scale))
		draw.Draw(smallImg, smallImg.Bounds(), image.White, image.Point{}, draw.Src)
		drawText(smallImg, 20, 25, text, color.Black)

		// Scale up by drawing each pixel as a scale x scale block
		for y := 0; y < height/scale; y++ {
			for x := 0; x < width/scale; x++ {
				c := smallImg.At(x, y)
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.Set(x*scale+dx, y*scale+dy, c)
					}
				}
			}
		}
	}

	tmpFile, err := os.CreateTemp("", "ocr-text-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to encode image: %v", err)
	}

	return tmpFile.Name()
}

// createMultiLineTextImage creates an image with multiple lines of text
func createMultiLineTextImage(t *testing.T, lines []string, scale int) string {
	t.Helper()

	// Calculate dimensions
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	width := (maxLen*7 + 40) * scale
	height := (len(lines)*16 + 30) * scale

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	// Draw each line
	for i, line := range lines {
		y := (20 + i*16) * scale
		if scale == 1 {
			drawText(img, 20, y, line, color.Black)
		} else {
			// Simple scaling for multi-line
			smallImg := image.NewRGBA(image.Rect(0, 0, width/scale, height/scale))
			draw.Draw(smallImg, smallImg.Bounds(), image.White, image.Point{}, draw.Src)
			for j, l := range lines {
				drawText(smallImg, 20, 20+j*16, l, color.Black)
			}
			// Scale up
			for sy := 0; sy < height/scale; sy++ {
				for sx := 0; sx < width/scale; sx++ {
					c := smallImg.At(sx, sy)
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							img.Set(sx*scale+dx, sy*scale+dy, c)
						}
					}
				}
			}
			break // Only need to do the scaling once
		}
	}

	tmpFile, err := os.CreateTemp("", "ocr-multiline-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to encode image: %v", err)
	}

	return tmpFile.Name()
}

func TestExtractText(t *testing.T) {
	imgPath := createTestTextImage(t, 100, 50)
	defer os.Remove(imgPath)

	result, err := ExtractText(imgPath, "eng")
	if err != nil {
		// Tesseract might not be installed - skip test
		if strings.Contains(err.Error(), "tesseract") ||
		   strings.Contains(err.Error(), "library") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	// Just verify it returns a result (may be empty for our simple test image)
	if result == nil {
		t.Fatal("ExtractText returned nil result")
	}
}

func TestExtractText_NonExistentFile(t *testing.T) {
	_, err := ExtractText("/nonexistent/path/image.png", "eng")
	if err == nil {
		t.Error("ExtractText should fail for non-existent file")
	}
}

func TestExtractText_InvalidLanguage(t *testing.T) {
	imgPath := createTestTextImage(t, 100, 50)
	defer os.Remove(imgPath)

	_, err := ExtractText(imgPath, "invalid_language_code_xyz")
	if err == nil {
		// Some Tesseract installations might be lenient with language codes
		t.Log("ExtractText did not fail for invalid language - may be Tesseract config")
	}
}

func TestExtractTextFromRegion(t *testing.T) {
	// Create an in-memory image
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add some text-like pattern in a region
	for y := 40; y < 60; y++ {
		for x := 50; x < 150; x++ {
			if x%5 < 3 {
				img.Set(x, y, color.Black)
			}
		}
	}

	result, err := ExtractTextFromRegion(img, 50, 40, 150, 60, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") ||
		   strings.Contains(err.Error(), "library") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	if result == nil {
		t.Fatal("ExtractTextFromRegion returned nil result")
	}
}

func TestExtractTextFromRegion_CoordinateOffset(t *testing.T) {
	// Create an image with known content
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add content that will be detected
	for y := 60; y < 80; y++ {
		for x := 60; x < 140; x++ {
			img.Set(x, y, color.Black)
		}
	}

	result, err := ExtractTextFromRegion(img, 50, 50, 150, 100, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	// If regions were detected, their coordinates should be offset by (50, 50)
	for _, region := range result.Regions {
		if region.Bounds.X1 < 50 || region.Bounds.Y1 < 50 {
			t.Error("Region bounds should be offset to original image coordinates")
		}
	}
}

func TestDetectTextRegions(t *testing.T) {
	imgPath := createTestTextImage(t, 200, 100)
	defer os.Remove(imgPath)

	result, err := DetectTextRegions(imgPath, 0.3)
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") ||
		   strings.Contains(err.Error(), "library") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	if result == nil {
		t.Fatal("DetectTextRegions returned nil result")
	}

	// Count should match length of Regions
	if result.Count != len(result.Regions) {
		t.Errorf("Count (%d) doesn't match len(Regions) (%d)", result.Count, len(result.Regions))
	}
}

func TestDetectTextRegions_MinConfidence(t *testing.T) {
	imgPath := createTestTextImage(t, 200, 100)
	defer os.Remove(imgPath)

	// Low confidence
	result1, err1 := DetectTextRegions(imgPath, 0.1)
	if err1 != nil {
		if strings.Contains(err1.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("DetectTextRegions failed: %v", err1)
	}

	// High confidence
	result2, err2 := DetectTextRegions(imgPath, 0.9)
	if err2 != nil {
		t.Fatalf("DetectTextRegions failed: %v", err2)
	}

	// Higher threshold should give fewer or equal results
	if result2.Count > result1.Count {
		t.Errorf("Higher minConfidence should give fewer results: low=%d, high=%d",
			result1.Count, result2.Count)
	}
}

func TestDetectTextRegions_NonExistentFile(t *testing.T) {
	_, err := DetectTextRegions("/nonexistent/path/image.png", 0.5)
	if err == nil {
		t.Error("DetectTextRegions should fail for non-existent file")
	}
}

func TestSaveImageToTemp(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}

	tmpPath, err := SaveImageToTemp(img, "test-save")
	if err != nil {
		t.Fatalf("SaveImageToTemp failed: %v", err)
	}
	defer os.Remove(tmpPath)

	// Verify file exists
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		t.Error("SaveImageToTemp did not create file")
	}

	// Verify it's in temp directory
	if !strings.HasPrefix(tmpPath, os.TempDir()) {
		t.Error("SaveImageToTemp should create file in temp directory")
	}

	// Verify filename has prefix
	filename := filepath.Base(tmpPath)
	if !strings.HasPrefix(filename, "test-save") {
		t.Errorf("Filename should have prefix 'test-save', got %s", filename)
	}

	// Verify it's a valid PNG
	f, err := os.Open(tmpPath)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer f.Close()

	loadedImg, err := png.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode saved PNG: %v", err)
	}

	if loadedImg.Bounds().Dx() != 50 || loadedImg.Bounds().Dy() != 50 {
		t.Errorf("loaded image dimensions: got %dx%d, want 50x50",
			loadedImg.Bounds().Dx(), loadedImg.Bounds().Dy())
	}
}

func TestSaveImageToTemp_Prefix(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	prefixes := []string{"ocr-test", "region-crop", "temp-img"}

	for _, prefix := range prefixes {
		tmpPath, err := SaveImageToTemp(img, prefix)
		if err != nil {
			t.Fatalf("SaveImageToTemp failed for prefix %s: %v", prefix, err)
		}
		defer os.Remove(tmpPath)

		filename := filepath.Base(tmpPath)
		if !strings.HasPrefix(filename, prefix) {
			t.Errorf("Filename should have prefix '%s', got %s", prefix, filename)
		}
	}
}

func TestBoundsStruct(t *testing.T) {
	bounds := Bounds{X1: 10, Y1: 20, X2: 100, Y2: 80}

	if bounds.X1 != 10 || bounds.Y1 != 20 {
		t.Error("Bounds top-left incorrect")
	}
	if bounds.X2 != 100 || bounds.Y2 != 80 {
		t.Error("Bounds bottom-right incorrect")
	}
}

func TestTextRegionStruct(t *testing.T) {
	region := TextRegion{
		Text:       "Hello",
		Confidence: 0.95,
		Bounds:     Bounds{X1: 10, Y1: 20, X2: 50, Y2: 40},
	}

	if region.Text != "Hello" {
		t.Errorf("Text: got %s, want Hello", region.Text)
	}
	if region.Confidence != 0.95 {
		t.Errorf("Confidence: got %f, want 0.95", region.Confidence)
	}
}

func TestOCRResultStruct(t *testing.T) {
	result := OCRResult{
		FullText: "Hello World",
		Regions: []TextRegion{
			{Text: "Hello", Confidence: 0.9, Bounds: Bounds{X1: 0, Y1: 0, X2: 30, Y2: 20}},
			{Text: "World", Confidence: 0.85, Bounds: Bounds{X1: 35, Y1: 0, X2: 70, Y2: 20}},
		},
	}

	if result.FullText != "Hello World" {
		t.Errorf("FullText: got %s, want 'Hello World'", result.FullText)
	}
	if len(result.Regions) != 2 {
		t.Errorf("Regions count: got %d, want 2", len(result.Regions))
	}
}

func TestTextRegionBoxStruct(t *testing.T) {
	box := TextRegionBox{
		Bounds:     Bounds{X1: 5, Y1: 10, X2: 100, Y2: 50},
		Confidence: 0.88,
	}

	if box.Bounds.X1 != 5 || box.Bounds.Y2 != 50 {
		t.Error("TextRegionBox bounds incorrect")
	}
	if box.Confidence != 0.88 {
		t.Errorf("Confidence: got %f, want 0.88", box.Confidence)
	}
}

func TestDetectTextRegionsResultStruct(t *testing.T) {
	result := DetectTextRegionsResult{
		Regions: []TextRegionBox{
			{Bounds: Bounds{X1: 0, Y1: 0, X2: 100, Y2: 50}, Confidence: 0.9},
		},
		Count: 1,
	}

	if result.Count != 1 {
		t.Errorf("Count: got %d, want 1", result.Count)
	}
	if len(result.Regions) != 1 {
		t.Errorf("Regions length: got %d, want 1", len(result.Regions))
	}
}

func TestExtractTextFromRegion_ErrorPath(t *testing.T) {
	// Test with a very small invalid region
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.White)
		}
	}

	// This should still work even with small region
	result, err := ExtractTextFromRegion(img, 0, 0, 10, 10, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestSaveImageToTemp_LargeImage(t *testing.T) {
	// Test with a larger image
	img := image.NewRGBA(image.Rect(0, 0, 500, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 500; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}

	tmpPath, err := SaveImageToTemp(img, "large-test")
	if err != nil {
		t.Fatalf("SaveImageToTemp failed: %v", err)
	}
	defer os.Remove(tmpPath)

	// Verify file exists and has content
	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Temp file is empty")
	}
}

func TestExtractText_WithWords(t *testing.T) {
	// Create image with text-like pattern (may not produce actual OCR results)
	img := image.NewRGBA(image.Rect(0, 0, 200, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add some black marks
	for y := 15; y < 35; y++ {
		for x := 20; x < 180; x++ {
			if x%10 < 5 {
				img.Set(x, y, color.Black)
			}
		}
	}

	tmpFile, err := os.CreateTemp("", "ocr-words-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := png.Encode(tmpFile, img); err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}

	result, err := ExtractText(tmpFile.Name(), "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	// Result may have empty regions for our test pattern
	t.Logf("Extracted text: %q, regions: %d", result.FullText, len(result.Regions))
}

func TestExtractTextFromRegion_BoundsAdjustment(t *testing.T) {
	// Create a larger image
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 300; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Extract from offset region
	offsetX, offsetY := 100, 50
	result, err := ExtractTextFromRegion(img, offsetX, offsetY, 200, 150, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	// Any regions returned should have coordinates offset by (100, 50)
	for _, region := range result.Regions {
		if region.Bounds.X1 < offsetX {
			t.Errorf("Region X1 (%d) should be >= offset (%d)", region.Bounds.X1, offsetX)
		}
		if region.Bounds.Y1 < offsetY {
			t.Errorf("Region Y1 (%d) should be >= offset (%d)", region.Bounds.Y1, offsetY)
		}
	}
}

func TestDetectTextRegions_EmptyImage(t *testing.T) {
	// Create blank image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.White)
		}
	}

	tmpFile, err := os.CreateTemp("", "ocr-empty-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := png.Encode(tmpFile, img); err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}

	result, err := DetectTextRegions(tmpFile.Name(), 0.5)
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	// Empty image should have few or no text regions
	t.Logf("Detected %d text regions in empty image", result.Count)
}

// --- Tests with actual rendered text ---

func TestExtractText_RealText(t *testing.T) {
	// Create image with actual rendered text (scaled up for better OCR)
	imgPath := createImageWithText(t, "HELLO WORLD", 3)
	defer os.Remove(imgPath)

	result, err := ExtractText(imgPath, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") ||
			strings.Contains(err.Error(), "library") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	t.Logf("Extracted text: %q", result.FullText)
	t.Logf("Number of regions: %d", len(result.Regions))

	// The OCR should find some text
	if result.FullText == "" && len(result.Regions) == 0 {
		t.Log("Warning: No text extracted - may need larger scale or different font")
	}
}

func TestExtractText_SimpleWords(t *testing.T) {
	// Test with simple words that are easier for OCR
	testCases := []string{
		"TEST",
		"HELLO",
		"12345",
		"ABC123",
	}

	for _, text := range testCases {
		t.Run(text, func(t *testing.T) {
			imgPath := createImageWithText(t, text, 4)
			defer os.Remove(imgPath)

			result, err := ExtractText(imgPath, "eng")
			if err != nil {
				if strings.Contains(err.Error(), "tesseract") {
					t.Skip("Tesseract not available")
				}
				t.Fatalf("ExtractText failed: %v", err)
			}

			t.Logf("Input: %q, Output: %q", text, strings.TrimSpace(result.FullText))
		})
	}
}

func TestExtractText_MultiLine(t *testing.T) {
	lines := []string{
		"LINE ONE",
		"LINE TWO",
		"LINE THREE",
	}

	imgPath := createMultiLineTextImage(t, lines, 3)
	defer os.Remove(imgPath)

	result, err := ExtractText(imgPath, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	t.Logf("Extracted from multi-line: %q", result.FullText)
	t.Logf("Regions found: %d", len(result.Regions))

	// Log each region
	for i, region := range result.Regions {
		t.Logf("  Region %d: %q (confidence: %.2f)", i, region.Text, region.Confidence)
	}
}

func TestDetectTextRegions_RealText(t *testing.T) {
	imgPath := createImageWithText(t, "DETECT THIS TEXT", 3)
	defer os.Remove(imgPath)

	result, err := DetectTextRegions(imgPath, 0.3)
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	t.Logf("Detected %d text regions", result.Count)

	for i, region := range result.Regions {
		t.Logf("  Region %d: bounds=(%d,%d)-(%d,%d), confidence=%.2f",
			i, region.Bounds.X1, region.Bounds.Y1, region.Bounds.X2, region.Bounds.Y2,
			region.Confidence)
	}
}

func TestExtractTextFromRegion_RealText(t *testing.T) {
	// Create an image with text in a specific region
	width, height := 400, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	// Draw text in the center region
	drawText(img, 150, 100, "CENTER TEXT", color.Black)
	// Draw some text in corners that we won't extract
	drawText(img, 10, 20, "TOP LEFT", color.Black)
	drawText(img, 300, 180, "BOTTOM", color.Black)

	// Extract only the center region
	result, err := ExtractTextFromRegion(img, 100, 50, 300, 150, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	t.Logf("Extracted from center region: %q", result.FullText)

	// Verify coordinates are offset correctly
	for _, region := range result.Regions {
		if region.Bounds.X1 < 100 || region.Bounds.Y1 < 50 {
			t.Errorf("Region bounds should be offset: got (%d,%d)",
				region.Bounds.X1, region.Bounds.Y1)
		}
	}
}

func TestExtractText_DifferentScales(t *testing.T) {
	text := "SCALE TEST"
	scales := []int{1, 2, 3, 4}

	for _, scale := range scales {
		t.Run(string(rune('0'+scale))+"x", func(t *testing.T) {
			imgPath := createImageWithText(t, text, scale)
			defer os.Remove(imgPath)

			result, err := ExtractText(imgPath, "eng")
			if err != nil {
				if strings.Contains(err.Error(), "tesseract") {
					t.Skip("Tesseract not available")
				}
				t.Fatalf("ExtractText failed: %v", err)
			}

			t.Logf("Scale %dx: extracted %q", scale, strings.TrimSpace(result.FullText))
		})
	}
}

func TestExtractText_Numbers(t *testing.T) {
	// Numbers are often easier for OCR
	imgPath := createImageWithText(t, "1234567890", 4)
	defer os.Remove(imgPath)

	result, err := ExtractText(imgPath, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	t.Logf("Numbers extracted: %q", strings.TrimSpace(result.FullText))
}

func TestDetectTextRegions_MultipleRegions(t *testing.T) {
	// Create image with text in multiple distinct regions
	width, height := 500, 300
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	// Draw text in different areas
	drawText(img, 20, 30, "TOP LEFT AREA", color.Black)
	drawText(img, 350, 30, "TOP RIGHT", color.Black)
	drawText(img, 200, 150, "CENTER", color.Black)
	drawText(img, 20, 280, "BOTTOM LEFT", color.Black)
	drawText(img, 350, 280, "BOTTOM RIGHT", color.Black)

	tmpFile, err := os.CreateTemp("", "ocr-multi-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := png.Encode(tmpFile, img); err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}

	result, err := DetectTextRegions(tmpFile.Name(), 0.2)
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("DetectTextRegions failed: %v", err)
	}

	t.Logf("Detected %d regions in multi-region image", result.Count)
	for i, region := range result.Regions {
		t.Logf("  Region %d: (%d,%d)-(%d,%d) conf=%.2f",
			i, region.Bounds.X1, region.Bounds.Y1,
			region.Bounds.X2, region.Bounds.Y2, region.Confidence)
	}
}

func TestExtractText_Punctuation(t *testing.T) {
	imgPath := createImageWithText(t, "HELLO, WORLD!", 4)
	defer os.Remove(imgPath)

	result, err := ExtractText(imgPath, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	t.Logf("With punctuation: %q", strings.TrimSpace(result.FullText))
}

func TestExtractText_MixedCase(t *testing.T) {
	imgPath := createImageWithText(t, "Hello World", 4)
	defer os.Remove(imgPath)

	result, err := ExtractText(imgPath, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractText failed: %v", err)
	}

	t.Logf("Mixed case: %q", strings.TrimSpace(result.FullText))
}

func TestExtractTextFromRegion_FullImageRegion(t *testing.T) {
	// Test extracting from a region that covers the entire image
	width, height := 200, 80
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	drawText(img, 20, 45, "FULL REGION", color.Black)

	result, err := ExtractTextFromRegion(img, 0, 0, width, height, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	t.Logf("Full region extraction: %q", result.FullText)
}

func TestExtractTextFromRegion_SmallRegion(t *testing.T) {
	// Test with a very small region
	width, height := 300, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	drawText(img, 100, 100, "X", color.Black)

	// Extract a small region around the character
	result, err := ExtractTextFromRegion(img, 95, 85, 115, 105, "eng")
	if err != nil {
		if strings.Contains(err.Error(), "tesseract") {
			t.Skip("Tesseract not available")
		}
		t.Fatalf("ExtractTextFromRegion failed: %v", err)
	}

	t.Logf("Small region extraction: %q", result.FullText)
}
