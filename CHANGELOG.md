# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.11] - 2026-05-03

### Fixed

- **Semi-transparent halos around interior boundaries.** Edge-blend pixels (those with partial alpha in the draft, marking a soft icon→bg blend) are correctly preserved at the icon's *outer* edge so it has a smooth anti-aliased outline. But edge-blend pixels at *interior* boundaries — the rim of an enclosed white region, or the line between two icon colors — were also being preserved, producing a translucent ring around every interior color transition.
- **Fix:** in the rewrite step, edge-blend pixels are now classified by whether they have any 4-connected outer-bg neighbor:
  - **Has outer-bg neighbor** → keep partial alpha (legitimate outer edge, soft anti-aliased blend looks correct).
  - **No outer-bg neighbor** → force fully opaque with source RGB. Interior boundary; no actual background here, so no fade needed.

This completes the principle: the only transparency in the output is at the icon's true outer perimeter. All interior pixels (including color transitions inside the icon) are fully opaque with their source RGB.

[1.2.11]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.11

## [1.2.10] - 2026-05-02

### Fixed

- **Whitish outline around the icon (regression from v1.2.9).** The new "outer-background-only" rewrite step was forcing every non-outer-bg pixel to α=255 with source RGB. For edge-blend pixels — which had partial alpha in the draft because they're a soft anti-aliased blend of the icon color and the underlying checker — the source RGB at those positions is the *blended* near-white color. Stomping the partial alpha to 255 made those near-white blends opaque, producing a thin whitish halo around the icon's silhouette.
- **Fix:** the rewrite step now snapshots the draft alpha channel before any mutation. Three cases per pixel: (a) outer-bg → fully transparent; (b) was-bg-but-enclosed → restore source RGB at α=255 (the recovery case); (c) was-fg in the draft → leave the draft pixel unchanged so its anti-aliased edge alpha is preserved.

[1.2.10]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.10

## [1.2.9] - 2026-05-02

### Fixed

