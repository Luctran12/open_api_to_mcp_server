package server

import (
	"context"
	"fmt"

	"open_api_to_mcp_server/internal/config"
	"open_api_to_mcp_server/internal/handler"
	"open_api_to_mcp_server/internal/openapi"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServer wraps the MCP server with tool management
type MCPServer struct {
	server       *server.MCPServer
	httpHandler  *handler.HTTPHandler
	toolHandlers map[string]server.ToolHandlerFunc
	config       *config.Config
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(cfg *config.Config) *MCPServer {
	mcpServer := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
		server.WithToolCapabilities(true),
	)

	return &MCPServer{
		server:       mcpServer,
		httpHandler:  handler.NewHTTPHandler(&cfg.Auth),
		toolHandlers: make(map[string]server.ToolHandlerFunc),
		config:       cfg,
	}
}

// LoadTools loads tools from an OpenAPI specification
func (s *MCPServer) LoadTools(specPath string) error {
	fmt.Println("🔄 Loading OpenAPI specification...")

	loader := openapi.NewLoader()
	spec, err := loader.Load(specPath)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	generator := openapi.NewGenerator()
	toolDefs, err := generator.Generate(spec)
	if err != nil {
		return fmt.Errorf("failed to generate tools: %w", err)
	}

	fmt.Printf("✅ Generated %d tools from OpenAPI spec\n", len(toolDefs))

	for _, toolDef := range toolDefs {
		s.addTool(toolDef)
		fmt.Printf("📋 Added tool: %s - %s\n", toolDef.Name, toolDef.Description)
	}

	return nil
}

func (s *MCPServer) addTool(toolDef *openapi.ToolDefinition) {
	tool := mcp.Tool{
		Name:        toolDef.Name,
		Description: toolDef.Description,
		InputSchema: toolDef.InputSchema,
	}

	handlerFunc := s.createHandler(toolDef)
	s.server.AddTool(tool, handlerFunc)
	s.toolHandlers[toolDef.Name] = handlerFunc
}

func (s *MCPServer) createHandler(toolDef *openapi.ToolDefinition) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		if args == nil {
			args = make(map[string]interface{})
		}

		return s.httpHandler.Execute(ctx, toolDef.Method, toolDef.URL, toolDef.RequiresAuth, args)
	}
}

// GetToolHandler returns a handler for a specific tool
func (s *MCPServer) GetToolHandler(name string) (server.ToolHandlerFunc, bool) {
	handler, ok := s.toolHandlers[name]
	return handler, ok
}

// GetServer returns the underlying MCP server
func (s *MCPServer) GetServer() *server.MCPServer {
	return s.server
}

// ReloadTools clears and reloads all tools
func (s *MCPServer) ReloadTools(specPath string) error {
	// Create new server instance to clear tools
	s.server = server.NewMCPServer(
		s.config.Server.Name,
		s.config.Server.Version,
		server.WithToolCapabilities(true),
	)
	s.toolHandlers = make(map[string]server.ToolHandlerFunc)

	return s.LoadTools(specPath)
}
