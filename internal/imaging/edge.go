package imaging

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

// EdgeDetectResult contains an edge-detected image encoded as base64 PNG.
//
// The result is a grayscale image where white pixels (255) represent detected
// edges and black pixels (0) represent non-edges.
type EdgeDetectResult struct {
	// Width of the output image in pixels (same as input).
	Width int `json:"width"`

	// Height of the output image in pixels (same as input).
	Height int `json:"height"`

	// ImageBase64 is the edge image encoded as base64 PNG.
	// The image is grayscale with edges marked in white (255).
	ImageBase64 string `json:"image_base64"`

	// MimeType is always "image/png" for edge detection results.
	MimeType string `json:"mime_type"`
}

// EdgeDetect performs Canny-style edge detection on an image.
//
// This function identifies edges (boundaries between regions) in an image,
// producing a binary output where edges are white and non-edges are black.
// It's useful for understanding diagram structure without color fills.
//
// Parameters:
//   - img: Source image (color or grayscale).
//   - thresholdLow: Low threshold for edge detection (0-255). Edges with gradient
//     magnitude below this are discarded. Typical value: 50.
//   - thresholdHigh: High threshold for edge detection (0-255). Edges above this
//     are always kept. Typical value: 150.
//
// Returns:
//   - *EdgeDetectResult: Grayscale edge image as base64 PNG.
//   - error: Non-nil if PNG encoding fails.
//
// # Algorithm
//
// The implementation follows the Canny edge detection algorithm:
//
//  1. Grayscale conversion: RGB -> luminance using ITU-R BT.601 weights
//     (0.299*R + 0.587*G + 0.114*B)
//
//  2. Gaussian blur: 5x5 kernel to reduce noise
//
//  3. Gradient computation: Sobel operators for X and Y gradients
//     magnitude = sqrt(Gx² + Gy²)
//     direction = atan2(Gy, Gx)
//
//  4. Non-maximum suppression: Thin edges to 1-pixel width by keeping only
//     local maxima in the gradient direction
//
//  5. Hysteresis thresholding:
//     - Pixels above thresholdHigh are strong edges (always kept)
//     - Pixels between thresholdLow and thresholdHigh are weak edges
//     (kept only if connected to strong edges)
//     - Pixels below thresholdLow are discarded
//
// # Threshold Selection
//
// Lower thresholds detect more edges but increase noise. Higher thresholds
// produce cleaner results but may miss faint edges.
//
// Recommended starting points:
//   - Clean diagrams: thresholdLow=50, thresholdHigh=150
//   - Photographs: thresholdLow=100, thresholdHigh=200
//   - Noisy images: thresholdLow=75, thresholdHigh=175
func EdgeDetect(img image.Image, thresholdLow, thresholdHigh int) (*EdgeDetectResult, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Convert to grayscale
	gray := make([][]float64, height)
	for y := 0; y < height; y++ {
		gray[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			// Convert to 8-bit and compute luminance
			rf := float64(r>>8) / 255.0
			gf := float64(g>>8) / 255.0
			bf := float64(b>>8) / 255.0
			gray[y][x] = 0.299*rf + 0.587*gf + 0.114*bf
		}
	}

	// Apply Gaussian blur to reduce noise
	blurred := gaussianBlur(gray, width, height)

	// Compute gradients using Sobel operator
	gradX := make([][]float64, height)
	gradY := make([][]float64, height)
	magnitude := make([][]float64, height)
	direction := make([][]float64, height)

	sobelX := [][]float64{
		{-1, 0, 1},
		{-2, 0, 2},
		{-1, 0, 1},
	}
	sobelY := [][]float64{
		{-1, -2, -1},
		{0, 0, 0},
		{1, 2, 1},
	}

	for y := 0; y < height; y++ {
		gradX[y] = make([]float64, width)
		gradY[y] = make([]float64, width)
		magnitude[y] = make([]float64, width)
		direction[y] = make([]float64, width)

		for x := 0; x < width; x++ {
			var gx, gy float64
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					py := clamp(y+ky, 0, height-1)
					px := clamp(x+kx, 0, width-1)
					gx += blurred[py][px] * sobelX[ky+1][kx+1]
					gy += blurred[py][px] * sobelY[ky+1][kx+1]
				}
			}
			gradX[y][x] = gx
			gradY[y][x] = gy
			magnitude[y][x] = math.Sqrt(gx*gx + gy*gy)
			direction[y][x] = math.Atan2(gy, gx)
		}
	}

	// Non-maximum suppression
	suppressed := make([][]float64, height)
	for y := 0; y < height; y++ {
		suppressed[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			if y == 0 || y == height-1 || x == 0 || x == width-1 {
				continue
			}

			angle := direction[y][x]
			mag := magnitude[y][x]

			// Determine neighbors to compare based on gradient direction
			var n1, n2 float64
			if (angle >= -math.Pi/8 && angle < math.Pi/8) || (angle >= 7*math.Pi/8 || angle < -7*math.Pi/8) {
				n1 = magnitude[y][x-1]
				n2 = magnitude[y][x+1]
			} else if (angle >= math.Pi/8 && angle < 3*math.Pi/8) || (angle >= -7*math.Pi/8 && angle < -5*math.Pi/8) {
				n1 = magnitude[y-1][x+1]
				n2 = magnitude[y+1][x-1]
			} else if (angle >= 3*math.Pi/8 && angle < 5*math.Pi/8) || (angle >= -5*math.Pi/8 && angle < -3*math.Pi/8) {
				n1 = magnitude[y-1][x]
				n2 = magnitude[y+1][x]
			} else {
				n1 = magnitude[y-1][x-1]
				n2 = magnitude[y+1][x+1]
			}

			if mag >= n1 && mag >= n2 {
				suppressed[y][x] = mag
			}
		}
	}

	// Double threshold and edge tracking by hysteresis
	result := image.NewGray(bounds)
	lowThresh := float64(thresholdLow) / 255.0
	highThresh := float64(thresholdHigh) / 255.0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			val := suppressed[y][x]
			if val >= highThresh {
				result.SetGray(x+bounds.Min.X, y+bounds.Min.Y, color.Gray{255})
			} else if val >= lowThresh {
				// Check if connected to a strong edge
				hasStrongNeighbor := false
				for ky := -1; ky <= 1 && !hasStrongNeighbor; ky++ {
					for kx := -1; kx <= 1 && !hasStrongNeighbor; kx++ {
						py := clamp(y+ky, 0, height-1)
						px := clamp(x+kx, 0, width-1)
						if suppressed[py][px] >= highThresh {
							hasStrongNeighbor = true
						}
					}
				}
				if hasStrongNeighbor {
					result.SetGray(x+bounds.Min.X, y+bounds.Min.Y, color.Gray{255})
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, result); err != nil {
		return nil, fmt.Errorf("failed to encode edge image: %w", err)
	}

	return &EdgeDetectResult{
		Width:       width,
		Height:      height,
		ImageBase64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		MimeType:    "image/png",
	}, nil
}

