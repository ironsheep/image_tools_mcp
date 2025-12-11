package main

import (
	"log"
	"os"

	"github.com/ironsheep/image-tools-mcp/internal/server"
)

var (
	Version   = "0.1.0"
	BuildTime = "unknown"
)

func main() {
	// Configure logging to stderr (stdout is for MCP protocol)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	logLevel := os.Getenv("IMAGE_MCP_LOG_LEVEL")
	if logLevel == "debug" {
		log.Printf("Image MCP Server v%s (built %s)", Version, BuildTime)
	}

	srv := server.New()
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
