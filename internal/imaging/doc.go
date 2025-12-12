// Package imaging provides core image processing operations for the MCP server.
//
// This package implements fundamental image manipulation and analysis functions
// including color sampling, cropping, measurement, edge detection, and grid overlay.
// All operations work with standard Go image.Image types and use a coordinate system
// where (0,0) is at the top-left corner, X increases rightward, and Y increases downward.
//
// # Coordinate System
//
// All pixel coordinates in this package are 0-based:
//   - X: horizontal position (0 = leftmost pixel)
//   - Y: vertical position (0 = topmost pixel)
//   - Coordinates are inclusive for single points
//   - For regions, (x1,y1) is inclusive (top-left), (x2,y2) is exclusive (bottom-right)
//
// # Thread Safety
//
// The ImageCache type is safe for concurrent use. Individual image operations
// are stateless and can be called concurrently on different images. Operations
// on the same image should be synchronized by the caller if the image is mutable.
//
// # Color Representation
//
// Colors are returned in multiple formats for flexibility:
//   - Hex: 6-character format "#RRGGBB" (alpha excluded)
//   - RGB: 8-bit components (0-255)
//   - RGBA: 8-bit components with alpha (0-255)
//   - HSL: Hue (0-360), Saturation (0-100), Lightness (0-100)
//
// # Error Handling
//
// Functions return errors for invalid inputs such as:
//   - Coordinates outside image bounds
//   - Invalid region specifications (x1 >= x2 or y1 >= y2)
//   - File I/O errors during image loading
//   - Encoding errors during image output
//
// # Performance Considerations
//
// For repeated operations on the same image, use ImageCache to avoid redundant
// disk reads. Large images may consume significant memory when cached.
// Consider using Evict() or Clear() to manage memory for long-running processes.
package imaging
