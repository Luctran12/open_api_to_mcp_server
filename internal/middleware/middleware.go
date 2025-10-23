// internal/middleware/middleware.go
package middleware

import (
	"context"
	// "crypto/sha256"
	// "encoding/hex"
	"log"
	"net/http"
	"open_api_to_mcp_server/internal/database"
	"strings"
	"time"
)

// CORS Middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging Middleware
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)

		log.Printf("Completed in %v", time.Since(start))
	})
}

// Authentication Middleware
func Authenticate(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
            log.Println(apiKey)
			if apiKey == "" {
				http.Error(w, `{"error": "X-API-Key header required"}`, http.StatusUnauthorized)
				return
			}

			// Hash the provided key
			// hashBytes := sha256.Sum256([]byte(apiKey))
			// apiKeyHash := hex.EncodeToString(hashBytes[:])

			// Verify with database
			developer, err := db.GetDeveloperByAPIKey(apiKey)
           
            log.Println(apiKey)
			if err != nil {
				log.Println("Invalid API key:", err)
				http.Error(w, `{"error": "Invalid API key"}`, http.StatusUnauthorized)
				return
			}

			// Add developer to context
			ctx := context.WithValue(r.Context(), "developer", developer)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Rate Limiting Middleware
func RateLimit(next http.Handler) http.Handler {
	// TODO: Implement with Redis
	return next
}