// gaussianBlur applies a 5x5 Gaussian blur to reduce noise before edge detection.
//
// Uses a standard 5x5 Gaussian kernel with sigma ≈ 1.4:
//
//	1  4  7  4  1
//	4 16 26 16  4
//	7 26 41 26  7
//	4 16 26 16  4
//	1  4  7  4  1
//
// Total kernel sum = 273, used for normalization.
// Border pixels use clamped (replicated) edge values.
func gaussianBlur(img [][]float64, width, height int) [][]float64 {
	kernel := [][]float64{
		{1, 4, 7, 4, 1},
		{4, 16, 26, 16, 4},
		{7, 26, 41, 26, 7},
		{4, 16, 26, 16, 4},
		{1, 4, 7, 4, 1},
	}
	kernelSum := 273.0

	result := make([][]float64, height)
	for y := 0; y < height; y++ {
		result[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			var sum float64
			for ky := -2; ky <= 2; ky++ {
				for kx := -2; kx <= 2; kx++ {
					py := clamp(y+ky, 0, height-1)
					px := clamp(x+kx, 0, width-1)
					sum += img[py][px] * kernel[ky+2][kx+2]
				}
			}
			result[y][x] = sum / kernelSum
		}
	}
	return result
}

// clamp constrains an integer value to the range [min, max].
// Used for boundary handling in convolution operations.
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
