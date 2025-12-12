#!/bin/bash
#
# ensure-tessdata.sh - Download Tesseract training data if not present
#
# Downloads eng.traineddata and osd.traineddata to internal/ocr/tessdata/
# for embedding in the Linux binary via go:embed.
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TESSDATA_DIR="${REPO_ROOT}/internal/ocr/tessdata"

# Tesseract tessdata version (tessdata_fast is smaller and faster)
TESSDATA_REPO="https://github.com/tesseract-ocr/tessdata_fast/raw/main"

# Required files
FILES=(
    "eng.traineddata"
    "osd.traineddata"
)

mkdir -p "${TESSDATA_DIR}"

for file in "${FILES[@]}"; do
    dest="${TESSDATA_DIR}/${file}"

    if [ -f "$dest" ]; then
        echo "  [OK] ${file} already exists"
    else
        echo "  Downloading ${file}..."

        # Try system tessdata first (faster, no download)
        system_path="/usr/share/tesseract-ocr/5/tessdata/${file}"
        if [ -f "$system_path" ]; then
            cp "$system_path" "$dest"
            echo "  [OK] Copied from system: ${file}"
        else
            # Download from GitHub
            curl -fsSL "${TESSDATA_REPO}/${file}" -o "$dest"
            echo "  [OK] Downloaded: ${file}"
        fi
    fi
done

echo ""
echo "Tessdata ready in ${TESSDATA_DIR}"
ls -lh "${TESSDATA_DIR}"
