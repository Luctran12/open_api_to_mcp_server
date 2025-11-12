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

	"github.com/golang-jwt/jwt"
)

type AuthHandler struct {
    db *database.DB
    SecretKey []byte
}

func NewAuthHandler(db *database.DB, jwtSecret []byte) *AuthHandler {
    return &AuthHandler{db: db, SecretKey: jwtSecret}
}

type LoginReq struct {
       Email    string `json:"email"`
       Password string `json:"password"`
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
   // Parse request
   var req LoginReq
   if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
       errStr := fmt.Sprintf("Invalid request body: %v", err)
       utils.SendError(w, 400, errStr)
       return
   }
    // Get developer by email
    developer, err := h.db.GetDeveloperByEmail(req.Email)
    if err != nil {
        utils.SendError(w, 401, "Invalid email or password")
        return
    }
    // Check password
    // passHashed, err := auth.HashPassword(req.Password)
    // if err != nil {
    //     utils.SendError(w, 500, "Failed to hash password")
    //     return
    // }
    if !auth.CheckPasswordHash(req.Password, developer.PasswordHash) {
        utils.SendError(w, 401, "Invalid  password")
        return
    }
    // Create JWT
    token, err := auth.CreateJWT(h.SecretKey, developer.ID, h.db)
    if err != nil {
        utils.SendError(w, 500, "Failed to create JWT")
        return
    }
    // Response
    utils.SendSuccess(w, map[string]interface{}{
        "JWTtoken": token,
    })
   
}

func (h *AuthHandler) GetAllDevelopers(w http.ResponseWriter, r *http.Request) {
	developers, err := h.db.GetAllDevelopers()
	if err != nil {
		utils.SendError(w, 500, "Failed to get all developers")
		return
	}
	utils.SendSuccess(w, developers)
}

func (h *AuthHandler) ValidateToken(token string) (*jwt.Token, error) {
    return auth.ValidateToken(token, h.SecretKey)
}