package imaging

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sync"
)

// ImageCache provides thread-safe caching of loaded images
type ImageCache struct {
	mu     sync.RWMutex
	images map[string]image.Image
}

// NewImageCache creates a new image cache
func NewImageCache() *ImageCache {
	return &ImageCache{
		images: make(map[string]image.Image),
	}
}

// Load loads an image from disk, using the cache if available
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

// Clear removes all images from the cache
func (c *ImageCache) Clear() {
	c.mu.Lock()
	c.images = make(map[string]image.Image)
	c.mu.Unlock()
}

// Evict removes a specific image from the cache
func (c *ImageCache) Evict(path string) {
	c.mu.Lock()
	delete(c.images, path)
	c.mu.Unlock()
}

// ImageInfo contains metadata about an image
type ImageInfo struct {
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Format        string `json:"format"`
	ColorDepth    string `json:"color_depth"`
	HasAlpha      bool   `json:"has_alpha"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

// LoadImageInfo loads an image and returns its metadata
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

// DimensionsResult contains just width and height
type DimensionsResult struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GetDimensions returns just the dimensions of an image
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
