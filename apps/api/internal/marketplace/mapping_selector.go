package marketplace

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

var errEmptyMappings = errors.New("no source mappings available")

func isExternalLiveSource(source domain.MarketplaceSource) bool {
	switch source {
	case domain.MarketplaceHepsiburada, domain.MarketplaceTrendyol, domain.MarketplaceAmazon, domain.MarketplaceN11:
		return true
	default:
		return false
	}
}

func isRegistrySupported(source domain.MarketplaceSource) bool {
	switch source {
	case domain.MarketplaceHepsiburada, domain.MarketplaceTrendyol:
		return true
	default:
		return false
	}
}

func mappingFreshnessScore(product *domain.Product, mapping domain.ProductSourceMapping) time.Time {
	if product != nil && product.LastSyncSource == string(mapping.Source) && product.LastSyncedAt != nil {
		return product.LastSyncedAt.UTC()
	}
	if product != nil {
		if ts := maxExtractedAtForSource(product, string(mapping.Source)); !ts.IsZero() {
			return ts
		}
	}
	if !mapping.UpdatedAt.IsZero() {
		return mapping.UpdatedAt.UTC()
	}
	return mapping.CreatedAt.UTC()
}

func maxExtractedAtForSource(product *domain.Product, source string) time.Time {
	fields := []domain.ProductFieldMeta{
		product.Name, product.Brand, product.Category, product.Description,
		product.CurrentPrice, product.Rating, product.ReviewCount, product.StockStatus,
	}
	var max time.Time
	for _, f := range fields {
		if f.Source != source || f.ExtractedAt == nil {
			continue
		}
		if f.ExtractedAt.After(max) {
			max = f.ExtractedAt.UTC()
		}
	}
	return max
}

func sourcePriority(source domain.MarketplaceSource) int {
	switch source {
	case domain.MarketplaceHepsiburada:
		return 1
	case domain.MarketplaceTrendyol:
		return 2
	default:
		return 10
	}
}

type mappingCandidate struct {
	mapping domain.ProductSourceMapping
	score   int
	fresh   time.Time
}

// selectSourceMapping picks the deterministic external mapping for live sync.
func selectSourceMapping(product *domain.Product, mappings []domain.ProductSourceMapping) (*domain.ProductSourceMapping, error) {
	if len(mappings) == 0 {
		return nil, errEmptyMappings
	}

	withURL := make([]domain.ProductSourceMapping, 0)
	external := make([]domain.ProductSourceMapping, 0)
	for _, m := range mappings {
		if !isExternalLiveSource(m.Source) || !isRegistrySupported(m.Source) {
			continue
		}
		external = append(external, m)
		if strings.TrimSpace(m.SourceURL) != "" {
			withURL = append(withURL, m)
		}
	}
	if len(external) == 0 {
		return nil, errEmptyMappings
	}

	pool := external
	if len(withURL) > 0 {
		pool = withURL
	}

	candidates := make([]mappingCandidate, 0, len(pool))
	for _, m := range pool {
		score := 0
		if strings.TrimSpace(m.SourceURL) != "" {
			score += 100
		}
		candidates = append(candidates, mappingCandidate{
			mapping: m,
			score:   score,
			fresh:   mappingFreshnessScore(product, m),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if !candidates[i].fresh.Equal(candidates[j].fresh) {
			return candidates[i].fresh.After(candidates[j].fresh)
		}
		pi := sourcePriority(candidates[i].mapping.Source)
		pj := sourcePriority(candidates[j].mapping.Source)
		if pi != pj {
			return pi < pj
		}
		return candidates[i].mapping.ID.Hex() < candidates[j].mapping.ID.Hex()
	})

	selected := candidates[0].mapping
	copy := selected
	return &copy, nil
}
