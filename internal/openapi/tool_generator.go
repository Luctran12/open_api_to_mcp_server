package openapi

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolDefinition represents a generated tool from OpenAPI
type ToolDefinition struct {
	Name        string
	Description string
	Method      string
	URL         string
	RequiresAuth bool
	InputSchema mcp.ToolInputSchema
}

// Generator generates MCP tools from OpenAPI specifications
type Generator struct{}

// NewGenerator creates a new tool generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates tool definitions from an OpenAPI spec
func (g *Generator) Generate(spec *Spec) ([]*ToolDefinition, error) {
	var tools []*ToolDefinition
	baseURL := spec.GetBaseURL()

	for path, pathItem := range spec.Paths {
		operations := g.extractOperations(pathItem)

		for method, operation := range operations {
			tool := g.createToolDefinition(method, path, operation, baseURL, spec)
			tools = append(tools, tool)
		}
	}

	return tools, nil
}

func (g *Generator) extractOperations(pathItem PathItem) map[string]*Operation {
	return map[string]*Operation{
		"GET":    pathItem.Get,
		"POST":   pathItem.Post,
		"PUT":    pathItem.Put,
		"DELETE": pathItem.Delete,
		"PATCH":  pathItem.Patch,
	}
}

func (g *Generator) createToolDefinition(method, path string, operation *Operation, baseURL string, spec *Spec) *ToolDefinition {
	toolName := g.generateToolName(method, path, operation)
	description := g.generateDescription(method, path, operation)
	requiresAuth := g.requiresAuthentication(operation, spec)
	fullURL := baseURL + path
	inputSchema := g.createInputSchema(operation)

	return &ToolDefinition{
		Name:        toolName,
		Description: description,
		Method:      method,
		URL:         fullURL,
		RequiresAuth: requiresAuth,
		InputSchema: inputSchema,
	}
}

func (g *Generator) generateToolName(method, path string, operation *Operation) string {
	if operation.OperationID != "" {
		return operation.OperationID
	}

	// Generate from method and path
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	var cleanParts []string

	for _, part := range pathParts {
		// Skip path parameters like {id}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		part = strings.ReplaceAll(part, "-", "_")
		cleanParts = append(cleanParts, part)
	}

	methodLower := strings.ToLower(method)
	if len(cleanParts) == 0 {
		return methodLower + "_root"
	}

	return methodLower + "_" + strings.Join(cleanParts, "_")
}

func (g *Generator) generateDescription(method, path string, operation *Operation) string {
	if operation.Summary != "" {
		return operation.Summary
	}
	if operation.Description != "" {
		return operation.Description
	}
	return fmt.Sprintf("%s %s", method, path)
}

func (g *Generator) requiresAuthentication(operation *Operation, spec *Spec) bool {
	return len(operation.Security) > 0 || spec.HasAuthentication()
}

func (g *Generator) createInputSchema(operation *Operation) mcp.ToolInputSchema {
	properties := make(map[string]interface{})
	var required []string

	// Add parameters
	for _, param := range operation.Parameters {
		properties[param.Name] = map[string]interface{}{
			"type":        g.getSchemaType(param.Schema),
			"description": param.Description,
		}
		if param.Required {
			required = append(required, param.Name)
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

func (g *Generator) getSchemaType(schema map[string]interface{}) string {
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
