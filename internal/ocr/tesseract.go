//go:build !cgo || !linux

// Package ocr provides optical character recognition using the Tesseract CLI.
//
// On macOS and Windows (or when CGO is disabled), this package shells out to
// the `tesseract` command-line tool, which must be installed separately.
//
// # Installation
//
// Install Tesseract for your platform:
//
//   - macOS: brew install tesseract
//   - Windows: Download from https://github.com/UB-Mannheim/tesseract/wiki
//
// # Language Data
//
// Tesseract requires language training data. English (eng) is usually included
// by default. For other languages:
//
//   - macOS: brew install tesseract-lang
//   - Windows: Select languages during installation
package ocr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

// DetectTextRegionsResult contains text region locations without the actual text content.
type DetectTextRegionsResult struct {
	// Regions is the list of detected text regions with bounding boxes.
	Regions []TextRegionBox `json:"regions"`

	// Count is the number of text regions detected.
	Count int `json:"count"`
}

// TextRegionBox represents a detected text region's location without its content.
type TextRegionBox struct {
	// Bounds is the bounding box around the text region.
	Bounds Bounds `json:"bounds"`

	// Confidence is Tesseract's confidence score for this being a text region (0.0 to 1.0).
	Confidence float64 `json:"confidence"`
}

// ErrTesseractNotFound is returned when the tesseract CLI is not installed.
type ErrTesseractNotFound struct {
	Platform string
}

func (e ErrTesseractNotFound) Error() string {
	instructions := map[string]string{
		"darwin":  "brew install tesseract",
		"linux":   "sudo apt install tesseract-ocr  # or: sudo dnf install tesseract",
		"windows": "Download from https://github.com/UB-Mannheim/tesseract/wiki",
	}

	inst, ok := instructions[e.Platform]
	if !ok {
		inst = "Visit https://tesseract-ocr.github.io/tessdoc/Installation.html"
	}

	return fmt.Sprintf("tesseract not found in PATH. Install with: %s", inst)
}

// findTesseract locates the tesseract executable.
func findTesseract() (string, error) {
	// Check common locations
	path, err := exec.LookPath("tesseract")
	if err == nil {
		return path, nil
	}

	// Windows-specific paths
	if runtime.GOOS == "windows" {
		commonPaths := []string{
			`C:\Program Files\Tesseract-OCR\tesseract.exe`,
			`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
		}
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", ErrTesseractNotFound{Platform: runtime.GOOS}
}

// ExtractText performs OCR on an entire image file and returns recognized text.
//
// This function extracts all text from an image using the Tesseract CLI, providing
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
//   - error: Non-nil if tesseract is not installed, the image cannot be loaded, or OCR fails.
func ExtractText(imagePath string, language string) (*OCRResult, error) {
	tesseract, err := findTesseract()
	if err != nil {
		return nil, err
	}

	// Verify file exists
	if _, err := os.Stat(imagePath); err != nil {
		return nil, fmt.Errorf("image file not found: %w", err)
	}

	// Get full text
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(tesseract, imagePath, "stdout", "-l", language)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tesseract failed: %v: %s", err, stderr.String())
	}

	fullText := strings.TrimSpace(stdout.String())

	// Get word-level bounding boxes using TSV output
	regions, _ := extractRegionsWithTSV(tesseract, imagePath, language)

	return &OCRResult{
		FullText: fullText,
		Regions:  regions,
	}, nil
}

// extractRegionsWithTSV gets word-level bounding boxes using tesseract's TSV output.
func extractRegionsWithTSV(tesseract, imagePath, language string) ([]TextRegion, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(tesseract, imagePath, "stdout", "-l", language, "tsv")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tesseract TSV failed: %v", err)
	}

	regions := []TextRegion{}
	lines := strings.Split(stdout.String(), "\n")

	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 12 {
			continue
		}

		// TSV fields: level, page_num, block_num, par_num, line_num, word_num,
		//             left, top, width, height, conf, text
		text := strings.TrimSpace(fields[11])
		if text == "" {
			continue
		}

		conf, _ := strconv.ParseFloat(fields[10], 64)
		left, _ := strconv.Atoi(fields[6])
		top, _ := strconv.Atoi(fields[7])
		width, _ := strconv.Atoi(fields[8])
		height, _ := strconv.Atoi(fields[9])

		// Skip low confidence or invalid entries
		if conf < 0 {
			continue
		}

		regions = append(regions, TextRegion{
			Text:       text,
			Confidence: conf / 100.0,
			Bounds: Bounds{
				X1: left,
				Y1: top,
				X2: left + width,
				Y2: top + height,
			},
		})
	}

	return regions, nil
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
func ExtractTextFromRegion(img image.Image, x1, y1, x2, y2 int, language string) (*OCRResult, error) {
	// Crop the region
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

	// Save to temporary file
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

// DetectTextRegions finds text regions in an image without performing full OCR.
//
// This function identifies where text is located in an image, useful for:
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
//   - error: Non-nil if tesseract is not installed or fails.
func DetectTextRegions(imagePath string, minConfidence float64) (*DetectTextRegionsResult, error) {
	tesseract, err := findTesseract()
	if err != nil {
		return nil, err
	}

	// Verify file exists
	if _, err := os.Stat(imagePath); err != nil {
		return nil, fmt.Errorf("image file not found: %w", err)
	}

	// Use TSV output to get bounding boxes
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(tesseract, imagePath, "stdout", "tsv")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tesseract failed: %v: %s", err, stderr.String())
	}

	regions := []TextRegionBox{}
	lines := strings.Split(stdout.String(), "\n")

	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 12 {
			continue
		}

		// Only include word-level entries (level 5) with text
		level, _ := strconv.Atoi(fields[0])
		if level != 5 {
			continue
		}

		text := strings.TrimSpace(fields[11])
		if text == "" {
			continue
		}

		conf, _ := strconv.ParseFloat(fields[10], 64)
		confidence := conf / 100.0

		if confidence < minConfidence {
			continue
		}

		left, _ := strconv.Atoi(fields[6])
		top, _ := strconv.Atoi(fields[7])
		width, _ := strconv.Atoi(fields[8])
		height, _ := strconv.Atoi(fields[9])

		regions = append(regions, TextRegionBox{
			Bounds: Bounds{
				X1: left,
				Y1: top,
				X2: left + width,
				Y2: top + height,
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

// TesseractVersion returns the installed Tesseract version, or an error if not installed.
func TesseractVersion() (string, error) {
	tesseract, err := findTesseract()
	if err != nil {
		return "", err
	}

	var stdout bytes.Buffer
	cmd := exec.Command(tesseract, "--version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout // Version info goes to stderr on some systems

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get tesseract version: %w", err)
	}

	// First line contains version
	lines := strings.Split(stdout.String(), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "unknown", nil
}

// OCRInfo contains information about the OCR subsystem.
type OCRInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
	Backend   string `json:"backend"`
}

// GetOCRInfo returns information about OCR availability.
func GetOCRInfo() OCRInfo {
	version, err := TesseractVersion()
	if err != nil {
		return OCRInfo{
			Available: false,
			Error:     err.Error(),
			Backend:   "tesseract CLI",
		}
	}

	return OCRInfo{
		Available: true,
		Version:   version,
		Backend:   "tesseract CLI",
	}
}

// MarshalJSON implements json.Marshaler for OCRInfo.
func (o OCRInfo) MarshalJSON() ([]byte, error) {
	type Alias OCRInfo
	return json.Marshal(Alias(o))
}
