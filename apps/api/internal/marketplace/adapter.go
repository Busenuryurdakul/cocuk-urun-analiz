package marketplace

import (
	"context"
	"errors"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
)

var ErrUnsupportedURL = errors.New("unsupported marketplace url")

const blockedProviderContractReason = "BLOCKED_PROVIDER_CONTRACT"

type FetchResult struct {
	Deferred        bool
	DeferredReason  string
	Source          domain.MarketplaceSource
	SourceURL       string
	SourceProductID string
	Product         *normalize.ProductInput
	Reviews         []importpkg.ReviewRow
	FetchedAt       time.Time
}

type Adapter interface {
	Source() domain.MarketplaceSource
	DetectURL(rawURL string) (bool, string)
	Fetch(ctx context.Context, rawURL string) (*FetchResult, error)
}
