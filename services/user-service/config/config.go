package config

import "os"

type Config struct {
	DatabaseURL   string
	GRPCPort      string
	RedisURL      string
	EncryptionKey string
	ResendAPIKey  string
	PlatformURL   string
	EmailFrom     string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://authservice:authservice@localhost:5432/authservice?sslmode=disable"),
		GRPCPort:      getEnv("GRPC_PORT", "50053"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "dev-encryption-key-32-bytes!!!!"),
		ResendAPIKey:  getEnv("RESEND_API_KEY", ""),
		PlatformURL:   getEnv("PLATFORM_URL", "http://localhost:3001"),
		EmailFrom:     getEnv("EMAIL_FROM", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
