package marketplace_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/marketplace"
)

func TestHepsiburadaURLDetection(t *testing.T) {
	adapter := marketplace.NewHepsiburadaAdapter()
	ok, id := adapter.DetectURL("https://www.hepsiburada.com/product/lego-set-hb123")
	if !ok || id != "lego-set-hb123" {
		t.Fatalf("unexpected detect result ok=%v id=%q", ok, id)
	}
}

func TestTrendyolURLDetection(t *testing.T) {
	adapter := marketplace.NewTrendyolAdapter()
	ok, id := adapter.DetectURL("https://www.trendyol.com/some-product-p-987654")
	if !ok || id != "987654" {
		t.Fatalf("unexpected detect result ok=%v id=%q", ok, id)
	}
}

func TestFetchDeferredWithReason(t *testing.T) {
	adapter := marketplace.NewTrendyolAdapter()
	result, err := adapter.Fetch(t.Context(), "https://www.trendyol.com/item-p-1")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !result.Deferred || result.DeferredReason != domain.DeferredFetchReason {
		t.Fatalf("expected deferred fetch, got %+v", result)
	}
}
