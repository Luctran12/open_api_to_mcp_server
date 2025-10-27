package server

import (
	"fmt"
	"os"

	"open_api_to_mcp_server/pkg/config"

	"github.com/mark3labs/mcp-go/server"
)

// Server manages both MCP and HTTP servers
type Server struct {
	mcpServer  *MCPServer
	httpServer *HTTPServer
	config     *config.Config
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	mcpServer := NewMCPServer(cfg)

	// Load initial tools
	if err := mcpServer.LoadTools(cfg.OpenAPI.SpecPath); err != nil {
		return nil, fmt.Errorf("failed to load initial tools: %w", err)
	}

	httpServer := NewHTTPServer(mcpServer, cfg)

	return &Server{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     cfg,
	}, nil
}

// Start starts both stdio and HTTP servers
func (s *Server) Start() error {
	// Start MCP stdio server in goroutine
	go func() {
		fmt.Println("🚀 Starting MCP server via stdio...")
		if err := server.ServeStdio(s.mcpServer.GetServer()); err != nil {
			fmt.Printf("❌ Stdio server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server (blocking)
	return s.httpServer.Start()
}
