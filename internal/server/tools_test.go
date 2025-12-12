package server

import (
	"testing"
)

func TestGetToolDefinitions(t *testing.T) {
	tools := GetToolDefinitions()

	if len(tools) == 0 {
		t.Fatal("GetToolDefinitions returned empty slice")
	}

	// Expected tools from CLAUDE.md
	expectedTools := []string{
		"image_load",
		"image_dimensions",
		"image_crop",
		"image_crop_quadrant",
		"image_sample_color",
		"image_sample_colors_multi",
		"image_dominant_colors",
		"image_measure_distance",
		"image_grid_overlay",
		"image_ocr_full",
		"image_ocr_region",
		"image_detect_text_regions",
		"image_detect_rectangles",
		"image_detect_lines",
		"image_detect_circles",
		"image_edge_detect",
		"image_check_alignment",
		"image_compare_regions",
	}

	toolMap := make(map[string]Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	// Check all expected tools exist
	for _, name := range expectedTools {
		if _, ok := toolMap[name]; !ok {
			t.Errorf("Expected tool %s not found", name)
		}
	}
}

func TestToolDefinitions_Structure(t *testing.T) {
	tools := GetToolDefinitions()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			// Name should not be empty
			if tool.Name == "" {
				t.Error("Tool name is empty")
			}

			// Description should not be empty
			if tool.Description == "" {
				t.Error("Tool description is empty")
			}

			// InputSchema should exist
			if tool.InputSchema == nil {
				t.Error("Tool InputSchema is nil")
			}

			// InputSchema should be an object type
			schemaType, ok := tool.InputSchema["type"]
			if !ok {
				t.Error("InputSchema missing 'type' field")
			}
			if schemaType != "object" {
				t.Errorf("InputSchema type: got %v, want 'object'", schemaType)
			}

			// InputSchema should have properties
			props, ok := tool.InputSchema["properties"]
			if !ok {
				t.Error("InputSchema missing 'properties' field")
			}
			if props == nil {
				t.Error("InputSchema properties is nil")
			}
		})
	}
}

func TestToolDefinitions_RequiredPath(t *testing.T) {
	// Most tools require a 'path' parameter
	toolsRequiringPath := []string{
		"image_load",
		"image_dimensions",
		"image_crop",
		"image_crop_quadrant",
		"image_sample_color",
		"image_sample_colors_multi",
		"image_dominant_colors",
		"image_measure_distance",
		"image_grid_overlay",
		"image_ocr_full",
		"image_ocr_region",
		"image_detect_text_regions",
		"image_detect_rectangles",
		"image_detect_lines",
		"image_detect_circles",
		"image_edge_detect",
		"image_check_alignment",
		"image_compare_regions",
	}

	tools := GetToolDefinitions()
	toolMap := make(map[string]Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	for _, name := range toolsRequiringPath {
		tool, ok := toolMap[name]
		if !ok {
			continue // Skip if tool not found
		}

		t.Run(name, func(t *testing.T) {
			required, ok := tool.InputSchema["required"]
			if !ok {
				t.Error("InputSchema missing 'required' field")
				return
			}

			requiredList, ok := required.([]string)
			if !ok {
				t.Error("'required' should be a string slice")
				return
			}

			hasPath := false
			for _, r := range requiredList {
				if r == "path" {
					hasPath = true
					break
				}
			}

			if !hasPath {
				t.Error("Tool should require 'path' parameter")
			}
		})
	}
}

func TestToolDefinitions_CropCoordinates(t *testing.T) {
	tools := GetToolDefinitions()

	var cropTool Tool
	for _, tool := range tools {
		if tool.Name == "image_crop" {
			cropTool = tool
			break
		}
	}

	if cropTool.Name == "" {
		t.Fatal("image_crop tool not found")
	}

	required, ok := cropTool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("required should be a string slice")
	}

	// image_crop requires path, x1, y1, x2, y2
	expectedRequired := map[string]bool{
		"path": true,
		"x1":   true,
		"y1":   true,
		"x2":   true,
		"y2":   true,
	}

	for _, r := range required {
		if expectedRequired[r] {
			delete(expectedRequired, r)
		}
	}

	for missing := range expectedRequired {
		t.Errorf("image_crop should require '%s' parameter", missing)
	}
}

