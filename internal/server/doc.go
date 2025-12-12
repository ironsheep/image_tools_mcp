// Package server implements the MCP (Model Context Protocol) server for image analysis tools.
//
// This package provides a JSON-RPC 2.0 server that exposes image processing capabilities
// through the MCP protocol. It's designed to work with Claude and other MCP-compatible
// clients, enabling AI systems to analyze images with precision.
//
// # Protocol
//
// The server communicates over stdio using JSON-RPC 2.0:
//   - Input: JSON-RPC requests on stdin (one per line)
//   - Output: JSON-RPC responses on stdout
//
// Supported MCP methods:
//   - initialize: Protocol handshake
//   - tools/list: Enumerate available tools
//   - tools/call: Execute a tool with arguments
//   - ping: Health check
//
// # Available Tools
//
// The server provides 19 image analysis tools organized into categories:
//
// Basic Image Information:
//   - image_load: Load image and get metadata
//   - image_dimensions: Get width and height
//
// Region Operations:
//   - image_crop: Extract rectangular region
//   - image_crop_quadrant: Extract named region (top-left, center, etc.)
//
// Color Operations:
//   - image_sample_color: Get color at pixel
//   - image_sample_colors_multi: Sample multiple points
//   - image_dominant_colors: Extract color palette
//
// Measurement Operations:
//   - image_measure_distance: Measure between points
//   - image_grid_overlay: Add coordinate grid
//
// OCR Operations:
//   - image_ocr_full: Extract all text
//   - image_ocr_region: Extract text from region
//   - image_detect_text_regions: Find text bounding boxes
//
// Shape Detection:
//   - image_detect_rectangles: Find rectangular shapes
//   - image_detect_lines: Find line segments
//   - image_detect_circles: Find circular shapes
//   - image_edge_detect: Canny edge detection
//
// Analysis Helpers:
//   - image_check_alignment: Check point alignment
//   - image_compare_regions: Compare two regions
//
// # Image Caching
//
// The server maintains an in-memory cache of loaded images. Images are cached
// by path and reused across multiple tool calls, avoiding redundant disk I/O.
// The cache persists for the lifetime of the server process.
//
// # Error Handling
//
// Tool execution errors are returned as JSON-RPC error responses with:
//   - code: -32000 (tool execution failure) or standard JSON-RPC codes
//   - message: Human-readable error description
//   - data: Additional error details (typically the Go error string)
//
// # Usage
//
// The server is typically started by an MCP client:
//
//	srv := server.New()
//	if err := srv.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// For Docker deployment, see the project's docker-compose.yml and MCP
// configuration examples.
package server
