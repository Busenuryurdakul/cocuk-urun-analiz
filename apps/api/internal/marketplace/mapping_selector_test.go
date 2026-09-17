package marketplace

import (
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSelectMappingPrefersSourceURL(t *testing.T) {
	product := &domain.Product{}
	mappings := []domain.ProductSourceMapping{
		{ID: primitive.NewObjectID(), Source: domain.MarketplaceTrendyol, SourceProductID: "1"},
		{ID: primitive.NewObjectID(), Source: domain.MarketplaceHepsiburada, SourceProductID: "2", SourceURL: "https://www.hepsiburada.com/p/x"},
	}
	got, err := selectSourceMapping(product, mappings)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Source != domain.MarketplaceHepsiburada {
		t.Fatalf("expected hepsiburada with url, got %s", got.Source)
	}
}

func TestSelectMappingMiyunaNotPreferred(t *testing.T) {
	product := &domain.Product{}
	mappings := []domain.ProductSourceMapping{
		{ID: primitive.NewObjectID(), Source: domain.MarketplaceMiyuna, SourceProductID: "m1", SourceURL: "https://miyuna.local/p/1"},
		{ID: primitive.NewObjectID(), Source: domain.MarketplaceTrendyol, SourceProductID: "t1", SourceURL: "https://www.trendyol.com/x-p-1"},
	}
	got, err := selectSourceMapping(product, mappings)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Source != domain.MarketplaceTrendyol {
		t.Fatalf("expected trendyol external mapping, got %s", got.Source)
	}
}

func TestSelectMappingEmptyReturnsError(t *testing.T) {
	_, err := selectSourceMapping(&domain.Product{}, nil)
	if err == nil {
		t.Fatalf("expected error for empty mappings")
	}
}

func TestSelectMappingUsesFreshnessMetadata(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-48 * time.Hour)
	product := &domain.Product{
		LastSyncedAt:   &now,
		LastSyncSource: string(domain.MarketplaceTrendyol),
	}
	mappings := []domain.ProductSourceMapping{
		{ID: primitive.NewObjectID(), Source: domain.MarketplaceHepsiburada, SourceProductID: "hb", SourceURL: "https://www.hepsiburada.com/p/hb", UpdatedAt: older},
		{ID: primitive.NewObjectID(), Source: domain.MarketplaceTrendyol, SourceProductID: "ty", SourceURL: "https://www.trendyol.com/x-p-1", UpdatedAt: older},
	}
	got, err := selectSourceMapping(product, mappings)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Source != domain.MarketplaceTrendyol {
		t.Fatalf("expected fresher trendyol mapping, got %s", got.Source)
	}
}
