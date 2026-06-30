package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBDSN              string
	JWTSecret          string
	JWTRefreshSecret   string
	RabbitMQURL        string
	EncryptionKey      string
	APIKeyHeader       string
	NombaClientID      string
	NombaClientSecret  string
	NombaAccountID     string
	NombaWebhookSecret string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DBDSN:              requireEnv("DB_DSN"),
		JWTSecret:          requireEnv("JWT_SECRET"),
		JWTRefreshSecret:   requireEnv("JWT_REFRESH_SECRET"),
		RabbitMQURL:        requireEnv("RABBITMQ_URL"),
		APIKeyHeader:       getEnv("API_KEY_HEADER", "X-Api-Key"),
		EncryptionKey:      requireEnv("ENCRYPTION_KEY"),
		NombaClientID:      requireEnv("NOMBA_CLIENT_ID"),
		NombaClientSecret:  requireEnv("NOMBA_CLIENT_SECRET"),
		NombaAccountID:     requireEnv("NOMBA_ACCOUNT_ID"),
		NombaWebhookSecret: requireEnv("NOMBA_WEBHOOK_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
