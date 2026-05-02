package server

import (
	"encoding/json"
	"fmt"

	"github.com/ironsheep/image-tools-mcp/internal/detection"
	"github.com/ironsheep/image-tools-mcp/internal/imaging"
	"github.com/ironsheep/image-tools-mcp/internal/ocr"
)

// ToolCallParams represents the parameters for a tools/call MCP request.
type ToolCallParams struct {
	// Name is the tool to invoke (e.g., "image_load", "image_crop").
	Name string `json:"name"`

	// Arguments contains the tool-specific parameters as JSON.
	Arguments json.RawMessage `json:"arguments"`
}

// handleToolsCall processes a tools/call request and executes the specified tool.
//
// The response wraps the tool result in MCP's content format:
//
//	{
//	  "content": [{"type": "text", "text": "<JSON result>"}]
//	}
//
// Tool execution errors return a JSON-RPC error response with code -32000.
func (s *Server) handleToolsCall(req *MCPRequest) *MCPResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	result, err := s.executeTool(params.Name, params.Arguments)
	if err != nil {
		return s.errorResponse(req.ID, -32000, "Tool execution failed", err.Error())
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": mustMarshalJSON(result),
				},
			},
		},
	}
}

// executeTool dispatches tool execution to the appropriate handler function.
//
// Each tool handler:
//  1. Unmarshals arguments from JSON
//  2. Applies default values for optional parameters
//  3. Loads images from cache as needed
//  4. Calls the appropriate imaging/detection/ocr function
//  5. Returns the result or error
func (s *Server) executeTool(name string, args json.RawMessage) (interface{}, error) {
	switch name {
	// Basic Image Information
	case "image_load":
		return s.handleImageLoad(args)
	case "image_dimensions":
		return s.handleImageDimensions(args)

	// Region Operations
	case "image_crop":
		return s.handleImageCrop(args)
	case "image_crop_quadrant":
		return s.handleImageCropQuadrant(args)

	// Color Operations
	case "image_sample_color":
		return s.handleImageSampleColor(args)
	case "image_sample_colors_multi":
		return s.handleImageSampleColorsMulti(args)
	case "image_dominant_colors":
		return s.handleImageDominantColors(args)

	// Measurement Operations
	case "image_measure_distance":
		return s.handleImageMeasureDistance(args)
	case "image_grid_overlay":
		return s.handleImageGridOverlay(args)

	// OCR Operations
	case "image_ocr_full":
		return s.handleImageOCRFull(args)
	case "image_ocr_region":
		return s.handleImageOCRRegion(args)
	case "image_detect_text_regions":
		return s.handleImageDetectTextRegions(args)

	// Shape Detection
	case "image_detect_rectangles":
		return s.handleImageDetectRectangles(args)
	case "image_detect_lines":
		return s.handleImageDetectLines(args)
	case "image_detect_circles":
		return s.handleImageDetectCircles(args)
	case "image_edge_detect":
		return s.handleImageEdgeDetect(args)

	// Color Counting & Vectorization
	case "image_count_colors":
		return s.handleImageCountColors(args)
	case "image_vectorize":
		return s.handleImageVectorize(args)
	case "image_unbake_transparency":
		return s.handleImageUnbakeTransparency(args)

	// Analysis Helpers
	case "image_check_alignment":
		return s.handleImageCheckAlignment(args)
	case "image_compare_regions":
		return s.handleImageCompareRegions(args)

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// errorResponse creates a JSON-RPC error response with the given details.
func (s *Server) errorResponse(id interface{}, code int, message, data string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// mustMarshalJSON converts a value to pretty-printed JSON string.
// Panics are suppressed; on marshal failure, returns an empty string.
func mustMarshalJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// === Basic Image Information Handlers ===

type imageLoadArgs struct {
	Path string `json:"path"`
}

func (s *Server) handleImageLoad(args json.RawMessage) (interface{}, error) {
	var a imageLoadArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	return imaging.LoadImageInfo(s.cache, a.Path)
}

func (s *Server) handleImageDimensions(args json.RawMessage) (interface{}, error) {
	var a imageLoadArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	return imaging.GetDimensions(s.cache, a.Path)
}

// === Region Operation Handlers ===

type imageCropArgs struct {
	Path  string  `json:"path"`
	X1    int     `json:"x1"`
	Y1    int     `json:"y1"`
	X2    int     `json:"x2"`
	Y2    int     `json:"y2"`
	Scale float64 `json:"scale"`
}

func (s *Server) handleImageCrop(args json.RawMessage) (interface{}, error) {
	var a imageCropArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.Scale == 0 {
		a.Scale = 1.0
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.Crop(img, a.X1, a.Y1, a.X2, a.Y2, a.Scale)
}

type imageCropQuadrantArgs struct {
	Path   string  `json:"path"`
	Region string  `json:"region"`
	Scale  float64 `json:"scale"`
}

func (s *Server) handleImageCropQuadrant(args json.RawMessage) (interface{}, error) {
	var a imageCropQuadrantArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.Scale == 0 {
		a.Scale = 1.0
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.CropQuadrant(img, a.Region, a.Scale)
}

// === Color Operation Handlers ===

type imageSampleColorArgs struct {
	Path string `json:"path"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

func (s *Server) handleImageSampleColor(args json.RawMessage) (interface{}, error) {
	var a imageSampleColorArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.SampleColor(img, a.X, a.Y)
}

type imageSampleColorsMultiArgs struct {
	Path   string `json:"path"`
	Points []struct {
		X     int    `json:"x"`
		Y     int    `json:"y"`
		Label string `json:"label,omitempty"`
	} `json:"points"`
}

func (s *Server) handleImageSampleColorsMulti(args json.RawMessage) (interface{}, error) {
	var a imageSampleColorsMultiArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}

	points := make([]imaging.LabeledPoint, len(a.Points))
	for i, p := range a.Points {
		points[i] = imaging.LabeledPoint{X: p.X, Y: p.Y, Label: p.Label}
	}
	return imaging.SampleColorsMulti(img, points)
}

type imageDominantColorsArgs struct {
	Path   string `json:"path"`
	Count  int    `json:"count"`
	Region *struct {
		X1 int `json:"x1"`
		Y1 int `json:"y1"`
		X2 int `json:"x2"`
		Y2 int `json:"y2"`
	} `json:"region,omitempty"`
}

func (s *Server) handleImageDominantColors(args json.RawMessage) (interface{}, error) {
	var a imageDominantColorsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.Count == 0 {
		a.Count = 5
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}

	var region *imaging.Region
	if a.Region != nil {
		region = &imaging.Region{X1: a.Region.X1, Y1: a.Region.Y1, X2: a.Region.X2, Y2: a.Region.Y2}
	}
	return imaging.DominantColors(img, a.Count, region)
}

// === Measurement Operation Handlers ===

type imageMeasureDistanceArgs struct {
	Path string `json:"path"`
	X1   int    `json:"x1"`
	Y1   int    `json:"y1"`
	X2   int    `json:"x2"`
	Y2   int    `json:"y2"`
}

func (s *Server) handleImageMeasureDistance(args json.RawMessage) (interface{}, error) {
	var a imageMeasureDistanceArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.MeasureDistance(img, a.X1, a.Y1, a.X2, a.Y2)
}

type imageGridOverlayArgs struct {
	Path            string `json:"path"`
	GridSpacing     int    `json:"grid_spacing"`
	ShowCoordinates bool   `json:"show_coordinates"`
	GridColor       string `json:"grid_color"`
}

func (s *Server) handleImageGridOverlay(args json.RawMessage) (interface{}, error) {
	var a imageGridOverlayArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.GridSpacing == 0 {
		a.GridSpacing = 50
	}
	if a.GridColor == "" {
		a.GridColor = "#FF000080"
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.GridOverlay(img, a.GridSpacing, a.ShowCoordinates, a.GridColor)
}

// === OCR Operation Handlers ===

type imageOCRFullArgs struct {
	Path     string `json:"path"`
	Language string `json:"language"`
}

func (s *Server) handleImageOCRFull(args json.RawMessage) (interface{}, error) {
	var a imageOCRFullArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.Language == "" {
		a.Language = "eng"
	}
	return ocr.ExtractText(a.Path, a.Language)
}

type imageOCRRegionArgs struct {
	Path     string `json:"path"`
	X1       int    `json:"x1"`
	Y1       int    `json:"y1"`
	X2       int    `json:"x2"`
	Y2       int    `json:"y2"`
	Language string `json:"language"`
}

func (s *Server) handleImageOCRRegion(args json.RawMessage) (interface{}, error) {
	var a imageOCRRegionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.Language == "" {
		a.Language = "eng"
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return ocr.ExtractTextFromRegion(img, a.X1, a.Y1, a.X2, a.Y2, a.Language)
}

type imageDetectTextRegionsArgs struct {
	Path          string  `json:"path"`
	MinConfidence float64 `json:"min_confidence"`
}

func (s *Server) handleImageDetectTextRegions(args json.RawMessage) (interface{}, error) {
	var a imageDetectTextRegionsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.MinConfidence == 0 {
		a.MinConfidence = 0.5
	}
	return ocr.DetectTextRegions(a.Path, a.MinConfidence)
}

// === Shape Detection Handlers ===

type imageDetectRectanglesArgs struct {
	Path      string  `json:"path"`
	MinArea   int     `json:"min_area"`
	Tolerance float64 `json:"tolerance"`
}

func (s *Server) handleImageDetectRectangles(args json.RawMessage) (interface{}, error) {
	var a imageDetectRectanglesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.MinArea == 0 {
		a.MinArea = 100
	}
	if a.Tolerance == 0 {
		a.Tolerance = 0.9
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return detection.DetectRectangles(img, a.MinArea, a.Tolerance)
}

type imageDetectLinesArgs struct {
	Path         string `json:"path"`
	MinLength    int    `json:"min_length"`
	DetectArrows bool   `json:"detect_arrows"`
}

func (s *Server) handleImageDetectLines(args json.RawMessage) (interface{}, error) {
	var a imageDetectLinesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.MinLength == 0 {
		a.MinLength = 20
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return detection.DetectLines(img, a.MinLength, a.DetectArrows)
}

type imageDetectCirclesArgs struct {
	Path      string `json:"path"`
	MinRadius int    `json:"min_radius"`
	MaxRadius int    `json:"max_radius"`
}

func (s *Server) handleImageDetectCircles(args json.RawMessage) (interface{}, error) {
	var a imageDetectCirclesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.MinRadius == 0 {
		a.MinRadius = 5
	}
	if a.MaxRadius == 0 {
		a.MaxRadius = 500
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return detection.DetectCircles(img, a.MinRadius, a.MaxRadius)
}

type imageEdgeDetectArgs struct {
	Path          string `json:"path"`
	ThresholdLow  int    `json:"threshold_low"`
	ThresholdHigh int    `json:"threshold_high"`
}

func (s *Server) handleImageEdgeDetect(args json.RawMessage) (interface{}, error) {
	var a imageEdgeDetectArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.ThresholdLow == 0 {
		a.ThresholdLow = 50
	}
	if a.ThresholdHigh == 0 {
		a.ThresholdHigh = 150
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.EdgeDetect(img, a.ThresholdLow, a.ThresholdHigh)
}

// === Color Counting & Vectorization Handlers ===

type imageCountColorsArgs struct {
	Path           string `json:"path"`
	Quantize       int    `json:"quantize"`
	AlphaThreshold int    `json:"alpha_threshold"`
}

func (s *Server) handleImageCountColors(args json.RawMessage) (interface{}, error) {
	var a imageCountColorsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.CountColors(img, a.Quantize, a.AlphaThreshold)
}

type imageVectorizeArgs struct {
	Path           string  `json:"path"`
	MaxColors      int     `json:"max_colors"`
	Quantize       int     `json:"quantize"`
	TurdSize       int     `json:"turd_size"`
	AlphaMax       float64 `json:"alpha_max"`
	AlphaThreshold int     `json:"alpha_threshold"`
}

func (s *Server) handleImageVectorize(args json.RawMessage) (interface{}, error) {
	var a imageVectorizeArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	return imaging.Vectorize(img, imaging.VectorizeOptions{
		MaxColors:      a.MaxColors,
		Quantize:       a.Quantize,
		TurdSize:       a.TurdSize,
		AlphaMax:       a.AlphaMax,
		AlphaThreshold: a.AlphaThreshold,
	})
}

type imageUnbakeTransparencyArgs struct {
	Path               string   `json:"path"`
	OutputPath         string   `json:"output_path"`
	ForegroundColors   []string `json:"foreground_colors"`
	BgTolerance        int      `json:"bg_tolerance"`
	FgTolerance        int      `json:"fg_tolerance"`
	EdgeBlendTolerance float64  `json:"edge_blend_tolerance"`
	AntiFringeRadius   int      `json:"anti_fringe_radius"`
	MaxPreviewDim      int      `json:"max_preview_dim"`
	Checker            *struct {
		Color1Hex   string `json:"color1"`
		Color2Hex   string `json:"color2"`
		Period      int    `json:"period"`
		OriginX     int    `json:"origin_x"`
		OriginY     int    `json:"origin_y"`
		LightParity int    `json:"light_parity"`
	} `json:"checker,omitempty"`
}

func (s *Server) handleImageUnbakeTransparency(args json.RawMessage) (interface{}, error) {
	var a imageUnbakeTransparencyArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}
	outPath := a.OutputPath
	if outPath == "" {
		// Default: <input>_unbaked.png next to source
		base := a.Path
		ext := ""
		for i := len(base) - 1; i >= 0; i-- {
			if base[i] == '.' {
				ext = base[i:]
				base = base[:i]
				break
			}
			if base[i] == '/' || base[i] == '\\' {
				break
			}
		}
		if ext == "" {
			ext = ".png"
		}
		outPath = base + "_unbaked" + ext
	}
	opts := imaging.UnbakeOptions{
		ForegroundColors:   a.ForegroundColors,
		BgTolerance:        a.BgTolerance,
		FgTolerance:        a.FgTolerance,
		EdgeBlendTolerance: a.EdgeBlendTolerance,
		AntiFringeRadius:   a.AntiFringeRadius,
		MaxPreviewDim:      a.MaxPreviewDim,
	}
	if a.Checker != nil {
		c1, err := parseHexLocal(a.Checker.Color1Hex)
		if err != nil {
			return nil, fmt.Errorf("bad checker.color1: %w", err)
		}
		c2, err := parseHexLocal(a.Checker.Color2Hex)
		if err != nil {
			return nil, fmt.Errorf("bad checker.color2: %w", err)
		}
		opts.Checker = &imaging.CheckerInfo{
			Color1:      c1,
			Color2:      c2,
			Period:      a.Checker.Period,
			OriginX:     a.Checker.OriginX,
			OriginY:     a.Checker.OriginY,
			LightParity: a.Checker.LightParity,
		}
	}
	return imaging.UnbakeTransparency(img, outPath, opts)
}

func parseHexLocal(s string) (imaging.RGBColor, error) {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	if len(s) != 6 {
		return imaging.RGBColor{}, fmt.Errorf("expected #RRGGBB, got %q", s)
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		if _, err := fmt.Sscanf(s, "%02X%02X%02X", &r, &g, &b); err != nil {
			return imaging.RGBColor{}, err
		}
	}
	return imaging.RGBColor{R: r, G: g, B: b}, nil
}

// === Analysis Helper Handlers ===

type imageCheckAlignmentArgs struct {
	Path      string `json:"path"`
	Points    []struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"points"`
	Tolerance int `json:"tolerance"`
}

func (s *Server) handleImageCheckAlignment(args json.RawMessage) (interface{}, error) {
	var a imageCheckAlignmentArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if a.Tolerance == 0 {
		a.Tolerance = 5
	}

	points := make([]imaging.Point, len(a.Points))
	for i, p := range a.Points {
		points[i] = imaging.Point{X: p.X, Y: p.Y}
	}
	return imaging.CheckAlignment(points, a.Tolerance)
}

type imageCompareRegionsArgs struct {
	Path    string `json:"path"`
	Region1 struct {
		X1 int `json:"x1"`
		Y1 int `json:"y1"`
		X2 int `json:"x2"`
		Y2 int `json:"y2"`
	} `json:"region1"`
	Region2 struct {
		X1 int `json:"x1"`
		Y1 int `json:"y1"`
		X2 int `json:"x2"`
		Y2 int `json:"y2"`
	} `json:"region2"`
}

func (s *Server) handleImageCompareRegions(args json.RawMessage) (interface{}, error) {
	var a imageCompareRegionsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	img, err := s.cache.Load(a.Path)
	if err != nil {
		return nil, err
	}

	r1 := imaging.Region{X1: a.Region1.X1, Y1: a.Region1.Y1, X2: a.Region1.X2, Y2: a.Region1.Y2}
	r2 := imaging.Region{X1: a.Region2.X1, Y1: a.Region2.Y1, X2: a.Region2.X2, Y2: a.Region2.Y2}
	return imaging.CompareRegions(img, r1, r2)
}
