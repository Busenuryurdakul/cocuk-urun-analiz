package turnstile_test

import (
	"context"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/turnstile"
)

func TestVerifierDisabledSkipsValidation(t *testing.T) {
	v := turnstile.NewVerifier("", false)
	if err := v.Verify(context.Background(), "", ""); err != nil {
		t.Fatalf("disabled verifier should pass: %v", err)
	}
}

func TestVerifierEnabledRequiresToken(t *testing.T) {
	v := turnstile.NewVerifier("secret", true)
	if err := v.Verify(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for missing token")
	}
}
