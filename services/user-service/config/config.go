package config

import (
	"fmt"
	"os"
)

const defaultEncryptionKey = "dev-encryption-key-32-bytes!!!!!"

type Config struct {
	DatabaseURL   string
	GRPCPort      string
	RedisURL      string
	EncryptionKey string
	ResendAPIKey  string
	PlatformURL   string
	APIPublicURL  string
	EmailFrom     string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://authservice:authservice@localhost:5432/authservice?sslmode=disable"),
		GRPCPort:      getEnv("GRPC_PORT", "50053"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", defaultEncryptionKey),
		ResendAPIKey:  getEnv("RESEND_API_KEY", ""),
		PlatformURL:   getEnv("PLATFORM_URL", "http://localhost:3001"),
		APIPublicURL:  getEnv("API_PUBLIC_URL", "http://localhost:8080"),
		EmailFrom:     getEnv("EMAIL_FROM", ""),
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
