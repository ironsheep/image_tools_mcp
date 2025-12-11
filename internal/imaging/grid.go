package imaging

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strconv"

	"github.com/disintegration/imaging"
)

// GridOverlayResult contains the image with grid overlay
type GridOverlayResult struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ImageBase64 string `json:"image_base64"`
	MimeType    string `json:"mime_type"`
	GridSpacing int    `json:"grid_spacing"`
}

// GridOverlay adds a coordinate grid overlay to an image
func GridOverlay(img image.Image, gridSpacing int, showCoordinates bool, gridColorHex string) (*GridOverlayResult, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Parse grid color
	gridColor, err := parseHexColor(gridColorHex)
	if err != nil {
		gridColor = color.RGBA{255, 0, 0, 128} // Default: semi-transparent red
	}

	// Create a new RGBA image
	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, img, bounds.Min, draw.Src)

	// Draw vertical lines
	for x := gridSpacing; x < width; x += gridSpacing {
		for y := 0; y < height; y++ {
			result.Set(x, y, gridColor)
		}
	}

	// Draw horizontal lines
	for y := gridSpacing; y < height; y += gridSpacing {
		for x := 0; x < width; x++ {
			result.Set(x, y, gridColor)
		}
	}

	// Draw coordinate labels if requested
	if showCoordinates {
		labelColor := color.RGBA{255, 255, 255, 255}
		bgColor := color.RGBA{0, 0, 0, 180}

		for y := gridSpacing; y < height; y += gridSpacing {
			for x := gridSpacing; x < width; x += gridSpacing {
				label := fmt.Sprintf("%d,%d", x, y)
				drawLabel(result, x+2, y+2, label, labelColor, bgColor)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, result); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return &GridOverlayResult{
		Width:       width,
		Height:      height,
		ImageBase64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		MimeType:    "image/png",
		GridSpacing: gridSpacing,
	}, nil
}

// parseHexColor parses a hex color string like "#FF0000" or "#FF000080"
func parseHexColor(hex string) (color.RGBA, error) {
	if len(hex) == 0 {
		return color.RGBA{}, fmt.Errorf("empty color string")
	}
	if hex[0] == '#' {
		hex = hex[1:]
	}

	var r, g, b, a uint8 = 0, 0, 0, 255

	switch len(hex) {
	case 6:
		val, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return color.RGBA{}, err
		}
		r = uint8(val >> 16)
		g = uint8(val >> 8)
		b = uint8(val)
	case 8:
		val, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return color.RGBA{}, err
		}
		r = uint8(val >> 24)
		g = uint8(val >> 16)
		b = uint8(val >> 8)
		a = uint8(val)
	default:
		return color.RGBA{}, fmt.Errorf("invalid hex color length")
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

// drawLabel draws a simple text label at the given position
// This is a basic implementation - for production, consider using a font library
func drawLabel(img *image.RGBA, x, y int, text string, fg, bg color.RGBA) {
	// Simple 3x5 pixel font for digits and comma
	glyphs := map[rune][]string{
		'0': {"111", "101", "101", "101", "111"},
		'1': {"010", "110", "010", "010", "111"},
		'2': {"111", "001", "111", "100", "111"},
		'3': {"111", "001", "111", "001", "111"},
		'4': {"101", "101", "111", "001", "001"},
		'5': {"111", "100", "111", "001", "111"},
		'6': {"111", "100", "111", "101", "111"},
		'7': {"111", "001", "001", "001", "001"},
		'8': {"111", "101", "111", "101", "111"},
		'9': {"111", "101", "111", "001", "111"},
		',': {"000", "000", "000", "010", "010"},
	}

	bounds := img.Bounds()
	charWidth := 4
	labelWidth := len(text) * charWidth
	labelHeight := 7

	// Draw background
	for dy := -1; dy < labelHeight; dy++ {
		for dx := -1; dx < labelWidth; dx++ {
			px, py := x+dx, y+dy
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				img.Set(px, py, bg)
			}
		}
	}

	// Draw text
	cx := x
	for _, ch := range text {
		glyph, ok := glyphs[ch]
		if !ok {
			cx += charWidth
			continue
		}
		for row, line := range glyph {
			for col, pixel := range line {
				if pixel == '1' {
					px, py := cx+col, y+row
					if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
						img.Set(px, py, fg)
					}
				}
			}
		}
		cx += charWidth
	}
}

// Ensure imaging package is used (it's used in crop.go but we import it here for consistency)
var _ = imaging.Crop
