// internal/auth/auth.go
package auth

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
   
    "golang.org/x/crypto/bcrypt"
)

// Generate API Key
func GenerateAPIKey() (key, hash, prefix string, err error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", "", "", err
    }
    
    key = "sk_live_" + hex.EncodeToString(bytes)
    prefix = key[:15] // "sk_live_abc1234"
    
    // Hash for storage
    hashBytes := sha256.Sum256([]byte(key))
    hash = hex.EncodeToString(hashBytes[:])
    
    return key, hash, prefix, nil
}

// Verify API Key
func VerifyAPIKey(providedKey, storedHash string) bool {
    hashBytes := sha256.Sum256([]byte(providedKey))
    providedHash := hex.EncodeToString(hashBytes[:])
    return providedHash == storedHash
}

// Hash password
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

// Verify password
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}