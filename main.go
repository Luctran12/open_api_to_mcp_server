// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/internal/handler"
	"open_api_to_mcp_server/internal/middleware"
	"os"

	"github.com/joho/godotenv"
)

func main() {
    // Load config

    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    dbURL := os.Getenv("DATABASE_URL")
    log.Println("DATABASE_URL:", dbURL)
    if dbURL == "" {
        log.Fatal("DATABASE_URL not set")
    }
    
    // Connect to database
    db, err := database.NewDB(dbURL)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()
    
    // Initialize handlers
    authHandler := handler.NewAuthHandler(db)
    specHandler := handler.NewSpecHandler(db)
    toolHandler := handler.NewToolHandler(db)
    executeHandler := handler.NewExecuteHandler(db)
    
    // Setup router
    mux := http.NewServeMux()
    
    // Public routes
    mux.HandleFunc("/api/auth/register", authHandler.Register)
    mux.HandleFunc("/api/auth/login", authHandler.Login)

    mux.HandleFunc("/api/developers", authHandler.GetAllDevelopers)
    
    // Protected routes (require authentication)
    mux.Handle("/api/specs", middleware.Authenticate(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "POST":
            specHandler.UploadSpec(w, r)
        case "GET":
            specHandler.ListSpecs(w, r)
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })))
    
    mux.Handle("/api/tools", middleware.Authenticate(db)(http.HandlerFunc(toolHandler.GetTools)))
    mux.Handle("/api/execute", middleware.Authenticate(db)(http.HandlerFunc(executeHandler.Execute)))
    
    // Apply global middleware
    handler := middleware.CORS(mux)
    handler = middleware.Logging(handler)
    
    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"
    }
    
    log.Printf("🚀 Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}