package marketplace

import (
	"context"
	"errors"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

var ErrUnsupportedURL = errors.New("unsupported marketplace url")

type FetchResult struct {
	Deferred        bool
	DeferredReason  string
	Source          domain.MarketplaceSource
	SourceURL       string
	SourceProductID string
}

type Adapter interface {
	Source() domain.MarketplaceSource
	DetectURL(rawURL string) (bool, string)
	Fetch(ctx context.Context, rawURL string) (*FetchResult, error)
}
