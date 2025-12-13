// internal/handlers/spec_handlers.go
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/pkg/openapi"

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
	const MAX_OPENAPI_SPEC_PER_DEVELOPER = 10
	// Get developer from context (set by auth middleware)
	developer := r.Context().Value("developer").(*database.Developer)
	numberOfOpenAPISpec, err := h.db.GetNumberOfOpenAPISpecByDeveloperID(developer.ID)
	if err != nil {
		utils.SendError(w, 500, "Failed to get number of openapi specs")
		return
	}
	if numberOfOpenAPISpec >= MAX_OPENAPI_SPEC_PER_DEVELOPER {
		utils.SendError(w, 400, "Maximum number of openapi specs reached")
		return
	}

	// Expect multipart/form-data with a file field named "file" (or fallback to "spec")
	if err := r.ParseMultipartForm(5 << 20); err != nil { // 5MB
		utils.SendError(w, 400, "Expected multipart/form-data with file upload")
		fmt.Println(err.Error())
		return
	}

	file, fileName, err := r.FormFile("file")
	if err != nil {
		// Fallback to a field named "spec"
		file, _, err = r.FormFile("spec")
		if err != nil {
			utils.SendError(w, 400, "Missing file field 'file' or 'spec'")
			return
		}
	}

	// check file extension and file size
	if _,err := utils.IsAllowedFileExtension(fileName); err != nil {
		utils.SendError(w, 400, err.Error())
		return
	}
	
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		utils.SendError(w, 400, "Unable to read uploaded file")
		return
	}

	// Parse OpenAPI spec from uploaded JSON file
	var spec openapi.Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		utils.SendError(w, 400, "Invalid OpenAPI spec JSON in file")
		return
	}

	// Validate spec
	if spec.OpenAPI == "" || spec.Info.Title == "" {
		utils.SendError(w, 400, "Invalid OpenAPI spec: missing required fields")
		return
	}

	// Save spec to database (store canonicalized JSON)
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

type SpecResponse struct {
	ID          string    `json:"id"`
    SpecName    string    `json:"spec_name"`
}

// GET /api/specs
func (h *SpecHandler) ListSpecs(w http.ResponseWriter, r *http.Request) {
	developer := r.Context().Value("developer").(*database.Developer)

	specs, err := h.db.GetDeveloperSpecs(developer.ID)
	specResponses := []SpecResponse{}
	for _, spec := range specs {
		specResponses = append(specResponses, SpecResponse{
			ID:          spec.ID,
			SpecName:    spec.SpecName,
		})
	}
	if err != nil {
		utils.SendError(w, 500, "Failed to load specs")
		return
	}

	utils.SendSuccess(w, map[string]interface{}{
		"specs": specResponses,
		"count": len(specResponses),
	})
}

// DELETE /api/specs/{spec_id}
func (h *SpecHandler) DeleteSpec(w http.ResponseWriter, r *http.Request) {
	developer := r.Context().Value("developer").(*database.Developer)	
	specID := r.URL.Path[len("/api/specs/"):]
	if specID == "" {
		utils.SendError(w, 400, "Missing spec ID in URL")
		return
	}
	err := h.db.DeleteOpenAPISpec(developer.ID, specID)
	if err != nil {
		utils.SendError(w, 500, "Failed to delete spec: "+err.Error())
		return
	}
}
