package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPublishedPolicyImmutable(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/miyuna_test"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	repo := repository.NewCompliancePolicyRepository(client.DB)
	orgID := primitive.NewObjectID()
	policy := &domain.CompliancePolicyVersion{
		OrganizationID: &orgID,
		Profile:        domain.ProfileKVKK,
		Version:        "9.9.9-test",
		Status:         domain.PolicyStatusPublished,
		Rules:          domain.PolicyRules{},
		CreatedBy:      primitive.NewObjectID(),
	}
	if err := repo.Insert(ctx, policy); err != nil {
		t.Fatal(err)
	}
	err = repo.UpdateStatus(ctx, policy.ID, domain.PolicyStatusArchived)
	if err == nil {
		t.Fatal("expected published policy update to fail or no-op without matching draft")
	}
}
