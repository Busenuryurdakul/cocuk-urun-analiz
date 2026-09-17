package marketplace

import (
	"context"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestTrendyolMissingCredentialsDeferred(t *testing.T) {
	adapter := NewTrendyolAdapter(ProviderSettings{})
	result, err := adapter.Fetch(context.Background(), "https://www.trendyol.com/item-p-1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deferred || result.DeferredReason != domain.DeferredFetchReason {
		t.Fatalf("expected deferred without credentials, got %+v", result)
	}
}

func TestTrendyolPartialCredentialsDeferred(t *testing.T) {
	adapter := NewTrendyolAdapter(ProviderSettings{
		TrendyolAPIBaseURL: "https://api.trendyol.com",
		TrendyolAPIKey:     "key-only",
	})
	if adapter.settings.TrendyolConfigured() {
		t.Fatal("partial config must not be configured")
	}
	result, err := adapter.Fetch(context.Background(), "https://www.trendyol.com/item-p-1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deferred {
		t.Fatal("expected deferred with partial credentials")
	}
}

func TestTrendyolFullConfigBlockedProviderContract(t *testing.T) {
	settings := ProviderSettings{
		TrendyolAPIBaseURL: "https://api.trendyol.com",
		TrendyolAPIKey:     "key",
		TrendyolAPISecret:  "secret",
		TrendyolSupplierID: "supplier",
	}
	if !settings.TrendyolConfigured() {
		t.Fatal("expected full trendyol config")
	}
	adapter := NewTrendyolAdapter(settings)
	result, err := adapter.Fetch(context.Background(), "https://www.trendyol.com/item-p-1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deferred || result.DeferredReason != blockedProviderContractReason {
		t.Fatalf("expected blocked contract deferred, got %+v", result)
	}
}

func TestHepsiburadaMissingCredentialsDeferred(t *testing.T) {
	adapter := NewHepsiburadaAdapter(ProviderSettings{})
	result, err := adapter.Fetch(context.Background(), "https://www.hepsiburada.com/product/test-hb123")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deferred || result.DeferredReason != domain.DeferredFetchReason {
		t.Fatalf("expected deferred, got %+v", result)
	}
}

func TestHepsiburadaPartialCredentialsDeferred(t *testing.T) {
	adapter := NewHepsiburadaAdapter(ProviderSettings{
		HepsiburadaAPIBaseURL: "https://api.hepsiburada.com",
	})
	if adapter.settings.HepsiburadaConfigured() {
		t.Fatal("partial config must not be configured")
	}
	result, err := adapter.Fetch(context.Background(), "https://www.hepsiburada.com/product/test-hb123")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deferred {
		t.Fatal("expected deferred")
	}
}

func TestHepsiburadaFullConfigBlockedProviderContract(t *testing.T) {
	settings := ProviderSettings{
		HepsiburadaAPIBaseURL: "https://api.hepsiburada.com",
		HepsiburadaAPIKey:     "key",
		HepsiburadaAPISecret:  "secret",
	}
	if !settings.HepsiburadaConfigured() {
		t.Fatal("expected full hepsiburada config")
	}
	adapter := NewHepsiburadaAdapter(settings)
	result, err := adapter.Fetch(context.Background(), "https://www.hepsiburada.com/product/test-hb123")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deferred || result.DeferredReason != blockedProviderContractReason {
		t.Fatalf("expected blocked contract, got %+v", result)
	}
}
