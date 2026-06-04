# STATIC-LINKING — Sprint Plan

**Goal:** Ship fully static, zero-dependency Linux binaries (amd64 + arm64) that
run identically on any base image, immune to shared-library / soname drift —
while *preserving and improving* image fidelity, the core purpose of this tool.

**Source proposal:** [`../PROPOSAL_ALPINE_STATIC_LINKING.md`](../PROPOSAL_ALPINE_STATIC_LINKING.md)
**Scope:** Linux build targets only. macOS/Windows already ship portable
`CGO_ENABLED=0` shell-out binaries and are explicitly out of scope.

---

## Design invariant — image fidelity (applies to every section)

This tool exists so an agent can understand what an image represents. Nothing
in this sprint may cost image resolution or quality. The binding rules:

- **Decode once, in Go, at full resolution.** All 21 tools reason about one
  identical pixel buffer (the cached `image.Image` from
  `internal/imaging/loader.go`).
- **Every inter-component hand-off uses a lossless format (PNG), never a lossy
  one (never JPEG).** PNG re-encoding is bit-exact — it discards nothing.
- **Never downsample, filter, or mutate the cached source.** Tools that need a
  derived image (edge detect, crop) work on copies.
- **To add input formats, add lossless decoders at the Go loader**
  (`x/image/tiff`, `x/image/webp`, …) so *every* tool benefits — never by
  re-introducing a lossy conversion or a fat codec library.

"Codec trimming" in this sprint means removing **file-format parsers** from
Leptonica, not quality filters. It is orthogonal to fidelity.

---

## 1. Unify the OCR decode path through the Go loader (fidelity + enables the trim)

**Why.** Today OCR hands Tesseract a *file path* —
`client.SetImage(imagePath)` at `internal/ocr/tesseract_cgo.go:162`
(`RecognizeText`) and `:268` (`DetectTextRegions`). Leptonica then opens and
decodes the file itself, so Leptonica's codec set dictates which formats OCR can
read — and OCR uses a *different* decode path than every other tool (the Go
loader at `internal/imaging/loader.go:6-8`, which registers GIF/JPEG/PNG). This
both blocks the codec trim and risks OCR "seeing" the image differently than the
measurement tools do.

**Current code.**
- `internal/ocr/tesseract_cgo.go:154-176` — `RecognizeText`: `NewClient` →
  `SetImage(path)` → `Text()` / `GetBoundingBoxes`.
- `internal/ocr/tesseract_cgo.go:261-272` — `DetectTextRegions`: same
  `SetImage(path)` pattern.
- `internal/imaging/loader.go` — `ImageCache` already decodes + caches
  `image.Image` at full resolution; imports `image/gif|jpeg|png`.

**Target behavior.** Replace `SetImage(path)` with: load the image via the
shared loader (cached, full-resolution `image.Image`), encode it to an in-memory
**PNG** buffer (`image/png`, already imported at `tesseract_cgo.go:14`), and feed
`client.SetImageFromBytes(pngBytes)`. Leptonica then only ever decodes PNG, so it
can be trimmed to libpng alone (§2). OCR gains every format the Go loader
supports and shares the exact pixels the other tools use.

**Integration points.** Both OCR functions; the loader's cache (OCR now benefits
from it instead of bypassing it). No public MCP tool signature changes.

**Verification — normal / edge / error.**
- *Normal:* OCR a PNG and a JPEG fixture → text matches today's output.
- *Edge:* paletted GIF, grayscale image, and a large (multi-MP) image → decode
  and OCR succeed at full resolution; assert output dimensions/word boxes are
  unchanged vs. the direct-path baseline (proves no resolution loss).
- *Error:* undecodable bytes → a clean error, not a panic.
- Add a regression test asserting PNG re-encode of a known fixture is pixel-
  identical to the source (lossless guarantee).

---

## 2. `Dockerfile.static` — Alpine builder, source-built static libs

**Why.** A genuinely static binary needs Leptonica + Tesseract built
`--enable-static` against musl; distro `-dev` packages don't reliably ship `.a`
archives, and the distro Leptonica is "fat" (drags in ~50 libraries).

**Target.** New multi-stage `Dockerfile.static`:
- **Builder** `FROM golang:1.25-alpine`: install `build-base git autoconf
  automake libtool pkgconf zlib-static libpng-static` (+ source-build any lib
  whose static `.a` Alpine omits).
- Source-build **Leptonica** `--disable-shared --enable-static
  --with-libpng --with-zlib --without-libjpeg --without-libtiff --without-giflib
  --without-libwebp --without-libopenjpeg`. (libpng-only is sufficient because
  §1 guarantees Leptonica only ever sees PNG.)