func TestToolDefinitions_CropQuadrantRegions(t *testing.T) {
	tools := GetToolDefinitions()

	var tool Tool
	for _, tt := range tools {
		if tt.Name == "image_crop_quadrant" {
			tool = tt
			break
		}
	}

	if tool.Name == "" {
		t.Fatal("image_crop_quadrant tool not found")
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties should be a map")
	}

	regionProp, ok := props["region"].(map[string]interface{})
	if !ok {
		t.Fatal("region property should exist and be a map")
	}

	enum, ok := regionProp["enum"].([]string)
	if !ok {
		t.Fatal("region should have enum")
	}

	expectedRegions := []string{
		"top-left", "top-right", "bottom-left", "bottom-right",
		"top-half", "bottom-half", "left-half", "right-half", "center",
	}

	enumMap := make(map[string]bool)
	for _, e := range enum {
		enumMap[e] = true
	}

	for _, region := range expectedRegions {
		if !enumMap[region] {
			t.Errorf("Expected region '%s' not in enum", region)
		}
	}
}

func TestToolDefinitions_OptionalDefaults(t *testing.T) {
	tools := GetToolDefinitions()

	// Tools with optional parameters that should have defaults
	toolDefaults := map[string]map[string]interface{}{
		"image_crop":             {"scale": 1.0},
		"image_crop_quadrant":    {"scale": 1.0},
		"image_dominant_colors":  {"count": 5},
		"image_grid_overlay":     {"grid_spacing": 50, "show_coordinates": true, "grid_color": "#FF000080"},
		"image_ocr_full":         {"language": "eng"},
		"image_detect_rectangles": {"min_area": 100, "tolerance": 0.9},
		"image_detect_lines":     {"min_length": 20, "detect_arrows": true},
		"image_detect_circles":   {"min_radius": 5, "max_radius": 500},
		"image_edge_detect":      {"threshold_low": 50, "threshold_high": 150},
		"image_check_alignment":  {"tolerance": 5},
	}

	toolMap := make(map[string]Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	for toolName, expectedDefaults := range toolDefaults {
		tool, ok := toolMap[toolName]
		if !ok {
			t.Errorf("Tool %s not found", toolName)
			continue
		}

		props, ok := tool.InputSchema["properties"].(map[string]interface{})
		if !ok {
			t.Errorf("%s: properties should be a map", toolName)
			continue
		}

		for paramName, expectedDefault := range expectedDefaults {
			param, ok := props[paramName].(map[string]interface{})
			if !ok {
				t.Errorf("%s.%s: parameter not found or not a map", toolName, paramName)
				continue
			}

			actualDefault, ok := param["default"]
			if !ok {
				t.Errorf("%s.%s: missing default value", toolName, paramName)
				continue
			}

			// Compare defaults (handle type differences)
			switch expected := expectedDefault.(type) {
			case float64:
				actual, ok := actualDefault.(float64)
				if !ok || actual != expected {
					t.Errorf("%s.%s: default got %v, want %v", toolName, paramName, actualDefault, expected)
				}
			case int:
				// JSON numbers are float64
				actual, ok := actualDefault.(int)
				if !ok {
					actualFloat, ok := actualDefault.(float64)
					if !ok || int(actualFloat) != expected {
						t.Errorf("%s.%s: default got %v, want %v", toolName, paramName, actualDefault, expected)
					}
				} else if actual != expected {
					t.Errorf("%s.%s: default got %v, want %v", toolName, paramName, actualDefault, expected)
				}
			case string:
				actual, ok := actualDefault.(string)
				if !ok || actual != expected {
					t.Errorf("%s.%s: default got %v, want %v", toolName, paramName, actualDefault, expected)
				}
			case bool:
				actual, ok := actualDefault.(bool)
				if !ok || actual != expected {
					t.Errorf("%s.%s: default got %v, want %v", toolName, paramName, actualDefault, expected)
				}
			}
		}
	}
}

func TestHandleToolsList(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
	}

	resp := s.handleToolsList(req)

	if resp == nil {
		t.Fatal("handleToolsList returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	tools, ok := result["tools"]
	if !ok {
		t.Fatal("Result should contain 'tools' key")
	}

	toolsList, ok := tools.([]Tool)
	if !ok {
		t.Fatal("tools should be a slice of Tool")
	}

	// Should match GetToolDefinitions
	expected := GetToolDefinitions()
	if len(toolsList) != len(expected) {
		t.Errorf("Tool count: got %d, want %d", len(toolsList), len(expected))
	}
}

func TestToolStruct(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "A test parameter",
				},
			},
			"required": []string{"param1"},
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("Name: got %s, want test_tool", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("Description: got %s, want 'A test tool'", tool.Description)
	}
	if tool.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}
}
