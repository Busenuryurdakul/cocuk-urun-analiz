package normalize_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
)

func TestProductDedupKeyDeterministic(t *testing.T) {
	a := normalize.ProductDedupKey(domain.MarketplaceHepsiburada, " ABC ", "", "", "", " sku-1 ")
	b := normalize.ProductDedupKey(domain.MarketplaceHepsiburada, "ABC", "", "", "", "SKU-1")
	if a != b {
		t.Fatalf("expected stable dedup key, got %q vs %q", a, b)
	}
}

func TestReviewFingerprintChangesWithText(t *testing.T) {
	fp1 := normalize.ReviewFingerprint(domain.MarketplaceTrendyol, "123", "r1", "great product")
	fp2 := normalize.ReviewFingerprint(domain.MarketplaceTrendyol, "123", "r1", "great  product")
	if fp1 != fp2 {
		t.Fatalf("normalized text should produce same fingerprint")
	}
	fp3 := normalize.ReviewFingerprint(domain.MarketplaceTrendyol, "123", "r1", "bad product")
	if fp1 == fp3 {
		t.Fatalf("different review text must change fingerprint")
	}
}

func TestDetectLanguageTurkish(t *testing.T) {
	if got := normalize.DetectLanguage("çok güzel bir ürün"); got != domain.LanguageTR {
		t.Fatalf("expected TR, got %s", got)
	}
}
