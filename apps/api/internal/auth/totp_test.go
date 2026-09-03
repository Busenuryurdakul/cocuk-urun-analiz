package auth_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
)

func TestTOTPGenerateAndValidate(t *testing.T) {
	secret, _, err := auth.GenerateTOTPSecret("Miyuna", "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	// Validation with invalid code should fail
	if auth.ValidateTOTP(secret, "000000") {
		t.Fatal("expected invalid TOTP code to fail")
	}
}
