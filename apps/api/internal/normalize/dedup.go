package normalize

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

// ProductDedupKey builds a deterministic dedup key from source identifiers.
func ProductDedupKey(source domain.MarketplaceSource, sourceProductID, gtin, ean, upc, sku string) string {
	parts := []string{
		string(source),
		NormalizeIdentifier(sourceProductID),
		NormalizeIdentifier(gtin),
		NormalizeIdentifier(ean),
		NormalizeIdentifier(upc),
		NormalizeIdentifier(sku),
	}
	return strings.Join(parts, "|")
}

// ReviewFingerprint computes a stable fingerprint for review deduplication.
func ReviewFingerprint(source domain.MarketplaceSource, sourceProductID, sourceReviewID, reviewText string) string {
	text := NormalizeReviewText(reviewText)
	payload := fmt.Sprintf("%s|%s|%s|%s", source, NormalizeIdentifier(sourceProductID), NormalizeIdentifier(sourceReviewID), text)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
