//go:build cgo && linux

// Package ocr provides optical character recognition using Tesseract.
//
// On Linux with CGO enabled, this uses the gosseract library with native
// Tesseract bindings. Training data is embedded in the binary and extracted
// on first use - no external installation required.
package ocr

import (
	"embed"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/otiai10/gosseract/v2"
)

//go:embed tessdata/eng.traineddata
//go:embed tessdata/osd.traineddata
var embeddedTessdata embed.FS

var (
	tessdataDir  string
	tessdataOnce sync.Once
	tessdataErr  error
)

// ensureTessdata extracts embedded training data to disk if needed.
// Returns the path to the tessdata directory.
func ensureTessdata() (string, error) {
	tessdataOnce.Do(func() {
		tessdataDir, tessdataErr = extractTessdata()
	})
	return tessdataDir, tessdataErr
}

// extractTessdata extracts embedded tessdata files to a directory near the binary.
func extractTessdata() (string, error) {
	// Find where our binary lives
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Resolve symlinks to get actual binary location
	realExePath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exeDir = filepath.Dir(realExePath)
	}

	// tessdata directory next to the binary
	tessdataPath := filepath.Join(exeDir, "tessdata")

	// Check if already extracted
	engPath := filepath.Join(tessdataPath, "eng.traineddata")
	if _, err := os.Stat(engPath); err == nil {
		// Already exists, verify osd too
		osdPath := filepath.Join(tessdataPath, "osd.traineddata")
		if _, err := os.Stat(osdPath); err == nil {
			return tessdataPath, nil
		}
	}

	// Create tessdata directory
	if err := os.MkdirAll(tessdataPath, 0755); err != nil {
		// If we can't write next to binary, use temp directory
		tessdataPath = filepath.Join(os.TempDir(), "image-tools-mcp", "tessdata")
		if err := os.MkdirAll(tessdataPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create tessdata directory: %w", err)
		}
	}

	// Extract embedded files
	entries, err := embeddedTessdata.ReadDir("tessdata")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded tessdata: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join("tessdata", entry.Name())
		dstPath := filepath.Join(tessdataPath, entry.Name())

		// Skip if already exists with correct size
		if info, err := os.Stat(dstPath); err == nil {
			embeddedInfo, _ := fs.Stat(embeddedTessdata, srcPath)
			if info.Size() == embeddedInfo.Size() {
				continue
			}
		}

		data, err := embeddedTessdata.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", entry.Name(), err)
		}
	}

	return tessdataPath, nil
}

// Bounds represents a rectangular bounding box in pixel coordinates.
type Bounds struct {
	X1 int `json:"x1"` // Left edge
	Y1 int `json:"y1"` // Top edge
	X2 int `json:"x2"` // Right edge
	Y2 int `json:"y2"` // Bottom edge
}

// TextRegion represents a word or text block with its location and OCR confidence.
type TextRegion struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Bounds     Bounds  `json:"bounds"`
}

// OCRResult contains the complete results of text extraction from an image.
type OCRResult struct {
	FullText string       `json:"full_text"`
	Regions  []TextRegion `json:"regions"`
}

// DetectTextRegionsResult contains text region locations without the actual text content.
type DetectTextRegionsResult struct {
	Regions []TextRegionBox `json:"regions"`
	Count   int             `json:"count"`
}

// TextRegionBox represents a detected text region's location without its content.
type TextRegionBox struct {
	Bounds     Bounds  `json:"bounds"`
	Confidence float64 `json:"confidence"`
}

