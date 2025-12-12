# TODO: Static OCR Builds for All Platforms

## Overview

Currently, we have:
- **Linux**: Static tesseract linking with embedded tessdata (zero dependencies)
- **macOS/Windows**: CLI fallback (requires user to install tesseract)

This document outlines how to achieve zero-dependency OCR on all platforms using GitHub Actions.

## Goal

Single binary distribution with built-in OCR for all platforms - no external tesseract installation required.

## GitHub Actions Approach

Each platform runner builds tesseract and dependencies as static libraries, then links them into the Go binary.

### Linux (ubuntu-latest / ubuntu-24.04-arm)

**Status**: ✅ Implemented

```yaml
- name: Install static build dependencies
  run: |
    sudo apt-get update
    sudo apt-get install -y \
      libtesseract-dev \
      libleptonica-dev \
      libpng-dev \
      libjpeg-dev \
      libtiff-dev \
      libgif-dev \
      libwebp-dev
```

Build with:
```bash
CGO_ENABLED=1 CGO_LDFLAGS="-static -ltesseract -llept ..." go build
```

### macOS (macos-latest for arm64, macos-13 for x86_64)

**Status**: 🔲 Not implemented

```yaml
- name: Install tesseract via Homebrew
  run: |
    brew install tesseract leptonica pkg-config

- name: Build static libraries
  run: |
    # Homebrew doesn't provide static libs by default
    # Option 1: Build tesseract from source with --enable-static
    # Option 2: Use vcpkg for static builds
    brew install automake autoconf libtool
    git clone https://github.com/tesseract-ocr/tesseract.git
    cd tesseract
    ./autogen.sh
    ./configure --enable-static --disable-shared
    make -j$(sysctl -n hw.ncpu)
```

Challenges:
- Homebrew prefers dynamic libraries
- May need to build tesseract + leptonica from source
- Universal binary (arm64 + x86_64) requires two builds + lipo

### Windows (windows-latest)

**Status**: 🔲 Not implemented

Options:
1. **vcpkg** (recommended):
```yaml
- name: Install vcpkg and tesseract
  run: |
    git clone https://github.com/Microsoft/vcpkg.git
    ./vcpkg/bootstrap-vcpkg.bat
    ./vcpkg/vcpkg install tesseract:x64-windows-static
```

2. **Pre-built static libs** from tesseract releases

3. **MSYS2/MinGW**:
```yaml
- name: Setup MSYS2
  uses: msys2/setup-msys2@v2
  with:
    install: mingw-w64-x86_64-tesseract-ocr
```

Challenges:
- CGO on Windows typically uses MinGW
- Static linking with MSVC requires different approach
- Need to handle Windows path separators in tessdata

## Embedded Tessdata

All platforms need the training data files embedded:

```go
//go:embed tessdata/eng.traineddata
//go:embed tessdata/osd.traineddata
var tessdata embed.FS
```

Extract on first run to:
- Linux/macOS: `<binary_dir>/tessdata/`
- Windows: `<binary_dir>\tessdata\`

## Estimated Effort

| Platform | Effort | Notes |
|----------|--------|-------|
| Linux x86_64 | ✅ Done | Static linking works |
| Linux arm64 | 1 day | Need arm64 runner or cross-compile |
| macOS arm64 | 2-3 days | Build tesseract from source |
| macOS x86_64 | 1 day | Same as arm64, different arch |
| Windows | 3-4 days | vcpkg setup, CGO/MinGW complexity |

## Binary Size Estimates

| Platform | Current (CLI) | With Static OCR |
|----------|---------------|-----------------|
| Linux | ~2.7 MB | ~25-30 MB |
| macOS | ~2.7 MB | ~25-30 MB |
| Windows | ~2.8 MB | ~30-35 MB |

The increase is from:
- Tesseract library: ~5-8 MB
- Leptonica library: ~2-3 MB
- Image libraries: ~3-5 MB
- Tessdata (eng + osd): ~15 MB

## References

- [tesseract-ocr/tesseract](https://github.com/tesseract-ocr/tesseract)
- [gosseract CGO bindings](https://github.com/otiai10/gosseract)
- [vcpkg tesseract port](https://github.com/microsoft/vcpkg/tree/master/ports/tesseract)
- [GitHub Actions runners](https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners)

## Priority

Low - Current CLI fallback works for macOS/Windows users who can easily install tesseract via Homebrew or the Windows installer. Container deployments (primary use case) use Linux where static linking is already implemented.
