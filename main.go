package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	//"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// urlValues là type alias để làm việc với URL parameters
type urlValues url.Values

// Add method cho urlValues
func (uv urlValues) Add(key, value string) {
	url.Values(uv).Add(key, value)
}

// Encode method cho urlValues
func (uv urlValues) Encode() string {
	return url.Values(uv).Encode()
}

// makeHandler creates a server.ToolHandlerFunc that performs an HTTP request to the specified urlPath using the given method.
// It supports path parameter substitution, query parameters, and request body serialization (JSON).
// Authentication headers (Bearer token or API key) are automatically added if requiresAuth is true and environment variables are set.
// The handler validates the HTTP method, constructs the request, sets appropriate headers, and returns the API response as text.
// Errors are returned as mcp.CallToolResult with descriptive messages.
//
// Parameters:
//   - method: The HTTP method to use (e.g., "GET", "POST", "PUT", etc.).
//   - urlPath: The URL path, which may contain path parameters in the form {param}.
//   - requiresAuth: Whether authentication headers should be added.
//
// Returns:
//   - server.ToolHandlerFunc: A function that handles MCP tool requests and returns the API response or error.
func makeHandler(method, urlPath string, requiresAuth bool) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Validate input method
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "HEAD": true, "OPTIONS": true,
		}

		method = strings.ToUpper(method)
		if !validMethods[method] {
			return mcp.NewToolResultError(fmt.Sprintf("INVALID_METHOD HTTP method không hợp lệ: %s", method)), nil
		}

		// Lấy arguments từ request
		// Theo MCP protocol, arguments nằm trực tiếp trong request
		var args map[string]interface{}
		if request.Params.Arguments != nil {
			args = request.GetArguments()
		} else {
			args = make(map[string]interface{})
		}

		// Thay thế path parameters {id} → giá trị thực
		finalURL := urlPath
		for k, v := range args {
			placeholder := "{" + k + "}"
			if strings.Contains(finalURL, placeholder) {
				// URL encode giá trị để tránh lỗi với ký tự đặc biệt
				finalURL = strings.ReplaceAll(finalURL, placeholder, url.PathEscape(fmt.Sprintf("%v", v)))
				delete(args, k)
			}
		}

		var httpReq *http.Request
		var err error

		switch method {
		case "GET", "DELETE", "HEAD", "OPTIONS":
			// Xử lý query parameters
			q := make(urlValues)
			for k, v := range args {
				if k != "body" { // Bỏ qua body parameter cho GET/DELETE
					q.Add(k, fmt.Sprintf("%v", v))
				}
			}
			if len(q) > 0 {
				finalURL += "?" + q.Encode()
			}
			httpReq, err = http.NewRequestWithContext(ctx, method, finalURL, nil)

		case "POST", "PUT", "PATCH":
			var body io.Reader

			// Kiểm tra xem có body parameter không
			if bodyValue, exists := args["body"]; exists {
				if bodyStr, ok := bodyValue.(string); ok && bodyStr != "" {
					body = strings.NewReader(bodyStr)
				} else {
					// Nếu body không phải string, convert thành JSON
					jsonData, jsonErr := json.Marshal(bodyValue)
					if jsonErr != nil {
						return mcp.NewToolResultError(fmt.Sprintf("INVALID_BODY Không thể serialize body: %v", jsonErr)), nil
					}
					body = strings.NewReader(string(jsonData))
				}
			} else {
				// Nếu không có body parameter, tạo JSON từ các args còn lại
				if len(args) > 0 {
					jsonData, jsonErr := json.Marshal(args)
					if jsonErr != nil {
						return mcp.NewToolResultError(fmt.Sprintf("INVALID_ARGS Không thể serialize arguments: %v", jsonErr)), nil
					}
					body = strings.NewReader(string(jsonData))
				} else {
					body = strings.NewReader("{}")
				}
			}

			httpReq, err = http.NewRequestWithContext(ctx, method, finalURL, body)
			if err == nil {
				httpReq.Header.Set("Content-Type", "application/json")
			}

		default:
			return mcp.NewToolResultError(fmt.Sprintf("UNSUPPORTED_METHOD HTTP method không được hỗ trợ: %s", method)), nil
		}

		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("REQUEST_CREATE_FAILED Lỗi tạo HTTP request: %v", err)), nil
		}

		// Xử lý authentication
		if requiresAuth {
			authHeaderSet := false

			// Ưu tiên Bearer token
			if token := strings.TrimSpace(os.Getenv("BEARER_TOKEN")); token != "" {
				httpReq.Header.Set("Authorization", "Bearer "+token)
				authHeaderSet = true
			}

			// API Key làm header bổ sung hoặc chính
			if apiKey := strings.TrimSpace(os.Getenv("API_KEY")); apiKey != "" {
				httpReq.Header.Set("X-API-Key", apiKey)
				authHeaderSet = true
			}

			// Kiểm tra xem có credential nào được cung cấp không
			if !authHeaderSet {
				return mcp.NewToolResultError("AUTH_REQUIRED Yêu cầu authentication nhưng không tìm thấy BEARER_TOKEN hoặc API_KEY"), nil
			}
		}

		// Thực hiện HTTP request với timeout
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("REQUEST_FAILED Lỗi khi gọi API: %v", err)), nil
		}
		defer resp.Body.Close()

		// Đọc response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("RESPONSE_READ_FAILED Lỗi đọc response: %v", err)), nil
		}

		// Kiểm tra HTTP status code
		if resp.StatusCode >= 400 {
			return mcp.NewToolResultError(fmt.Sprintf("HTTP_ERROR API trả về lỗi %d: %s", resp.StatusCode, string(body))), nil
		}

		// Trả về kết quả thành công
		return mcp.NewToolResultText(string(body)), nil
	}
}

