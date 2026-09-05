package marketplace

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

var trendyolProductRE = regexp.MustCompile(`(?i)-p-(\d+)`)

type TrendyolAdapter struct{}

func NewTrendyolAdapter() *TrendyolAdapter {
	return &TrendyolAdapter{}
}

func (a *TrendyolAdapter) Source() domain.MarketplaceSource {
	return domain.MarketplaceTrendyol
}

func (a *TrendyolAdapter) DetectURL(rawURL string) (bool, string) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false, ""
	}
	if !hostMatches(parsed, "trendyol.com") {
		return false, ""
	}
	if m := trendyolProductRE.FindStringSubmatch(parsed.Path); len(m) == 2 {
		return true, strings.TrimSpace(m[1])
	}
	return false, ""
}

func (a *TrendyolAdapter) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	ok, productID := a.DetectURL(rawURL)
	if !ok {
		return nil, ErrUnsupportedURL
	}
	_ = ctx
	return &FetchResult{
		Deferred:        true,
		DeferredReason:  domain.DeferredFetchReason,
		Source:          domain.MarketplaceTrendyol,
		SourceURL:       rawURL,
		SourceProductID: productID,
	}, nil
}
