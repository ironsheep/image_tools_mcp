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

// Bounds represents a bounding box
type Bounds struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

// TextRegion represents a detected text region with its content
type TextRegion struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Bounds     Bounds  `json:"bounds"`
}

// OCRResult contains the full OCR results
type OCRResult struct {
	FullText string       `json:"full_text"`
	Regions  []TextRegion `json:"regions"`
}

// ExtractText performs OCR on an entire image file
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

// ExtractTextFromRegion performs OCR on a specific region of an image
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

// DetectTextRegionsResult contains just the bounding boxes
type DetectTextRegionsResult struct {
	Regions []TextRegionBox `json:"regions"`
	Count   int             `json:"count"`
}

// TextRegionBox is a bounding box without text content
type TextRegionBox struct {
	Bounds     Bounds  `json:"bounds"`
	Confidence float64 `json:"confidence"`
}

// DetectTextRegions finds text regions without full OCR
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

// SaveImageToTemp saves an image to a temporary file and returns the path
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
