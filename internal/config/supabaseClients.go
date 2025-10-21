package config

import (
	"log"
	"open_api_to_mcp_server/internal/database"
	"os"
)

var DB *database.DB

func InitDB() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := database.NewDB(dbURL)
	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)
		return err
	}

	DB = db
	log.Println("✅ Connected to Supabase successfully")
	return nil
}

func CloseDB() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
