package config

import (
	"os"
)

type Config struct {
	DatabaseURL   string
	GRPCPort      string
	JWTSecret     string
	EncryptionKey string // 32 bytes for AES-256
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://authservice:authservice@localhost:5432/authservice?sslmode=disable"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "dev-encryption-key-32-bytes!!!!"), // exactly 32 bytes
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
