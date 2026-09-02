package config

import (
	"os"
)

// Config holds Phase 1 runtime configuration from environment.
type Config struct {
	Host                 string
	Port                 string
	MongoURI             string
	RedisURL             string
	AgentOrchestratorURL string
	// AllowGraphQLPlayground enables GraphQL playground UI (dev default: true via env).
	// Production intent: ALLOW_GRAPHQL_PLAYGROUND=false — public prod GraphQL playground is OUT OF SCOPE (FINAL_MASTER_PROMPT.md).
	AllowGraphQLPlayground bool
}

// Load reads configuration from environment with dev defaults.
func Load() Config {
	return Config{
		Host:                   getEnv("API_HOST", "0.0.0.0"),
		Port:                   getEnv("API_PORT", "8080"),
		MongoURI:               getEnv("MONGODB_URI", "mongodb://localhost:27017/miyuna"),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AgentOrchestratorURL:   getEnv("AGENT_ORCHESTRATOR_URL", "http://127.0.0.1:8090"),
		AllowGraphQLPlayground: getEnv("ALLOW_GRAPHQL_PLAYGROUND", "true") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
