package mail

import "strings"

type ProviderConfig struct {
	Provider string
	SMTP     SMTPConfig
	APIKey   string
	From     string
}

func NewProvider(cfg ProviderConfig) Service {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" && strings.EqualFold(cfg.SMTP.Host, "smtp.resend.com") {
		provider = "resend"
	}
	if provider == "resend" {
		key := cfg.APIKey
		if key == "" {
			key = cfg.SMTP.Password
		}
		from := cfg.From
		if from == "" {
			from = cfg.SMTP.From
		}
		return NewResend(ResendConfig{APIKey: key, From: from})
	}
	return NewSMTP(cfg.SMTP)
}
