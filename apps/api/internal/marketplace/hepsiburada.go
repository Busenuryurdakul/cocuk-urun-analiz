package marketplace

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

var hepsiburadaProductRE = regexp.MustCompile(`(?i)/(?:p|product)/([^/?#]+)`)

type HepsiburadaAdapter struct{}

func NewHepsiburadaAdapter() *HepsiburadaAdapter {
	return &HepsiburadaAdapter{}
}

func (a *HepsiburadaAdapter) Source() domain.MarketplaceSource {
	return domain.MarketplaceHepsiburada
}

func (a *HepsiburadaAdapter) DetectURL(rawURL string) (bool, string) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false, ""
	}
	if !hostMatches(parsed, "hepsiburada.com") {
		return false, ""
	}
	if m := hepsiburadaProductRE.FindStringSubmatch(parsed.Path); len(m) == 2 {
		return true, strings.TrimSpace(m[1])
	}
	return false, ""
}

func (a *HepsiburadaAdapter) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	ok, productID := a.DetectURL(rawURL)
	if !ok {
		return nil, ErrUnsupportedURL
	}
	_ = ctx
	return &FetchResult{
		Deferred:        true,
		DeferredReason:  domain.DeferredFetchReason,
		Source:          domain.MarketplaceHepsiburada,
		SourceURL:       rawURL,
		SourceProductID: productID,
	}, nil
}
