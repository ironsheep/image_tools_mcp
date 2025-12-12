package imaging

import (
	"fmt"
	"image"
	_ "image/gif"  // Register GIF format decoder
	_ "image/jpeg" // Register JPEG format decoder
	_ "image/png"  // Register PNG format decoder
	"os"
	"path/filepath"
	"sync"
)

// ImageCache provides thread-safe caching of loaded images to avoid redundant disk reads.
//
// The cache stores decoded image.Image objects keyed by their file path. Once an image
// is loaded, subsequent Load() calls for the same path return the cached copy without
// disk I/O.
//
// ImageCache is safe for concurrent use by multiple goroutines. All methods use
// appropriate locking to prevent data races.
//
// # Memory Management
//
// Cached images remain in memory until explicitly removed via Evict() or Clear().
// For long-running processes handling many images, consider periodic cleanup to
// prevent unbounded memory growth.
//
// # Example Usage
//
//	cache := imaging.NewImageCache()
//	img, err := cache.Load("/path/to/image.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Use img...
//	cache.Evict("/path/to/image.png") // Optional: free memory
type ImageCache struct {
	mu     sync.RWMutex
	images map[string]image.Image
}

// NewImageCache creates and initializes a new empty image cache.
//
// The returned cache is ready for immediate use and is safe for concurrent access.
func NewImageCache() *ImageCache {
	return &ImageCache{
		images: make(map[string]image.Image),
	}
}

// Load retrieves an image from the cache or loads it from disk if not cached.
//
// Parameters:
//   - path: Absolute or relative file path to the image. Supported formats are
//     PNG, JPEG, and GIF.
//
// Returns:
//   - image.Image: The decoded image. The concrete type depends on the image format
//     and color model (e.g., *image.RGBA, *image.NRGBA, *image.YCbCr).
//   - error: Non-nil if the file cannot be opened or decoded.
//
// The image is cached using the exact path string provided. Different paths to the
// same file (e.g., relative vs absolute) will result in separate cache entries.
//
// # Errors
//
//   - Returns error if the file does not exist or cannot be read
//   - Returns error if the file is not a valid PNG, JPEG, or GIF image
func (c *ImageCache) Load(path string) (image.Image, error) {
	c.mu.RLock()
	if img, ok := c.images[path]; ok {
		c.mu.RUnlock()
		return img, nil
	}
	c.mu.RUnlock()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	c.mu.Lock()
	c.images[path] = img
	c.mu.Unlock()

	return img, nil
}

// Clear removes all images from the cache, freeing the associated memory.
//
// This method is useful for long-running processes that need to release memory
// after processing a batch of images. After Clear(), all images must be reloaded
// from disk on subsequent Load() calls.
func (c *ImageCache) Clear() {
	c.mu.Lock()
	c.images = make(map[string]image.Image)
	c.mu.Unlock()
}

// Evict removes a specific image from the cache by its path.
//
// Parameters:
//   - path: The exact path string used when the image was loaded.
//
// If the path is not in the cache, this method does nothing.
// After eviction, the next Load() call for this path will read from disk.
func (c *ImageCache) Evict(path string) {
	c.mu.Lock()
	delete(c.images, path)
	c.mu.Unlock()
}

// ImageInfo contains metadata about a loaded image file.
//
// This struct provides essential information about an image without requiring
// the caller to analyze the image data directly.
type ImageInfo struct {
	// Width is the image width in pixels.
	Width int `json:"width"`

	// Height is the image height in pixels.
	Height int `json:"height"`

	// Format is the detected image format: "png", "jpeg", "gif", or "unknown".
	// Detection is based on file extension, not file contents.
	Format string `json:"format"`

	// ColorDepth indicates the bit depth per channel: "8-bit" or "16-bit".
	ColorDepth string `json:"color_depth"`

	// HasAlpha indicates whether the image has an alpha (transparency) channel.
	HasAlpha bool `json:"has_alpha"`

	// FileSizeBytes is the size of the image file on disk in bytes.
	FileSizeBytes int64 `json:"file_size_bytes"`
}

// LoadImageInfo loads an image and returns comprehensive metadata about it.
//
// This function loads the image into the cache (if not already cached) and
// extracts metadata including dimensions, format, color depth, alpha channel
// presence, and file size.
//
// Parameters:
//   - cache: The image cache to use for loading. Must not be nil.
//   - path: Path to the image file.
//
// Returns:
//   - *ImageInfo: Metadata about the image.
//   - error: Non-nil if the image cannot be loaded or the file cannot be stat'd.
//
// # Format Detection
//
// The format is determined by file extension:
//   - ".png" -> "png"
//   - ".jpg", ".jpeg" -> "jpeg"
//   - ".gif" -> "gif"
//   - Other extensions -> "unknown"
//
// # Color Depth Detection
//
// Color depth is determined by the Go image type:
//   - *image.RGBA64, *image.NRGBA64, *image.Gray16 -> "16-bit"
//   - All other types -> "8-bit"
func LoadImageInfo(cache *ImageCache, path string) (*ImageInfo, error) {
	img, err := cache.Load(path)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()

	// Get file info for size
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Determine format from extension
	ext := filepath.Ext(path)
	format := "unknown"
	switch ext {
	case ".png":
		format = "png"
	case ".jpg", ".jpeg":
		format = "jpeg"
	case ".gif":
		format = "gif"
	}

	// Check for alpha channel
	hasAlpha := false
	colorDepth := "8-bit"
	switch img.(type) {
	case *image.RGBA, *image.NRGBA:
		hasAlpha = true
	case *image.RGBA64, *image.NRGBA64:
		hasAlpha = true
		colorDepth = "16-bit"
	case *image.Gray16:
		colorDepth = "16-bit"
	}

	return &ImageInfo{
		Width:         bounds.Dx(),
		Height:        bounds.Dy(),
		Format:        format,
		ColorDepth:    colorDepth,
		HasAlpha:      hasAlpha,
		FileSizeBytes: stat.Size(),
	}, nil
}

// DimensionsResult contains the width and height of an image.
//
// This is a lightweight result type for when only dimensions are needed,
// without the additional metadata provided by ImageInfo.
type DimensionsResult struct {
	// Width is the image width in pixels.
	Width int `json:"width"`

	// Height is the image height in pixels.
	Height int `json:"height"`
}

// GetDimensions returns the dimensions of an image without additional metadata.
//
// This is a lightweight alternative to LoadImageInfo when only the width and
// height are needed. The image is loaded into the cache if not already present.
//
// Parameters:
//   - cache: The image cache to use for loading. Must not be nil.
//   - path: Path to the image file.
//
// Returns:
//   - *DimensionsResult: The image dimensions.
//   - error: Non-nil if the image cannot be loaded.
func GetDimensions(cache *ImageCache, path string) (*DimensionsResult, error) {
	img, err := cache.Load(path)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	return &DimensionsResult{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}, nil
}
