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

// EdgeDetectResult contains the edge-detected image
type EdgeDetectResult struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ImageBase64 string `json:"image_base64"`
	MimeType    string `json:"mime_type"`
}

// EdgeDetect performs Canny-style edge detection on an image
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

// gaussianBlur applies a 5x5 Gaussian blur
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

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
