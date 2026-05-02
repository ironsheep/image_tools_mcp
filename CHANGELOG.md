# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.5] - 2026-05-02

### Added

- **`image_unbake_transparency` tool** — reconstructs a properly-transparent PNG from an image whose original transparency was flattened onto a "fake transparency" checkerboard background (e.g., screenshots of web preview panes, exported icons saved against a checker, etc.). Pipeline:
  1. **Auto-detect the checker** via a square-wave matched filter on luminance strips. Robust to JPEG block-correlation noise that breaks naive autocorrelation. Reports period, both cell colors, origin, and confidence.
  2. **Identify foreground icon colors** by histogramming pixels far from the checker palette, then single-link clustering (RGB distance < 36) to merge JPEG-bridge-noise variants of the same color.
  3. **Per-pixel classification** into pure-background, pure-foreground, edge-blend, or ambiguous — using a tolerant background test that absorbs JPEG halo (pixels near checker colors but not exact).
  4. **Recover lost alpha** for edge-blend pixels by inverting the alpha-compositing equation: `α = ‖p − bg(x,y)‖ / ‖fg − bg(x,y)‖`. Snaps to fully opaque/transparent at the extremes for crisp tracing.
  5. **Edge enhancements**: 3×3 majority filter to close JPEG-noise pinholes inside the icon body; anti-fringe extension to absorb leftover halo.
  6. Writes the reconstructed PNG to `output_path` (defaults to `<input>_unbaked.png` next to the source) **and** returns a base64 preview for inspection.
- All checker parameters and foreground colors can be supplied manually for cases where auto-detection fails or when the user wants explicit control.
- Output PNG flows directly into `image_vectorize` — no special handling needed downstream.

### Why this matters

PNG icons captured from web previews or LLM outputs frequently come with a baked-in checker pattern instead of real transparency. Without unbaking: `image_vectorize` traces the checker squares as foreground, the palette is dominated by the checker grays, and edge anti-aliasing pixels (which now blend gold-with-checker rather than gold-with-transparent) produce scalloped edges and inflated SVG sizes. Unbaking lifts the icon back onto a true transparent background; the trace pipeline then works as if the image had been exported correctly in the first place.

[1.2.5]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.5

## [1.2.4] - 2026-04-28

### Fixed

- **Anti-aliased fringe leaked into palette analysis** in `image_count_colors` and `image_vectorize`. Previously only fully-transparent pixels (alpha == 0) were excluded, so PNGs whose anti-aliased edges store near-white RGB under low alpha — including most ChatGPT-generated logos — had those fringe pixels dominate the color palette. Symptoms: a 2-color gold-on-transparent logo would report 4+ "colors" and trace into a 600+ KB SVG with a whitish "frosting" halo, or with `max_colors=2` the actual logo color would be dropped entirely in favor of the fringe.

### Added

- **`alpha_threshold` parameter** on both `image_count_colors` and `image_vectorize` (default 128). Pixels with alpha below this are treated as transparent background and excluded from palette analysis, color counting, and tracing. The default sits at the perceptual midpoint and reliably suppresses anti-aliased fringe on typical PNG icons. Pass `1` to restore legacy behavior (only alpha==0 excluded); pass `255` to require full opacity. Effective value is reported in the result as `alpha_threshold`.

[1.2.4]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.4

## [1.2.3] - 2026-04-27

### Added

- **`image_count_colors` tool** - reports the number of discrete opaque colors in an image, capped at 10. Fully transparent pixels are ignored so a transparent background does not count. Returns per-color hex/RGB/percentage and a `truncated` flag if the image had more than the cap.
- **`image_vectorize` tool** - converts a low-color raster icon (PNG with optional transparent background) to SVG by tracing each color layer separately. Pure-Go implementation via `github.com/dennwc/gotrace` — no `potrace` system dependency required. Output preserves transparency and returns both SVG text and a base64-encoded form. Tunable via `max_colors` (1-10), `quantize`, `turd_size`, and `alpha_max`.
- **Auto-quantize selection** - when `quantize` is omitted from either tool, a fast pre-pass counts exact opaque colors. If the image already has ≤ 10 distinct colors (clean solid-color art), `quantize=1` is used so the SVG palette matches the source PNG exactly; otherwise it falls back to `quantize=8` to absorb anti-aliasing/noise. Effective value is reported as `quantize` and `quantize_auto: true` in the result.
- **Auto-detected `max_colors` for vectorization** - omitting `max_colors` (or passing 0) routes the image through `image_count_colors` and uses that value, capped at 10.

### Changed

- Tool count grew from 18 to 20; `CLAUDE.md` updated to list the new "Color Counting & Vectorization" category.

[1.2.3]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.3

## [1.2.2] - 2026-04-27

### Fixed

- **MCP protocol compliance issues** identified by stdio audit against recent Claude Code versions
  - **Notification leak**: `handleRequest` previously returned a `-32601` "method not found" response for any unknown method, including JSON-RPC notifications (requests without an `id`). Strict clients reject stray `{"id":null}` responses and drop the connection. The handler now filters notifications (`id == nil`) before dispatching, per JSON-RPC 2.0
  - **protocolVersion negotiation**: `initialize` previously returned a hardcoded `"2024-11-05"` and ignored the client's requested version. Server now negotiates from a supported set (`2024-11-05`, `2025-03-26`, `2025-06-18`), defaulting to `2025-06-18` and echoing the client's version only when it is in the supported set

### Changed

- **`serverInfo.version` in the `initialize` response** now reflects the actual build version (via the `main.Version` ldflag) instead of a hardcoded `"0.1.0"`

[1.2.2]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.2

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

- Built with Go 1.23+
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
