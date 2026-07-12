package config

import (
	"log"
	"os"
	"strings"

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
	NombaBaseURL       string
	NombaAccountID     string
	NombaSubAccountID  string
	NombaWebhookSecret string
	NombaSenderName    string
	MailerUser         string
	MailerPassword     string
	ClientURL          string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DBDSN:              requireEnv("DB_DSN"),
		JWTSecret:          requireSecret("JWT_SECRET", 32),
		JWTRefreshSecret:   requireSecret("JWT_REFRESH_SECRET", 32),
		RabbitMQURL:        requireEnv("RABBITMQ_URL"),
		APIKeyHeader:       getEnv("API_KEY_HEADER", "X-Api-Key"),
		EncryptionKey:      requireEnv("ENCRYPTION_KEY"),
		NombaClientID:      requireEnv("NOMBA_CLIENT_ID"),
		NombaBaseURL:       getEnv("NOMBA_BASE_URL", "https://api.nomba.com"),
		NombaSubAccountID:  requireEnv("NOMBA_SUBACCOUNT_ID"),
		NombaClientSecret:  requireEnv("NOMBA_CLIENT_SECRET"),
		NombaAccountID:     requireEnv("NOMBA_ACCOUNT_ID"),
		NombaWebhookSecret: requireEnv("NOMBA_WEBHOOK_SECRET"),
		NombaSenderName:    getEnv("NOMBA_SENDER_NAME", "NombaSub Platform"),
		MailerUser:         requireEnv("MAILER_USER"),
		MailerPassword:     requireEnv("MAILER_PASSWORD"),
		ClientURL:          requireEnv("CLIENT_URL"),
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

// requireSecret is like requireEnv but additionally rejects secrets that are too
// short or still set to the shipped placeholder, so the app refuses to boot with
// a weak or default signing key.
func requireSecret(key string, minLen int) string {
	v := requireEnv(key)
	if len(v) < minLen {
		log.Fatalf("env var %s must be at least %d characters", key, minLen)
	}
	if strings.Contains(strings.ToLower(v), "change-me") {
		log.Fatalf("env var %s is still set to a placeholder value; set a strong random secret", key)
	}
	return v
}
