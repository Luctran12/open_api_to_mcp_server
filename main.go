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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultTimeout     = 30 * time.Second
	defaultServerPort  = ":8080"
	envBearerToken     = "BEARER_TOKEN"
	envAPIKey          = "API_KEY"
	contentTypeJSON    = "application/json"
	headerAuthorization = "Authorization"
	headerAPIKey       = "X-API-Key"
	headerContentType  = "Content-Type"
)

var validHTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
}

type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers"`
	Paths      map[string]PathItem    `json:"paths"`
	Components *Components            `json:"components,omitempty"`
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
	In          string                 `json:"in"`
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

type HTTPRequestBuilder struct {
	method       string
	url          string
	args         map[string]interface{}
	requiresAuth bool
}

func NewHTTPRequestBuilder(method, urlPath string, requiresAuth bool) *HTTPRequestBuilder {
	return &HTTPRequestBuilder{
		method:       strings.ToUpper(method),
		url:          urlPath,
		requiresAuth: requiresAuth,
	}
}

func (b *HTTPRequestBuilder) WithArguments(args map[string]interface{}) *HTTPRequestBuilder {
	b.args = args
	return b
}

func (b *HTTPRequestBuilder) Build(ctx context.Context) (*http.Request, error) {
	if !validHTTPMethods[b.method] {
		return nil, fmt.Errorf("invalid HTTP method: %s", b.method)
	}

	finalURL := b.substitutePathParams()
	req, err := b.createHTTPRequest(ctx, finalURL)
	if err != nil {
		return nil, err
	}

	if b.requiresAuth {
		if err := b.addAuthHeaders(req); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (b *HTTPRequestBuilder) substitutePathParams() string {
	finalURL := b.url
	for key, value := range b.args {
		placeholder := "{" + key + "}"
		if strings.Contains(finalURL, placeholder) {
			finalURL = strings.ReplaceAll(finalURL, placeholder, url.PathEscape(fmt.Sprintf("%v", value)))
			delete(b.args, key)
		}
	}
	return finalURL
}

func (b *HTTPRequestBuilder) createHTTPRequest(ctx context.Context, finalURL string) (*http.Request, error) {
	switch b.method {
	case "GET", "DELETE", "HEAD", "OPTIONS":
		return b.createRequestWithQuery(ctx, finalURL)
	case "POST", "PUT", "PATCH":
		return b.createRequestWithBody(ctx, finalURL)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", b.method)
	}
}

func (b *HTTPRequestBuilder) createRequestWithQuery(ctx context.Context, baseURL string) (*http.Request, error) {
	query := url.Values{}
	for key, value := range b.args {
		if key != "body" {
			query.Add(key, fmt.Sprintf("%v", value))
		}
	}

	finalURL := baseURL
	if len(query) > 0 {
		finalURL += "?" + query.Encode()
	}

	return http.NewRequestWithContext(ctx, b.method, finalURL, nil)
}

func (b *HTTPRequestBuilder) createRequestWithBody(ctx context.Context, finalURL string) (*http.Request, error) {
	body := b.buildRequestBody()
	req, err := http.NewRequestWithContext(ctx, b.method, finalURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerContentType, contentTypeJSON)
	return req, nil
}

func (b *HTTPRequestBuilder) buildRequestBody() io.Reader {
	if bodyValue, exists := b.args["body"]; exists {
		if bodyStr, ok := bodyValue.(string); ok && bodyStr != "" {
			return strings.NewReader(bodyStr)
		}
		if jsonData, err := json.Marshal(bodyValue); err == nil {
			return strings.NewReader(string(jsonData))
		}
	}

	if len(b.args) > 0 {
		if jsonData, err := json.Marshal(b.args); err == nil {
			return strings.NewReader(string(jsonData))
		}
	}

	return strings.NewReader("{}")
}

func (b *HTTPRequestBuilder) addAuthHeaders(req *http.Request) error {
	authHeaderSet := false

	if token := strings.TrimSpace(os.Getenv(envBearerToken)); token != "" {
		req.Header.Set(headerAuthorization, "Bearer "+token)
		authHeaderSet = true
	}

	if apiKey := strings.TrimSpace(os.Getenv(envAPIKey)); apiKey != "" {
		req.Header.Set(headerAPIKey, apiKey)
		authHeaderSet = true
	}

	if !authHeaderSet {
		return fmt.Errorf("authentication required but no credentials found")
	}

	return nil
}

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *HTTPClient) Execute(req *http.Request) (string, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

type ToolHandler struct {
	builder *HTTPRequestBuilder
	client  *HTTPClient
}

func NewToolHandler(method, urlPath string, requiresAuth bool) *ToolHandler {
	return &ToolHandler{
		builder: NewHTTPRequestBuilder(method, urlPath, requiresAuth),
		client:  NewHTTPClient(),
	}
}

func (h *ToolHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := h.extractArguments(request)
	
	req, err := h.builder.WithArguments(args).Build(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := h.client.Execute(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *ToolHandler) extractArguments(request mcp.CallToolRequest) map[string]interface{} {
	if request.Params.Arguments != nil {
		return request.GetArguments()
	}
	return make(map[string]interface{})
}

type OpenAPILoader struct {
	httpClient *http.Client
}

func NewOpenAPILoader() *OpenAPILoader {
	return &OpenAPILoader{
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (l *OpenAPILoader) Load(specURLOrPath string) (*OpenAPISpec, error) {
	data, err := l.readSpec(specURLOrPath)
	if err != nil {
		return nil, err
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return &spec, nil
}

func (l *OpenAPILoader) readSpec(specURLOrPath string) ([]byte, error) {
	if strings.HasPrefix(specURLOrPath, "http://") || strings.HasPrefix(specURLOrPath, "https://") {
		return l.readFromURL(specURLOrPath)
	}
	return os.ReadFile(specURLOrPath)
}

func (l *OpenAPILoader) readFromURL(url string) ([]byte, error) {
	resp, err := l.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

type ToolGenerator struct {
	spec *OpenAPISpec
}

func NewToolGenerator(spec *OpenAPISpec) *ToolGenerator {
	return &ToolGenerator{spec: spec}
}

func (g *ToolGenerator) Generate() []*server.ServerTool {
	var tools []*server.ServerTool
	baseURL := g.getBaseURL()

	for path, pathItem := range g.spec.Paths {
		tools = append(tools, g.generateToolsForPath(path, pathItem, baseURL)...)
	}

	return tools
}

func (g *ToolGenerator) generateToolsForPath(path string, pathItem PathItem, baseURL string) []*server.ServerTool {
	var tools []*server.ServerTool
	operations := map[string]*Operation{
		"GET": pathItem.Get, "POST": pathItem.Post, "PUT": pathItem.Put,
		"DELETE": pathItem.Delete, "PATCH": pathItem.Patch,
	}

	for method, operation := range operations {
		if operation != nil {
			tool := g.createTool(method, path, operation, baseURL)
			tools = append(tools, tool)
		}
	}

	return tools
}

func (g *ToolGenerator) createTool(method, path string, operation *Operation, baseURL string) *server.ServerTool {
	toolName := g.generateToolName(method, path, operation)
	description := g.generateDescription(method, path, operation)
	requiresAuth := g.requiresAuthentication(operation)
	fullURL := baseURL + path
	inputSchema := g.createInputSchema(operation)

	tool := mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: inputSchema,
	}

	handler := NewToolHandler(method, fullURL, requiresAuth)

	return &server.ServerTool{
		Tool:    tool,
		Handler: handler.Handle,
	}
}

func (g *ToolGenerator) generateToolName(method, path string, operation *Operation) string {
	if operation.OperationID != "" {
		return operation.OperationID
	}

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	var cleanParts []string

	for _, part := range pathParts {
		if !strings.HasPrefix(part, "{") || !strings.HasSuffix(part, "}") {
			part = strings.ReplaceAll(part, "-", "_")
			cleanParts = append(cleanParts, part)
		}
	}

	methodLower := strings.ToLower(method)
	if len(cleanParts) == 0 {
		return methodLower + "_root"
	}

	return methodLower + "_" + strings.Join(cleanParts, "_")
}

func (g *ToolGenerator) generateDescription(method, path string, operation *Operation) string {
	if operation.Summary != "" {
		return operation.Summary
	}
	if operation.Description != "" {
		return operation.Description
	}
	return fmt.Sprintf("%s %s", method, path)
}

func (g *ToolGenerator) requiresAuthentication(operation *Operation) bool {
	if len(operation.Security) > 0 {
		return true
	}
	if g.spec.Components != nil && len(g.spec.Components.SecuritySchemes) > 0 {
		return true
	}
	return false
}

func (g *ToolGenerator) createInputSchema(operation *Operation) mcp.ToolInputSchema {
	properties := make(map[string]interface{})
	var required []string

	for _, param := range operation.Parameters {
		if param.In == "path" || param.In == "query" {
			properties[param.Name] = map[string]interface{}{
				"type":        g.getSchemaType(param.Schema),
				"description": param.Description,
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}
	}

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

func (g *ToolGenerator) getSchemaType(schema map[string]interface{}) string {
	if schema != nil {
		if schemaType, exists := schema["type"]; exists {
			if typeStr, ok := schemaType.(string); ok {
				return typeStr
			}
		}
	}
	return "string"
}

func (g *ToolGenerator) getBaseURL() string {
	if len(g.spec.Servers) > 0 {
		return strings.TrimSuffix(g.spec.Servers[0].URL, "/")
	}
	return ""
}

type MCPServerManager struct {
	server       *server.MCPServer
	toolHandlers map[string]server.ToolHandlerFunc
	specPath     string
}

func NewMCPServerManager(specPath string) *MCPServerManager {
	return &MCPServerManager{
		server:       g.createMCPServer(),
		toolHandlers: make(map[string]server.ToolHandlerFunc),
		specPath:     specPath,
	}
}

func (m *MCPServerManager) createMCPServer() *server.MCPServer {
	return server.NewMCPServer(
		"openapi-client",
		"1.0.0",
		server.WithToolCapabilities(true),
	)
}

func (m *MCPServerManager) LoadTools() error {
	loader := NewOpenAPILoader()
	spec, err := loader.Load(m.specPath)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	generator := NewToolGenerator(spec)
	tools := generator.Generate()

	for _, tool := range tools {
		m.server.AddTool(tool.Tool, tool.Handler)
		m.toolHandlers[tool.Tool.Name] = tool.Handler
		fmt.Printf("📋 Added tool: %s - %s\n", tool.Tool.Name, tool.Tool.Description)
	}

	fmt.Printf("✅ Generated %d tools from OpenAPI spec\n", len(tools))
	return nil
}

func (m *MCPServerManager) ReloadTools(newSpecData []byte) error {
	if err := os.WriteFile(m.specPath, newSpecData, 0644); err != nil {
		return fmt.Errorf("failed to save spec file: %w", err)
	}

	m.server = m.createMCPServer()
	m.toolHandlers = make(map[string]server.ToolHandlerFunc)

	return m.LoadTools()
}

func (m *MCPServerManager) HandleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mcp.CallToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid MCP request", http.StatusBadRequest)
		return
	}

	handler, ok := m.toolHandlers[req.Params.Name]
	if !ok {
		http.Error(w, "Tool not found", http.StatusNotFound)
		return
	}

	result, err := handler(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(result)
}

func (m *MCPServerManager) HandleUpload(w http.ResponseWriter, r *http.Request) {
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

	if err := m.ReloadTools(data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Write([]byte("Spec uploaded and tools updated successfully."))
}

func (m *MCPServerManager) ServeStdio() error {
	return server.ServeStdio(m.server)
}

func main() {
	specPath := "C:/Users/Lenovo/Desktop/open_api_to_mcp_server/openapi.json"
	
	manager := NewMCPServerManager(specPath)

	fmt.Println("🔄 Loading OpenAPI specification...")
	if err := manager.LoadTools(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/mcp", manager.HandleMCPRequest)
	http.HandleFunc("/upload", manager.HandleUpload)

	go func() {
		fmt.Println("🚀 Starting MCP server via stdio...")
		if err := manager.ServeStdio(); err != nil {
			fmt.Printf("❌ Stdio server error: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Println("🚀 Starting MCP HTTP server on :8080...")
	if err := http.ListenAndServe(defaultServerPort, nil); err != nil {
		fmt.Printf("❌ HTTP server error: %v\n", err)
		os.Exit(1)
	}
}