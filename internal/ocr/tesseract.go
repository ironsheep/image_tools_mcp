package ocr

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/otiai10/gosseract/v2"
)

// Bounds represents a rectangular bounding box in pixel coordinates.
type Bounds struct {
	X1 int `json:"x1"` // Left edge
	Y1 int `json:"y1"` // Top edge
	X2 int `json:"x2"` // Right edge
	Y2 int `json:"y2"` // Bottom edge
}

// TextRegion represents a word or text block with its location and OCR confidence.
type TextRegion struct {
	// Text is the recognized text content.
	Text string `json:"text"`

	// Confidence is the OCR confidence score (0.0 to 1.0).
	// Higher values indicate more certain recognition.
	Confidence float64 `json:"confidence"`

	// Bounds is the bounding box around this text in the image.
	Bounds Bounds `json:"bounds"`
}

// OCRResult contains the complete results of text extraction from an image.
type OCRResult struct {
	// FullText is all recognized text as a single string with original spacing/newlines.
	FullText string `json:"full_text"`

	// Regions contains individual words with their bounding boxes and confidence scores.
	// May be empty if bounding box extraction fails (text will still be in FullText).
	Regions []TextRegion `json:"regions"`
}

// ExtractText performs OCR on an entire image file and returns recognized text.
//
// This function extracts all text from an image using Tesseract OCR, providing
// both the full text and individual word locations with confidence scores.
//
// Parameters:
//   - imagePath: Absolute path to the image file. Supports PNG, JPEG, TIFF, BMP.
//   - language: Tesseract language code (e.g., "eng" for English). The corresponding
//     language data must be installed on the system.
//
// Returns:
//   - *OCRResult: Contains FullText (complete recognized text) and Regions
//     (individual words with bounding boxes and confidence).
//   - error: Non-nil if the image cannot be loaded or OCR fails.
//
// # Word-Level Results
//
// The Regions field provides word-level granularity using Tesseract's RIL_WORD
// iterator level. Each region includes:
//   - The recognized word text
//   - Confidence score (0-1, where 1 is certain)
//   - Bounding box coordinates in the original image
//
// Empty words are filtered out from the results.
//
// # Error Handling
//
// If word-level bounding box extraction fails (which can happen with some
// Tesseract configurations), the function still returns the full text in
// FullText with an empty Regions slice.
//
// # Performance
//
// OCR is CPU-intensive. For large images or batch processing, consider:
//   - Cropping to regions of interest first
//   - Using lower resolution images when high precision isn't needed
//   - Running OCR in background goroutines for concurrent processing
func ExtractText(imagePath string, language string) (*OCRResult, error) {
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	if err := client.SetImage(imagePath); err != nil {
		return nil, fmt.Errorf("failed to set image: %w", err)
	}

	text, err := client.Text()
	if err != nil {
		return nil, fmt.Errorf("OCR failed: %w", err)
	}

	// Get bounding boxes for words
	boxes, err := client.GetBoundingBoxes(gosseract.RIL_WORD)
	if err != nil {
		// Return just text if boxes fail
		return &OCRResult{
			FullText: text,
			Regions:  []TextRegion{},
		}, nil
	}

	regions := make([]TextRegion, 0, len(boxes))
	for _, box := range boxes {
		if box.Word == "" {
			continue
		}
		regions = append(regions, TextRegion{
			Text:       box.Word,
			Confidence: float64(box.Confidence) / 100.0,
			Bounds: Bounds{
				X1: box.Box.Min.X,
				Y1: box.Box.Min.Y,
				X2: box.Box.Max.X,
				Y2: box.Box.Max.Y,
			},
		})
	}

	return &OCRResult{
		FullText: text,
		Regions:  regions,
	}, nil
}

