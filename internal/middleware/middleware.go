// internal/middleware/middleware.go
package middleware

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"strings"

	// "crypto/sha256"
	// "encoding/hex"
	"log"
	"net/http"
	"open_api_to_mcp_server/internal/auth"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/pkg/utils"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

// define Middleware type
type Middleware func(http.Handler) http.Handler

//Gzip Compression Middleware


func GzipCompression(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Kiểm tra xem client có hỗ trợ gzip không
        if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            next.ServeHTTP(w, r)
            return
        }
        // set the response header to indicate gzip encoding
        w.Header().Set("Content-Encoding", "gzip")
        gz := gzip.NewWriter(w)
        defer gz.Close()

        //wrap the response writer with gzip writer
        w = &gzipResponseWriter{ResponseWriter: w, writer: gz}
        next.ServeHTTP(w, r)
    })
}

type gzipResponseWriter struct {
    http.ResponseWriter
    writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
    return g.writer.Write(b)
}



// CORS Middleware

var whiteList = []string{
    "http://localhost:3000",
    "https://open-api-to-mcp-server-fe.vercel.app",
}

func CORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")

        // Kiểm tra Origin hợp lệ
        if origin != "" && isOriginAllowed(origin) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        } else {
            w.WriteHeader(http.StatusForbidden)
            utils.SendError(w, http.StatusForbidden, "Forbidden: Origin not allowed")
            return
        }

        // Cho phép browser cache theo Origin
        w.Header().Set("Vary", "Origin")

        // Các header CORS cần thiết
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization, X-Requested-With")

        // Preflight request
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func isOriginAllowed(origin string) bool {
    for _, allowed := range whiteList {
        if origin == allowed {
            return true
        }
    }
    return false
}



// Logging Middleware - Ghi log chi tiết các yêu cầu HTTP dưới dạng JSON vào file log.json
func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Tạo Request ID nếu chưa có (giúp dễ dàng truy vấn log)
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String() // Tạo một Request ID mới nếu không có
        }

        // Tính toán thời gian xử lý
        start := time.Now()

        // Thêm một wrapper để ghi lại mã trạng thái HTTP và nội dung đã gửi trả
        rw := &responseWriter{w, http.StatusOK}

        // Tiến hành xử lý yêu cầu tiếp theo trong middleware chain
        next.ServeHTTP(rw, r)

        // Tạo một log object với thông tin cần ghi
        logEntry := struct {
            RequestID    string `json:"request_id"`
            Method       string `json:"method"`
            URL          string `json:"url"`
            ClientIP     string `json:"client_ip"`
            UserAgent    string `json:"user_agent"`
            Referer      string `json:"referer"`
            StatusCode   int    `json:"status_code"`
            ResponseTime string `json:"response_time"`
        }{
            RequestID:    requestID,
            Method:       r.Method,
            URL:          r.URL.Path,
            ClientIP:     r.RemoteAddr,
            UserAgent:    r.UserAgent(),
            Referer:      r.Referer(),
            StatusCode:   rw.statusCode,
            ResponseTime: time.Since(start).String(),
        }

        // Mở hoặc tạo file log.json để ghi log
        file, err := os.OpenFile("log.json", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
        if err != nil {
            log.Fatalf("Could not open log file: %v", err)
        }
        defer file.Close()

        // Chuyển log entry thành JSON
        logData, err := json.Marshal(logEntry)
        if err != nil {
            log.Printf("Error marshaling log entry: %v", err)
            return
        }

        // Ghi log dưới dạng JSON vào file log.json
        file.Write(logData)
        file.Write([]byte("\n")) // Thêm dòng mới sau mỗi log
    })
}

// responseWriter là một wrapper để giữ mã trạng thái HTTP
// vì http.ResponseWriter không cho phép lấy mã trạng thái trực tiếp.
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

// Overwrite WriteHeader để ghi lại mã trạng thái HTTP
func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// Authentication Middleware
func Authenticate(db *database.DB, secretKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//get jwt from header
			tokenStr := r.Header.Get("Authorization")
			if tokenStr == "" {
				utils.SendError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}
			//validate token
			token, err := auth.ValidateToken(strings.TrimSpace(strings.TrimPrefix(tokenStr,"Bearer ")), secretKey)
			if err != nil || !token.Valid {
				log.Println("Invalid token:", err)
				utils.SendError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			claims := token.Claims.(jwt.MapClaims)
			//userID := claims["user_id"].(string)
			apiKey := claims["X-API-KEY"].(string)
			// apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
            // log.Println(apiKey)
			if apiKey == "" {
				utils.SendError(w, http.StatusUnauthorized, "invalid API KEY")
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
				utils.SendError(w, http.StatusUnauthorized, "invalid API KEY")
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

func ChainMiddleware(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}