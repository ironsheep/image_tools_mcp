# Dev container for Image MCP development
FROM mcr.microsoft.com/devcontainers/go:1.22-bookworm

# Install Tesseract OCR and development dependencies
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        tesseract-ocr \
        tesseract-ocr-eng \
        libtesseract-dev \
        libleptonica-dev \
        pkg-config \
        # Image processing libraries
        libpng-dev \
        libjpeg-dev \
        libgif-dev \
        # Useful tools
        jq \
        imagemagick \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set up Go environment
ENV CGO_ENABLED=1
ENV GOPROXY=https://proxy.golang.org,direct

# Create images directory for testing
RUN mkdir -p /images && chown vscode:vscode /images

USER vscode

WORKDIR /workspace
