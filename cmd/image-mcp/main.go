package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ironsheep/image-tools-mcp/internal/server"
)

// Version information - set by ldflags during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Handle --version and -v flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("image-tools-mcp %s\n", Version)
			fmt.Printf("  Build time: %s\n", BuildTime)
			fmt.Printf("  Git commit: %s\n", GitCommit)
			return
		case "--help", "-h", "help":
			fmt.Println("image-tools-mcp - MCP server for image analysis")
			fmt.Println()
			fmt.Println("Usage: image-tools-mcp [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --version, -v    Print version information")
			fmt.Println("  --help, -h       Print this help message")
			fmt.Println()
			fmt.Println("Environment variables:")
			fmt.Println("  IMAGE_MCP_LOG_LEVEL=debug    Enable debug logging")
			fmt.Println()
			fmt.Println("This server communicates via MCP protocol over stdin/stdout.")
			fmt.Println("Configure it in your MCP client (e.g., Claude Desktop).")
			return
		}
	}

	// Configure logging to stderr (stdout is for MCP protocol)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	logLevel := os.Getenv("IMAGE_MCP_LOG_LEVEL")
	if logLevel == "debug" {
		log.Printf("Image MCP Server v%s (built %s, commit %s)", Version, BuildTime, GitCommit)
	}

	srv := server.New()
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
