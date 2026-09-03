package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds runtime configuration from environment.
type Config struct {
	Host                   string
	Port                   string
	MongoURI               string
	RedisURL               string
	AgentOrchestratorURL   string
	AllowGraphQLPlayground bool
	MailSMTPHost           string
	MailSMTPPort           string
	MailFrom               string
	WebBaseURL             string
	MFAIssuer              string
	CookieSecure           bool
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	MaxOTPAttempts         int
	LoginMaxAttempts       int
	LoginLockoutDuration   time.Duration
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
		MailSMTPHost:           getEnv("MAIL_SMTP_HOST", "localhost"),
		MailSMTPPort:           getEnv("MAIL_SMTP_PORT", "1025"),
		MailFrom:               getEnv("MAIL_FROM", "noreply@miyuna.local"),
		WebBaseURL:             getEnv("WEB_BASE_URL", "http://localhost:3000"),
		MFAIssuer:              getEnv("MFA_ISSUER", "Miyuna"),
		CookieSecure:           getEnv("COOKIE_SECURE", "false") == "true",
		JWTSecret:              getEnv("JWT_SECRET", "change-me-jwt-dev-secret-32chars"),
		AccessTokenTTL:         durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:        durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		MaxOTPAttempts:         intEnv("MAX_OTP_ATTEMPTS", 5),
		LoginMaxAttempts:       intEnv("LOGIN_MAX_ATTEMPTS", 5),
		LoginLockoutDuration:   durationEnv("LOGIN_LOCKOUT_DURATION", 15*time.Minute),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}
