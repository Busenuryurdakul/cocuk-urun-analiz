package marketplace

import (
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
)

func TestMergePreservesLocalWhenExternalEmpty(t *testing.T) {
	current := domain.Product{
		Name:  domain.ProductFieldMeta{Value: "Local Name", Missing: false},
		Brand: domain.ProductFieldMeta{Value: "acme", Missing: false},
	}
	incoming := normalize.ProductInput{Name: "", Brand: ""}
	out := mergeProductFromFetch(current, incoming, "TRENDYOL", "1", time.Now().UTC())
	if out.Product.Name.Value != "Local Name" {
		t.Fatalf("expected local name preserved, got %v", out.Product.Name.Value)
	}
	if out.FieldsChanged {
		t.Fatalf("expected no field changes")
	}
}

func TestMergeUpdatesPriceAndSlug(t *testing.T) {
	current := domain.Product{
		CurrentPrice: domain.ProductFieldMeta{Value: 10.0, Missing: false},
	}
	incoming := normalize.ProductInput{CurrentPrice: 12.5}
	out := mergeProductFromFetch(current, incoming, "TRENDYOL", "1", time.Now().UTC())
	if !out.FieldsChanged {
		t.Fatalf("expected price change")
	}
	if len(out.UpdatedFields) != 1 || out.UpdatedFields[0] != "price" {
		t.Fatalf("unexpected updated fields: %v", out.UpdatedFields)
	}
}

func TestMergeUnchangedFieldNotListed(t *testing.T) {
	current := domain.Product{
		Rating: domain.ProductFieldMeta{Value: 4.5, Missing: false},
	}
	incoming := normalize.ProductInput{Rating: 4.5}
	out := mergeProductFromFetch(current, incoming, "TRENDYOL", "1", time.Now().UTC())
	if out.FieldsChanged || len(out.UpdatedFields) > 0 {
		t.Fatalf("expected no updates for equal rating")
	}
}
