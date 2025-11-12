package auth

import (
	"fmt"
	"log"
	"open_api_to_mcp_server/internal/database"
	"time"

	"github.com/golang-jwt/jwt"
)

//create jwt token include user id and expiration time
func CreateJWT(secret []byte, userID string, db *database.DB) (string, error) {
	apiKey, err := db.GetDeveloperAPIKeyHashByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get API key for user %s: %v", userID, err)
	}
	expiration := time.Second * 3600 // 1 hour
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(expiration).Unix(),
		"X-API-KEY": apiKey,
	})
	return token.SignedString(secret)
}


// func ValidateToken(t string) (*jwt.Token, error) {
// 	log.Println("Validating token:", t)
// 	return jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
// 		if _,ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
// 			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
// 		}
// 		return []byte("your-256-bit-secret"), nil
// 	})
// }

// Hàm xác thực JWT
func ValidateToken(t string, secret []byte) (*jwt.Token, error) {
	log.Println("Validating token:", t)

	// Parse token và kiểm tra phương thức ký
	token, err := jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
		// Kiểm tra phương thức ký là HS256 (HMAC-SHA256)
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		// Trả về khóa bí mật để kiểm tra chữ ký
		return secret, nil
	})

	// Kiểm tra nếu có lỗi khi parse hoặc token không hợp lệ
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

