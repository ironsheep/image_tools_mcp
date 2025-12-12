// Package ocr provides Optical Character Recognition (OCR) functionality using Tesseract.
//
// This package wraps the Tesseract OCR engine (via gosseract/v2) to extract text
// from images. It supports full-image OCR, region-based OCR, and text region detection.
//
// # Prerequisites
//
// Tesseract must be installed on the system:
//   - Ubuntu/Debian: apt-get install tesseract-ocr
//   - macOS: brew install tesseract
//   - Windows: Download from https://github.com/UB-Mannheim/tesseract/wiki
//
// Language data files are required for each language:
//   - Ubuntu/Debian: apt-get install tesseract-ocr-eng (for English)
//   - Other languages: tesseract-ocr-<lang> packages
//
// # Supported Languages
//
// The default language is English ("eng"). Other languages can be specified
// using their Tesseract language codes:
//   - "eng" - English
//   - "deu" - German
//   - "fra" - French
//   - "spa" - Spanish
//   - "chi_sim" - Chinese (Simplified)
//   - See Tesseract documentation for full list
//
// # Functions
//
// The package provides three main functions:
//
//   - ExtractText: Full-image OCR, returns all text with word bounding boxes
//   - ExtractTextFromRegion: OCR on a specific rectangular region
//   - DetectTextRegions: Find text regions without performing full OCR
//
// # Performance Considerations
//
// OCR is computationally expensive. For large images or many regions:
//   - Crop to regions of interest first when possible
//   - Use DetectTextRegions to find text areas before full OCR
//   - Consider image preprocessing (contrast, scaling) for better results
//
// # Temporary Files
//
// ExtractTextFromRegion creates a temporary PNG file for Tesseract processing.
// This file is automatically deleted after OCR completes. Ensure the system's
// temporary directory has sufficient space for image files.
//
// # Error Handling
//
// Functions return errors for:
//   - Missing or invalid image files
//   - Unsupported language codes
//   - Tesseract initialization failures
//   - Temporary file I/O errors
//
// If bounding box extraction fails (e.g., Tesseract version mismatch),
// ExtractText still returns the extracted text with an empty Regions slice.
package ocr
