package main

import (
	"fmt"
	"os"

	"open_api_to_mcp_server/internal/config"
	"open_api_to_mcp_server/internal/server"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to create server: %v\n", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		fmt.Printf("❌ Server error: %v\n", err)
		os.Exit(1)
	}
}
