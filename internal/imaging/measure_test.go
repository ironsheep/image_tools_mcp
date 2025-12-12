package imaging

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestMeasureDistance(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	tests := []struct {
		name           string
		x1, y1, x2, y2 int
		wantDistance   float64
		wantDeltaX     int
		wantDeltaY     int
		wantAngle      float64
	}{
		{"horizontal right", 0, 50, 100, 50, 100, 100, 0, 0},
		{"horizontal left", 100, 50, 0, 50, 100, -100, 0, 180},
		{"vertical down", 50, 0, 50, 100, 100, 0, 100, 90},
		{"vertical up", 50, 100, 50, 0, 100, 0, -100, -90},
		{"diagonal", 0, 0, 100, 100, 141.42, 100, 100, 45},
		{"same point", 50, 50, 50, 50, 0, 0, 0, 0},
		{"3-4-5 triangle", 0, 0, 3, 4, 5, 3, 4, 53.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MeasureDistance(img, tt.x1, tt.y1, tt.x2, tt.y2)
			if err != nil {
				t.Fatalf("MeasureDistance failed: %v", err)
			}

			if result.DeltaX != tt.wantDeltaX {
				t.Errorf("DeltaX: got %d, want %d", result.DeltaX, tt.wantDeltaX)
			}
			if result.DeltaY != tt.wantDeltaY {
				t.Errorf("DeltaY: got %d, want %d", result.DeltaY, tt.wantDeltaY)
			}

			// Allow small tolerance for floating point
			if math.Abs(result.DistancePixels-tt.wantDistance) > 0.1 {
				t.Errorf("DistancePixels: got %.2f, want %.2f", result.DistancePixels, tt.wantDistance)
			}
			if math.Abs(result.AngleDegrees-tt.wantAngle) > 0.5 {
				t.Errorf("AngleDegrees: got %.1f, want %.1f", result.AngleDegrees, tt.wantAngle)
			}
		})
	}
}

func TestMeasureDistance_PercentValues(t *testing.T) {
	img := createInMemoryImage(200, 100, color.RGBA{255, 0, 0, 255})

	result, err := MeasureDistance(img, 0, 0, 100, 50)
	if err != nil {
		t.Fatalf("MeasureDistance failed: %v", err)
	}

	// Distance of ~111.8 pixels
	// As percentage of 200 width = ~56%
	// As percentage of 100 height = ~112%
	if result.DistancePercentWidth < 50 || result.DistancePercentWidth > 60 {
		t.Errorf("DistancePercentWidth: got %.1f, expected ~56", result.DistancePercentWidth)
	}
	if result.DistancePercentHeight < 100 || result.DistancePercentHeight > 120 {
		t.Errorf("DistancePercentHeight: got %.1f, expected ~112", result.DistancePercentHeight)
	}
}

func TestCheckAlignment(t *testing.T) {
	tests := []struct {
		name       string
		points     []Point
		tolerance  int
		wantHoriz  bool
		wantVert   bool
	}{
		{
			"horizontal line",
			[]Point{{10, 50}, {50, 50}, {90, 50}},
			1,
			true,
			false,
		},
		{
			"vertical line",
			[]Point{{50, 10}, {50, 50}, {50, 90}},
			1,
			false,
			true,
		},
		{
			"both aligned (single point)",
			[]Point{{50, 50}},
			1,
			true,
			true,
		},
		{
			"both aligned (empty)",
			[]Point{},
			1,
			true,
			true,
		},
		{
			"diagonal",
			[]Point{{10, 10}, {50, 50}, {90, 90}},
			1,
			false,
			false,
		},
		{
			"nearly horizontal",
			[]Point{{10, 50}, {50, 51}, {90, 49}},
			5,
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckAlignment(tt.points, tt.tolerance)
			if err != nil {
				t.Fatalf("CheckAlignment failed: %v", err)
			}

			if result.HorizontallyAligned != tt.wantHoriz {
				t.Errorf("HorizontallyAligned: got %v, want %v", result.HorizontallyAligned, tt.wantHoriz)
			}
			if result.VerticallyAligned != tt.wantVert {
				t.Errorf("VerticallyAligned: got %v, want %v", result.VerticallyAligned, tt.wantVert)
			}
		})
	}
}

