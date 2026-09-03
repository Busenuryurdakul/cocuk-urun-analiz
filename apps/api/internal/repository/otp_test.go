package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestEmailVerificationConsumeAndReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoclient.Connect(ctx, "mongodb://localhost:27017/miyuna_test_otp")
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	repo := repository.NewEmailVerificationRepository(client.DB)
	hash := "test-token-hash-" + primitive.NewObjectID().Hex()
	ev := &domain.EmailVerification{
		UserID:    primitive.NewObjectID(),
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := repo.Create(ctx, ev); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.ConsumeByTokenHash(ctx, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ConsumeByTokenHash(ctx, hash); err == nil {
		t.Fatal("replay must be rejected")
	}
}

func TestEmailVerificationExpiredRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoclient.Connect(ctx, "mongodb://localhost:27017/miyuna_test_otp")
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	repo := repository.NewEmailVerificationRepository(client.DB)
	hash := "expired-" + primitive.NewObjectID().Hex()
	ev := &domain.EmailVerification{
		UserID:    primitive.NewObjectID(),
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.Create(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ConsumeByTokenHash(ctx, hash); err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestMFASetupAttemptLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoclient.Connect(ctx, "mongodb://localhost:27017/miyuna_test_otp")
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	repo := repository.NewMFASetupRepository(client.DB)
	hash := "mfa-" + primitive.NewObjectID().Hex()
	ch := &domain.MFASetupChallenge{
		UserID:    primitive.NewObjectID(),
		TokenHash: hash,
		Secret:    "SECRET",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatal(err)
	}
	max := 3
	for i := 0; i < max-1; i++ {
		locked, err := repo.IncrementFailedAttempts(ctx, ch.ID, max)
		if err != nil || locked {
			t.Fatalf("attempt %d: locked=%v err=%v", i, locked, err)
		}
	}
	locked, err := repo.IncrementFailedAttempts(ctx, ch.ID, max)
	if err != nil || !locked {
		t.Fatal("expected lock at max attempts")
	}
	if _, err := repo.ConsumeByTokenHash(ctx, hash); err == nil {
		t.Fatal("locked challenge must not consume")
	}
}
