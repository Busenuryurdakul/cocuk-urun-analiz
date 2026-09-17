package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProductRepositoryUpdateTenantScoped(t *testing.T) {
	if !mongoTestsEnabled {
		t.Skip("mongodb unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dbName := fmt.Sprintf("miyuna_product_repo_%d", time.Now().UnixNano())
	client, err := mongoclient.Connect(ctx, fmt.Sprintf("mongodb://localhost:27017/%s", dbName))
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	repo := repository.NewProductRepository(client.DB)
	orgA := primitive.NewObjectID()
	orgB := primitive.NewObjectID()
	otherID := primitive.NewObjectID()

	other := &domain.Product{
		ID:             otherID,
		OrganizationID: orgA,
		Name:           domain.ProductFieldMeta{Value: "Unrelated"},
	}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}

	target := &domain.Product{
		OrganizationID: orgA,
		Name:           domain.ProductFieldMeta{Value: "Original"},
		CurrentPrice:   domain.ProductFieldMeta{Value: "100"},
	}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	createdAt := target.CreatedAt
	targetID := target.ID

	synced := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	target.Name = domain.ProductFieldMeta{Value: "Updated"}
	target.LastSyncedAt = &synced
	target.LastSyncSource = "TRENDYOL"
	if err := repo.Update(ctx, orgA, target); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.FindByID(ctx, orgA, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name.Value != "Updated" {
		t.Fatalf("expected updated name")
	}
	if got.CreatedAt != createdAt {
		t.Fatal("createdAt must be preserved")
	}
	if got.ID != targetID || got.OrganizationID != orgA {
		t.Fatal("id/org must be preserved")
	}
	if got.LastSyncedAt == nil || !got.LastSyncedAt.Equal(synced) {
		t.Fatal("lastSyncedAt must persist")
	}

	target.Name = domain.ProductFieldMeta{Value: "Cross Tenant Attempt"}
	if err := repo.Update(ctx, orgB, target); err != repository.ErrNotFound {
		t.Fatalf("cross-tenant update must fail, got %v", err)
	}

	unchanged, err := repo.FindByID(ctx, orgA, otherID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Name.Value != "Unrelated" {
		t.Fatal("unrelated product must remain unchanged")
	}
}
