// internal/handlers/auth_handlers.go
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"open_api_to_mcp_server/internal/auth"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/pkg/utils"
)

type AuthHandler struct {
    db *database.DB
}

func NewAuthHandler(db *database.DB) *AuthHandler {
    return &AuthHandler{db: db}
}

// POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email       string `json:"email"`
        Password    string `json:"password"`
        CompanyName string `json:"company_name"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errStr := fmt.Sprintf("Invalid request body: %v", err)
        utils.SendError(w, 400, errStr)
        return
    }
    
    // Validate
    if req.Email == "" || req.Password == "" {
        utils.SendError(w, 400, "Email and password required")
        return
    }
    
    // Hash password
    passwordHash, err := auth.HashPassword(req.Password)
    if err != nil {
        utils.SendError(w, 500, "Failed to hash password")
        return
    }
    
    // Generate API key
    apiKey, apiKeyHash, apiKeyPrefix, err := auth.GenerateAPIKey()
    log.Println("API Key:", apiKey)
    log.Println("API Key Hash:", apiKeyHash)
    log.Println("API Key Prefix:", apiKeyPrefix)
    if err != nil {
        utils.SendError(w, 500, "Failed to generate API key")
        return
    }
    
    // Save to database
    developerID, err := h.db.CreateDeveloper(req.Email, passwordHash, apiKeyHash, apiKeyPrefix)
    if err != nil {
        errStr := fmt.Sprintf("Failed to create developer account: %v", err)
        utils.SendError(w, 500, errStr)
        return
    }
    
    // Response
    utils.SendSuccess(w, map[string]interface{}{
        "developer_id": developerID,
        "email":        req.Email,
        "api_key":      apiKey, // Show ONLY once!
        "message":      "Save your API key - it won't be shown again",
    })
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement JWT-based login for dashboard
    utils.SendError(w, 501, "Not implemented yet")
}

func (h *AuthHandler) GetAllDevelopers(w http.ResponseWriter, r *http.Request) {
	developers, err := h.db.GetAllDevelopers()
	if err != nil {
		utils.SendError(w, 500, "Failed to get all developers")
		return
	}
	utils.SendSuccess(w, developers)
}