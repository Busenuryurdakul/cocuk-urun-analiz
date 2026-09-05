package integration

import (
	"context"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAuthRegisterBlocksLoginUntilEmailVerified(t *testing.T) {
	h := NewPhase5Harness(t)
	ctx := context.Background()
	email := "auth-e2e-" + primitive.NewObjectID().Hex() + "@example.com"

	if _, err := h.App.Auth.Register(ctx, email, "password1234"); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := h.App.Auth.Login(ctx, auth.LoginRequest{
		Email:             email,
		Password:          "password1234",
		DeviceFingerprint: "e2e-fp",
	})
	if err != auth.ErrEmailNotVerified {
		t.Fatalf("login before verify: got %v want EMAIL_NOT_VERIFIED", err)
	}
}
