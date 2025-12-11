# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Image Tools MCP server
- 19 MCP tools for image analysis:
  - **Basic Info**: `image_load`, `image_dimensions`
  - **Region Operations**: `image_crop`, `image_crop_quadrant`
  - **Color Operations**: `image_sample_color`, `image_sample_colors_multi`, `image_dominant_colors`
  - **Measurement**: `image_measure_distance`, `image_grid_overlay`, `image_check_alignment`, `image_compare_regions`
  - **OCR**: `image_ocr_full`, `image_ocr_region`, `image_detect_text_regions`
  - **Shape Detection**: `image_detect_rectangles`, `image_detect_lines`, `image_detect_circles`, `image_edge_detect`
- Docker support with multi-architecture images (amd64, arm64)
- Tesseract OCR integration for text extraction
- Canny edge detection implementation
- Hough transform for line and circle detection
- Thread-safe image caching
- VS Code Dev Container configuration
- GitHub Actions CI/CD pipeline for releases
- Comprehensive documentation and API reference

### Technical Details
- Built with Go 1.22+
- MCP protocol version 2024-11-05
- JSON-RPC 2.0 over stdio

## [0.1.0] - 2025-01-XX

### Added
- Initial public release

---

## Release Notes Format

Each release includes:
- **Added**: New features
- **Changed**: Changes to existing functionality
- **Deprecated**: Features to be removed in future versions
- **Removed**: Features removed in this release
- **Fixed**: Bug fixes
- **Security**: Security vulnerability fixes
