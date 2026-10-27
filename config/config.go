package config

import (
	"os"
)

type Config struct {
	AppPort           string
	DBHost            string
	DBPort            string
	DBName            string
	DBUser            string
	DBPassword        string
	DBSSLMode         string
	LogLevel          string
	ProductServiceURL string
	AWSRegion         string
	JWTSecret         string
}

// LoadConfig loads application configuration strictly from environment variables.
// Sensitive database credentials, endpoints, and secrets must be passed via OS environment variables.
func LoadConfig() *Config {
	return &Config{
		AppPort:           getEnv("APP_PORT", "8080"),
		DBHost:            os.Getenv("DB_HOST"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBName:            os.Getenv("DB_NAME"),
		DBUser:            os.Getenv("DB_USER"),
		DBPassword:        os.Getenv("DB_PASSWORD"),
		DBSSLMode:         getEnv("DB_SSL_MODE", "require"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		ProductServiceURL: getEnv("PRODUCT_SERVICE_URL", "http://localhost:8081"),
		AWSRegion:         getEnv("AWS_REGION", "eu-north-1"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
