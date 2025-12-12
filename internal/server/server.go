package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ironsheep/image-tools-mcp/internal/imaging"
)

// Server handles MCP protocol communication over stdio.
//
// The server maintains an image cache for efficient repeated access to images
// and processes JSON-RPC requests to execute image analysis tools.
type Server struct {
	cache *imaging.ImageCache
}

// MCPRequest represents an incoming JSON-RPC 2.0 request.
//
// The ID field can be a string, number, or null. Requests without an ID
// are notifications and don't receive responses.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"` // Must be "2.0"
	ID      interface{}     `json:"id"`      // Request identifier (string, number, or null)
	Method  string          `json:"method"`  // Method name to invoke
	Params  json.RawMessage `json:"params,omitempty"` // Method parameters (optional)
}

// MCPResponse represents an outgoing JSON-RPC 2.0 response.
//
// Either Result or Error will be set, never both.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`         // Always "2.0"
	ID      interface{} `json:"id"`              // Matches request ID
	Result  interface{} `json:"result,omitempty"` // Success result (mutually exclusive with Error)
	Error   *MCPError   `json:"error,omitempty"`  // Error details (mutually exclusive with Result)
}

// MCPError represents a JSON-RPC 2.0 error object.
//
// Standard error codes:
//   - -32700: Parse error
//   - -32600: Invalid request
//   - -32601: Method not found
//   - -32602: Invalid params
//   - -32603: Internal error
//   - -32000: Tool execution failure (custom)
type MCPError struct {
	Code    int         `json:"code"`           // Error code (negative for standard errors)
	Message string      `json:"message"`        // Human-readable error message
	Data    interface{} `json:"data,omitempty"` // Additional error information
}

// MCPNotification represents an outgoing JSON-RPC 2.0 notification.
//
// Notifications are messages without an ID that don't expect a response.
// Currently unused but defined for protocol completeness.
type MCPNotification struct {
	JSONRPC string      `json:"jsonrpc"` // Always "2.0"
	Method  string      `json:"method"`  // Notification method name
	Params  interface{} `json:"params,omitempty"` // Notification parameters
}

// New creates and initializes a new MCP server instance.
//
// The server is ready to process requests immediately after creation.
// It maintains an internal image cache that persists for the server's lifetime.
func New() *Server {
	return &Server{
		cache: imaging.NewImageCache(),
	}
}

// Run starts the MCP server's main loop, processing requests from stdin.
//
// The server reads JSON-RPC requests line-by-line from stdin and writes
// responses to stdout. It runs until stdin is closed or an unrecoverable
// error occurs.
//
// The input buffer supports requests up to 1MB in size, accommodating
// large base64-encoded images in responses.
//
// Returns an error only if the scanner encounters an I/O error.
// Individual request parsing or handling errors are logged and don't
// terminate the server.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large requests
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("Failed to parse request: %v", err)
			continue
		}

		resp := s.handleRequest(&req)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				log.Printf("Failed to encode response: %v", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest routes JSON-RPC requests to the appropriate handler method.
//
// Returns nil for notifications that don't require a response.
func (s *Server) handleRequest(req *MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		// Client acknowledgment, no response needed
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// handleInitialize responds to the MCP initialize request with server capabilities.
//
// This is the first request in the MCP handshake, establishing protocol version
// and advertising available capabilities.
func (s *Server) handleInitialize(req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "image-tools-mcp",
				"version": "0.1.0",
			},
		},
	}
}
