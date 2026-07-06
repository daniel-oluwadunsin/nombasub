package mcp

import "os"

type Config struct {
	Port      string
	EngineURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:      getEnv("MCP_PORT", "8081"),
		EngineURL: getEnv("NOMBASUB_ENGINE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