- Source-build **Tesseract** `--disable-shared --enable-static --disable-legacy`
  against the static Leptonica; drop training tools.
- Run `./scripts/ensure-tessdata.sh` before `go build` so `//go:embed` (at
  `internal/ocr/tesseract_cgo.go:22-24`) finds the data.
- Build: `CGO_ENABLED=1 go build -tags cgo -ldflags '-s -w -linkmode external
  -extldflags "-static -static-libstdc++ -static-libgcc"' ./cmd/image-mcp`.
- **Export stage** (`FROM scratch AS export-binaries`) carrying just the binary,
  for `buildx -o type=local`.

**Verification.** `ldd` → "not a dynamic executable"; `file` → "ELF … statically
linked"; `--version` exits 0; OCR smoke on `testdata/simple_diagram.png` returns
text with no system Tesseract present.

---

## 3. `Makefile` — add a native static build target

**Why.** Releases need a static build entry point; local dev does not change.

**Target.**
- Add `dist-static`: builds the **host architecture** natively via
  `Dockerfile.static` and extracts the binary to `dist/`. Cross-arch is the
  release workflow's job (native runners, §4) — the Makefile builds the other
  arch via `buildx --platform` only when explicitly requested, accepting qemu
  for that local-only case.
- **Unchanged:** `build`, `test`, `dist-linux` remain dynamic CGO for the fast
  local inner loop (we already run inside the dev container with libtesseract-dev
  installed). `make build`/`make test` behavior is identical to today.

**Verification.** `make dist-static` on this dev container yields a binary that
passes the §2 `ldd`/`file`/OCR checks.

---

## 4. `release.yml` — static Linux legs on native runners

**Why.** The released Linux binaries are the ones that broke on soname drift;
they must become static. Build natively per arch — least emulation (per
decision).

**Current code.** `.github/workflows/release.yml`:
- `build-linux-amd64` (`:13`, `runs-on: ubuntu-latest`) and `build-linux-arm64`
  (`:99`, `runs-on: ubuntu-24.04-arm`): each `apt-get install libtesseract-dev
  libleptonica-dev`, `ensure-tessdata.sh`, then a **dynamic**
  `CGO_ENABLED=1 … go build` (`:44`, `:130`), smoke tests, upload artifact.
- `assemble-package` (`:490`) consumes `binary-linux-{amd64,arm64}` by name;
  `create-release` (`:1231`, `softprops/action-gh-release`) publishes.

**Target.** In both Linux legs, replace the apt-install + raw `go build` with a
native `Dockerfile.static` build on the runner's own architecture, then extract
the binary under the **same artifact filename** so `assemble-package` and
`create-release` need no change. Keep all existing smoke tests (version,
tools/list, image ops, OCR) and **add static-assertion steps**: `ldd` →
not-dynamic and `file` → statically-linked, failing the build if not.

**Verification — normal / edge / error.**
- *Normal:* both legs produce static binaries; all existing smoke tests pass.
- *Edge:* OCR smoke proves embedded tessdata works with no system Tesseract on
  the runner.
- *Error:* a dynamic build (regression) fails the new `ldd` assertion → red.

---

## 5. `ci.yml` — catch static regressions on PRs

**Why.** Static breakage should fail a PR, not first appear at release.

**Current code.** `.github/workflows/ci.yml`: `lint` (golangci-lint, Go 1.25),
`test` (`go test -race`), `build` (plain `go build`) — all dynamic.

**Target.** Add a `static-build` job (host arch) that builds `Dockerfile.static`,
asserts `ldd` static + runs the OCR smoke. Keep lint/test/build as the fast
dynamic gates. (Per the baseline-health overlay, golangci-lint + gofmt remain the
warnings gate.)

**Verification.** PR with a deliberately dynamic tweak → `static-build` fails.

---

## 6. Production `Dockerfile` — rebuild on the static binary (dogfood)

**Why.** We test in the container we ship. If the image is built differently from
the shipped binary, we won't catch the breakage we're eliminating. Build it the
same static way; the runtime then needs no Tesseract/Leptonica at all.

**Current code.** `Dockerfile`: builder does a **dynamic** CGO build (`:16-18`);
runtime `alpine:3.20` `apk add`s `tesseract-ocr tesseract-ocr-data-eng
ca-certificates nodejs npm` (`:24-29`) to satisfy it.

**Target.** Reuse the §2 static builder; runtime stage becomes minimal — copy the
single static binary, keep only genuinely-needed runtime bits (`ca-certificates`;
**audit whether `nodejs`/`npm` are still required** and drop if not). Remove
`tesseract-ocr`, `tesseract-ocr-data-eng`, and the Leptonica runtime dep —
they're now embedded/static.