- **Interior pixel transforms eliminated.** The unbake pipeline was modifying pixels inside the icon (creating "border" rings around enclosed white regions, dropping near-checker-colored pixels along edges). The new rule is principled: only the *outer* background — the connected region of bg-classified pixels reachable from the image border — becomes transparent. Every other pixel keeps its source RGB at α=255. No transforms happen inside the icon.
- **Per-edge gap closure for white-extending-to-image-border.** When the icon contains a near-checker-colored region (e.g. a white shirt under a rider's collar) that extends to an image border, the previous flood-fill would "escape" through the gap and turn the entire region transparent. The new pipeline runs a 1D convex-hull pass on each of the four image borders before the flood-fill: between the outermost foreground pixels on each border, all pixels are marked foreground so flood-fill cannot escape. Closes the case where the icon's outermost gold pixels bracket a white gap at the image edge.

### Removed

- **Parity-pair cell recovery, anti-fringe ring, and 1-pixel hole-fill passes** are gone — all subsumed by the new "outer background only" rule. The `recover_color_matched_icon`, `fill_enclosed_background`, and `anti_fringe_radius` options are accepted for backward compatibility but no longer affect output (the pipeline always handles enclosed and bracketed regions correctly).

### Internal

- Removed unused `medianInt` helper and an ineffectual assignment in `clusterCornerColors` flagged by `golangci-lint`. Lint passes clean.

[1.2.9]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.9

## [1.2.8] - 2026-05-02

### Fixed

- **Perimeter artifacts in unbaked output (regression from v1.2.7).** The parity-pair cell recovery pass was being applied by default and produced visible blocky halos around icons with irregular boundaries: any background cell just outside the icon adjacent to ≥2 fg opposite-parity cells got flipped to foreground, leaving cell-period-sized gray blocks at the perimeter. The pass is now opt-in.

### Changed

- **`recover_color_matched_icon` default flipped to `false`.** This single flag now controls a coupled pair of behaviors: predicted-color background matching AND the parity-pair cell recovery. They have to move together — predicted-color matching alone produces a checkerboard pattern of opaque/transparent pixels in white-icon areas (since light-cell-position pixels still match the predicted light color and become bg). Default behavior now matches v1.2.6 exactly: permissive "either color" matching for background, no parity-pair pass. Users with icons that genuinely contain pure-white (or other near-checker-colored) regions can opt in by passing `recover_color_matched_icon: true`.
- **`bg_tolerance` default reverted to 28** (was 24 in v1.2.7). Aligns with v1.2.4-v1.2.6. When `recover_color_matched_icon` is on, a tighter internal tolerance is used (`bg_tolerance - 6`) to make predicted-color matching discriminate pure-white from dark-cell predictions.

### Kept (safe v1.2.7 improvements)

- **`fill_enclosed_background`** (case 2 fix) remains on by default — it's a safe fix that uses border flood-fill to recover background-classified pixels enclosed inside the icon. No artifacts on irregular boundaries.

[1.2.8]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.8

## [1.2.7] - 2026-05-02

### Fixed

- **White (and other near-checker-colored) icon regions were being dropped.** Previously, foreground pixels whose color happened to match either checker color were classified as background. This affected:
  - **Case 1: white icon regions extending to the image border.** A pure-white area in the icon would have its light-cell-position pixels match `~#FEFEFE` (the light checker color) and become transparent.
  - **Case 2: white icon regions fully enclosed by other foreground.** Same cause — pixels in the eye of a face logo, the white in a horse's blaze, etc., all turned transparent.

### Changed

- **`isBackgroundLike` now uses predicted-color matching** instead of "either checker color." A pixel is only classified as background if it matches the *specific* checker color expected at its `(x, y)` cell location. This means pure-white pixels in dark-cell positions (which are far from the dark checker color) are correctly classified as foreground; only the light-cell-position pixels of a white region remain ambiguous. The two new recovery passes catch those.
- **Default `bg_tolerance` reduced from 28 to 24.** Tighter enforcement of the predicted-color match so pure-white pixels in dark-cell positions definitely don't slip through.

### Added

- **Parity-pair cell-recovery pass** (Case 1 fix). After per-pixel classification, runs a cell-level check: for each cell that's mostly background, looks at its 4 opposite-parity cell neighbors; if ≥2 of them contain foreground, this cell is sitting in an icon region overriding the checker pattern. All its pixels are flipped to foreground, with source RGB restored. The threshold of 2 (not 1) prevents over-growing the icon at its actual boundary, where typically only one parity-neighbor cell is foreground.
- **Border flood-fill pass** (Case 2 fix). Background pixels not connected (4-connectivity) to any image border are necessarily holes inside foreground regions — they're flipped to foreground with source RGB restored.
- **`recover_color_matched_icon`** parameter (default `true`). Disables the parity-pair pass.
- **`fill_enclosed_background`** parameter (default `true`). Disables the border flood-fill pass.
- New `pixel_stats` fields: `cells_recovered` and `enclosed_background_filled` count how many pixels each pass restored.

### Why this matters

The previous version would silently lose icon detail in any image with white or other near-checker-colored regions — extremely common (white teeth in a face logo, white shirt on a person, the white of an eye, blank space in a logo design that's intentionally white). The two recovery passes restore those regions without requiring the user to manually identify them. Validated on the existing fixture: `cells_recovered: 14653` (the rider's white helmet/clothing now correctly preserved); `enclosed_background_filled: 1320` (smaller enclosed regions); palette unchanged at `#E8B868 / #A87828`.

[1.2.7]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.7

## [1.2.6] - 2026-05-02

### Changed

- **`image_unbake_transparency` now preserves source colors by default.** Previously every foreground pixel was snapped to the nearest detected palette entry — a destructive quantization that threw away color information downstream tools (especially `image_vectorize`) need to make good decisions. The default behavior is now: only the alpha channel is changed; pure-fg and ambiguous-fg pixels keep their original RGB values. The unbake step is responsible for "lift the icon off the checker," not for "cleaned palette" — palette decisions now happen in `image_vectorize`.
- **Ambiguous pixels default to foreground rather than transparent.** JPEG-noise pixels that don't fit any clean category are now treated as opaque foreground (with source color preserved) instead of transparent. Eliminates the "pinhole" speckle that previously appeared inside icon bodies and required an additional 3×3 hole-fill pass.

### Added

- **`preserve_source_colors`** parameter (default `true`). Set to `false` to restore the previous "snap to palette" behavior for use cases where a cleaned/quantized output is preferred. Edge-blend pixels are always set to the canonical palette color regardless of this flag (the alpha-recovery formula requires it).
- **`ambiguous_to_foreground`** parameter (default `true`). Set to `false` to restore the previous conservative behavior (ambiguous pixels become transparent unless much closer to fg than bg).

### Why this matters

The user reported that the v1.2.5 unbake was hurting their downstream vectorization quality because color quantization at the unbake step left the trace with less information to work with. The fix is correct separation of concerns: unbake produces "the image as it would have been before flattening" (full source data, transparent background); vectorize produces "the image you want to trace" (palette decisions, noise filtering). Validated against `testdata/fake-transparent-image.png`: the unbaked PNG now visually matches the source minus the checker (full color richness preserved), and `image_vectorize` with `max_colors=2 quantize=24 turd_size=80` produces a 56KB SVG with the exact source palette.

[1.2.6]: https://github.com/ironsheep/image_tools_mcp/releases/tag/v1.2.6

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
