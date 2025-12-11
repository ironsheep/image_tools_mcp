# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev tesseract-ocr-dev leptonica-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o /image-mcp ./cmd/image-mcp

# Runtime stage
FROM alpine:3.20

# Install Tesseract OCR, language data, and Node.js
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    ca-certificates \
    nodejs \
    npm

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /image-mcp /app/image-mcp

# Create images directory
RUN mkdir -p /images && chown appuser:appuser /images

USER appuser

ENTRYPOINT ["/app/image-mcp"]
