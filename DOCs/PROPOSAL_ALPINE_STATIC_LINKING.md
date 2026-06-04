# Proposal: Alpine/musl Static Linking with Trimmed Codecs

**Status:** ✅ **Implemented** in build 1.3.0 (2026-06-04). Approach A was executed as specified; see the implementation note below for the one refinement the plan did not anticipate.
**Author:** drafted with Claude (session 2026-06-03)
**Scope:** Linux build targets only (the Docker fleet). macOS/Windows are explicitly out of scope — see [Non-goals](#non-goals).
**Supersedes (Linux portion of):** [`TODO_STATIC_OCR_ALL_PLATFORMS.md`](./TODO_STATIC_OCR_ALL_PLATFORMS.md)

> ### Implementation note (build 1.3.0)
>
> Delivered as proposed (Approach A — Alpine/musl, source-built static Leptonica + Tesseract, embedded tessdata preserved), with **one refinement the proposal did not foresee.** The proposal assumed the codec trim was essentially free — drop to `--with-libpng --with-libjpeg` and move on (§3, §7.1). In practice the trim was **not** safe until the OCR decode path was changed first: both OCR entry points previously passed Tesseract a *file path*, so Leptonica opened and decoded the file itself and OCR's format support was coupled to Leptonica's codec set. Trimming codecs there would have silently dropped JPEG/GIF/etc. OCR input.
>
> The fix (sprint §1) routes OCR through the shared Go image loader and hands Tesseract in-memory **lossless PNG** bytes via `SetImageFromBytes`. Now Leptonica only ever decodes PNG, so the static build trims all the way to **libpng-only** (even tighter than the proposal's libpng+libjpeg) with no loss of OCR input formats — the Go loader supplies them. This is why the realized codec surface is ~5 libraries, not the ~6 estimated here.
>
> Realized deliverables: `Dockerfile.static` (`export-binaries` + `runtime` stages), `make dist-static`, static release legs (native per-arch, no qemu), CI `static-build` + `cross-base` gates (bookworm/trixie/alpine), and the production image rebuilt on the static binary. The symlink stopgap (§1.1) is retired.

---

## 1. Problem statement

`image-tools-mcp` links Tesseract + Leptonica via CGO (`//go:build cgo && linux` in
`internal/ocr/tesseract_cgo.go`). The shipped release binaries are **dynamically linked**,
which couples each binary to the exact shared-library set of whatever container it lands in.

### 1.1 The failure we actually hit

A `v1.2.3-linux-arm64` binary built against Debian bookworm (Leptonica 1.82, soname
`liblept.so.5`) was run in a container carrying Leptonica 1.84 (soname `libleptonica.so.6`).
The loader could not resolve `liblept.so.5` and aborted. The symbols were all present — only
the **file name** (soname) differed. The emergency fix was a compatibility symlink
(`ln -sf libleptonica.so.6.0.0 .../liblept.so.5 && ldconfig`), which is a stopgap, not a fix,
and which **vanishes on container rebuild**.

### 1.2 It is not a Leptonica problem — it is a ~50-library drift surface

`ldd` on the current binary resolves **far more than Leptonica**:

```
liblept.so.5  libtesseract.so.5  libstdc++.so.6  libgcc_s.so.1  libc.so.6
libpng16  libjpeg.so.62  libgif.so.7  libtiff.so.6  libwebp  libwebpmux  libopenjp2
libz  libzstd  liblzma  libLerc  libjbig  libdeflate
libxml2  libarchive  libicuuc.so.72  libicudata.so.72
libcurl  libssl.so.3  libcrypto.so.3  libgssapi_krb5  libkrb5  libk5crypto  libkrb5support
libldap-2.5  liblber-2.5  libsasl2  libgnutls  libnettle  libhogweed  libgmp  libtasn1
libnghttp2  libidn2  librtmp  libssh2  libpsl  libbrotlidec  libbrotlicommon
libp11-kit  libkeyutils  libresolv  libffi  libunistring  libm
```

**Every one of those is a soname that can drift across base images.** Leptonica was simply the
first to break. Standardizing the fleet on a common base only hides this until the *next* base
bump revs ICU, OpenSSL, libtiff, or any of the other ~50.

### 1.3 The existing TODO doc overstates Linux status

[`TODO_STATIC_OCR_ALL_PLATFORMS.md`](./TODO_STATIC_OCR_ALL_PLATFORMS.md) marks Linux static
linking as **"✅ Implemented."** The shipped `v1.2.3` binaries contradict that — they are
dynamic. Whatever static path that doc describes is either aspirational, regressed, or applied
on a build path the releases don't use. This proposal treats Linux static linking as **not yet
truly achieved** and specifies how to get there durably.

---

## 2. Goal

One **fully static, self-contained binary per Linux architecture** that:

- reports `ldd ./image-tools-mcp` → **"not a dynamic executable"**
- runs **identically** on bookworm, trixie, Alpine, or any future fleet base, with **zero**
  external shared-library or soname dependencies
- needs **no** runtime `tesseract` install and **no** `TESSDATA_PREFIX` (language data is
  already embedded — see §5)

Net effect: the binary stops caring what base image it runs on. Future base upgrades can never
reintroduce the soname/ABI break.

---

## 3. Two approaches compared

The user framing: **Alpine static vs. "the other one."** "The other one" is the
Ubuntu/apt + glibc `-static` approach sketched in the existing TODO doc.

### Approach A — Alpine / musl static, trimmed codecs **(recommended)**

Build inside Alpine (musl libc), statically linking Tesseract + Leptonica + **only the image
codecs we actually use**, plus static libstdc++/libgcc.

```
-extldflags "-static -static-libstdc++ -static-libgcc"
CGO_ENABLED=1
```

**Why musl:** musl is designed to static-link cleanly. glibc actively discourages full static
linking (NSS, `getaddrinfo`, locale machinery) and emits warnings or fails for code that
touches DNS/users. Our tool does image decode + OCR — no networking, no NSS — so musl-static is
both clean and a natural fit.

**Why trim codecs:** the ~50-library tree above is inherited because distro Leptonica is built
"fat" (curl/ssl/krb5/ldap/gnutls/ICU are dragged in by optional Leptonica features we never
call). The tool only needs to **decode PNG/JPEG and run Tesseract.** Building Leptonica +
Tesseract from source with `--disable-shared --enable-static` and only `--with-libpng
--with-libjpeg` (plus zlib) collapses the static surface from ~50 libraries to ~6, producing a
binary that is both smaller and dramatically easier to link.

| Pros | Cons |
|------|------|
| Truly zero shared-lib deps | Must build Leptonica + Tesseract from source (`--enable-static`); Alpine `-dev` packages don't reliably ship `.a` archives |
| musl static is the well-trodden path | musl ≠ glibc for some edge numeric/locale behavior (irrelevant to our codepaths, but worth a smoke test) |
| Trimmed codec set → smaller binary, fewer CVEs to track | One-time build-script authoring effort |
| Alpine builder already exists in `Dockerfile` | |

### Approach B — Ubuntu/apt + glibc `-static` (the existing TODO approach)

Install `libtesseract-dev`/`libleptonica-dev`/codecs via apt, link `CGO_LDFLAGS="-static
-ltesseract -llept ..."` on glibc.

| Pros | Cons |
|------|------|
| No source builds — apt provides the dev packages | glibc full-static is fragile/discouraged; partial-static often still leaves dynamic refs |
| Matches what the TODO doc already sketched | Links the **fat** distro Leptonica → pulls the entire ~50-lib tree statically (huge binary, many CVEs) |
| | apt `.a` static archives are inconsistently provided across releases |
| | This is effectively what the "✅ Implemented" claim rests on — and the shipped binaries are still dynamic, so it has not actually delivered |

### Verdict

**Approach A (Alpine/musl + trimmed codecs).** It is the only one that reliably yields a
genuine zero-dependency binary, it shrinks the attack/maintenance surface instead of statically
baking 50 libraries, and it builds on the Alpine builder stage we already have in `Dockerfile`.

---

## 4. One binary per architecture

Static linking removes **distro/libc/soname** coupling. It does **not** remove **CPU
architecture** — we still ship one binary per arch:

- `linux/amd64`
- `linux/arm64`

(Additional Linux arches — arm/v7, ppc64le, riscv64 — follow the identical recipe if ever
needed, provided musl + static codec libs exist for that arch.)

### 4.1 Build mechanism: buildx multi-arch, not Go cross-compile

The current `Makefile` `dist` target uses Go cross-compilation (`GOARCH=arm64 go build`). That
works for the `CGO_ENABLED=0` mac/Windows targets, but **you cannot cross-compile a static CGO
arm64 binary from an amd64 host** without a full cross C toolchain *and* cross-built static
codec libs — a maintenance trap.

The clean path is to build **each arch natively inside its own Alpine container**:

```
docker buildx build --platform linux/amd64,linux/arm64 \
  --target export-binaries \
  -f Dockerfile.static -o type=local,dest=dist/ .
```

Each platform leg runs in a native (or qemu-emulated) Alpine where that arch's static `.a`
libraries are present. In CI, prefer native runners (e.g. `ubuntu-24.04-arm` for arm64) over
qemu for build speed.

---

## 5. Tessdata is already solved — do not regress it

`internal/ocr/tesseract_cgo.go` already embeds the language data and self-extracts at startup:

```go
//go:embed tessdata/eng.traineddata
//go:embed tessdata/osd.traineddata
var embeddedTessdata embed.FS
```

So a static binary is genuinely **one self-contained file** — no `TESSDATA_PREFIX`, no sidecar
data directory. Preserve this `//go:embed` path in the static build; it is the reason "static"
here equals "drop one file anywhere and it runs."

---

## 6. Non-goals

- **macOS (darwin/amd64, darwin/arm64):** Apple does not support static libc — a fully static
  Mach-O is impossible. These targets already build `CGO_ENABLED=0` and **shell out to a
  `tesseract` CLI** (`//go:build !cgo || !linux`), so they link no Tesseract/Leptonica and have
  **no soname-drift problem**. Their only requirement is `tesseract` on `PATH`
  (`brew install tesseract`) — a documentation concern, not a linking one.
- **Windows (windows/amd64, windows/arm64):** Same shape — pure-Go `.exe` that shells out to
  `tesseract.exe`. Self-contained for its own code; OCR needs an installed Tesseract.

These targets are **not a gap**. They are built in a fundamentally portable shape and need no
static work. If zero-dependency OCR on mac/Windows is ever wanted, that is the separate, larger
effort already catalogued in `TODO_STATIC_OCR_ALL_PLATFORMS.md` (2–4 days/platform) and is
**not** part of this proposal.

---

## 7. Implementation plan

> **Sequencing:** execute **after** the container base-image upgrade. The base upgrade is for
> fleet hygiene/standardization; static linking is what actually frees the tool from the base.
> Because a static binary is decoupled from the base, the two are independent — but doing the
> upgrade first means we build/verify the static binary against the new common base in one pass.

1. **Author `Dockerfile.static`** — Alpine builder stage:
   - Install build toolchain (`build-base`, `git`, `autoconf`, `automake`, `libtool`,
     `pkgconf`, `zlib-static`, `libpng-static` or source, `libjpeg-turbo-static` or source).
   - Build **Leptonica** from source: `./configure --disable-shared --enable-static
     --without-libtiff --without-libwebp --without-libopenjp2 --without-libgif`
     (keep only `--with-libpng --with-libjpeg`, plus zlib).
   - Build **Tesseract** from source against the static Leptonica:
     `./configure --disable-shared --enable-static --disable-legacy` (drop training tools).
   - Build the Go binary:
     `CGO_ENABLED=1 go build -ldflags '-s -w -linkmode external -extldflags "-static -static-libstdc++ -static-libgcc"' -tags cgo ./cmd/image-mcp`
2. **Wire multi-arch** via `docker buildx --platform linux/amd64,linux/arm64`, exporting
   binaries to `dist/`.
3. **Update `Makefile`** — add a `static` / `dist-static` target that drives the buildx build,
   replacing the dynamic `dist-linux` path for releases.
4. **Update release/CI** to publish the static binaries as the `linux-{amd64,arm64}` artifacts.
5. **Retire the symlink stopgap** from any container Dockerfiles once static binaries ship.
6. **Correct `TODO_STATIC_OCR_ALL_PLATFORMS.md`** — change Linux status from "✅ Implemented"
   to reflect that this proposal is the real implementation.

---

## 8. Verification / acceptance criteria

A build is accepted only when, for **each** of `linux/amd64` and `linux/arm64`:

1. `ldd ./image-tools-mcp` → **`not a dynamic executable`** (or `statically linked`).
2. `file ./image-tools-mcp` → `ELF ... statically linked`.
3. `./image-tools-mcp --version` → exits 0 and prints the expected version.
4. **OCR smoke test:** an MCP `image_ocr_full` (or `image_ocr_region`) call on a known
   `testdata` image returns the expected text — proving Tesseract + embedded tessdata work
   without any system Tesseract present.
5. **Cross-base test:** the same binary runs (steps 1–4) when dropped into **at least two
   different base images** (e.g. bookworm, trixie, and bare Alpine) — proving base/soname
   independence.

---

## 9. Costs and risks

| Item | Estimate / note |
|------|-----------------|
| Binary size | ~25–30 MB (≈6 MB Tesseract + Leptonica + trimmed codecs, plus ~15 MB embedded tessdata). Up from ~2.7 MB dynamic — acceptable for a self-contained tool. |
| Build time | Source-building Leptonica + Tesseract per arch adds minutes; cache the builder layers. |
| Effort | ~1–2 days to author + validate `Dockerfile.static` and the buildx pipeline. |
| Risk: missing static `.a` in Alpine packages | Mitigated by building Leptonica/Tesseract from source (the plan already does this). |
| Risk: musl vs glibc numeric/locale edge differences | Low — our codepaths are image decode + OCR; covered by the OCR smoke test (§8.4). |
| Risk: qemu arm64 build slowness in CI | Mitigated by native arm64 runners where available. |

---

## 10. Summary

The soname break was a symptom of distributing a **dynamically linked** binary across a fleet
whose base images drift. The durable fix is a **fully static, single-file-per-arch** binary
built in **Alpine/musl** with a **trimmed codec set** (PNG/JPEG only), preserving the existing
embedded-tessdata design. This decouples the tool from every base image forever — this upgrade
and the next. macOS/Windows need no change: they already ship in a portable, shell-out shape.

**Do it after the container upgrade; build and verify the static binaries against the new common
base in the same pass.**