**Verification.** `docker run` the image → `--version`, tools/list, and OCR all
work; image is smaller; `ldd` inside the container confirms the binary is static.

---

## 7. Cross-base acceptance gate (proposal §8.5)

**Why.** The whole point: one binary, any base. Prove it.

**Target.** A verification step (in CI and/or a documented manual exercise) that
drops the **same** static binary into at least three bases — Debian **bookworm**,
Debian **trixie**, and bare **Alpine** — and runs `ldd` (static), `--version`,
tools/list, and the OCR smoke in each. All must pass identically.

**Verification.** Identical pass across all three bases = base/soname
independence proven. This is the sprint's definition of done.

---

## 8. Documentation deliverables

**Why.** Docs are part of the work; several currently claim or imply the old
dynamic reality.

**Target.**
- `DOCs/TODO_STATIC_OCR_ALL_PLATFORMS.md` — correct the Linux status (currently
  "✅ Implemented" against dynamic binaries) to point at this implementation.
- `DOCs/PROPOSAL_ALPINE_STATIC_LINKING.md` — stamp **Status: Implemented**; note
  the §1 re-encode refinement (the proposal assumed the trim was free; it wasn't
  until OCR decoded through the Go loader).
- `CHANGELOG.md` — Keep-a-Changelog entry under the release version; note the OCR
  decode-path change and that Linux binaries are now fully static (developer/
  integrator audience, per the build-wrapup overlay).
- `README.md` / `INSTALL.md` — audit and update any statement that Linux needs a
  runtime Tesseract install.
- `DOCs/IMAGE_MCP_BLUEPRINT.md` (`SPEC_DOC`) — add a short build/linking note if
  it covers deployment/runtime requirements.

---

## Explicitly out of scope (decided, not open)

- **macOS/Windows static OCR** — they already ship portable; unchanged.
- **TIFF/WebP/other input formats** — *not* added this sprint. When wanted, the
  fidelity invariant says to add lossless Go decoders at the loader
  (`x/image/tiff`, `x/image/webp`), benefiting all tools — a separate, small
  enhancement, not a Leptonica codec.
- **External container-fleet symlink stopgap** (`liblept.so.5`) — not present in
  this repo; it was an external fleet patch. Nothing to retire here.

---

## Exit gate

Research complete; no open questions. Decisions locked: OCR re-encode to PNG ·
native per-arch builds (least emulation) · Docker image built static (dogfood) ·
local dynamic build unchanged. Ready for `sprint-start`.

---

## Sprint-start record — 2026-06-04

**Build number:** 1.3.0 (minor bump from 1.2.11; committed in `VERSION`).

**Working-tree audit (Step 2):** clean — no uncommitted edits, no untracked
files in the sprint blast radius. Foundation committed; 7 commits ahead of
`origin/main` at start.

**tracking-readiness entry check (Step 3):** READY. todo-mcp tasks: 0 (empty
board); context: 0 keys; `MEMORY.md`: 1 line. Nothing to archive or prune.

**Entry baseline (Step 4 — baseline-health):** clean and green, no decisions
required.
- Build (`go build ./...` + `go vet ./...`): clean, 0 warnings.
- gofmt: clean. golangci-lint (`make lint`): clean, 0 findings.
- Tests (`make test`): all pass, **0 skips**; runner (`go test ./...`) covers
  all 15 test files across 4 packages.
- **Failure groups: none.** Exit baseline at closeout must hold this — clean
  build, 0 warnings, 0 new failures, 0 new skips.

---

## Section ↔ task cross-reference

Sprint tag: `static-linking` · build 1.3.0 · total estimate ~15.5h.

| Plan § | Deliverable | Task | seq | est |
| ------ | ----------- | ---- | --- | --- |
| §1 | OCR decode-path unification (re-encode to PNG) | «#1» | 1 | 2h |
| §2 | `Dockerfile.static` — static Leptonica + Tesseract | «#2» | 2 | 5h |
| §3 | Makefile `dist-static` target | «#3» | 3 | 1h |
| §4 | `release.yml` — static Linux legs, native runners | «#4» | 4 | 2h |
| §5 | `ci.yml` — static-build regression job | «#5» | 5 | 1h |
| §6 | Production `Dockerfile` rebuilt static (dogfood) | «#6» | 6 | 1.5h |
| §7 | Cross-base acceptance gate (bookworm/trixie/alpine) | «#7» | 7 | 1.5h |
| §8 | Documentation updates | «#8» | 8 | 1.5h |

Dependency spine: §1 → §2 → {§3, §4, §5, §6} → §7 → §8. §1 establishes the
lossless decode standard the §2 codec trim relies on; §2 is the build substrate
for §3–§7; §8 documents final behavior last.