func TestCheckAlignment_Averages(t *testing.T) {
	points := []Point{{10, 20}, {30, 40}, {50, 60}}

	result, err := CheckAlignment(points, 1)
	if err != nil {
		t.Fatalf("CheckAlignment failed: %v", err)
	}

	// Average X = (10+30+50)/3 = 30
	// Average Y = (20+40+60)/3 = 40
	if result.AverageX != 30 {
		t.Errorf("AverageX: got %.2f, want 30", result.AverageX)
	}
	if result.AverageY != 40 {
		t.Errorf("AverageY: got %.2f, want 40", result.AverageY)
	}
}

func TestCompareRegions(t *testing.T) {
	img := createPatternImage(100, 100)

	tests := []struct {
		name             string
		r1, r2           Region
		wantSimilar      bool   // expect > 0.9 similarity
		wantSameSize     bool
	}{
		{
			"identical regions",
			Region{X1: 0, Y1: 0, X2: 50, Y2: 50},
			Region{X1: 0, Y1: 0, X2: 50, Y2: 50},
			true,
			true,
		},
		{
			"different regions (red vs green)",
			Region{X1: 0, Y1: 0, X2: 50, Y2: 50},     // red
			Region{X1: 50, Y1: 0, X2: 100, Y2: 50},   // green
			false,
			true,
		},
		{
			"different sizes",
			Region{X1: 0, Y1: 0, X2: 50, Y2: 50},
			Region{X1: 0, Y1: 0, X2: 30, Y2: 30},
			true, // overlap is identical (both red top-left)
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareRegions(img, tt.r1, tt.r2)
			if err != nil {
				t.Fatalf("CompareRegions failed: %v", err)
			}

			if result.SameSize != tt.wantSameSize {
				t.Errorf("SameSize: got %v, want %v", result.SameSize, tt.wantSameSize)
			}

			highSimilarity := result.SimilarityScore > 0.9
			if highSimilarity != tt.wantSimilar {
				t.Errorf("SimilarityScore: got %.3f, wantSimilar=%v", result.SimilarityScore, tt.wantSimilar)
			}
		})
	}
}

func TestCompareRegions_Identical(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := CompareRegions(img,
		Region{X1: 10, Y1: 10, X2: 40, Y2: 40},
		Region{X1: 50, Y1: 50, X2: 80, Y2: 80},
	)
	if err != nil {
		t.Fatalf("CompareRegions failed: %v", err)
	}

	// Both regions are the same uniform color
	if result.SimilarityScore != 1.0 {
		t.Errorf("SimilarityScore for identical regions: got %.3f, want 1.0", result.SimilarityScore)
	}
	if result.PixelsDifferent != 0 {
		t.Errorf("PixelsDifferent: got %d, want 0", result.PixelsDifferent)
	}
	if result.AverageColorDiff != 0 {
		t.Errorf("AverageColorDiff: got %.2f, want 0", result.AverageColorDiff)
	}
}

func TestCompareRegions_RegionSizes(t *testing.T) {
	img := createInMemoryImage(100, 100, color.RGBA{255, 0, 0, 255})

	result, err := CompareRegions(img,
		Region{X1: 0, Y1: 0, X2: 30, Y2: 40},
		Region{X1: 50, Y1: 50, X2: 70, Y2: 80},
	)
	if err != nil {
		t.Fatalf("CompareRegions failed: %v", err)
	}

	if result.Region1Size.X != 30 || result.Region1Size.Y != 40 {
		t.Errorf("Region1Size: got %dx%d, want 30x40", result.Region1Size.X, result.Region1Size.Y)
	}
	if result.Region2Size.X != 20 || result.Region2Size.Y != 30 {
		t.Errorf("Region2Size: got %dx%d, want 20x30", result.Region2Size.X, result.Region2Size.Y)
	}

	// Total pixels should be min(30,20) * min(40,30) = 20 * 30 = 600
	if result.TotalPixels != 600 {
		t.Errorf("TotalPixels: got %d, want 600", result.TotalPixels)
	}
}

func TestAbsDiff(t *testing.T) {
	tests := []struct {
		a, b uint8
		want int
	}{
		{100, 50, 50},
		{50, 100, 50},
		{0, 255, 255},
		{255, 0, 255},
		{128, 128, 0},
	}

	for _, tt := range tests {
		got := absDiff(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("absDiff(%d, %d): got %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// createSolidColorRegion creates an image with a specific region filled with a color
func createSolidColorRegion(width, height int, r Region, c color.Color, bg color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x >= r.X1 && x < r.X2 && y >= r.Y1 && y < r.Y2 {
				img.Set(x, y, c)
			} else {
				img.Set(x, y, bg)
			}
		}
	}
	return img
}
