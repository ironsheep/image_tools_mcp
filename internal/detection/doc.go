// Package detection provides shape and feature detection algorithms for images.
//
// This package implements computer vision algorithms to detect geometric shapes
// (rectangles, circles, lines) and text regions within images. It's designed for
// analyzing diagrams, flowcharts, and structured visual content.
//
// # Shape Detection
//
// The package provides detection for common diagram elements:
//
//   - Rectangles: Using edge detection and contour analysis
//   - Circles: Using the Hough circle transform
//   - Lines: Using the Hough line transform with arrow detection
//   - Text regions: Using edge density heuristics
//
// # Algorithm Overview
//
// Most detection functions follow a similar pipeline:
//
//  1. Edge Detection: Convert to grayscale and detect edges using gradient thresholds
//  2. Feature Extraction: Apply shape-specific algorithms (Hough transform, contour finding)
//  3. Filtering: Remove duplicates and shapes below size/confidence thresholds
//  4. Result Formatting: Return structured data with coordinates, dimensions, and metadata
//
// # Coordinate System
//
// All coordinates use the standard image convention:
//   - Origin (0, 0) at top-left corner
//   - X increases rightward
//   - Y increases downward
//   - Bounding boxes use inclusive top-left and exclusive bottom-right
//
// # Confidence Scores
//
// Detection functions return confidence scores (0.0 to 1.0) indicating how well
// a detected shape matches the expected pattern:
//   - 1.0 = Perfect match
//   - 0.5 = Moderate confidence
//   - Lower values indicate uncertain detections
//
// Confidence calculation varies by shape type:
//   - Rectangles: Based on rectangularity (perimeter vs expected rectangle perimeter)
//   - Circles: Based on edge votes in Hough accumulator
//   - Lines: Based on vote count in Hough space
//   - Text regions: Based on edge density and horizontal structure
//
// # Performance Considerations
//
// Detection algorithms iterate over all pixels and may be computationally intensive
// for large images. The Hough transforms have O(n²) or O(n³) complexity depending
// on the parameter space searched.
//
// For large images, consider:
//   - Cropping to regions of interest first
//   - Using higher minimum size thresholds to reduce false positives
//   - Limiting the search space (e.g., min/max radius for circles)
//
// # Limitations
//
// These algorithms work best on clean, high-contrast images:
//   - Diagrams with solid lines and fills
//   - Images without heavy compression artifacts
//   - Shapes that are reasonably close to their ideal forms
//
// Noisy images, photographs, or hand-drawn content may produce poor results.
package detection
