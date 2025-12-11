package server

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns all available tools
func GetToolDefinitions() []Tool {
	return []Tool{
		// Basic Image Information
		{
			Name:        "image_load",
			Description: "Load an image file and return its dimensions and format. Sets this as the active image for subsequent operations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "image_dimensions",
			Description: "Get the width and height of an image file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
				},
				"required": []string{"path"},
			},
		},

		// Region Operations
		{
			Name:        "image_crop",
			Description: "Crop a rectangular region from an image and return it as base64-encoded PNG. Use this to zoom into areas that need detailed examination.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"x1": map[string]interface{}{
						"type":        "integer",
						"description": "Left edge X coordinate (0-based)",
					},
					"y1": map[string]interface{}{
						"type":        "integer",
						"description": "Top edge Y coordinate (0-based)",
					},
					"x2": map[string]interface{}{
						"type":        "integer",
						"description": "Right edge X coordinate (exclusive)",
					},
					"y2": map[string]interface{}{
						"type":        "integer",
						"description": "Bottom edge Y coordinate (exclusive)",
					},
					"scale": map[string]interface{}{
						"type":        "number",
						"description": "Optional scale factor (e.g., 2.0 to double size). Default 1.0",
						"default":     1.0,
					},
				},
				"required": []string{"path", "x1", "y1", "x2", "y2"},
			},
		},
		{
			Name:        "image_crop_quadrant",
			Description: "Crop a named region of the image (top-left, top-right, bottom-left, bottom-right, top-half, bottom-half, left-half, right-half, center).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"region": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"top-left", "top-right", "bottom-left", "bottom-right", "top-half", "bottom-half", "left-half", "right-half", "center"},
						"description": "Named region to extract",
					},
					"scale": map[string]interface{}{
						"type":        "number",
						"description": "Optional scale factor. Default 1.0",
						"default":     1.0,
					},
				},
				"required": []string{"path", "region"},
			},
		},

		// Color Operations
		{
			Name:        "image_sample_color",
			Description: "Get the exact color value at a specific pixel coordinate.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"x": map[string]interface{}{
						"type":        "integer",
						"description": "X coordinate (0-based, from left)",
					},
					"y": map[string]interface{}{
						"type":        "integer",
						"description": "Y coordinate (0-based, from top)",
					},
				},
				"required": []string{"path", "x", "y"},
			},
		},
		{
			Name:        "image_sample_colors_multi",
			Description: "Get color values at multiple pixel coordinates in a single call.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"points": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"x":     map[string]interface{}{"type": "integer"},
								"y":     map[string]interface{}{"type": "integer"},
								"label": map[string]interface{}{"type": "string", "description": "Optional label for this point"},
							},
							"required": []string{"x", "y"},
						},
						"description": "Array of points to sample",
					},
				},
				"required": []string{"path", "points"},
			},
		},
		{
			Name:        "image_dominant_colors",
			Description: "Analyze an image and return the N most dominant colors (color palette extraction).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"count": map[string]interface{}{
						"type":        "integer",
						"description": "Number of dominant colors to return (default 5)",
						"default":     5,
					},
					"region": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"x1": map[string]interface{}{"type": "integer"},
							"y1": map[string]interface{}{"type": "integer"},
							"x2": map[string]interface{}{"type": "integer"},
							"y2": map[string]interface{}{"type": "integer"},
						},
						"description": "Optional region to analyze. If omitted, analyzes entire image.",
					},
				},
				"required": []string{"path"},
			},
		},

		// Measurement Operations
		{
			Name:        "image_measure_distance",
			Description: "Measure the distance in pixels between two points.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"x1": map[string]interface{}{"type": "integer", "description": "First point X"},
					"y1": map[string]interface{}{"type": "integer", "description": "First point Y"},
					"x2": map[string]interface{}{"type": "integer", "description": "Second point X"},
					"y2": map[string]interface{}{"type": "integer", "description": "Second point Y"},
				},
				"required": []string{"path", "x1", "y1", "x2", "y2"},
			},
		},
		{
			Name:        "image_grid_overlay",
			Description: "Return a version of the image with a coordinate grid overlay for precise positioning reference.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"grid_spacing": map[string]interface{}{
						"type":        "integer",
						"description": "Pixels between grid lines (default 50)",
						"default":     50,
					},
					"show_coordinates": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to label grid intersections with coordinates",
						"default":     true,
					},
					"grid_color": map[string]interface{}{
						"type":        "string",
						"description": "Grid line color as hex (default #FF000080 - semi-transparent red)",
						"default":     "#FF000080",
					},
				},
				"required": []string{"path"},
			},
		},

		// OCR Operations
		{
			Name:        "image_ocr_full",
			Description: "Extract all text from the image using OCR. Returns text with approximate bounding boxes.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"language": map[string]interface{}{
						"type":        "string",
						"description": "OCR language hint (default 'eng')",
						"default":     "eng",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "image_ocr_region",
			Description: "Extract text from a specific rectangular region of the image.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"x1": map[string]interface{}{"type": "integer"},
					"y1": map[string]interface{}{"type": "integer"},
					"x2": map[string]interface{}{"type": "integer"},
					"y2": map[string]interface{}{"type": "integer"},
					"language": map[string]interface{}{
						"type":    "string",
						"default": "eng",
					},
				},
				"required": []string{"path", "x1", "y1", "x2", "y2"},
			},
		},
		{
			Name:        "image_detect_text_regions",
			Description: "Detect all regions in the image that contain text. Returns bounding boxes without performing full OCR.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"min_confidence": map[string]interface{}{
						"type":        "number",
						"description": "Minimum confidence threshold (0-1, default 0.5)",
						"default":     0.5,
					},
				},
				"required": []string{"path"},
			},
		},

		// Shape Detection
		{
			Name:        "image_detect_rectangles",
			Description: "Detect rectangular shapes in the image. Useful for finding boxes in diagrams.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"min_area": map[string]interface{}{
						"type":        "integer",
						"description": "Minimum area in pixels to consider (default 100)",
						"default":     100,
					},
					"tolerance": map[string]interface{}{
						"type":        "number",
						"description": "How close to rectangular a shape must be (0-1, default 0.9)",
						"default":     0.9,
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "image_detect_lines",
			Description: "Detect line segments in the image. Useful for finding connections between elements.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"min_length": map[string]interface{}{
						"type":        "integer",
						"description": "Minimum line length in pixels (default 20)",
						"default":     20,
					},
					"detect_arrows": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to detect arrow heads at line endpoints",
						"default":     true,
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "image_detect_circles",
			Description: "Detect circular shapes in the image. Useful for finding nodes, connectors, or bullets.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"min_radius": map[string]interface{}{
						"type":        "integer",
						"description": "Minimum radius in pixels (default 5)",
						"default":     5,
					},
					"max_radius": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum radius in pixels (default 500)",
						"default":     500,
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "image_edge_detect",
			Description: "Return an edge-detected version of the image, showing only structural lines. Useful for understanding diagram structure without color fills.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"threshold_low": map[string]interface{}{
						"type":        "integer",
						"description": "Low threshold for Canny edge detection (default 50)",
						"default":     50,
					},
					"threshold_high": map[string]interface{}{
						"type":        "integer",
						"description": "High threshold for Canny edge detection (default 150)",
						"default":     150,
					},
				},
				"required": []string{"path"},
			},
		},

		// Analysis Helpers
		{
			Name:        "image_check_alignment",
			Description: "Check if multiple points or regions are horizontally or vertically aligned.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"points": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"x": map[string]interface{}{"type": "integer"},
								"y": map[string]interface{}{"type": "integer"},
							},
							"required": []string{"x", "y"},
						},
						"description": "Points to check for alignment",
					},
					"tolerance": map[string]interface{}{
						"type":        "integer",
						"description": "Pixel tolerance for alignment (default 5)",
						"default":     5,
					},
				},
				"required": []string{"path", "points"},
			},
		},
		{
			Name:        "image_compare_regions",
			Description: "Compare two regions of an image to determine if they contain similar content (useful for detecting repeated elements).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the image file",
					},
					"region1": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"x1": map[string]interface{}{"type": "integer"},
							"y1": map[string]interface{}{"type": "integer"},
							"x2": map[string]interface{}{"type": "integer"},
							"y2": map[string]interface{}{"type": "integer"},
						},
						"required": []string{"x1", "y1", "x2", "y2"},
					},
					"region2": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"x1": map[string]interface{}{"type": "integer"},
							"y1": map[string]interface{}{"type": "integer"},
							"x2": map[string]interface{}{"type": "integer"},
							"y2": map[string]interface{}{"type": "integer"},
						},
						"required": []string{"x1", "y1", "x2", "y2"},
					},
				},
				"required": []string{"path", "region1", "region2"},
			},
		},
	}
}

// handleToolsList returns the list of available tools
func (s *Server) handleToolsList(req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": GetToolDefinitions(),
		},
	}
}
