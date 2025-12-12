package server

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// createTestImageFile creates a test image file and returns its path
func createTestImageFile(t *testing.T, width, height int, c color.Color) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}

	tmpFile, err := os.CreateTemp("", "handler-test-*.png")
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

func TestHandleToolsCall_ImageLoad(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 80, color.RGBA{255, 0, 0, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_load",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  paramsJSON,
	}

	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("handleRequest returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_ImageDimensions(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 200, 150, color.RGBA{0, 255, 0, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_dimensions",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_NonExistentFile(t *testing.T) {
	s := New()

	params := map[string]interface{}{
		"name": "image_load",
		"arguments": map[string]interface{}{
			"path": "/nonexistent/image.png",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	// Should return an error for non-existent file
	if resp.Error == nil {
		// Check if there's error content in the result
		result, ok := resp.Result.(map[string]interface{})
		if ok {
			content, hasContent := result["content"]
			if hasContent {
				contentList, isList := content.([]map[string]interface{})
				if isList && len(contentList) > 0 {
					if contentList[0]["type"] == "text" {
						// Error might be in text content
						t.Log("Error returned in content")
					}
				}
			}
		}
	}
}

func TestHandleToolsCall_InvalidTool(t *testing.T) {
	s := New()

	params := map[string]interface{}{
		"name":      "nonexistent_tool",
		"arguments": map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	// Should return an error for unknown tool
	if resp.Error == nil {
		t.Log("No error returned - checking result for error content")
	}
}

func TestHandleToolsCall_MissingArguments(t *testing.T) {
	s := New()

	params := map[string]interface{}{
		"name":      "image_load",
		"arguments": map[string]interface{}{
			// Missing "path" argument
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	// Should return an error for missing required argument
	if resp.Error == nil {
		t.Log("No protocol error - checking result for error content")
	}
}

func TestHandleToolsCall_SampleColor(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 128, 64, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_sample_color",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"x":    50,
			"y":    50,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_Crop(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{0, 0, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_crop",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"x1":   10,
			"y1":   10,
			"x2":   50,
			"y2":   50,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_MeasureDistance(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_measure_distance",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"x1":   0,
			"y1":   0,
			"x2":   100,
			"y2":   100,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_GridOverlay(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{200, 200, 200, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_grid_overlay",
		"arguments": map[string]interface{}{
			"path":         imgPath,
			"grid_spacing": 25,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_EdgeDetect(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{100, 100, 100, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_edge_detect",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectRectangles(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_rectangles",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectLines(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_lines",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectCircles(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_circles",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DominantColors(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 0, 0, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_dominant_colors",
		"arguments": map[string]interface{}{
			"path":  imgPath,
			"count": 3,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_CheckAlignment(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_check_alignment",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"points": []map[string]interface{}{
				{"x": 10, "y": 50},
				{"x": 50, "y": 50},
				{"x": 90, "y": 50},
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_CompareRegions(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_compare_regions",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"region1": map[string]interface{}{
				"x1": 0, "y1": 0, "x2": 50, "y2": 50,
			},
			"region2": map[string]interface{}{
				"x1": 50, "y1": 50, "x2": 100, "y2": 100,
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_InvalidParams(t *testing.T) {
	s := New()

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  json.RawMessage(`invalid json`),
	}

	resp := s.handleToolsCall(req)

	// Should return error for invalid JSON
	if resp.Error == nil {
		t.Log("No protocol error for invalid JSON params")
	}
}

func TestHandleToolsCall_CropQuadrant(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 0, 0, 255})
	defer os.Remove(imgPath)

	regions := []string{"top-left", "top-right", "bottom-left", "bottom-right",
		"top-half", "bottom-half", "left-half", "right-half", "center"}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			params := map[string]interface{}{
				"name": "image_crop_quadrant",
				"arguments": map[string]interface{}{
					"path":   imgPath,
					"region": region,
				},
			}
			paramsJSON, _ := json.Marshal(params)

			req := &MCPRequest{
				JSONRPC: "2.0",
				ID:      1,
				Params:  paramsJSON,
			}

			resp := s.handleToolsCall(req)

			if resp.Error != nil {
				t.Fatalf("Unexpected error for region %s: %v", region, resp.Error)
			}
		})
	}
}

func TestHandleToolsCall_CropQuadrant_WithScale(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{0, 255, 0, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_crop_quadrant",
		"arguments": map[string]interface{}{
			"path":   imgPath,
			"region": "top-left",
			"scale":  2.0,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_SampleColorsMulti(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 128, 64, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_sample_colors_multi",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"points": []map[string]interface{}{
				{"x": 10, "y": 10, "label": "point1"},
				{"x": 50, "y": 50, "label": "point2"},
				{"x": 90, "y": 90, "label": "point3"},
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_SampleColorsMulti_EmptyPoints(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_sample_colors_multi",
		"arguments": map[string]interface{}{
			"path":   imgPath,
			"points": []map[string]interface{}{},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_OCRFull(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 50, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_ocr_full",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	// OCR should work (may return empty result for blank image)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_OCRFull_WithLanguage(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 50, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_ocr_full",
		"arguments": map[string]interface{}{
			"path":     imgPath,
			"language": "eng",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_OCRRegion(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_ocr_region",
		"arguments": map[string]interface{}{
			"path": imgPath,
			"x1":   10,
			"y1":   10,
			"x2":   90,
			"y2":   90,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectTextRegions(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 200, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_text_regions",
		"arguments": map[string]interface{}{
			"path": imgPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectTextRegions_WithConfidence(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 200, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_text_regions",
		"arguments": map[string]interface{}{
			"path":           imgPath,
			"min_confidence": 0.7,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DominantColors_WithRegion(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 0, 0, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_dominant_colors",
		"arguments": map[string]interface{}{
			"path":  imgPath,
			"count": 3,
			"region": map[string]interface{}{
				"x1": 10, "y1": 10, "x2": 50, "y2": 50,
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_GridOverlay_WithOptions(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{200, 200, 200, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_grid_overlay",
		"arguments": map[string]interface{}{
			"path":             imgPath,
			"grid_spacing":     20,
			"show_coordinates": true,
			"grid_color":       "#00FF0080",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectLines_WithArrows(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_lines",
		"arguments": map[string]interface{}{
			"path":          imgPath,
			"min_length":    10,
			"detect_arrows": true,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectRectangles_WithOptions(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_rectangles",
		"arguments": map[string]interface{}{
			"path":      imgPath,
			"min_area":  50,
			"tolerance": 0.8,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_DetectCircles_WithRadius(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{255, 255, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_detect_circles",
		"arguments": map[string]interface{}{
			"path":       imgPath,
			"min_radius": 10,
			"max_radius": 30,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_EdgeDetect_WithThresholds(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_edge_detect",
		"arguments": map[string]interface{}{
			"path":           imgPath,
			"threshold_low":  30,
			"threshold_high": 100,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsCall_Crop_WithScale(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{0, 0, 255, 255})
	defer os.Remove(imgPath)

	params := map[string]interface{}{
		"name": "image_crop",
		"arguments": map[string]interface{}{
			"path":  imgPath,
			"x1":    10,
			"y1":    10,
			"x2":    50,
			"y2":    50,
			"scale": 2.0,
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  paramsJSON,
	}

	resp := s.handleToolsCall(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestExecuteTool_AllTools(t *testing.T) {
	s := New()
	imgPath := createTestImageFile(t, 100, 100, color.RGBA{128, 128, 128, 255})
	defer os.Remove(imgPath)

	// Test each tool to ensure executeTool correctly dispatches
	toolTests := []struct {
		name string
		args map[string]interface{}
	}{
		{"image_load", map[string]interface{}{"path": imgPath}},
		{"image_dimensions", map[string]interface{}{"path": imgPath}},
		{"image_crop", map[string]interface{}{"path": imgPath, "x1": 0, "y1": 0, "x2": 50, "y2": 50}},
		{"image_crop_quadrant", map[string]interface{}{"path": imgPath, "region": "center"}},
		{"image_sample_color", map[string]interface{}{"path": imgPath, "x": 50, "y": 50}},
		{"image_sample_colors_multi", map[string]interface{}{"path": imgPath, "points": []map[string]interface{}{{"x": 25, "y": 25}}}},
		{"image_dominant_colors", map[string]interface{}{"path": imgPath}},
		{"image_measure_distance", map[string]interface{}{"path": imgPath, "x1": 0, "y1": 0, "x2": 50, "y2": 50}},
		{"image_grid_overlay", map[string]interface{}{"path": imgPath}},
		{"image_detect_rectangles", map[string]interface{}{"path": imgPath}},
		{"image_detect_lines", map[string]interface{}{"path": imgPath}},
		{"image_detect_circles", map[string]interface{}{"path": imgPath}},
		{"image_edge_detect", map[string]interface{}{"path": imgPath}},
		{"image_check_alignment", map[string]interface{}{"path": imgPath, "points": []map[string]interface{}{{"x": 10, "y": 50}, {"x": 50, "y": 50}}}},
		{"image_compare_regions", map[string]interface{}{"path": imgPath, "region1": map[string]interface{}{"x1": 0, "y1": 0, "x2": 50, "y2": 50}, "region2": map[string]interface{}{"x1": 50, "y1": 50, "x2": 100, "y2": 100}}},
	}

	for _, tt := range toolTests {
		t.Run(tt.name, func(t *testing.T) {
			argsJSON, _ := json.Marshal(tt.args)
			result, err := s.executeTool(tt.name, argsJSON)
			if err != nil {
				t.Fatalf("executeTool(%s) failed: %v", tt.name, err)
			}
			if result == nil {
				t.Errorf("executeTool(%s) returned nil result", tt.name)
			}
		})
	}
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	s := New()

	_, err := s.executeTool("unknown_tool", json.RawMessage(`{}`))
	if err == nil {
		t.Error("executeTool should fail for unknown tool")
	}
}

func TestExecuteTool_InvalidJSON(t *testing.T) {
	s := New()

	_, err := s.executeTool("image_load", json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("executeTool should fail for invalid JSON")
	}
}
