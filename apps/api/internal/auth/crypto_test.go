package auth_test

import (
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
)

func TestArgon2idHashAndVerify(t *testing.T) {
	hash, err := auth.HashPassword("secure-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id prefix, got %q", hash)
	}
	if !auth.CheckPassword(hash, "secure-password-123") {
		t.Fatal("expected password to match")
	}
	if auth.CheckPassword(hash, "wrong-password") {
		t.Fatal("expected password mismatch")
	}
}

func TestArgon2idRejectsBcrypt(t *testing.T) {
	if auth.CheckPassword("$2a$12$abcdefghijklmnopqrstuv", "password") {
		t.Fatal("bcrypt hash must not verify under argon2-only checker")
	}
}

func TestTokenHashDeterministic(t *testing.T) {
	a := auth.HashToken("abc")
	b := auth.HashToken("abc")
	if a != b {
		t.Fatal("token hash should be deterministic")
	}
}