// OpenAPI structures (simplified for common cases)
type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi"`
	Info       OpenAPIInfo         `json:"info"`
	Servers    []OpenAPIServer     `json:"servers"`
	Paths      map[string]PathItem `json:"paths"`
	Components *Components         `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	OperationID string                 `json:"operationId,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]interface{} `json:"responses,omitempty"`
	Security    []map[string][]string  `json:"security,omitempty"`
}

type Parameter struct {
	Name        string                 `json:"name"`
	In          string                 `json:"in"` // path, query, header
	Required    bool                   `json:"required,omitempty"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
}

type RequestBody struct {
	Required bool                              `json:"required,omitempty"`
	Content  map[string]map[string]interface{} `json:"content,omitempty"`
}

type Components struct {
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme,omitempty"`
	Name   string `json:"name,omitempty"`
	In     string `json:"in,omitempty"`
}

// GenerateToolsFromOpenAPI tạo tools từ OpenAPI spec
func GenerateToolsFromOpenAPI(specURLOrPath string) ([]*server.ServerTool, error) {
	// Load OpenAPI spec
	spec, err := loadOpenAPISpec(specURLOrPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	var tools []*server.ServerTool
	baseURL := getBaseURL(spec)

	// Iterate through all paths and operations
	for path, pathItem := range spec.Paths {
		operations := map[string]*Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"DELETE": pathItem.Delete,
			"PATCH":  pathItem.Patch,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			tool := createToolFromOperation(method, path, operation, baseURL, spec)
			tools = append(tools, tool)
		}
	}

	return tools, nil
}

