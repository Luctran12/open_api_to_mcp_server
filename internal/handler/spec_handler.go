// internal/handlers/spec_handlers.go
package handler

import (
	"encoding/json"
	"net/http"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/internal/openapi"

	"open_api_to_mcp_server/pkg/utils"
)

type SpecHandler struct {
	db *database.DB
}

func NewSpecHandler(db *database.DB) *SpecHandler {
	return &SpecHandler{db: db}
}

// POST /api/specs
func (h *SpecHandler) UploadSpec(w http.ResponseWriter, r *http.Request) {
	// Get developer from context (set by auth middleware)
	developer := r.Context().Value("developer").(*database.Developer)

	// Parse OpenAPI spec
	var spec openapi.Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		utils.SendError(w, 400, "Invalid OpenAPI spec format")
		return
	}

	// Validate spec
	if spec.OpenAPI == "" || spec.Info.Title == "" {
		utils.SendError(w, 400, "Invalid OpenAPI spec: missing required fields")
		return
	}

	// Save spec to database
	specJSON, _ := json.Marshal(spec)
	specID, err := h.db.SaveOpenAPISpec(developer.ID, spec.Info.Title, specJSON)
	if err != nil {
		utils.SendError(w, 500, "Failed to save OpenAPI spec")
		return
	}

	// Generate tools from spec
	tools, err := openapi.NewGenerator().Generate(&spec)
	if err != nil {
		utils.SendError(w, 400, "Failed to generate tools: "+err.Error())
		return
	}

	// Save tools to database
	toolNames := []string{}
	for _, tool := range tools {
		// Convert mcp.ToolInputSchema -> map[string]interface{} to match DB signature
		// This preserves schema shape as JSON for storage
		inputSchema := map[string]interface{}{
			"type":       tool.InputSchema.Type,
			"properties": tool.InputSchema.Properties,
			"required":   tool.InputSchema.Required,
		}

		err := h.db.SaveTool(
			developer.ID,
			specID,
			tool.Name,
			tool.Description,
			tool.Method,
			tool.URL,
			inputSchema,
			tool.RequiresAuth,
		)
		if err != nil {
			// Log error but continue
			continue
		}
		toolNames = append(toolNames, tool.Name)
	}

	// Response
	utils.SendSuccess(w, map[string]interface{}{
		"spec_id":     specID,
		"spec_name":   spec.Info.Title,
		"tools_count": len(toolNames),
		"tools":       toolNames,
	})
}

// GET /api/specs
func (h *SpecHandler) ListSpecs(w http.ResponseWriter, r *http.Request) {
	developer := r.Context().Value("developer").(*database.Developer)

	specs, err := h.db.GetDeveloperSpecs(developer.ID)
	if err != nil {
		utils.SendError(w, 500, "Failed to load specs")
		return
	}

	utils.SendSuccess(w, map[string]interface{}{
		"specs": specs,
		"count": len(specs),
	})
}
