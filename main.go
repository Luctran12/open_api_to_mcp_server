package main

import (
	"log"
	"os"

	iconfig "open_api_to_mcp_server/internal/config"
	"open_api_to_mcp_server/pkg/config"
	"open_api_to_mcp_server/pkg/server"
	
)

func main() {
	// Load config
	cfg := config.Load()

	// err := godotenv.Load()
	// if err != nil {
	// 	log.Println("No .env file found, proceeding with environment variables")
	// }
	// Load database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	//Load jwt secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	log.Println("DATABASE_URL:", dbURL)
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	if err := iconfig.InitDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer iconfig.CloseDB()

	// Create a new server
	srv, err := server.New(cfg, iconfig.DB,[]byte(jwtSecret))
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
