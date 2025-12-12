package imaging

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// createTestImage creates a simple test image file and returns its path.
// The caller is responsible for removing the file.
func createTestImage(t *testing.T, width, height int, c color.Color) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}

	tmpFile, err := os.CreateTemp("", "test-image-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to encode image: %v", err)
	}

	return tmpFile.Name()
}

// createTestImageWithPattern creates a test image with a specific pattern
func createTestImageWithPattern(t *testing.T, width, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a pattern: red top-left, green top-right, blue bottom-left, white bottom-right
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var c color.Color
			if x < width/2 && y < height/2 {
				c = color.RGBA{255, 0, 0, 255} // Red
			} else if x >= width/2 && y < height/2 {
				c = color.RGBA{0, 255, 0, 255} // Green
			} else if x < width/2 && y >= height/2 {
				c = color.RGBA{0, 0, 255, 255} // Blue
			} else {
				c = color.RGBA{255, 255, 255, 255} // White
			}
			img.Set(x, y, c)
		}
	}

	tmpFile, err := os.CreateTemp("", "test-pattern-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to encode image: %v", err)
	}

	return tmpFile.Name()
}

func TestNewImageCache(t *testing.T) {
	cache := NewImageCache()
	if cache == nil {
		t.Fatal("NewImageCache returned nil")
	}
	if cache.images == nil {
		t.Fatal("NewImageCache did not initialize images map")
	}
}

func TestImageCache_Load(t *testing.T) {
	cache := NewImageCache()
	imgPath := createTestImage(t, 100, 100, color.RGBA{255, 0, 0, 255})
	defer os.Remove(imgPath)

	// First load
	img1, err := cache.Load(imgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if img1 == nil {
		t.Fatal("Load returned nil image")
	}

	bounds := img1.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("unexpected dimensions: got %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}

	// Second load should return cached image
	img2, err := cache.Load(imgPath)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if img1 != img2 {
		t.Error("second Load did not return cached image")
	}
}

func TestImageCache_Load_NonExistent(t *testing.T) {
	cache := NewImageCache()
	_, err := cache.Load("/nonexistent/path/to/image.png")
	if err == nil {
		t.Error("Load should fail for non-existent file")
	}
}

func TestImageCache_Load_InvalidImage(t *testing.T) {
	cache := NewImageCache()

	// Create a file with invalid image data
	tmpFile, err := os.CreateTemp("", "invalid-image-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.WriteString("not an image")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = cache.Load(tmpFile.Name())
	if err == nil {
		t.Error("Load should fail for invalid image data")
	}
}

func TestImageCache_Clear(t *testing.T) {
	cache := NewImageCache()
	imgPath := createTestImage(t, 50, 50, color.RGBA{0, 255, 0, 255})
	defer os.Remove(imgPath)

	// Load image
	_, err := cache.Load(imgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Clear cache
	cache.Clear()

	// Verify cache is empty by checking internal state
	cache.mu.RLock()
	count := len(cache.images)
	cache.mu.RUnlock()

	if count != 0 {
		t.Errorf("Clear did not empty cache: %d images remain", count)
	}
}

func TestImageCache_Evict(t *testing.T) {
	cache := NewImageCache()
	imgPath := createTestImage(t, 50, 50, color.RGBA{0, 0, 255, 255})
	defer os.Remove(imgPath)

	// Load image
	_, err := cache.Load(imgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Evict
	cache.Evict(imgPath)

	// Verify image is evicted
	cache.mu.RLock()
	_, exists := cache.images[imgPath]
	cache.mu.RUnlock()

	if exists {
		t.Error("Evict did not remove image from cache")
	}
}

func TestImageCache_Evict_NonExistent(t *testing.T) {
	cache := NewImageCache()
	// Should not panic
	cache.Evict("/nonexistent/path")
}

func TestImageCache_ConcurrentAccess(t *testing.T) {
	cache := NewImageCache()
	imgPath := createTestImage(t, 50, 50, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent loads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.Load(imgPath)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent Load error: %v", err)
	}
}

func TestLoadImageInfo(t *testing.T) {
	cache := NewImageCache()
	imgPath := createTestImage(t, 200, 150, color.RGBA{255, 128, 64, 255})
	defer os.Remove(imgPath)

	info, err := LoadImageInfo(cache, imgPath)
	if err != nil {
		t.Fatalf("LoadImageInfo failed: %v", err)
	}

	if info.Width != 200 {
		t.Errorf("Width: got %d, want 200", info.Width)
	}
	if info.Height != 150 {
		t.Errorf("Height: got %d, want 150", info.Height)
	}
	if info.Format != "png" {
		t.Errorf("Format: got %s, want png", info.Format)
	}
	if info.FileSizeBytes <= 0 {
		t.Error("FileSizeBytes should be positive")
	}
}

func TestLoadImageInfo_FormatDetection(t *testing.T) {
	cache := NewImageCache()

	tests := []struct {
		ext    string
		format string
	}{
		{".png", "png"},
		{".jpg", "jpeg"},
		{".jpeg", "jpeg"},
		{".gif", "gif"},
		{".xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			// Create a temp file with specific extension
			tmpDir := os.TempDir()
			tmpPath := filepath.Join(tmpDir, "test-format"+tt.ext)

			// Create a valid PNG regardless of extension
			img := image.NewRGBA(image.Rect(0, 0, 10, 10))
			f, err := os.Create(tmpPath)
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
			png.Encode(f, img)
			f.Close()
			defer os.Remove(tmpPath)

			info, err := LoadImageInfo(cache, tmpPath)
			if err != nil {
				t.Fatalf("LoadImageInfo failed: %v", err)
			}

			if info.Format != tt.format {
				t.Errorf("Format for %s: got %s, want %s", tt.ext, info.Format, tt.format)
			}
		})
	}
}

func TestLoadImageInfo_NonExistent(t *testing.T) {
	cache := NewImageCache()
	_, err := LoadImageInfo(cache, "/nonexistent/image.png")
	if err == nil {
		t.Error("LoadImageInfo should fail for non-existent file")
	}
}

func TestGetDimensions(t *testing.T) {
	cache := NewImageCache()
	imgPath := createTestImage(t, 300, 200, color.RGBA{100, 100, 100, 255})
	defer os.Remove(imgPath)

	dims, err := GetDimensions(cache, imgPath)
	if err != nil {
		t.Fatalf("GetDimensions failed: %v", err)
	}

	if dims.Width != 300 {
		t.Errorf("Width: got %d, want 300", dims.Width)
	}
	if dims.Height != 200 {
		t.Errorf("Height: got %d, want 200", dims.Height)
	}
}

func TestGetDimensions_NonExistent(t *testing.T) {
	cache := NewImageCache()
	_, err := GetDimensions(cache, "/nonexistent/image.png")
	if err == nil {
		t.Error("GetDimensions should fail for non-existent file")
	}
}