// loadOpenAPISpec load spec từ URL hoặc file path
func loadOpenAPISpec(specURLOrPath string) (*OpenAPISpec, error) {
	var data []byte
	var err error

	if strings.HasPrefix(specURLOrPath, "http://") || strings.HasPrefix(specURLOrPath, "https://") {
		// Load from URL
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(specURLOrPath)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		// Load from file
		data, err = os.ReadFile(specURLOrPath)
		if err != nil {
			return nil, err
		}
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return &spec, nil
}

// getBaseURL lấy base URL từ servers
func getBaseURL(spec *OpenAPISpec) string {
	if len(spec.Servers) > 0 {
		return strings.TrimSuffix(spec.Servers[0].URL, "/")
	}
	return ""
}

// createToolFromOperation tạo MCP tool từ OpenAPI operation
func createToolFromOperation(method, path string, operation *Operation, baseURL string, spec *OpenAPISpec) *server.ServerTool {
	// Tạo tool name từ operationId hoặc method + path
	toolName := operation.OperationID
	if toolName == "" {
		toolName = generateToolName(method, path)
	}

	// Tạo description
	description := operation.Summary
	if description == "" {
		description = operation.Description
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", method, path)
	}

	// Determine if authentication is required
	requiresAuth := hasAuthentication(operation, spec)

	// Tạo full URL
	fullURL := baseURL + path

	// Tạo input schema từ parameters
	inputSchema := createInputSchema(operation)

	tool := mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: inputSchema,
	}

	handler := makeHandler(method, fullURL, requiresAuth)

	return &server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// generateToolName tạo tên tool từ method và path
func generateToolName(method, path string) string {
	// Remove path parameters and convert to snake_case
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	var cleanParts []string

	for _, part := range pathParts {
		// Skip path parameters like {id}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}

		// Convert kebab-case to snake_case
		part = strings.ReplaceAll(part, "-", "_")
		cleanParts = append(cleanParts, part)
	}

	methodLower := strings.ToLower(method)
	if len(cleanParts) == 0 {
		return methodLower + "_root"
	}

	return methodLower + "_" + strings.Join(cleanParts, "_")
}

// createInputSchema tạo JSON schema từ parameters
func createInputSchema(operation *Operation) mcp.ToolInputSchema {
	properties := make(map[string]interface{})
	var required []string

	// Add path parameters
	for _, param := range operation.Parameters {
		if param.In == "path" {
			properties[param.Name] = map[string]interface{}{
				"type":        getSchemaType(param.Schema),
				"description": param.Description,
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}
	}

	// Add query parameters
	for _, param := range operation.Parameters {
		if param.In == "query" {
			properties[param.Name] = map[string]interface{}{
				"type":        getSchemaType(param.Schema),
				"description": param.Description,
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}
	}

	// Add request body for POST/PUT/PATCH
	if operation.RequestBody != nil {
		properties["body"] = map[string]interface{}{
			"type":        "string",
			"description": "Request body (JSON string)",
		}
		if operation.RequestBody.Required {
			required = append(required, "body")
		}
	}

	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// getSchemaType lấy type từ schema
func getSchemaType(schema map[string]interface{}) string {
	if schema == nil {
		return "string"
	}

	if schemaType, exists := schema["type"]; exists {
		if typeStr, ok := schemaType.(string); ok {
			return typeStr
		}
	}

	return "string"
}

// hasAuthentication kiểm tra xem operation có cần auth không
func hasAuthentication(operation *Operation, spec *OpenAPISpec) bool {
	// Check operation-level security
	if len(operation.Security) > 0 {
		return true
	}

	// Check if there are any security schemes defined
	if spec.Components != nil && len(spec.Components.SecuritySchemes) > 0 {
		return true
	}

	return false
}

// Main function example
func main() {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"openapi-client",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Load initial OpenAPI spec
	specPath := "C:/Users/Lenovo/Desktop/open_api_to_mcp_server/openapi.json"
	fmt.Println("🔄 Loading OpenAPI specification...")
	tools, err := GenerateToolsFromOpenAPI(specPath)
	if err != nil {
		fmt.Printf("❌ Error generating tools: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Generated %d tools from OpenAPI spec\n", len(tools))
	// Maintain a local map of tool handlers for HTTP access
	toolHandlers := make(map[string]server.ToolHandlerFunc)
	for _, tool := range tools {
		mcpServer.AddTool(tool.Tool, tool.Handler)
		toolHandlers[tool.Tool.Name] = tool.Handler
		fmt.Printf("📋 Added tool: %s - %s\n", tool.Tool.Name, tool.Tool.Description)
	}

	// HTTP server for MCP and OpenAPI upload
	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Simple MCP HTTP handler: expects JSON body with MCP request
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req mcp.CallToolRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "Invalid MCP request", http.StatusBadRequest)
			return
		}
		// Use req.Params.Name for tool name
		toolName := req.Params.Name
		handler, ok := toolHandlers[toolName]
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
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
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
		// Save uploaded spec to disk
		err = os.WriteFile(specPath, data, 0644)
		if err != nil {
			http.Error(w, "Failed to save spec file", http.StatusInternalServerError)
			return
		}
		// Regenerate tools
		newTools, err := GenerateToolsFromOpenAPI(specPath)
		if err != nil {
			http.Error(w, "Failed to parse OpenAPI spec", http.StatusBadRequest)
			return
		}
		// Remove all tools by reinitializing the server (workaround)
		mcpServer = server.NewMCPServer(
			"openapi-client",
			"1.0.0",
			server.WithToolCapabilities(true),
		)
		toolHandlers = make(map[string]server.ToolHandlerFunc)
		for _, tool := range newTools {
			mcpServer.AddTool(tool.Tool, tool.Handler)
			toolHandlers[tool.Tool.Name] = tool.Handler
		}
		w.Write([]byte("Spec uploaded and tools updated successfully."))
	})

	// Start MCP server via stdio in a goroutine
	go func() {
		fmt.Println("🚀 Starting MCP server via stdio ...")
		if err := server.ServeStdio(mcpServer); err != nil {
			fmt.Printf("❌ Stdio server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Start MCP HTTP server
	fmt.Println("🚀 Starting MCP HTTP server on :8080 ...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("❌ HTTP server error: %v\n", err)
		os.Exit(1)
	}
}
