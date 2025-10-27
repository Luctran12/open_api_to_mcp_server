package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"open_api_to_mcp_server/pkg/config"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// HTTPServer handles HTTP endpoints for MCP and spec upload
type HTTPServer struct {
	mcpServer *MCPServer
	config    *config.Config
	specPath  string
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(mcpServer *MCPServer, cfg *config.Config) *HTTPServer {
	return &HTTPServer{
		mcpServer: mcpServer,
		config:    cfg,
		specPath:  cfg.OpenAPI.SpecPath,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	http.HandleFunc("/mcp", s.handleMCP)
	http.HandleFunc("/upload", s.handleUpload)
	http.HandleFunc("/health", s.handleHealth)

	fmt.Printf("🚀 Starting HTTP server on %s ...\n", s.config.Server.HTTPPort)
	return http.ListenAndServe(s.config.Server.HTTPPort, nil)
}

func (s *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mcp.CallToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid MCP request", http.StatusBadRequest)
		return
	}

	toolName := req.Params.Name
	handler, ok := s.mcpServer.GetToolHandler(toolName)
	if !ok {
		http.Error(w, "Tool not found", http.StatusNotFound)
		return
	}

	result, err := handler(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *HTTPServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("spec")
	if err != nil {
		http.Error(w, "Failed to read spec file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file data", http.StatusInternalServerError)
		return
	}

	// Save uploaded spec
	if err := os.WriteFile(s.specPath, data, 0644); err != nil {
		http.Error(w, "Failed to save spec file", http.StatusInternalServerError)
		return
	}

	// Reload tools
	if err := s.mcpServer.ReloadTools(s.specPath); err != nil {
		http.Error(w, "Failed to parse OpenAPI spec", http.StatusBadRequest)
		return
	}

	w.Write([]byte("Spec uploaded and tools updated successfully"))
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"server": s.config.Server.Name,
		"version": s.config.Server.Version,
	})
}
