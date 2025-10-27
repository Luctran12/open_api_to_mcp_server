package config

import (
	"os"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Auth     AuthConfig
	OpenAPI  OpenAPIConfig
}

type ServerConfig struct {
	Name       string
	Version    string
	HTTPPort   string
	HTTPTimeout time.Duration
}

type AuthConfig struct {
	BearerToken string
	APIKey      string
}

type OpenAPIConfig struct {
	SpecPath string
}

// Load creates configuration from environment variables and defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Name:       getEnv("SERVER_NAME", "openapi-client"),
			Version:    getEnv("SERVER_VERSION", "1.0.0"),
			HTTPPort:   getEnv("HTTP_PORT", ":8080"),
			HTTPTimeout: 30 * time.Second,
		},
		Auth: AuthConfig{
			BearerToken: os.Getenv("BEARER_TOKEN"),
			APIKey:      os.Getenv("API_KEY"),
		},
		OpenAPI: OpenAPIConfig{
			SpecPath: getEnv("OPENAPI_SPEC_PATH", "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
