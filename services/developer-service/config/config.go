package config

import (
	"fmt"
	"os"
)

const defaultEncryptionKey = "dev-encryption-key-32-bytes!!!!!"

type Config struct {
	DatabaseURL   string
	GRPCPort      string
	JWTSecret     string
	EncryptionKey string // 32 bytes for AES-256
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://authservice:authservice@localhost:5432/authservice?sslmode=disable"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", defaultEncryptionKey),
	}
	if len(cfg.EncryptionKey) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes, got %d", len(cfg.EncryptionKey))
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
