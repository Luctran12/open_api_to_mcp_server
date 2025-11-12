package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/internal/handler"
	"open_api_to_mcp_server/internal/middleware"
	"open_api_to_mcp_server/pkg/config"
	"os"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/swaggo/http-swagger"
	_ "open_api_to_mcp_server/docs"
)

// @title MCP Server API
// @version 1.0
// @description Simple HTTP server exposing MCP endpoints
// @BasePath /
// @host localhost:8081

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description Provide your API key here to access the endpoints

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your Bearer token in the format: Bearer <token>
// HTTPServer handles HTTP endpoints for MCP and spec upload
type HTTPServer struct {
	mcpServer      *MCPServer
	config         *config.Config
	specPath       string
	executeHandler *handler.ExecuteHandler
	authHandler    *handler.AuthHandler
	httpHandler    *handler.HTTPHandler
	specHandler    *handler.SpecHandler
	toolHandler    *handler.ToolHandler
	db             *database.DB
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(mcpServer *MCPServer, cfg *config.Config, executeHandler *handler.ExecuteHandler, authHandler *handler.AuthHandler,
	httpHandler *handler.HTTPHandler,
	specHandler *handler.SpecHandler,
	toolHandler *handler.ToolHandler,
	db *database.DB) *HTTPServer {
	return &HTTPServer{
		mcpServer:      mcpServer,
		config:         cfg,
		specPath:       cfg.OpenAPI.SpecPath,
		executeHandler: executeHandler,
		authHandler:    authHandler,
		httpHandler:    httpHandler,
		specHandler:    specHandler,
		toolHandler:    toolHandler,
		db:             db,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	//init HTTP routes
	mux := http.NewServeMux()
	
	//init middlewares
	authMiddleware := middleware.Authenticate(s.db, s.authHandler.SecretKey)
	
	//chaining middlewares
	chainMiddleware := middleware.ChainMiddleware(mux,
		middleware.CORS,
		middleware.Logging,
	)
	

	// add swagger route
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	mux.Handle("/api/register", http.HandlerFunc(s.authHandler.Register))
	mux.Handle("/api/login", http.HandlerFunc(s.authHandler.Login))
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.Handle("/api/execute", authMiddleware(http.HandlerFunc(s.executeHandler.Execute)))
	mux.Handle("/upload", authMiddleware(http.HandlerFunc(s.specHandler.UploadSpec)))
	mux.Handle("/api/specs", authMiddleware(http.HandlerFunc(s.specHandler.ListSpecs)))
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/api/build", authMiddleware(http.HandlerFunc(s.executeHandler.Build)))
	mux.Handle("/api/build/download/", http.HandlerFunc(s.executeHandler.Download))

	fmt.Printf("🚀 Starting HTTP server on %s ...\n", s.config.Server.HTTPPort)
	return http.ListenAndServe(s.config.Server.HTTPPort, chainMiddleware)
}

// handleMCP godoc
// @Summary Handle MCP tool call
// @Description Process MCP CallToolRequest and route to correct handler
// @Accept json
// @Produce json
// @Param request body server.CallToolRequestExample true "MCP Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Tool not found"
// @Router /mcp [post]
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

// handleUpload godoc
// @Summary Upload OpenAPI spec
// @Description Upload new OpenAPI spec file and reload tools
// @Accept multipart/form-data
// @Param spec formData file true "OpenAPI spec file"
// @Success 200 {string} string "Upload successful"
// @Failure 400 {string} string "Bad request"
// @Router /upload [post]
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

// handleHealth godoc
// @Summary Health check
// @Description Return server status and version
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"server": s.config.Server.Name,
		"version": s.config.Server.Version,
	})
}
