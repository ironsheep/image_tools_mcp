package server

import (
	"encoding/json"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.cache == nil {
		t.Fatal("New() did not initialize cache")
	}
}

func TestMCPRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantID  interface{}
		wantMethod string
	}{
		{
			"string id",
			`{"jsonrpc":"2.0","id":"test-1","method":"tools/list"}`,
			"test-1",
			"tools/list",
		},
		{
			"number id",
			`{"jsonrpc":"2.0","id":42,"method":"ping"}`,
			float64(42), // JSON numbers decode as float64
			"ping",
		},
		{
			"null id",
			`{"jsonrpc":"2.0","id":null,"method":"initialize"}`,
			nil,
			"initialize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req MCPRequest
			if err := json.Unmarshal([]byte(tt.json), &req); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if req.ID != tt.wantID {
				t.Errorf("ID: got %v (%T), want %v (%T)", req.ID, req.ID, tt.wantID, tt.wantID)
			}
			if req.Method != tt.wantMethod {
				t.Errorf("Method: got %s, want %s", req.Method, tt.wantMethod)
			}
			if req.JSONRPC != "2.0" {
				t.Errorf("JSONRPC: got %s, want 2.0", req.JSONRPC)
			}
		})
	}
}

func TestMCPRequest_WithParams(t *testing.T) {
	jsonStr := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"image_load","arguments":{"path":"/test.png"}}}`

	var req MCPRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Params == nil {
		t.Error("Params should not be nil")
	}

	// Verify params can be parsed
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("Failed to unmarshal params: %v", err)
	}

	if params["name"] != "image_load" {
		t.Errorf("params[name]: got %v, want image_load", params["name"])
	}
}

func TestMCPResponse_Marshal(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"status": "ok"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify roundtrip
	var decoded MCPResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %s, want 2.0", decoded.JSONRPC)
	}
}

func TestMCPResponse_WithError(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &MCPError{
			Code:    -32601,
			Message: "Method not found",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MCPResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if decoded.Error.Code != -32601 {
		t.Errorf("Error.Code: got %d, want -32601", decoded.Error.Code)
	}
}

func TestMCPError_Marshal(t *testing.T) {
	mcpErr := MCPError{
		Code:    -32000,
		Message: "Tool execution failed",
		Data:    map[string]string{"details": "file not found"},
	}

	data, err := json.Marshal(mcpErr)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MCPError
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Code != -32000 {
		t.Errorf("Code: got %d, want -32000", decoded.Code)
	}
	if decoded.Message != "Tool execution failed" {
		t.Errorf("Message: got %s, want 'Tool execution failed'", decoded.Message)
	}
}

func TestHandleRequest_Initialize(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("handleRequest returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
	if resp.ID != 1 {
		t.Errorf("ID: got %v, want 1", resp.ID)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion: got %v", result["protocolVersion"])
	}
}

func TestHandleRequest_Ping(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "ping-1",
		Method:  "ping",
	}

	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("handleRequest returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
	if resp.ID != "ping-1" {
		t.Errorf("ID: got %v, want ping-1", resp.ID)
	}
}

func TestHandleRequest_ToolsList(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("handleRequest returned nil")
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

	// Should have multiple tools defined
	if len(toolsList) < 10 {
		t.Errorf("Expected at least 10 tools, got %d", len(toolsList))
	}
}

func TestHandleRequest_NotificationsInitialized(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	resp := s.handleRequest(req)

	// Notifications don't get responses
	if resp != nil {
		t.Error("notifications/initialized should return nil response")
	}
}

func TestHandleRequest_MethodNotFound(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "nonexistent/method",
	}

	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("handleRequest returned nil")
	}
	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Error code: got %d, want -32601", resp.Error.Code)
	}
}

func TestHandleInitialize(t *testing.T) {
	s := New()
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "init-1",
	}

	resp := s.handleInitialize(req)

	if resp.ID != "init-1" {
		t.Errorf("ID: got %v, want init-1", resp.ID)
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %s, want 2.0", resp.JSONRPC)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("serverInfo should be a map")
	}

	if serverInfo["name"] != "image-tools-mcp" {
		t.Errorf("serverInfo.name: got %v", serverInfo["name"])
	}
	if serverInfo["version"] != "0.1.0" {
		t.Errorf("serverInfo.version: got %v", serverInfo["version"])
	}
}

func TestMCPNotification_Marshal(t *testing.T) {
	notification := MCPNotification{
		JSONRPC: "2.0",
		Method:  "test/notification",
		Params:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MCPNotification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Method != "test/notification" {
		t.Errorf("Method: got %s, want test/notification", decoded.Method)
	}
}
