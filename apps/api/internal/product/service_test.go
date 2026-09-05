package product_test

import (
	"context"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type memoryProducts struct {
	items []domain.Product
}

func (m *memoryProducts) Create(_ context.Context, p *domain.Product) error {
	p.ID = primitive.NewObjectID()
	m.items = append(m.items, *p)
	return nil
}

func (m *memoryProducts) FindByID(_ context.Context, organizationID, productID primitive.ObjectID) (*domain.Product, error) {
	for i := range m.items {
		if m.items[i].ID == productID && m.items[i].OrganizationID == organizationID {
			return &m.items[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

type memoryMappings struct {
	items []domain.ProductSourceMapping
}

func (m *memoryMappings) Create(_ context.Context, mapping *domain.ProductSourceMapping) error {
	mapping.ID = primitive.NewObjectID()
	m.items = append(m.items, *mapping)
	return nil
}

func (m *memoryMappings) FindBySourceProduct(_ context.Context, organizationID primitive.ObjectID, source domain.MarketplaceSource, sourceProductID string) (*domain.ProductSourceMapping, error) {
	for i := range m.items {
		item := m.items[i]
		if item.OrganizationID == organizationID && item.Source == source && item.SourceProductID == sourceProductID {
			return &item, nil
		}
	}
	return nil, repository.ErrNotFound
}

func TestProductServiceDedupViaMapping(t *testing.T) {
	orgID := primitive.NewObjectID()
	productID := primitive.NewObjectID()

	products := &memoryProducts{items: []domain.Product{{
		ID:             productID,
		OrganizationID: orgID,
	}}}
	mappings := &memoryMappings{items: []domain.ProductSourceMapping{{
		OrganizationID:  orgID,
		ProductID:       productID,
		Source:          domain.MarketplaceHepsiburada,
		SourceProductID: "HB-1",
	}}}

	got, dedup, err := dedupCreate(products, mappings, product.CreateInput{
		OrganizationID:  orgID,
		Source:          domain.MarketplaceHepsiburada,
		SourceProductID: "HB-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !dedup || got.ID != productID {
		t.Fatalf("expected deduplicated existing product")
	}
}

func dedupCreate(products *memoryProducts, mappings *memoryMappings, input product.CreateInput) (*domain.Product, bool, error) {
	existing, err := mappings.FindBySourceProduct(context.Background(), input.OrganizationID, input.Source, input.SourceProductID)
	if err == nil {
		p, err := products.FindByID(context.Background(), input.OrganizationID, existing.ProductID)
		return p, true, err
	}
	p := &domain.Product{OrganizationID: input.OrganizationID}
	if err := products.Create(context.Background(), p); err != nil {
		return nil, false, err
	}
	_ = mappings.Create(context.Background(), &domain.ProductSourceMapping{
		OrganizationID:  input.OrganizationID,
		ProductID:       p.ID,
		Source:          input.Source,
		SourceProductID: input.SourceProductID,
	})
	return p, false, nil
}
