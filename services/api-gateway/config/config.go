package config

import "os"

type Config struct {
	HTTPPort             string
	RedisURL             string
	DeveloperServiceAddr string
	UserServiceAddr      string
	TokenServiceAddr     string
	JWTSecret            string
}

func Load() *Config {
	return &Config{
		HTTPPort:             getEnv("HTTP_PORT", "8080"),
		RedisURL:             getEnv("REDIS_URL", "localhost:6379"),
		DeveloperServiceAddr: getEnv("DEVELOPER_SERVICE_ADDR", "localhost:50051"),
		UserServiceAddr:      getEnv("USER_SERVICE_ADDR", "localhost:50053"),
		TokenServiceAddr:     getEnv("TOKEN_SERVICE_ADDR", "localhost:50052"),
		JWTSecret:            getEnv("JWT_SECRET", "dev-secret-change-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
