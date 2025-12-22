# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.1] - 2025-12-22

### Removed

- **Removed deprecated hooks-dispatcher pattern** from container-tools package
  - This MCP is stateless and doesn't require hooks to function
  - Removed `etc/hooks-dispatcher.sh` and `etc/hooks.d/` from package
  - Simplified install.sh to only manage `mcp.json` (no settings.json hooks)
  - `mcp.json` now only contains `mcpServers` key (no hooks key)

### Added

- **Legacy cleanup function** in installer to migrate existing installations
  - Automatically removes old hooks-dispatcher infrastructure
  - Cleans up obsolete hooks key from mcp.json

### Changed

- **Simplified package structure** - now contains only essential files:
  - `image-tools-mcp/bin/` - launcher and platform binaries
  - `image-tools-mcp/install.sh` - simplified installer
  - Documentation files (README, LICENSE, CHANGELOG, VERSION_MANIFEST)

[1.2.1]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.1

## [1.2.0] - 2025-12-18

### Changed

- **Restructured container-tools package layout**:
  - `install.sh`, `LICENSE`, `CHANGELOG.md`, and `VERSION_MANIFEST.txt` now live inside the MCP folder (`image-tools-mcp/`) rather than at package root
  - Installation command changed from `./install.sh` to `./image-tools-mcp/install.sh`
  - Backup location changed from peer folder (`{mcp}-prior/`) to inside MCP folder (`{mcp}/backup/prior/`)
  - Removed `test-platforms.sh` from distribution package

- **Fixed mcp.json configuration** to include required `--mode stdio` argument for MCP server startup

### Updated

- Container Tools MCP Integration Guide updated to reflect new package structure
- All installation documentation updated with new paths and commands

[1.2.0]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.0

## [1.1.1] - 2025-12-16

### Changed

- **Renamed container-tools tarball** from `image-tools-mcp-v*.tar.gz` to `container-tools-image-tools-mcp-v*.tar.gz` for clear identification as the container-tools distribution package

[1.1.1]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.1.1

## [1.1.0] - 2025-12-16

### Changed

- **Container Tools packaging now follows the Integration Guide specification**
  - Directory structure changed from `/opt/container-tools/opt/{mcp}/` to `/opt/container-tools/{mcp}/`
  - Symlinks created in `/opt/container-tools/bin/` for easy PATH addition
  - Hooks system support with `hooks-dispatcher.sh` and `hooks.d/` directory
  - mcp.json now includes hooks configuration pointing to dispatcher

### Added

- **Enhanced install.sh** with full Container Tools Integration Guide compliance:
  - `--target DIR` parameter for custom installation locations
  - `--uninstall` flag with intelligent rollback (restores prior installation if available)
  - `--help` option
  - Skip-if-identical MD5 optimization (skips reinstall if binary unchanged)
  - Single-depth backups (changed to `backup/prior/` in v1.2.0)
  - mcp.json backup to MCP's territory before modifications
  - Colored output and post-install verification
- App-start hook for image-tools-mcp (placeholder for future initialization)
- Container Tools MCP Integration Guide documentation

### Removed

- Old MCP Coexistence and Container Packaging guides (replaced by unified Integration Guide)

[1.1.0]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.1.0

## [1.0.1] - 2025-12-13

### Changed

- **Linux ARM64 now includes embedded Tesseract OCR** - Both Linux platforms (AMD64 and ARM64) are now fully self-contained with no external dependencies required for OCR functionality
- GitHub Actions release workflow now uses native ARM64 runners (`ubuntu-24.04-arm`) for Linux ARM64 builds, enabling CGO and embedded OCR support
- Updated documentation to reflect that both Linux platforms have embedded OCR

### Technical Details

- Linux ARM64 binary now built with `CGO_ENABLED=1` on native ARM64 GitHub Actions runners (previously cross-compiled without CGO)
- Container-tools package now includes two fully self-contained Linux binaries

[1.0.1]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.0.1

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

### Test Coverage

- **Overall**: 87.1%
- `internal/imaging`: 97.1% - Image loading, cropping, color sampling, measurement
- `internal/detection`: 96.8% - Shape detection (rectangles, lines, circles, text regions)
- `internal/server`: 77.1% - MCP protocol handling and tool execution
- `internal/ocr`: 67.7% - Tesseract OCR integration

### Platform Notes

- **Linux AMD64**: Includes embedded Tesseract OCR - no additional setup required for full functionality
- **All other platforms**: Use CLI fallback for OCR - install Tesseract for OCR features (see [INSTALL.md](INSTALL.md))
- All platforms support full functionality for non-OCR tools without additional dependencies

[1.0.0]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.0.0
