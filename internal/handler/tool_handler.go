// internal/handlers/tool_handlers.go
package handler

import (
    "net/http"
    "open_api_to_mcp_server/internal/database"
    "open_api_to_mcp_server/pkg/utils"
)

type ToolHandler struct {
    db *database.DB
}

func NewToolHandler(db *database.DB) *ToolHandler {
    return &ToolHandler{db: db}
}

// GET /api/tools
func (h *ToolHandler) GetTools(w http.ResponseWriter, r *http.Request) {
    developer := r.Context().Value("developer").(*database.Developer)
    
    defer r.Body.Close()
    

    // Load tools from database
    tools, err := h.db.GetDeveloperTools(developer.ID)
    if err != nil {
        utils.SendError(w, 500, "Failed to load tools")
        return
    }
    
    // Convert to LLM-compatible format
    toolDefinitions := make([]map[string]interface{}, 0)
    for _, tool := range tools {
        toolDefinitions = append(toolDefinitions, map[string]interface{}{
            "name":         tool.Name,
            "description":  tool.Description,
            "input_schema": tool.InputSchema,
        })
    }
    
    utils.SendSuccess(w, map[string]interface{}{
        "tools": toolDefinitions,
        "count": len(toolDefinitions),
    })
}