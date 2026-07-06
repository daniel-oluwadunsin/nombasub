package mcp

import (
	"fmt"
	"os"
)

type Config struct {
	Port              string
	EngineURL         string
	RequestsPerMinute int
}

func LoadConfig() *Config {
	return &Config{
		Port:              getEnv("MCP_PORT", "8081"),
		EngineURL:         getEnv("NOMBASUB_ENGINE_URL", "http://localhost:8080"),
		RequestsPerMinute: getEnvInt("MCP_RATE_LIMIT_PER_MINUTE", 120),
	}
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
