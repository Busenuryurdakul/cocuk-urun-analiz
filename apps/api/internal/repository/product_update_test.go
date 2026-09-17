package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type memoryProductRepo struct {
	items []domain.Product
}

func (m *memoryProductRepo) Create(_ context.Context, p *domain.Product) error {
	p.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	m.items = append(m.items, *p)
	return nil
}

func (m *memoryProductRepo) Update(_ context.Context, organizationID primitive.ObjectID, product *domain.Product) error {
	for i := range m.items {
		if m.items[i].ID == product.ID && m.items[i].OrganizationID == organizationID {
			created := m.items[i].CreatedAt
			org := m.items[i].OrganizationID
			id := m.items[i].ID
			m.items[i] = *product
			m.items[i].CreatedAt = created
			m.items[i].OrganizationID = org
			m.items[i].ID = id
			return nil
		}
	}
	return repository.ErrNotFound
}

func TestProductUpdatePreservesCreatedAtAndBlocksCrossTenant(t *testing.T) {
	orgA := primitive.NewObjectID()
	orgB := primitive.NewObjectID()
	productID := primitive.NewObjectID()
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &memoryProductRepo{items: []domain.Product{{
		ID: productID, OrganizationID: orgA, CreatedAt: created,
		Name: domain.ProductFieldMeta{Value: "Old"},
	}}}

	synced := time.Now().UTC()
	updated := domain.Product{
		ID:             productID,
		OrganizationID: orgA,
		Name:           domain.ProductFieldMeta{Value: "New"},
		LastSyncedAt:   &synced,
		LastSyncSource: "TRENDYOL",
	}
	if err := repo.Update(context.Background(), orgA, &updated); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := repo.Update(context.Background(), orgB, &updated); err != repository.ErrNotFound {
		t.Fatalf("expected cross-tenant update failure")
	}
	got := repo.items[0]
	if got.CreatedAt != created {
		t.Fatalf("createdAt changed")
	}
	if got.Name.Value != "New" {
		t.Fatalf("expected updated name")
	}
	if got.LastSyncedAt == nil {
		t.Fatalf("expected lastSyncedAt persisted")
	}
}