// ExtractTextFromRegion performs OCR on a specific rectangular region of an image.
//
// This function extracts text only from the specified region, useful when you
// know where text is located or want to OCR a specific area without processing
// the entire image.
//
// Parameters:
//   - img: The source image (already loaded into memory).
//   - x1, y1: Top-left corner of the region (inclusive).
//   - x2, y2: Bottom-right corner of the region (exclusive).
//   - language: Tesseract language code (e.g., "eng").
//
// Returns:
//   - *OCRResult: Text extracted from the region. Bounding boxes in Regions are
//     adjusted to be relative to the original image (not the cropped region).
//   - error: Non-nil if cropping, temporary file creation, or OCR fails.
//
// # Coordinate Adjustment
//
// The returned bounding boxes are adjusted to the original image coordinates.
// For example, if the region starts at (100, 50) and a word is detected at
// (10, 20) within the cropped region, the returned bounds will be (110, 70).
//
// # Implementation Details
//
// This function:
//  1. Crops the specified region from the image
//  2. Saves the crop to a temporary PNG file
//  3. Runs Tesseract OCR on the temporary file
//  4. Deletes the temporary file
//  5. Adjusts bounding boxes to original image coordinates
//
// The temporary file is stored in the system's temp directory and is
// automatically cleaned up after OCR completes.
func ExtractTextFromRegion(img image.Image, x1, y1, x2, y2 int, language string) (*OCRResult, error) {
	// Crop the region
	cropped := imaging.Crop(img, image.Rect(x1, y1, x2, y2))

	// Save to temporary file (tesseract needs a file path)
	tmpFile, err := os.CreateTemp("", "ocr-region-*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := png.Encode(tmpFile, cropped); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to encode temp image: %w", err)
	}
	tmpFile.Close()

	// Perform OCR
	result, err := ExtractText(tmpPath, language)
	if err != nil {
		return nil, err
	}

	// Adjust bounds to be relative to original image
	for i := range result.Regions {
		result.Regions[i].Bounds.X1 += x1
		result.Regions[i].Bounds.Y1 += y1
		result.Regions[i].Bounds.X2 += x1
		result.Regions[i].Bounds.Y2 += y1
	}

	return result, nil
}

// DetectTextRegionsResult contains text region locations without the actual text content.
type DetectTextRegionsResult struct {
	// Regions is the list of detected text regions with bounding boxes.
	Regions []TextRegionBox `json:"regions"`

	// Count is the number of text regions detected.
	Count int `json:"count"`
}

// TextRegionBox represents a detected text region's location without its content.
//
// This is a lightweight result type for when you only need to know WHERE text
// is located, not WHAT it says. Use full OCR if you need the text content.
type TextRegionBox struct {
	// Bounds is the bounding box around the text region.
	Bounds Bounds `json:"bounds"`

	// Confidence is Tesseract's confidence score for this being a text region (0.0 to 1.0).
	Confidence float64 `json:"confidence"`
}

// DetectTextRegions finds text regions in an image without performing full OCR.
//
// This function is faster than full OCR when you only need to know WHERE text
// is located, not what it says. Useful for:
//   - Identifying text areas before selective OCR
//   - Layout analysis
//   - Finding regions to crop for later processing
//
// Parameters:
//   - imagePath: Absolute path to the image file.
//   - minConfidence: Minimum confidence threshold (0.0 to 1.0) for including
//     a region. Higher values return fewer, more certain regions.
//
// Returns:
//   - *DetectTextRegionsResult: Bounding boxes of detected text regions.
//   - error: Non-nil if the image cannot be loaded or Tesseract fails.
//
// # Block-Level Detection
//
// Uses Tesseract's RIL_BLOCK iterator level, which groups text into paragraph-like
// blocks. This is faster than word-level detection and provides larger regions
// suitable for subsequent detailed OCR.
//
// # Confidence Filtering
//
// Regions with confidence below minConfidence are excluded from results.
// Tesseract's confidence represents its certainty that the region contains text,
// not the accuracy of any text recognition.
func DetectTextRegions(imagePath string, minConfidence float64) (*DetectTextRegionsResult, error) {
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImage(imagePath); err != nil {
		return nil, fmt.Errorf("failed to set image: %w", err)
	}

	// Get bounding boxes at block level (faster than word level)
	boxes, err := client.GetBoundingBoxes(gosseract.RIL_BLOCK)
	if err != nil {
		return nil, fmt.Errorf("failed to get text regions: %w", err)
	}

	regions := make([]TextRegionBox, 0)
	for _, box := range boxes {
		confidence := float64(box.Confidence) / 100.0
		if confidence < minConfidence {
			continue
		}
		regions = append(regions, TextRegionBox{
			Bounds: Bounds{
				X1: box.Box.Min.X,
				Y1: box.Box.Min.Y,
				X2: box.Box.Max.X,
				Y2: box.Box.Max.Y,
			},
			Confidence: confidence,
		})
	}

	return &DetectTextRegionsResult{
		Regions: regions,
		Count:   len(regions),
	}, nil
}

// SaveImageToTemp saves an image to a temporary PNG file and returns its path.
//
// This is a utility function for preparing images for external tools that
// require file paths (like Tesseract).
//
// Parameters:
//   - img: The image to save.
//   - prefix: Filename prefix for identification (e.g., "ocr-region").
//
// Returns:
//   - string: Absolute path to the temporary file.
//   - error: Non-nil if file creation or encoding fails.
//
// The file is created in the system's temp directory with the format:
// <prefix>-<pid>.png
//
// IMPORTANT: The caller is responsible for deleting the temporary file
// after use with os.Remove().
func SaveImageToTemp(img image.Image, prefix string) (string, error) {
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("%s-%d.png", prefix, os.Getpid()))

	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	if _, err := f.Write(buf.Bytes()); err != nil {
		return "", err
	}

	return tmpPath, nil
}
