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
        # Check multiple possible paths
        system_file=""
        for sys_path in \
            "/usr/share/tesseract-ocr/5/tessdata/${file}" \
            "/usr/share/tesseract-ocr/4/tessdata/${file}" \
            "/usr/share/tesseract-ocr/tessdata/${file}" \
            "/usr/share/tessdata/${file}"; do
            if [ -f "$sys_path" ]; then
                system_file="$sys_path"
                break
            fi
        done

        if [ -n "$system_file" ]; then
            cp "$system_file" "$dest"
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
