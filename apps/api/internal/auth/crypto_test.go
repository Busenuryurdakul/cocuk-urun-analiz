package auth_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := auth.HashPassword("secure-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(hash, "secure-password-123") {
		t.Fatal("expected password to match")
	}
	if auth.CheckPassword(hash, "wrong-password") {
		t.Fatal("expected password mismatch")
	}
}

func TestTokenHashDeterministic(t *testing.T) {
	a := auth.HashToken("abc")
	b := auth.HashToken("abc")
	if a != b {
		t.Fatal("token hash should be deterministic")
	}
}
