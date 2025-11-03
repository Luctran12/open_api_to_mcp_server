// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"open_api_to_mcp_server/internal/config"
	"open_api_to_mcp_server/internal/handler"
	"open_api_to_mcp_server/internal/middleware"
	"os"
	"path/filepath"

	
)

func main() {
	// Load config


	// Load database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	log.Println("DATABASE_URL:", dbURL)
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	if err := config.InitDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer config.CloseDB()

	// Initialize handlers
	authHandler := handler.NewAuthHandler(config.DB)
	specHandler := handler.NewSpecHandler(config.DB)
	toolHandler := handler.NewToolHandler(config.DB)
	executeHandler := handler.NewExecuteHandler(config.DB)

	// Setup router
	mux := http.NewServeMux()

	//api for connection test
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("bay bong"))
	})

	// Public routes
	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)

	mux.HandleFunc("/api/developers", authHandler.GetAllDevelopers)

	// Protected routes (require authentication)
	mux.Handle("/api/specs", middleware.Authenticate(config.DB)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			specHandler.UploadSpec(w, r)
		case "GET":
			specHandler.ListSpecs(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/tools", middleware.Authenticate(config.DB)(http.HandlerFunc(toolHandler.GetTools)))
	mux.Handle("/api/execute", middleware.Authenticate(config.DB)(http.HandlerFunc(executeHandler.Execute)))
	mux.Handle("/api/build", middleware.Authenticate(config.DB)(http.HandlerFunc(executeHandler.Build)))
	mux.Handle("/api/build/download/",
	http.StripPrefix("/api/build/download/",
		http.FileServer(http.Dir(filepath.Join(os.TempDir(), "builds"))),
	),
)


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
