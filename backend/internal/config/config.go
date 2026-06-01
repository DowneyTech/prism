package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	FrontendURL string
	JWTSecret   string
	AESKey      string // 32 bytes hex for AES-256
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://prism:prism@db:5432/prism?sslmode=disable"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		AESKey:      getEnv("AES_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
