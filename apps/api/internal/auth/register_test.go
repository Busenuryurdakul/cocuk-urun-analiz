package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
)

type recordingMail struct {
	messages []mail.Message
}

func (r *recordingMail) Send(_ context.Context, msg mail.Message) error {
	r.messages = append(r.messages, msg)
	return nil
}

func TestRegisterThenLoginRequiresEmailVerification(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()
	svc.Policy.EmailVerifyTTL = 24 * time.Hour
	box := &recordingMail{}
	svc.Mail = box

	ctx := context.Background()
	email := "verify-login@example.com"
	password := "password1"

	ack, err := svc.Register(ctx, email, password)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if !strings.Contains(ack.Message, "doğrulama") {
		t.Fatalf("unexpected register message: %q", ack.Message)
	}
	if len(box.messages) != 1 {
		t.Fatalf("expected 1 verification mail, got %d", len(box.messages))
	}

	_, err = svc.Login(ctx, LoginRequest{Email: email, Password: password, DeviceFingerprint: "fp-1"})
	if err != ErrEmailNotVerified {
		t.Fatalf("login before verify: got %v, want EMAIL_NOT_VERIFIED", err)
	}
}

func TestResendEmailVerificationSendsNewLink(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()
	svc.Policy.EmailVerifyTTL = 24 * time.Hour
	box := &recordingMail{}
	svc.Mail = box

	ctx := context.Background()
	email := "resend@example.com"
	password := "password1"

	if _, err := svc.Register(ctx, email, password); err != nil {
		t.Fatalf("register: %v", err)
	}

	ack, err := svc.ResendEmailVerification(ctx, email, password)
	if err != nil {
		t.Fatalf("resend: %v", err)
	}
	if ack.Message == "" {
		t.Fatal("expected acknowledgement message")
	}
	if len(box.messages) < 2 {
		t.Fatalf("expected a second verification mail, got %d", len(box.messages))
	}
}

func TestRegisterDuplicateResendsWhenUnverified(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()
	svc.Policy.EmailVerifyTTL = 24 * time.Hour
	box := &recordingMail{}
	svc.Mail = box

	ctx := context.Background()
	email := "dup-resend@example.com"
	password := "password1"

	if _, err := svc.Register(ctx, email, password); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.Register(ctx, email, password); err != nil {
		t.Fatalf("duplicate register: %v", err)
	}
	if len(box.messages) < 2 {
		t.Fatalf("expected resend on duplicate register, got %d mails", len(box.messages))
	}
}
