package mail

import "testing"

func TestNewProviderResend(t *testing.T) {
	p := NewProvider(ProviderConfig{Provider: "resend", APIKey: "re_test", From: "onboarding@resend.dev"})
	if _, ok := p.(*ResendService); !ok {
		t.Fatalf("expected ResendService, got %T", p)
	}
}

func TestNewProviderSMTPDefault(t *testing.T) {
	p := NewProvider(ProviderConfig{SMTP: SMTPConfig{Host: "localhost", Port: "1025", From: "noreply@test.local"}})
	if _, ok := p.(*SMTPService); !ok {
		t.Fatalf("expected SMTPService, got %T", p)
	}
}

func TestNewProviderResendFromHost(t *testing.T) {
	p := NewProvider(ProviderConfig{
		SMTP: SMTPConfig{Host: "smtp.resend.com", Port: "465", Password: "re_test", From: "onboarding@resend.dev"},
	})
	if _, ok := p.(*ResendService); !ok {
		t.Fatalf("expected ResendService, got %T", p)
	}
}
