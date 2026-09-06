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
	AgentInternalToken     string
	AgentIPCTimeout        time.Duration
	GoInternalAPIURL       string
	AllowGraphQLPlayground bool
	MailSMTPHost           string
	MailSMTPPort           string
	MailSMTPUser           string
	MailSMTPPass           string
	MailSMTPTLS            bool
	MailFrom               string
	MailProvider           string
	MailQueueEnabled       bool
	MailRetryMax           int
	TurnstileSecretKey     string
	TurnstileEnabled       bool
	LoginEmailOTPTTL       time.Duration
	WebBaseURL             string
	MFAIssuer              string
	CookieSecure           bool
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	MaxOTPAttempts         int
	LoginMaxAttempts       int
	LoginLockoutDuration   time.Duration
	S3Endpoint             string
	S3AccessKey            string
	S3SecretKey            string
	S3Bucket               string
	S3ForcePathStyle       bool
	LLMUseMock             bool
	LLMRequestTimeout      time.Duration
	LLMMaxRetries          int
}

// Load reads configuration from environment with dev defaults.
func Load() Config {
	return Config{
		Host:                   getEnv("API_HOST", "0.0.0.0"),
		Port:                   getEnv("PORT", getEnv("API_PORT", "8080")),
		MongoURI:               getEnv("MONGODB_URI", "mongodb://localhost:27017/miyuna"),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AgentOrchestratorURL:   getEnv("AGENT_ORCHESTRATOR_URL", "http://127.0.0.1:8090"),
		AgentInternalToken:     getEnv("AGENT_INTERNAL_TOKEN", "dev-internal-token-change-me"),
		AgentIPCTimeout:        durationEnv("AGENT_IPC_TIMEOUT", 15*time.Second),
		GoInternalAPIURL:       getEnv("GO_INTERNAL_API_URL", "http://127.0.0.1:8080"),
		AllowGraphQLPlayground: getEnv("ALLOW_GRAPHQL_PLAYGROUND", "true") == "true",
		MailSMTPHost:           getEnv("MAIL_SMTP_HOST", "localhost"),
		MailSMTPPort:           getEnv("MAIL_SMTP_PORT", "1025"),
		MailSMTPUser:           getEnv("MAIL_SMTP_USER", ""),
		MailSMTPPass:           getEnv("MAIL_SMTP_PASS", ""),
		MailSMTPTLS:            getEnv("MAIL_SMTP_TLS", "false") == "true",
		MailFrom:               getEnv("MAIL_FROM", "noreply@miyuna.local"),
		MailProvider:           getEnv("MAIL_PROVIDER", ""),
		MailQueueEnabled:       getEnv("MAIL_QUEUE_ENABLED", "true") == "true",
		MailRetryMax:           intEnv("MAIL_RETRY_MAX", 3),
		TurnstileSecretKey:     getEnv("TURNSTILE_SECRET_KEY", ""),
		TurnstileEnabled:       getEnv("TURNSTILE_ENABLED", "false") == "true",
		LoginEmailOTPTTL:       durationEnv("LOGIN_EMAIL_OTP_TTL", 10*time.Minute),
		WebBaseURL:             getEnv("WEB_BASE_URL", "http://localhost:3000"),
		MFAIssuer:              getEnv("MFA_ISSUER", "Miyuna"),
		CookieSecure:           getEnv("COOKIE_SECURE", "false") == "true",
		JWTSecret:              getEnv("JWT_SECRET", "change-me-jwt-dev-secret-32chars"),
		AccessTokenTTL:         durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:        durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		MaxOTPAttempts:         intEnv("MAX_OTP_ATTEMPTS", 5),
		LoginMaxAttempts:       intEnv("LOGIN_MAX_ATTEMPTS", 5),
		LoginLockoutDuration:   durationEnv("LOGIN_LOCKOUT_DURATION", 15*time.Minute),
		S3Endpoint:             getEnv("S3_ENDPOINT", ""),
		S3AccessKey:            getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:            getEnv("S3_SECRET_KEY", ""),
		S3Bucket:               getEnv("S3_BUCKET", ""),
		S3ForcePathStyle:       getEnv("S3_FORCE_PATH_STYLE", "true") == "true",
		LLMUseMock:             getEnv("LLM_USE_MOCK", "true") == "true",
		LLMRequestTimeout:      durationEnv("LLM_REQUEST_TIMEOUT", 30*time.Second),
		LLMMaxRetries:          intEnv("LLM_MAX_RETRIES", 1),
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
