# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-12-12

### Added

- Initial release of Image Tools MCP Server
- **18 image analysis tools** for Claude via Model Context Protocol:
  - **Basic Info**: `image_load`, `image_dimensions`
  - **Region Operations**: `image_crop`, `image_crop_quadrant`
  - **Color Operations**: `image_sample_color`, `image_sample_colors_multi`, `image_dominant_colors`
  - **Measurement**: `image_measure_distance`, `image_grid_overlay`, `image_check_alignment`, `image_compare_regions`
  - **OCR**: `image_ocr_full`, `image_ocr_region`, `image_detect_text_regions`
  - **Shape Detection**: `image_detect_rectangles`, `image_detect_lines`, `image_detect_circles`, `image_edge_detect`

- **6-platform binary releases**:
  - Linux AMD64 (with embedded Tesseract OCR)
  - Linux ARM64
  - macOS AMD64 (Intel)
  - macOS ARM64 (Apple Silicon)
  - Windows AMD64
  - Windows ARM64

- **Container deployment package** (`container-tools-*.tar.gz`) for adding image analysis capabilities to existing Docker containers

- **macOS code signing and notarization** support (when signing is enabled)

- Tesseract OCR integration for text extraction
- Canny edge detection implementation
- Hough transform for line and circle detection
- Thread-safe image caching
- Comprehensive smoke tests on all platforms

### Technical Details

- Built with Go 1.22+
- MCP protocol version 2024-11-05
- JSON-RPC 2.0 over stdio

### Platform Notes

- **Linux AMD64**: Includes embedded Tesseract OCR - no additional setup required for full functionality
- **All other platforms**: Use CLI fallback for OCR - install Tesseract for OCR features (see [INSTALL.md](INSTALL.md))
- All platforms support full functionality for non-OCR tools without additional dependencies

[1.0.0]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.0.0