// ExtractText performs OCR on an entire image file and returns recognized text.
func ExtractText(imagePath string, language string) (*OCRResult, error) {
	tessdataPath, err := ensureTessdata()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tessdata: %w", err)
	}

	client := gosseract.NewClient()
	defer client.Close()

	// Set tessdata path
	if err := client.SetTessdataPrefix(tessdataPath); err != nil {
		return nil, fmt.Errorf("failed to set tessdata path: %w", err)
	}

	if err := client.SetImage(imagePath); err != nil {
		return nil, fmt.Errorf("failed to set image: %w", err)
	}

	if err := client.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	text, err := client.Text()
	if err != nil {
		return nil, fmt.Errorf("OCR failed: %w", err)
	}

	// Get word-level bounding boxes
	boxes, err := client.GetBoundingBoxes(gosseract.RIL_WORD)
	regions := make([]TextRegion, 0, len(boxes))
	if err == nil {
		for _, box := range boxes {
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
	}

	return &OCRResult{
		FullText: text,
		Regions:  regions,
	}, nil
}

// ExtractTextFromRegion performs OCR on a specific rectangular region of an image.
func ExtractTextFromRegion(img image.Image, x1, y1, x2, y2 int, language string) (*OCRResult, error) {
	// Clamp bounds
	bounds := img.Bounds()
	if x1 < bounds.Min.X {
		x1 = bounds.Min.X
	}
	if y1 < bounds.Min.Y {
		y1 = bounds.Min.Y
	}
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}

	// Create cropped image
	cropped := image.NewRGBA(image.Rect(0, 0, x2-x1, y2-y1))
	for cy := y1; cy < y2; cy++ {
		for cx := x1; cx < x2; cx++ {
			cropped.Set(cx-x1, cy-y1, img.At(cx, cy))
		}
	}

	// Save to temp file
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

// DetectTextRegions finds text regions in an image without performing full OCR.
func DetectTextRegions(imagePath string, minConfidence float64) (*DetectTextRegionsResult, error) {
	tessdataPath, err := ensureTessdata()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tessdata: %w", err)
	}

	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetTessdataPrefix(tessdataPath); err != nil {
		return nil, fmt.Errorf("failed to set tessdata path: %w", err)
	}

	if err := client.SetImage(imagePath); err != nil {
		return nil, fmt.Errorf("failed to set image: %w", err)
	}

	boxes, err := client.GetBoundingBoxes(gosseract.RIL_WORD)
	if err != nil {
		return nil, fmt.Errorf("failed to get bounding boxes: %w", err)
	}

	regions := make([]TextRegionBox, 0)
	for _, box := range boxes {
		confidence := float64(box.Confidence) / 100.0
		if confidence >= minConfidence {
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
	}

	return &DetectTextRegionsResult{
		Regions: regions,
		Count:   len(regions),
	}, nil
}

// TesseractVersion returns the installed Tesseract version.
func TesseractVersion() (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	return client.Version(), nil
}

// OCRInfo contains information about the OCR subsystem.
type OCRInfo struct {
	Available    bool   `json:"available"`
	Version      string `json:"version,omitempty"`
	Error        string `json:"error,omitempty"`
	Backend      string `json:"backend"`
	TessdataPath string `json:"tessdata_path,omitempty"`
}

// GetOCRInfo returns information about OCR availability.
func GetOCRInfo() OCRInfo {
	tessdataPath, tessdataErr := ensureTessdata()

	version, err := TesseractVersion()
	if err != nil {
		errMsg := err.Error()
		if tessdataErr != nil {
			errMsg = fmt.Sprintf("%s; tessdata error: %s", errMsg, tessdataErr.Error())
		}
		return OCRInfo{
			Available: false,
			Error:     errMsg,
			Backend:   "gosseract (embedded)",
		}
	}

	return OCRInfo{
		Available:    true,
		Version:      version,
		Backend:      "gosseract (embedded)",
		TessdataPath: tessdataPath,
	}
}

// SaveImageToTemp saves an image to a temporary PNG file and returns its path.
//
// This is a utility function for preparing images for external tools that
// require file paths.
//
// Parameters:
//   - img: The image to save.
//   - prefix: Filename prefix for identification (e.g., "ocr-region").
//
// Returns:
//   - string: Absolute path to the temporary file.
//   - error: Non-nil if file creation or encoding fails.
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

	if err := png.Encode(f, img); err != nil {
		return "", err
	}

	return tmpPath, nil
}
